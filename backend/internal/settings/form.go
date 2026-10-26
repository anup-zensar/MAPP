package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
)

// The limits of attribute values (data model §9, D-053).
const (
	maxTextValue     = 120
	maxLongTextValue = 2000
	maxNumberValue   = 999_999_999_999
)

// Messages of the plan's "User-visible messages" table, rows "Check results (API)".
const (
	sharesTotalMessage      = "Shares add up to %s %% — they must add up to 100.00 %%."
	chooseOptionMessage     = "Choose one of the options of %s."
	inactiveOptionMessage   = "'%s' is inactive — choose another option."
	notLinkedMessage        = "'%s' doesn't belong to the chosen %s — choose another %s."
	enterTextMessage        = "Enter text."
	wholeNumberMessage      = "Enter a whole number between -999,999,999,999 and 999,999,999,999."
	chooseYesOrNoMessage    = "Choose Yes or No."
	shareMessage            = "Enter a share above 0 % with at most two decimals."
	onceMessage             = "Each option can appear only once."
	inactiveAttributeChange = "%s is inactive and can't be changed."
	unknownAttributeMessage = "This Shell attribute no longer exists — reload the page."
	otherLevelMessage       = "%s belongs to the %s form."
)

// formData is what the attribute rules need about one level: its attributes in display order, active and
// inactive, every attribute by id (to tell another level's attribute from an unknown one), every option
// and the plan-level Business unit attribute.
type formData struct {
	level        Level
	attributes   []Attribute
	byID         map[string]Attribute
	options      map[string][]Option // attribute id → its options in display order
	optionByID   map[string]Option
	businessUnit *Attribute
}

// formOf is loadForm for the rules that only read the form (completeness, the needed-field checks and the
// attribute checks of a save): the form is read once per transaction and shared, because these rules run
// once per record and a page such as Home or the Plans list checks every record of every plan. The form
// must not be changed by its callers. Code that changes Settings uses loadForm.
func (s *Service) formOf(ctx context.Context, level Level) (formData, error) {
	return store.Memo(ctx, "settings:form:"+string(level), func(ctx context.Context) (formData, error) {
		return s.loadForm(ctx, level)
	})
}

// loadForm reads the form of level inside the caller's transaction.
func (s *Service) loadForm(ctx context.Context, level Level) (formData, error) {
	attributes, err := s.repo.ListAttributes(ctx)
	if err != nil {
		return formData{}, err
	}
	options, err := s.repo.ListOptions(ctx)
	if err != nil {
		return formData{}, err
	}
	f := formData{
		level: level, attributes: ofLevel(attributes, level), byID: map[string]Attribute{},
		options: map[string][]Option{}, optionByID: map[string]Option{},
	}
	for _, a := range attributes {
		f.byID[a.ID] = a
		if a.Role == RoleBusinessUnit {
			businessUnit := a
			f.businessUnit = &businessUnit
		}
	}
	for _, o := range optionOrder(options) {
		f.options[o.AttributeID] = append(f.options[o.AttributeID], o)
		f.optionByID[o.ID] = o
	}
	return f, nil
}

// Form returns everything a level's form needs to show its Shell attributes (FR-024, GET
// /api/v1/shell-attribute-forms/{level}): its sections in display order with every attribute, active and
// inactive, and every option; and the Business unit attribute's options for condition notes. Every
// signed-in person may read it (FR-008), so a is not used yet.
func (s *Service) Form(ctx context.Context, a actor.Actor, level Level) (FormView, error) {
	var view FormView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.loadForm(ctx, level)
		if err != nil {
			return err
		}
		view = f.view()
		return nil
	})
	return view, err
}

// view returns the form as the API answers it.
func (f formData) view() FormView {
	view := FormView{Level: f.level, Sections: []FormSection{}, BusinessUnitOptions: []FormOption{}}
	for _, a := range f.attributes {
		last := len(view.Sections) - 1
		if last < 0 || view.Sections[last].Name != a.Section {
			view.Sections = append(view.Sections, FormSection{Name: a.Section, Conditional: true, Attributes: []FormAttribute{}})
			last++
		}
		section := &view.Sections[last]
		section.Attributes = append(section.Attributes, f.formAttribute(a))
		section.Conditional = section.Conditional && len(a.Condition) > 0
	}
	if f.businessUnit != nil {
		id := f.businessUnit.ID
		view.BusinessUnitAttributeID = &id
		view.BusinessUnitOptions = f.formOptions(id)
	}
	return view
}

