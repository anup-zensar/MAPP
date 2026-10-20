package people_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/auth"
	"northstar/internal/people"
	"northstar/internal/platform/apitest"
)

const (
	alexsEmail   = "alex.morgan@northstar.demo"
	danielsEmail = "daniel.okoro@northstar.demo"
	janesEmail   = "jane.doe@northstar.demo" // the person the tests create
)

// The bodies of the API's refusals about people.
const (
	personGoneBody         = `{"code":"not_found","message":"This person doesn't exist.","fields":[]}`
	noSuchPathBody         = `{"code":"not_found","message":"This page or record doesn't exist.","fields":[]}`
	incorrectSignInBody    = `{"code":"incorrect_credentials","message":"Incorrect e-mail or password.","fields":[]}`
	deactivateYourselfBody = `{"code":"not_allowed","message":"You can't deactivate yourself — ask someone else to do it.","fields":[]}`
)

// Every person picker filters GET /api/people in the browser (D-069), so the list gives everyone with what
// the picker matches and shows — name, e-mail, job title, initials — whether they are active (inactive
// people are never offered but are labelled where a field holds them) and their teams, for the
// team-restricted mode (FR-017, FR-018, FR-019).
func TestListPeople_GivesPickerData_FR017_FR018(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/people", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/people = %d %s", response.Status, response.Body)
	}
	var page people.PersonPage
	response.Decode(t, &page)
	var names []string
	for _, p := range page.Items {
		names = append(names, p.Name)
	}
	want := []string{"Alex Morgan", "Carlos Mendes", "Daniel Okoro", "Mei Lin", "Priya Raman", "Sara Novak"}
	if !slices.Equal(names, want) || page.Total != len(want) {
		t.Fatalf("people = %q (total %d), want %q", names, page.Total, want)
	}
	sara := page.Items[5]
	wantTeams := []people.TeamRef{
		{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue", Active: true},
		{ID: "T-1002", Name: "Mobility China", Colour: "violet", Active: true},
	}
	if sara.ID != "U-1006" || sara.Email != "sara.novak@northstar.demo" || sara.JobTitle != "Local finance advisor" ||
		sara.Initials != "SN" || !sara.Active || !slices.Equal(sara.Teams, wantTeams) {
		t.Errorf("Sara = %+v, want her e-mail, job title, initials, status and two teams", sara)
	}
}

// A person's details answer their teams and what they own or approve; an unknown person answers 404 with
// the message the details panel shows (FR-004).
func TestGetPerson_DetailsOrNotFound_FR004(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	found := alex.Do(t, http.MethodGet, "/api/people/U-1005", nil)
	missing := alex.Do(t, http.MethodGet, "/api/people/U-9999", nil)

	var carlos people.PersonDetails
	found.Decode(t, &carlos)
	wantTeams := []people.TeamRef{
		{ID: "T-1003", Name: "Global Marketing Ops", Colour: "teal", Active: true},
		{ID: "T-1002", Name: "Mobility China", Colour: "violet", Active: true},
	}
	if found.Status != http.StatusOK || carlos.Name != "Carlos Mendes" || !slices.Equal(carlos.Teams, wantTeams) ||
		carlos.Usage == nil || carlos.Version != 1 {
		t.Errorf("GET /api/people/U-1005 = %d %s", found.Status, found.Body)
	}
	assertAnswer(t, missing, http.StatusNotFound, personGoneBody)
}

// People are never deleted (FR-008): there is no DELETE operation, so it answers the JSON 404 of any
// unknown path, and the person is still there.
func TestDeleteNotOffered_FR008(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodDelete, "/api/people/U-1002", nil, apitest.SkipValidation())

	assertAnswer(t, response, http.StatusNotFound, noSuchPathBody)
	if still := alex.Do(t, http.MethodGet, "/api/people/U-1002", nil); still.Status != http.StatusOK {
		t.Errorf("GET /api/people/U-1002 after DELETE = %d %s", still.Status, still.Body)
	}
}

