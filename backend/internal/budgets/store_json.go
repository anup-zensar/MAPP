package budgets

import (
	"context"

	"mapp/internal/platform/store"
)

// The JSON store tables of this package (014 data model §1.4–§1.5).
const (
	budgetsTable     = "global_budgets"
	allocationsTable = "budget_allocations"
)

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// GetBudget reads one global budget from the global_budgets table.
func (jsonRepository) GetBudget(ctx context.Context, id string) (GlobalBudget, bool, error) {
	return store.Get[GlobalBudget](ctx, budgetsTable, id)
}

// ListBudgets reads the whole global_budgets table.
func (jsonRepository) ListBudgets(ctx context.Context) ([]GlobalBudget, error) {
	return store.List[GlobalBudget](ctx, budgetsTable)
}

// PutBudget saves a global budget in the global_budgets table under its id.
func (jsonRepository) PutBudget(ctx context.Context, b GlobalBudget) error {
	return store.Put(ctx, budgetsTable, b.ID, b)
}

// DeleteBudget removes one global budget from the global_budgets table.
func (jsonRepository) DeleteBudget(ctx context.Context, id string) error {
	return store.Delete(ctx, budgetsTable, id)
}

// GetAllocation reads one allocation from the budget_allocations table.
func (jsonRepository) GetAllocation(ctx context.Context, id string) (Allocation, bool, error) {
	return store.Get[Allocation](ctx, allocationsTable, id)
}

// ListAllocations reads the whole budget_allocations table.
func (jsonRepository) ListAllocations(ctx context.Context) ([]Allocation, error) {
	return store.List[Allocation](ctx, allocationsTable)
}

// AllocationsOf reads every allocation to find those of one global budget, which is quick for the few
// global budgets of a year. PostgreSQL would use an index instead.
func (r jsonRepository) AllocationsOf(ctx context.Context, globalBudgetID string) ([]Allocation, error) {
	all, err := r.ListAllocations(ctx)
	if err != nil {
		return nil, err
	}
	of := []Allocation{}
	for _, a := range all {
		if a.GlobalBudgetID == globalBudgetID {
			of = append(of, a)
		}
	}
	return of, nil
}

// AllocationOfPlan reads every allocation to find the one of a plan (at most one, 014 data model §1.5).
func (r jsonRepository) AllocationOfPlan(ctx context.Context, planID string) (Allocation, bool, error) {
	all, err := r.ListAllocations(ctx)
	if err != nil {
		return Allocation{}, false, err
	}
	for _, a := range all {
		if a.PlanID == planID {
			return a, true, nil
		}
	}
	return Allocation{}, false, nil
}

// PutAllocation saves an allocation in the budget_allocations table under its id.
func (jsonRepository) PutAllocation(ctx context.Context, a Allocation) error {
	return store.Put(ctx, allocationsTable, a.ID, a)
}

// DeleteAllocation removes one allocation from the budget_allocations table.
func (jsonRepository) DeleteAllocation(ctx context.Context, id string) error {
	return store.Delete(ctx, allocationsTable, id)
}
