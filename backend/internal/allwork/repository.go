package allwork

import "context"

// Repository stores All work's personal settings: saved views and the remembered columns (data model §1).
// The JSON store implements it now (store_json.go) and PostgreSQL can later. Every method runs inside the
// caller's transaction, which the context carries. Lists are never nil.
type Repository interface {
	// GetPreferences returns the preferences of the person personID, and false when they never saved any.
	GetPreferences(ctx context.Context, personID string) (PreferencesRecord, bool, error)
	// PutPreferences saves p under p.PersonID, replacing what was saved.
	PutPreferences(ctx context.Context, p PreferencesRecord) error
	// GetView returns the saved view with id, and false when there is none.
	GetView(ctx context.Context, id string) (StoredView, bool, error)
	// ListViews returns the saved views of the person personID, ordered by id.
	ListViews(ctx context.Context, personID string) ([]StoredView, error)
	// PutView saves v, replacing the view with the same id.
	PutView(ctx context.Context, v StoredView) error
	// DeleteView removes the view with id; removing one that does not exist changes nothing.
	DeleteView(ctx context.Context, id string) error
}
