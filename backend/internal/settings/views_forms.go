package settings

import (
	"mapp/internal/platform/errs"
)

// FormOption is an option as forms need it (contract FormOption).
type FormOption struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Active          bool     `json:"active"`
	ParentOptionIDs []string `json:"parentOptionIds"`
}

// FormAttribute is an attribute as forms need it (contract FormAttribute).
type FormAttribute struct {
	ID                 string        `json:"id"`
	Label              string        `json:"label"`
	HelpText           string        `json:"helpText"`
	Type               AttributeType `json:"type"`
	Marking            Marking       `json:"marking"`
	Active             bool          `json:"active"`
	ParentAttributeID  *string       `json:"parentAttributeId"` // null when it depends on nothing
	ConditionOptionIDs []string      `json:"conditionOptionIds"`
	Options            []FormOption  `json:"options"` // display order, inactive ones included
}

// FormSection is one section of a level's form (contract FormSection).
type FormSection struct {
	Name        string          `json:"name"`
	Conditional bool            `json:"conditional"` // every attribute of the section has a condition
	Attributes  []FormAttribute `json:"attributes"`
}

// FormView is the answer of GET /api/v1/shell-attribute-forms/{level} (contract AttributeForm).
type FormView struct {
	Level                   Level         `json:"level"`
	Sections                []FormSection `json:"sections"`
	BusinessUnitAttributeID *string       `json:"businessUnitAttributeId"` // null when none exists
	BusinessUnitOptions     []FormOption  `json:"businessUnitOptions"`
}

// CheckRequest is the body of POST /api/v1/shell-attribute-forms/{level}/check (contract
// AttributeCheckRequest).
type CheckRequest struct {
	Values                AttributeValues `json:"values"`
	PreviousValues        AttributeValues `json:"previousValues"`
	BusinessUnitOptionIDs []string        `json:"businessUnitOptionIds"`
}

// CheckResult is the answer of the check (contract AttributeCheckResult).
type CheckResult struct {
	Problems []errs.FieldError `json:"problems"`
	Missing  []MissingField    `json:"missing"`
	Cleared  []ClearedValue    `json:"cleared"`
	Values   AttributeValues   `json:"values"`
}