// Creating a person answers 201 with the person as saved — trimmed, e-mail in lower case, active, with
// their teams; a form with problems answers 400 listing every one of them (FR-001, FR-002, SC-007).
func TestCreatePerson_CreatedOrRefused_FR001_FR002(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	created := alex.Do(t, http.MethodPost, "/api/people", map[string]any{"name": " Jane Doe ",
		"email": " Jane.Doe@NorthStar.Demo", "jobTitle": "Marketer", "teamIds": []string{"T-1001"}})
	refused := alex.Do(t, http.MethodPost, "/api/people", map[string]any{"name": " ",
		"email": "PRIYA.raman@northstar.demo", "jobTitle": strings.Repeat("j", 121), "teamIds": []string{"T-9999"}})
	invalidEmail := alex.Do(t, http.MethodPost, "/api/people", map[string]any{"name": "Jane", "email": "jane@northstar"})

	assertAnswer(t, created, http.StatusCreated, `{"id":"U-1007","name":"Jane Doe","email":"jane.doe@northstar.demo",`+
		`"jobTitle":"Marketer","initials":"JD","active":true,`+
		`"teams":[{"id":"T-1001","name":"Lubricants Europe","colour":"blue","active":true}],`+
		`"version":1,"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1001","name":"Alex Morgan"},`+
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1001","name":"Alex Morgan"},"usage":[]}`)
	assertAnswer(t, refused, http.StatusBadRequest, validationBody(`{"field":"name","message":"Enter a name."},`+
		`{"field":"email","message":"priya.raman@northstar.demo already belongs to Priya Raman."},`+
		`{"field":"jobTitle","message":"Use 120 characters or fewer."},`+
		`{"field":"teamIds","message":"Choose teams from the list."}`))
	assertAnswer(t, invalidEmail, http.StatusBadRequest,
		validationBody(`{"field":"email","message":"Enter a valid e-mail address."}`))
}

// A change from the current version answers the person with their new version; one from an out-of-date
// copy answers 409, naming who changed the person and when; a form with problems answers 400 with every
// problem, a missing version included; an unknown person answers 404 (FR-002, FR-012, 001 FR-029).
func TestUpdatePerson_ChangedConflictInvalidOrNotFound_FR002(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	jane := createPerson(t, alex, map[string]any{"name": "Jane Doe", "email": janesEmail})
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "administrator", "all") // 014: changing people
	server.Advance(time.Hour)
	update := map[string]any{"name": "Jane Smith", "email": "jane.smith@northstar.demo", "jobTitle": "Marketer",
		"teamIds": []string{"T-1001"}, "version": 1}

	changed := priya.Do(t, http.MethodPatch, "/api/people/"+jane.ID, update)
	outOfDate := alex.Do(t, http.MethodPatch, "/api/people/"+jane.ID, update)
	invalid := alex.Do(t, http.MethodPatch, "/api/people/"+jane.ID, map[string]any{"name": "",
		"email": "jane.smith@", "jobTitle": "", "teamIds": []string{}}, apitest.SkipRequestValidation())
	missing := alex.Do(t, http.MethodPatch, "/api/people/U-9999", update)

	var person people.PersonDetails
	changed.Decode(t, &person)
	if changed.Status != http.StatusOK || person.Name != "Jane Smith" || person.Email != "jane.smith@northstar.demo" ||
		person.Version != 2 || len(person.Teams) != 1 || person.UpdatedBy.Name != "Priya Raman" {
		t.Errorf("PATCH = %d %s", changed.Status, changed.Body)
	}
	assertAnswer(t, outOfDate, http.StatusConflict, `{"code":"version_conflict",`+
		`"message":"Not saved — Priya Raman changed this person","fields":[],`+
		`"conflict":{"changedBy":"Priya Raman","changedAt":"2026-03-12T10:00:00Z"}}`)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"name","message":"Enter a name."},`+
		`{"field":"email","message":"Enter a valid e-mail address."},`+
		`{"field":"version","message":"Reload the page and try again."}`))
	assertAnswer(t, missing, http.StatusNotFound, personGoneBody)
}

// Without plans the person owns, leaving a team needs no confirmation; the check validates the form like
// the save, finds an unknown person, and saves nothing (FR-013, D-058).
func TestPersonImpact_NoPlansNoWarnings_FR013(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	jane := createPerson(t, alex, map[string]any{"name": "Jane Doe", "email": janesEmail, "teamIds": []string{"T-1001"}})
	form := map[string]any{"name": "Jane Doe", "email": janesEmail, "jobTitle": "", "teamIds": []string{}, "version": 1}

	impact := alex.Do(t, http.MethodPost, "/api/people/"+jane.ID+"/impact", form)
	form["email"] = "jane"
	invalid := alex.Do(t, http.MethodPost, "/api/people/"+jane.ID+"/impact", form)
	missing := alex.Do(t, http.MethodPost, "/api/people/U-9999/impact", form)

	assertAnswer(t, impact, http.StatusOK, `{"warnings":[]}`)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"email","message":"Enter a valid e-mail address."}`))
	assertAnswer(t, missing, http.StatusNotFound, personGoneBody)
	var person people.PersonDetails
	alex.Do(t, http.MethodGet, "/api/people/"+jane.ID, nil).Decode(t, &person)
	if person.Version != 1 || len(person.Teams) != 1 {
		t.Errorf("after the impact check the person = %+v, want them unchanged", person)
	}
}

