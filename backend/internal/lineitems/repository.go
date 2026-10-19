package lineitems

import "context"

// Repository stores line items. The JSON store implements it now (store_json.go) and PostgreSQL can later
// (index on line_items(activity_id)). Every method runs inside the caller's transaction, which the context
// carries. Lists are ordered by id — the order the line items were created in — and never nil; line items in
// the recycle bin are not in the table.
type Repository interface {
	// Get returns the line item with id, and false when there is none.
	Get(ctx context.Context, id string) (LineItem, bool, error)
	// List returns every line item (usage counts of feature 002).
	List(ctx context.Context) ([]LineItem, error)
	// ListByActivity returns the line items of the activity activityID.
	ListByActivity(ctx context.Context, activityID string) ([]LineItem, error)
	// ListByActivities returns the line items of every activity in activityIDs (summaries of many
	// activities at once).
	ListByActivities(ctx context.Context, activityIDs []string) ([]LineItem, error)
	// Put saves li, replacing the line item with the same id.
	Put(ctx context.Context, li LineItem) error
	// Delete removes the line item with id; removing one that does not exist changes nothing.
	Delete(ctx context.Context, id string) error
}
