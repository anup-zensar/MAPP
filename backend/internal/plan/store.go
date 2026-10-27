package plan

import (
	"context"

	"mapp/internal/platform/authz"
)

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists plans ordered by id, never nil; ListInScope keeps those the scope sees (authz.Scope.Sees).
type Store interface {
	Get(ctx context.Context, id string) (Plan, bool, error)
	List(ctx context.Context) ([]Plan, error)
	ListInScope(ctx context.Context, scope authz.Scope) ([]Plan, error)
	ListByIDs(ctx context.Context, ids []string) ([]Plan, error)
	Put(ctx context.Context, p Plan) error
	Delete(ctx context.Context, id string) error
}

var _ Store = (*PostgresStore)(nil)
