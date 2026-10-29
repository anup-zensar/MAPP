// Package spa is the SP&A view of a plan or campaign (feature 013): planned cost against committed and actual
// spend, by period, by dimension and by activity, with the budget and procurement panels. It stores nothing.
package spa

import (
	"time"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/money"
)

type RecordRef struct {
	Type Level  `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Contribution struct {
	ActivityID string      `json:"activityId"`
	Name       string      `json:"name"`
	Amount     money.Money `json:"amount"`
}

type PlanRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Year   int    `json:"year"`
	Status string `json:"status"`
	BCPIR  bool   `json:"bcpir"`
}

type Headline struct {
	PlannedCost         money.Money    `json:"plannedCost"`
	Committed           *money.Money   `json:"committed"`
	ActualSpend         *money.Money   `json:"actualSpend"`
	Invoiced            *money.Money   `json:"invoiced"`
	Remaining           *money.Money   `json:"remaining"`
	UtilisationPercent  *string        `json:"utilisationPercent"`
	VariancePercent     *string        `json:"variancePercent"`
	Overspend           bool           `json:"overspend"`
	NoPlannedCostReason *string        `json:"noPlannedCostReason"`
	PlannedCostTrace    []Contribution `json:"plannedCostTrace"`
	CommittedTrace      []Contribution `json:"committedTrace"`
	ActualSpendTrace    []Contribution `json:"actualSpendTrace"`
}

type Period struct {
	Key          string         `json:"key"`
	Label        string         `json:"label"`
	Planned      money.Money    `json:"planned"`
	Actual       *money.Money   `json:"actual"`
	PlannedBar   int            `json:"plannedBar"`
	ActualBar    int            `json:"actualBar"`
	PlannedTrace []Contribution `json:"plannedTrace"`
	ActualTrace  []Contribution `json:"actualTrace"`
}

type Periods struct {
	Quarters          []Period       `json:"quarters"`
	Months            []Period       `json:"months"`
	NotScheduled      *money.Money   `json:"notScheduled"`
	NotScheduledTrace []Contribution `json:"notScheduledTrace"`
}

type DimensionChoice struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type DimensionRow struct {
	Key    string         `json:"key"`
	Label  string         `json:"label"`
	Amount money.Money    `json:"amount"`
	Share  string         `json:"share"`
	Bar    int            `json:"bar"`
	Trace  []Contribution `json:"trace"`
}

type DimensionBreakdown struct {
	Key          string         `json:"key"`
	Label        string         `json:"label"`
	Rows         []DimensionRow `json:"rows"`
	Total        money.Money    `json:"total"`
	CountsInEach bool           `json:"countsInEach"`
}

type LineItemRow struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	PlannedCost money.Money  `json:"plannedCost"`
	Committed   *money.Money `json:"committed"`
	ActualSpend *money.Money `json:"actualSpend"`
}

type ActivityRow struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Campaign             RecordRef          `json:"campaign"`
	Initiative           RecordRef          `json:"initiative"`
	Owner                *string            `json:"owner"`
	ActivityType         *string            `json:"activityType"`
	Channel              *string            `json:"channel"`
	StartQuarter         *string            `json:"startQuarter"`
	Status               string             `json:"status"`
	PlannedCost          money.Money        `json:"plannedCost"`
	Committed            *money.Money       `json:"committed"`
	ActualSpend          *money.Money       `json:"actualSpend"`
	VariancePercent      *string            `json:"variancePercent"`
	Overspend            bool               `json:"overspend"`
	LineItems            []LineItemRow      `json:"lineItems"`
	RaisePurchaseRequest authz.Availability `json:"raisePurchaseRequest"`
}

type BudgetPanel struct {
	Budget                 *money.Money `json:"budget"`
	PlannedCost            money.Money  `json:"plannedCost"`
	Committed              *money.Money `json:"committed"`
	PlannedVsBudgetPercent *string      `json:"plannedVsBudgetPercent"`
	OverBudget             bool         `json:"overBudget"`
	NoBudgetReason         *string      `json:"noBudgetReason"`
	Linked                 money.Money  `json:"linked"`
	NotLinked              money.Money  `json:"notLinked"`
}

type PurchaseRequestRow struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Vendor  string      `json:"vendor"`
	Status  string      `json:"status"`
	Amount  money.Money `json:"amount"`
	Counted bool        `json:"counted"`
}

type ProcurementPanel struct {
	Count           int                  `json:"count"`
	Approved        money.Money          `json:"approved"`
	PendingApproval money.Money          `json:"pendingApproval"`
	Draft           money.Money          `json:"draft"`
	Invoiced        *money.Money         `json:"invoiced"`
	Items           []PurchaseRequestRow `json:"items"`
}

// View is the SP&A view (contract SpaView); every amount is in the person's display currency.
type View struct {
	Plan            PlanRef            `json:"plan"`
	Scope           RecordRef          `json:"scope"`
	Campaigns       []RecordRef        `json:"campaigns"`
	Currency        string             `json:"currency"`
	FinanceAccess   bool               `json:"financeAccess"`
	FinanceNote     *string            `json:"financeNote"`
	FinanceDataAsOf *time.Time         `json:"financeDataAsOf"`
	Headline        Headline           `json:"headline"`
	Periods         Periods            `json:"periods"`
	Dimension       DimensionBreakdown `json:"dimension"`
	Dimensions      []DimensionChoice  `json:"dimensions"`
	Activities      []ActivityRow      `json:"activities"`
	Budget          BudgetPanel        `json:"budget"`
	Procurement     ProcurementPanel   `json:"procurement"`
	Notes           []string           `json:"notes"`
	EmptyMessage    *string            `json:"emptyMessage"`
}

// Query is what the person chose: a campaign of the plan ("" for the whole plan) and a dimension key ("" for
// the default).
type Query struct {
	CampaignID string
	Dimension  string
}
