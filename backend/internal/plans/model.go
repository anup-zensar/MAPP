// Package plans holds plans (feature 004): one fiscal year of marketing for one business, the top of the
// planning tree Plan → Campaign → Initiative → Activity → Line item.
//
// The shared foundation wrote the data layer: the stored plan and the API types (this file), the level
// registry types the features below the plan implement (levels.go) and the repository (repository.go,
// store_json.go). Feature 004 adds the service, the Settings and Directory interfaces with their adapters,
// the history kind and the handler.
package plans

import (
	"mapp/internal/people"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// Status is a plan's status (domain model §2.1): changed only by Publish and Unpublish.
type Status string

// The plan statuses (contract PlanStatus).
const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
)

// GoalStage is the stage at which the plan's goal is typed (contract GoalStage).
type GoalStage string

// The goal stages.
const (
	StageINQ     GoalStage = "inq"
	StageMQL     GoalStage = "mql"
	StageRevenue GoalStage = "revenue"
)

// Spread is how a line item's planned cost is spread over the months — the plan's default line-item
// spread (contract SpreadMethod; feature 007 uses the same values).
type Spread string

// The spread methods.
const (
	SpreadEvenly     Spread = "evenly"
	SpreadFirstMonth Spread = "firstMonth"
	SpreadLastMonth  Spread = "lastMonth"
	SpreadCustom     Spread = "custom"
)

// ApprovalProgress is how far the activities of a plan are approved (contract ApprovalProgress, D-026).
type ApprovalProgress string

// The approval progress states.
const (
	ProgressNoActivities ApprovalProgress = "noActivities"
	ProgressInProgress   ApprovalProgress = "inProgress"
	ProgressAllApproved  ApprovalProgress = "allApproved"
)

// PlanType is a plan's type (contract PlanType, 016 FR-032, D-268); the Strategic / AOP flag stays.
type PlanType string

// The plan types.
const (
	TypeOperational PlanType = "operational" // the default
	TypeStrategic   PlanType = "strategic"
	TypeCrossMarket PlanType = "crossMarket"
)

// planTypes are the plan types in the order of the form.
var planTypes = []PlanType{TypeOperational, TypeStrategic, TypeCrossMarket}

// The team scopes of the Plans list (ListFilter.Scope).
const (
	ScopeAll     = "all"
	ScopeMyTeams = "myTeams"
)

// Plan is a plan as stored in the table "plans" (data model §1). Every field, rule and message is in the
// data model; derived values (figures, goals, completeness) are computed on read, never stored (D-041).
type Plan struct {
	ID                     string                   `json:"id"`                     // "P-1001"
	Name                   string                   `json:"name"`                   // 1–120 characters, unique among plans ignoring case
	Year                   int                      `json:"year"`                   // never changes after the first save
	ModelID                string                   `json:"modelId"`                // a model of feature 002
	TeamIDs                []string                 `json:"teamIds"`                // teams of feature 003, sorted by team name; never nil
	OwnerID                string                   `json:"ownerId"`                // a person; "" = none
	AOP                    bool                     `json:"aop"`                    // Strategic / AOP plan
	Description            string                   `json:"description"`            // 0–2,000 characters
	Budget                 *money.Amount            `json:"budget"`                 // in the plan currency (016); nil = none
	DefaultFinanceBudgetID string                   `json:"defaultFinanceBudgetId"` // a finance budget of the plan's year; "" = none
	DefaultSpread          Spread                   `json:"defaultSpread"`          // the default of new line items
	BCPIR                  bool                     `json:"bcpir"`                  // BCPIR plan (Q-001)
	GoalStage              GoalStage                `json:"goalStage"`
	GoalAmount             string                   `json:"goalAmount"` // "" = none; a whole number for inq and mql, a USD amount for revenue
	Attributes             settings.AttributeValues `json:"attributes"` // plan-level Shell attribute values, Business unit included; never nil
	Status                 Status                   `json:"status"`
	// 016: the plan currency every budget, phasing and planned cost of the plan is in (FR-011; "" in older
	// data reads as USD), the plan type (FR-032; "" reads as operational) and the plan budget's phasing
	// (FR-020; missing reads as not phased).
	Currency      string         `json:"currency"`
	PlanType      PlanType       `json:"planType"`
	BudgetPhasing rollup.Phasing `json:"budgetPhasing"`
	record.Meta
}

// Normalised returns p with the defaults of data stored before 016 (data-model §4.5).
func (p Plan) Normalised() Plan {
	if p.Currency == "" {
		p.Currency = money.BaseCurrency
	}
	if p.PlanType == "" {
		p.PlanType = TypeOperational
	}
	p.BudgetPhasing = p.BudgetPhasing.Normalised()
	return p
}

