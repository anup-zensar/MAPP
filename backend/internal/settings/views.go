package settings

import (
	"encoding/json"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
)

// The API types below are what the API returns and receives, with the JSON names of the contract
// (backend/api/openapi.yaml, feature 002's schemas, which allow no other properties). They are kept apart
// from the stored records of model.go (001 research R10). Lists are never nil. Updates send only the fields
// to change (D-203): a field left out is nil; Version is a pointer so a missing version can be told apart.

// SettingRef is a Settings record referred to by another one, with its status (contract SettingRef).
type SettingRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// UsagePart is one line of a record's "Used by" (contract UsagePart, data model §10).
type UsagePart struct {
	Kind         string `json:"kind"`         // plan, campaign, initiative, activity, lineItem, person, model, dependentAttribute, condition, linkedOption
	Text         string `json:"text"`         // "Used by 12 activities (1 in the recycle bin)"
	Count        int    `json:"count"`        // records using it, those in the recycle bin included
	InRecycleBin int    `json:"inRecycleBin"` // of which in the recycle bin
}

// Usage is where a Settings record is used (contract Usage, FR-003).
type Usage struct {
	InUse   bool                `json:"inUse"`   // anything uses it: it cannot be deleted, only set inactive
	Summary string              `json:"summary"` // the parts joined with " · ", or "Not used yet"
	Parts   []UsagePart         `json:"parts"`
	Records []history.RecordRef `json:"records"` // models and finance budgets: the live plans using it, by name, at most 50
}

// Lock is a field or action that cannot be used, and why (contract Lock).
type Lock struct {
	Field  string `json:"field"` // type, level, code, rate, year, section, position, condition, parentAttributeId, active, delete
	Reason string `json:"reason"`
}

// The kinds of impact warnings (contract ImpactWarning.kind).
const (
	ImpactGoalsRecalculated   = "goalsRecalculated"
	ImpactActivityTypeRemoved = "activityTypeRemoved"
	ImpactBecomesIncomplete   = "becomesIncomplete"
	ImpactOptionsRemoved      = "optionsRemoved"
	ImpactAmountsRecalculated = "amountsRecalculated"
)

// ImpactWarning is one statement shown before a change is saved (contract ImpactWarning, D-044).
type ImpactWarning struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// ChangeImpact is the answer of every POST …/impact (contract ChangeImpact).
type ChangeImpact struct {
	Warnings []ImpactWarning `json:"warnings"`
}

// StatusUpdate is the body of POST …/status: Set inactive (false) or Reactivate (true), with the version
// the person started from (contract StatusUpdate).
type StatusUpdate struct {
	Active  *bool `json:"active"`
	Version *int  `json:"version"`
}

// SettingsOverview is the number of active records of each Settings section (GET /api/settings/overview).
type SettingsOverview struct {
	Models             int `json:"models"`
	ActivityTypes      int `json:"activityTypes"`
	ShellAttributes    int `json:"shellAttributes"`
	Currencies         int `json:"currencies"`
	FinanceBudgetItems int `json:"financeBudgetItems"`
}

// OptionalMoney is a money field of an update that may be left out, sent as null (remove the value) or
// sent with an amount: Set tells whether it was sent, Value is nil for null.
type OptionalMoney struct {
	Set   bool
	Value *money.Input
}

// UnmarshalJSON records that the field was sent, and its amount unless it is null.
func (o *OptionalMoney) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var in money.Input
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	o.Value = &in
	return nil
}

// Models.

// ModelView is a model as the API returns it (contract Model).
type ModelView struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	InqToMqlRate    money.Percent `json:"inqToMqlRate"`
	MqlToDealRate   money.Percent `json:"mqlToDealRate"`
	AverageDealSize money.Money   `json:"averageDealSize"`
	ActivityTypes   []SettingRef  `json:"activityTypes"` // the allowed types, by name
	Active          bool          `json:"active"`
	Usage           Usage         `json:"usage"`
	record.Meta
}

// ModelPage is the answer of GET /api/models (contract ModelPage).
type ModelPage struct {
	Items []ModelView `json:"items"`
	Total int         `json:"total"`
}

