package activity

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists activities ordered by id, never nil; an activity in the recycle bin is not stored.
type Store interface {
	Get(ctx context.Context, id string) (Activity, bool, error)
	List(ctx context.Context) ([]Activity, error)
	ListByIDs(ctx context.Context, ids []string) ([]Activity, error)
	ListByInitiative(ctx context.Context, initiativeID string) ([]Activity, error)
	ListByInitiatives(ctx context.Context, initiativeIDs []string) ([]Activity, error)
	Put(ctx context.Context, a Activity) error
	Delete(ctx context.Context, id string) error
	ListSplitFrom(ctx context.Context, activityID string) ([]Activity, error)
	AddMany(ctx context.Context, list []Activity) error
	DeleteMany(ctx context.Context, ids []string) error
}
