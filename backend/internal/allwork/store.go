package allwork

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store keeps saved views and remembered columns; ListViews is ordered by id, never nil.
type Store interface {
	GetPreferences(ctx context.Context, personID string) (PreferencesRecord, bool, error)
	PutPreferences(ctx context.Context, p PreferencesRecord) error
	GetView(ctx context.Context, id string) (StoredView, bool, error)
	ListViews(ctx context.Context, personID string) ([]StoredView, error)
	PutView(ctx context.Context, v StoredView) error
	DeleteView(ctx context.Context, id string) error
}

var _ Store = (*PostgresStore)(nil)
