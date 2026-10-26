package access_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/access"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/record"
)

var pgMeta = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))

// pgStore gives a store whose database already holds the people U-1001 and U-1002 and the teams T-1001 and
// T-1002, which assignments refer to.
func pgStore(t *testing.T) (*database.DB, *access.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	directory := people.NewPostgresStore(db)
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(
			directory.Put(ctx, people.Person{ID: "U-1001", Name: "Priya Raman", Email: "priya@example.com", Meta: pgMeta}),
			directory.Put(ctx, people.Person{ID: "U-1002", Name: "Daniel Okoro", Email: "daniel@example.com", Meta: pgMeta}),
			directory.PutTeam(ctx, people.Team{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue", Meta: pgMeta}),
			directory.PutTeam(ctx, people.Team{ID: "T-1002", Name: "Mobility China", Colour: "teal", Meta: pgMeta}))
	})
	return db, access.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgRole(id, name, builtIn string, permissions ...authz.Permission) access.Role {
	return access.Role{ID: id, Name: name, BuiltIn: builtIn, Permissions: permissions,
		DefaultScope: authz.ScopeMyTeams, Meta: pgMeta}
}

func pgAssignment(id, personID, roleID string, teamIDs ...string) access.Assignment {
	scope := authz.ScopeMyTeams
	if len(teamIDs) > 0 {
		scope = authz.ScopeTeams
	}
	return access.Assignment{ID: id, PersonID: personID, RoleID: roleID, ScopeType: scope,
		TeamIDs: append([]string{}, teamIDs...), Meta: pgMeta}
}

func TestPostgresStoreRolesAssignmentsAndRulesFR008(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	roles := []access.Role{
		pgRole("RL-1001", "Planner", authz.BuiltInPlanner, authz.PlansView, authz.PlansEdit),
		pgRole("RL-1002", "Viewer", "", authz.PlansView),
	}
	assignments := []access.Assignment{
		pgAssignment("RA-1001", "U-1001", "RL-1001", "T-1002", "T-1001"),
		pgAssignment("RA-1002", "U-1002", "RL-1001"),
		pgAssignment("RA-1003", "U-1001", "RL-1002"),
	}
	rules := access.Rules{NoSelfApproval: true, NewPersonRoleID: "RL-1002", Meta: pgMeta}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutRole(ctx, roles[1]), s.PutRole(ctx, roles[0]), s.PutAssignment(ctx, assignments[0]),
			s.PutAssignment(ctx, assignments[1]), s.PutAssignment(ctx, assignments[2]), s.PutRules(ctx, rules))
	})
	pgWrite(t, db, func(ctx context.Context) error { return s.DeleteAssignment(ctx, "RA-1003") })

	gotRoles, rolesErr := s.ListRoles(t.Context())
	ofPriya, assignmentsErr := s.AssignmentsOfPerson(t.Context(), "U-1001")
	gotRules, found, rulesErr := s.GetRules(t.Context())

	if err := errors.Join(rolesErr, assignmentsErr, rulesErr); err != nil || !found {
		t.Fatalf("ListRoles(), AssignmentsOfPerson(), GetRules() = found %v, error %v", found, err)
	}
	if diff := cmp.Diff(roles, gotRoles); diff != "" {
		t.Errorf("ListRoles() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(assignments[:1], ofPriya); diff != "" {
		t.Errorf("AssignmentsOfPerson() after the delete mismatch (-want +got):\n%s", diff)
	}
	rules.ID = "access-rules" // the one record of the rules has this id, whatever was passed
	if diff := cmp.Diff(rules, gotRules); diff != "" {
		t.Errorf("GetRules() mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreNoAccessRulesUntilSavedFR008(t *testing.T) {
	t.Parallel()
	_, s := pgStore(t)

	_, found, err := s.GetRules(t.Context())

	if err != nil || found {
		t.Errorf("GetRules() = found %v, error %v; want not found, so the defaults apply", found, err)
	}
}

func TestPostgresStoreRefusesBrokenAccessDataFR004(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		write func(ctx context.Context, s *access.PostgresStore) error
	}{
		{name: "a permission that is not in Table A", write: func(ctx context.Context, s *access.PostgresStore) error {
			return s.PutRole(ctx, pgRole("RL-1002", "Approver", "", "plans.approve"))
		}},
		{name: "a role without permissions", write: func(ctx context.Context, s *access.PostgresStore) error {
			return s.PutRole(ctx, pgRole("RL-1002", "Nobody", ""))
		}},
		{name: "a second built-in Planner role", write: func(ctx context.Context, s *access.PostgresStore) error {
			return s.PutRole(ctx, pgRole("RL-1002", "Planner 2", authz.BuiltInPlanner, authz.PlansView))
		}},
		{name: "the same role twice for one person", write: func(ctx context.Context, s *access.PostgresStore) error {
			return s.PutAssignment(ctx, pgAssignment("RA-1002", "U-1001", "RL-1001"))
		}},
		{name: "a role deleted while someone holds it", write: func(ctx context.Context, s *access.PostgresStore) error {
			return s.DeleteRole(ctx, "RL-1001")
		}},
		{name: "chosen teams that do not exist", write: func(ctx context.Context, s *access.PostgresStore) error {
			return s.PutAssignment(ctx, pgAssignment("RA-1002", "U-1002", "RL-1001", "T-9999"))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s := pgStore(t)
			pgWrite(t, db, func(ctx context.Context) error {
				return errors.Join(s.PutRole(ctx, pgRole("RL-1001", "Planner", authz.BuiltInPlanner, authz.PlansView)),
					s.PutAssignment(ctx, pgAssignment("RA-1001", "U-1001", "RL-1001")))
			})

			err := db.Write(t.Context(), func(ctx context.Context) error { return tt.write(ctx, s) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}
