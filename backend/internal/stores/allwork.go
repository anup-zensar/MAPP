package stores

import "mapp/internal/allwork"

// AllWorkStore is the All work store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type AllWorkStore interface {
	allwork.Store
}
