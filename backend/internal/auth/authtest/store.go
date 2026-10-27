// Package authtest is the in-memory fake of the sessions store, for tests that build services or the whole
// API without a database.
package authtest

import (
	"context"

	"mapp/internal/auth"
	"mapp/internal/platform/fake"
)

type Store struct {
	sessions *fake.Table[auth.Session]
}

func NewStore(db *fake.DB) Store {
	return Store{sessions: fake.TableOf(db, "sessions", func(s auth.Session) string { return s.ID })}
}

func (s Store) Get(_ context.Context, id string) (auth.Session, bool, error) {
	return s.sessions.Get(id)
}

func (s Store) Put(ctx context.Context, session auth.Session) error {
	return s.sessions.Put(ctx, session)
}

func (s Store) Delete(ctx context.Context, id string) error {
	return s.sessions.Delete(ctx, id)
}

func (s Store) List(context.Context) ([]auth.Session, error) {
	return s.sessions.List()
}
