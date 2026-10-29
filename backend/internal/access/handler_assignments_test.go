package access_test

import (
	"net/http"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
)

// Saving roles answers 400 with every problem, 409 for an out-of-date version and for the last
// administrator, 404 for an unknown person (FR-017, FR-021).
func TestSetPersonRoles_Refusals_FR017_FR021(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	invalid := alex.Do(t, http.MethodPut, "/api/v1/people/U-1006/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1001", "scope": map[string]any{"type": "myTeams"}},
		{"roleId": "RL-1003", "scope": map[string]any{"type": "teams", "teamIds": []string{}}},
	}})
	stale := alex.Do(t, http.MethodPut, "/api/v1/people/U-1006/roles", map[string]any{"version": 7, "roles": []any{}})
	lastAdmin := alex.Do(t, http.MethodPut, "/api/v1/people/U-1001/roles", map[string]any{"version": 1, "roles": []any{}})
	alex.Do(t, http.MethodPost, "/api/v1/roles", map[string]any{"name": "People manager", "permissions": []string{"people.manage"},
		"defaultScope": "all"})
	alex.Do(t, http.MethodPut, "/api/v1/people/U-1002/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1007", "scope": map[string]any{"type": "all"}}}})
	deactivate := server.SignIn(t, priyasEmail).Do(t, http.MethodPost, "/api/v1/people/U-1001/status", map[string]any{"active": false, "version": 1})
	missing := alex.Do(t, http.MethodGet, "/api/v1/people/U-9999/roles", nil)

	assertError(t, invalid, http.StatusBadRequest, "validation_failed", "")
	var answer errorAnswer
	invalid.Decode(t, &answer)
	problems := answer.Error
	if len(problems.Fields) != 2 || problems.Fields[0].Message != "The Administrator role always applies to all markets." ||
		problems.Fields[1].Field != "roles.1.teamIds" || problems.Fields[1].Message != "Choose at least one team." {
		t.Errorf("problems = %+v", problems.Fields)
	}
	assertError(t, stale, http.StatusConflict, "version_conflict", "Not saved — SP&A One changed this person")
	only := "Alex Morgan is the only administrator. Give someone else the Administrator role first."
	assertError(t, lastAdmin, http.StatusConflict, "not_allowed", only)
	assertError(t, deactivate, http.StatusConflict, "not_allowed", only)
	assertError(t, missing, http.StatusNotFound, "not_found", "This person doesn't exist.")
}

// The impact check warns before Daniel loses the right to approve and before Alex removes his own
// administration, and saves nothing (FR-022, US3 scenario 5).
func TestCheckPersonRolesImpact_Warnings_FR022(t *testing.T) {
	t.Parallel()
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
	alex.Do(t, http.MethodPost, "/api/v1/people/U-1003/roles/impact", plannerOnly).Decode(t, &daniels)
	alex.Do(t, http.MethodPost, "/api/v1/people/U-1001/roles/impact", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1001", "scope": map[string]any{"type": "all"}}}}).Decode(t, &alexs)
	var after personRoles
	alex.Do(t, http.MethodGet, "/api/v1/people/U-1003/roles", nil).Decode(t, &after)

	if len(daniels.Warnings) != 1 || daniels.Warnings[0].Kind != "approverCannotApprove" ||
		!strings.HasPrefix(daniels.Warnings[0].Message, "Daniel Okoro is an approver in 2 approval workflows and has ") {
		t.Errorf("Daniel's warnings = %+v", daniels.Warnings)
	}
	if len(alexs.Warnings) != 0 {
		t.Errorf("keeping Alex's Administrator role warns %+v", alexs.Warnings)
	}
	if after.Version != 1 || len(after.Roles) != 2 {
		t.Errorf("after the impact check Daniel = %+v, want unchanged", after)
	}
}

// A deleted team leaves every Chosen teams scope; the People tab flags the empty scope (data model §1.2).
func TestTeamDelete_LeavesChosenScopes_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	var team struct{ ID string }
	alex.Do(t, http.MethodPost, "/api/v1/teams", map[string]any{"name": "Lubricants Asia"}).Decode(t, &team)
	if saved := alex.Do(t, http.MethodPut, "/api/v1/people/U-1006/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1003", "scope": map[string]any{"type": "teams", "teamIds": []string{team.ID}}}}}); saved.Status != http.StatusOK {
		t.Fatalf("PUT = %d %s", saved.Status, saved.Body)
	}

	deleted := alex.Do(t, http.MethodDelete, "/api/v1/teams/"+team.ID, nil)

	var saras personRoles
	alex.Do(t, http.MethodGet, "/api/v1/people/U-1006/roles", nil).Decode(t, &saras)
	if deleted.Status != http.StatusOK || len(saras.Roles) != 1 || !saras.Roles[0].ScopeEmpty || len(saras.Roles[0].Scope.Teams) != 0 {
		t.Errorf("team delete %d; Sara's roles %+v", deleted.Status, saras.Roles)
	}
}
