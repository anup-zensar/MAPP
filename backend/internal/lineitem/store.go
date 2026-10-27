package lineitem

import "context"

// Store stores line items. The JSON store implements it now (store_json.go) and PostgreSQL can later
// (index on line_items(activity_id)). Every method runs inside the caller's transaction, which the context
// carries. Lists are ordered by id — the order the line items were created in — and never nil; line items in
// the recycle bin are not in the table.
type Store interface {
	// Get returns the line item with id, and false when there is none.
	Get(ctx context.Context, id string) (LineItem, bool, error)
	// ListByActivity returns the line items of the activity activityID.
	ListByActivity(ctx context.Context, activityID string) ([]LineItem, error)
	// ListByActivities returns the line items of every activity in activityIDs (summaries of many
	// activities at once).
	ListByActivities(ctx context.Context, activityIDs []string) ([]LineItem, error)
	// Put saves li, replacing the line item with the same id.
	Put(ctx context.Context, li LineItem) error
	// Delete removes the line item with id; removing one that does not exist changes nothing.
	Delete(ctx context.Context, id string) error
	// AddMany saves new line items, whose ids are not stored yet, all at once.
	AddMany(ctx context.Context, list []LineItem) error
	// ListByIDs returns the line items with ids that are stored.
	ListByIDs(ctx context.Context, ids []string) ([]LineItem, error)
	// DeleteMany removes the line items with ids all at once.
	DeleteMany(ctx context.Context, ids []string) error
}
