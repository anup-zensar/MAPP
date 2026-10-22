package activities_test

import (
	"net/http"
	"testing"
	"time"

	"mapp/internal/activities"
	"mapp/internal/platform/apitest"
)

// The destinations of Launch film are the four initiatives of its plan, its own first, with why it can't go
// to Retail events (FR-029, FR-030).
func TestActivityDestinations_Contract_FR029(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/activities/A-1001/destinations", nil)
	gone := priya.Do(t, http.MethodGet, "/api/activities/A-9999/destinations", nil)

	assertStatus(t, response, http.StatusOK)
	var page activities.DestinationPage
	response.Decode(t, &page)
	if page.Total != 4 || len(page.Items) != 4 || !page.Items[0].Current || page.Items[0].Initiative.ID != "I-1001" ||
		page.Items[3].Initiative.Name != "Retail events" || page.Items[3].MoveProblem == nil || page.Items[3].CopyProblem == nil ||
		page.Items[1].MoveProblem != nil {
		t.Errorf("destinations = %s", response.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, activityGoneBody)
}

// Moving Launch film to Print & outdoor answers it there with a new version; Retail events is refused with
// the dates message, and a move from an out-of-date version with version_conflict (FR-029).
func TestMoveActivity_Contract_FR029(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, "daniel.okoro@shell.com")
	server.Advance(time.Hour)

	refused := priya.Do(t, http.MethodPost, "/api/activities/A-1001/move", map[string]any{"initiativeId": "I-1004", "version": 1})
	moved := priya.Do(t, http.MethodPost, "/api/activities/A-1001/move", map[string]any{"initiativeId": "I-1002", "version": 1})
	outOfDate := daniel.Do(t, http.MethodPost, "/api/activities/A-1001/move", map[string]any{"initiativeId": "I-1003", "version": 1})

	assertAnswer(t, refused, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"initiativeId","message":"'Launch film' runs 01 Mar – 31 May 2026, but 'Retail events' runs `+
		`01 Jun – 31 Dec 2026. Change the activity's dates first, or pick another initiative."}]}`)
	assertStatus(t, moved, http.StatusOK)
	var view activities.View
	moved.Decode(t, &view)
	if view.Initiative.ID != "I-1002" || view.Version != 2 || view.Status != activities.StatusCreated {
		t.Errorf("moved = %s", moved.Body)
	}
	assertAnswer(t, outOfDate, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Priya Raman changed this activity",`+
		`"fields":[],"conflict":{"changedBy":"Priya Raman","changedAt":"2026-03-12T10:00:00Z"}}`)
}

// Copying the Approved Social always-on answers 201 with "Social always-on (copy)", Created, with copies of
// its 2 line items (FR-030, US5 scenario 2).
func TestCopyActivity_Contract_FR030(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/activities/A-1002/copy", map[string]any{"initiativeId": "I-1001"})
	refused := priya.Do(t, http.MethodPost, "/api/activities/A-1002/copy", map[string]any{"initiativeId": "I-1005"})

	assertStatus(t, response, http.StatusCreated)
	var copied activities.View
	response.Decode(t, &copied)
	if copied.ID == "A-1002" || copied.Name != "Social always-on (copy)" || copied.Status != activities.StatusCreated ||
		copied.LineItems.Count != 2 || copied.Initiative.ID != "I-1001" {
		t.Errorf("copy = %s", response.Body)
	}
	assertAnswer(t, refused, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"initiativeId","message":"Choose another initiative of this plan."}]}`)
	var original activities.View
	priya.Do(t, http.MethodGet, "/api/activities/A-1002", nil).Decode(t, &original)
	if original.Status != activities.StatusApproved {
		t.Errorf("original is %s, want it still Approved", original.Status)
	}
}
