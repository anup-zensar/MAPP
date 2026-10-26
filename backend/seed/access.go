package seed

import (
	"context"

	"mapp/internal/access"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/ids"
)

// The roles of feature 014 (spec Table B, plan "Demo data"), RL-1001 … RL-1005, with their permissions in
// Table A order and the scope the People tab proposes.
var demoRoles = []struct {
	name, builtIn, description string
	permissions                []authz.Permission
	defaultScope               authz.ScopeType
}{
	{"Administrator", authz.BuiltInAdministrator, "Changes everything, including Settings and access.",
		authz.All(), authz.ScopeAll},
	{"Global planner", authz.BuiltInGlobalPlanner, "Plans for every market and shares global budgets.",
		[]authz.Permission{
			authz.PlansView, authz.PlansCreate, authz.PlansEdit, authz.PlansPublish, authz.PlansDelete,
			authz.CampaignsEdit, authz.CampaignsDelete, authz.ActivitiesEdit, authz.ActivitiesDelete,
			authz.ActivitiesSubmit, authz.DataExport, authz.FinanceView, authz.FinanceLink, authz.ApprovalsDecide,
			authz.GlobalBudgetsView, authz.GlobalBudgetsManage,
		}, authz.ScopeAll},
	{"Planner", authz.BuiltInPlanner, "Plans for their own markets.",
		[]authz.Permission{
			authz.PlansView, authz.PlansCreate, authz.PlansEdit, authz.PlansPublish,
			authz.CampaignsEdit, authz.CampaignsDelete, authz.ActivitiesEdit, authz.ActivitiesDelete,
			authz.ActivitiesSubmit, authz.DataExport, authz.FinanceView, authz.FinanceLink,
		}, authz.ScopeMyTeams},
	{"Approver", authz.BuiltInApprover, "Approves activities they are named on.",
		[]authz.Permission{authz.PlansView, authz.DataExport, authz.FinanceView, authz.ApprovalsDecide}, authz.ScopeMyTeams},
	{"Viewer", authz.BuiltInViewer, "Sees plans of their markets without changing them.",
		[]authz.Permission{authz.PlansView, authz.DataExport, authz.FinanceView}, authz.ScopeMyTeams},
}

// demoRoleAssignments are the roles of the six demo people (plan "Demo data"), RA-1001 … RA-1007, in this
// order: the Administrator and the Global planner for All markets, everyone else for their own teams.
var demoRoleAssignments = []struct {
	person, role string
	scope        authz.ScopeType
}{
	{"Alex Morgan", "Administrator", authz.ScopeAll},
	{"Carlos Mendes", "Global planner", authz.ScopeAll},
	{"Daniel Okoro", "Planner", authz.ScopeMyTeams},
	{"Daniel Okoro", "Approver", authz.ScopeMyTeams},
	{"Priya Raman", "Planner", authz.ScopeMyTeams},
	{"Mei Lin", "Planner", authz.ScopeMyTeams},
	{"Sara Novak", "Viewer", authz.ScopeMyTeams},
}

// addAccess creates the five roles, the demo people's roles and the access rules (segregation of duties on,
// the Viewer role for new people) through the access repository, after the teams (FR-040, D-306).
func (d *demo) addAccess(ctx context.Context) error {
	repo := access.NewJSONRepository()
	roles := map[string]string{} // role name → ID
	for _, r := range demoRoles {
		id, err := d.next(ctx, ids.PrefixRole)
		if err != nil {
			return err
		}
		role := access.Role{ID: id, Name: r.name, Description: r.description, BuiltIn: r.builtIn,
			Permissions: authz.InOrder(r.permissions), DefaultScope: r.defaultScope, Meta: d.meta()}
		if err := repo.PutRole(ctx, role); err != nil {
			return err
		}
		roles[r.name] = id
	}
	for _, a := range demoRoleAssignments {
		var r resolver
		personID, roleID := r.id(d.people, "person", a.person), r.id(roles, "role", a.role)
		if r.err != nil {
			return r.err
		}
		id, err := d.next(ctx, ids.PrefixRoleAssignment)
		if err != nil {
			return err
		}
		assignment := access.Assignment{ID: id, PersonID: personID, RoleID: roleID, ScopeType: a.scope,
			TeamIDs: []string{}, Meta: d.meta()}
		if err := repo.PutAssignment(ctx, assignment); err != nil {
			return err
		}
	}
	viewer, err := find(roles, "role", "Viewer")
	if err != nil {
		return err
	}
	return repo.PutRules(ctx, access.Rules{NoSelfApproval: true, NewPersonRoleID: viewer, Meta: d.meta()})
}
