// Package budgets holds global budgets and their allocations to market plans (feature 014, BUS.BUD.040): a
// global planner creates a global budget for a year and shares it out top-down; each allocation becomes the
// plan's budget, which local planners see but cannot change (FR-029 – FR-036, D-235, D-236).
//
// Budgets reads plans, people and exchange rates through the small interfaces of deps.go, and asks 014's
// access rules what the signed-in person may do through Access. The plan budget itself stays with package
// plans: an allocation sets it through plans' SetBudgetFromAllocation (one write path).
package budget

import (
	"mapp/internal/people"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// The readable-ID prefixes of global budgets and allocations (014 data model §1.6): GB-1001, BA-1001.
const (
	PrefixGlobalBudget = "GB"
	PrefixAllocation   = "BA"
)

// GlobalBudget is a global budget as stored in the table "global_budgets" (014 data model §1.4).
type GlobalBudget struct {
	ID          string       `json:"id"`          // "GB-1001"
	Name        string       `json:"name"`        // 1–120 characters, unique among global budgets ignoring case
	Year        int          `json:"year"`        // never changes after the first save
	Amount      money.Amount `json:"amount"`      // USD, 0 or more
	OwnerID     string       `json:"ownerId"`     // a person who manages global budgets for All markets
	Description string       `json:"description"` // 0–2,000 characters
	record.Meta
}

// Allocation is the part of a global budget given to one plan, as stored in the table "budget_allocations"
// (014 data model §1.5). It sets the plan's budget; a plan has at most one allocation across all global
// budgets.
type Allocation struct {
	ID             string      `json:"id"`             // "BA-1001"
	GlobalBudgetID string      `json:"globalBudgetId"` // an existing global budget
	PlanID         string      `json:"planId"`         // a plan of the global budget's year
	Amount         money.Money `json:"amount"`         // in the plan's currency, 0 or more (D-303)
	record.Meta
}

// The API types below are what the API returns and receives (contract schema in brackets, JSON names as the
// contract). Lists are never nil.

// Availability says whether the signed-in person may take one action now, and why not (ActionAvailability).
// Reason is "" when the action is available.
type Availability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// available is an action the person may take.
var available = Availability{Available: true}

// PlanRef is a plan as a global budget's rows and options name it (AllocationPlanRef). Version is what an
// allocation change sends back as planVersion.
type PlanRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Year    int    `json:"year"`
	Status  string `json:"status"`
	Version int    `json:"version"`
}

// Figures are the four totals of a global budget, every one in USD (GlobalBudgetFigures, FR-032, D-303).
type Figures struct {
	Amount        money.Money `json:"amount"`
	Allocated     money.Money `json:"allocated"`   // every allocation, Other markets included
	Unallocated   money.Money `json:"unallocated"` // amount − allocated; negative when over-allocated
	PlannedCost   money.Money `json:"plannedCost"` // planned cost of every allocated plan
	OverAllocated bool        `json:"overAllocated"`
}

// RowAccess is what the person may do with one allocation (BudgetAllocationRow.access).
type RowAccess struct {
	Edit   Availability `json:"edit"`
	Remove Availability `json:"remove"`
}

// AllocationRow is one allocation to a plan the person can see (BudgetAllocationRow). Amount, PlannedCost
// and LeftToPlan are in the plan's currency; AmountUSD is the allocation as the figures count it.
type AllocationRow struct {
	ID                string            `json:"id"`
	Plan              PlanRef           `json:"plan"`
	Teams             []people.TeamRef  `json:"teams"`
	Owner             *people.PersonRef `json:"owner"`
	Amount            money.Money       `json:"amount"`
	AmountUSD         money.Money       `json:"amountUsd"`
	PlannedCost       *money.Money      `json:"plannedCost"` // null when nothing in the plan has a planned cost
	LeftToPlan        money.Money       `json:"leftToPlan"`
	BudgetUsedPercent *string           `json:"budgetUsedPercent"` // "62.50"; null when the allocation is 0.00
	Version           int               `json:"version"`
	Access            RowAccess         `json:"access"`
}