// The API types below are what the API returns and receives (contract schema in brackets, JSON names as
// the contract). Lists are never nil.

// ModelSummary is the plan's model with what the goal waterfall needs (ModelSummary).
type ModelSummary struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Active          bool          `json:"active"`
	InqToMqlRate    money.Percent `json:"inqToMqlRate"`
	MqlToDealRate   money.Percent `json:"mqlToDealRate"`
	AverageDealSize money.Money   `json:"averageDealSize"`
}

// FinanceBudgetRef is the plan's default finance budget (FinanceBudgetRef); finance budgets have no status.
type FinanceBudgetRef struct {
	ID   string `json:"id"`
	Year int    `json:"year"`
	Name string `json:"name"`
}

// OptionRef is a Shell attribute option a record refers to, such as a business unit (OptionRef).
type OptionRef struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Active bool   `json:"active"`
}

// PlanView is a plan as the API returns it (Plan): stored amounts in the plan currency, calculated values
// in the display currency (figures) and in the plan currency (planFigures, 016 FR-013).
type PlanView struct {
	ID                   string                   `json:"id"`
	Name                 string                   `json:"name"`
	Year                 int                      `json:"year"`
	Status               Status                   `json:"status"`
	Model                ModelSummary             `json:"model"`
	Teams                []people.TeamRef         `json:"teams"` // by name; inactive ones kept and flagged
	Owner                *people.PersonRef        `json:"owner"`
	OwnerProblem         string                   `json:"ownerProblem"` // the R1 message; "" otherwise
	AOP                  bool                     `json:"aop"`
	Description          string                   `json:"description"`
	Budget               *money.Money             `json:"budget"` // in the plan currency, as stored
	DefaultFinanceBudget *FinanceBudgetRef        `json:"defaultFinanceBudget"`
	DefaultSpread        Spread                   `json:"defaultSpread"`
	BCPIR                bool                     `json:"bcpir"`
	GoalStage            GoalStage                `json:"goalStage"`
	GoalAmount           *string                  `json:"goalAmount"`
	Attributes           settings.AttributeValues `json:"attributes"`
	BusinessUnit         []OptionRef              `json:"businessUnit"`
	Figures              Figures                  `json:"figures"`
	Goals                Goals                    `json:"goals"`
	Completeness         Completeness             `json:"completeness"`
	Contents             Contents                 `json:"contents"`
	Allocation           *PlanAllocation          `json:"allocation"`       // the global budget that sets the budget (014); null when none
	Access               *authz.PlanAccess        `json:"access,omitempty"` // what the reader may do (014 FR-011)
	Currency             string                   `json:"currency"`         // 016 FR-011
	PlanType             PlanType                 `json:"planType"`         // 016 FR-032
	BudgetPhasing        rollup.PhasingView       `json:"budgetPhasing"`    // 016 FR-020, in the plan currency
	PlanFigures          Figures                  `json:"planFigures"`      // 016 FR-013: figures in the plan currency
	CurrencyLock         *string                  `json:"currencyLock"`     // 016 FR-012: why the currency can't change; null when it can
	record.Meta
}

// Defaults are what a new plan starts with (PlanDefaults, GET /api/v1/plan-defaults).
type Defaults struct {
	CurrentYear   int       `json:"currentYear"`
	Years         []int     `json:"years"` // current − 1 … current + 5
	Year          int       `json:"year"`
	TeamIDs       []string  `json:"teamIds"`
	OwnerID       *string   `json:"ownerId"`
	ModelID       *string   `json:"modelId"` // the only active model; null when none or several
	AOP           bool      `json:"aop"`
	BCPIR         bool      `json:"bcpir"`
	DefaultSpread Spread    `json:"defaultSpread"`
	GoalStage     GoalStage `json:"goalStage"`
	Currency      string    `json:"currency"` // 016: USD
	PlanType      PlanType  `json:"planType"` // 016: operational
}

