package apitest

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"mapp/internal/access"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/stores"
)

// GrantRole stores the role assignment directly, with no request and no log entry, so a test that is not
// about access gives its persona the role its steps need: GrantRole(t, email, "planner", "teams", "Mobility China").
func (s *Server) GrantRole(t testing.TB, email, builtIn, scope string, teamNames ...string) {
	t.Helper()
	err := s.set.Tx.Write(t.Context(), func(ctx context.Context) error {
		peopleRepo, accessRepo := s.set.People, s.set.Access
		everyone, err := peopleRepo.List(ctx)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(everyone, func(p people.Person) bool { return strings.EqualFold(p.Email, email) })
		if i < 0 {
			return fmt.Errorf("there is no person with e-mail %s", email)
		}
		personID := everyone[i].ID
		roles, err := accessRepo.ListRoles(ctx)
		if err != nil {
			return err
		}
		j := slices.IndexFunc(roles, func(r access.Role) bool { return r.BuiltIn == builtIn })
		if j < 0 {
			return fmt.Errorf("there is no built-in role %q", builtIn)
		}
		teamIDs, err := teamIDsOf(ctx, peopleRepo, teamNames)
		if err != nil {
			return err
		}
		mine, err := accessRepo.AssignmentsOfPerson(ctx, personID)
		if err != nil {
			return err
		}
		k := slices.IndexFunc(mine, func(a access.Assignment) bool { return a.RoleID == roles[j].ID })
		var assignment access.Assignment
		if k >= 0 {
			assignment = mine[k]
			assignment.Touch(actor.System, s.now)
		} else {
			id, err := ids.NewGenerator(s.set.Counters).Next(ctx, ids.PrefixRoleAssignment)
			if err != nil {
				return err
			}
			assignment = access.Assignment{ID: id, PersonID: personID, RoleID: roles[j].ID,
				Meta: record.NewMeta(actor.System, s.now)}
		}
		assignment.ScopeType, assignment.TeamIDs = authz.ScopeType(scope), teamIDs
		return accessRepo.PutAssignment(ctx, assignment)
	})
	if err != nil {
		t.Fatalf("apitest: giving %s the role %s (%s %v): %v", email, builtIn, scope, teamNames, err)
	}
}

func teamIDsOf(ctx context.Context, repo stores.PeopleStore, names []string) ([]string, error) {
	teams, err := repo.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	teamIDs := []string{}
	for _, name := range names {
		i := slices.IndexFunc(teams, func(team people.Team) bool { return team.Name == name })
		if i < 0 {
			return nil, fmt.Errorf("there is no team called '%s'", name)
		}
		teamIDs = append(teamIDs, teams[i].ID)
	}
	return teamIDs, nil
}
