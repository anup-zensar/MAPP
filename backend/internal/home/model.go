// Package home is the Home dashboard (feature 009): the summary cards, Plans at a glance, Needs your
// attention, Waiting for my approval and My activities. It is read-only and stores nothing (009 plan,
// D-171): it reads the plans, campaigns, initiatives, activities and approvals of features 003–008
// through Sources and calculates every figure on the server, so each equals the one on the record's own
// page (FR-017). Each card has its own endpoint, so one failing card never hides the others (FR-022).
package home

import (
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// Scope is Home's My teams / Everything switch (FR-003, contract HomeScope).
type Scope string

// The scopes. My teams covers the plans with at least one of the person's teams; Everything covers every
// plan.
const (
	ScopeMyTeams    Scope = "myTeams"
	ScopeEverything Scope = "everything"
)

// chooseScopeMessage is the answer to any other scope (plan "Validation messages").
const chooseScopeMessage = "Choose My teams or Everything."

// ParseScope reads the query parameter scope; anything but myTeams or everything is validation_failed on
// the field scope.
func ParseScope(text string) (Scope, error) {
	switch scope := Scope(text); scope {
	case ScopeMyTeams, ScopeEverything:
		return scope, nil
	}
	return "", errs.Validation(errs.FieldError{Field: "scope", Message: chooseScopeMessage})
}

// The API types below are what the API returns (contract schema in brackets, JSON names as the contract).
// Every amount is in the person's display currency. Lists are never nil.

// SummaryView is the four summary cards and whether to show Getting started (HomeSummary, FR-005, FR-006,
// FR-020).
type SummaryView struct {
	Scope           Scope            `json:"scope"`
	DisplayCurrency string           `json:"displayCurrency"` // an inactive one stays in use (D-063, D-201)
	GettingStarted  *string          `json:"gettingStarted"`  // notInTeam, noTeamPlans or null
	Plans           PlanCounts       `json:"plans"`
	ActivitiesOwned ActivityCounts   `json:"activitiesOwned"`
	WaitingForMe    WaitingCounts    `json:"waitingForMe"`
	PlannedCost     PlannedCostTotal `json:"plannedCost"`
}

// PlanCounts are the plans in scope by status (HomePlanCounts).
type PlanCounts struct {
	Count     int `json:"count"`
	Published int `json:"published"`
	Draft     int `json:"draft"`
}

// ActivityCounts are the activities the person owns, in every plan (HomeActivityCounts).
type ActivityCounts struct {
	Count     int `json:"count"`
	Declined  int `json:"declined"`
	Submitted int `json:"submitted"`
}

// WaitingCounts are the activities waiting for the person's decision (HomeWaitingCounts).
type WaitingCounts struct {
	Count              int        `json:"count"`
	OldestWaitingSince *time.Time `json:"oldestWaitingSince"` // null when nothing waits
}

// PlannedCostTotal is the planned cost of the plans in scope against their plan budgets (HomePlannedCost).
type PlannedCostTotal struct {
	PlanCount          int          `json:"planCount"`
	PlannedCost        money.Money  `json:"plannedCost"`
	PlanBudget         *money.Money `json:"planBudget"`        // null when no plan in scope has a budget
	BudgetUsedPercent  *int         `json:"budgetUsedPercent"` // null without a budget
	OverBudget         bool         `json:"overBudget"`
	PlansWithoutBudget int          `json:"plansWithoutBudget"`
}

// TeamView is one team of a plan row as every record shows a team (003's TeamRef: its colour, and
// whether it is active), and whether the person is a member (HomeTeam, D-217).
type TeamView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Colour string `json:"colour"`
	Active bool   `json:"active"`
	Mine   bool   `json:"mine"`
}

