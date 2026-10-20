package people_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/people"
	"northstar/internal/platform/apitest"
)

// The body of the API's answer for a team that does not exist.
const teamGoneBody = `{"code":"not_found","message":"This team no longer exists — someone may have deleted it.","fields":[]}`

// The Teams list and the team picker get every team, sorted by name, with its colour, status and members
// (FR-010).
func TestListTeams_TeamsWithMembers_FR010(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/teams", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/teams = %d %s", response.Status, response.Body)
	}
	var page people.TeamPage
	response.Decode(t, &page)
	var teams []string
	for _, team := range page.Items {
		teams = append(teams, team.ID+" "+team.Name+" "+team.Colour+": "+memberNames(team.Members))
	}
	want := []string{
		"T-1003 Global Marketing Ops teal: Alex Morgan, Carlos Mendes",
		"T-1001 Lubricants Europe blue: Daniel Okoro, Priya Raman, Sara Novak",
		"T-1002 Mobility China violet: Carlos Mendes, Mei Lin, Sara Novak",
	}
	if !slices.Equal(teams, want) || page.Total != 3 {
		t.Errorf("teams = %q (total %d), want %q", teams, page.Total, want)
	}
}

// A team's details answer its description, members and the plans using it; an unknown team answers 404
// with the message the details panel shows (US2 scenario 3).
func TestGetTeam_DetailsOrNotFound_FR010(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	found := priya.Do(t, http.MethodGet, "/api/teams/T-1001", nil)
	missing := priya.Do(t, http.MethodGet, "/api/teams/T-9999", nil)

	var europe people.TeamView
	found.Decode(t, &europe)
	if found.Status != http.StatusOK || europe.Description != "Marketing for lubricants across Europe." ||
		memberNames(europe.Members) != "Daniel Okoro, Priya Raman, Sara Novak" || europe.Plans == nil {
		t.Errorf("GET /api/teams/T-1001 = %d %s", found.Status, found.Body)
	}
	assertAnswer(t, missing, http.StatusNotFound, teamGoneBody)
}

// Creating a team answers 201 with the team, its members by name and the least-used colour; a form with
// problems answers 400 listing every one (FR-010, FR-011).
func TestCreateTeam_CreatedOrRefused_FR010_FR011(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	created := alex.Do(t, http.MethodPost, "/api/teams", map[string]any{"name": " Lubricants Asia ",
		"description": "Marketing for lubricants across Asia.", "memberIds": []string{"U-1002", "U-1003"}})
	refused := alex.Do(t, http.MethodPost, "/api/teams", map[string]any{"name": "mobility CHINA", "colour": "pink"})

	assertAnswer(t, created, http.StatusCreated, `{"id":"T-1004","name":"Lubricants Asia",`+
		`"description":"Marketing for lubricants across Asia.","colour":"magenta","active":true,"members":[`+
		`{"id":"U-1003","name":"Daniel Okoro","initials":"DO","jobTitle":"Marketing team leader","active":true},`+
		`{"id":"U-1002","name":"Priya Raman","initials":"PR","jobTitle":"Marketer","active":true}],"plans":[],`+
		`"version":1,"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1001","name":"Alex Morgan"},`+
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1001","name":"Alex Morgan"}}`)
	assertAnswer(t, refused, http.StatusBadRequest, validationBody(
		`{"field":"name","message":"A team called 'Mobility China' already exists."},`+
			`{"field":"colour","message":"Choose one of the colours."}`))
}

// A change from the current version answers the team with its new version; one from an out-of-date copy
// answers 409, naming who changed the team and when; an unknown team answers 404 (FR-011, 001 FR-029).
// Someone else than the creator changes a team (FR-011); Daniel and Priya hold the Administrator role for it
// (014 FR-009).
func TestUpdateTeam_ChangedConflictOrNotFound_FR011(t *testing.T) {
	server := apitest.New(t)
	asia := createTeam(t, server.SignIn(t, alexsEmail), map[string]any{"name": "Lubricants Asia", "memberIds": []string{"U-1002"}})
	daniel := server.SignIn(t, danielsEmail)
	server.GrantRole(t, danielsEmail, "administrator", "all") // 014: changing teams
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "administrator", "all") // 014: changing teams
	server.Advance(time.Hour)
	update := map[string]any{"name": "Lubricants Asia Pacific", "description": "", "colour": asia.Colour,
		"memberIds": []string{"U-1002", "U-1003"}, "version": 1}

	changed := daniel.Do(t, http.MethodPatch, "/api/teams/"+asia.ID, update)
	outOfDate := priya.Do(t, http.MethodPatch, "/api/teams/"+asia.ID, update)
	missing := priya.Do(t, http.MethodPatch, "/api/teams/T-9999", update)

	var team people.TeamView
	changed.Decode(t, &team)
	if changed.Status != http.StatusOK || team.Name != "Lubricants Asia Pacific" || team.Version != 2 ||
		memberNames(team.Members) != "Daniel Okoro, Priya Raman" || team.UpdatedBy.Name != "Daniel Okoro" {
		t.Errorf("PATCH = %d %s", changed.Status, changed.Body)
	}
	assertAnswer(t, outOfDate, http.StatusConflict, `{"code":"version_conflict",`+
		`"message":"Not saved — Daniel Okoro changed this team","fields":[],`+
		`"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T10:00:00Z"}}`)
	assertAnswer(t, missing, http.StatusNotFound, teamGoneBody)
}

