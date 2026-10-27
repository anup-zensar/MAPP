package budget

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists records ordered by id, never nil; a record in the recycle bin is not stored.
type Store interface {
	GetBudget(ctx context.Context, id string) (GlobalBudget, bool, error)
	ListBudgets(ctx context.Context) ([]GlobalBudget, error)
	PutBudget(ctx context.Context, b GlobalBudget) error
	DeleteBudget(ctx context.Context, id string) error

	GetAllocation(ctx context.Context, id string) (Allocation, bool, error)
	ListAllocations(ctx context.Context) ([]Allocation, error)
	AllocationsOf(ctx context.Context, globalBudgetID string) ([]Allocation, error)
	AllocationOfPlan(ctx context.Context, planID string) (Allocation, bool, error)
	PutAllocation(ctx context.Context, a Allocation) error
	DeleteAllocation(ctx context.Context, id string) error
}
