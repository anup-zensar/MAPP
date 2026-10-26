package activities

import (
	"context"

	"mapp/internal/platform/authz"
)

// Repository stores activities. The JSON store implements it now (store_json.go) and PostgreSQL can later
// (index on activities(initiative_id)). Every method runs inside the caller's transaction, which the context
// carries. Lists are ordered by id (creation order) and never nil; activities in the recycle bin are not in
// the table.
type Repository interface {
	// Get returns the activity with id, and false when there is none.
	Get(ctx context.Context, id string) (Activity, bool, error)
	// List returns every activity (008's inbox and usage counts).
	List(ctx context.Context) ([]Activity, error)
	// ListInScope returns the activities the scope sees (authz.Scope.SeesActivity): the lists people read.
	ListInScope(ctx context.Context, scope authz.Scope) ([]Activity, error)
	// ListByInitiative returns the activities of the initiative initiativeID.
	ListByInitiative(ctx context.Context, initiativeID string) ([]Activity, error)
	// ListByInitiatives returns the activities of every initiative in initiativeIDs (a plan's or a
	// campaign's activities: pass their initiatives).
	ListByInitiatives(ctx context.Context, initiativeIDs []string) ([]Activity, error)
	// Put saves a, replacing the activity with the same id.
	Put(ctx context.Context, a Activity) error
	// Delete removes the activity with id; removing one that does not exist changes nothing.
	Delete(ctx context.Context, id string) error
}
