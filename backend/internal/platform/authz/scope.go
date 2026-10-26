package authz

import (
	"context"
	"slices"
)

// Scope is where the person acting holds a permission; an internal call, or All markets, covers Everything.
type Scope struct {
	Everything bool
	PersonID   string
	Holds      bool
	TeamIDs    []string
}

func ScopeOf(ctx context.Context) Scope {
	return PermissionScopeOf(ctx, PlansView)
}

func PermissionScopeOf(ctx context.Context, p Permission) Scope {
	g, ok := From(ctx)
	if !ok {
		return Scope{Everything: true, TeamIDs: []string{}}
	}
	return g.scopeOf(p)
}

// Covers, Sees and SeesActivity are the migrations' plan_in_permission_scope(), plan_in_scope() and
// activity_in_scope() in Go; Sees and SeesActivity take a scope of PlansView.
func (s Scope) Covers(plan PlanFacts) bool {
	if s.Everything {
		return true
	}
	if !s.Holds {
		return false
	}
	if slices.ContainsFunc(plan.TeamIDs, func(id string) bool { return slices.Contains(s.TeamIDs, id) }) {
		return true
	}
	created := plan.CreatedByID != "" && plan.CreatedByID == s.PersonID
	return created && len(plan.TeamIDs) == 0
}

func (s Scope) Sees(plan PlanFacts) bool {
	return s.Covers(plan) || (plan.OwnerID != "" && plan.OwnerID == s.PersonID)
}

func (s Scope) SeesActivity(a ActivityFacts) bool {
	owns := a.OwnerID != "" && a.OwnerID == s.PersonID
	return owns || s.Sees(a.Plan) || slices.Contains(a.ApproverIDs, s.PersonID)
}

func (g Grants) scopeOf(p Permission) Scope {
	all, teamIDs := g.Teams(p)
	if teamIDs == nil {
		teamIDs = []string{}
	}
	return Scope{Everything: all, PersonID: g.PersonID, Holds: g.Has(p), TeamIDs: teamIDs}
}
