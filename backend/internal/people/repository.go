package people

import "context"

// Repository stores people and their preferences. The JSON store implements it now (store_json.go) and
// PostgreSQL will later, without any change to the service. Every method runs inside the caller's
// transaction, which the context carries.
type Repository interface {
	// Get returns the person with id, and false when there is none.
	Get(ctx context.Context, id string) (Person, bool, error)
	// FindByEmail returns the person whose stored e-mail is email (people's e-mails are stored in lower
	// case), and false when there is none.
	FindByEmail(ctx context.Context, email string) (Person, bool, error)
	// List returns every person, active or not, ordered by id.
	List(ctx context.Context) ([]Person, error)
	// Put saves p, replacing the person with the same id.
	Put(ctx context.Context, p Person) error
	// GetPreferences returns the preferences of a person, and false when they have none saved.
	GetPreferences(ctx context.Context, personID string) (Preferences, bool, error)
	// PutPreferences saves the preferences of p.PersonID.
	PutPreferences(ctx context.Context, p Preferences) error
}
