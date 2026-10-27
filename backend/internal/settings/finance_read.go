package settings

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// ListFinanceBudgets returns the finance budgets of year (the clock's UTC year when nil) that have at least
// one item, by name, each with its items by name, its active item count and its usage; and the years that
// can be chosen (FR-014, FR-015, US5-3). Every signed-in person may see them (FR-008).
func (s *Service) ListFinanceBudgets(ctx context.Context, _ actor.Actor, year *int) (FinanceBudgetPage, error) {
	current := s.now().UTC().Year()
	shown := current
	if year != nil {
		shown = *year
	}
	page := FinanceBudgetPage{Items: []FinanceBudgetView{}, Year: shown}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		budgets, err := s.repo.ListFinanceBudgets(ctx)
		if err != nil {
			return err
		}
		items, err := s.repo.ListFinanceBudgetItems(ctx)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		itemsOf := map[string][]FinanceBudgetItem{}
		for _, i := range items {
			itemsOf[i.BudgetID] = append(itemsOf[i.BudgetID], i)
		}
		years := map[int]bool{}
		for y := current - yearsBefore; y <= current+yearsAfter; y++ {
			years[y] = true
		}
		slices.SortStableFunc(budgets, func(x, y FinanceBudget) int { return compareNames(x.Name, y.Name) })
		for _, b := range budgets {
			if len(itemsOf[b.ID]) == 0 {
				continue
			}
			years[b.Year] = true
			if b.Year == shown {
				page.Items = append(page.Items, budgetView(b, itemsOf[b.ID], u))
			}
		}
		for y := range years {
			page.Years = append(page.Years, y)
		}
		slices.Sort(page.Years)
		return nil
	})
	page.Total = len(page.Items)
	return page, err
}

// GetFinanceBudgetItem returns the finance budget item id with its year, budget name, usage and locks
// (US5-4).
func (s *Service) GetFinanceBudgetItem(ctx context.Context, _ actor.Actor, id string) (FinanceBudgetItemView, error) {
	var view FinanceBudgetItemView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		item, err := s.storedItem(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.itemViewOf(ctx, item)
		return err
	})
	return view, err
}

// storedItem reads the finance budget item id, or answers not found.
func (s *Service) storedItem(ctx context.Context, id string) (FinanceBudgetItem, error) {
	item, found, err := s.repo.GetFinanceBudgetItem(ctx, id)
	if err != nil {
		return FinanceBudgetItem{}, err
	}
	if !found {
		return FinanceBudgetItem{}, errs.NotFound(itemGoneMessage)
	}
	return item, nil
}

// itemViewOf returns the item as the API answers it, reading its budget and the usage index.
func (s *Service) itemViewOf(ctx context.Context, item FinanceBudgetItem) (FinanceBudgetItemView, error) {
	budget, _, err := s.repo.GetFinanceBudget(ctx, item.BudgetID)
	if err != nil {
		return FinanceBudgetItemView{}, err
	}
	u, err := s.usage(ctx)
	if err != nil {
		return FinanceBudgetItemView{}, err
	}
	return itemView(item, budget, u), nil
}

// budgetView returns the budget b with its items as the API answers it.
func budgetView(b FinanceBudget, items []FinanceBudgetItem, u *usageIndex) FinanceBudgetView {
	view := FinanceBudgetView{ID: b.ID, Year: b.Year, Name: b.Name, Items: []FinanceBudgetItemView{}, Usage: u.budget(b.ID), Meta: b.Meta}
	sorted := slices.Clone(items)
	slices.SortStableFunc(sorted, func(x, y FinanceBudgetItem) int { return compareNames(x.Name, y.Name) })
	for _, i := range sorted {
		view.Items = append(view.Items, itemView(i, b, u))
		if i.Active {
			view.ActiveItemCount++
		}
	}
	return view
}

// itemView returns the item i of the budget b as the API answers it. Its year never changes (data model §8).
func itemView(i FinanceBudgetItem, b FinanceBudget, u *usageIndex) FinanceBudgetItemView {
	return FinanceBudgetItemView{
		ID: i.ID, BudgetID: i.BudgetID, BudgetName: b.Name, Year: b.Year, Name: i.Name, Code: i.Code, Active: i.Active,
		Usage: u.item(i.ID), Locks: []Lock{{Field: "year", Reason: yearLockedMessage}}, Meta: i.Meta,
	}
}