// formAttribute returns the attribute a as forms need it.
func (f formData) formAttribute(a Attribute) FormAttribute {
	var parent *string
	if a.ParentAttributeID != "" {
		id := a.ParentAttributeID
		parent = &id
	}
	return FormAttribute{
		ID: a.ID, Label: a.Label, HelpText: a.HelpText, Type: a.Type, Marking: a.Marking, Active: a.Active,
		ParentAttributeID: parent, ConditionOptionIDs: nonNil(a.Condition), Options: f.formOptions(a.ID),
	}
}

// formOptions returns the options of the attribute attributeID as forms need them, in display order.
func (f formData) formOptions(attributeID string) []FormOption {
	options := []FormOption{}
	for _, o := range f.options[attributeID] {
		options = append(options, FormOption{ID: o.ID, Label: o.Label, Active: o.Active, ParentOptionIDs: nonNil(o.ParentOptionIDs)})
	}
	return options
}

// nonNil returns list, or an empty list for nil.
func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

// CheckAttributes checks the Shell attribute values of a record of level as a save would (data model §9,
// FR-024 – FR-027): the problems that block saving, the dependent values cleared because their parent
// value changed, the needed attributes still empty, and the values to store. Features 004–007 call it on
// every save, inside their own transaction (ctx); it takes no actor because it only reads Settings.
func (s *Service) CheckAttributes(ctx context.Context, level Level, in AttributeInput) (AttributeResult, error) {
	var result AttributeResult
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.formOf(ctx, level)
		if err != nil {
			return err
		}
		result = f.check(in)
		return nil
	})
	return result, err
}

// CheckValues answers POST /api/v1/shell-attribute-forms/{level}/check: CheckAttributes for the preview's Try
// saving, which saves nothing (FR-020). Every signed-in person may use it (FR-008).
func (s *Service) CheckValues(ctx context.Context, a actor.Actor, level Level, request CheckRequest) (CheckResult, error) {
	result, err := s.CheckAttributes(ctx, level, AttributeInput{
		Values: request.Values, Previous: request.PreviousValues, BusinessUnit: request.BusinessUnitOptionIDs,
	})
	if err != nil {
		return CheckResult{}, err
	}
	return CheckResult{Problems: result.Problems, Missing: result.Missing, Cleared: result.Cleared, Values: result.Values}, nil
}

// MissingAttributes lists the needed Shell attributes of a record of level that are still empty, for its
// completeness and for the publish and submit checks of features 004–008 (FR-026, FR-027): active, shown
// for the plan's business unit businessUnit, not yes/no, and empty — or a % split not adding up to
// 100.00 %. It joins the caller's transaction.
func (s *Service) MissingAttributes(ctx context.Context, level Level, values AttributeValues, businessUnit []string) ([]MissingField, error) {
	var missing []MissingField
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.formOf(ctx, level)
		if err != nil {
			return err
		}
		normalized := map[string]json.RawMessage{}
		for _, a := range f.attributes {
			if raw, problem := f.normalize(a, values[a.ID]); problem == "" && raw != nil {
				normalized[a.ID] = raw
			}
		}
		if level == LevelPlan {
			businessUnit = f.businessUnitOf(normalized)
		}
		missing = f.missing(normalized, businessUnit)
		return nil
	})
	return missing, err
}

// BusinessUnit returns the plan's business unit: the option ids of the Business unit attribute in the
// plan's values planValues, or an empty list when none is chosen (FR-023, FR-026).
func (s *Service) BusinessUnit(ctx context.Context, planValues AttributeValues) ([]string, error) {
	var businessUnit []string
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.formOf(ctx, LevelPlan)
		if err != nil {
			return err
		}
		businessUnit = f.businessUnitOf(planValues)
		return nil
	})
	return businessUnit, err
}

