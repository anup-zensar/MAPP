package access

import (
	"fmt"
	"slices"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// checkAssignments returns the assignments a person would hold after a change to inputs, and adds every
// problem (data model §1.2): a known role, listed once; a scope type; the Administrator role for All markets
// only (message 12); Chosen teams with at least one known team (message 16). Field problems are keyed
// roles.<i>.roleId, roles.<i>.scope and roles.<i>.teamIds. An assignment of a role the person already holds
// keeps its id and bookkeeping.
func (w world) checkAssignments(problems *errs.Problems, personID string, inputs []AssignmentInput) []Assignment {
	current := map[string]Assignment{}
	for _, a := range w.assignmentsOf(personID) {
		current[a.RoleID] = a
	}
	var wanted []Assignment
	var seen []string
	for i, in := range inputs {
		field := func(name string) string { return fmt.Sprintf("roles.%d.%s", i, name) }
		r, known := w.roles[in.RoleID]
		if !known {
			problems.Add(field("roleId"), msgChooseRole)
			continue
		}
		if slices.Contains(seen, r.ID) {
			problems.Add(field("roleId"), fmt.Sprintf(msgRoleTwice, r.Name))
			continue
		}
		seen = append(seen, r.ID)
		a := current[r.ID]
		a.PersonID, a.RoleID, a.ScopeType, a.TeamIDs = personID, r.ID, in.Scope.Type, []string{}
		switch in.Scope.Type {
		case authz.ScopeAll, authz.ScopeMyTeams:
		case authz.ScopeTeams:
			a.TeamIDs = w.checkTeams(problems, field("teamIds"), in.Scope.TeamIDs)
		default:
			problems.Add(field("scope"), msgChooseScope)
		}
		if r.administrator() && in.Scope.Type != authz.ScopeAll {
			problems.Add(field("scope"), msgAdministratorAllMarkets)
		}
		wanted = append(wanted, a)
	}
	slices.SortFunc(wanted, func(x, y Assignment) int { return compareRoles(w.roles[x.RoleID], w.roles[y.RoleID]) })
	return wanted
}

// checkTeams returns the chosen teams without repeats, sorted by name, and adds the problem of an empty or
// unknown choice.
func (w world) checkTeams(problems *errs.Problems, field string, teamIDs []string) []string {
	var unique []string
	for _, id := range teamIDs {
		if _, known := w.teams[id]; !known {
			problems.Add(field, msgChooseTeamsFromList)
			return []string{}
		}
		if !slices.Contains(unique, id) {
			unique = append(unique, id)
		}
	}
	if len(unique) == 0 {
		problems.Add(field, msgChooseTeam)
		return []string{}
	}
	sorted := []string{}
	for _, t := range w.teamRefs(unique) {
		sorted = append(sorted, t.ID)
	}
	return sorted
}

// sameAssignments reports whether two lists give the same roles with the same scopes.
func sameAssignments(before, after []Assignment) bool {
	if len(before) != len(after) {
		return false
	}
	for _, b := range before {
		i := slices.IndexFunc(after, func(a Assignment) bool { return a.RoleID == b.RoleID })
		if i < 0 || after[i].ScopeType != b.ScopeType || !slices.Equal(after[i].TeamIDs, b.TeamIDs) {
			return false
		}
	}
	return true
}

// checkAdministratorLeft refuses a change of the person's assignments to wanted that would leave no active
// person with the Administrator role (FR-021, message 10). active says whether the person is active.
func (w world) checkAdministratorLeft(personID string, wanted []Assignment, active bool) error {
	admins := w.administratorIDs()
	if !slices.Contains(admins, personID) || len(admins) > 1 {
		return nil
	}
	keeps := active && slices.ContainsFunc(wanted, func(a Assignment) bool { return w.roles[a.RoleID].administrator() })
	if keeps {
		return nil
	}
	return errs.NotAllowed(fmtOnlyAdministrator(w.people[personID].Ref.Name))
}
