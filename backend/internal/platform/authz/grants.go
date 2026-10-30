package authz

import (
	"slices"

	"mapp/internal/platform/actor"
)

// ScopeType says where a role applies; My teams are the person's teams at the moment of the action (D-225).
type ScopeType string

const (
	ScopeAll     ScopeType = "all"
	ScopeMyTeams ScopeType = "myTeams"
	ScopeTeams   ScopeType = "teams"
)

const (
	BuiltInAdministrator = "administrator"
	BuiltInGlobalPlanner = "globalPlanner"
	BuiltInPlanner       = "planner"
	BuiltInApprover      = "approver"
	BuiltInViewer        = "viewer"
)

type GrantedRole struct {
	ID        string
	Name      string
	BuiltIn   string
	Scope     ScopeType
	TeamIDs   []string
	ScopeText string
}

type Rules struct {
	NoSelfApproval bool
}

type PlanFacts struct {
	ID          string
	TeamIDs     []string
	OwnerID     string
	CreatedByID string
}

type ActivityFacts struct {
	Plan        PlanFacts
	OwnerID     string
	ApproverIDs []string
}

type reach struct {
	all   bool
	teams []string // sorted team ids; empty when all
}

// Grants is what one signed-in person may do, built once per request; its zero value grants nothing.
type Grants struct {
	PersonID       string
	Roles          []GrantedRole
	Administrators []actor.Actor
	Rules          Rules
	TeamNames      map[string]string

	perms      map[Permission]reach
	everything bool // holds the Administrator role: every permission, also ones added later
}

// NewGrants merges the roles a person holds: My teams contributes myTeamIDs, the person's teams now, active or
// not; unscoped permissions apply everywhere; finance.link without finance.import is not held.
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
	if _, imports := g.perms[FinanceImport]; !imports {
		delete(g.perms, FinanceLink) // linking is only for people who see not-linked transactions
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

func (g Grants) Has(p Permission) bool {
	if g.everything {
		return true
	}
	_, held := g.perms[p]
	return held
}

func (g Grants) IsAdministrator() bool {
	return g.everything
}

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

// InScope also holds when the plan has no team yet and the person created it.
func (g Grants) InScope(p Permission, plan PlanFacts) bool {
	return g.scopeOf(p).Covers(plan)
}

func (g Grants) CanViewPlan(plan PlanFacts) bool {
	return g.scopeOf(PlansView).Sees(plan)
}

func (g Grants) CanViewActivity(a ActivityFacts) bool {
	return g.scopeOf(PlansView).SeesActivity(a)
}

func (g Grants) RoleNames() []string {
	names := make([]string, 0, len(g.Roles))
	for _, role := range g.Roles {
		names = append(names, role.Name)
	}
	return names
}

func (g Grants) ScopeTeamNames(p Permission) []string {
	_, teams := g.Teams(p)
	return g.teamNames(teams)
}

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
