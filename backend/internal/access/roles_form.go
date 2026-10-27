package access

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// The longest role name and description, in characters (data model §1.1).
const (
	maxRoleNameLength    = 60
	maxDescriptionLength = 300
)

// checkRoleName returns name trimmed and adds its problem: required, at most 60 characters, and no other role
// than exceptID has it, ignoring letter case (message 14).
func checkRoleName(problems *errs.Problems, w world, name, exceptID string) string {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		problems.Add("name", msgEnterRoleName)
	case utf8.RuneCountInString(name) > maxRoleNameLength:
		problems.Add("name", msgUse60)
	case nameTaken(sortedRoles(w.roles), name, exceptID):
		problems.Add("name", fmt.Sprintf(msgRoleNameTaken, name))
	}
	return name
}

// checkDescription returns the description trimmed and adds its problem: at most 300 characters.
func checkDescription(problems *errs.Problems, description string) string {
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		problems.Add("description", msgUse300)
	}
	return description
}

// checkPermissions returns the permissions without repeats in Table A order and adds their problems: at
// least one (message 15), each a key of Table A.
func checkPermissions(problems *errs.Problems, keys []authz.Permission) []authz.Permission {
	for _, key := range keys {
		if _, known := authz.Lookup(string(key)); !known {
			problems.Add("permissions", fmt.Sprintf(msgUnknownPermission, key))
		}
	}
	ordered := authz.InOrder(keys)
	if len(ordered) == 0 && len(keys) == 0 {
		problems.Add("permissions", msgChoosePermission)
	}
	return ordered
}

// checkDefaultScope adds the problem of a default scope that is neither All markets nor My teams.
func checkDefaultScope(problems *errs.Problems, scope authz.ScopeType) authz.ScopeType {
	if scope != authz.ScopeAll && scope != authz.ScopeMyTeams {
		problems.Add("defaultScope", msgChooseDefaultScope)
	}
	return scope
}
