// Package dashboard is the planning dashboards and the export of plans and dashboards (feature 036): read-only, it
// works every figure out from what the owning features calculate (ports.go) and stores nothing.
package dashboard

import (
	"time"

	"mapp/internal/people"
	"mapp/internal/platform/money"
)

type Kind string

const (
	KindExecutive Kind = "executive"
	KindRegional  Kind = "regional"
)

// View is the contract's PlanningDashboard; a nil amount or percent is "—".
type View struct {
	Kind               Kind             `json:"kind"`
	Filters            Filters          `json:"filters"`
	Choices            Choices          `json:"choices"`
	Team               *people.TeamRef  `json:"team"`
	PlanCount          int              `json:"planCount"`
	PlansWithoutBudget int              `json:"plansWithoutBudget"`
	Finance            Finance          `json:"finance"`
	FinanceAsOf        *time.Time       `json:"financeAsOf"`
	Totals             Figures          `json:"totals"`
	Markets            []MarketRow      `json:"markets"`
	Plans              []PlanRow        `json:"plans"`
	OverBudget         []CampaignRow    `json:"overBudget"`
	LeftToPlan         []CampaignRow    `json:"leftToPlan"`
	PlanCampaigns      *PlanCampaigns   `json:"planCampaigns"`
	Months             Months           `json:"months"`
	Approvals          Approvals        `json:"approvals"`
	PurchaseRequests   PurchaseRequests `json:"purchaseRequests"`
	Completeness       []Completeness   `json:"completeness"`
	GlobalBudgets      []GlobalBudget   `json:"globalBudgets"`
	CanExport          bool             `json:"canExport"`
}

type Filters struct {
	Year     int     `json:"year"`
	PlanType *string `json:"planType"`
	Currency string  `json:"currency"`
	TeamID   *string `json:"teamId"`
	PlanID   *string `json:"planId"`
}

type Choices struct {
	Years      []int            `json:"years"`
	Teams      []TeamChoice     `json:"teams"`
	Currencies []CurrencyChoice `json:"currencies"`
}

type TeamChoice struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
	Mine   bool   `json:"mine"`
}

type CurrencyChoice struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Finance struct {
	Visible   bool `json:"visible"`
	PlanCount int  `json:"planCount"`
}

type Figures struct {
	Budget             *money.Money `json:"budget"`
	PlannedCost        *money.Money `json:"plannedCost"`
	Committed          *money.Money `json:"committed"`
	Actual             *money.Money `json:"actual"`
	BudgetUsedPercent  *int         `json:"budgetUsedPercent"`
	BudgetSpentPercent *int         `json:"budgetSpentPercent"`
	LeftToPlan         *money.Money `json:"leftToPlan"`
	UnspentBudget      *money.Money `json:"unspentBudget"`
	OverBudget         bool         `json:"overBudget"`
}

type MarketRow struct {
	Team      people.TeamRef `json:"team"`
	PlanCount int            `json:"planCount"`
	Figures   Figures        `json:"figures"`
}

type PlanRef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	PlanType string `json:"planType"`
}

type PlanRow struct {
	Plan         PlanRef          `json:"plan"`
	Teams        []people.TeamRef `json:"teams"`
	Figures      Figures          `json:"figures"`
	FinanceShown bool             `json:"financeShown"`
}

type RecordRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CampaignRow struct {
	Campaign   RecordRef    `json:"campaign"`
	Plan       RecordRef    `json:"plan"`
	Figures    Figures      `json:"figures"`
	Difference *money.Money `json:"difference"`
}

type PlanCampaigns struct {
	Plan  RecordRef     `json:"plan"`
	Items []CampaignRow `json:"items"`
}

type Months struct {
	Items           []Month      `json:"items"`
	NoDates         *money.Money `json:"noDates"`
	OtherYears      *money.Money `json:"otherYears"`
	PaidOutsideYear *money.Money `json:"paidOutsideYear"`

	NegativeBalanceCount int `json:"negativeBalanceCount"`
}

type Month struct {
	Month         string       `json:"month"`
	Planned       money.Money  `json:"planned"`
	Actual        *money.Money `json:"actual"`
	PlannedToDate money.Money  `json:"plannedToDate"`
	ActualToDate  *money.Money `json:"actualToDate"`
}

type Approvals struct {
	Waiting             int           `json:"waiting"`
	OldestWaitingSince  *time.Time    `json:"oldestWaitingSince"`
	Approved            int           `json:"approved"`
	Declined            int           `json:"declined"`
	AutoApproved        int           `json:"autoApproved"`
	AverageDecisionDays *string       `json:"averageDecisionDays"`
	ActivitiesByStatus  []StatusCount `json:"activitiesByStatus"`
}

type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type PurchaseRequests struct {
	Available      bool                    `json:"available"`
	ByStatus       []PurchaseRequestStatus `json:"byStatus"`
	Count          int                     `json:"count"`
	Amount         money.Money             `json:"amount"`
	WaitingForGSAP int                     `json:"waitingForGsap"`
}

type PurchaseRequestStatus struct {
	Status string      `json:"status"`
	Count  int         `json:"count"`
	Amount money.Money `json:"amount"`
}

type Completeness struct {
	Level    string `json:"level"`
	Total    int    `json:"total"`
	Complete int    `json:"complete"`
	Percent  *int   `json:"percent"`
}

type GlobalBudget struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Amount        money.Money `json:"amount"`
	Allocated     money.Money `json:"allocated"`
	Unallocated   money.Money `json:"unallocated"`
	PlannedCost   money.Money `json:"plannedCost"`
	OverAllocated bool        `json:"overAllocated"`
}
