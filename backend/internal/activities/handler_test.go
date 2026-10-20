package activities_test

import (
	"net/http"
	"slices"
	"testing"

	"northstar/internal/activities"
	"northstar/internal/history"
	"northstar/internal/platform/apitest"
)

// The contract tests of the activity operations: the real API on the demo data, every request and answer
// checked against backend/api/openapi.yaml by apitest. They check the parts of each answer the operation is
// about; the rules themselves are proven by the service tests.

const priyasEmail = "priya.raman@northstar.demo"

// The not-found answers.
const (
	activityGoneBody   = `{"code":"not_found","message":"This activity no longer exists — someone may have deleted it.","fields":[]}`
	initiativeGoneBody = `{"code":"not_found","message":"This initiative no longer exists — someone may have deleted it.","fields":[]}`
	planGoneBody       = `{"code":"not_found","message":"This plan no longer exists — someone may have deleted it.","fields":[]}`
)

// The new-activity form answers the context, choices and defaults of the initiative (FR-001); a gone
// initiative answers not found.
func TestNewActivityForm_Contract_FR001(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/initiatives/I-1001/activity-form", nil)

	var form activities.FormView
	assertStatus(t, response, http.StatusOK)
	response.Decode(t, &form)
	if form.Context.Initiative.Name != "Digital & social" || form.Defaults == nil || form.Defaults.Owner.Name != "Priya Raman" ||
		len(form.ActivityTypes) != 5 || len(form.ApprovalWorkflows) != 2 || len(form.Attributes) != 20 {
		t.Errorf("form = %+v", form)
	}
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/initiatives/I-9999/activity-form", nil), http.StatusNotFound, initiativeGoneBody)
}

// Creating an activity answers 201 with the new activity, Created (FR-004); every field problem is listed
// at once (FR-006, FR-009); a gone initiative answers not found.
func TestCreateActivity_Contract_FR004(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	created := priya.Do(t, http.MethodPost, "/api/initiatives/I-1001/activities", map[string]any{
		"name": "Launch video", "activityTypeId": "AT-1002", "plannedCost": map[string]string{"amount": "25000.00", "currency": "USD"},
	})
	refused := priya.Do(t, http.MethodPost, "/api/initiatives/I-1001/activities", map[string]any{
		"name": "launch film", "activityTypeId": "AT-1006", "startDate": "2026-05-01", "endDate": "2026-04-01",
	})
	gone := priya.Do(t, http.MethodPost, "/api/initiatives/I-9999/activities", map[string]any{"name": "Launch video", "activityTypeId": "AT-1002"})

	var view activities.View
	assertStatus(t, created, http.StatusCreated)
	created.Decode(t, &view)
	// A-1011 is the activity of the demo history's deleted initiative, in the recycle bin (011 T019).
	if view.ID != "A-1012" || view.Status != activities.StatusCreated || view.PlannedCost.Format() != "25,000.00 USD" {
		t.Errorf("created = %+v", view)
	}
	assertAnswer(t, refused, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.","fields":[`+
		`{"field":"name","message":"'Digital `+"\\u0026"+` social' already has an activity called 'Launch film'."},`+
		`{"field":"activityTypeId","message":"Choose an activity type that the model 'Lubricants 1.0' allows."},`+
		`{"field":"endDate","message":"End date must be on or after the start date (01 May 2026)."}]}`)
	assertAnswer(t, gone, http.StatusNotFound, initiativeGoneBody)
}

// The plan's activities are its plan-tree rows, ordered by start date, then name, without dates last
// (FR-033, D-111); a gone plan answers not found.
func TestListPlanActivities_Contract_FR033(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all") // 014: every market's plans

	response := priya.Do(t, http.MethodGet, "/api/plans/P-1002/activities", nil)

	var page activities.SummaryPage
	assertStatus(t, response, http.StatusOK)
	response.Decode(t, &page)
	var rows []string
	for _, item := range page.Items {
		rows = append(rows, item.ID+" "+string(item.DisplayStatus))
	}
	if want := []string{"A-1008 created", "A-1009 submitted", "A-1010 created"}; !slices.Equal(rows, want) || page.Total != 3 {
		t.Errorf("rows = %q (total %d), want %q", rows, page.Total, want)
	}
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/plans/P-9999/activities", nil), http.StatusNotFound, planGoneBody)
}

