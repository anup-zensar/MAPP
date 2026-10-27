package auth

import (
	"context"

	"mapp/internal/platform/store"
)

// sessionsTable is the JSON store table of this package.
const sessionsTable = "sessions"

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// Get reads one session from the sessions table.
func (JSONStore) Get(ctx context.Context, id string) (Session, bool, error) {
	return store.Get[Session](ctx, sessionsTable, id)
}

// Put saves a session in the sessions table under its id.
func (JSONStore) Put(ctx context.Context, s Session) error {
	return store.Put(ctx, sessionsTable, s.ID, s)
}

// Delete removes one session from the sessions table.
func (JSONStore) Delete(ctx context.Context, id string) error {
	return store.Delete(ctx, sessionsTable, id)
}

// List reads the whole sessions table.
func (JSONStore) List(ctx context.Context) ([]Session, error) {
	return store.List[Session](ctx, sessionsTable)
}
