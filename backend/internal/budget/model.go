// Package budget holds global budgets and their allocations to market plans (feature 014, BUS.BUD.040): each
// allocation becomes the plan's budget, set through plans' SetBudgetFromAllocation (D-235, D-236).
package budget

import (
	"mapp/internal/overspend"
	"mapp/internal/people"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

const (
	PrefixGlobalBudget = "GB"
	PrefixAllocation   = "BA"
)

type GlobalBudget struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Year        int            `json:"year"`
	Amount      money.Amount   `json:"amount"`
	OwnerID     string         `json:"ownerId"`
	Description string         `json:"description"`
	Overspend   overspend.Mode `json:"overspend"`
	record.Meta
}

type Allocation struct {
	ID             string      `json:"id"`
	GlobalBudgetID string      `json:"globalBudgetId"`
	PlanID         string      `json:"planId"`
	Amount         money.Money `json:"amount"`
	record.Meta
}

type Availability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

var available = Availability{Available: true}

type PlanRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Year    int    `json:"year"`
	Status  string `json:"status"`
	Version int    `json:"version"`
}

type Figures struct {
	Amount        money.Money `json:"amount"`
	Allocated     money.Money `json:"allocated"`
	Unallocated   money.Money `json:"unallocated"`
	PlannedCost   money.Money `json:"plannedCost"`
	OverAllocated bool        `json:"overAllocated"`
}

type RowAccess struct {
	Edit   Availability `json:"edit"`
	Remove Availability `json:"remove"`
}

type AllocationRow struct {
	ID                string            `json:"id"`
	Plan              PlanRef           `json:"plan"`
	Teams             []people.TeamRef  `json:"teams"`
	Owner             *people.PersonRef `json:"owner"`
	Amount            money.Money       `json:"amount"`
	AmountUSD         money.Money       `json:"amountUsd"`
	PlannedCost       *money.Money      `json:"plannedCost"`
	LeftToPlan        money.Money       `json:"leftToPlan"`
	BudgetUsedPercent *string           `json:"budgetUsedPercent"`
	Overspend         overspend.Mode    `json:"overspend"`
	Version           int               `json:"version"`
	Access            RowAccess         `json:"access"`
}

type OtherMarkets struct {
	PlanCount   int         `json:"planCount"`
	Amount      money.Money `json:"amount"`
	PlannedCost money.Money `json:"plannedCost"`
}

type AllocationOption struct {
	Plan        PlanRef           `json:"plan"`
	Teams       []people.TeamRef  `json:"teams"`
	Owner       *people.PersonRef `json:"owner"`
	Budget      *money.Money      `json:"budget"`
	Overspend   overspend.Mode    `json:"overspend"`
	CanAllocate Availability      `json:"canAllocate"`
}

type AllocationOptionPage struct {
	Items []AllocationOption `json:"items"`
	Total int                `json:"total"`
}

type BudgetAccess struct {
	Edit     Availability `json:"edit"`
	Delete   Availability `json:"delete"`
	Allocate Availability `json:"allocate"`
}

type GlobalBudgetView struct {
	ID                     string             `json:"id"`
	Name                   string             `json:"name"`
	Year                   int                `json:"year"`
	Amount                 money.Money        `json:"amount"`
	Owner                  *people.PersonRef  `json:"owner"`
	Description            string             `json:"description"`
	Figures                Figures            `json:"figures"`
	Allocations            []AllocationRow    `json:"allocations"`
	OtherMarkets           *OtherMarkets      `json:"otherMarkets"`
	PlansWithoutAllocation []AllocationOption `json:"plansWithoutAllocation"`
	Locks                  []settings.Lock    `json:"locks"`
	Access                 BudgetAccess       `json:"access"`
	Overspend              overspend.Control  `json:"overspend"`
	record.Meta
}

type Summary struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Year            int               `json:"year"`
	Owner           *people.PersonRef `json:"owner"`
	Figures         Figures           `json:"figures"`
	AllocationCount int               `json:"allocationCount"`
}

type Page struct {
	Items  []Summary    `json:"items"`
	Total  int          `json:"total"`
	Years  []int        `json:"years"`
	Create Availability `json:"create"`
}

type GlobalBudgetInput struct {
	Name        string       `json:"name"`
	Year        *int         `json:"year"`
	Amount      *money.Input `json:"amount"`
	OwnerID     string       `json:"ownerId"`
	Description string       `json:"description"`
	Overspend   *string      `json:"overspend"`
}

type GlobalBudgetUpdate struct {
	Name        *string      `json:"name"`
	Amount      *money.Input `json:"amount"`
	OwnerID     *string      `json:"ownerId"`
	Description *string      `json:"description"`
	Overspend   *string      `json:"overspend"`
	Version     *int         `json:"version"`
}

type AllocationInput struct {
	PlanID      string       `json:"planId"`
	Amount      *money.Input `json:"amount"`
	Overspend   *string      `json:"overspend"`
	PlanVersion *int         `json:"planVersion"`
}

type AllocationUpdate struct {
	Amount      *money.Input `json:"amount"`
	Overspend   *string      `json:"overspend"`
	Version     *int         `json:"version"`
	PlanVersion *int         `json:"planVersion"`
}
