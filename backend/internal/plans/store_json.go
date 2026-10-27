package plans

import (
	"context"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/store"
)

// plansTable is the JSON store table of this package.
const plansTable = "plans"

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// Get reads one plan, with the defaults of older data (016 data-model §4.5).
func (JSONStore) Get(ctx context.Context, id string) (Plan, bool, error) {
	p, found, err := store.Get[Plan](ctx, plansTable, id)
	if !found {
		return p, found, err
	}
	return p.Normalised(), found, err
}

// List reads the whole plans table, each plan with the defaults of older data.
func (JSONStore) List(ctx context.Context) ([]Plan, error) {
	plans, err := store.List[Plan](ctx, plansTable)
	for i, p := range plans {
		plans[i] = p.Normalised()
	}
	return plans, err
}

func (r JSONStore) GetInScope(ctx context.Context, scope authz.Scope, id string) (Plan, bool, error) {
	p, found, err := r.Get(ctx, id)
	if err != nil || !found || !scope.Sees(AccessFacts(p)) {
		return Plan{}, false, err
	}
	return p, true, nil
}

// ListInScope reads every plan and keeps those the scope sees, as plan_in_scope() does in SQL.
func (r JSONStore) ListInScope(ctx context.Context, scope authz.Scope) ([]Plan, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	seen := []Plan{}
	for _, p := range all {
		if scope.Sees(AccessFacts(p)) {
			seen = append(seen, p)
		}
	}
	return seen, nil
}

// PlansOfYear reads every plan to find those of one year: quick for the plans of a demo. PostgreSQL would
// use an index.
func (r JSONStore) PlansOfYear(ctx context.Context, year int) ([]Plan, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	ofYear := []Plan{}
	for _, p := range all {
		if p.Year == year {
			ofYear = append(ofYear, p)
		}
	}
	return ofYear, nil
}

// Put saves a plan under its id.
func (JSONStore) Put(ctx context.Context, p Plan) error {
	return store.Put(ctx, plansTable, p.ID, p)
}

// Delete removes one plan.
func (JSONStore) Delete(ctx context.Context, id string) error {
	return store.Delete(ctx, plansTable, id)
}