// businessUnitOf returns the Business unit option ids in plan values.
func (f formData) businessUnitOf(values map[string]json.RawMessage) []string {
	if f.businessUnit == nil {
		return []string{}
	}
	return optionIDsOf(*f.businessUnit, values[f.businessUnit.ID])
}

// check is CheckAttributes on the form f, read already.
func (f formData) check(in AttributeInput) AttributeResult {
	result := AttributeResult{Problems: []errs.FieldError{}, Cleared: []ClearedValue{}, Missing: []MissingField{}, Values: AttributeValues{}}
	problem := func(id, message string) {
		result.Problems = append(result.Problems, errs.FieldError{Field: "attributes." + id, Message: message})
	}

	// Values of another level, or of an attribute that no longer exists.
	for _, id := range slices.SortedFunc(maps.Keys(in.Values), compareIDs) {
		a, known := f.byID[id]
		switch {
		case !known:
			problem(id, unknownAttributeMessage)
		case a.Level != f.level:
			problem(id, fmt.Sprintf(otherLevelMessage, a.Label, levelWords[a.Level])) // the form it belongs to
		}
	}

	previous := map[string]json.RawMessage{} // the record's values before, as they would be stored
	for _, a := range f.attributes {
		if raw, p := f.normalize(a, in.Previous[a.ID]); p == "" && raw != nil {
			previous[a.ID] = raw
		}
	}

	values := map[string]json.RawMessage{}
	for _, a := range f.attributes {
		raw, sent := in.Values[a.ID]
		value, p := f.normalize(a, raw)
		if p != "" {
			problem(a.ID, p)
			continue
		}
		before := previous[a.ID]
		if !a.Active {
			// An inactive attribute keeps its value, read-only (FR-027).
			if !sent {
				value = before
			} else if !bytes.Equal(value, before) {
				problem(a.ID, fmt.Sprintf(inactiveAttributeChange, a.Label))
				continue
			}
		}
		if p := f.newInactiveOption(a, value, before); p != "" {
			problem(a.ID, p)
			continue
		}
		if value != nil {
			values[a.ID] = value
		}
	}

	// Dependent attributes (FR-025): parents come before their dependents in display order, so a cleared
	// parent value is already cleared when its dependents are checked.
	for _, a := range f.attributes {
		value := values[a.ID]
		parent, dependent := f.byID[a.ParentAttributeID]
		if !dependent || value == nil {
			continue
		}
		selected := optionIDsOf(parent, values[parent.ID])
		if !bytes.Equal(values[parent.ID], previous[parent.ID]) {
			cleared := f.clearUnlinked(a, value, selected)
			if len(cleared.OptionIDs) == 0 {
				continue
			}
			result.Cleared = append(result.Cleared, cleared)
			if kept := withoutOptions(a, value, cleared.OptionIDs); kept != nil {
				values[a.ID] = kept
			} else {
				delete(values, a.ID)
			}
			continue
		}
		for _, id := range newOptionIDs(a, value, previous[a.ID]) {
			if !f.linked(id, selected) {
				problem(a.ID, fmt.Sprintf(notLinkedMessage, f.optionByID[id].Label, midSentence(parent.Label), midSentence(a.Label)))
				break
			}
		}
	}

	businessUnit := in.BusinessUnit
	if f.level == LevelPlan {
		businessUnit = f.businessUnitOf(values)
	}
	result.Missing = f.missing(values, businessUnit)
	for id, value := range values {
		result.Values[id] = value
	}
	return result
}

