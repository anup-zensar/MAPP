package settings

import (
	"context"
)

// budgetNamed returns the budget of year, other than exceptID, named name ignoring letter case and
// surrounding spaces, and false when there is none.
func (s *Service) budgetNamed(ctx context.Context, year int, name, exceptID string) (FinanceBudget, bool, error) {
	budgets, err := s.repo.ListFinanceBudgets(ctx)
	if err != nil {
		return FinanceBudget{}, false, err
	}
	sameYear := filter(budgets, func(b FinanceBudget) bool { return b.Year == year })
	b, found := findNamed(sameYear, name, exceptID, func(b FinanceBudget) (string, string) { return b.ID, b.Name })
	return b, found, nil
}

// itemWithCode returns the item, other than exceptID, with the code code ignoring letter case in the year
// of the budget budgetID, and false when there is none (FR-014).
func (s *Service) itemWithCode(ctx context.Context, budgetID, code, exceptID string) (FinanceBudgetItem, bool, error) {
	budget, found, err := s.repo.GetFinanceBudget(ctx, budgetID)
	if err != nil || !found {
		return FinanceBudgetItem{}, false, err
	}
	return s.itemWithCodeInYear(ctx, budget.Year, code, exceptID)
}

// itemWithCodeInYear returns the item, other than exceptID, with the code code ignoring letter case among
// the items of the budgets of year, and false when there is none (FR-014).
func (s *Service) itemWithCodeInYear(ctx context.Context, year int, code, exceptID string) (FinanceBudgetItem, bool, error) {
	budgets, err := s.repo.ListFinanceBudgets(ctx)
	if err != nil {
		return FinanceBudgetItem{}, false, err
	}
	items, err := s.repo.ListFinanceBudgetItems(ctx)
	if err != nil {
		return FinanceBudgetItem{}, false, err
	}
	sameYear := map[string]bool{}
	for _, b := range budgets {
		sameYear[b.ID] = b.Year == year
	}
	for _, i := range items {
		if i.ID != exceptID && sameYear[i.BudgetID] && i.Code != "" && sameName(i.Code, code) {
			return i, true, nil
		}
	}
	return FinanceBudgetItem{}, false, nil
}
