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
	// ListInScopeOf returns the activities among ids that the scope sees, in the order of their ids.
	ListInScopeOf(ctx context.Context, scope authz.Scope, ids []string) ([]Activity, error)
	// WorkflowIDs returns the Approval workflow field of every stored activity by its id; "" when empty.
	WorkflowIDs(ctx context.Context) (map[string]string, error)
	// ListByIDs returns the stored activities among ids, at once, in the order of their ids.
	ListByIDs(ctx context.Context, ids []string) ([]Activity, error)
	// ListByInitiative returns the activities of the initiative initiativeID.
	ListByInitiative(ctx context.Context, initiativeID string) ([]Activity, error)
	// ListByInitiatives returns the activities of every initiative in initiativeIDs (a plan's or a
	// campaign's activities: pass their initiatives).
	ListByInitiatives(ctx context.Context, initiativeIDs []string) ([]Activity, error)
	// Put saves a, replacing the activity with the same id.
	Put(ctx context.Context, a Activity) error
	// Delete removes the activity with id; removing one that does not exist changes nothing.
	Delete(ctx context.Context, id string) error
	// ListSplitFrom returns the activities split off the activity activityID.
	ListSplitFrom(ctx context.Context, activityID string) ([]Activity, error)
	// AddMany saves new activities, whose ids are not stored yet, all at once.
	AddMany(ctx context.Context, list []Activity) error
	// DeleteMany removes the activities with ids all at once.
	DeleteMany(ctx context.Context, ids []string) error
	// CountOwnedBy counts the stored activities the person owns.
	CountOwnedBy(ctx context.Context, personID string) (int, error)
}
