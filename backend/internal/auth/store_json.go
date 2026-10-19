package auth

import (
	"context"

	"northstar/internal/platform/store"
)

// sessionsTable is the JSON store table of this package.
const sessionsTable = "sessions"

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// Get reads one session from the sessions table.
func (jsonRepository) Get(ctx context.Context, id string) (Session, bool, error) {
	return store.Get[Session](ctx, sessionsTable, id)
}

// Put saves a session in the sessions table under its id.
func (jsonRepository) Put(ctx context.Context, s Session) error {
	return store.Put(ctx, sessionsTable, s.ID, s)
}

// Delete removes one session from the sessions table.
func (jsonRepository) Delete(ctx context.Context, id string) error {
	return store.Delete(ctx, sessionsTable, id)
}

// List reads the whole sessions table.
func (jsonRepository) List(ctx context.Context) ([]Session, error) {
	return store.List[Session](ctx, sessionsTable)
}
