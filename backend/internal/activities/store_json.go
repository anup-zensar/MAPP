package activities

import (
	"context"
	"slices"

	"northstar/internal/platform/store"
)

// activitiesTable is the JSON store table of this package.
const activitiesTable = "activities"

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// Get reads one activity.
func (jsonRepository) Get(ctx context.Context, id string) (Activity, bool, error) {
	return store.Get[Activity](ctx, activitiesTable, id)
}

// List reads the whole activities table.
func (jsonRepository) List(ctx context.Context) ([]Activity, error) {
	return store.List[Activity](ctx, activitiesTable)
}

// ListByInitiative reads every activity to find those of one initiative.
func (r jsonRepository) ListByInitiative(ctx context.Context, initiativeID string) ([]Activity, error) {
	return r.ListByInitiatives(ctx, []string{initiativeID})
}

// ListByInitiatives reads every activity to find those of the initiatives: quick for the records of a demo.
func (r jsonRepository) ListByInitiatives(ctx context.Context, initiativeIDs []string) ([]Activity, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	kept := []Activity{}
	for _, a := range all {
		if slices.Contains(initiativeIDs, a.InitiativeID) {
			kept = append(kept, a)
		}
	}
	return kept, nil
}

// Put saves an activity under its id.
func (jsonRepository) Put(ctx context.Context, a Activity) error {
	return store.Put(ctx, activitiesTable, a.ID, a)
}

// Delete removes one activity.
func (jsonRepository) Delete(ctx context.Context, id string) error {
	return store.Delete(ctx, activitiesTable, id)
}
