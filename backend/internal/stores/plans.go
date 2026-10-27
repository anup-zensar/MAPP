package stores

import "mapp/internal/plans"

// PlanStore is the plans store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type PlanStore interface {
	plans.Store
}
