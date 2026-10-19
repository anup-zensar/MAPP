package activities

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// settingsStandIn is the SettingsReader of a build without feature 002's service: the same reader API,
// worked out from the settings repository with the rules of 002 data model §9 and the messages of 002's
// "Check results" table. When feature 002 is merged, internal/app passes *settings.Service instead.
// It knows nothing of where Settings records are used, so ListAttributes answers every attribute as
// "Not used yet" and without locks; the activity form does not show either.
type settingsStandIn struct {
	repo settings.Repository
}

// NewSettingsStandIn returns a SettingsReader over the settings repository, until feature 002's service
// is wired in its place.
func NewSettingsStandIn(repo settings.Repository) SettingsReader {
	return settingsStandIn{repo: repo}
}

// Messages of 002's "Check results (API)" table.
const (
	attributeGoneMessage   = "This Shell attribute no longer exists — reload the page."
	otherLevelMessage      = "%s belongs to the %s form."
	inactiveChangedMessage = "%s is inactive and can't be changed."
	chooseOptionMessage    = "Choose one of the options of %s."
	inactiveOptionMessage  = "'%s' is inactive — choose another option."
	notLinkedMessage       = "'%s' doesn't belong to the chosen %s — choose another %s."
	enterTextMessage       = "Enter text."
	textTooLongMessage     = "Use 120 characters or fewer."
	longTextTooLongMessage = "Use 2,000 characters or fewer."
	wholeNumberMessage     = "Enter a whole number between -999,999,999,999 and 999,999,999,999."
	yesOrNoMessage         = "Choose Yes or No."
	shareMessage           = "Enter a share above 0 % with at most two decimals."
	optionTwiceMessage     = "Each option can appear only once."
	splitReason            = "Shares add up to %s %% — they must add up to 100.00 %%."
)

// The limits of attribute values (002 data model §9).
const (
	maxTextLength     = 120
	maxLongTextLength = 2000
	maxNumber         = 999_999_999_999
)

// levelWords are the levels as the "belongs to the … form" message names them.
var levelWords = map[settings.Level]string{
	settings.LevelPlan: "Plan", settings.LevelCampaign: "Campaign", settings.LevelInitiative: "Initiative",
	settings.LevelActivity: "Activity", settings.LevelLineItem: "Line item",
}

func (s settingsStandIn) Model(ctx context.Context, id string) (settings.Model, bool, error) {
	return s.repo.GetModel(ctx, id)
}

func (s settingsStandIn) ActivityType(ctx context.Context, id string) (settings.ActivityType, bool, error) {
	return s.repo.GetActivityType(ctx, id)
}

