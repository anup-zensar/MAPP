package stores

import "mapp/internal/access"

// AccessStore is the access store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type AccessStore interface {
	access.Store
}
