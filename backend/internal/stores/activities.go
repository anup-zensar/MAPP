package stores

import (
	"context"

	"mapp/internal/activity"
	"mapp/internal/platform/authz"
)

// ActivityStore is the activities store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type ActivityStore interface {
	activity.Store
	// ListInScope returns the activities the scope sees (authz.Scope.SeesActivity): the lists people read.
	ListInScope(ctx context.Context, scope authz.Scope) ([]activity.Activity, error)
	// ListInScopeOf returns the activities among ids that the scope sees, in the order of their ids.
	ListInScopeOf(ctx context.Context, scope authz.Scope, ids []string) ([]activity.Activity, error)
	// WorkflowIDs returns the Approval workflow field of every stored activity by its id; "" when empty.
	WorkflowIDs(ctx context.Context) (map[string]string, error)
	// CountOwnedBy counts the stored activities the person owns.
	CountOwnedBy(ctx context.Context, personID string) (int, error)
}
