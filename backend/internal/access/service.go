package access

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/store"
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
	repo      Repository
	ids       *ids.Generator
	history   *history.Service
	directory Directory
	approvals ApprovalFacts // nil until UseApprovalFacts: nobody is an approver
	now       clock.Clock
}

// NewService returns the access service. It keeps its records in repo, gives them readable IDs with
// generator, logs their changes through hist, reads people and teams through directory, works in
// transactions of tx and reads the time from now.
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator, hist *history.Service,
	directory Directory, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, ids: generator, history: hist, directory: directory, now: now}
}

// UseApprovalFacts tells access where approvals are (feature 008), for the warnings of the roles impact
// check (FR-022).
func (s *Service) UseApprovalFacts(facts ApprovalFacts) {
	s.approvals = facts
}

// world is everything a decision about access reads, read once inside one transaction.
type world struct {
	roles       map[string]Role
	assignments []Assignment // ordered by id
	people      map[string]Person
	teams       map[string]authz.TeamRef
	rules       Rules
}

// read reads the world inside the caller's transaction (or a new Read).
func (s *Service) read(ctx context.Context) (world, error) {
	w := world{roles: map[string]Role{}, people: map[string]Person{}, teams: map[string]authz.TeamRef{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		roles, err := s.repo.ListRoles(ctx)
		if err != nil {
			return err
		}
		for _, r := range roles {
			w.roles[r.ID] = r
		}
		if w.assignments, err = s.repo.ListAssignments(ctx); err != nil {
			return err
		}
		everyone, err := s.directory.People(ctx)
		if err != nil {
			return err
		}
		for _, p := range everyone {
			w.people[p.Ref.ID] = p
		}
		teams, err := s.directory.Teams(ctx)
		if err != nil {
			return err
		}
		for _, t := range teams {
			w.teams[t.ID] = t
		}
		w.rules, err = s.storedRules(ctx, w.roles)
		return err
	})
	return w, err
}

// storedRules returns the stored access rules, or the defaults while none are stored: segregation of
// duties on, and the Viewer role for new people (data model §1.3).
func (s *Service) storedRules(ctx context.Context, roles map[string]Role) (Rules, error) {
	rules, found, err := s.repo.GetRules(ctx)
	if err != nil || found {
		return rules, err
	}
	rules = Rules{ID: rulesID, NoSelfApproval: true}
	for _, r := range sortedRoles(roles) {
		if r.BuiltIn == authz.BuiltInViewer {
			rules.NewPersonRoleID = r.ID
		}
	}
	return rules, nil
}

// GrantsFor returns what the person personID may do now (data model §2.1): built from their assignments,
// roles and teams and the access rules as they are stored at this moment, so a change applies to their next
// request without signing in again (FR-020). The access middleware calls it once per request.
func (s *Service) GrantsFor(ctx context.Context, personID string) (authz.Grants, error) {
	w, err := s.read(ctx)
	if err != nil {
		return authz.Grants{}, err
	}
	return w.grantsOf(personID, w.assignmentsOf(personID)), nil
}

// Summary returns the person's access for GET /api/me (FR-024): their roles with scopes, what they allow
// together, the administrators to ask and the segregation-of-duties rule.
func (s *Service) Summary(ctx context.Context, personID string) (authz.MyAccess, error) {
	w, err := s.read(ctx)
	if err != nil {
		return authz.MyAccess{}, err
	}
	mine := w.assignmentsOf(personID)
	g := w.grantsOf(personID, mine)
	return authz.MyAccess{
		Roles:          w.assignmentViews(personID, mine),
		Permissions:    g.Permissions(w.teamRefs),
		Administrators: w.administratorRefs(),
		NoSelfApproval: w.rules.NoSelfApproval,
	}, nil
}

// assignmentsOf returns the assignments of the person whose role still exists, sorted by role name.
func (w world) assignmentsOf(personID string) []Assignment {
	mine := []Assignment{}
	for _, a := range w.assignments {
		if _, known := w.roles[a.RoleID]; known && a.PersonID == personID {
			mine = append(mine, a)
		}
	}
	slices.SortFunc(mine, func(x, y Assignment) int { return compareRoles(w.roles[x.RoleID], w.roles[y.RoleID]) })
	return mine
}

// grantsOf merges the assignments of the person into their Grants.
func (w world) grantsOf(personID string, assignments []Assignment) authz.Grants {
	granted := make([]authz.GrantedRole, 0, len(assignments))
	permsByRole := map[string][]authz.Permission{}
	for _, a := range assignments {
		r := w.roles[a.RoleID]
		granted = append(granted, authz.GrantedRole{
			ID: r.ID, Name: r.Name, BuiltIn: r.BuiltIn, Scope: a.ScopeType, TeamIDs: a.TeamIDs,
			ScopeText: w.scopeText(personID, a),
		})
		permsByRole[r.ID] = r.permissions()
	}
	g := authz.NewGrants(personID, granted, w.people[personID].TeamIDs, permsByRole, w.administrators(),
		authz.Rules{NoSelfApproval: w.rules.NoSelfApproval})
	for id, t := range w.teams {
		g.TeamNames[id] = t.Name
	}
	return g
}

// administratorIDs returns the ids of the active people who hold the Administrator role.
func (w world) administratorIDs() []string {
	var adminIDs []string
	for _, a := range w.assignments {
		if r, known := w.roles[a.RoleID]; known && r.administrator() && w.people[a.PersonID].Ref.Active &&
			!slices.Contains(adminIDs, a.PersonID) {
			adminIDs = append(adminIDs, a.PersonID)
		}
	}
	return adminIDs
}

// administrators returns the active holders of the Administrator role, sorted by name: whom to ask.
func (w world) administrators() []actor.Actor {
	admins := []actor.Actor{}
	for _, ref := range w.administratorRefs() {
		admins = append(admins, actor.Actor{ID: ref.ID, Name: ref.Name})
	}
	return admins
}

// administratorRefs returns the active holders of the Administrator role as references, sorted by name.
func (w world) administratorRefs() []authz.PersonRef {
	return w.personRefs(w.administratorIDs())
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
