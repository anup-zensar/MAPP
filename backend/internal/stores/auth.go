package stores

import "mapp/internal/auth"

// AuthStore is the sign-in store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type AuthStore interface {
	auth.Store
}
