package allwork

import (
	"mapp/internal/platform/money"
)

// Grouping is one grouping the Breakdown offers (contract AllWorkGrouping).
type Grouping struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Level     *string `json:"level"` // the attribute's level; null for the built-in groupings
	HasBudget bool    `json:"hasBudget"`

	attribute *Attribute // attribute groupings
}

// BreakdownSubgroup is a "then by" group inside a group (contract AllWorkBreakdownSubgroup).
type BreakdownSubgroup struct {
	Key               string       `json:"key"`
	Label             string       `json:"label"`
	Activities        int          `json:"activities"`
	LineItems         int          `json:"lineItems"`
	PlannedCost       money.Money  `json:"plannedCost"`
	Share             string       `json:"share"`
	Budget            *money.Money `json:"budget"`
	BudgetUsedPercent *int         `json:"budgetUsedPercent"`
	OverBudget        bool         `json:"overBudget"`
}

// BreakdownGroup is one group of the Breakdown, with its "then by" groups (contract AllWorkBreakdownGroup).
type BreakdownGroup struct {
	Key               string              `json:"key"`
	Label             string              `json:"label"`
	Activities        int                 `json:"activities"`
	LineItems         int                 `json:"lineItems"`
	PlannedCost       money.Money         `json:"plannedCost"`
	Share             string              `json:"share"`
	Budget            *money.Money        `json:"budget"`
	BudgetUsedPercent *int                `json:"budgetUsedPercent"`
	OverBudget        bool                `json:"overBudget"`
	Children          []BreakdownSubgroup `json:"children"`
}

// Breakdown is the Breakdown of a filter set (contract AllWorkBreakdown).
type Breakdown struct {
	By           Grouping         `json:"by"`
	ThenBy       *Grouping        `json:"thenBy"`
	Currency     string           `json:"currency"`
	Groups       []BreakdownGroup `json:"groups"`
	Total        BreakdownGroup   `json:"total"`
	CountsInEach bool             `json:"countsInEach"`
	Notes        []string         `json:"notes"`
	Groupings    []Grouping       `json:"groupings"`
}