// ActivityTypes returns every activity type sorted by name, ignoring letter case.
func (s settingsStandIn) ActivityTypes(ctx context.Context) ([]settings.ActivityType, error) {
	types, err := s.repo.ListActivityTypes(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(types, func(x, y settings.ActivityType) int {
		return cmp.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
	})
	return types, nil
}

// Attributes returns the attributes of level in display order (by position).
func (s settingsStandIn) Attributes(ctx context.Context, level settings.Level) ([]settings.Attribute, error) {
	all, err := s.repo.ListAttributes(ctx)
	if err != nil {
		return nil, err
	}
	list := []settings.Attribute{}
	for _, a := range all {
		if a.Level == level {
			list = append(list, a)
		}
	}
	slices.SortStableFunc(list, func(x, y settings.Attribute) int { return cmp.Compare(x.Position, y.Position) })
	return list, nil
}

func (s settingsStandIn) Option(ctx context.Context, id string) (settings.Option, bool, error) {
	return s.repo.GetOption(ctx, id)
}

// ListAttributes returns the attributes of level (every level when nil) as the API shows them.
func (s settingsStandIn) ListAttributes(ctx context.Context, _ actor.Actor, level *settings.Level) (settings.ShellAttributePage, error) {
	levels := settings.Levels
	if level != nil {
		levels = []settings.Level{*level}
	}
	views := []settings.ShellAttributeView{}
	for _, l := range levels {
		attrs, err := s.Attributes(ctx, l)
		if err != nil {
			return settings.ShellAttributePage{}, err
		}
		for _, a := range attrs {
			view, err := s.attributeView(ctx, a)
			if err != nil {
				return settings.ShellAttributePage{}, err
			}
			views = append(views, view)
		}
	}
	return settings.ShellAttributePage{Items: views, Total: len(views)}, nil
}

// attributeView is one attribute as the API shows it.
func (s settingsStandIn) attributeView(ctx context.Context, a settings.Attribute) (settings.ShellAttributeView, error) {
	options, err := s.repo.OptionsOf(ctx, a.ID)
	if err != nil {
		return settings.ShellAttributeView{}, err
	}
	active := 0
	for _, o := range options {
		if o.Active {
			active++
		}
	}
	view := settings.ShellAttributeView{
		ID: a.ID, Label: a.Label, HelpText: a.HelpText, Level: a.Level, Type: a.Type, Marking: a.Marking,
		Section: a.Section, Position: a.Position, Condition: []settings.SettingRef{},
		BusinessUnit: a.Role == settings.RoleBusinessUnit, Active: a.Active, OptionCount: len(options),
		ActiveOptionCount: active, Locks: []settings.Lock{}, Meta: a.Meta,
		Usage: settings.Usage{Summary: "Not used yet", Parts: []settings.UsagePart{}, Records: []errs.RecordRef{}},
	}
	if a.ParentAttributeID != "" {
		parent, found, err := s.repo.GetAttribute(ctx, a.ParentAttributeID)
		if err != nil {
			return settings.ShellAttributeView{}, err
		}
		if found {
			view.ParentAttribute = &settings.SettingRef{ID: parent.ID, Name: parent.Label, Active: parent.Active}
		}
	}
	for _, id := range a.Condition {
		o, found, err := s.repo.GetOption(ctx, id)
		if err != nil {
			return settings.ShellAttributeView{}, err
		}
		if found {
			view.Condition = append(view.Condition, settings.SettingRef{ID: o.ID, Name: o.Label, Active: o.Active})
		}
	}
	return view, nil
}

// BusinessUnit returns the option ids of the plan-level Business unit attribute among planValues.
func (s settingsStandIn) BusinessUnit(ctx context.Context, planValues settings.AttributeValues) ([]string, error) {
	attrs, err := s.Attributes(ctx, settings.LevelPlan)
	if err != nil {
		return nil, err
	}
	for _, a := range attrs {
		if a.Role == settings.RoleBusinessUnit {
			return optionIDsOf(a, planValues[a.ID]), nil
		}
	}
	return []string{}, nil
}

// CheckAttributes checks the values of a save of a record of level (002 data model §9).
func (s settingsStandIn) CheckAttributes(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error) {
	result := settings.AttributeResult{
		Problems: []errs.FieldError{}, Cleared: []settings.ClearedValue{}, Values: settings.AttributeValues{},
	}
	attrs, err := s.attributesByID(ctx)
	if err != nil {
		return result, err
	}
	options, err := s.optionsByID(ctx)
	if err != nil {
		return result, err
	}

	for _, id := range valueOrder(in.Values, attrs) {
		raw := in.Values[id]
		field := "attributes." + id
		attr, found := attrs[id]
		switch {
		case !found:
			result.Problems = append(result.Problems, errs.FieldError{Field: field, Message: attributeGoneMessage})
			continue
		case attr.Level != level:
			result.Problems = append(result.Problems,
				errs.FieldError{Field: field, Message: fmt.Sprintf(otherLevelMessage, attr.Label, levelWords[attr.Level])})
			continue
		case isEmptyValue(raw):
			continue
		case !attr.Active:
			if !sameJSON(raw, in.Previous[id]) {
				result.Problems = append(result.Problems,
					errs.FieldError{Field: field, Message: fmt.Sprintf(inactiveChangedMessage, attr.Label)})
				continue
			}
			result.Values[id] = raw
			continue
		}
		value, message := normalise(attr, raw, options, optionIDsOf(attr, in.Previous[id]))
		if message != "" {
			result.Problems = append(result.Problems, errs.FieldError{Field: field, Message: message})
			continue
		}
		if value != nil {
			result.Values[id] = value
		}
	}

	s.checkDependents(attrs, options, in, &result)
	missing, err := s.MissingAttributes(ctx, level, result.Values, in.BusinessUnit)
	if err != nil {
		return result, err
	}
	result.Missing = missing
	return result, nil
}

// checkDependents applies the parent rule to the values of dependent attributes: an option that does not
// fit the chosen parent options is removed when the parent value changed in this save (listed as
// cleared), and refused when it was newly chosen (FR-025).
func (s settingsStandIn) checkDependents(attrs map[string]settings.Attribute, options map[string]settings.Option,
	in settings.AttributeInput, result *settings.AttributeResult) {
	for _, id := range slices.Sorted(maps.Keys(result.Values)) {
		attr := attrs[id]
		parent, hasParent := attrs[attr.ParentAttributeID]
		if !hasParent || !attr.Active {
			continue
		}
		parentChosen := optionIDsOf(parent, result.Values[parent.ID])
		parentChanged := !slices.Equal(sortedCopy(parentChosen), sortedCopy(optionIDsOf(parent, in.Previous[parent.ID])))
		previous := optionIDsOf(attr, in.Previous[id])
		var cleared []string
		for _, optionID := range optionIDsOf(attr, result.Values[id]) {
			option := options[optionID]
			if linkedToAny(option, parentChosen) {
				continue
			}
			if parentChanged {
				cleared = append(cleared, optionID)
			} else if !slices.Contains(previous, optionID) {
				result.Problems = append(result.Problems, errs.FieldError{Field: "attributes." + id, Message: fmt.Sprintf(
					notLinkedMessage, option.Label, midSentence(parent.Label), midSentence(attr.Label))})
			}
		}
		if len(cleared) == 0 {
			continue
		}
		labels := make([]string, 0, len(cleared))
		for _, optionID := range cleared {
			labels = append(labels, options[optionID].Label)
		}
		result.Cleared = append(result.Cleared,
			settings.ClearedValue{AttributeID: id, Label: attr.Label, OptionIDs: cleared, OptionLabels: labels})
		if kept := withoutOptions(attr, result.Values[id], cleared); kept == nil {
			delete(result.Values, id)
		} else {
			result.Values[id] = kept
		}
	}
}

// MissingAttributes lists the needed attributes of level that are active, shown for businessUnit and empty,
// and the shown splits whose shares do not total 100.00 % (002 data model §9).
func (s settingsStandIn) MissingAttributes(ctx context.Context, level settings.Level, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error) {
	attrs, err := s.Attributes(ctx, level)
	if err != nil {
		return nil, err
	}
	needed := settings.MarkingNeededToPublish
	if level == settings.LevelActivity || level == settings.LevelLineItem {
		needed = settings.MarkingNeededToSubmit
	}
	byID, err := s.attributesByID(ctx)
	if err != nil {
		return nil, err
	}
	missing := []settings.MissingField{}
	for _, a := range attrs {
		if !a.Active || a.Type == settings.TypeYesNo || a.Marking != needed || !visible(a, businessUnit) {
			continue
		}
		raw := values[a.ID]
		if isEmptyValue(raw) {
			if parent, ok := byID[a.ParentAttributeID]; ok {
				fits, err := s.hasOptionsFor(ctx, a, optionIDsOf(parent, values[parent.ID]))
				if err != nil {
					return nil, err
				}
				if !fits {
					continue // D-034: nothing to choose for the chosen parent options
				}
			}
			missing = append(missing, settings.MissingField{Field: "attributes." + a.ID, Label: a.Label})
			continue
		}
		if a.Type == settings.TypeMultipleChoiceSplit {
			if total := splitTotal(raw); total.Cmp(money.Hundred) != 0 {
				missing = append(missing, settings.MissingField{
					Field: "attributes." + a.ID, Label: a.Label, Reason: fmt.Sprintf(splitReason, total.String()),
				})
			}
		}
	}
	return missing, nil
}

// hasOptionsFor reports whether the dependent attribute a has an active option linked to any of the
// chosen parent options.
func (s settingsStandIn) hasOptionsFor(ctx context.Context, a settings.Attribute, parentChosen []string) (bool, error) {
	options, err := s.repo.OptionsOf(ctx, a.ID)
	if err != nil {
		return false, err
	}
	for _, o := range options {
		if o.Active && linkedToAny(o, parentChosen) {
			return true, nil
		}
	}
	return false, nil
}

// DescribeAttribute gives the label and text of an "attributes.<id>" field: option labels joined with ", ",
// shares as "Germany 50.00 %", Yes or No, the text or the number. Other fields get "" and "".
func (s settingsStandIn) DescribeAttribute(ctx context.Context, field string, value any) (string, string, error) {
	id, isAttribute := strings.CutPrefix(field, "attributes.")
	if !isAttribute {
		return "", "", nil
	}
	attr, found, err := s.repo.GetAttribute(ctx, id)
	if err != nil || !found {
		return "", "", err
	}
	raw, err := json.Marshal(value)
	if err != nil || value == nil {
		return attr.Label, "", err
	}
	var parts []string
	switch attr.Type {
	case settings.TypeMultipleChoiceSplit:
		for _, share := range splitShares(raw) {
			label, err := s.optionLabel(ctx, share.OptionID)
			if err != nil {
				return "", "", err
			}
			parts = append(parts, label+" "+share.Share.Format())
		}
	case settings.TypeSingleChoice, settings.TypeMultipleChoice:
		for _, optionID := range optionIDsOf(attr, raw) {
			label, err := s.optionLabel(ctx, optionID)
			if err != nil {
				return "", "", err
			}
			parts = append(parts, label)
		}
	default:
		return attr.Label, "", nil // yes/no, texts and numbers read well as they are
	}
	return attr.Label, strings.Join(parts, ", "), nil
}

// optionLabel returns the label of the option id, or the id when it no longer exists.
func (s settingsStandIn) optionLabel(ctx context.Context, id string) (string, error) {
	o, found, err := s.repo.GetOption(ctx, id)
	if err != nil || !found {
		return id, err
	}
	return o.Label, nil
}

// attributesByID returns every attribute of every level by id.
func (s settingsStandIn) attributesByID(ctx context.Context) (map[string]settings.Attribute, error) {
	all, err := s.repo.ListAttributes(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]settings.Attribute, len(all))
	for _, a := range all {
		byID[a.ID] = a
	}
	return byID, nil
}

