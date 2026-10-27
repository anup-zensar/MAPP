package allwork

import (
	"context"

	"mapp/internal/platform/fake"
)

// fakeStore is allworktest.Store for this package's own tests, which cannot import allworktest.
type fakeStore struct {
	preferences *fake.Table[PreferencesRecord]
	views       *fake.Table[StoredView]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{
		preferences: fake.TableOf(db, "all_work_preferences", func(p PreferencesRecord) string { return p.PersonID }),
		views:       fake.TableOf(db, "saved_views", func(v StoredView) string { return v.ID }),
	}
}

func (s fakeStore) GetPreferences(_ context.Context, personID string) (PreferencesRecord, bool, error) {
	return s.preferences.Get(personID)
}

func (s fakeStore) PutPreferences(ctx context.Context, p PreferencesRecord) error {
	return s.preferences.Put(ctx, p)
}

func (s fakeStore) GetView(_ context.Context, id string) (StoredView, bool, error) {
	return s.views.Get(id)
}

func (s fakeStore) ListViews(_ context.Context, personID string) ([]StoredView, error) {
	return s.views.Where(func(v StoredView) bool { return v.PersonID == personID })
}

func (s fakeStore) PutView(ctx context.Context, v StoredView) error {
	return s.views.Put(ctx, v)
}

func (s fakeStore) DeleteView(ctx context.Context, id string) error {
	return s.views.Delete(ctx, id)
}
