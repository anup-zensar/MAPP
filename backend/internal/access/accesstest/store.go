// Package accesstest is the in-memory fake of the roles, role assignments and access rules store, for tests
// that build services or the whole API without a database.
package accesstest

import (
	"context"

	"mapp/internal/access"
	"mapp/internal/platform/fake"
)

const rulesID = "access-rules"

type Store struct {
	roles       *fake.Table[access.Role]
	assignments *fake.Table[access.Assignment]
	rules       *fake.Table[access.Rules]
}

func NewStore(db *fake.DB) Store {
	return Store{
		roles:       fake.TableOf(db, "roles", func(r access.Role) string { return r.ID }),
		assignments: fake.TableOf(db, "role_assignments", func(a access.Assignment) string { return a.ID }),
		rules:       fake.TableOf(db, "access_rules", func(r access.Rules) string { return r.ID }),
	}
}

func (s Store) GetRole(_ context.Context, id string) (access.Role, bool, error) {
	return s.roles.Get(id)
}

func (s Store) ListRoles(context.Context) ([]access.Role, error) {
	return s.roles.List()
}

func (s Store) PutRole(ctx context.Context, r access.Role) error {
	return s.roles.Put(ctx, r)
}

func (s Store) DeleteRole(ctx context.Context, id string) error {
	return s.roles.Delete(ctx, id)
}

func (s Store) ListAssignments(context.Context) ([]access.Assignment, error) {
	return s.assignments.List()
}

func (s Store) AssignmentsOfPerson(_ context.Context, personID string) ([]access.Assignment, error) {
	return s.assignments.Where(func(a access.Assignment) bool { return a.PersonID == personID })
}

// PutAssignment keeps no teams as [], as PostgreSQL reads back an assignment without teams.
func (s Store) PutAssignment(ctx context.Context, a access.Assignment) error {
	if a.TeamIDs == nil {
		a.TeamIDs = []string{}
	}
	return s.assignments.Put(ctx, a)
}

func (s Store) DeleteAssignment(ctx context.Context, id string) error {
	return s.assignments.Delete(ctx, id)
}

func (s Store) GetRules(context.Context) (access.Rules, bool, error) {
	return s.rules.Get(rulesID)
}

func (s Store) PutRules(ctx context.Context, r access.Rules) error {
	r.ID = rulesID
	return s.rules.Put(ctx, r)
}
