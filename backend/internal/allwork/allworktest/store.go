// Package allworktest is the in-memory fake of All work's store of saved views and preferences, for tests
// that build services or the whole API without a database.
package allworktest

import (
	"context"

	"mapp/internal/allwork"
	"mapp/internal/platform/fake"
)

type Store struct {
	preferences *fake.Table[allwork.PreferencesRecord]
	views       *fake.Table[allwork.StoredView]
}

func NewStore(db *fake.DB) Store {
	return Store{
		preferences: fake.TableOf(db, "all_work_preferences", func(p allwork.PreferencesRecord) string { return p.PersonID }),
		views:       fake.TableOf(db, "saved_views", func(v allwork.StoredView) string { return v.ID }),
	}
}

func (s Store) GetPreferences(_ context.Context, personID string) (allwork.PreferencesRecord, bool, error) {
	return s.preferences.Get(personID)
}

func (s Store) PutPreferences(ctx context.Context, p allwork.PreferencesRecord) error {
	return s.preferences.Put(ctx, p)
}

func (s Store) GetView(_ context.Context, id string) (allwork.StoredView, bool, error) {
	return s.views.Get(id)
}

func (s Store) ListViews(_ context.Context, personID string) ([]allwork.StoredView, error) {
	return s.views.Where(func(v allwork.StoredView) bool { return v.PersonID == personID })
}

func (s Store) PutView(ctx context.Context, v allwork.StoredView) error {
	return s.views.Put(ctx, v)
}

func (s Store) DeleteView(ctx context.Context, id string) error {
	return s.views.Delete(ctx, id)
}
