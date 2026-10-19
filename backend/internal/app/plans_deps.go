package app

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/people"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// The adapters below give plans what it reads from Settings (002) and from people and teams (003): the
// plans.Settings and plans.Directory interfaces (004 plan "Dependencies as interfaces", D-196).
//
// They read those features' repositories directly, because their services were built at the same time as
// plans. Their methods have the names and signatures of the readers 002 and 003 publish, so once those
// services exist, internal/app passes them to plans instead and these adapters are removed (004 report,
// cross-lane needs). Only the plan level's needs are covered: the Shell attribute check knows the value
// shapes, options and inactive rule of 002 data model §9, but not dependent attributes, which have no
// plan-level example.

// settingsForPlans reads Settings for plans.
type settingsForPlans struct {
	repo settings.Repository
}

// Model returns the model with id.
func (s settingsForPlans) Model(ctx context.Context, id string) (settings.Model, bool, error) {
	return s.repo.GetModel(ctx, id)
}

// Models returns every model, sorted by name.
func (s settingsForPlans) Models(ctx context.Context) ([]settings.Model, error) {
	models, err := s.repo.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(models, func(x, y settings.Model) int {
		return cmp.Or(strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), strings.Compare(x.ID, y.ID))
	})
	return models, nil
}

// FinanceBudget returns the finance budget with id.
func (s settingsForPlans) FinanceBudget(ctx context.Context, id string) (settings.FinanceBudget, bool, error) {
	return s.repo.GetFinanceBudget(ctx, id)
}

// Attributes returns the Shell attributes of level in display order.
func (s settingsForPlans) Attributes(ctx context.Context, level settings.Level) ([]settings.Attribute, error) {
	all, err := s.repo.ListAttributes(ctx)
	if err != nil {
		return nil, err
	}
	ofLevel := []settings.Attribute{}
	for _, a := range all {
		if a.Level == level {
			ofLevel = append(ofLevel, a)
		}
	}
	slices.SortFunc(ofLevel, func(x, y settings.Attribute) int {
		return cmp.Or(cmp.Compare(x.Position, y.Position), strings.Compare(x.ID, y.ID))
	})
	return ofLevel, nil
}

// Option returns the option with id.
func (s settingsForPlans) Option(ctx context.Context, id string) (settings.Option, bool, error) {
	return s.repo.GetOption(ctx, id)
}

// BusinessUnit returns the option ids of the plan's Business unit value.
func (s settingsForPlans) BusinessUnit(ctx context.Context, planValues settings.AttributeValues) ([]string, error) {
	attributes, err := s.Attributes(ctx, settings.LevelPlan)
	if err != nil {
		return nil, err
	}
	for _, a := range attributes {
		if a.Role == settings.RoleBusinessUnit {
			return optionIDs(planValues[a.ID]), nil
		}
	}
	return []string{}, nil
}

// optionIDs reads the option ids of a choice value: one id or a list of ids; never nil.
func optionIDs(value json.RawMessage) []string {
	var list []string
	if json.Unmarshal(value, &list) == nil && list != nil {
		return list
	}
	var one string
	if json.Unmarshal(value, &one) == nil && one != "" {
		return []string{one}
	}
	return []string{}
}

// MissingAttributes lists the needed attributes of level that are active, shown and empty.
func (s settingsForPlans) MissingAttributes(ctx context.Context, level settings.Level, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error) {
	if level == settings.LevelPlan {
		var err error
		if businessUnit, err = s.BusinessUnit(ctx, values); err != nil {
			return nil, err
		}
	}
	attributes, err := s.Attributes(ctx, level)
	if err != nil {
		return nil, err
	}
	missing := []settings.MissingField{}
	for _, a := range attributes {
		needed := a.Active && a.Marking != settings.MarkingOptional && a.Type != settings.TypeYesNo
		if needed && visible(a, businessUnit) && emptyValue(values[a.ID]) {
			missing = append(missing, settings.MissingField{Field: "attributes." + a.ID, Label: a.Label})
		}
	}
	return missing, nil
}

// visible reports whether a is shown for the business unit: it has no condition, or the business unit
// includes one of its options (002 FR-023).
func visible(a settings.Attribute, businessUnit []string) bool {
	return len(a.Condition) == 0 || slices.ContainsFunc(a.Condition, func(id string) bool { return slices.Contains(businessUnit, id) })
}

// emptyValue reports whether a stored value counts as no value.
func emptyValue(value json.RawMessage) bool {
	switch strings.TrimSpace(string(value)) {
	case "", "null", `""`, "[]":
		return true
	}
	return false
}

