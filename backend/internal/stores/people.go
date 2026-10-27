package stores

import "mapp/internal/people"

// PeopleStore is the people and teams store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type PeopleStore interface {
	people.Store
}
