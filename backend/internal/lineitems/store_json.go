package lineitems

import (
	"context"
	"slices"

	"mapp/internal/platform/store"
)

// lineItemsTable is the JSON store table of this package.
const lineItemsTable = "line_items"

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// Get reads one line item.
func (JSONStore) Get(ctx context.Context, id string) (LineItem, bool, error) {
	return store.Get[LineItem](ctx, lineItemsTable, id)
}

// List reads the whole line_items table.
func (JSONStore) List(ctx context.Context) ([]LineItem, error) {
	return store.List[LineItem](ctx, lineItemsTable)
}

// ListByActivity reads every line item to find those of one activity.
func (r JSONStore) ListByActivity(ctx context.Context, activityID string) ([]LineItem, error) {
	return r.ListByActivities(ctx, []string{activityID})
}

// ListByActivities reads every line item to find those of the activities: quick for the records of a demo.
func (r JSONStore) ListByActivities(ctx context.Context, activityIDs []string) ([]LineItem, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	kept := []LineItem{}
	for _, li := range all {
		if slices.Contains(activityIDs, li.ActivityID) {
			kept = append(kept, li)
		}
	}
	return kept, nil
}

// Put saves a line item under its id.
func (JSONStore) Put(ctx context.Context, li LineItem) error {
	return store.Put(ctx, lineItemsTable, li.ID, li)
}

// Delete removes one line item.
func (JSONStore) Delete(ctx context.Context, id string) error {
	return store.Delete(ctx, lineItemsTable, id)
}
