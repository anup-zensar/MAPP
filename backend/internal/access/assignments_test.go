package access

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// The People tab lists everyone, active people first and then by name, with their teams, roles and what
// the roles allow together (FR-017).
func TestListPeopleAccess_EveryoneWithRoles_FR017(t *testing.T) {
	f := newFixture(t)
	f.directory.person(mei.ID).Ref.Active = false

	list, err := f.service.ListPeopleAccess(t.Context(), alex)

	if err != nil {
		t.Fatalf("ListPeopleAccess: %v", err)
	}
	var rows []string
	for _, p := range list {
		row := p.Person.Name + ":"
		for _, r := range p.Roles {
			row += " " + r.Role.Name + " · " + r.Scope.Text
		}
		rows = append(rows, row)
	}
	want := []string{"Alex Morgan: Administrator · All markets", "Carlos Mendes: Global planner · All markets",
		"Daniel Okoro: Approver · My teams Planner · My teams", "Priya Raman: Planner · My teams",
		"Sara Novak: Viewer · My teams", "Mei Lin: Planner · My teams"}
	if !slices.Equal(rows, want) {
		t.Errorf("rows =\n%q\nwant\n%q", rows, want)
	}
	sara := list[4]
	if !reflect.DeepEqual(sara.Teams, []authz.TeamRef{lubricants, mobility}) || sara.Version != 1 || len(sara.Permissions) != 3 {
		t.Errorf("Sara's row = %+v", sara)
	}
}

// A person's roles are saved as a whole with the person's version: the person gets a new version and one
// "changed" entry "Roles: old → new", with roles sorted by name and scopes as screens show them (FR-017,
// FR-037, D-233, US3 scenario 2).
func TestSetPersonRoles_WholeSetLoggedOnThePerson_FR017_FR037(t *testing.T) {
	f := newFixture(t)

	got, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{
		{RoleID: viewerID, Scope: ScopeInput{Type: authz.ScopeMyTeams}},
		{RoleID: plannerID, Scope: ScopeInput{Type: authz.ScopeTeams, TeamIDs: []string{mobility.ID, mobility.ID}}},
	}})

	if err != nil {
		t.Fatalf("SetPersonRoles: %v", err)
	}
	if got.Version != 2 || len(got.Roles) != 2 || got.Roles[0].Role.Name != "Planner" || got.Roles[0].Scope.Text != "Mobility China" ||
		!reflect.DeepEqual(got.Roles[0].Scope.Teams, []authz.TeamRef{mobility}) || got.Roles[1].ID != "RA-1007" {
		t.Errorf("Sara's roles = %+v", got)
	}
	entries := f.log(t)
	if len(entries) != 1 {
		t.Fatalf("log = %+v, want one entry", entries)
	}
	e := entries[0]
	if e.RecordType != "person" || e.RecordID != sara.ID || e.Verb != "changed" || len(e.Changes) != 1 ||
		e.Changes[0].Field != "roles" || e.Changes[0].Label != "Roles" || e.Changes[0].OldText != "Viewer · My teams" ||
		e.Changes[0].NewText != "Planner · Mobility China, Viewer · My teams" {
		t.Errorf("entry = %+v", e)
	}

	_, err = f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{}})
	if got := apiError(t, err); got.Code != errs.CodeVersionConflict {
		t.Errorf("an out-of-date save = %+v, want version_conflict", got)
	}
	same, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(2), Roles: []AssignmentInput{
		{RoleID: plannerID, Scope: ScopeInput{Type: authz.ScopeTeams, TeamIDs: []string{mobility.ID}}},
		{RoleID: viewerID, Scope: ScopeInput{Type: authz.ScopeMyTeams}},
	}})
	if err != nil || same.Version != 2 || len(f.log(t)) != 1 {
		t.Errorf("a save that changes nothing = %+v, %v with %d log entries; want nothing written", same, err, len(f.log(t)))
	}

	removed, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(2), Roles: []AssignmentInput{}})
	entries = f.log(t)
	if err != nil || len(removed.Roles) != 0 || removed.Version != 3 || entries[1].Changes[0].NewText != "No role" {
		t.Errorf("removing every role = %+v, %v; last entry %+v", removed, err, entries[1])
	}
}

