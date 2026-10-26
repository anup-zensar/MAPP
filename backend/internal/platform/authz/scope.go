package authz

import "context"

// Scope is where the person acting holds a permission, as store queries filter on it (Principle VII): see
// InScope; an internal call, or the permission in All markets, covers Everything.
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
	all, teamIDs := g.Teams(p)
	if teamIDs == nil {
		teamIDs = []string{}
	}
	return Scope{Everything: all, PersonID: g.PersonID, Holds: g.Has(p), TeamIDs: teamIDs}
}
