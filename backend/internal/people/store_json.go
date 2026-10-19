package people

import (
	"context"

	"northstar/internal/platform/store"
)

// The JSON store tables of this package.
const (
	peopleTable      = "people"
	preferencesTable = "preferences"
)

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// Get reads one person from the people table.
func (jsonRepository) Get(ctx context.Context, id string) (Person, bool, error) {
	return store.Get[Person](ctx, peopleTable, id)
}

// FindByEmail reads every person to find the e-mail, which is quick for the few people of a demo.
// PostgreSQL will use an index instead.
func (jsonRepository) FindByEmail(ctx context.Context, email string) (Person, bool, error) {
	everyone, err := store.List[Person](ctx, peopleTable)
	if err != nil {
		return Person{}, false, err
	}
	for _, p := range everyone {
		if p.Email == email {
			return p, true, nil
		}
	}
	return Person{}, false, nil
}

// List reads the whole people table.
func (jsonRepository) List(ctx context.Context) ([]Person, error) {
	return store.List[Person](ctx, peopleTable)
}

// Put saves a person in the people table under their id.
func (jsonRepository) Put(ctx context.Context, p Person) error {
	return store.Put(ctx, peopleTable, p.ID, p)
}

// GetPreferences reads one person's preferences from the preferences table.
func (jsonRepository) GetPreferences(ctx context.Context, personID string) (Preferences, bool, error) {
	return store.Get[Preferences](ctx, preferencesTable, personID)
}

// PutPreferences saves preferences in the preferences table under the person's id.
func (jsonRepository) PutPreferences(ctx context.Context, p Preferences) error {
	return store.Put(ctx, preferencesTable, p.PersonID, p)
}
