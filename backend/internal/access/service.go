package access

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/ids"
)

type Person struct {
	Ref     authz.PersonRef
	Email   string
	TeamIDs []string
	Version int // a change of the person's roles is a change of the person
}

// Directory is the people and teams of feature 003, active or not; every method joins the caller's transaction.
type Directory interface {
	People(ctx context.Context) ([]Person, error)
	Teams(ctx context.Context) ([]authz.TeamRef, error)
	// CheckPersonVersion refuses, with version_conflict, a change based on an old version of the person.
	CheckPersonVersion(ctx context.Context, personID string, version int) error
	// TouchPerson gives the person a new version, changed by a now, and returns it.
	TouchPerson(ctx context.Context, a actor.Actor, personID string) (int, error)
}

// ApprovalFacts counts the active approval workflows a person approves in and the activities waiting for them.
type ApprovalFacts interface {
	ApproverIn(ctx context.Context, personID string) (workflows, waiting int, err error)
}

type Service struct {
	tx        Transactor
	repo      Store
	ids       *ids.Generator
	history   *history.Service
	directory Directory
	approvals ApprovalFacts           // nil until UseApprovalFacts: nobody is an approver
	targets   map[string]TargetFinder // by path parameter name; none: the services check records
	now       clock.Clock
}

func NewService(tools history.Tools, repo Store, directory Directory) *Service {
	return &Service{
		tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, directory: directory, now: tools.Now,
	}
}

func (s *Service) UseApprovalFacts(facts ApprovalFacts) {
	s.approvals = facts
}

func (w world) personRefs(ids []string) []authz.PersonRef {
	refs := []authz.PersonRef{}
	for _, id := range ids {
		if p, known := w.people[id]; known {
			refs = append(refs, p.Ref)
		}
	}
	slices.SortFunc(refs, func(x, y authz.PersonRef) int {
		return cmp.Or(compareNames(x.Name, y.Name), strings.Compare(x.ID, y.ID))
	})
	return refs
}

func (w world) teamRefs(ids []string) []authz.TeamRef {
	refs := []authz.TeamRef{}
	for _, id := range ids {
		if t, known := w.teams[id]; known {
			refs = append(refs, t)
		}
	}
	slices.SortFunc(refs, func(x, y authz.TeamRef) int {
		return cmp.Or(compareNames(x.Name, y.Name), strings.Compare(x.ID, y.ID))
	})
	return refs
}

const (
	allMarketsText  = "All markets"
	myTeamsText     = "My teams"
	chosenTeamsText = "Chosen teams" // a Chosen teams scope whose teams were all deleted
	noRoleText      = "No role"
)

func (w world) scopeTeams(personID string, a Assignment) []authz.TeamRef {
	switch a.ScopeType {
	case authz.ScopeMyTeams:
		return w.teamRefs(w.people[personID].TeamIDs)
	case authz.ScopeTeams:
		return w.teamRefs(a.TeamIDs)
	}
	return []authz.TeamRef{}
}

func (w world) scopeText(personID string, a Assignment) string {
	switch a.ScopeType {
	case authz.ScopeAll:
		return allMarketsText
	case authz.ScopeMyTeams:
		return myTeamsText
	}
	names := []string{}
	for _, t := range w.scopeTeams(personID, a) {
		names = append(names, t.Name)
	}
	if len(names) == 0 {
		return chosenTeamsText
	}
	return strings.Join(names, ", ")
}

func (w world) assignmentViews(personID string, assignments []Assignment) []authz.RoleAssignment {
	views := make([]authz.RoleAssignment, 0, len(assignments))
	for _, a := range assignments {
		teams := w.scopeTeams(personID, a)
		views = append(views, authz.RoleAssignment{
			ID:         a.ID,
			Role:       w.roles[a.RoleID].ref(),
			Scope:      authz.AccessScope{Type: a.ScopeType, Teams: teams, Text: w.scopeText(personID, a)},
			ScopeEmpty: a.ScopeType != authz.ScopeAll && len(teams) == 0,
		})
	}
	return views
}

func (w world) rolesText(personID string, assignments []Assignment) string {
	if len(assignments) == 0 {
		return noRoleText
	}
	sorted := slices.Clone(assignments)
	slices.SortFunc(sorted, func(x, y Assignment) int { return compareRoles(w.roles[x.RoleID], w.roles[y.RoleID]) })
	parts := make([]string, 0, len(sorted))
	for _, a := range sorted {
		parts = append(parts, w.roles[a.RoleID].Name+" · "+w.scopeText(personID, a))
	}
	return strings.Join(parts, ", ")
}

func (w world) personRoles(p Person) PersonRoles {
	mine := w.assignmentsOf(p.Ref.ID)
	return PersonRoles{
		Person:      p.Ref,
		Teams:       w.teamRefs(p.TeamIDs),
		Roles:       w.assignmentViews(p.Ref.ID, mine),
		Permissions: w.grantsOf(p.Ref.ID, mine).Permissions(w.teamRefs),
		Version:     p.Version,
	}
}

func compareNames(a, b string) int {
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

func compareRoles(a, b Role) int {
	return cmp.Or(compareNames(a.Name, b.Name), strings.Compare(a.ID, b.ID))
}

func sortedRoles(roles map[string]Role) []Role {
	list := make([]Role, 0, len(roles))
	for _, r := range roles {
		list = append(list, r)
	}
	slices.SortFunc(list, compareRoles)
	return list
}
