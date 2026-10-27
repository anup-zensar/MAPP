package plan

import (
	"context"

	"mapp/internal/platform/authz"
)

// Store stores plans. The JSON store implements it now (store_json.go) and PostgreSQL can later.
// Every method runs inside the caller's transaction, which the context carries. Lists are ordered by id and
// never nil; plans in the recycle bin are not in the table.
type Store interface {
	// Get returns the plan with id, and false when there is none.
	Get(ctx context.Context, id string) (Plan, bool, error)
	// List returns every plan.
	List(ctx context.Context) ([]Plan, error)
	// ListInScope returns the plans the scope sees (authz.Scope.Sees): the lists people read.
	ListInScope(ctx context.Context, scope authz.Scope) ([]Plan, error)
	// ListByIDs returns the stored plans among ids, at once, in the order of their ids.
	ListByIDs(ctx context.Context, ids []string) ([]Plan, error)
	// Put saves p, replacing the plan with the same id.
	Put(ctx context.Context, p Plan) error
	// Delete removes the plan with id; removing one that does not exist changes nothing.
	Delete(ctx context.Context, id string) error
}

var (
	_ Store = (*PostgresStore)(nil)
	_ Store = JSONStore{}
)
