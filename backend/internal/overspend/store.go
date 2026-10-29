package overspend

import (
	"context"

	"mapp/internal/platform/money"
)

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Budgets reads every budget with the planned cost under it, for whoever asks: it serves a rule, not a list.
type Budgets interface {
	PlanningBudgets(ctx context.Context) ([]Budget, error)
	GlobalBudgets(ctx context.Context) ([]GlobalBudget, error)
}

type Store interface {
	Budgets
	Configuration(ctx context.Context) (Configuration, error)
	PutConfiguration(ctx context.Context, c Configuration) error
}

type Rates interface {
	Rates(ctx context.Context) (money.Rates, error)
}