// PlanRow is one plan of Plans at a glance (HomePlanRow, FR-008, FR-009).
type PlanRow struct {
	ID                    string       `json:"id"`
	Name                  string       `json:"name"`
	Year                  int          `json:"year"`
	StrategicAOP          bool         `json:"strategicAop"`
	Status                string       `json:"status"`
	Teams                 []TeamView   `json:"teams"` // the person's teams first, then by name
	CampaignCount         int          `json:"campaignCount"`
	ActivityCount         int          `json:"activityCount"`
	ApprovedActivityCount int          `json:"approvedActivityCount"`
	Budget                *money.Money `json:"budget"`      // null: "—" ("No budget")
	PlannedCost           *money.Money `json:"plannedCost"` // null when nothing is planned, as on the plan page (D-217)
	BudgetUsedPercent     *int         `json:"budgetUsedPercent"`
	OverBudget            bool         `json:"overBudget"`
}

// PlanPage is Plans at a glance: at most 10 rows, and every plan in scope counted (HomePlanPage, FR-010).
type PlanPage struct {
	Items []PlanRow `json:"items"`
	Total int       `json:"total"`
}

// HomeAttentionReason is one reason a record needs attention (HomeAttentionReason). Fields that do not
// apply to the kind are null, empty or 0.
type HomeAttentionReason struct {
	Kind       string         `json:"kind"`
	Amount     *money.Money   `json:"amount"`     // negativeBalance, overAllocated, overBudget: how much over
	DeclinedBy *actor.Actor   `json:"declinedBy"` // declined
	DeclinedAt *time.Time     `json:"declinedAt"`
	Comment    *string        `json:"comment"`
	Missing    []MissingField `json:"missing"`    // missingFields: the record's own empty needed fields
	ItemsBelow int            `json:"itemsBelow"` // missingFields: records below that need attention
}

// HomeAttentionItem is a record the person owns that needs attention, with all its reasons
// (HomeAttentionItem, FR-012).
type HomeAttentionItem struct {
	RecordType      string                `json:"recordType"` // plan, campaign, initiative or activity
	RecordTypeLabel string                `json:"recordTypeLabel"`
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Plan            errs.RecordRef        `json:"plan"` // the plan itself for a plan
	Path            []errs.RecordRef      `json:"path"` // ancestors, plan first; empty for a plan
	ChangedAt       time.Time             `json:"changedAt"`
	Reasons         []HomeAttentionReason `json:"reasons"`
}

// HomeAttentionPage is Needs your attention: every item, in display order (HomeAttentionPage, D-173).
type HomeAttentionPage struct {
	Items []HomeAttentionItem `json:"items"`
	Total int                 `json:"total"`
}

// WaitingItem is one activity waiting for the person's decision (WaitingItem, FR-013).
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

// WaitingPage is Waiting for my approval: the first 5, oldest first, and how many wait in all (WaitingPage).
type WaitingPage struct {
	Items []WaitingItem `json:"items"`
	Total int           `json:"total"`
}

// MyActivityRow is one activity of My activities (MyActivityRow, FR-014).
type MyActivityRow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Plan        errs.RecordRef `json:"plan"`
	Campaign    errs.RecordRef `json:"campaign"`
	Initiative  errs.RecordRef `json:"initiative"`
	Status      string         `json:"status"` // displayed (domain model §2.3)
	StartDate   *date.Date     `json:"startDate"`
	EndDate     *date.Date     `json:"endDate"`
	Ongoing     bool           `json:"ongoing"`
	PlannedCost *money.Money   `json:"plannedCost"`
	Missing     []MissingField `json:"missing"`
	ItemsBelow  int            `json:"itemsBelow"` // line items that need attention
}

// MyActivityGroup is the person's activities of one displayed status (MyActivityGroup).
type MyActivityGroup struct {
	Status string          `json:"status"`
	Count  int             `json:"count"`
	Items  []MyActivityRow `json:"items"`
}

// MyActivitiesView is My activities: the groups in display order, empty ones left out (MyActivitiesPage).
type MyActivitiesView struct {
	Groups []MyActivityGroup `json:"groups"`
	Total  int               `json:"total"`
}
