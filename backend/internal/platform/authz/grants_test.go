package authz_test

import (
	"slices"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

const (
	lubricants = "T-1001"
	mobility   = "T-1002"
	globalOps  = "T-1003"
)

var teamNames = map[string]string{lubricants: "Lubricants Europe", mobility: "Mobility China", globalOps: "Global Marketing Ops"}

// The permissions of the seeded roles (spec Table B), by role ID.
var tableB = map[string][]authz.Permission{
	"RL-1002": {authz.PlansView, authz.PlansCreate, authz.PlansEdit, authz.PlansPublish, authz.PlansDelete,
		authz.CampaignsEdit, authz.CampaignsDelete, authz.ActivitiesEdit, authz.ActivitiesDelete, authz.ActivitiesSubmit,
		authz.DataExport, authz.FinanceView, authz.ApprovalsDecide, authz.GlobalBudgetsView,
		authz.GlobalBudgetsManage},
	"RL-1003": {authz.PlansView, authz.PlansCreate, authz.PlansEdit, authz.PlansPublish, authz.CampaignsEdit,
		authz.CampaignsDelete, authz.ActivitiesEdit, authz.ActivitiesDelete, authz.ActivitiesSubmit, authz.DataExport,
		authz.FinanceView},
	"RL-1004": {authz.PlansView, authz.DataExport, authz.FinanceView, authz.ApprovalsDecide},
	"RL-1005": {authz.PlansView, authz.DataExport, authz.FinanceView},
}

func role(id, name, builtIn string, scope authz.ScopeType, teams ...string) authz.GrantedRole {
	return authz.GrantedRole{ID: id, Name: name, BuiltIn: builtIn, Scope: scope, TeamIDs: teams}
}

var (
	administrator = role("RL-1001", "Administrator", authz.BuiltInAdministrator, authz.ScopeAll)
	globalPlanner = role("RL-1002", "Global planner", authz.BuiltInGlobalPlanner, authz.ScopeAll)
	planner       = role("RL-1003", "Planner", authz.BuiltInPlanner, authz.ScopeMyTeams)
	approver      = role("RL-1004", "Approver", authz.BuiltInApprover, authz.ScopeMyTeams)
	viewer        = role("RL-1005", "Viewer", authz.BuiltInViewer, authz.ScopeMyTeams)
)

func grants(personID string, myTeams []string, roles ...authz.GrantedRole) authz.Grants {
	g := authz.NewGrants(personID, roles, myTeams, tableB, []actor.Actor{{ID: "U-1001", Name: "Alex Morgan"}},
		authz.Rules{NoSelfApproval: true})
	g.TeamNames = teamNames
	return g
}

func plan(teams ...string) authz.PlanFacts {
	return authz.PlanFacts{ID: "P-1001", TeamIDs: teams, OwnerID: "U-1002", CreatedByID: "system"}
}

func TestGrants_UnionOfRoles_FR002(t *testing.T) {
	t.Parallel()
	daniel := grants("U-1003", []string{lubricants}, planner, approver)

	for _, p := range []authz.Permission{authz.PlansEdit, authz.ApprovalsDecide, authz.DataExport} {
		if !daniel.Has(p) {
			t.Errorf("Daniel does not hold %s, want the union of Planner and Approver", p)
		}
	}
	if daniel.Has(authz.PlansDelete) || daniel.Has(authz.SettingsManage) {
		t.Error("Daniel holds a permission none of his roles includes")
	}
}

// My teams is the membership now, Chosen teams the assignment's list; unscoped permissions apply everywhere.
func TestGrants_Scopes_FR004(t *testing.T) {
	t.Parallel()
	priya := grants("U-1002", []string{lubricants}, planner)
	sara := grants("U-1006", []string{lubricants, mobility}, viewer, role("RL-1003", "Planner", authz.BuiltInPlanner, authz.ScopeTeams, mobility))
	carlos := grants("U-1005", []string{mobility, globalOps}, globalPlanner)

	tests := []struct {
		name string
		g    authz.Grants
		p    authz.Permission
		plan authz.PlanFacts
		want bool
	}{
		{"planner edits a plan of her team", priya, authz.PlansEdit, plan(lubricants), true},
		{"planner cannot edit another market's plan", priya, authz.PlansEdit, plan(mobility), false},
		{"one shared team is enough", priya, authz.PlansEdit, plan(mobility, lubricants), true},
		{"viewer sees both her teams", sara, authz.PlansView, plan(mobility), true},
		{"chosen teams: planner only in Mobility", sara, authz.ActivitiesEdit, plan(mobility), true},
		{"chosen teams: not in Lubricants", sara, authz.ActivitiesEdit, plan(lubricants), false},
		{"all markets: every plan", carlos, authz.PlansDelete, plan(lubricants), true},
		{"all markets: a plan without teams", carlos, authz.PlansEdit, plan(), true},
	}
	for _, tt := range tests {
		if got := tt.g.InScope(tt.p, tt.plan); got != tt.want {
			t.Errorf("%s: InScope(%s) = %v, want %v", tt.name, tt.p, got, tt.want)
		}
	}

	all, teams := sara.Teams(authz.PlansView)
	if all || !slices.Equal(teams, []string{lubricants, mobility}) {
		t.Errorf("Sara's plans.view = all %v teams %v, want her two teams", all, teams)
	}
	if all, _ := carlos.Teams(authz.PlansView); !all {
		t.Error("Carlos's plans.view is not All markets")
	}
}

// Daniel's Approver role is My teams, yet "Approve or decline when named" is his everywhere.
func TestGrants_UnscopedPermissionsApplyEverywhere_FR004(t *testing.T) {
	t.Parallel()
	daniel := grants("U-1003", []string{lubricants}, approver)

	if all, _ := daniel.Teams(authz.ApprovalsDecide); !all {
		t.Error("approvals.decide is not everywhere for a My teams role")
	}
	if !daniel.InScope(authz.ApprovalsDecide, plan(mobility)) {
		t.Error("approvals.decide refused for another market's plan")
	}
}

func TestGrants_TeamlessPlanForItsCreator_FR005(t *testing.T) {
	t.Parallel()
	priya := grants("U-1002", []string{lubricants}, planner)
	mei := grants("U-1004", []string{mobility}, planner)
	draft := authz.PlanFacts{ID: "P-1004", CreatedByID: "U-1002", OwnerID: ""}

	if !priya.InScope(authz.PlansEdit, draft) {
		t.Error("the creator cannot edit her plan without teams")
	}
	if mei.InScope(authz.PlansEdit, draft) {
		t.Error("someone else can edit a plan without teams")
	}
	if priya.InScope(authz.PlansDelete, draft) {
		t.Error("the creator can delete it although Planner does not include plans.delete")
	}
}

// An activity is also seen by its owner and its named approvers.
func TestGrants_CanView_FR006(t *testing.T) {
	t.Parallel()
	mei := grants("U-1004", []string{mobility}, planner)
	lubricantsPlan := authz.PlanFacts{ID: "P-1001", TeamIDs: []string{lubricants}, OwnerID: "U-1002"}
	ownedByMei := authz.PlanFacts{ID: "P-1001", TeamIDs: []string{lubricants}, OwnerID: "U-1004"}

	if mei.CanViewPlan(lubricantsPlan) {
		t.Error("Mei sees a Lubricants plan")
	}
	if !mei.CanViewPlan(ownedByMei) {
		t.Error("Mei does not see the plan she owns")
	}
	if !mei.CanViewActivity(authz.ActivityFacts{Plan: lubricantsPlan, OwnerID: "U-1004"}) {
		t.Error("Mei does not see the activity she owns")
	}
	if !mei.CanViewActivity(authz.ActivityFacts{Plan: lubricantsPlan, ApproverIDs: []string{"U-1003", "U-1004"}}) {
		t.Error("Mei does not see an activity she is named to approve")
	}
	if mei.CanViewActivity(authz.ActivityFacts{Plan: lubricantsPlan, OwnerID: "U-1002"}) {
		t.Error("Mei sees another market's activity")
	}
}

func TestGrants_AdministratorHoldsEverything_FR003(t *testing.T) {
	t.Parallel()
	alex := grants("U-1001", nil, administrator)

	for _, p := range append(authz.All(), authz.Permission("future.permission")) {
		if !alex.Has(p) || !alex.InScope(p, plan(mobility)) {
			t.Errorf("the administrator lacks %s", p)
		}
	}
	if !alex.IsAdministrator() {
		t.Error("IsAdministrator = false")
	}
}

// A custom role saved before 038 grants nothing with link alone; with import from any role, link applies in its scope.
func TestGrants_LinkNeedsImport_038_FR006_H4(t *testing.T) {
	t.Parallel()
	perms := map[string][]authz.Permission{"RL-1010": {authz.FinanceView, authz.FinanceLink}, "RL-1011": {authz.FinanceImport}}
	linker := role("RL-1010", "Linker", "", authz.ScopeMyTeams)
	importer := role("RL-1011", "Importer", "", authz.ScopeAll)
	alone := authz.NewGrants("U-1004", []authz.GrantedRole{linker}, []string{lubricants}, perms, nil, authz.Rules{})
	both := authz.NewGrants("U-1004", []authz.GrantedRole{linker, importer}, []string{lubricants}, perms, nil, authz.Rules{})

	if alone.Has(authz.FinanceLink) || alone.InScope(authz.FinanceLink, plan(lubricants)) {
		t.Error("finance.link without finance.import is held")
	}
	if !both.InScope(authz.FinanceLink, plan(lubricants)) || both.InScope(authz.FinanceLink, plan(mobility)) {
		t.Error("finance.link with finance.import does not apply in the link role's scope only")
	}
}

func TestGrants_NoRole_HoldsNothing_FR001(t *testing.T) {
	t.Parallel()
	nobody := grants("U-1007", []string{lubricants})

	for _, p := range authz.All() {
		if nobody.Has(p) {
			t.Errorf("a person without a role holds %s", p)
		}
	}
	if nobody.CanViewPlan(plan(lubricants)) {
		t.Error("a person without a role sees a plan of their team")
	}
}

// An empty scope grants nothing in any plan, while the permission is still "held" for the message.
func TestGrants_EmptyScopeGrantsNothing_FR004(t *testing.T) {
	t.Parallel()
	newcomer := grants("U-1007", nil, planner)

	if !newcomer.Has(authz.PlansCreate) {
		t.Error("Has(plans.create) = false; the role includes it")
	}
	if newcomer.InScope(authz.PlansCreate, plan(lubricants)) {
		t.Error("an empty scope covers a plan")
	}
}

func TestGrants_Permissions_InTableAOrder_FR024(t *testing.T) {
	t.Parallel()
	sara := grants("U-1006", []string{lubricants, mobility}, viewer)
	names := func(ids []string) []authz.TeamRef {
		refs := []authz.TeamRef{}
		for _, id := range ids {
			refs = append(refs, authz.TeamRef{ID: id, Name: teamNames[id]})
		}
		return refs
	}

	got := sara.Permissions(names)

	var keys []authz.Permission
	for _, p := range got {
		keys = append(keys, p.Key)
	}
	if !slices.Equal(keys, []authz.Permission{authz.PlansView, authz.DataExport, authz.FinanceView}) {
		t.Fatalf("keys = %v", keys)
	}
	if got[0].All || len(got[0].Teams) != 2 {
		t.Errorf("plans.view = %+v, want her two teams", got[0])
	}
}
