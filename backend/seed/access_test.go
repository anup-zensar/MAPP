package seed_test

import (
	"slices"
	"strings"
	"testing"

	"mapp/internal/access"
	"mapp/internal/platform/authz"
)

// The seed holds the five roles of spec Table B (RL-1001 … RL-1005) and 032's Requisitioner with their permissions,
// descriptions and default scopes, the roles of the seven demo people and the access rules' defaults (FR-040).
func TestSeed_RolesAssignmentsAndRules_FR040(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)

	type role struct {
		id, name, builtIn string
		permissions       int
		scope             authz.ScopeType
	}
	var roles []role
	for _, r := range d.roles {
		roles = append(roles, role{r.ID, r.Name, r.BuiltIn, len(r.Permissions), r.DefaultScope})
		if r.Description == "" || r.Version != 1 {
			t.Errorf("role %s: description %q, version %d", r.ID, r.Description, r.Version)
		}
		if !slices.Equal(r.Permissions, authz.InOrder(r.Permissions)) {
			t.Errorf("role %s: permissions %v are not in Table A order", r.ID, r.Permissions)
		}
	}
	wantRoles := []role{
		{"RL-1001", "Administrator", authz.BuiltInAdministrator, 29, authz.ScopeAll},
		{"RL-1002", "Global planner", authz.BuiltInGlobalPlanner, 20, authz.ScopeAll},
		{"RL-1003", "Planner", authz.BuiltInPlanner, 15, authz.ScopeMyTeams},
		{"RL-1004", "Approver", authz.BuiltInApprover, 4, authz.ScopeMyTeams},
		{"RL-1005", "Viewer", authz.BuiltInViewer, 3, authz.ScopeMyTeams},
		{"RL-1006", "Requisitioner", "", 3, authz.ScopeAll},
	}
	if !slices.Equal(roles, wantRoles) {
		t.Errorf("roles =\n%v\nwant\n%v", roles, wantRoles)
	}

	// Table B, row by row: who holds what.
	holds := func(builtIn string, p authz.Permission) bool {
		i := slices.IndexFunc(d.roles, func(r access.Role) bool { return r.BuiltIn == builtIn })
		return i >= 0 && slices.Contains(d.roles[i].Permissions, p)
	}
	tableB := []struct {
		p                                                   authz.Permission
		admin, globalPlanner, planner, approver, viewerRole bool
	}{
		{authz.PlansView, true, true, true, true, true},
		{authz.PlansCreate, true, true, true, false, false},
		{authz.PlansDelete, true, true, false, false, false},
		{authz.ActivitiesSubmit, true, true, true, false, false},
		{authz.DataExport, true, true, true, true, true},
		{authz.FinanceView, true, true, true, true, true},
		{authz.FinanceLink, true, true, true, false, false},
		{authz.FinanceImport, true, false, false, false, false},
		{authz.ApprovalsDecide, true, true, false, true, false},
		{authz.ApprovalsDecideOnBehalf, true, false, false, false, false},
		{authz.GlobalBudgetsManage, true, true, false, false, false},
		{authz.SettingsManage, true, false, false, false, false},
		{authz.AccessManage, true, false, false, false, false},
	}
	for _, row := range tableB {
		got := []bool{holds(authz.BuiltInAdministrator, row.p), holds(authz.BuiltInGlobalPlanner, row.p),
			holds(authz.BuiltInPlanner, row.p), holds(authz.BuiltInApprover, row.p), holds(authz.BuiltInViewer, row.p)}
		want := []bool{row.admin, row.globalPlanner, row.planner, row.approver, row.viewerRole}
		if !slices.Equal(got, want) {
			t.Errorf("%s held by %v, want %v (Administrator, Global planner, Planner, Approver, Viewer)", row.p, got, want)
		}
	}

	var assignments []string
	for _, a := range d.assignments {
		assignments = append(assignments, strings.Join([]string{a.ID, a.PersonID, a.RoleID, string(a.ScopeType)}, " "))
		if len(a.TeamIDs) != 0 {
			t.Errorf("assignment %s has chosen teams %v", a.ID, a.TeamIDs)
		}
	}
	wantAssignments := []string{
		"RA-1001 U-1001 RL-1001 all",
		"RA-1002 U-1005 RL-1002 all",
		"RA-1003 U-1003 RL-1003 myTeams",
		"RA-1004 U-1003 RL-1004 myTeams",
		"RA-1005 U-1002 RL-1003 myTeams",
		"RA-1006 U-1004 RL-1003 myTeams",
		"RA-1007 U-1006 RL-1005 myTeams",
		"RA-1008 U-1007 RL-1006 all",
	}
	if !slices.Equal(assignments, wantAssignments) {
		t.Errorf("assignments =\n%q\nwant\n%q", assignments, wantAssignments)
	}

	if !d.accessRules.NoSelfApproval || d.accessRules.NewPersonRoleID != "RL-1005" || d.accessRules.Version != 1 {
		t.Errorf("access rules = %+v, want segregation of duties on and the Viewer role for new people", d.accessRules)
	}
}
