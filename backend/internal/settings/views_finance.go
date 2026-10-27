package settings

import (
	"mapp/internal/platform/record"
)

// FinanceBudgetItemView is a finance budget item as the API returns it (contract FinanceBudgetItem).
type FinanceBudgetItemView struct {
	ID         string `json:"id"`
	BudgetID   string `json:"budgetId"`
	BudgetName string `json:"budgetName"`
	Year       int    `json:"year"`
	Name       string `json:"name"`
	Code       string `json:"code"`
	Active     bool   `json:"active"`
	Usage      Usage  `json:"usage"`
	Locks      []Lock `json:"locks"`
	record.Meta
}

// FinanceBudgetView is a finance budget with its items (contract FinanceBudget).
type FinanceBudgetView struct {
	ID              string                  `json:"id"`
	Year            int                     `json:"year"`
	Name            string                  `json:"name"`
	Items           []FinanceBudgetItemView `json:"items"` // active and inactive, by name
	ActiveItemCount int                     `json:"activeItemCount"`
	Usage           Usage                   `json:"usage"`
	record.Meta
}

// FinanceBudgetPage is the answer of GET /api/v1/finance-budgets?year= (contract FinanceBudgetPage).
type FinanceBudgetPage struct {
	Items []FinanceBudgetView `json:"items"` // the year's budgets with at least one item, by name
	Total int                 `json:"total"`
	Year  int                 `json:"year"`  // the year shown
	Years []int               `json:"years"` // current − 1 … + 5 and any other year with items, ascending
}

// RenameInput is the body of PATCH /api/v1/finance-budgets/{budgetId} (contract FinanceBudgetRename).
type RenameInput struct {
	Name    string `json:"name"`
	Version *int   `json:"version"`
}

// ItemInput is the body of POST /api/v1/finance-budget-items (contract FinanceBudgetItemCreate): the budget
// is found by year and name, or created.
type ItemInput struct {
	Year       *int   `json:"year"`
	BudgetName string `json:"budgetName"`
	Name       string `json:"name"`
	Code       string `json:"code"`
}

// ItemUpdate is the body of PATCH /api/v1/finance-budget-items/{itemId} (contract FinanceBudgetItemUpdate). A
// new budget name moves the item to that budget of the same year.
type ItemUpdate struct {
	BudgetName *string `json:"budgetName"`
	Name       *string `json:"name"`
	Code       *string `json:"code"`
	Version    *int    `json:"version"`
}
