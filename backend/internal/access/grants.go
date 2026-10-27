package access

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

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

// Summary returns the person's access for GET /api/v1/me (FR-024): their roles with scopes, what they allow
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
