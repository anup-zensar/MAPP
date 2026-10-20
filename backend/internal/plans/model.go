// Package plans holds plans (feature 004): one fiscal year of marketing for one business, the top of the
// planning tree Plan → Campaign → Initiative → Activity → Line item.
//
// The shared foundation wrote the data layer: the stored plan and the API types (this file), the level
// registry types the features below the plan implement (levels.go) and the repository (repository.go,
// store_json.go). Feature 004 adds the service, the Settings and Directory interfaces with their adapters,
// the history kind and the handler.
package plans

import (
	"northstar/internal/people"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/settings"
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
	Budget                 *money.Amount            `json:"budget"`                 // USD; nil = none
	DefaultFinanceBudgetID string                   `json:"defaultFinanceBudgetId"` // a finance budget of the plan's year; "" = none
	DefaultSpread          Spread                   `json:"defaultSpread"`          // the default of new line items
	BCPIR                  bool                     `json:"bcpir"`                  // BCPIR plan (Q-001)
	GoalStage              GoalStage                `json:"goalStage"`
	GoalAmount             string                   `json:"goalAmount"` // "" = none; a whole number for inq and mql, a USD amount for revenue
	Attributes             settings.AttributeValues `json:"attributes"` // plan-level Shell attribute values, Business unit included; never nil
	Status                 Status                   `json:"status"`
	record.Meta
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

// Completeness is the plan's own missing needed-to-publish fields and every record below that is not
// complete (Completeness, D-199).
type Completeness struct {
	Missing    []settings.MissingField `json:"missing"`
	ItemsBelow []AttentionItem         `json:"itemsBelow"`
}

// Contents counts what is inside the plan (PlanContents).
type Contents struct {
	Campaigns          int `json:"campaigns"`
	Initiatives        int `json:"initiatives"`
	Activities         int `json:"activities"`
	ApprovedActivities int `json:"approvedActivities"` // Approved, In progress or Complete
	LineItems          int `json:"lineItems"`
}

// Figures are the plan's calculated money and progress in the reader's display currency (PlanFigures).
type Figures struct {
	Currency           string           `json:"currency"` // the display currency of every amount below
	Budget             *money.Money     `json:"budget"`
	PlannedCost        *money.Money     `json:"plannedCost"` // null when nothing below has a planned cost
	CampaignBudgets    money.Money      `json:"campaignBudgets"`
	RemainingBudget    *money.Money     `json:"remainingBudget"`   // null without a plan budget
	OverAllocatedBy    *money.Money     `json:"overAllocatedBy"`   // null unless campaign budgets exceed the plan budget
	BudgetUsedPercent  *int             `json:"budgetUsedPercent"` // whole percent; null without a budget or with 0
	OverBudget         bool             `json:"overBudget"`
	Activities         int              `json:"activities"`
	ActivitiesApproved int              `json:"activitiesApproved"`
	ApprovalProgress   ApprovalProgress `json:"approvalProgress"`
}

// GoalReasons say why a goal stage could not be calculated although a goal amount is set; "" otherwise.
type GoalReasons struct {
	INQ     string `json:"inq,omitempty"`
	MQL     string `json:"mql,omitempty"`
	Deals   string `json:"deals,omitempty"`
	Revenue string `json:"revenue,omitempty"`
}

// GoalFigures are the plan goal at every stage: the typed stage as typed, the others calculated
// (GoalFigures).
type GoalFigures struct {
	INQ     *int64       `json:"inq"`
	MQL     *int64       `json:"mql"`
	Deals   *int64       `json:"deals"`
	Revenue *money.Money `json:"revenue"` // in the display currency
	Reasons GoalReasons  `json:"reasons"`
}

// GoalTotals are the sums of the goals of every activity of the plan (GoalTotals). In Facts the revenue
// is in USD; in the API it is in the reader's display currency.
type GoalTotals struct {
	INQ     int64       `json:"inq"`
	MQL     int64       `json:"mql"`
	Revenue money.Money `json:"revenue"`
}

// Goals are the plan's goals and the activities' goals (PlanGoals).
type Goals struct {
	Stage          GoalStage   `json:"stage"`
	Plan           GoalFigures `json:"plan"`
	Activities     GoalTotals  `json:"activities"`
	PlannedPercent *int        `json:"plannedPercent"` // null without a goal or with a goal of 0
}

// PlanView is a plan as the API returns it (Plan): stored amounts in USD, calculated values in the
// display currency.
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
	Budget               *money.Money             `json:"budget"` // USD, as stored
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
	Allocation           *PlanAllocation          `json:"allocation"` // the global budget that sets the budget (014); null when none
	record.Meta
}

