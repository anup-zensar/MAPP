package plans

import "context"

// Repository stores plans. The JSON store implements it now (store_json.go) and PostgreSQL can later.
// Every method runs inside the caller's transaction, which the context carries. Lists are ordered by id and
// never nil; plans in the recycle bin are not in the table.
type Repository interface {
	// Get returns the plan with id, and false when there is none.
	Get(ctx context.Context, id string) (Plan, bool, error)
	// List returns every plan.
	List(ctx context.Context) ([]Plan, error)
	// PlansOfYear returns the plans of year, any status (005's moves and copies stay inside a year).
	PlansOfYear(ctx context.Context, year int) ([]Plan, error)
	// Put saves p, replacing the plan with the same id.
	Put(ctx context.Context, p Plan) error
	// Delete removes the plan with id; removing one that does not exist changes nothing.
	Delete(ctx context.Context, id string) error
}
