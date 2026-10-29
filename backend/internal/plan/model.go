// Package plan holds plans (feature 004): one fiscal year of marketing for one business, the top of the
// planning tree Plan → Campaign → Initiative → Activity → Line item.
package plan

import (
	"mapp/internal/overspend"
	"mapp/internal/people"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// Status is a plan's status, changed only by Publish and Unpublish.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
)

type GoalStage string

const (
	StageINQ     GoalStage = "inq"
	StageMQL     GoalStage = "mql"
	StageRevenue GoalStage = "revenue"
)

type Spread string

const (
	SpreadEvenly     Spread = "evenly"
	SpreadFirstMonth Spread = "firstMonth"
	SpreadLastMonth  Spread = "lastMonth"
	SpreadCustom     Spread = "custom"
)

type ApprovalProgress string

const (
	ProgressNoActivities ApprovalProgress = "noActivities"
	ProgressInProgress   ApprovalProgress = "inProgress"
	ProgressAllApproved  ApprovalProgress = "allApproved"
)

type PlanType string

const (
	TypeOperational PlanType = "operational"
	TypeStrategic   PlanType = "strategic"
	TypeCrossMarket PlanType = "crossMarket"
)

var planTypes = []PlanType{TypeOperational, TypeStrategic, TypeCrossMarket}

const (
	ScopeAll     = "all"
	ScopeMyTeams = "myTeams"
)

// Plan is a stored plan; figures, goals and completeness are computed on read, never stored (D-041).
type Plan struct {
	ID                     string                   `json:"id"`
	Name                   string                   `json:"name"`
	Year                   int                      `json:"year"`
	ModelID                string                   `json:"modelId"`
	TeamIDs                []string                 `json:"teamIds"` // sorted by team name; never nil
	OwnerID                string                   `json:"ownerId"`
	AOP                    bool                     `json:"aop"`
	Description            string                   `json:"description"`
	Budget                 *money.Amount            `json:"budget"`
	DefaultFinanceBudgetID string                   `json:"defaultFinanceBudgetId"`
	DefaultSpread          Spread                   `json:"defaultSpread"`
	BCPIR                  bool                     `json:"bcpir"`
	GoalStage              GoalStage                `json:"goalStage"`
	GoalAmount             string                   `json:"goalAmount"` // "" = none; a whole number for inq and mql, a USD amount for revenue
	Attributes             settings.AttributeValues `json:"attributes"`
	Status                 Status                   `json:"status"`
	Currency               string                   `json:"currency"`
	PlanType               PlanType                 `json:"planType"`
	BudgetPhasing          rollup.Phasing           `json:"budgetPhasing"`
	Overspend              overspend.Mode           `json:"overspend"`
	record.Meta
}

// Normalised gives data stored before 016 its defaults: USD, the operational type and no phasing.
func (p Plan) Normalised() Plan {
	if p.Currency == "" {
		p.Currency = money.BaseCurrency
	}
	if p.PlanType == "" {
		p.PlanType = TypeOperational
	}
	p.BudgetPhasing = p.BudgetPhasing.Normalised()
	p.Overspend = p.Overspend.Normalised()
	return p
}

type ModelSummary struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Active          bool          `json:"active"`
	InqToMqlRate    money.Percent `json:"inqToMqlRate"`
	MqlToDealRate   money.Percent `json:"mqlToDealRate"`
	AverageDealSize money.Money   `json:"averageDealSize"`
}

type FinanceBudgetRef struct {
	ID   string `json:"id"`
	Year int    `json:"year"`
	Name string `json:"name"`
}

type OptionRef struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Active bool   `json:"active"`
}

// PlanView shows Figures in the reader's display currency, and PlanFigures and stored amounts in the plan currency.
type PlanView struct {
	ID                   string                   `json:"id"`
	Name                 string                   `json:"name"`
	Year                 int                      `json:"year"`
	Status               Status                   `json:"status"`
	Model                ModelSummary             `json:"model"`
	Teams                []people.TeamRef         `json:"teams"`
	Owner                *people.PersonRef        `json:"owner"`
	OwnerProblem         string                   `json:"ownerProblem"`
	AOP                  bool                     `json:"aop"`
	Description          string                   `json:"description"`
	Budget               *money.Money             `json:"budget"`
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
	Allocation           *PlanAllocation          `json:"allocation"`
	Access               *authz.PlanAccess        `json:"access,omitempty"`
	Currency             string                   `json:"currency"`
	PlanType             PlanType                 `json:"planType"`
	BudgetPhasing        rollup.PhasingView       `json:"budgetPhasing"`
	PlanFigures          Figures                  `json:"planFigures"`
	CurrencyLock         *string                  `json:"currencyLock"`
	Overspend            overspend.Control        `json:"overspend"`
	record.Meta
}

type Defaults struct {
	CurrentYear   int            `json:"currentYear"`
	Years         []int          `json:"years"` // current … current + 10 (D-326)
	Year          int            `json:"year"`
	TeamIDs       []string       `json:"teamIds"`
	OwnerID       *string        `json:"ownerId"`
	ModelID       *string        `json:"modelId"` // the only active model; null when none or several
	AOP           bool           `json:"aop"`
	BCPIR         bool           `json:"bcpir"`
	DefaultSpread Spread         `json:"defaultSpread"`
	GoalStage     GoalStage      `json:"goalStage"`
	Currency      string         `json:"currency"`
	PlanType      PlanType       `json:"planType"`
	Overspend     overspend.Mode `json:"overspend"`
}

// PlanInput takes the enums as texts and the budget as a money input, so every problem becomes a field message.
type PlanInput struct {
	Name                   string                   `json:"name"`
	Year                   *int                     `json:"year"`
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
	Currency               *string                  `json:"currency"`
	PlanType               *string                  `json:"planType"`
	Overspend              *string                  `json:"overspend"`
}

// PlanUpdate carries the version the person started from (D-203); a null currency or plan type keeps the plan's.
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
	Currency               *string                  `json:"currency"`
	PlanType               *string                  `json:"planType"`
	Overspend              *string                  `json:"overspend"`
}

type UnpublishRequest struct {
	Reason string `json:"reason"`
}

type NextYearOption struct {
	Year      int    `json:"year"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

type CopyOptions struct {
	Name     string         `json:"name"`
	SameYear int            `json:"sameYear"`
	NextYear NextYearOption `json:"nextYear"`
	Contents Contents       `json:"contents"`
}

type CopyRequest struct {
	Name            string `json:"name"`
	Year            *int   `json:"year"`
	IncludeContents bool   `json:"includeContents"`
}

type CopyResult struct {
	Plan  PlanView `json:"plan"`
	Notes []string `json:"notes"`
}

type ChangeEffectsRequest struct {
	ModelID    *string                  `json:"modelId"`
	Attributes settings.AttributeValues `json:"attributes"`
}

type AttributeGroupChange struct {
	Unit       string   `json:"unit"`
	Levels     []string `json:"levels"`
	Attributes []string `json:"attributes"`
}

type ChangeEffects struct {
	ModelChanged         bool                   `json:"modelChanged"`
	DisallowedActivities []errs.RecordRef       `json:"disallowedActivities"`
	HiddenGroups         []AttributeGroupChange `json:"hiddenGroups"`
	ShownGroups          []AttributeGroupChange `json:"shownGroups"`
}

type References struct {
	PlanID, Name           string
	ModelID                string
	OwnerID                string
	DefaultFinanceBudgetID string
	TeamIDs                []string
	Attributes             settings.AttributeValues
	InRecycleBin           bool
	Currency               string
	Budget                 *money.Amount
}
