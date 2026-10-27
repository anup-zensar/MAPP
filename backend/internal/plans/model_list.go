package plans

import (
	"mapp/internal/people"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/money"
)

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
	Access       *authz.PlanAccess `json:"access,omitempty"` // what the reader may do (014 FR-011)
	// 016: the type and currency, and the budget and planned cost in the plan currency (FR-014, FR-032).
	PlanType        PlanType     `json:"planType"`
	Currency        string       `json:"currency"`
	PlanBudget      *money.Money `json:"planBudget"`
	PlanPlannedCost *money.Money `json:"planPlannedCost"`
}

// ListSummary is the summary cards of the Plans page (PlanListSummary).
type ListSummary struct {
	Plans       int          `json:"plans"`
	Draft       int          `json:"draft"`
	Published   int          `json:"published"`
	PlannedCost *money.Money `json:"plannedCost"` // null when no plan has one
	Budget      *money.Money `json:"budget"`      // null when no plan has one
}

// PlanList is the answer of GET /api/v1/plans (PlanList).
type PlanList struct {
	Items   []PlanSummary `json:"items"`
	Total   int           `json:"total"` // every plan in the team scope
	Summary ListSummary   `json:"summary"`
	Years   []int         `json:"years"` // years with a plan in the scope, newest first
}

// ListFilter is the query of GET /api/v1/plans.
type ListFilter struct {
	Scope  string // ScopeAll or ScopeMyTeams
	Status string // "" (every status), "draft" or "published"
	Year   *int   // the alias of YearFrom = YearTo = Year (016 D-269)
	Q      string // contained in the name or ID, ignoring case
	// 016 FR-033: the years from–to (inclusive; nil = no bound) and the plan types (empty = every type).
	YearFrom, YearTo *int
	Types            []string
}