// The messages of 002's attribute check (002 plan "Check results").
const (
	attributeGoneMessage   = "This Shell attribute no longer exists — reload the page."
	otherLevelMessage      = "%s belongs to the %s form."
	chooseOptionMessage    = "Choose one of the options of %s."
	inactiveOptionMessage  = "'%s' is inactive — choose another option."
	optionTwiceMessage     = "Each option can appear only once."
	inactiveChangeMessage  = "%s is inactive and can't be changed."
	enterTextMessage       = "Enter text."
	textTooLongMessage     = "Use 120 characters or fewer."
	longTextTooLongMessage = "Use 2,000 characters or fewer."
	yesNoMessage           = "Choose Yes or No."
	numberMessage          = "Enter a whole number between -999,999,999,999 and 999,999,999,999."
	maxAttributeNumber     = 999_999_999_999
)

// CheckAttributes checks the values of a form of level: problems, the values to store and what is still
// missing.
func (s settingsForPlans) CheckAttributes(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error) {
	result := settings.AttributeResult{
		Problems: []errs.FieldError{}, Cleared: []settings.ClearedValue{}, Values: settings.AttributeValues{},
	}
	for _, id := range slices.Sorted(maps.Keys(in.Values)) {
		field := "attributes." + id
		a, found, err := s.repo.GetAttribute(ctx, id)
		if err != nil {
			return settings.AttributeResult{}, err
		}
		switch {
		case !found:
			result.Problems = append(result.Problems, errs.FieldError{Field: field, Message: attributeGoneMessage})
			continue
		case a.Level != level:
			result.Problems = append(result.Problems, errs.FieldError{Field: field, Message: fmt.Sprintf(otherLevelMessage, a.Label, levelWord(a.Level))})
			continue
		}
		value, message, err := s.checkValue(ctx, a, in.Values[id], in.Previous[id])
		if err != nil {
			return settings.AttributeResult{}, err
		}
		if message == "" && !a.Active && !sameValue(value, in.Previous[id]) {
			message = fmt.Sprintf(inactiveChangeMessage, a.Label)
		}
		if message != "" {
			result.Problems = append(result.Problems, errs.FieldError{Field: field, Message: message})
		} else if value != nil {
			result.Values[id] = value
		}
	}
	missing, err := s.MissingAttributes(ctx, level, result.Values, in.BusinessUnit)
	if err != nil {
		return settings.AttributeResult{}, err
	}
	result.Missing = missing
	return result, nil
}

// levelWord is a level as a sentence names it.
func levelWord(level settings.Level) string {
	if level == settings.LevelLineItem {
		return "line item"
	}
	return string(level)
}

// sameValue reports whether two stored values are the same, empty values counting as equal.
func sameValue(x, y json.RawMessage) bool {
	if emptyValue(x) || emptyValue(y) {
		return emptyValue(x) && emptyValue(y)
	}
	var compactX, compactY bytes.Buffer
	if json.Compact(&compactX, x) != nil || json.Compact(&compactY, y) != nil {
		return false
	}
	return bytes.Equal(compactX.Bytes(), compactY.Bytes())
}

// checkValue checks one value against its attribute's type and returns it as it is stored (nil for no
// value) or the problem's message.
func (s settingsForPlans) checkValue(ctx context.Context, a settings.Attribute, value, previous json.RawMessage) (json.RawMessage, string, error) {
	if emptyValue(value) {
		return nil, "", nil
	}
	switch a.Type {
	case settings.TypeSingleChoice, settings.TypeMultipleChoice:
		var ids []string
		if a.Type == settings.TypeSingleChoice {
			var id string
			if json.Unmarshal(value, &id) != nil {
				return nil, fmt.Sprintf(chooseOptionMessage, a.Label), nil
			}
			ids = []string{id}
		} else if json.Unmarshal(value, &ids) != nil {
			return nil, fmt.Sprintf(chooseOptionMessage, a.Label), nil
		}
		message, err := s.checkOptions(ctx, a, ids, optionIDs(previous))
		if err != nil || message != "" {
			return nil, message, err
		}
		return value, "", nil
	case settings.TypeYesNo:
		var yes bool
		if json.Unmarshal(value, &yes) != nil {
			return nil, yesNoMessage, nil
		}
		return value, "", nil
	case settings.TypeText, settings.TypeLongText:
		var text string
		if json.Unmarshal(value, &text) != nil {
			return nil, enterTextMessage, nil
		}
		text = strings.TrimSpace(text)
		switch {
		case text == "":
			return nil, "", nil
		case a.Type == settings.TypeText && utf8.RuneCountInString(text) > 120:
			return nil, textTooLongMessage, nil
		case utf8.RuneCountInString(text) > 2000:
			return nil, longTextTooLongMessage, nil
		}
		trimmed, err := json.Marshal(text)
		return trimmed, "", err
	case settings.TypeNumber:
		var n int64
		if json.Unmarshal(value, &n) != nil || n < -maxAttributeNumber || n > maxAttributeNumber {
			return nil, numberMessage, nil
		}
		return value, "", nil
	}
	return nil, fmt.Sprintf(chooseOptionMessage, a.Label), nil // a % split is not a plan-level type
}

