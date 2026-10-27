package stores

import (
	"context"

	"mapp/internal/people"
)

// PeopleStore is the people and teams store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type PeopleStore interface {
	people.Store
	// FindByEmail returns the person whose stored e-mail is email (people's e-mails are stored in lower
	// case), and false when there is none.
	FindByEmail(ctx context.Context, email string) (people.Person, bool, error)
	// ListPreferences returns the saved preferences of everyone who changed one, ordered by person id.
	ListPreferences(ctx context.Context) ([]people.Preferences, error)
}
