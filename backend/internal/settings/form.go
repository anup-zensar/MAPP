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
