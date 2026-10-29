// Package bulktest is the in-memory fake of the bulk actions store, for tests that build the service or the
// whole API without a database.
package bulktest

import (
	"context"
	"time"

	"mapp/internal/bulk"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/fake"
)

type Store struct {
	actions *fake.Table[bulk.Action]
}

func NewStore(db *fake.DB) Store {
	return Store{actions: fake.TableOf(db, "bulk_actions", func(a bulk.Action) string { return a.ID })}
}

func (s Store) AddAction(ctx context.Context, action bulk.Action) error {
	if action.BinEntryIDs == nil {
		action.BinEntryIDs = []string{}
	}
	return s.actions.Put(ctx, action)
}

func (s Store) GetAction(_ context.Context, id, actorID string) (bulk.Action, bool, error) {
	action, found, err := s.actions.Get(id)
	if err != nil || !found || action.Actor.ID != actorID {
		return bulk.Action{}, false, err
	}
	return action, true, nil
}

func (s Store) MarkUndone(ctx context.Context, id string, by actor.Actor, at time.Time) error {
	action, found, err := s.actions.Get(id)
	if err != nil || !found {
		return err
	}
	action.UndoneAt, action.UndoneBy = &at, &by
	return s.actions.Put(ctx, action)
}
