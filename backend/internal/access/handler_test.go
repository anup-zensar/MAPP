package access_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"northstar/internal/platform/apitest"
)

// The demo people's e-mails.
const (
	alexsEmail   = "alex.morgan@northstar.demo"
	priyasEmail  = "priya.raman@northstar.demo"
	danielsEmail = "daniel.okoro@northstar.demo"
	sarasEmail   = "sara.novak@northstar.demo"
)

// The answers of the access API, as far as the tests read them.
type (
	roleRef struct {
		ID, Name, BuiltIn string
	}
	scope struct {
		Type  string
		Teams []struct{ ID, Name string }
		Text  string
	}
	assignment struct {
		ID         string
		Role       roleRef
		Scope      scope
		ScopeEmpty bool
	}
	permission struct {
		Key   string
		All   bool
		Teams []struct{ ID, Name string }
	}
	personRoles struct {
		Person      struct{ ID, Name string }
		Teams       []struct{ ID, Name string }
		Roles       []assignment
		Permissions []permission
		Version     int
	}
	role struct {
		ID, Name, Description, BuiltIn, DefaultScope string
		Permissions                                  []string
		People                                       []struct{ ID, Name string }
		Locks                                        []struct{ Field, Reason string }
		Version                                      int
	}
	apiError struct {
		Code, Message string
		Fields        []struct{ Field, Message string }
	}
)

// Every access operation needs "Give roles and change roles and access rules": Priya, a Planner, is refused
// with 403 and the role message; Alex, the administrator, is served (FR-009, FR-017).
func TestAccessOperations_NeedAccessManage_FR009_FR017(t *testing.T) {
	server := apitest.New(t)
	priya, alex := server.SignIn(t, priyasEmail), server.SignIn(t, alexsEmail)
	operations := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/access/permissions", nil},
		{http.MethodGet, "/api/access/people", nil},
		{http.MethodGet, "/api/access/changes", nil},
		{http.MethodGet, "/api/roles", nil},
		{http.MethodPost, "/api/roles", map[string]any{"name": "Analyst", "permissions": []string{"plans.view"}, "defaultScope": "all"}},
		{http.MethodGet, "/api/roles/RL-1005", nil},
		{http.MethodPatch, "/api/roles/RL-1005", map[string]any{"description": "Sees plans.", "version": 1}},
		{http.MethodDelete, "/api/roles/RL-1006", nil},
		{http.MethodGet, "/api/people/U-1006/roles", nil},
		{http.MethodPost, "/api/people/U-1006/roles/impact", map[string]any{"roles": []any{}, "version": 1}},
		{http.MethodPut, "/api/people/U-1006/roles", map[string]any{"roles": []any{}, "version": 1}},
		{http.MethodGet, "/api/access-rules", nil},
		{http.MethodPatch, "/api/access-rules", map[string]any{"noSelfApproval": false, "version": 1}},
	}
	for _, op := range operations {
		refused := priya.Do(t, op.method, op.path, op.body)
		var refusal apiError
		refused.Decode(t, &refusal)
		if refused.Status != http.StatusForbidden || refusal.Code != "forbidden" ||
			refusal.Message != "Changing roles and access isn't part of your role (Planner). Ask an administrator if you need it." {
			t.Errorf("Priya: %s %s = %d %s", op.method, op.path, refused.Status, refused.Body)
		}
		if served := alex.Do(t, op.method, op.path, op.body); served.Status >= 300 {
			t.Errorf("Alex: %s %s = %d %s", op.method, op.path, served.Status, served.Body)
		}
	}
}