// Deactivate and Reactivate answer the person in their new status; both fields of the body are needed, an
// out-of-date version is refused and an unknown person answers 404 (FR-005, FR-007).
func TestSetPersonStatus_InactiveAndBack_FR005_FR007(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	jane := createPerson(t, alex, map[string]any{"name": "Jane Doe", "email": janesEmail, "teamIds": []string{"T-1001"}})
	path := "/api/people/" + jane.ID + "/status"

	var inactive, active people.PersonDetails
	alex.Do(t, http.MethodPost, path, map[string]any{"active": false, "version": 1}).Decode(t, &inactive)
	alex.Do(t, http.MethodPost, path, map[string]any{"active": true, "version": 2}).Decode(t, &active)
	incomplete := alex.Do(t, http.MethodPost, path, map[string]any{}, apitest.SkipRequestValidation())
	outOfDate := alex.Do(t, http.MethodPost, path, map[string]any{"active": false, "version": 1})
	missing := alex.Do(t, http.MethodPost, "/api/people/U-9999/status", map[string]any{"active": false, "version": 1})

	if inactive.Active || inactive.Version != 2 || len(inactive.Teams) != 1 ||
		!active.Active || active.Version != 3 || len(active.Teams) != 1 {
		t.Errorf("status = %+v then %+v, want inactive at version 2, then active at version 3, keeping the team",
			inactive, active)
	}
	assertAnswer(t, incomplete, http.StatusBadRequest, validationBody(
		`{"field":"active","message":"Choose active or inactive."},{"field":"version","message":"Reload the page and try again."}`))
	if outOfDate.Status != http.StatusConflict || !strings.Contains(string(outOfDate.Body), `"code":"version_conflict"`) {
		t.Errorf("out-of-date status = %d %s, want 409 version_conflict", outOfDate.Status, outOfDate.Body)
	}
	assertAnswer(t, missing, http.StatusNotFound, personGoneBody)
}

// Nobody can deactivate themselves: the API answers 409 not_allowed and they stay active and signed in
// (FR-006).
func TestDeactivateYourself_409NotAllowed_FR006(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodPost, "/api/people/U-1001/status", map[string]any{"active": false, "version": 1})

	assertAnswer(t, response, http.StatusConflict, deactivateYourselfBody)
	var me people.PersonDetails
	still := alex.Do(t, http.MethodGet, "/api/people/U-1001", nil)
	still.Decode(t, &me)
	if !me.Active || me.Version != 1 {
		t.Errorf("Alex after the refusal = %d %s, want active at version 1", still.Status, still.Body)
	}
}

// A new person is offered on the sign-in page and can sign in at once with the demo password (FR-003).
func TestNewPersonCanSignIn_FR003(t *testing.T) {
	server := apitest.New(t)
	createPerson(t, server.SignIn(t, alexsEmail), map[string]any{"name": "Jane Doe", "email": janesEmail})

	offered := slices.Contains(demoEmails(t, server), janesEmail)
	jane := server.SignIn(t, janesEmail)
	me := jane.Do(t, http.MethodGet, "/api/me", nil)

	if !offered {
		t.Errorf("the sign-in page does not offer %s", janesEmail)
	}
	if me.Status != http.StatusOK || !strings.Contains(string(me.Body), `"name":"Jane Doe"`) {
		t.Errorf("GET /api/me as the new person = %d %s", me.Status, me.Body)
	}
}

// A deactivated person is no longer offered on the sign-in page, cannot sign in, and their open session ends
// at their next request (FR-005).
func TestDeactivatedPersonCannotSignInAndSessionEnds_FR005(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	jane := createPerson(t, alex, map[string]any{"name": "Jane Doe", "email": janesEmail})
	janesSession := server.SignIn(t, janesEmail)

	setStatus(t, alex, jane.ID, false, 1)
	next := janesSession.Do(t, http.MethodGet, "/api/people", nil)
	signIn := server.Do(t, http.MethodPost, "/api/auth/sign-in", map[string]string{"email": janesEmail, "password": demoPassword(t, server)})

	assertAnswer(t, next, http.StatusUnauthorized, signInRequiredBody)
	assertAnswer(t, signIn, http.StatusUnauthorized, incorrectSignInBody)
	if slices.Contains(demoEmails(t, server), janesEmail) {
		t.Errorf("the sign-in page still offers %s", janesEmail)
	}
}

