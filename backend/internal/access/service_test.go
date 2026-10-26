package access

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/store"
)

// A person may do what any of their roles allows (FR-002): Daniel is a Planner and an Approver, both for his
// teams; Sara, a Viewer, may only see and export.
func TestGrantsFor_UnionOfRoles_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	danielG, saraG := f.grants(t, daniel.ID), f.grants(t, sara.ID)

	for _, p := range []authz.Permission{authz.PlansCreate, authz.ActivitiesEdit, authz.ApprovalsDecide, authz.FinanceLink} {
		if !danielG.InScope(p, planOf(lubricants.ID)) {
			t.Errorf("Daniel does not hold %s in Lubricants Europe", p)
		}
	}
	if danielG.Has(authz.PlansDelete) || danielG.Has(authz.SettingsManage) || danielG.InScope(authz.PlansView, planOf(mobility.ID)) {
		t.Error("Daniel holds more than Planner and Approver for his team")
	}
	if !saraG.InScope(authz.PlansView, planOf(mobility.ID)) || saraG.Has(authz.ActivitiesEdit) {
		t.Error("Sara is not exactly a Viewer of her two teams")
	}
	if got := danielG.RoleNames(); !slices.Equal(got, []string{"Approver", "Planner"}) {
		t.Errorf("Daniel's roles = %q, want sorted by name", got)
	}
	if danielG.Roles[0].ScopeText != "My teams" || len(danielG.Administrators) != 1 || danielG.Administrators[0] != alex {
		t.Errorf("Daniel's grants = %+v", danielG)
	}
}

// Scopes (FR-004, FR-005): All markets applies everywhere, My teams follows the person's teams at the moment
// of the action, Chosen teams only the teams chosen; the Administrator role holds every permission.
func TestGrantsFor_Scopes_FR004_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	if g := f.grants(t, carlos.ID); !g.InScope(authz.PlansDelete, planOf(lubricants.ID)) || g.Has(authz.SettingsManage) {
		t.Error("Carlos, Global planner for All markets, does not delete Lubricants plans, or changes Settings")
	}
	if g := f.grants(t, alex.ID); !g.IsAdministrator() || !g.Has(authz.RecycleBinPurge) || !g.InScope(authz.PlansDelete, planOf()) {
		t.Error("Alex, Administrator, does not hold every permission everywhere")
	}

	f.directory.person(priya.ID).TeamIDs = []string{mobility.ID} // Priya moves to Mobility China
	g := f.grants(t, priya.ID)
	if !g.InScope(authz.ActivitiesEdit, planOf(mobility.ID)) || g.InScope(authz.PlansView, planOf(lubricants.ID)) {
		t.Error("My teams does not follow Priya's teams now")
	}

	if _, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{
		{RoleID: plannerID, Scope: ScopeInput{Type: authz.ScopeTeams, TeamIDs: []string{mobility.ID}}},
		{RoleID: viewerID, Scope: ScopeInput{Type: authz.ScopeMyTeams}},
	}}); err != nil {
		t.Fatalf("SetPersonRoles: %v", err)
	}
	g = f.grants(t, sara.ID)
	if !g.InScope(authz.ActivitiesEdit, planOf(mobility.ID)) || g.InScope(authz.ActivitiesEdit, planOf(lubricants.ID)) ||
		!g.InScope(authz.PlansView, planOf(lubricants.ID)) {
		t.Error("Sara's Chosen teams scope is not Mobility China only, or her Viewer role is lost")
	}
}