// normalize checks the shape of one value of the attribute a (data model §9) and returns it as it would be
// stored — trimmed, compact, nil when empty — or the problem with it.
func (f formData) normalize(a Attribute, raw json.RawMessage) (json.RawMessage, string) {
	if isEmptyValue(raw) {
		return nil, ""
	}
	choose := fmt.Sprintf(chooseOptionMessage, a.Label)
	switch a.Type {
	case TypeSingleChoice:
		var id string
		if json.Unmarshal(raw, &id) != nil || !f.isOptionOf(a, id) {
			return nil, choose
		}
		return jsonOf(id), ""
	case TypeMultipleChoice:
		var ids []string
		if json.Unmarshal(raw, &ids) != nil {
			return nil, choose
		}
		for i, id := range ids {
			if !f.isOptionOf(a, id) {
				return nil, choose
			}
			if slices.Contains(ids[:i], id) {
				return nil, onceMessage
			}
		}
		return jsonOf(ids), ""
	case TypeMultipleChoiceSplit:
		var shares []struct {
			OptionID string `json:"optionId"`
			Share    string `json:"share"`
		}
		if json.Unmarshal(raw, &shares) != nil {
			return nil, choose
		}
		stored := make([]SplitShare, 0, len(shares))
		for _, s := range shares {
			if !f.isOptionOf(a, s.OptionID) {
				return nil, choose
			}
			share, err := money.ParsePercent(strings.TrimSpace(s.Share))
			if err != nil || share.IsZero() {
				return nil, shareMessage
			}
			for _, earlier := range stored {
				if earlier.OptionID == s.OptionID {
					return nil, onceMessage
				}
			}
			stored = append(stored, SplitShare{OptionID: s.OptionID, Share: share})
		}
		return jsonOf(stored), ""
	case TypeYesNo:
		var yes bool
		if json.Unmarshal(raw, &yes) != nil {
			return nil, chooseYesOrNoMessage
		}
		return jsonOf(yes), ""
	case TypeText, TypeLongText:
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return nil, enterTextMessage
		}
		text = strings.TrimSpace(text)
		limit := maxTextValue
		if a.Type == TypeLongText {
			limit = maxLongTextValue
		}
		if text == "" {
			return nil, ""
		}
		if utf8.RuneCountInString(text) > limit {
			return nil, tooLongMessage(limit)
		}
		return jsonOf(text), ""
	case TypeNumber:
		number, err := strconv.ParseInt(string(bytes.TrimSpace(raw)), 10, 64)
		if err != nil || number < -maxNumberValue || number > maxNumberValue {
			return nil, wholeNumberMessage
		}
		return jsonOf(number), ""
	}
	return nil, choose
}

// isOptionOf reports whether id is an option of the attribute a.
func (f formData) isOptionOf(a Attribute, id string) bool {
	o, found := f.optionByID[id]
	return found && o.AttributeID == a.ID
}

// newOptionIDs returns the options of value that before did not have: the ones newly chosen.
func newOptionIDs(a Attribute, value, before json.RawMessage) []string {
	if !a.IsChoice() || value == nil {
		return nil
	}
	had := optionIDsOf(a, before)
	return filter(optionIDsOf(a, value), func(id string) bool { return !slices.Contains(had, id) })
}

// newInactiveOption returns the problem of an inactive option chosen newly in value (FR-004): one the record
// already had stays (FR-027).
func (f formData) newInactiveOption(a Attribute, value, before json.RawMessage) string {
	for _, id := range newOptionIDs(a, value, before) {
		if o := f.optionByID[id]; !o.Active {
			return fmt.Sprintf(inactiveOptionMessage, o.Label)
		}
	}
	return ""
}

// linked reports whether the option id is linked to any of the parent options selected.
func (f formData) linked(id string, selected []string) bool {
	for _, parentID := range f.optionByID[id].ParentOptionIDs {
		if slices.Contains(selected, parentID) {
			return true
		}
	}
	return false
}

// clearUnlinked returns the options of the dependent attribute a's value that no longer fit the parent
// options selected (FR-025).
func (f formData) clearUnlinked(a Attribute, value json.RawMessage, selected []string) ClearedValue {
	cleared := ClearedValue{AttributeID: a.ID, Label: a.Label, OptionIDs: []string{}, OptionLabels: []string{}}
	for _, id := range optionIDsOf(a, value) {
		if !f.linked(id, selected) {
			cleared.OptionIDs = append(cleared.OptionIDs, id)
			cleared.OptionLabels = append(cleared.OptionLabels, f.optionByID[id].Label)
		}
	}
	return cleared
}