// OtherMarkets adds up the allocations to plans the person cannot see, in USD (OtherMarketsRow, FR-033).
type OtherMarkets struct {
	PlanCount   int         `json:"planCount"`
	Amount      money.Money `json:"amount"`
	PlannedCost money.Money `json:"plannedCost"`
}

// AllocationOption is a plan of the global budget's year without an allocation (AllocationOption).
type AllocationOption struct {
	Plan        PlanRef           `json:"plan"`
	Teams       []people.TeamRef  `json:"teams"`
	Owner       *people.PersonRef `json:"owner"`
	Budget      *money.Money      `json:"budget"` // the plan's budget now; null when none
	CanAllocate Availability      `json:"canAllocate"`
}

// AllocationOptionPage is the answer of GET /api/v1/global-budgets/{globalBudgetId}/allocation-options
// (AllocationOptionPage).
type AllocationOptionPage struct {
	Items []AllocationOption `json:"items"`
	Total int                `json:"total"`
}

// BudgetAccess is what the person may do with a global budget (GlobalBudget.access).
type BudgetAccess struct {
	Edit     Availability `json:"edit"`
	Delete   Availability `json:"delete"`
	Allocate Availability `json:"allocate"`
}

// GlobalBudgetView is a global budget as its page shows it (GlobalBudget, FR-032, FR-033).
type GlobalBudgetView struct {
	ID                     string             `json:"id"`
	Name                   string             `json:"name"`
	Year                   int                `json:"year"`
	Amount                 money.Money        `json:"amount"`
	Owner                  *people.PersonRef  `json:"owner"`
	Description            string             `json:"description"`
	Figures                Figures            `json:"figures"`
	Allocations            []AllocationRow    `json:"allocations"`  // to plans the person can see, by plan name
	OtherMarkets           *OtherMarkets      `json:"otherMarkets"` // null when every allocated plan is visible
	PlansWithoutAllocation []AllocationOption `json:"plansWithoutAllocation"`
	Locks                  []settings.Lock    `json:"locks"`
	Access                 BudgetAccess       `json:"access"`
	record.Meta
}

// Summary is a global budget as the Global budgets list shows it (GlobalBudgetSummary).
type Summary struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Year            int               `json:"year"`
	Owner           *people.PersonRef `json:"owner"`
	Figures         Figures           `json:"figures"`
	AllocationCount int               `json:"allocationCount"` // every allocation of the global budget
}

// Page is the answer of GET /api/v1/global-budgets (GlobalBudgetPage).
type Page struct {
	Items  []Summary    `json:"items"`
	Total  int          `json:"total"`
	Years  []int        `json:"years"`  // every year with a global budget, newest first
	Create Availability `json:"create"` // New global budget
}

// GlobalBudgetInput is the body of POST /api/v1/global-budgets (GlobalBudgetCreate). The amount is a money
// input, so a wrong one becomes a field message.
type GlobalBudgetInput struct {
	Name        string       `json:"name"`
	Year        *int         `json:"year"`
	Amount      *money.Input `json:"amount"`
	OwnerID     string       `json:"ownerId"`
	Description string       `json:"description"`
}

// GlobalBudgetUpdate is the body of PATCH /api/v1/global-budgets/{globalBudgetId} (GlobalBudgetUpdate): only the
// fields to change, and the version the person started from (D-203). The year never changes.
type GlobalBudgetUpdate struct {
	Name        *string      `json:"name"`
	Amount      *money.Input `json:"amount"`
	OwnerID     *string      `json:"ownerId"`
	Description *string      `json:"description"`
	Version     *int         `json:"version"`
}

// AllocationInput is the body of POST /api/v1/global-budgets/{globalBudgetId}/allocations
// (BudgetAllocationCreate). PlanVersion is the plan's version the person saw.
type AllocationInput struct {
	PlanID      string       `json:"planId"`
	Amount      *money.Input `json:"amount"`
	PlanVersion *int         `json:"planVersion"`
}

// AllocationUpdate is the body of PATCH /api/v1/budget-allocations/{allocationId} (BudgetAllocationUpdate).
type AllocationUpdate struct {
	Amount      *money.Input `json:"amount"`
	Version     *int         `json:"version"`     // the allocation's version
	PlanVersion *int         `json:"planVersion"` // the plan's version the person saw
}