// Every problem of a roles list is reported at once, keyed by its position (FR-017, messages 12 and 16).
func TestSetPersonRoles_Problems_FR017(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Roles: []AssignmentInput{
		{RoleID: administratorID, Scope: ScopeInput{Type: authz.ScopeMyTeams}},
		{RoleID: plannerID, Scope: ScopeInput{Type: authz.ScopeTeams}},
		{RoleID: "RL-9999", Scope: ScopeInput{Type: authz.ScopeAll}},
		{RoleID: plannerID, Scope: ScopeInput{Type: authz.ScopeAll}},
		{RoleID: viewerID, Scope: ScopeInput{Type: "everywhere"}},
		{RoleID: approverID, Scope: ScopeInput{Type: authz.ScopeTeams, TeamIDs: []string{"T-9999"}}},
	}})

	got := apiError(t, err)
	want := []errs.FieldError{
		{Field: "roles.0.scope", Message: "The Administrator role always applies to all markets."},
		{Field: "roles.1.teamIds", Message: "Choose at least one team."},
		{Field: "roles.2.roleId", Message: "Choose a role from the list."},
		{Field: "roles.3.roleId", Message: "'Planner' is listed twice."},
		{Field: "roles.4.scope", Message: "Choose All markets, My teams or Chosen teams."},
		{Field: "roles.5.teamIds", Message: "Choose teams from the list."},
		{Field: "version", Message: "Reload the page and try again."},
	}
	if got.Status != http.StatusBadRequest || !reflect.DeepEqual(got.Fields, want) {
		t.Errorf("problems = %d\n%+v\nwant\n%+v", got.Status, got.Fields, want)
	}
	if _, err := f.service.SetPersonRoles(t.Context(), alex, "U-9999", PersonRolesUpdate{Version: version(1)}); apiError(t, err).Status != http.StatusNotFound {
		t.Errorf("an unknown person = %v, want 404", err)
	}
}

// MAPP always keeps an active administrator (FR-021, message 10): the only administrator can't lose
// the role or be deactivated; with a second administrator, both are possible.
func TestLastAdministrator_Guarded_FR021(t *testing.T) {
	f := newFixture(t)
	only := "Alex Morgan is the only administrator. Give someone else the Administrator role first."

	_, setErr := f.service.SetPersonRoles(t.Context(), alex, alex.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{
		{RoleID: plannerID, Scope: ScopeInput{Type: authz.ScopeAll}}}})
	deactivateErr := f.service.CheckDeactivate(t.Context(), alex.ID)

	for _, err := range []error{setErr, deactivateErr} {
		if got := apiError(t, err); got.Status != http.StatusConflict || got.Code != errs.CodeNotAllowed || got.Message != only {
			t.Errorf("err = %+v, want 409 not_allowed: %s", got, only)
		}
	}
	if err := f.service.CheckDeactivate(t.Context(), sara.ID); err != nil {
		t.Errorf("deactivating Sara = %v, want allowed", err)
	}

	if _, err := f.service.SetPersonRoles(t.Context(), alex, carlos.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{
		{RoleID: administratorID, Scope: ScopeInput{Type: authz.ScopeAll}}}}); err != nil {
		t.Fatalf("making Carlos an administrator: %v", err)
	}
	if err := f.service.CheckDeactivate(t.Context(), alex.ID); err != nil {
		t.Errorf("deactivating Alex with a second administrator = %v", err)
	}
	if _, err := f.service.SetPersonRoles(t.Context(), carlos, alex.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{}}); err != nil {
		t.Errorf("taking Alex's role with a second administrator = %v", err)
	}
}