// optionsByID returns every option by id.
func (s settingsStandIn) optionsByID(ctx context.Context) (map[string]settings.Option, error) {
	all, err := s.repo.ListOptions(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]settings.Option, len(all))
	for _, o := range all {
		byID[o.ID] = o
	}
	return byID, nil
}

// valueOrder returns the attribute ids of values: known attributes by level and position, unknown ones
// last, by id, so that problems come in the order of the form.
func valueOrder(values settings.AttributeValues, attrs map[string]settings.Attribute) []string {
	keys := slices.Collect(maps.Keys(values))
	slices.SortFunc(keys, func(x, y string) int {
		ax, knownX := attrs[x]
		ay, knownY := attrs[y]
		switch {
		case knownX && !knownY:
			return -1
		case !knownX && knownY:
			return 1
		case knownX && knownY && ax.Position != ay.Position:
			return cmp.Compare(ax.Position, ay.Position)
		}
		return cmp.Compare(x, y)
	})
	return keys
}

// normalise checks one value of an active attribute against its type and returns it as it is stored, or
// nil when it is empty, or a message for the person. previous are the options the record had before: an
// inactive option already chosen may stay (FR-027).
func normalise(attr settings.Attribute, raw json.RawMessage, options map[string]settings.Option, previous []string) (json.RawMessage, string) {
	chooseMessage := fmt.Sprintf(chooseOptionMessage, attr.Label)
	checkOption := func(id string) string {
		option, found := options[id]
		switch {
		case !found || option.AttributeID != attr.ID:
			return chooseMessage
		case !option.Active && !slices.Contains(previous, id):
			return fmt.Sprintf(inactiveOptionMessage, option.Label)
		}
		return ""
	}
	switch attr.Type {
	case settings.TypeSingleChoice:
		var id string
		if json.Unmarshal(raw, &id) != nil {
			return nil, chooseMessage
		}
		if message := checkOption(id); message != "" {
			return nil, message
		}
		return mustJSON(id), ""
	case settings.TypeMultipleChoice:
		var ids []string
		if json.Unmarshal(raw, &ids) != nil {
			return nil, chooseMessage
		}
		for i, id := range ids {
			if slices.Contains(ids[:i], id) {
				return nil, optionTwiceMessage
			}
			if message := checkOption(id); message != "" {
				return nil, message
			}
		}
		return mustJSON(ids), ""
	case settings.TypeMultipleChoiceSplit:
		var typed []struct {
			OptionID string `json:"optionId"`
			Share    string `json:"share"` // text, so a wrong share is a message, not an unreadable value
		}
		if json.Unmarshal(raw, &typed) != nil {
			return nil, chooseMessage
		}
		shares := make([]settings.SplitShare, 0, len(typed))
		for _, t := range typed {
			share, err := money.ParsePercent(t.Share)
			if err != nil || share.IsZero() || share.Cmp(money.Hundred) > 0 {
				return nil, shareMessage
			}
			if slices.ContainsFunc(shares, func(s settings.SplitShare) bool { return s.OptionID == t.OptionID }) {
				return nil, optionTwiceMessage
			}
			if message := checkOption(t.OptionID); message != "" {
				return nil, message
			}
			shares = append(shares, settings.SplitShare{OptionID: t.OptionID, Share: share})
		}
		return mustJSON(shares), ""
	case settings.TypeYesNo:
		var yes bool
		if json.Unmarshal(raw, &yes) != nil {
			return nil, yesOrNoMessage
		}
		return mustJSON(yes), ""
	case settings.TypeText, settings.TypeLongText:
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return nil, enterTextMessage
		}
		text = strings.TrimSpace(text)
		limit, message := maxTextLength, textTooLongMessage
		if attr.Type == settings.TypeLongText {
			limit, message = maxLongTextLength, longTextTooLongMessage
		}
		if utf8.RuneCountInString(text) > limit {
			return nil, message
		}
		if text == "" {
			return nil, ""
		}
		return mustJSON(text), ""
	case settings.TypeNumber:
		var n int64
		if json.Unmarshal(raw, &n) != nil || n < -maxNumber || n > maxNumber {
			return nil, wholeNumberMessage
		}
		return mustJSON(n), ""
	}
	return nil, chooseMessage
}

