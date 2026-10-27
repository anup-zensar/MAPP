package lineitem

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists line items ordered by id, never nil; a line item in the recycle bin is not stored.
type Store interface {
	Get(ctx context.Context, id string) (LineItem, bool, error)
	ListByActivity(ctx context.Context, activityID string) ([]LineItem, error)
	ListByActivities(ctx context.Context, activityIDs []string) ([]LineItem, error)
	Put(ctx context.Context, li LineItem) error
	Delete(ctx context.Context, id string) error
	AddMany(ctx context.Context, list []LineItem) error
	ListByIDs(ctx context.Context, ids []string) ([]LineItem, error)
	DeleteMany(ctx context.Context, ids []string) error
}
