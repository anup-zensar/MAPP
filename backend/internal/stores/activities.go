package stores

import "mapp/internal/activities"

// ActivityStore is the activities store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type ActivityStore interface {
	activities.Store
}
