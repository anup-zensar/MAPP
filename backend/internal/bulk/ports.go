package bulk

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// Records reads the selection; Find answers not_found for a record the person cannot see.
type Records interface {
	Find(ctx context.Context, a actor.Actor, id string) (Record, error)
	Below(ctx context.Context, r Record) (Below, error)
	Fields(ctx context.Context, level Level) ([]Field, error)
	Destinations(ctx context.Context, a actor.Actor, action Kind, r Record) ([]Target, error)
	Consequences(ctx context.Context, a actor.Actor, r Record, targetID string) (Consequences, error)
	Person(ctx context.Context, id string) (name string, active, found bool, err error)
}

// Changes runs the owning areas' single operations; SetOwner and SetField report false when nothing changed.
type Changes interface {
	SetOwner(ctx context.Context, a actor.Actor, r Record, ownerID string) (bool, error)
	SetField(ctx context.Context, a actor.Actor, r Record, change SetField) (bool, error)
	Submit(ctx context.Context, a actor.Actor, r Record, comment string) error
	Copy(ctx context.Context, a actor.Actor, r Record, targetID string) (string, error)
	Move(ctx context.Context, a actor.Actor, r Record, targetID string) error
	Delete(ctx context.Context, a actor.Actor, r Record) (history.BinEntry, error)
}

type SetField struct {
	Field           Field
	Value           *FieldValue
	RestartApproval bool
}

type Planning interface {
	Records
	Changes
}
