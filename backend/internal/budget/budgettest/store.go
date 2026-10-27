// Package budgettest is the in-memory fake of the global budgets and allocations store, for tests that build
// services or the whole API without a database.
package budgettest

import (
	"context"

	"mapp/internal/budget"
	"mapp/internal/platform/fake"
)

type Store struct {
	budgets     *fake.Table[budget.GlobalBudget]
	allocations *fake.Table[budget.Allocation]
}

func NewStore(db *fake.DB) Store {
	return Store{
		budgets:     fake.TableOf(db, "global_budgets", func(b budget.GlobalBudget) string { return b.ID }),
		allocations: fake.TableOf(db, "budget_allocations", func(a budget.Allocation) string { return a.ID }),
	}
}

func (s Store) GetBudget(_ context.Context, id string) (budget.GlobalBudget, bool, error) {
	return s.budgets.Get(id)
}

func (s Store) ListBudgets(context.Context) ([]budget.GlobalBudget, error) {
	return s.budgets.List()
}

func (s Store) PutBudget(ctx context.Context, b budget.GlobalBudget) error {
	return s.budgets.Put(ctx, b)
}

func (s Store) DeleteBudget(ctx context.Context, id string) error {
	return s.budgets.Delete(ctx, id)
}

func (s Store) GetAllocation(_ context.Context, id string) (budget.Allocation, bool, error) {
	return s.allocations.Get(id)
}

func (s Store) ListAllocations(context.Context) ([]budget.Allocation, error) {
	return s.allocations.List()
}

func (s Store) AllocationsOf(_ context.Context, globalBudgetID string) ([]budget.Allocation, error) {
	return s.allocations.Where(func(a budget.Allocation) bool { return a.GlobalBudgetID == globalBudgetID })
}

func (s Store) AllocationOfPlan(_ context.Context, planID string) (budget.Allocation, bool, error) {
	of, err := s.allocations.Where(func(a budget.Allocation) bool { return a.PlanID == planID })
	if err != nil || len(of) == 0 {
		return budget.Allocation{}, false, err
	}
	return of[0], true, nil
}

func (s Store) PutAllocation(ctx context.Context, a budget.Allocation) error {
	return s.allocations.Put(ctx, a)
}

func (s Store) DeleteAllocation(ctx context.Context, id string) error {
	return s.allocations.Delete(ctx, id)
}
