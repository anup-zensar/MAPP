package plans

import (
	"context"

	"northstar/internal/platform/store"
)

// plansTable is the JSON store table of this package.
const plansTable = "plans"

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// Get reads one plan.
func (jsonRepository) Get(ctx context.Context, id string) (Plan, bool, error) {
	return store.Get[Plan](ctx, plansTable, id)
}

// List reads the whole plans table.
func (jsonRepository) List(ctx context.Context) ([]Plan, error) {
	return store.List[Plan](ctx, plansTable)
}

// PlansOfYear reads every plan to find those of one year: quick for the plans of a demo. PostgreSQL would
// use an index.
func (r jsonRepository) PlansOfYear(ctx context.Context, year int) ([]Plan, error) {
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
func (jsonRepository) Put(ctx context.Context, p Plan) error {
	return store.Put(ctx, plansTable, p.ID, p)
}

// Delete removes one plan.
func (jsonRepository) Delete(ctx context.Context, id string) error {
	return store.Delete(ctx, plansTable, id)
}
