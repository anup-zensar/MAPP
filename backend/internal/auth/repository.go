package auth

import "context"

// Repository stores sessions. The JSON store implements it now (store_json.go) and PostgreSQL will later,
// without any change to the service. Every method runs inside the caller's transaction, which the context
// carries.
type Repository interface {
	// Get returns the session with id, and false when there is none.
	Get(ctx context.Context, id string) (Session, bool, error)
	// Put saves s, replacing the session with the same id.
	Put(ctx context.Context, s Session) error
	// Delete removes the session with id. Deleting a session that does not exist changes nothing.
	Delete(ctx context.Context, id string) error
	// List returns every session, of every person.
	List(ctx context.Context) ([]Session, error)
}