// visible reports whether attribute a is shown for a plan whose business unit is businessUnit: always
// when it has no condition, otherwise when the business unit includes one of its units (R9).
func visible(a settings.Attribute, businessUnit []string) bool {
	if len(a.Condition) == 0 {
		return true
	}
	return slices.ContainsFunc(a.Condition, func(id string) bool { return slices.Contains(businessUnit, id) })
}

// optionIDsOf returns the option ids of a choice value (none for other types or unreadable values).
func optionIDsOf(a settings.Attribute, raw json.RawMessage) []string {
	ids := []string{}
	switch a.Type {
	case settings.TypeSingleChoice:
		var id string
		if json.Unmarshal(raw, &id) == nil && id != "" {
			ids = append(ids, id)
		}
	case settings.TypeMultipleChoice:
		var list []string
		if json.Unmarshal(raw, &list) == nil {
			ids = append(ids, list...)
		}
	case settings.TypeMultipleChoiceSplit:
		for _, s := range splitShares(raw) {
			ids = append(ids, s.OptionID)
		}
	}
	return ids
}

// withoutOptions returns a choice value without the options remove, or nil when nothing is left.
func withoutOptions(a settings.Attribute, raw json.RawMessage, remove []string) json.RawMessage {
	switch a.Type {
	case settings.TypeMultipleChoice:
		kept := slices.DeleteFunc(optionIDsOf(a, raw), func(id string) bool { return slices.Contains(remove, id) })
		if len(kept) > 0 {
			return mustJSON(kept)
		}
	case settings.TypeMultipleChoiceSplit:
		kept := slices.DeleteFunc(splitShares(raw), func(s settings.SplitShare) bool { return slices.Contains(remove, s.OptionID) })
		if len(kept) > 0 {
			return mustJSON(kept)
		}
	}
	return nil
}