// fakeApprovals is approvals as the impact check reads them: per person, how many workflows name them and
// how many activities wait for them.
type fakeApprovals map[string][2]int

func (f fakeApprovals) ApproverIn(_ context.Context, personID string) (int, int, error) {
	return f[personID][0], f[personID][1], nil
}

// Before saving, the impact check warns when an approver loses the right to approve (message 18, singular
// and plural) and when administrators remove their own administration (message 19), saving nothing
// (FR-022, US3 scenario 5).
func TestCheckRolesImpact_Warnings_FR022(t *testing.T) {
	f := newFixture(t)
	f.service.UseApprovalFacts(fakeApprovals{daniel.ID: {1, 2}, carlos.ID: {2, 1}})
	noApproving := PersonRolesUpdate{Roles: []AssignmentInput{{RoleID: plannerID, Scope: ScopeInput{Type: authz.ScopeMyTeams}}}}

	daniels, err := f.service.CheckRolesImpact(t.Context(), alex, daniel.ID, noApproving)
	if err != nil {
		t.Fatalf("CheckRolesImpact: %v", err)
	}
	carloss, _ := f.service.CheckRolesImpact(t.Context(), alex, carlos.ID, noApproving)
	alexs, _ := f.service.CheckRolesImpact(t.Context(), alex, alex.ID, noApproving)
	saras, _ := f.service.CheckRolesImpact(t.Context(), alex, sara.ID, noApproving)

	want := []ImpactWarning{{Kind: ImpactApproverCannotApprove, Count: 2, Message: "Daniel Okoro is an approver in 1 approval " +
		"workflow and has 2 activities waiting. Without a role that approves, an administrator will have to decide on their behalf."}}
	if !reflect.DeepEqual(daniels.Warnings, want) {
		t.Errorf("Daniel's warnings =\n%+v\nwant\n%+v", daniels.Warnings, want)
	}
	if len(carloss.Warnings) != 1 || carloss.Warnings[0].Message != "Carlos Mendes is an approver in 2 approval workflows and "+
		"has 1 activity waiting. Without a role that approves, an administrator will have to decide on their behalf." {
		t.Errorf("Carlos's warnings = %+v", carloss.Warnings)
	}
	wantAlex := []ImpactWarning{{Kind: ImpactRemovesOwnAdministration, Count: 1,
		Message: "You will no longer be able to change Settings, people or access."}}
	if !reflect.DeepEqual(alexs.Warnings, wantAlex) || len(saras.Warnings) != 0 {
		t.Errorf("Alex's warnings %+v, Sara's %+v", alexs.Warnings, saras.Warnings)
	}
	if entries := f.log(t); len(entries) != 0 {
		t.Errorf("the impact check wrote %d log entries", len(entries))
	}
}

// A person added in Settings › People gets the role for new people with My teams, logged on them, and
// keeps version 1; with "No role" they get nothing (FR-023).
func TestAssignNewPerson_RoleForNewPeople_FR023(t *testing.T) {
	f := newFixture(t)
	jane := person(janeActor, "Marketer", lubricants.ID)
	f.directory.people = append(f.directory.people, jane)
	f.write(t, func(ctx context.Context) error { return f.service.AssignNewPerson(ctx, alex, jane.Ref.ID) })

	got, _ := f.service.PersonRoles(t.Context(), alex, jane.Ref.ID)
	entries := f.log(t)
	if len(got.Roles) != 1 || got.Roles[0].Role.Name != "Viewer" || got.Roles[0].Scope.Type != authz.ScopeMyTeams || got.Version != 1 {
		t.Errorf("Jane's roles = %+v, want Viewer · My teams at version 1", got)
	}
	if len(entries) != 1 || entries[0].RecordID != jane.Ref.ID || entries[0].Changes[0].OldText != "No role" ||
		entries[0].Changes[0].NewText != "Viewer · My teams" {
		t.Errorf("log = %+v", entries)
	}

	if _, err := f.service.UpdateRules(t.Context(), alex, RulesUpdate{Version: version(1), NewPersonRoleID: new(string)}); err != nil {
		t.Fatalf("UpdateRules: %v", err)
	}
	john := person(johnActor, "Marketer")
	f.directory.people = append(f.directory.people, john)
	f.write(t, func(ctx context.Context) error { return f.service.AssignNewPerson(ctx, alex, john.Ref.ID) })
	if got, _ := f.service.PersonRoles(t.Context(), alex, john.Ref.ID); len(got.Roles) != 0 {
		t.Errorf("with no role for new people John got %+v", got.Roles)
	}
}