// Deleting an unused team answers its recycle-bin entry with its memberships; Undo brings both back
// (FR-015). A team that is gone answers 404.
func TestDeleteTeam_ThenUndo_FR015(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	asia := createTeam(t, alex, map[string]any{"name": "Lubricants Asia", "memberIds": []string{"U-1002", "U-1003"}})

	deleted := alex.Do(t, http.MethodDelete, "/api/teams/"+asia.ID, nil)
	gone := alex.Do(t, http.MethodDelete, "/api/teams/"+asia.ID, nil)

	var result history.DeleteResult
	deleted.Decode(t, &result)
	entry := result.RecycleBinEntry
	wantContents := []history.ContentCount{{RecordType: "teamMembership", Label: "team memberships", Count: 2}}
	if deleted.Status != http.StatusOK || entry.RecordType != "team" || entry.RecordID != asia.ID ||
		entry.RecordName != "Lubricants Asia" || !slices.Equal(entry.Contents, wantContents) {
		t.Errorf("DELETE = %d %s", deleted.Status, deleted.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, teamGoneBody)

	undone := alex.Do(t, http.MethodPost, "/api/recycle-bin/"+entry.ID+"/undo", nil)
	back := alex.Do(t, http.MethodGet, "/api/teams/"+asia.ID, nil)

	var team people.TeamView
	back.Decode(t, &team)
	if undone.Status != http.StatusOK || back.Status != http.StatusOK || memberNames(team.Members) != "Daniel Okoro, Priya Raman" {
		t.Errorf("after Undo: %d %s; GET %d %s", undone.Status, undone.Body, back.Status, back.Body)
	}
}

// A Restore from the recycle bin of a team whose name was taken meanwhile renames it (FR-015).
func TestRestoreTeam_NameTaken_FR015(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	asia := createTeam(t, alex, map[string]any{"name": "Lubricants Asia"})
	var result history.DeleteResult
	alex.Do(t, http.MethodDelete, "/api/teams/"+asia.ID, nil).Decode(t, &result)
	createTeam(t, alex, map[string]any{"name": "Lubricants Asia"})

	response := alex.Do(t, http.MethodPost, "/api/recycle-bin/"+result.RecycleBinEntry.ID+"/restore", nil)

	assertAnswer(t, response, http.StatusOK, `{"recordType":"team","recordTypeLabel":"team","recordId":"`+asia.ID+`",`+
		`"recordName":"Lubricants Asia (restored)","details":"Renamed to 'Lubricants Asia (restored)' because the name was taken."}`)
}

// Without plans that use the team, removing a member needs no confirmation; the check validates the form
// like the save and saves nothing (FR-013, D-058).
func TestTeamImpact_NoPlansNoWarnings_FR013(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	asia := createTeam(t, alex, map[string]any{"name": "Lubricants Asia", "memberIds": []string{"U-1002", "U-1003"}})
	form := map[string]any{"name": "Lubricants Asia", "description": "", "colour": asia.Colour,
		"memberIds": []string{"U-1003"}, "version": 1}

	impact := alex.Do(t, http.MethodPost, "/api/teams/"+asia.ID+"/impact", form)
	form["name"] = " "
	invalid := alex.Do(t, http.MethodPost, "/api/teams/"+asia.ID+"/impact", form)
	missing := alex.Do(t, http.MethodPost, "/api/teams/T-9999/impact", form)

	assertAnswer(t, impact, http.StatusOK, `{"warnings":[]}`)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"name","message":"Enter a name."}`))
	assertAnswer(t, missing, http.StatusNotFound, teamGoneBody)
	var team people.TeamView
	alex.Do(t, http.MethodGet, "/api/teams/"+asia.ID, nil).Decode(t, &team)
	if team.Version != 1 || len(team.Members) != 2 {
		t.Errorf("after the impact check the team = %+v, want it unchanged", team)
	}
}

// Set team inactive and Reactivate answer the team in its new status; both fields of the body are needed,
// and an out-of-date version is refused (FR-014).
func TestSetTeamStatus_InactiveAndBack_FR014(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	asia := createTeam(t, alex, map[string]any{"name": "Lubricants Asia"})
	path := "/api/teams/" + asia.ID + "/status"

	var inactive, active people.TeamView
	alex.Do(t, http.MethodPost, path, map[string]any{"active": false, "version": 1}).Decode(t, &inactive)
	alex.Do(t, http.MethodPost, path, map[string]any{"active": true, "version": 2}).Decode(t, &active)
	missing := alex.Do(t, http.MethodPost, path, map[string]any{}, apitest.SkipRequestValidation())
	outOfDate := alex.Do(t, http.MethodPost, path, map[string]any{"active": false, "version": 1})

	if inactive.Active || inactive.Version != 2 || !active.Active || active.Version != 3 {
		t.Errorf("status = %+v then %+v, want inactive at version 2, then active at version 3", inactive, active)
	}
	assertAnswer(t, missing, http.StatusBadRequest, validationBody(
		`{"field":"active","message":"Choose active or inactive."},{"field":"version","message":"Reload the page and try again."}`))
	if outOfDate.Status != http.StatusConflict {
		t.Errorf("out-of-date status = %d %s, want 409", outOfDate.Status, outOfDate.Body)
	}
}

// createTeam creates a team through POST /api/teams and stops the test when that fails.
func createTeam(t *testing.T, client *apitest.Client, body map[string]any) people.TeamView {
	t.Helper()
	response := client.Do(t, http.MethodPost, "/api/teams", body)
	if response.Status != http.StatusCreated {
		t.Fatalf("POST /api/teams %v = %d %s", body, response.Status, response.Body)
	}
	var team people.TeamView
	response.Decode(t, &team)
	return team
}

// validationBody is the body of a validation_failed answer with the field problems fields.
func validationBody(fields string) string {
	return `{"code":"validation_failed","message":"Some fields need your attention.","fields":[` + fields + `]}`
}

// memberNames joins the names of members with ", ".
func memberNames(members []people.PersonRef) string {
	var names string
	for i, m := range members {
		if i > 0 {
			names += ", "
		}
		names += m.Name
	}
	return names
}