// splitShares reads a split value; an unreadable one has no shares.
func splitShares(raw json.RawMessage) []settings.SplitShare {
	var shares []settings.SplitShare
	if json.Unmarshal(raw, &shares) != nil {
		return nil
	}
	return shares
}

// splitTotal adds up the shares of a split value.
func splitTotal(raw json.RawMessage) money.Percent {
	var total money.Percent
	for _, s := range splitShares(raw) {
		total = money.SumPercent(total, s.Share)
	}
	return total
}

// linkedToAny reports whether option o is linked to one of the parent options chosen.
func linkedToAny(o settings.Option, chosen []string) bool {
	return slices.ContainsFunc(o.ParentOptionIDs, func(id string) bool { return slices.Contains(chosen, id) })
}

// isEmptyValue reports whether an attribute value is no value: missing, null, "" or [].
func isEmptyValue(raw json.RawMessage) bool {
	switch string(bytes.TrimSpace(raw)) {
	case "", "null", `""`, "[]":
		return true
	}
	return false
}

// sameJSON reports whether two values are the same JSON, ignoring spaces.
func sameJSON(a, b json.RawMessage) bool {
	var x, y bytes.Buffer
	if json.Compact(&x, a) != nil || json.Compact(&y, b) != nil {
		return bytes.Equal(a, b)
	}
	return bytes.Equal(x.Bytes(), y.Bytes())
}

// midSentence writes a label inside a sentence: the first letter lower-cased only when the second is lower
// case, so "Region" becomes "region" and "GL code" stays.
func midSentence(label string) string {
	first, size := utf8.DecodeRuneInString(label)
	second, _ := utf8.DecodeRuneInString(label[size:])
	if unicode.IsLower(second) {
		return string(unicode.ToLower(first)) + label[size:]
	}
	return label
}

// mustJSON encodes values that always encode: strings, numbers, bools and lists of them.
func mustJSON(v any) json.RawMessage {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return encoded
}

// sortedCopy returns a sorted copy of ids.
func sortedCopy(ids []string) []string {
	sorted := slices.Clone(ids)
	slices.Sort(sorted)
	return sorted
}