// PlanInput is the body of POST /api/v1/plans (PlanCreate): the whole form. The budget is a money input and
// the enums are texts, so that every problem becomes a field message.
type PlanInput struct {
	Name                   string                   `json:"name"`
	Year                   *int                     `json:"year"`
	ModelID                *string                  `json:"modelId"`
	TeamIDs                []string                 `json:"teamIds"`
	OwnerID                *string                  `json:"ownerId"`
	AOP                    bool                     `json:"aop"`
	Description            string                   `json:"description"`
	Budget                 *money.Input             `json:"budget"` // in the plan currency; null for no budget
	DefaultFinanceBudgetID *string                  `json:"defaultFinanceBudgetId"`
	DefaultSpread          string                   `json:"defaultSpread"`
	BCPIR                  bool                     `json:"bcpir"`
	GoalStage              string                   `json:"goalStage"`
	GoalAmount             *string                  `json:"goalAmount"`
	Attributes             settings.AttributeValues `json:"attributes"`
	Currency               *string                  `json:"currency"` // 016: the plan currency; null or missing = USD
	PlanType               *string                  `json:"planType"` // 016: null or missing = operational
}

// PlanUpdate is the body of PATCH /api/v1/plans/{planId} (PlanUpdate): the whole form without the year, and
// the version the person started from (D-203).
type PlanUpdate struct {
	Version                *int                     `json:"version"`
	Name                   string                   `json:"name"`
	ModelID                *string                  `json:"modelId"`
	TeamIDs                []string                 `json:"teamIds"`
	OwnerID                *string                  `json:"ownerId"`
	AOP                    bool                     `json:"aop"`
	Description            string                   `json:"description"`
	Budget                 *money.Input             `json:"budget"`
	DefaultFinanceBudgetID *string                  `json:"defaultFinanceBudgetId"`
	DefaultSpread          string                   `json:"defaultSpread"`
	BCPIR                  bool                     `json:"bcpir"`
	GoalStage              string                   `json:"goalStage"`
	GoalAmount             *string                  `json:"goalAmount"`
	Attributes             settings.AttributeValues `json:"attributes"`
	Currency               *string                  `json:"currency"` // 016: null or missing keeps the plan currency
	PlanType               *string                  `json:"planType"` // 016: null or missing keeps the plan type
}

// UnpublishRequest is the body of POST /api/v1/plans/{planId}/unpublish (UnpublishRequest).
type UnpublishRequest struct {
	Reason string `json:"reason"` // 1–500 characters after trimming
}

// NextYearOption says whether a copy may go to the next year (NextYearOption).
type NextYearOption struct {
	Year      int    `json:"year"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"` // "" when available
}

// CopyOptions is the answer of GET /api/v1/plans/{planId}/copy-options (PlanCopyOptions).
type CopyOptions struct {
	Name     string         `json:"name"` // the suggested "<name> (copy)"
	SameYear int            `json:"sameYear"`
	NextYear NextYearOption `json:"nextYear"`
	Contents Contents       `json:"contents"`
}

// CopyRequest is the body of POST /api/v1/plans/{planId}/copy (PlanCopyRequest).
type CopyRequest struct {
	Name            string `json:"name"`
	Year            *int   `json:"year"` // the source plan's year or the next one
	IncludeContents bool   `json:"includeContents"`
}

// CopyResult is the answer of the copy (PlanCopyResult).
type CopyResult struct {
	Plan  PlanView `json:"plan"`
	Notes []string `json:"notes"` // what the copy changed on purpose
}

// ChangeEffectsRequest is the body of POST /api/v1/plans/{planId}/change-effects (PlanChangeEffectsRequest).
type ChangeEffectsRequest struct {
	ModelID    *string                  `json:"modelId"`
	Attributes settings.AttributeValues `json:"attributes"`
}

// AttributeGroupChange is the Shell attributes tied to one business-unit option that a change would hide or
// show (AttributeGroupChange).
type AttributeGroupChange struct {
	Unit       string   `json:"unit"`       // the business-unit option's label, "Lubricants"
	Levels     []string `json:"levels"`     // where its attributes are, top-down: "plan", "campaign", …
	Attributes []string `json:"attributes"` // their labels, in display order
}

// ChangeEffects is the answer of the change-effects check (PlanChangeEffects).
type ChangeEffects struct {
	ModelChanged         bool                   `json:"modelChanged"`
	DisallowedActivities []errs.RecordRef       `json:"disallowedActivities"`
	HiddenGroups         []AttributeGroupChange `json:"hiddenGroups"`
	ShownGroups          []AttributeGroupChange `json:"shownGroups"`
}

// References is what one plan uses, for "Used by" counts of features 002 and 003 (Go only).
type References struct {
	PlanID, Name           string
	ModelID                string
	OwnerID                string
	DefaultFinanceBudgetID string
	TeamIDs                []string
	Attributes             settings.AttributeValues
	InRecycleBin           bool
	Currency               string        // 016: the plan currency (Settings' "Used by", FR-010)
	Budget                 *money.Amount // 016: the plan budget in Currency (the rate impact, FR-005)
}