// checkOptions checks chosen option ids: each once, an option of the attribute, and active unless it was
// already chosen.
func (s settingsForPlans) checkOptions(ctx context.Context, a settings.Attribute, ids, previous []string) (string, error) {
	for i, id := range ids {
		if slices.Contains(ids[:i], id) {
			return optionTwiceMessage, nil
		}
		o, found, err := s.repo.GetOption(ctx, id)
		if err != nil {
			return "", err
		}
		switch {
		case !found || o.AttributeID != a.ID:
			return fmt.Sprintf(chooseOptionMessage, a.Label), nil
		case !o.Active && !slices.Contains(previous, id):
			return fmt.Sprintf(inactiveOptionMessage, o.Label), nil
		}
	}
	return "", nil
}

// DescribeAttribute gives the label of a field "attributes.<id>" and the text of its value: option labels
// for choices; "" for anything else, which the log writes with its defaults.
func (s settingsForPlans) DescribeAttribute(ctx context.Context, field string, value any) (label, text string, err error) {
	id, isAttribute := strings.CutPrefix(field, "attributes.")
	if !isAttribute {
		return "", "", nil
	}
	a, found, err := s.repo.GetAttribute(ctx, id)
	if err != nil || !found {
		return "", "", err
	}
	if !a.IsChoice() {
		return a.Label, "", nil
	}
	var ids []any
	switch v := value.(type) {
	case string:
		ids = []any{v}
	case []any:
		ids = v
	}
	labels := []string{}
	for _, item := range ids {
		optionID, _ := item.(string)
		o, found, err := s.repo.GetOption(ctx, optionID)
		if err != nil {
			return "", "", err
		}
		if !found {
			o.Label = optionID // an option that no longer exists is shown by its id
		}
		labels = append(labels, o.Label)
	}
	return a.Label, strings.Join(labels, ", "), nil
}

// directoryForPlans reads people and teams for plans, and the display currency's rate from Settings.
type directoryForPlans struct {
	people   people.Repository
	settings settings.Repository
}

// PersonRefs returns the people with the ids; unknown ids are left out.
func (d directoryForPlans) PersonRefs(ctx context.Context, ids []string) (map[string]people.PersonRef, error) {
	refs := map[string]people.PersonRef{}
	for _, id := range ids {
		p, found, err := d.people.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if found {
			refs[id] = p.Ref()
		}
	}
	return refs, nil
}

// TeamRefs returns the teams with the ids; unknown ids are left out.
func (d directoryForPlans) TeamRefs(ctx context.Context, ids []string) (map[string]people.TeamRef, error) {
	refs := map[string]people.TeamRef{}
	for _, id := range ids {
		t, found, err := d.people.GetTeam(ctx, id)
		if err != nil {
			return nil, err
		}
		if found {
			refs[id] = t.Ref()
		}
	}
	return refs, nil
}

// MemberIDs returns the ids of each team's members.
func (d directoryForPlans) MemberIDs(ctx context.Context, teamIDs []string) (map[string][]string, error) {
	members := map[string][]string{}
	for _, teamID := range teamIDs {
		memberships, err := d.people.MembershipsOfTeam(ctx, teamID)
		if err != nil {
			return nil, err
		}
		for _, m := range memberships {
			members[teamID] = append(members[teamID], m.PersonID)
		}
	}
	return members, nil
}

// TeamIDsOf returns the ids of the person's teams.
func (d directoryForPlans) TeamIDsOf(ctx context.Context, personID string) ([]string, error) {
	memberships, err := d.people.MembershipsOfPerson(ctx, personID)
	if err != nil {
		return nil, err
	}
	teamIDs := []string{}
	for _, m := range memberships {
		teamIDs = append(teamIDs, m.TeamID)
	}
	return teamIDs, nil
}

// DisplayRateOf returns the person's display currency and its current rate; USD and "1" when they chose
// none or the currency is unknown.
func (d directoryForPlans) DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error) {
	usd := people.DisplayRate{Code: money.BaseCurrency, Rate: money.One.String()}
	prefs, found, err := d.people.GetPreferences(ctx, personID)
	if err != nil || !found || prefs.DisplayCurrency == "" || prefs.DisplayCurrency == money.BaseCurrency {
		return usd, err
	}
	c, found, err := d.settings.GetCurrency(ctx, prefs.DisplayCurrency)
	if err != nil || !found {
		return usd, err
	}
	return people.DisplayRate{Code: c.Code, Rate: c.Rate.String()}, nil
}