// ModelInput is the body of POST /api/models (contract ModelCreate). Rates are percent texts ("" when not
// filled in) and the deal size a money input (nil when not filled in), so problems become field messages.
type ModelInput struct {
	Name            string       `json:"name"`
	InqToMqlRate    string       `json:"inqToMqlRate"`
	MqlToDealRate   string       `json:"mqlToDealRate"`
	AverageDealSize *money.Input `json:"averageDealSize"`
	ActivityTypeIDs []string     `json:"activityTypeIds"`
}

// ModelUpdate is the body of PATCH /api/models/{modelId} and of its impact check (contract ModelUpdate).
type ModelUpdate struct {
	Name            *string       `json:"name"`
	InqToMqlRate    *string       `json:"inqToMqlRate"`
	MqlToDealRate   *string       `json:"mqlToDealRate"`
	AverageDealSize OptionalMoney `json:"averageDealSize"`
	ActivityTypeIDs *[]string     `json:"activityTypeIds"`
	Version         *int          `json:"version"`
}

// Activity types.

// ActivityTypeView is an activity type as the API returns it (contract ActivityType).
type ActivityTypeView struct {
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	DefaultPlannedCost *money.Money `json:"defaultPlannedCost"` // USD; null when none
	Models             []SettingRef `json:"models"`             // the models that allow it, by name
	Active             bool         `json:"active"`
	Usage              Usage        `json:"usage"`
	record.Meta
}

// ActivityTypePage is the answer of GET /api/activity-types (contract ActivityTypePage).
type ActivityTypePage struct {
	Items []ActivityTypeView `json:"items"`
	Total int                `json:"total"`
}

// ActivityTypeInput is the body of POST /api/activity-types (contract ActivityTypeCreate).
type ActivityTypeInput struct {
	Name               string       `json:"name"`
	DefaultPlannedCost *money.Input `json:"defaultPlannedCost"`
	ModelIDs           []string     `json:"modelIds"` // the models that allow the new type
}

// ActivityTypeUpdate is the body of PATCH /api/activity-types/{activityTypeId} and of its impact check
// (contract ActivityTypeUpdate). DefaultPlannedCost sent as null removes the default; ModelIDs, when sent,
// is the complete new list of models allowing the type.
type ActivityTypeUpdate struct {
	Name               *string       `json:"name"`
	DefaultPlannedCost OptionalMoney `json:"defaultPlannedCost"`
	ModelIDs           *[]string     `json:"modelIds"`
	Version            *int          `json:"version"`
}

// Shell attributes and options.

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

// ShellAttributePage is the answer of GET /api/shell-attributes?level= (contract ShellAttributePage), in
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

// ShellAttributeInput is the body of POST /api/shell-attributes (contract ShellAttributeCreate). Level,
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

// ShellAttributeUpdate is the body of PATCH /api/shell-attributes/{attributeId} and of its impact check
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

// MoveInput is the body of POST /api/shell-attributes/{attributeId}/move (contract AttributeMove).
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

// OptionPage is the answer of GET /api/shell-attributes/{attributeId}/options (contract
// ShellAttributeOptionPage); Total counts every match, also beyond the limit.
type OptionPage struct {
	Items []OptionView `json:"items"`
	Total int          `json:"total"`
}

// OptionFilter is the query of GET /api/shell-attributes/{attributeId}/options.
type OptionFilter struct {
	Q      string // any part of the label, ignoring case
	Status string // "", "active" or "inactive"
	Limit  int    // 1–500, 100 when not sent
	Offset int    // 0 or more
}

// OptionInput is the body of POST /api/shell-attributes/{attributeId}/options (contract OptionCreate).
type OptionInput struct {
	Label           string   `json:"label"`
	ParentOptionIDs []string `json:"parentOptionIds"`
}

// OptionUpdate is the body of PATCH /api/shell-attribute-options/{optionId} (contract OptionUpdate).
type OptionUpdate struct {
	Label           *string   `json:"label"`
	ParentOptionIDs *[]string `json:"parentOptionIds"`
	Version         *int      `json:"version"`
}

// OptionMoveInput is the body of POST /api/shell-attribute-options/{optionId}/move (contract OptionMove).
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

// Forms and values.

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

