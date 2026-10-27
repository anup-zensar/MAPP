package settings

import (
	"encoding/json"

	"mapp/internal/history"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
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

// SettingsOverview is the number of active records of each Settings section (GET /api/v1/settings/overview).
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

// ModelPage is the answer of GET /api/v1/models (contract ModelPage).
type ModelPage struct {
	Items []ModelView `json:"items"`
	Total int         `json:"total"`
}

// ModelInput is the body of POST /api/v1/models (contract ModelCreate). Rates are percent texts ("" when not
// filled in) and the deal size a money input (nil when not filled in), so problems become field messages.
type ModelInput struct {
	Name            string       `json:"name"`
	InqToMqlRate    string       `json:"inqToMqlRate"`
	MqlToDealRate   string       `json:"mqlToDealRate"`
	AverageDealSize *money.Input `json:"averageDealSize"`
	ActivityTypeIDs []string     `json:"activityTypeIds"`
}

// ModelUpdate is the body of PATCH /api/v1/models/{modelId} and of its impact check (contract ModelUpdate).
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

// ActivityTypePage is the answer of GET /api/v1/activity-types (contract ActivityTypePage).
type ActivityTypePage struct {
	Items []ActivityTypeView `json:"items"`
	Total int                `json:"total"`
}

// ActivityTypeInput is the body of POST /api/v1/activity-types (contract ActivityTypeCreate).
type ActivityTypeInput struct {
	Name               string       `json:"name"`
	DefaultPlannedCost *money.Input `json:"defaultPlannedCost"`
	ModelIDs           []string     `json:"modelIds"` // the models that allow the new type
}

// ActivityTypeUpdate is the body of PATCH /api/v1/activity-types/{activityTypeId} and of its impact check
// (contract ActivityTypeUpdate). DefaultPlannedCost sent as null removes the default; ModelIDs, when sent,
// is the complete new list of models allowing the type.
type ActivityTypeUpdate struct {
	Name               *string       `json:"name"`
	DefaultPlannedCost OptionalMoney `json:"defaultPlannedCost"`
	ModelIDs           *[]string     `json:"modelIds"`
	Version            *int          `json:"version"`
}
