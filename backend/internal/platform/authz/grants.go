package authz

import (
	"slices"

	"northstar/internal/platform/actor"
)

// ScopeType says where a role applies (FR-004): All markets, the teams the person belongs to at the moment
// of the action, or a list of chosen teams. A market is a team (D-225).
type ScopeType string

// The scope types of a role assignment (contract AccessScopeType).
const (
	ScopeAll     ScopeType = "all"
	ScopeMyTeams ScopeType = "myTeams"
	ScopeTeams   ScopeType = "teams"
)

// The built-in kinds of the five seeded roles (contract RoleBuiltIn); "" for roles made in Settings.
const (
	BuiltInAdministrator = "administrator"
	BuiltInGlobalPlanner = "globalPlanner"
	BuiltInPlanner       = "planner"
	BuiltInApprover      = "approver"
	BuiltInViewer        = "viewer"
)

// GrantedRole is one role a person holds, with its scope: what the refusal messages and the View only note
// name ("your role (Viewer)").
type GrantedRole struct {
	ID        string
	Name      string
	BuiltIn   string    // BuiltInAdministrator … or ""
	Scope     ScopeType // where the role applies
	TeamIDs   []string  // the chosen teams; only with ScopeTeams
	ScopeText string    // as screens show it: "All markets", "My teams" or the chosen teams' names
}

// Rules are the access rules that change decisions (Settings › Roles and access › Rules).
type Rules struct {
	// NoSelfApproval: nobody approves or declines an activity they own or submitted (FR-026, D-302).
	NoSelfApproval bool
}

// PlanFacts is what a decision needs to know about a plan (data model §2.2), from a stored plan or from a
// plan in the recycle bin (plans.AccessFacts).
type PlanFacts struct {
	ID          string
	TeamIDs     []string
	OwnerID     string
	CreatedByID string
}

// ActivityFacts is what deciding whether a person sees an activity needs (data model §2.2): its plan, its
// owner and the people named to approve it in any approval round.
type ActivityFacts struct {
	Plan        PlanFacts
	OwnerID     string
	ApproverIDs []string
}

// reach is where a person holds one permission: everywhere (All markets) or in some teams.
type reach struct {
	all   bool
	teams []string // sorted team ids; empty when all
}

// Grants is what one signed-in person may do, built once per request by the access middleware from their
// role assignments, their roles, their teams and the access rules (data model §2.1). Its zero value grants
// nothing.
type Grants struct {
	PersonID       string
	Roles          []GrantedRole     // sorted by role name
	Administrators []actor.Actor     // the active people who hold the Administrator role, by name: "ask …"
	Rules          Rules             // the access rules in force
	TeamNames      map[string]string // team id → name, for the messages that name teams

	perms      map[Permission]reach // the union of every role's permissions
	everything bool                 // holds the Administrator role: every permission, also ones added later
}

// NewGrants merges the roles a person holds (D-223, FR-002, FR-004): they may do what any role allows. A
// role's permissions come from permsByRole (by role ID). A role with scope My teams contributes myTeamIDs,
// the person's teams now, active or not; Chosen teams contributes its TeamIDs; All markets makes a
// permission apply everywhere. Permissions that are not scoped apply everywhere whatever the scope. The
// Administrator role gives every permission everywhere.
func NewGrants(personID string, roles []GrantedRole, myTeamIDs []string, permsByRole map[string][]Permission,
	admins []actor.Actor, rules Rules) Grants {
	g := Grants{PersonID: personID, Roles: slices.Clone(roles), Administrators: slices.Clone(admins), Rules: rules,
		TeamNames: map[string]string{}, perms: map[Permission]reach{}}
	for _, role := range roles {
		if role.BuiltIn == BuiltInAdministrator {
			g.everything = true
			continue
		}
		var teams []string
		switch role.Scope {
		case ScopeMyTeams:
			teams = myTeamIDs
		case ScopeTeams:
			teams = role.TeamIDs
		}
		for _, p := range permsByRole[role.ID] {
			r := g.perms[p]
			if role.Scope == ScopeAll || !info(p).Scoped {
				r.all = true
			}
			for _, id := range teams {
				if !slices.Contains(r.teams, id) {
					r.teams = append(r.teams, id)
				}
			}
			g.perms[p] = r
		}
	}
	for p, r := range g.perms {
		if r.all {
			r.teams = nil
		}
		slices.Sort(r.teams)
		g.perms[p] = r
	}
	return g
}

// Has reports whether one of the person's roles includes p, in any scope (the middleware's pre-check).
func (g Grants) Has(p Permission) bool {
	if g.everything {
		return true
	}
	_, held := g.perms[p]
	return held
}

// IsAdministrator reports whether the person holds the Administrator role.
func (g Grants) IsAdministrator() bool {
	return g.everything
}

// Teams says where the person holds p: everywhere (all true), or in the teams teamIDs, sorted by id. A
// permission the person does not hold gives false and no teams.
func (g Grants) Teams(p Permission) (all bool, teamIDs []string) {
	if g.everything {
		return true, nil
	}
	r, held := g.perms[p]
	if !held {
		return false, nil
	}
	return r.all, slices.Clone(r.teams)
}

// InScope reports whether the person holds p for the plan (FR-005): p applies to All markets, or one of the
// plan's teams is in its scope, or the plan has no team yet and the person created it (and holds p in some
// scope).
func (g Grants) InScope(p Permission, plan PlanFacts) bool {
	if !g.Has(p) {
		return false
	}
	all, teams := g.Teams(p)
	if all {
		return true
	}
	if slices.ContainsFunc(plan.TeamIDs, func(id string) bool { return slices.Contains(teams, id) }) {
		return true
	}
	return len(plan.TeamIDs) == 0 && plan.CreatedByID != "" && plan.CreatedByID == g.PersonID
}

// CanViewPlan reports whether the person sees the plan and everything in it (FR-006): "View plans" in its
// scope, or they own the plan.
func (g Grants) CanViewPlan(plan PlanFacts) bool {
	return g.InScope(PlansView, plan) || (plan.OwnerID != "" && plan.OwnerID == g.PersonID)
}

// CanViewActivity reports whether the person sees the activity (FR-006, data model §2.2): they see its plan,
// own it, or are named to approve it in any approval round.
func (g Grants) CanViewActivity(a ActivityFacts) bool {
	return g.CanViewPlan(a.Plan) || (a.OwnerID != "" && a.OwnerID == g.PersonID) || slices.Contains(a.ApproverIDs, g.PersonID)
}

// RoleNames returns the names of the person's roles, in their order (sorted by name).
func (g Grants) RoleNames() []string {
	names := make([]string, 0, len(g.Roles))
	for _, role := range g.Roles {
		names = append(names, role.Name)
	}
	return names
}

// teamNames returns the names of the teams teamIDs, sorted; a team without a known name is left out.
func (g Grants) teamNames(teamIDs []string) []string {
	names := []string{}
	for _, id := range teamIDs {
		if name, known := g.TeamNames[id]; known {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}
