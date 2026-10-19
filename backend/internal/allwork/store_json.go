package allwork

import (
	"context"

	"northstar/internal/platform/store"
)

// The JSON store tables of this package (data model §1).
const (
	preferencesTable = "all_work_preferences"
	viewsTable       = "saved_views"
)

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// GetPreferences reads one person's preferences.
func (jsonRepository) GetPreferences(ctx context.Context, personID string) (PreferencesRecord, bool, error) {
	return store.Get[PreferencesRecord](ctx, preferencesTable, personID)
}

// PutPreferences saves one person's preferences under their person ID.
func (jsonRepository) PutPreferences(ctx context.Context, p PreferencesRecord) error {
	return store.Put(ctx, preferencesTable, p.PersonID, p)
}

// GetView reads one saved view.
func (jsonRepository) GetView(ctx context.Context, id string) (StoredView, bool, error) {
	return store.Get[StoredView](ctx, viewsTable, id)
}

// ListViews reads every saved view to find one person's: quick for the views of a few people. PostgreSQL
// would use an index on person_id.
func (jsonRepository) ListViews(ctx context.Context, personID string) ([]StoredView, error) {
	all, err := store.List[StoredView](ctx, viewsTable)
	if err != nil {
		return nil, err
	}
	own := []StoredView{}
	for _, v := range all {
		if v.PersonID == personID {
			own = append(own, v)
		}
	}
	return own, nil
}

// PutView saves a view under its id.
func (jsonRepository) PutView(ctx context.Context, v StoredView) error {
	return store.Put(ctx, viewsTable, v.ID, v)
}

// DeleteView removes one view.
func (jsonRepository) DeleteView(ctx context.Context, id string) error {
	return store.Delete(ctx, viewsTable, id)
}
