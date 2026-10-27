package plan

import (
	"context"
	"slices"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/fake"
)

// fakeStore is plantest.Store for this package's own tests, which cannot import plantest.
type fakeStore struct {
	plans *fake.Table[Plan]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{plans: fake.TableOf(db, "plans", func(p Plan) string { return p.ID })}
}

func (s fakeStore) Get(_ context.Context, id string) (Plan, bool, error) {
	return s.plans.Get(id)
}

func (s fakeStore) List(context.Context) ([]Plan, error) {
	return s.plans.List()
}

func (s fakeStore) ListInScope(_ context.Context, scope authz.Scope) ([]Plan, error) {
	return s.plans.Where(func(p Plan) bool { return scope.Sees(AccessFacts(p)) })
}

func (s fakeStore) ListByIDs(_ context.Context, ids []string) ([]Plan, error) {
	return s.plans.Where(func(p Plan) bool { return slices.Contains(ids, p.ID) })
}

func (s fakeStore) Put(ctx context.Context, p Plan) error {
	return s.plans.Put(ctx, p.Normalised())
}

func (s fakeStore) Delete(ctx context.Context, id string) error {
	return s.plans.Delete(ctx, id)
}
