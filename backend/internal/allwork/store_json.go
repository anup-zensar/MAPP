package allwork

import (
	"context"

	"mapp/internal/platform/store"
)

// The JSON store tables of this package (data model §1).
const (
	preferencesTable = "all_work_preferences"
	viewsTable       = "saved_views"
)

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// GetPreferences reads one person's preferences.
func (JSONStore) GetPreferences(ctx context.Context, personID string) (PreferencesRecord, bool, error) {
	return store.Get[PreferencesRecord](ctx, preferencesTable, personID)
}

// PutPreferences saves one person's preferences under their person ID.
func (JSONStore) PutPreferences(ctx context.Context, p PreferencesRecord) error {
	return store.Put(ctx, preferencesTable, p.PersonID, p)
}

// GetView reads one saved view.
func (JSONStore) GetView(ctx context.Context, id string) (StoredView, bool, error) {
	return store.Get[StoredView](ctx, viewsTable, id)
}

// ListViews reads every saved view to find one person's: quick for the views of a few people. PostgreSQL
// would use an index on person_id.
func (JSONStore) ListViews(ctx context.Context, personID string) ([]StoredView, error) {
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
func (JSONStore) PutView(ctx context.Context, v StoredView) error {
	return store.Put(ctx, viewsTable, v.ID, v)
}

// DeleteView removes one view.
func (JSONStore) DeleteView(ctx context.Context, id string) error {
	return store.Delete(ctx, viewsTable, id)
}
