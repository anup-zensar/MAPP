package access

import (
	"context"

	"mapp/internal/platform/fake"
)

// fakeStore is accesstest.Store for this package's own tests, which cannot import accesstest.
type fakeStore struct {
	roles       *fake.Table[Role]
	assignments *fake.Table[Assignment]
	rules       *fake.Table[Rules]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{
		roles:       fake.TableOf(db, "roles", func(r Role) string { return r.ID }),
		assignments: fake.TableOf(db, "role_assignments", func(a Assignment) string { return a.ID }),
		rules:       fake.TableOf(db, "access_rules", func(r Rules) string { return r.ID }),
	}
}

func (s fakeStore) GetRole(_ context.Context, id string) (Role, bool, error) {
	return s.roles.Get(id)
}

func (s fakeStore) ListRoles(context.Context) ([]Role, error) {
	return s.roles.List()
}

func (s fakeStore) PutRole(ctx context.Context, r Role) error {
	return s.roles.Put(ctx, r)
}

func (s fakeStore) DeleteRole(ctx context.Context, id string) error {
	return s.roles.Delete(ctx, id)
}

func (s fakeStore) ListAssignments(context.Context) ([]Assignment, error) {
	return s.assignments.List()
}

func (s fakeStore) PutAssignment(ctx context.Context, a Assignment) error {
	if a.TeamIDs == nil {
		a.TeamIDs = []string{}
	}
	return s.assignments.Put(ctx, a)
}

func (s fakeStore) DeleteAssignment(ctx context.Context, id string) error {
	return s.assignments.Delete(ctx, id)
}

func (s fakeStore) GetRules(context.Context) (Rules, bool, error) {
	return s.rules.Get(rulesID)
}

func (s fakeStore) PutRules(ctx context.Context, r Rules) error {
	r.ID = rulesID
	return s.rules.Put(ctx, r)
}
