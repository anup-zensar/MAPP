package settings

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// AttributeValues are the Shell attribute values of one planning record — a plan, campaign, initiative,
// activity or line item — kept inside that record in its field "attributes" (D-048, data model §9): the
// attribute id → its value. Never nil on a stored record ({} when empty). The value's JSON follows the
// attribute's type:
//
//	singleChoice         "SO-1040"                                   option id
//	multipleChoice       ["SO-1010", "SO-1011"]                      distinct option ids
//	multipleChoiceSplit  [{"optionId": "SO-1050", "share": "60.00"}] SplitShare list
//	yesNo                true                                        missing reads as No
//	text, longText       "FIT4-0042"
//	number               1200                                        whole number
type AttributeValues map[string]json.RawMessage

// SplitShare is one option of a multiple-choice-with-%-split value, with its share above 0.00 (R10). Any
// total may be saved; only a total of exactly 100.00 % counts as filled in.
type SplitShare struct {
	OptionID string        `json:"optionId"`
	Share    money.Percent `json:"share"`
}

// MissingField is a field that is still needed to publish or submit — the one MissingField of the Plan
// module, defined in errs so that not_ready errors carry it too (D-193). Reason is optional.
type MissingField = errs.MissingField

// AttributeInput is what CheckAttributes checks: the values a form sends, the values the record had before
// (Previous: an inactive attribute or option already there may stay, FR-027) and the plan's business unit
// (option ids of the Business unit attribute; ignored at plan level, where Values hold it).
type AttributeInput struct {
	Values, Previous AttributeValues
	BusinessUnit     []string
}

// AttributeResult is the answer of CheckAttributes (data model §9).
type AttributeResult struct {
	Problems []errs.FieldError // field "attributes.<id>"; they block saving
	Cleared  []ClearedValue    // dependent values removed because their parent value changed (FR-025)
	Missing  []MissingField    // needed attributes still empty; they block publishing or submitting
	Values   AttributeValues   // the values to store: trimmed, empty and cleared values removed
}

// ClearedValue is a dependent attribute's value that no longer fits its changed parent value and is
// removed on save (FR-025), as the API returns it (contract schema ClearedValue).
type ClearedValue struct {
	AttributeID  string   `json:"attributeId"`
	Label        string   `json:"label"`
	OptionIDs    []string `json:"optionIds"`
	OptionLabels []string `json:"optionLabels"`
}

// isEmptyValue reports whether a stored value is no value: missing, null, "", [] or {} (data model §9).
func isEmptyValue(raw json.RawMessage) bool {
	switch string(bytes.TrimSpace(raw)) {
	case "", "null", `""`, "[]", "{}":
		return true
	}
	return false
}

// optionIDsOf returns the option ids in the value raw of the choice attribute a, each once: the option of
// a single choice, the options of a multiple choice, the options of a % split. A value of another shape
// has none.
func optionIDsOf(a Attribute, raw json.RawMessage) []string {
	if isEmptyValue(raw) {
		return []string{}
	}
	switch a.Type {
	case TypeSingleChoice:
		var id string
		if json.Unmarshal(raw, &id) == nil {
			return unique([]string{id})
		}
	case TypeMultipleChoice:
		var list []string
		if json.Unmarshal(raw, &list) == nil {
			return unique(list)
		}
	case TypeMultipleChoiceSplit:
		var shares []struct {
			OptionID string `json:"optionId"`
		}
		if json.Unmarshal(raw, &shares) == nil {
			list := make([]string, 0, len(shares))
			for _, share := range shares {
				list = append(list, share.OptionID)
			}
			return unique(list)
		}
	}
	return []string{}
}

// Diff compares two versions of a planning record that keeps Shell attribute values in its field
// "attributes" (D-048) and returns every changed field, sorted by field name, for history.Log (FR-006). It is
// history.Diff for every field except "attributes", which is compared attribute by attribute: one change
// per attribute, with the field "attributes.<attributeId>", which DescribeAttribute describes. Empty values
// (missing, null, "", [] or {}) are all the same.
func Diff(before, after any) ([]history.Change, error) {
	oldFields, oldValues, err := splitAttributes(before)
	if err != nil {
		return nil, err
	}
	newFields, newValues, err := splitAttributes(after)
	if err != nil {
		return nil, err
	}
	changes, err := history.Diff(oldFields, newFields)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for id := range oldValues {
		ids[id] = true
	}
	for id := range newValues {
		ids[id] = true
	}
	for _, id := range slices.SortedFunc(maps.Keys(ids), compareIDs) {
		oldValue, newValue := compactJSON(oldValues[id]), compactJSON(newValues[id])
		if bytes.Equal(oldValue, newValue) || (isEmptyValue(oldValue) && isEmptyValue(newValue)) {
			continue
		}
		changes = append(changes, history.Change{Field: "attributes." + id, Old: orNull(oldValue), New: orNull(newValue)})
	}
	slices.SortStableFunc(changes, func(a, b history.Change) int { return strings.Compare(a.Field, b.Field) })
	return changes, nil
}

// splitAttributes encodes record as a JSON object and returns its fields without "attributes", and the
// attribute values that field held.
func splitAttributes(record any) (map[string]json.RawMessage, AttributeValues, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, nil, err
	}
	var values AttributeValues
	if raw, found := fields["attributes"]; found {
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, nil, err
		}
		delete(fields, "attributes")
	}
	return fields, values, nil
}

// compactJSON returns raw without spaces, so that equal values compare equal; nil for no value.
func compactJSON(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	var b bytes.Buffer
	if err := json.Compact(&b, raw); err != nil {
		return raw
	}
	return b.Bytes()
}

// orNull returns raw, or JSON null for no value.
func orNull(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return json.RawMessage("null")
	}
	return raw
}
