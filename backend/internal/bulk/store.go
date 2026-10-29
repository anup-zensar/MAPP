package bulk

import (
	"context"
	"time"

	"mapp/internal/platform/actor"
)

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store keeps bulk actions; GetAction finds one only for the person who ran it.
type Store interface {
	AddAction(ctx context.Context, action Action) error
	GetAction(ctx context.Context, id, actorID string) (Action, bool, error)
	MarkUndone(ctx context.Context, id string, by actor.Actor, at time.Time) error
}
