package settings

import (
	"mapp/internal/platform/record"
)

// ShellAttributeView is a Shell attribute as the API returns it (contract ShellAttribute).
type ShellAttributeView struct {
	ID                string        `json:"id"`
	Label             string        `json:"label"`
	HelpText          string        `json:"helpText"`
	Level             Level         `json:"level"`
	Type              AttributeType `json:"type"`
	Marking           Marking       `json:"marking"`
	Section           string        `json:"section"`
	Position          int           `json:"position"`
	ParentAttribute   *SettingRef   `json:"parentAttribute"` // Depends on; null when none
	Condition         []SettingRef  `json:"condition"`       // Business unit options; empty when always shown
	BusinessUnit      bool          `json:"businessUnit"`    // the plan-level Business unit attribute
	Active            bool          `json:"active"`
	OptionCount       int           `json:"optionCount"`
	ActiveOptionCount int           `json:"activeOptionCount"`
	Usage             Usage         `json:"usage"`
	Locks             []Lock        `json:"locks"`
	record.Meta
}

// ShellAttributePage is the answer of GET /api/v1/shell-attributes?level= (contract ShellAttributePage), in
// display order.
type ShellAttributePage struct {
	Items []ShellAttributeView `json:"items"`
	Total int                  `json:"total"`
}

// NewOption is one first option of a new choice attribute (contract NewOption).
type NewOption struct {
	Label           string   `json:"label"`
	ParentOptionIDs []string `json:"parentOptionIds"` // dependent attributes: the parent options (at least one)
}

// ShellAttributeInput is the body of POST /api/v1/shell-attributes (contract ShellAttributeCreate). Level,
// type and marking are texts ("" when not chosen) so the service reports them as field problems.
type ShellAttributeInput struct {
	Label              string      `json:"label"`
	HelpText           string      `json:"helpText"`
	Level              string      `json:"level"`
	Type               string      `json:"type"`
	Marking            string      `json:"marking"`
	Section            string      `json:"section"`           // "" means DefaultSection
	ParentAttributeID  string      `json:"parentAttributeId"` // "" for none
	ConditionOptionIDs []string    `json:"conditionOptionIds"`
	Options            []NewOption `json:"options"`
}

// OptionLinks are the parent options one option is linked to, sent when an attribute becomes dependent
// (contract OptionLinks).
type OptionLinks struct {
	OptionID        string   `json:"optionId"`
	ParentOptionIDs []string `json:"parentOptionIds"`
}

// ShellAttributeUpdate is the body of PATCH /api/v1/shell-attributes/{attributeId} and of its impact check
// (contract ShellAttributeUpdate).
type ShellAttributeUpdate struct {
	Label              *string        `json:"label"`
	HelpText           *string        `json:"helpText"`
	Level              *string        `json:"level"`
	Type               *string        `json:"type"`
	Marking            *string        `json:"marking"`
	Section            *string        `json:"section"`
	ParentAttributeID  *string        `json:"parentAttributeId"`
	ConditionOptionIDs *[]string      `json:"conditionOptionIds"`
	OptionLinks        *[]OptionLinks `json:"optionLinks"`
	Version            *int           `json:"version"`
}

// What a move moves (MoveInput.What) and where (Direction).
const (
	MoveAttribute = "attribute"
	MoveSection   = "section"
	MoveUp        = "up"
	MoveDown      = "down"
)

// MoveInput is the body of POST /api/v1/shell-attributes/{attributeId}/move (contract AttributeMove).
type MoveInput struct {
	What      string `json:"what"`      // MoveAttribute or MoveSection
	Direction string `json:"direction"` // MoveUp or MoveDown
	Version   *int   `json:"version"`   // the version of the attribute the person started from
}

// OptionView is a Shell attribute option as the API returns it (contract ShellAttributeOption).
type OptionView struct {
	ID            string       `json:"id"`
	AttributeID   string       `json:"attributeId"`
	Label         string       `json:"label"`
	Position      int          `json:"position"`
	ParentOptions []SettingRef `json:"parentOptions"`
	Active        bool         `json:"active"`
	Usage         Usage        `json:"usage"`
	Locks         []Lock       `json:"locks"` // the last option of its attribute can't be deleted (D-211); never nil
	record.Meta
}

// OptionPage is the answer of GET /api/v1/shell-attributes/{attributeId}/options (contract
// ShellAttributeOptionPage); Total counts every match, also beyond the limit.
type OptionPage struct {
	Items []OptionView `json:"items"`
	Total int          `json:"total"`
}

// OptionFilter is the query of GET /api/v1/shell-attributes/{attributeId}/options.
type OptionFilter struct {
	Q      string // any part of the label, ignoring case
	Status string // "", "active" or "inactive"
	Limit  int    // 1–500, 100 when not sent
	Offset int    // 0 or more
}

// OptionInput is the body of POST /api/v1/shell-attributes/{attributeId}/options (contract OptionCreate).
type OptionInput struct {
	Label           string   `json:"label"`
	ParentOptionIDs []string `json:"parentOptionIds"`
}

// OptionUpdate is the body of PATCH /api/v1/shell-attribute-options/{optionId} (contract OptionUpdate).
type OptionUpdate struct {
	Label           *string   `json:"label"`
	ParentOptionIDs *[]string `json:"parentOptionIds"`
	Version         *int      `json:"version"`
}

// OptionMoveInput is the body of POST /api/v1/shell-attribute-options/{optionId}/move (contract OptionMove).
type OptionMoveInput struct {
	Direction string `json:"direction"`
	Version   *int   `json:"version"`
}

// ImportFile is the body of the option import check and import (contract OptionImportRequest): a .csv or
// .xlsx file of at most 700 KB, base64 encoded.
type ImportFile struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

// ImportRow is one row of an import check (contract ImportRow).
type ImportRow struct {
	Row         int    `json:"row"` // the header is row 1
	Label       string `json:"label"`
	ParentLabel string `json:"parentLabel"`
	Message     string `json:"message"` // why it is skipped or has a problem; "" for rows to add
}

// ImportCheck is the answer of the import check (contract OptionImportCheck).
type ImportCheck struct {
	TotalRows      int         `json:"totalRows"`
	ToAdd          []ImportRow `json:"toAdd"`
	AlreadyPresent []ImportRow `json:"alreadyPresent"`
	Problems       []ImportRow `json:"problems"`
}

// ImportResult is the answer of the import (contract OptionImportResult).
type ImportResult struct {
	Added          int `json:"added"`
	AlreadyPresent int `json:"alreadyPresent"`
	Problems       int `json:"problems"`
}