// GET /api/access/permissions answers Table A in its five groups (FR-001).
func TestListPermissions_TableA_FR001(t *testing.T) {
	server := apitest.New(t)
	var catalogue struct {
		Groups []struct {
			Name        string
			Permissions []struct {
				Key, Label string
				Scoped     bool
			}
		}
	}

	server.SignIn(t, alexsEmail).Do(t, http.MethodGet, "/api/access/permissions", nil).Decode(t, &catalogue)

	var groups []string
	for _, g := range catalogue.Groups {
		groups = append(groups, g.Name+" "+string(rune('0'+len(g.Permissions))))
	}
	if want := []string{"Plans and planning ;", "Financial data 3", "Approvals 2", "Global budgets 2", "Administration 7"}; !slices.Equal(groups, want) {
		t.Errorf("groups = %q, want %q", groups, want) // ";" is 11 permissions
	}
}

// The People tab: everyone with their roles; Daniel holds two roles (FR-017).
func TestListPeopleAccess_FR017(t *testing.T) {
	server := apitest.New(t)
	var page struct {
		Items []personRoles
		Total int
	}

	server.SignIn(t, alexsEmail).Do(t, http.MethodGet, "/api/access/people", nil).Decode(t, &page)

	var rows []string
	for _, p := range page.Items {
		var roles []string
		for _, r := range p.Roles {
			roles = append(roles, r.Role.Name+" · "+r.Scope.Text)
		}
		rows = append(rows, p.Person.Name+": "+strings.Join(roles, ", "))
	}
	want := []string{"Alex Morgan: Administrator · All markets", "Carlos Mendes: Global planner · All markets",
		"Daniel Okoro: Approver · My teams, Planner · My teams", "Mei Lin: Planner · My teams", "Priya Raman: Planner · My teams",
		"Sara Novak: Viewer · My teams"}
	if page.Total != 6 || !slices.Equal(rows, want) {
		t.Errorf("people (%d) =\n%q\nwant\n%q", page.Total, rows, want)
	}
}

// An administrator gives Sara the Planner role for Mobility China; her next request follows it without
// signing in again, she may now change Mobility activities, and the change is in the activity log and the
// recent access changes (US3 scenarios 1–3, FR-017, FR-020, FR-037, FR-038, SC-003).
func TestSetPersonRoles_AppliesToTheNextRequest_FR017_FR020(t *testing.T) {
	server := apitest.New(t)
	alex, sara := server.SignIn(t, alexsEmail), server.SignIn(t, sarasEmail)
	refusedBefore := sara.Do(t, http.MethodGet, "/api/activities/A-1001/form", nil)

	saved := alex.Do(t, http.MethodPut, "/api/people/U-1006/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1005", "scope": map[string]any{"type": "myTeams"}},
		{"roleId": "RL-1003", "scope": map[string]any{"type": "teams", "teamIds": []string{"T-1002"}}},
	}})

	var got personRoles
	saved.Decode(t, &got)
	if saved.Status != http.StatusOK || got.Version != 2 || len(got.Roles) != 2 || got.Roles[0].Role.Name != "Planner" ||
		got.Roles[0].Scope.Text != "Mobility China" {
		t.Fatalf("PUT = %d %s", saved.Status, saved.Body)
	}
	var me struct {
		Access struct{ Permissions []permission }
	}
	sara.Do(t, http.MethodGet, "/api/me", nil).Decode(t, &me)
	i := slices.IndexFunc(me.Access.Permissions, func(p permission) bool { return p.Key == "activities.edit" })
	if i < 0 || me.Access.Permissions[i].All || len(me.Access.Permissions[i].Teams) != 1 || me.Access.Permissions[i].Teams[0].Name != "Mobility China" {
		t.Errorf("Sara's activities.edit = %+v", me.Access.Permissions)
	}
	if refusedBefore.Status != http.StatusForbidden {
		t.Errorf("before: Sara opening an activity form = %d, want 403", refusedBefore.Status)
	}
	if form := sara.Do(t, http.MethodGet, "/api/activities/A-1010/form", nil); form.Status != http.StatusOK {
		t.Errorf("after: Sara opening a Mobility activity's form = %d %s, want 200", form.Status, form.Body)
	}

	var changes struct {
		Items []struct {
			RecordType, RecordName, Verb string
			Actor                        struct{ Name string }
			Changes                      []struct{ Label, OldText, NewText string }
		}
		Total int
	}
	alex.Do(t, http.MethodGet, "/api/access/changes", nil).Decode(t, &changes)
	if changes.Total != 1 || changes.Items[0].RecordType != "person" || changes.Items[0].RecordName != "Sara Novak" ||
		changes.Items[0].Actor.Name != "Alex Morgan" || changes.Items[0].Changes[0].Label != "Roles" ||
		changes.Items[0].Changes[0].OldText != "Viewer · My teams" ||
		changes.Items[0].Changes[0].NewText != "Planner · Mobility China, Viewer · My teams" {
		t.Errorf("recent access changes = %+v", changes)
	}
}