// PlanSummary is a plan as the Plans list shows it (PlanSummary).
type PlanSummary struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Year         int               `json:"year"`
	Status       Status            `json:"status"`
	AOP          bool              `json:"aop"`
	Owner        *people.PersonRef `json:"owner"`
	Teams        []people.TeamRef  `json:"teams"`
	BusinessUnit []OptionRef       `json:"businessUnit"`
	Figures      Figures           `json:"figures"`
	Completeness Completeness      `json:"completeness"`
}

// ListSummary is the summary cards of the Plans page (PlanListSummary).
type ListSummary struct {
	Plans       int          `json:"plans"`
	Draft       int          `json:"draft"`
	Published   int          `json:"published"`
	PlannedCost *money.Money `json:"plannedCost"` // null when no plan has one
	Budget      *money.Money `json:"budget"`      // null when no plan has one
}

// PlanList is the answer of GET /api/plans (PlanList).
type PlanList struct {
	Items   []PlanSummary `json:"items"`
	Total   int           `json:"total"` // every plan in the team scope
	Summary ListSummary   `json:"summary"`
	Years   []int         `json:"years"` // years with a plan in the scope, newest first
}

// ListFilter is the query of GET /api/plans.
type ListFilter struct {
	Scope  string // ScopeAll or ScopeMyTeams
	Status string // "" (every status), "draft" or "published"
	Year   *int
	Q      string // contained in the name or ID, ignoring case
}

// Defaults are what a new plan starts with (PlanDefaults, GET /api/plan-defaults).
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
}

// PlanInput is the body of POST /api/plans (PlanCreate): the whole form. The budget is a money input and
// the enums are texts, so that every problem becomes a field message.
type PlanInput struct {
	Name                   string                   `json:"name"`
	Year                   *int                     `json:"year"`
	ModelID                *string                  `json:"modelId"`
	TeamIDs                []string                 `json:"teamIds"`
	OwnerID                *string                  `json:"ownerId"`
	AOP                    bool                     `json:"aop"`
	Description            string                   `json:"description"`
	Budget                 *money.Input             `json:"budget"` // USD; null for no budget
	DefaultFinanceBudgetID *string                  `json:"defaultFinanceBudgetId"`
	DefaultSpread          string                   `json:"defaultSpread"`
	BCPIR                  bool                     `json:"bcpir"`
	GoalStage              string                   `json:"goalStage"`
	GoalAmount             *string                  `json:"goalAmount"`
	Attributes             settings.AttributeValues `json:"attributes"`
}

// PlanUpdate is the body of PATCH /api/plans/{planId} (PlanUpdate): the whole form without the year, and
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
}

// UnpublishRequest is the body of POST /api/plans/{planId}/unpublish (UnpublishRequest).
type UnpublishRequest struct {
	Reason string `json:"reason"` // 1–500 characters after trimming
}

// NextYearOption says whether a copy may go to the next year (NextYearOption).
type NextYearOption struct {
	Year      int    `json:"year"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"` // "" when available
}

// CopyOptions is the answer of GET /api/plans/{planId}/copy-options (PlanCopyOptions).
type CopyOptions struct {
	Name     string         `json:"name"` // the suggested "<name> (copy)"
	SameYear int            `json:"sameYear"`
	NextYear NextYearOption `json:"nextYear"`
	Contents Contents       `json:"contents"`
}

// CopyRequest is the body of POST /api/plans/{planId}/copy (PlanCopyRequest).
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

// ChangeEffectsRequest is the body of POST /api/plans/{planId}/change-effects (PlanChangeEffectsRequest).
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
}