// withoutOptions returns the value of the choice attribute a without the options removed, or nil when
// nothing is left.
func withoutOptions(a Attribute, value json.RawMessage, removed []string) json.RawMessage {
	keep := func(id string) bool { return !slices.Contains(removed, id) }
	switch a.Type {
	case TypeMultipleChoice:
		var ids []string
		_ = json.Unmarshal(value, &ids) // normalized already
		if kept := filter(ids, keep); len(kept) > 0 {
			return jsonOf(kept)
		}
	case TypeMultipleChoiceSplit:
		var shares []SplitShare
		_ = json.Unmarshal(value, &shares) // normalized already
		if kept := filter(shares, func(s SplitShare) bool { return keep(s.OptionID) }); len(kept) > 0 {
			return jsonOf(kept)
		}
	}
	return nil // a single choice whose option is removed has no value left
}

// missing lists the needed attributes still empty in values on a record whose plan's business unit is
// businessUnit (data model §9 "Missing").
func (f formData) missing(values map[string]json.RawMessage, businessUnit []string) []MissingField {
	missing := []MissingField{}
	for _, a := range f.attributes {
		if !a.Active || a.Marking == MarkingOptional || a.Type == TypeYesNo || !Visible(a, businessUnit) {
			continue
		}
		value := values[a.ID]
		field := MissingField{Field: "attributes." + a.ID, Label: a.Label}
		switch {
		case value == nil && !f.noOptionsForParent(a, values):
			missing = append(missing, field)
		case value != nil && a.Type == TypeMultipleChoiceSplit:
			var shares []SplitShare
			_ = json.Unmarshal(value, &shares) // normalized already
			total := money.PercentFromHundredths(0)
			for _, s := range shares {
				total = money.SumPercent(total, s.Share)
			}
			if total != money.Hundred {
				field.Reason = fmt.Sprintf(sharesTotalMessage, total.String())
				missing = append(missing, field)
			}
		}
	}
	return missing
}

// noOptionsForParent reports whether the dependent attribute a has no active option linked to any of the
// parent options chosen in values (D-034): it then cannot be filled in, so it is never missing.
func (f formData) noOptionsForParent(a Attribute, values map[string]json.RawMessage) bool {
	parent, dependent := f.byID[a.ParentAttributeID]
	if !dependent {
		return false
	}
	selected := optionIDsOf(parent, values[parent.ID])
	if len(selected) == 0 {
		return false
	}
	for _, o := range f.options[a.ID] {
		if o.Active && f.linked(o.ID, selected) {
			return false
		}
	}
	return true
}

// DescribeAttribute gives the label and text of a Shell attribute value in the activity log (data model
// §12): for the field "attributes.<id>", the attribute's label and its value's text — option labels joined
// with ", ", shares as "China 60.00 %, India 40.00 %", "Yes" or "No", the text or the number. Other fields
// give "" and "", meaning "use the default". The Describe functions of the planning kinds of 004–007 call
// it; value is the stored value decoded, as history.Kind.Describe receives it.
func (s *Service) DescribeAttribute(ctx context.Context, field string, value any) (label, text string, err error) {
	id, isAttribute := strings.CutPrefix(field, "attributes.")
	if !isAttribute {
		return "", "", nil
	}
	err = s.tx.Read(ctx, func(ctx context.Context) error {
		a, found, err := s.repo.GetAttribute(ctx, id)
		if err != nil || !found {
			return err
		}
		label = a.Label
		text, err = s.valueText(ctx, a, value)
		return err
	})
	return label, text, err
}

// valueText writes a decoded value of the attribute a as people read it; "" for an empty value.
func (s *Service) valueText(ctx context.Context, a Attribute, value any) (string, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return "Yes", nil
		}
		return "No", nil
	case string:
		if a.IsChoice() {
			return s.namesOf(ctx, v, s.optionLabel)
		}
		return v, nil
	case []any:
		if a.Type != TypeMultipleChoiceSplit {
			return s.namesOf(ctx, v, s.optionLabel)
		}
		texts := make([]string, 0, len(v))
		for _, item := range v {
			share, _ := item.(map[string]any)
			id, _ := share["optionId"].(string)
			label, err := s.namesOf(ctx, id, s.optionLabel)
			if err != nil {
				return "", err
			}
			texts = append(texts, strings.TrimSpace(label+" "+percentText(share["share"])))
		}
		return strings.Join(texts, ", "), nil
	case json.Number:
		return v.String(), nil
	}
	return "", nil
}