// Saving roles answers 400 with every problem, 409 for an out-of-date version and for the last
// administrator, 404 for an unknown person (FR-017, FR-021).
func TestSetPersonRoles_Refusals_FR017_FR021(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	invalid := alex.Do(t, http.MethodPut, "/api/people/U-1006/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1001", "scope": map[string]any{"type": "myTeams"}},
		{"roleId": "RL-1003", "scope": map[string]any{"type": "teams", "teamIds": []string{}}},
	}})
	stale := alex.Do(t, http.MethodPut, "/api/people/U-1006/roles", map[string]any{"version": 7, "roles": []any{}})
	lastAdmin := alex.Do(t, http.MethodPut, "/api/people/U-1001/roles", map[string]any{"version": 1, "roles": []any{}})
	alex.Do(t, http.MethodPost, "/api/roles", map[string]any{"name": "People manager", "permissions": []string{"people.manage"},
		"defaultScope": "all"}) // RL-1006, for Priya, who then tries to deactivate Alex
	alex.Do(t, http.MethodPut, "/api/people/U-1002/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1006", "scope": map[string]any{"type": "all"}}}})
	deactivate := server.SignIn(t, priyasEmail).Do(t, http.MethodPost, "/api/people/U-1001/status", map[string]any{"active": false, "version": 1})
	missing := alex.Do(t, http.MethodGet, "/api/people/U-9999/roles", nil)

	assertError(t, invalid, http.StatusBadRequest, "validation_failed", "")
	var problems apiError
	invalid.Decode(t, &problems)
	if len(problems.Fields) != 2 || problems.Fields[0].Message != "The Administrator role always applies to all markets." ||
		problems.Fields[1].Field != "roles.1.teamIds" || problems.Fields[1].Message != "Choose at least one team." {
		t.Errorf("problems = %+v", problems.Fields)
	}
	assertError(t, stale, http.StatusConflict, "version_conflict", "Not saved — NorthStar changed this person")
	only := "Alex Morgan is the only administrator. Give someone else the Administrator role first."
	assertError(t, lastAdmin, http.StatusConflict, "not_allowed", only)
	assertError(t, deactivate, http.StatusConflict, "not_allowed", only)
	assertError(t, missing, http.StatusNotFound, "not_found", "This person doesn't exist.")
}

