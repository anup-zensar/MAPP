package workbook

import (
	"context"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store reads only imports of plans the scope sees, newest first; ListImports without their problems.
type Store interface {
	AddImport(ctx context.Context, imp Import) error
	ListImports(ctx context.Context, scope authz.Scope, planID string, limit int) ([]Import, error)
	GetImport(ctx context.Context, scope authz.Scope, id string) (Import, bool, error)
	MarkUndone(ctx context.Context, id string, by actor.Actor, at time.Time) error
}
