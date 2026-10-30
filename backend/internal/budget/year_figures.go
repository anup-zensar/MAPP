package budget

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
)

// YearFigures gives feature 036's dashboards year's global budgets, by name, with List's figures, in three
// statements whatever their number (B5). It checks no access: the caller holds globalBudgets.view (D-343, D-356).
func (s *Service) YearFigures(ctx context.Context, a actor.Actor, year int) ([]Summary, error) {
	list := []Summary{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		all, err := s.repo.ListBudgets(ctx)
		if err != nil {
			return err
		}
		chosen := slices.DeleteFunc(all, func(b GlobalBudget) bool { return b.Year != year })
		slices.SortFunc(chosen, byName)
		ids := make([]string, 0, len(chosen))
		for _, b := range chosen {
			ids = append(ids, b.ID)
		}
		allocations, err := s.repo.AllocationsOfBudgets(ctx, ids)
		if err != nil {
			return err
		}
		of, err := s.amounts(ctx, allocations)
		if err != nil {
			return err
		}
		for _, b := range chosen {
			list = append(list, Summary{ID: b.ID, Name: b.Name, Year: b.Year, Figures: figuresOf(b.Amount, of[b.ID]),
				AllocationCount: len(of[b.ID])})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("global budgets of %d for %s: %w", year, a.ID, err)
	}
	return list, nil
}

// amounts are the allocations in USD with their plans' planned cost, by global budget.
func (s *Service) amounts(ctx context.Context, allocations []Allocation) (map[string][]allocated, error) {
	f, err := s.newFigurer(ctx)
	if err != nil {
		return nil, err
	}
	planIDs := make([]string, 0, len(allocations))
	for _, a := range allocations {
		planIDs = append(planIDs, a.PlanID)
	}
	planned, err := s.plans.PlannedCosts(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	of := map[string][]allocated{}
	for _, a := range allocations {
		usd, err := f.rates.ToUSD(a.Amount)
		if err != nil {
			return nil, err
		}
		of[a.GlobalBudgetID] = append(of[a.GlobalBudgetID], allocated{allocation: a, amountUSD: usd, plannedUSD: planned[a.PlanID]})
	}
	return of, nil
}
