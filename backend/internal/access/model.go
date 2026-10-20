// Package access holds roles, role assignments and the access rules (feature 014, data model §1.1–§1.3), builds
// the Grants of the signed-in person on every request (middleware.go) and declares the access rule of every
// API operation (routes.go). The permissions themselves are fixed in code, in platform/authz.
//
// Administrators change roles, people's roles and the rules in Settings › Roles and access; every change
// applies to the person's next request, because Grants are built per request from the stored records
// (FR-020).
package access

import (
	"northstar/internal/platform/authz"
	"northstar/internal/platform/record"
)

// Role is a named set of permissions, stored in the table "roles" (data model §1.1).
type Role struct {
	ID           string             `json:"id"`           // "RL-1001", never reused
	Name         string             `json:"name"`         // 1–60 characters, unique ignoring letter case
	Description  string             `json:"description"`  // 0–300 characters
	BuiltIn      string             `json:"builtIn"`      // authz.BuiltInAdministrator … for seeded roles, "" otherwise
	Permissions  []authz.Permission `json:"permissions"`  // at least one, Table A order; every key for Administrator
	DefaultScope authz.ScopeType    `json:"defaultScope"` // all or myTeams: what the People tab proposes
	record.Meta
}

// administrator reports whether r is the Administrator role, which always holds every permission for All
// markets and cannot be changed or deleted.
func (r Role) administrator() bool {
	return r.BuiltIn == authz.BuiltInAdministrator
}

// permissions returns what r allows: every permission for the Administrator role, whatever is stored.
func (r Role) permissions() []authz.Permission {
	if r.administrator() {
		return authz.All()
	}
	return authz.InOrder(r.Permissions)
}

// ref returns r as other records show it.
func (r Role) ref() authz.RoleRef {
	return authz.RoleRef{ID: r.ID, Name: r.Name, BuiltIn: r.BuiltIn}
}

// Assignment is one role a person holds with its scope, stored in the table "role_assignments" (data model
// §1.2). The pair (PersonID, RoleID) is unique. Assignments are links: they are changed as a person's whole
// set and deleted outright, never to the recycle bin (D-233).
type Assignment struct {
	ID        string          `json:"id"`        // "RA-1001"
	PersonID  string          `json:"personId"`  // an existing person, active or not
	RoleID    string          `json:"roleId"`    // an existing role
	ScopeType authz.ScopeType `json:"scopeType"` // all, myTeams or teams; always all for Administrator
	TeamIDs   []string        `json:"teamIds"`   // only with teams: 1 or more teams; [] otherwise
	record.Meta
}

// rulesID is the id of the one access-rules record.
const rulesID = "access-rules"

// Rules are the access rules, one record in the table "access_rules" (data model §1.3). While none is
// stored, the defaults apply (defaultRules) and the first change creates it.
type Rules struct {
	ID              string `json:"id"`              // always "access-rules"
	NoSelfApproval  bool   `json:"noSelfApproval"`  // nobody approves an activity they own or submitted (FR-026)
	NewPersonRoleID string `json:"newPersonRoleId"` // the role new people get with My teams; "" for none
	record.Meta
}

// The API types below are what the API returns and receives (contract schemas in brackets).

// PermissionInfo is one permission on the role page (PermissionInfo).
type PermissionInfo struct {
	Key    authz.Permission `json:"key"`
	Label  string           `json:"label"`
	Scoped bool             `json:"scoped"`
}

// PermissionGroup is one group of Table A (PermissionGroup).
type PermissionGroup struct {
	Name        string           `json:"name"`
	Permissions []PermissionInfo `json:"permissions"`
}

// PermissionCatalogue is the answer of GET /api/access/permissions (PermissionCatalogue).
type PermissionCatalogue struct {
	Groups []PermissionGroup `json:"groups"`
}

// Lock is a field or action of a role that cannot be used, and why (Lock).
type Lock struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// RoleView is a role as the API returns it (Role): with the people who hold it and its locks.
type RoleView struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	BuiltIn      string             `json:"builtIn"`
	Permissions  []authz.Permission `json:"permissions"`
	DefaultScope authz.ScopeType    `json:"defaultScope"`
	People       []authz.PersonRef  `json:"people"`
	Locks        []Lock             `json:"locks"`
	record.Meta
}

// RolePage is the answer of GET /api/roles (RolePage).
type RolePage struct {
	Items []RoleView `json:"items"`
	Total int        `json:"total"`
}

// RoleInput is the body of POST /api/roles (RoleCreate).
type RoleInput struct {
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Permissions  []authz.Permission `json:"permissions"`
	DefaultScope authz.ScopeType    `json:"defaultScope"`
}

// RoleUpdate is the body of PATCH /api/roles/{roleId} (RoleUpdate): only the fields to change, plus the
// version. A field that is not sent (nil) keeps its value.
type RoleUpdate struct {
	Name         *string             `json:"name"`
	Description  *string             `json:"description"`
	Permissions  *[]authz.Permission `json:"permissions"`
	DefaultScope *authz.ScopeType    `json:"defaultScope"`
	Version      *int                `json:"version"`
}

// ScopeInput is a scope as the roles panel sends it (AccessScopeInput).
type ScopeInput struct {
	Type    authz.ScopeType `json:"type"`
	TeamIDs []string        `json:"teamIds"`
}

// AssignmentInput is one role of a person's roles as the panel sends it (RoleAssignmentInput).
type AssignmentInput struct {
	RoleID string     `json:"roleId"`
	Scope  ScopeInput `json:"scope"`
}

// PersonRolesUpdate is the body of PUT /api/people/{personId}/roles and of its impact check
// (PersonRolesUpdate): the complete list after the change and the person's version.
type PersonRolesUpdate struct {
	Roles   []AssignmentInput `json:"roles"`
	Version *int              `json:"version"`
}

// PersonRoles is a person with their roles (PersonRoles).
type PersonRoles struct {
	Person      authz.PersonRef        `json:"person"`
	Teams       []authz.TeamRef        `json:"teams"`
	Roles       []authz.RoleAssignment `json:"roles"`
	Permissions []authz.MyPermission   `json:"permissions"`
	Version     int                    `json:"version"`
}

// PeopleAccessPage is the answer of GET /api/access/people (PeopleAccessPage).
type PeopleAccessPage struct {
	Items []PersonRoles `json:"items"`
	Total int           `json:"total"`
}

// RulesView is the access rules as the API returns them (AccessRules). Version is 0 while the defaults have
// never been saved; then UpdatedAt and UpdatedBy are null.
type RulesView struct {
	NoSelfApproval bool           `json:"noSelfApproval"`
	NewPersonRole  *authz.RoleRef `json:"newPersonRole"`
	Version        int            `json:"version"`
	UpdatedAt      any            `json:"updatedAt"`
	UpdatedBy      any            `json:"updatedBy"`
}

// RulesUpdate is the body of PATCH /api/access-rules (AccessRulesUpdate).
type RulesUpdate struct {
	NoSelfApproval  *bool   `json:"noSelfApproval"`
	NewPersonRoleID *string `json:"newPersonRoleId"`
	Version         *int    `json:"version"`
}

// The kinds of the two warnings of the roles impact check (ImpactWarning.kind).
const (
	ImpactApproverCannotApprove    = "approverCannotApprove"
	ImpactRemovesOwnAdministration = "removesOwnAdministration"
)

// ImpactWarning is one warning before saving a person's roles (ImpactWarning).
type ImpactWarning struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// ChangeImpact is the answer of POST /api/people/{personId}/roles/impact (ChangeImpact).
type ChangeImpact struct {
	Warnings []ImpactWarning `json:"warnings"`
}
