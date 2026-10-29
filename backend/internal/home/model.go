// Package home is the read-only Home dashboard (feature 009): every figure is calculated on the server, so it
// equals the one on the record's own page (FR-017), and each card has its own endpoint (FR-022).
package home

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

type Scope string

const (
	ScopeMyTeams    Scope = "myTeams"
	ScopeEverything Scope = "everything"
)

const chooseScopeMessage = "Choose My teams or Everything."

func ParseScope(text string) (Scope, error) {
	switch scope := Scope(text); scope {
	case ScopeMyTeams, ScopeEverything:
		return scope, nil
	}
	return "", errs.Validation(errs.FieldError{Field: "scope", Message: chooseScopeMessage})
}

// Every amount is in the person's display currency. Lists are never nil.

type SummaryView struct {
	Scope           Scope            `json:"scope"`
	DisplayCurrency string           `json:"displayCurrency"` // an inactive one stays in use (D-063, D-201)
	GettingStarted  *string          `json:"gettingStarted"`
	Plans           PlanCounts       `json:"plans"`
	ActivitiesOwned ActivityCounts   `json:"activitiesOwned"`
	WaitingForMe    WaitingCounts    `json:"waitingForMe"`
	PlannedCost     PlannedCostTotal `json:"plannedCost"`
}

type PlanCounts struct {
	Count     int `json:"count"`
	Published int `json:"published"`
	Draft     int `json:"draft"`
}

type ActivityCounts struct {
	Count     int  `json:"count"`
	Declined  int  `json:"declined"`
	Submitted int  `json:"submitted"`
	YearFrom  *int `json:"yearFrom"`
	YearTo    *int `json:"yearTo"`
}

type WaitingCounts struct {
	Count              int        `json:"count"`
	OldestWaitingSince *time.Time `json:"oldestWaitingSince"`
}

type PlannedCostTotal struct {
	PlanCount          int          `json:"planCount"`
	PlannedCost        money.Money  `json:"plannedCost"`
	PlanBudget         *money.Money `json:"planBudget"`
	BudgetUsedPercent  *int         `json:"budgetUsedPercent"`
	OverBudget         bool         `json:"overBudget"`
	PlansWithoutBudget int          `json:"plansWithoutBudget"`
}

type TeamView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Colour string `json:"colour"`
	Active bool   `json:"active"`
	Mine   bool   `json:"mine"`
}

type PlanRow struct {
	ID                    string       `json:"id"`
	Name                  string       `json:"name"`
	Year                  int          `json:"year"`
	StrategicAOP          bool         `json:"strategicAop"`
	Status                string       `json:"status"`
	Teams                 []TeamView   `json:"teams"`
	CampaignCount         int          `json:"campaignCount"`
	ActivityCount         int          `json:"activityCount"`
	ApprovedActivityCount int          `json:"approvedActivityCount"`
	Budget                *money.Money `json:"budget"`      // null: "—" ("No budget")
	PlannedCost           *money.Money `json:"plannedCost"` // null when nothing is planned, as on the plan page (D-217)
	BudgetUsedPercent     *int         `json:"budgetUsedPercent"`
	OverBudget            bool         `json:"overBudget"`
}

type PlanPage struct {
	Items []PlanRow `json:"items"`
	Total int       `json:"total"`
}

type HomeAttentionReason struct {
	Kind       string         `json:"kind"`
	Amount     *money.Money   `json:"amount"` // negativeBalance, overAllocated, overBudget: how much over
	DeclinedBy *actor.Actor   `json:"declinedBy"`
	DeclinedAt *time.Time     `json:"declinedAt"`
	Comment    *string        `json:"comment"`
	Missing    []MissingField `json:"missing"`
	ItemsBelow int            `json:"itemsBelow"`
}

type HomeAttentionItem struct {
	RecordType      string                `json:"recordType"`
	RecordTypeLabel string                `json:"recordTypeLabel"`
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Plan            errs.RecordRef        `json:"plan"`
	Path            []errs.RecordRef      `json:"path"` // ancestors, plan first; empty for a plan
	ChangedAt       time.Time             `json:"changedAt"`
	Reasons         []HomeAttentionReason `json:"reasons"`
}

type HomeAttentionPage struct {
	Items []HomeAttentionItem `json:"items"`
	Total int                 `json:"total"`
}

type WaitingItem struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Plan         errs.RecordRef `json:"plan"`
	Owner        *actor.Actor   `json:"owner"`
	PlannedCost  *money.Money   `json:"plannedCost"`
	Level        int            `json:"level"`
	Levels       int            `json:"levels"`
	WaitingSince time.Time      `json:"waitingSince"`
}

type WaitingPage struct {
	Items []WaitingItem `json:"items"`
	Total int           `json:"total"`
}

type MyActivityRow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Plan        errs.RecordRef `json:"plan"`
	Campaign    errs.RecordRef `json:"campaign"`
	Initiative  errs.RecordRef `json:"initiative"`
	Status      string         `json:"status"`
	StartDate   *date.Date     `json:"startDate"`
	EndDate     *date.Date     `json:"endDate"`
	Ongoing     bool           `json:"ongoing"`
	PlannedCost *money.Money   `json:"plannedCost"`
	Missing     []MissingField `json:"missing"`
	ItemsBelow  int            `json:"itemsBelow"`
}

type MyActivityGroup struct {
	Status string          `json:"status"`
	Count  int             `json:"count"`
	Items  []MyActivityRow `json:"items"`
}

type MyActivitiesView struct {
	Groups []MyActivityGroup `json:"groups"`
	Total  int               `json:"total"`
}

type RecentChangesPage struct {
	Items []any `json:"items"`
	Total int   `json:"total"`
}
