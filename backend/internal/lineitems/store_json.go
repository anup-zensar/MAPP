package lineitems

import (
	"context"
	"slices"

	"northstar/internal/platform/store"
)

// lineItemsTable is the JSON store table of this package.
const lineItemsTable = "line_items"

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// Get reads one line item.
func (jsonRepository) Get(ctx context.Context, id string) (LineItem, bool, error) {
	return store.Get[LineItem](ctx, lineItemsTable, id)
}

// List reads the whole line_items table.
func (jsonRepository) List(ctx context.Context) ([]LineItem, error) {
	return store.List[LineItem](ctx, lineItemsTable)
}

// ListByActivity reads every line item to find those of one activity.
func (r jsonRepository) ListByActivity(ctx context.Context, activityID string) ([]LineItem, error) {
	return r.ListByActivities(ctx, []string{activityID})
}

// ListByActivities reads every line item to find those of the activities: quick for the records of a demo.
func (r jsonRepository) ListByActivities(ctx context.Context, activityIDs []string) ([]LineItem, error) {
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
func (jsonRepository) Put(ctx context.Context, li LineItem) error {
	return store.Put(ctx, lineItemsTable, li.ID, li)
}

// Delete removes one line item.
func (jsonRepository) Delete(ctx context.Context, id string) error {
	return store.Delete(ctx, lineItemsTable, id)
}
