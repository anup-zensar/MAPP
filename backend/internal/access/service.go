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
	"mapp/internal/platform/store"
)

// Person is a person as access needs them, read from feature 003 through the Directory.
type Person struct {
	Ref     authz.PersonRef
	Email   string
	TeamIDs []string // the person's teams, active or not
	Version int      // the person's version: a change of their roles is a change of the person (D-233)
}

// Directory is what access reads and changes of people and teams (feature 003). internal/app implements it
// over the people service (access_adapters.go). Every method runs inside the caller's transaction.
type Directory interface {
	// People returns everyone, active or not, with their teams.
	People(ctx context.Context) ([]Person, error)
	// Teams returns every team, active or not.
	Teams(ctx context.Context) ([]authz.TeamRef, error)
	// CheckPersonVersion refuses, with version_conflict, a change based on an old version of the person.
	CheckPersonVersion(ctx context.Context, personID string, version int) error
	// TouchPerson records that a changed the person now (a new version) and returns the new version.
	TouchPerson(ctx context.Context, a actor.Actor, personID string) (int, error)
}

// ApprovalFacts is what the roles impact check needs to know of approvals (feature 008): in how many active
// approval workflows a person is an approver, and how many activities wait for them. internal/app
// implements it over approvals.
type ApprovalFacts interface {
	ApproverIn(ctx context.Context, personID string) (workflows, waiting int, err error)
}

// Service holds the rules of roles, role assignments and the access rules, and builds every request's
// Grants.
type Service struct {
	tx        store.Transactor
	repo      Store
	ids       *ids.Generator
	history   *history.Service
	directory Directory
	approvals ApprovalFacts           // nil until UseApprovalFacts: nobody is an approver
	targets   map[string]TargetFinder // by path parameter name (UseTargets); none: services check records
	now       clock.Clock
}

// NewService returns the access service. It keeps its records in repo, changes them with tools and reads
// people and teams through directory.
func NewService(tools history.Tools, repo Store, directory Directory) *Service {
	return &Service{
		tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, directory: directory, now: tools.Now,
	}
}

// UseApprovalFacts tells access where approvals are (feature 008), for the warnings of the roles impact
// check (FR-022).
func (s *Service) UseApprovalFacts(facts ApprovalFacts) {
	s.approvals = facts
}

// personRefs returns the known people among ids as references, sorted by name. Never nil.
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

// teamRefs returns the known teams among ids as references, sorted by name. Never nil.
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

// The texts of scopes as screens and the activity log show them.
const (
	allMarketsText  = "All markets"
	myTeamsText     = "My teams"
	chosenTeamsText = "Chosen teams" // a Chosen teams scope whose teams were all deleted
	noRoleText      = "No role"
)

// scopeTeams returns the teams the assignment covers now: the person's teams for My teams, the chosen teams
// that still exist for Chosen teams, none for All markets.
func (w world) scopeTeams(personID string, a Assignment) []authz.TeamRef {
	switch a.ScopeType {
	case authz.ScopeMyTeams:
		return w.teamRefs(w.people[personID].TeamIDs)
	case authz.ScopeTeams:
		return w.teamRefs(a.TeamIDs)
	}
	return []authz.TeamRef{}
}

// scopeText returns the assignment's scope as screens show it: "All markets", "My teams" or the chosen teams'
// names, sorted.
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

// assignmentViews returns the person's assignments as the API returns them, sorted by role name.
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

// rolesText returns a person's roles as the activity log shows them (data model §3): "Planner · My teams,
// Approver · Mobility China", sorted by role name; "No role" when there is none.
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

// personRoles returns the person p with their roles, their teams and what the roles allow together.
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

// compareNames orders names the way people sort them: ignoring letter case.
func compareNames(a, b string) int {
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

// compareRoles orders roles by name ignoring letter case, then by id.
func compareRoles(a, b Role) int {
	return cmp.Or(compareNames(a.Name, b.Name), strings.Compare(a.ID, b.ID))
}

// sortedRoles returns the roles sorted by name.
func sortedRoles(roles map[string]Role) []Role {
	list := make([]Role, 0, len(roles))
	for _, r := range roles {
		list = append(list, r)
	}
	slices.SortFunc(list, compareRoles)
	return list
}
