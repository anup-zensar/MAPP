package settings

import (
	"encoding/json"

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
