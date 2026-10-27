package activity

import (
	"context"
	"slices"

	"mapp/internal/platform/store"
)

// activitiesTable is the JSON store table of this package.
const activitiesTable = "activities"

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// Get reads one activity.
func (JSONStore) Get(ctx context.Context, id string) (Activity, bool, error) {
	return store.Get[Activity](ctx, activitiesTable, id)
}

// List reads the whole activities table.
func (JSONStore) List(ctx context.Context) ([]Activity, error) {
	return store.List[Activity](ctx, activitiesTable)
}

// ListSplitFrom reads every activity to find those split off one.
func (r JSONStore) ListSplitFrom(ctx context.Context, activityID string) ([]Activity, error) {
	all, err := r.List(ctx)
	return slices.DeleteFunc(all, func(a Activity) bool { return a.SplitFromID != activityID }), err
}

// ListByInitiative reads every activity to find those of one initiative.
func (r JSONStore) ListByInitiative(ctx context.Context, initiativeID string) ([]Activity, error) {
	return r.ListByInitiatives(ctx, []string{initiativeID})
}

// ListByInitiatives reads every activity to find those of the initiatives: quick for the records of a demo.
func (r JSONStore) ListByInitiatives(ctx context.Context, initiativeIDs []string) ([]Activity, error) {
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
func (JSONStore) Put(ctx context.Context, a Activity) error {
	return store.Put(ctx, activitiesTable, a.ID, a)
}

// Delete removes one activity.
func (JSONStore) Delete(ctx context.Context, id string) error {
	return store.Delete(ctx, activitiesTable, id)
}