// The impact check warns before Daniel loses the right to approve and before Alex removes his own
// administration, and saves nothing (FR-022, US3 scenario 5).
func TestCheckPersonRolesImpact_Warnings_FR022(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	plannerOnly := map[string]any{"version": 1, "roles": []map[string]any{{"roleId": "RL-1003", "scope": map[string]any{"type": "myTeams"}}}}
	type warnings struct {
		Warnings []struct {
			Kind, Message string
			Count         int
		}
	}

	var daniels, alexs warnings
	alex.Do(t, http.MethodPost, "/api/people/U-1003/roles/impact", plannerOnly).Decode(t, &daniels)
	alex.Do(t, http.MethodPost, "/api/people/U-1001/roles/impact", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1001", "scope": map[string]any{"type": "all"}}}}).Decode(t, &alexs)
	var after personRoles
	alex.Do(t, http.MethodGet, "/api/people/U-1003/roles", nil).Decode(t, &after)

	if len(daniels.Warnings) != 1 || daniels.Warnings[0].Kind != "approverCannotApprove" ||
		!strings.HasPrefix(daniels.Warnings[0].Message, "Daniel Okoro is an approver in 1 approval workflow and has ") {
		t.Errorf("Daniel's warnings = %+v", daniels.Warnings)
	}
	if len(alexs.Warnings) != 0 {
		t.Errorf("keeping Alex's Administrator role warns %+v", alexs.Warnings)
	}
	if after.Version != 1 || len(after.Roles) != 2 {
		t.Errorf("after the impact check Daniel = %+v, want unchanged", after)
	}
}

// Roles in Settings: create, read, change, delete to the recycle bin and Undo; the Administrator role is
// locked and a role people hold can't be deleted (US4, FR-018, messages 11, 13, 14).
func TestRoles_CreateChangeDelete_FR018(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	created := alex.Do(t, http.MethodPost, "/api/roles", map[string]any{"name": "Analyst", "description": "Reads and exports.",
		"permissions": []string{"data.export", "plans.view"}, "defaultScope": "myTeams"})
	taken := alex.Do(t, http.MethodPost, "/api/roles", map[string]any{"name": "analyst", "permissions": []string{"plans.view"}, "defaultScope": "all"})
	changed := alex.Do(t, http.MethodPatch, "/api/roles/RL-1006", map[string]any{"permissions": []string{"plans.view", "finance.view"}, "version": 1})
	adminChange := alex.Do(t, http.MethodPatch, "/api/roles/RL-1001", map[string]any{"name": "Boss", "version": 1})
	inUse := alex.Do(t, http.MethodDelete, "/api/roles/RL-1003", nil)
	deleted := alex.Do(t, http.MethodDelete, "/api/roles/RL-1006", nil)
	gone := alex.Do(t, http.MethodGet, "/api/roles/RL-1006", nil)

	var r role
	created.Decode(t, &r)
	if created.Status != http.StatusCreated || r.ID != "RL-1006" || !slices.Equal(r.Permissions, []string{"plans.view", "data.export"}) {
		t.Errorf("POST = %d %s", created.Status, created.Body)
	}
	assertError(t, taken, http.StatusBadRequest, "validation_failed", "")
	changed.Decode(t, &r)
	if changed.Status != http.StatusOK || r.Version != 2 || !slices.Equal(r.Permissions, []string{"plans.view", "finance.view"}) {
		t.Errorf("PATCH = %d %s", changed.Status, changed.Body)
	}
	assertError(t, adminChange, http.StatusConflict, "not_allowed", "The Administrator role always has every permission and can't be changed.")
	assertError(t, inUse, http.StatusConflict, "in_use", "'Planner' is given to 3 people. Take it away from them first.")
	var result struct {
		RecycleBinEntry struct{ ID, RecordType, RecordName string }
	}
	deleted.Decode(t, &result)
	if deleted.Status != http.StatusOK || result.RecycleBinEntry.RecordType != "role" || result.RecycleBinEntry.RecordName != "Analyst" {
		t.Errorf("DELETE = %d %s", deleted.Status, deleted.Body)
	}
	assertError(t, gone, http.StatusNotFound, "not_found", "This role no longer exists — someone may have deleted it.")

	undone := alex.Do(t, http.MethodPost, "/api/recycle-bin/"+result.RecycleBinEntry.ID+"/undo", nil)
	back := alex.Do(t, http.MethodGet, "/api/roles/RL-1006", nil)
	if undone.Status != http.StatusOK || back.Status != http.StatusOK {
		t.Errorf("Undo = %d %s; GET after = %d", undone.Status, undone.Body, back.Status)
	}

	var page struct {
		Items []role
		Total int
	}
	alex.Do(t, http.MethodGet, "/api/roles", nil).Decode(t, &page)
	if page.Total != 6 || page.Items[0].Name != "Administrator" || len(page.Items[0].Locks) != 5 || len(page.Items[0].Permissions) != 25 {
		t.Errorf("roles = %d, first %+v", page.Total, page.Items[0])
	}
}

// The access rules: read with their defaults, changed with the version, Administrator refused for new
// people; a person added in Settings › People then gets the chosen role (FR-019, FR-023).
func TestAccessRules_ChangeAndNewPeople_FR019_FR023(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	type rules struct {
		NoSelfApproval bool
		NewPersonRole  *roleRef
		Version        int
	}

	var before rules
	alex.Do(t, http.MethodGet, "/api/access-rules", nil).Decode(t, &before)
	admin := alex.Do(t, http.MethodPatch, "/api/access-rules", map[string]any{"newPersonRoleId": "RL-1001", "version": 1})
	changed := alex.Do(t, http.MethodPatch, "/api/access-rules", map[string]any{"newPersonRoleId": "RL-1004", "version": 1})
	created := alex.Do(t, http.MethodPost, "/api/people", map[string]any{"name": "Jane Doe", "email": "jane.doe@northstar.demo",
		"teamIds": []string{"T-1002"}})

	if !before.NoSelfApproval || before.NewPersonRole == nil || before.NewPersonRole.Name != "Viewer" || before.Version != 1 {
		t.Errorf("rules before = %+v", before)
	}
	assertError(t, admin, http.StatusBadRequest, "validation_failed", "")
	var after rules
	changed.Decode(t, &after)
	if changed.Status != http.StatusOK || after.NewPersonRole == nil || after.NewPersonRole.Name != "Approver" || after.Version != 2 {
		t.Errorf("PATCH = %d %s", changed.Status, changed.Body)
	}
	var jane struct {
		ID      string
		Version int
	}
	created.Decode(t, &jane)
	var janes personRoles
	alex.Do(t, http.MethodGet, "/api/people/"+jane.ID+"/roles", nil).Decode(t, &janes)
	if jane.Version != 1 || len(janes.Roles) != 1 || janes.Roles[0].Role.Name != "Approver" || janes.Roles[0].Scope.Text != "My teams" ||
		janes.Roles[0].Scope.Teams[0].Name != "Mobility China" {
		t.Errorf("Jane (version %d) has roles %+v, want Approver · My teams", jane.Version, janes.Roles)
	}
}

// A deleted team leaves every Chosen teams scope; the People tab flags the empty scope (data model §1.2).
func TestTeamDelete_LeavesChosenScopes_FR004(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	var team struct{ ID string }
	alex.Do(t, http.MethodPost, "/api/teams", map[string]any{"name": "Lubricants Asia"}).Decode(t, &team)
	if saved := alex.Do(t, http.MethodPut, "/api/people/U-1006/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1003", "scope": map[string]any{"type": "teams", "teamIds": []string{team.ID}}}}}); saved.Status != http.StatusOK {
		t.Fatalf("PUT = %d %s", saved.Status, saved.Body)
	}

	deleted := alex.Do(t, http.MethodDelete, "/api/teams/"+team.ID, nil)

	var saras personRoles
	alex.Do(t, http.MethodGet, "/api/people/U-1006/roles", nil).Decode(t, &saras)
	if deleted.Status != http.StatusOK || len(saras.Roles) != 1 || !saras.Roles[0].ScopeEmpty || len(saras.Roles[0].Scope.Teams) != 0 {
		t.Errorf("team delete %d; Sara's roles %+v", deleted.Status, saras.Roles)
	}
}

// assertError checks an error answer's status, code and, when not empty, message.
func assertError(t *testing.T, r *apitest.Response, status int, code, message string) {
	t.Helper()
	var e apiError
	r.Decode(t, &e)
	if r.Status != status || e.Code != code || (message != "" && e.Message != message) {
		t.Errorf("answer = %d %s\nwant %d %s %q", r.Status, r.Body, status, code, message)
	}
}
