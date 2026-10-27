package plans

import (
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

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

// Figures are the plan's calculated money and progress in one currency (PlanFigures): the reader's display
// currency in figures, the plan currency in planFigures (016 FR-013).
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
