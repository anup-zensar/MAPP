package budget

import (
	"context"
	"slices"

	"mapp/internal/platform/fake"
)

// fakeStore is budgettest.Store for this package's own tests, which cannot import budgettest.
type fakeStore struct {
	budgets     *fake.Table[GlobalBudget]
	allocations *fake.Table[Allocation]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{
		budgets:     fake.TableOf(db, "global_budgets", func(b GlobalBudget) string { return b.ID }),
		allocations: fake.TableOf(db, "budget_allocations", func(a Allocation) string { return a.ID }),
	}
}

func (s fakeStore) GetBudget(_ context.Context, id string) (GlobalBudget, bool, error) {
	return s.budgets.Get(id)
}

func (s fakeStore) ListBudgets(context.Context) ([]GlobalBudget, error) {
	return s.budgets.List()
}

func (s fakeStore) PutBudget(ctx context.Context, b GlobalBudget) error {
	return s.budgets.Put(ctx, b)
}

func (s fakeStore) DeleteBudget(ctx context.Context, id string) error {
	return s.budgets.Delete(ctx, id)
}

func (s fakeStore) GetAllocation(_ context.Context, id string) (Allocation, bool, error) {
	return s.allocations.Get(id)
}

func (s fakeStore) ListAllocations(context.Context) ([]Allocation, error) {
	return s.allocations.List()
}

func (s fakeStore) AllocationsOf(_ context.Context, globalBudgetID string) ([]Allocation, error) {
	return s.allocations.Where(func(a Allocation) bool { return a.GlobalBudgetID == globalBudgetID })
}

func (s fakeStore) AllocationsOfBudgets(_ context.Context, globalBudgetIDs []string) ([]Allocation, error) {
	return s.allocations.Where(func(a Allocation) bool { return slices.Contains(globalBudgetIDs, a.GlobalBudgetID) })
}

func (s fakeStore) AllocationOfPlan(_ context.Context, planID string) (Allocation, bool, error) {
	of, err := s.allocations.Where(func(a Allocation) bool { return a.PlanID == planID })
	if err != nil || len(of) == 0 {
		return Allocation{}, false, err
	}
	return of[0], true, nil
}

func (s fakeStore) PutAllocation(ctx context.Context, a Allocation) error {
	return s.allocations.Put(ctx, a)
}

func (s fakeStore) DeleteAllocation(ctx context.Context, id string) error {
	return s.allocations.Delete(ctx, id)
}
