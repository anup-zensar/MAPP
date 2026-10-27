package stores

import "mapp/internal/financials"

// FinancialsStore is the financials store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type FinancialsStore interface {
	financials.Store
}