// A deleted team leaves every Chosen teams scope; an assignment left without a team stays, grants nothing in
// any plan and is flagged (data model §1.2, FR-004).
func TestTeamDeleted_LeavesChosenScopes_FR004(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.SetPersonRoles(t.Context(), alex, sara.ID, PersonRolesUpdate{Version: version(1), Roles: []AssignmentInput{
		{RoleID: plannerID, Scope: ScopeInput{Type: authz.ScopeTeams, TeamIDs: []string{mobility.ID}}}}}); err != nil {
		t.Fatalf("SetPersonRoles: %v", err)
	}

	f.write(t, func(ctx context.Context) error { return f.service.TeamDeleted(ctx, alex, mobility.ID) })
	f.directory.teams = []authz.TeamRef{lubricants, globalOps}

	got, _ := f.service.PersonRoles(t.Context(), alex, sara.ID)
	if len(got.Roles) != 1 || !got.Roles[0].ScopeEmpty || len(got.Roles[0].Scope.Teams) != 0 || got.Roles[0].Scope.Text != "Chosen teams" {
		t.Errorf("Sara's roles after the delete = %+v", got.Roles)
	}
	if g := f.grants(t, sara.ID); g.InScope(authz.ActivitiesEdit, planOf(mobility.ID)) || g.InScope(authz.PlansView, planOf(lubricants.ID)) {
		t.Error("the emptied scope still grants something")
	}
}

// Approval workflows ask who may approve (FR-028), global budgets and 015–017 who holds a permission, and
// 017 the holders of a role for a plan and the roles that approve.
func TestHooks_HoldersAndApprovers_FR028(t *testing.T) {
	f := newFixture(t)

	can, _ := f.service.CanApprove(t.Context(), []string{daniel.ID, priya.ID, carlos.ID, alex.ID})
	if !reflect.DeepEqual(can, map[string]bool{daniel.ID: true, priya.ID: false, carlos.ID: true, alex.ID: true}) {
		t.Errorf("CanApprove = %v", can)
	}
	managers, _ := f.service.HoldersOf(t.Context(), authz.GlobalBudgetsManage, true)
	editors, _ := f.service.HoldersOf(t.Context(), authz.ActivitiesEdit, false)
	if !slices.Equal(managers, []string{alex.ID, carlos.ID}) ||
		!slices.Equal(editors, []string{alex.ID, carlos.ID, daniel.ID, mei.ID, priya.ID}) {
		t.Errorf("HoldersOf = %v and %v", managers, editors)
	}
	lubricantsPlanners, _ := f.service.RoleHolders(t.Context(), plannerID, planOf(lubricants.ID))
	if !slices.Equal(lubricantsPlanners, []string{daniel.ID, priya.ID}) {
		t.Errorf("Planners of a Lubricants plan = %v", lubricantsPlanners)
	}
	roles, _ := f.service.ApprovingRoles(t.Context())
	var names []string
	for _, r := range roles {
		names = append(names, r.Name)
	}
	if !slices.Equal(names, []string{"Administrator", "Approver", "Global planner"}) {
		t.Errorf("ApprovingRoles = %q", names)
	}
}
