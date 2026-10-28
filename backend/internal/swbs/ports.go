package swbs

import (
	"context"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

type Candidate struct {
	Request
	Ready bool
}

type Receiver interface {
	Receive(ctx context.Context, a actor.Actor, answers []Answer, at time.Time) error
}

// LineItems (internal/app): Candidates of the activities scope reaches, only ids when not nil, in any order.
type LineItems interface {
	Candidates(ctx context.Context, scope authz.Scope, ids []string) ([]Candidate, error)
	Receiver
}
