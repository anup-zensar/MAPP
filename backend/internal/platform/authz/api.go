package authz

// The API types of a person's access (contract schemas in brackets). They live here, not in package access,
// so that package people can answer CurrentUser.access without importing access (which imports people's
// records through an adapter): authz imports no business package. TeamRef and PersonRef have the JSON shape
// of people's TeamRef and PersonRef.

// TeamRef is a team as records show it (TeamRef).
type TeamRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Colour string `json:"colour"`
	Active bool   `json:"active"`
}

// PersonRef is a person as records show them (PersonRef).
type PersonRef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	JobTitle string `json:"jobTitle"`
	Active   bool   `json:"active"`
}

// RoleRef is a role as other records show it (RoleRef).
type RoleRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BuiltIn string `json:"builtIn"`
}

// AccessScope is where one role assignment applies, as screens show it (AccessScope): Teams are the teams it
// covers now, sorted by name (the person's teams for My teams, the chosen teams for Chosen teams; empty for
// All markets).
type AccessScope struct {
	Type  ScopeType `json:"type"`
	Teams []TeamRef `json:"teams"`
	Text  string    `json:"text"`
}

// RoleAssignment is one role a person holds, with its scope (RoleAssignment). ScopeEmpty is true when the
// scope covers no team: chosen teams that were all deleted, or My teams for a person without a team.
type RoleAssignment struct {
	ID         string      `json:"id"`
	Role       RoleRef     `json:"role"`
	Scope      AccessScope `json:"scope"`
	ScopeEmpty bool        `json:"scopeEmpty"`
}

// MyPermission is one permission a person holds, merged over their roles (MyPermission): for All markets
// (always for permissions that are not scoped), or for the teams listed, sorted by name.
type MyPermission struct {
	Key   Permission `json:"key"`
	All   bool       `json:"all"`
	Teams []TeamRef  `json:"teams"`
}

// MyAccess is the signed-in person's access (MyAccess), answered with GET /api/me for navigation, page-level
// gates, the View only note and My profile. Per-record decisions come from the records' own access objects.
type MyAccess struct {
	Roles          []RoleAssignment `json:"roles"`
	Permissions    []MyPermission   `json:"permissions"`    // Table A order; permissions not held are left out
	Administrators []PersonRef      `json:"administrators"` // active holders of the Administrator role, by name
	NoSelfApproval bool             `json:"noSelfApproval"`
}

// Permissions returns what g allows, merged over its roles, in Table A order: the answer's permissions of
// MyAccess and PersonRoles. teams turns team ids into references (sorted by name by the caller).
func (g Grants) Permissions(teams func(ids []string) []TeamRef) []MyPermission {
	list := []MyPermission{}
	for _, p := range catalogue {
		if !g.Has(p.Key) {
			continue
		}
		all, teamIDs := g.Teams(p.Key)
		refs := []TeamRef{}
		if !all {
			refs = teams(teamIDs)
		}
		list = append(list, MyPermission{Key: p.Key, All: all, Teams: refs})
	}
	return list
}
