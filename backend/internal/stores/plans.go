package stores

import (
	"context"

	"mapp/internal/plans"
	"mapp/internal/platform/authz"
)

// PlanStore is the plans store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type PlanStore interface {
	plans.Store
	// GetInScope returns the plan with id when the scope sees it, and false when it does not or there is none.
	GetInScope(ctx context.Context, scope authz.Scope, id string) (plans.Plan, bool, error)
	// PlansOfYear returns the plans of year, any status (005's moves and copies stay inside a year).
	PlansOfYear(ctx context.Context, year int) ([]plans.Plan, error)
}