// A role or rule change applies to the person's next request, without signing in again (FR-020).
func TestGrantsFor_ChangesApplyAtOnce_FR020(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if f.grants(t, sara.ID).Has(authz.ActivitiesEdit) {
		t.Fatal("Sara may already change activities")
	}

	if _, err := f.service.UpdateRole(t.Context(), alex, viewerID, RoleUpdate{Version: version(1),
		Permissions: &[]authz.Permission{authz.PlansView, authz.ActivitiesEdit}}); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if _, err := f.service.UpdateRules(t.Context(), alex, RulesUpdate{Version: version(1), NoSelfApproval: new(bool)}); err != nil {
		t.Fatalf("UpdateRules: %v", err)
	}

	g := f.grants(t, sara.ID)
	if !g.Has(authz.ActivitiesEdit) || g.Has(authz.DataExport) || g.Rules.NoSelfApproval {
		t.Errorf("Sara's next grants = %+v, want the changed Viewer role and the rule off", g)
	}
}

// A person without a role holds nothing, and is told so (FR-001, message 1).
func TestGrantsFor_NoRole_FR001(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if _, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{}}); err != nil {
		t.Fatalf("SetPersonRoles: %v", err)
	}

	g := f.grants(t, sara.ID)

	if g.Has(authz.PlansView) || len(g.Roles) != 0 {
		t.Errorf("Sara without a role = %+v", g)
	}
	if got, want := g.Refusal(authz.PlansView, nil), "Viewing this plan needs a role, and you don't have one yet. Ask an administrator."; got != want {
		t.Errorf("refusal = %q, want %q", got, want)
	}
}

// GET /api/v1/me's access (FR-024): the roles with scopes, the merged permissions in Table A order (for All
// markets, or the teams sorted by name), the administrators and the segregation-of-duties rule.
func TestSummary_RolesPermissionsAdministrators_FR024(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	got, err := f.service.Summary(t.Context(), sara.ID)

	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	both := []authz.TeamRef{lubricants, mobility}
	want := authz.MyAccess{
		Roles: []authz.RoleAssignment{{ID: "RA-1007", Role: authz.RoleRef{ID: viewerID, Name: "Viewer", BuiltIn: authz.BuiltInViewer},
			Scope: authz.AccessScope{Type: authz.ScopeMyTeams, Teams: both, Text: "My teams"}}},
		Permissions: []authz.MyPermission{
			{Key: authz.PlansView, Teams: both}, {Key: authz.DataExport, Teams: both}, {Key: authz.FinanceView, Teams: both}},
		Administrators: []authz.PersonRef{{ID: alex.ID, Name: alex.Name, Initials: "AM", JobTitle: "Marketing administrator", Active: true}},
		NoSelfApproval: true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Summary =\n%+v\nwant\n%+v", got, want)
	}

	alexs, _ := f.service.Summary(t.Context(), alex.ID)
	if len(alexs.Permissions) != 25 || !alexs.Permissions[0].All || len(alexs.Permissions[0].Teams) != 0 {
		t.Errorf("Alex's permissions = %+v, want all 25 for All markets", alexs.Permissions)
	}
}

// Only active administrators are named to ask (FR-024): an inactive one is left out.
func TestSummary_ActiveAdministratorsOnly_FR024(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if _, err := f.service.SetPersonRoles(t.Context(), alex, carlos.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{
		{RoleID: administratorID, Scope: ScopeInput{Type: authz.ScopeAll}},
	}}); err != nil {
		t.Fatalf("SetPersonRoles: %v", err)
	}
	f.directory.person(alex.ID).Ref.Active = false

	got, _ := f.service.Summary(t.Context(), sara.ID)

	if len(got.Administrators) != 1 || got.Administrators[0].ID != carlos.ID {
		t.Errorf("administrators = %+v, want Carlos only", got.Administrators)
	}
}

// newEmptyStoreService returns an access service on a data folder without roles or access rules.
func newEmptyStoreService(t *testing.T) *Service {
	t.Helper()
	f := newFixture(t)
	f.write(t, func(ctx context.Context) error {
		roles, err := NewJSONRepository().ListRoles(ctx)
		if err != nil {
			return err
		}
		for _, r := range roles {
			if err := store.Delete(ctx, rolesTable, r.ID); err != nil {
				return err
			}
		}
		return store.Delete(ctx, rulesTable, rulesID)
	})
	return f.service
}
