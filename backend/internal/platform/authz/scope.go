package authz

import "context"

// Scope is what the person acting may see of plan-owned records, in the form a store query filters on
// (Principle VII). It mirrors CanView: an internal call, or View plans in All markets, sees every plan.
type Scope struct {
	Everything bool
	PersonID   string
	ViewsPlans bool
	TeamIDs    []string
}

func ScopeOf(ctx context.Context) Scope {
	g, ok := From(ctx)
	if !ok {
		return Scope{Everything: true, TeamIDs: []string{}}
	}
	all, teamIDs := g.Teams(PlansView)
	if teamIDs == nil {
		teamIDs = []string{}
	}
	return Scope{Everything: all, PersonID: g.PersonID, ViewsPlans: g.Has(PlansView), TeamIDs: teamIDs}
}