// A reactivated person can sign in again, with their teams as before (FR-007, SC-004).
func TestReactivatedPersonCanSignIn_FR007_SC004(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	jane := createPerson(t, alex, map[string]any{"name": "Jane Doe", "email": janesEmail, "teamIds": []string{"T-1001"}})
	setStatus(t, alex, jane.ID, false, 1)

	back := setStatus(t, alex, jane.ID, true, 2)
	me := server.SignIn(t, janesEmail).Do(t, http.MethodGet, "/api/me", nil)

	if !back.Active || len(back.Teams) != 1 || back.Teams[0].ID != "T-1001" {
		t.Errorf("reactivated person = %+v, want active in Lubricants Europe", back)
	}
	if me.Status != http.StatusOK {
		t.Errorf("GET /api/me after reactivation = %d %s", me.Status, me.Body)
	}
}

// Everyone sees people, and a person whose role allows "Add and change people and teams" manages them (FR-027
// as feature 014 rewrote it): Daniel, a Planner and Approver, is refused with 403 and nothing changes; given
// the Administrator role, he edits, deactivates and reactivates a person Alex created.
func TestPeopleManagedWithPeopleManage_FR027(t *testing.T) {
	server := apitest.New(t)
	jane := createPerson(t, server.SignIn(t, alexsEmail), map[string]any{"name": "Jane Doe", "email": janesEmail})
	daniel := server.SignIn(t, danielsEmail)
	change := map[string]any{"name": "Jane Doe", "email": janesEmail, "jobTitle": "Brand manager", "teamIds": []string{"T-1002"}, "version": 1}

	refused := daniel.Do(t, http.MethodPatch, "/api/people/"+jane.ID, change)
	read := daniel.Do(t, http.MethodGet, "/api/people/"+jane.ID, nil)
	assertAnswer(t, refused, http.StatusForbidden, `{"code":"forbidden","message":"Changing people and teams isn't part of `+
		`your roles (Approver, Planner). Ask an administrator if you need it.","fields":[]}`)
	var unchanged people.PersonDetails
	read.Decode(t, &unchanged)
	if read.Status != http.StatusOK || unchanged.Version != 1 || unchanged.JobTitle != "" {
		t.Errorf("after the refusal Daniel reads %d %s, want Jane unchanged at version 1", read.Status, read.Body)
	}

	server.GrantRole(t, danielsEmail, "administrator", "all")
	edited := daniel.Do(t, http.MethodPatch, "/api/people/"+jane.ID, change)
	inactive := setStatus(t, daniel, jane.ID, false, 2)
	active := setStatus(t, daniel, jane.ID, true, 3)

	var person people.PersonDetails
	edited.Decode(t, &person)
	if edited.Status != http.StatusOK || person.JobTitle != "Brand manager" || person.UpdatedBy.Name != "Daniel Okoro" {
		t.Errorf("Daniel's PATCH = %d %s", edited.Status, edited.Body)
	}
	if inactive.Active || !active.Active || active.UpdatedBy.Name != "Daniel Okoro" {
		t.Errorf("Daniel's status changes = %+v then %+v", inactive, active)
	}
}

// createPerson creates a person through POST /api/people and stops the test when that fails.
func createPerson(t *testing.T, client *apitest.Client, body map[string]any) people.PersonDetails {
	t.Helper()
	response := client.Do(t, http.MethodPost, "/api/people", body)
	if response.Status != http.StatusCreated {
		t.Fatalf("POST /api/people %v = %d %s", body, response.Status, response.Body)
	}
	var person people.PersonDetails
	response.Decode(t, &person)
	return person
}

// setStatus deactivates (active false) or reactivates the person id through POST /api/people/{id}/status
// from version, and stops the test when that fails.
func setStatus(t *testing.T, client *apitest.Client, id string, active bool, version int) people.PersonDetails {
	t.Helper()
	response := client.Do(t, http.MethodPost, "/api/people/"+id+"/status", map[string]any{"active": active, "version": version})
	if response.Status != http.StatusOK {
		t.Fatalf("POST /api/people/%s/status active=%t = %d %s", id, active, response.Status, response.Body)
	}
	var person people.PersonDetails
	response.Decode(t, &person)
	return person
}

// demoSignIn returns what the sign-in page shows: the people it offers and the demo password.
func demoSignIn(t *testing.T, server *apitest.Server) auth.DemoPeople {
	t.Helper()
	var demo auth.DemoPeople
	server.Do(t, http.MethodGet, "/api/auth/demo-people", nil).Decode(t, &demo)
	return demo
}

// demoEmails returns the e-mails of the people the sign-in page offers.
func demoEmails(t *testing.T, server *apitest.Server) []string {
	t.Helper()
	var emails []string
	for _, p := range demoSignIn(t, server).People {
		emails = append(emails, p.Email)
	}
	return emails
}

// demoPassword returns the demo password the sign-in page shows.
func demoPassword(t *testing.T, server *apitest.Server) string {
	t.Helper()
	return demoSignIn(t, server).Password
}