// FormView is the answer of GET /api/shell-attribute-forms/{level} (contract AttributeForm).
type FormView struct {
	Level                   Level         `json:"level"`
	Sections                []FormSection `json:"sections"`
	BusinessUnitAttributeID *string       `json:"businessUnitAttributeId"` // null when none exists
	BusinessUnitOptions     []FormOption  `json:"businessUnitOptions"`
}

// CheckRequest is the body of POST /api/shell-attribute-forms/{level}/check (contract
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

// Currencies.

// CurrencyView is a currency as the API returns it (contract Currency).
type CurrencyView struct {
	Code          string      `json:"code"`
	Name          string      `json:"name"`
	Symbol        string      `json:"symbol"`
	Rate          money.Rate  `json:"rate"`
	InverseRate   money.Rate  `json:"inverseRate"`
	RateChangedAt time.Time   `json:"rateChangedAt"`
	RateChangedBy actor.Actor `json:"rateChangedBy"`
	Active        bool        `json:"active"`
	Usage         Usage       `json:"usage"`
	Locks         []Lock      `json:"locks"`
	record.Meta
}

// CurrencyPage is the answer of GET /api/currencies (contract CurrencyPage): USD first, then by code.
type CurrencyPage struct {
	Items []CurrencyView `json:"items"`
	Total int            `json:"total"`
}

// CurrencyInput is the body of POST /api/currencies (contract CurrencyCreate); the rate is text ("" when
// not filled in).
type CurrencyInput struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Rate   string `json:"rate"`
}

// CurrencyUpdate is the body of PATCH /api/currencies/{code} and of its impact check (contract
// CurrencyUpdate). The code never changes.
type CurrencyUpdate struct {
	Name    *string `json:"name"`
	Symbol  *string `json:"symbol"`
	Rate    *string `json:"rate"`
	Version *int    `json:"version"`
}

// Finance budgets and items.

// FinanceBudgetItemView is a finance budget item as the API returns it (contract FinanceBudgetItem).
type FinanceBudgetItemView struct {
	ID         string `json:"id"`
	BudgetID   string `json:"budgetId"`
	BudgetName string `json:"budgetName"`
	Year       int    `json:"year"`
	Name       string `json:"name"`
	Code       string `json:"code"`
	Active     bool   `json:"active"`
	Usage      Usage  `json:"usage"`
	Locks      []Lock `json:"locks"`
	record.Meta
}

// FinanceBudgetView is a finance budget with its items (contract FinanceBudget).
type FinanceBudgetView struct {
	ID              string                  `json:"id"`
	Year            int                     `json:"year"`
	Name            string                  `json:"name"`
	Items           []FinanceBudgetItemView `json:"items"` // active and inactive, by name
	ActiveItemCount int                     `json:"activeItemCount"`
	Usage           Usage                   `json:"usage"`
	record.Meta
}

// FinanceBudgetPage is the answer of GET /api/finance-budgets?year= (contract FinanceBudgetPage).
type FinanceBudgetPage struct {
	Items []FinanceBudgetView `json:"items"` // the year's budgets with at least one item, by name
	Total int                 `json:"total"`
	Year  int                 `json:"year"`  // the year shown
	Years []int               `json:"years"` // current − 1 … + 5 and any other year with items, ascending
}

// RenameInput is the body of PATCH /api/finance-budgets/{budgetId} (contract FinanceBudgetRename).
type RenameInput struct {
	Name    string `json:"name"`
	Version *int   `json:"version"`
}

// ItemInput is the body of POST /api/finance-budget-items (contract FinanceBudgetItemCreate): the budget
// is found by year and name, or created.
type ItemInput struct {
	Year       *int   `json:"year"`
	BudgetName string `json:"budgetName"`
	Name       string `json:"name"`
	Code       string `json:"code"`
}

// ItemUpdate is the body of PATCH /api/finance-budget-items/{itemId} (contract FinanceBudgetItemUpdate). A
// new budget name moves the item to that budget of the same year.
type ItemUpdate struct {
	BudgetName *string `json:"budgetName"`
	Name       *string `json:"name"`
	Code       *string `json:"code"`
	Version    *int    `json:"version"`
}
