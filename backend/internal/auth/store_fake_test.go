package auth

import (
	"context"

	"mapp/internal/platform/fake"
)

// fakeStore is authtest.Store for this package's own tests, which cannot import authtest.
type fakeStore struct {
	sessions *fake.Table[Session]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{sessions: fake.TableOf(db, "sessions", func(s Session) string { return s.ID })}
}

func (s fakeStore) Get(_ context.Context, id string) (Session, bool, error) {
	return s.sessions.Get(id)
}

func (s fakeStore) Put(ctx context.Context, session Session) error {
	return s.sessions.Put(ctx, session)
}

func (s fakeStore) Delete(ctx context.Context, id string) error {
	return s.sessions.Delete(ctx, id)
}

func (s fakeStore) List(context.Context) ([]Session, error) {
	return s.sessions.List()
}
