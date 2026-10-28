package plan

import (
	"mapp/internal/people"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/money"
)

type PlanSummary struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Year            int               `json:"year"`
	Status          Status            `json:"status"`
	AOP             bool              `json:"aop"`
	BCPIR           bool              `json:"bcpir"`
	Owner           *people.PersonRef `json:"owner"`
	Teams           []people.TeamRef  `json:"teams"`
	BusinessUnit    []OptionRef       `json:"businessUnit"`
	Figures         Figures           `json:"figures"`
	Completeness    Completeness      `json:"completeness"`
	Access          *authz.PlanAccess `json:"access,omitempty"`
	PlanType        PlanType          `json:"planType"`
	Currency        string            `json:"currency"`
	PlanBudget      *money.Money      `json:"planBudget"`
	PlanPlannedCost *money.Money      `json:"planPlannedCost"`
}

type ListSummary struct {
	Plans       int          `json:"plans"`
	Draft       int          `json:"draft"`
	Published   int          `json:"published"`
	PlannedCost *money.Money `json:"plannedCost"`
	Budget      *money.Money `json:"budget"`
}

type PlanList struct {
	Items   []PlanSummary `json:"items"`
	Total   int           `json:"total"`
	Summary ListSummary   `json:"summary"`
	Years   []int         `json:"years"`
}

type ListFilter struct {
	Scope            string
	Status           string
	Year             *int
	Q                string
	YearFrom, YearTo *int
	Types            []string
	BCPIR            string
}
