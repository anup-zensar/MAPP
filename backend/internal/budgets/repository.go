package budgets

import "context"

// Repository stores global budgets and allocations. The JSON store implements it now (store_json.go) and
// PostgreSQL can later. Every method runs inside the caller's transaction, which the context carries. Lists
// are ordered by id and never nil; records in the recycle bin are not in the tables.
type Repository interface {
	// GetBudget returns the global budget with id, and false when there is none.
	GetBudget(ctx context.Context, id string) (GlobalBudget, bool, error)
	// ListBudgets returns every global budget.
	ListBudgets(ctx context.Context) ([]GlobalBudget, error)
	// PutBudget saves b, replacing the global budget with the same id.
	PutBudget(ctx context.Context, b GlobalBudget) error
	// DeleteBudget removes the global budget with id; removing one that does not exist changes nothing.
	DeleteBudget(ctx context.Context, id string) error

	// GetAllocation returns the allocation with id, and false when there is none.
	GetAllocation(ctx context.Context, id string) (Allocation, bool, error)
	// ListAllocations returns every allocation of every global budget.
	ListAllocations(ctx context.Context) ([]Allocation, error)
	// AllocationsOf returns the allocations of the global budget globalBudgetID.
	AllocationsOf(ctx context.Context, globalBudgetID string) ([]Allocation, error)
	// AllocationOfPlan returns the allocation of the plan planID, and false when it has none.
	AllocationOfPlan(ctx context.Context, planID string) (Allocation, bool, error)
	// PutAllocation saves a, replacing the allocation with the same id.
	PutAllocation(ctx context.Context, a Allocation) error
	// DeleteAllocation removes the allocation with id; removing one that does not exist changes nothing.
	DeleteAllocation(ctx context.Context, id string) error
}