// An activity answers with its status, completeness and ready-to-submit check (FR-019 – FR-023): A-1001
// "Launch film" has three needed fields empty; a gone activity answers not found.
func TestGetActivity_Contract_FR019(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/activities/A-1001", nil)

	var view activities.View
	assertStatus(t, response, http.StatusOK)
	response.Decode(t, &view)
	var messages []string
	for _, p := range view.Readiness.Activity {
		messages = append(messages, p.Message)
	}
	want := []string{"Company code — needed to submit", "Cost centre — needed to submit", "PIR — needed to submit"}
	if view.Completeness.MissingFields != 3 || !slices.Equal(messages, want) || view.DisplayStatus != activities.DisplayCreated ||
		view.Goals.Revenue.Format() != "2,500,000.00 USD" || view.Plan.Name != "FY2026 Lubricants Europe" {
		t.Errorf("A-1001 = %+v", view)
	}
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/activities/A-9999", nil), http.StatusNotFound, activityGoneBody)
}

// The edit form has no defaults and leaves the activity's own name out of the names in use (FR-013,
// FR-026); a gone activity answers not found.
func TestEditActivityForm_Contract_FR013(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/activities/A-1001/form", nil)

	var form activities.FormView
	assertStatus(t, response, http.StatusOK)
	response.Decode(t, &form)
	if form.Defaults != nil || !slices.Equal(form.ExistingNames, []string{"Social always-on"}) {
		t.Errorf("form defaults %+v, names %q", form.Defaults, form.ExistingNames)
	}
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/activities/A-9999/form", nil), http.StatusNotFound, activityGoneBody)
}

// Saving an edit answers the activity after the save (FR-026); a re-approval change on a Submitted
// activity needs restartApproval (FR-027, D-106); a save from an out-of-date version is refused with
// version_conflict.
func TestUpdateActivity_Contract_FR026(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, "daniel.okoro@northstar.demo")
	var a1004 activities.View
	priya.Do(t, http.MethodGet, "/api/activities/A-1004", nil).Decode(t, &a1004)
	form := formOf(a1004)
	form.PlannedCost = usdInput("35000.00")

	refused := priya.Do(t, http.MethodPatch, "/api/activities/A-1004", form)
	form.RestartApproval = true
	saved := priya.Do(t, http.MethodPatch, "/api/activities/A-1004", form)
	outOfDate := daniel.Do(t, http.MethodPatch, "/api/activities/A-1004", form)

	assertAnswer(t, refused, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"restartApproval","message":"This activity will need approval again — confirm the change to save it."}]}`)
	var view activities.View
	assertStatus(t, saved, http.StatusOK)
	saved.Decode(t, &view)
	if view.Status != activities.StatusCreated || view.Version != 2 || view.PlannedCost.Format() != "35,000.00 USD" {
		t.Errorf("saved = status %s, version %d, cost %v", view.Status, view.Version, view.PlannedCost)
	}
	assertAnswer(t, outOfDate, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Priya Raman changed this activity",`+
		`"fields":[],"conflict":{"changedBy":"Priya Raman","changedAt":"2026-03-12T09:00:00Z"}}`)
	assertAnswer(t, priya.Do(t, http.MethodPatch, "/api/activities/A-9999", form), http.StatusNotFound, activityGoneBody)
}

// Deleting an activity answers its recycle-bin entry, and Undo brings it back exactly (FR-031).
func TestDeleteActivity_ThenUndo_Contract_FR031(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	before := priya.Do(t, http.MethodGet, "/api/activities/A-1002", nil)

	deleted := priya.Do(t, http.MethodDelete, "/api/activities/A-1002", nil)
	gone := priya.Do(t, http.MethodGet, "/api/activities/A-1002", nil)

	var result history.DeleteResult
	assertStatus(t, deleted, http.StatusOK)
	deleted.Decode(t, &result)
	entry := result.RecycleBinEntry
	if entry.RecordType != "activity" || entry.RecordID != "A-1002" || entry.RecordName != "Social always-on" ||
		entry.Parent == nil || entry.Parent.ID != "I-1001" {
		t.Errorf("recycle-bin entry = %+v", entry)
	}
	assertAnswer(t, gone, http.StatusNotFound, activityGoneBody)
	undone := priya.Do(t, http.MethodPost, "/api/recycle-bin/"+entry.ID+"/undo", nil)
	assertStatus(t, undone, http.StatusOK)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/activities/A-1002", nil), http.StatusOK, string(before.Body))
	assertAnswer(t, priya.Do(t, http.MethodDelete, "/api/activities/A-9999", nil), http.StatusNotFound, activityGoneBody)
}

// assertStatus checks the status of an answer.
func assertStatus(t *testing.T, r *apitest.Response, status int) {
	t.Helper()
	if r.Status != status {
		t.Fatalf("status = %d %s, want %d", r.Status, r.Body, status)
	}
}

// assertAnswer checks the status and the exact body of an answer.
func assertAnswer(t *testing.T, r *apitest.Response, status int, body string) {
	t.Helper()
	if r.Status != status || string(r.Body) != body {
		t.Errorf("answer = %d %s\nwant     %d %s", r.Status, r.Body, status, body)
	}
}
