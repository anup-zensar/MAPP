package approval_test

import (
	"net/http"
	"slices"
	"testing"

	"mapp/internal/approval"
	"mapp/internal/platform/apitest"
)

// The two seeded workflows, sorted by name, with their levels text and usage: China BC is used by the two
// Mobility activities, one waiting; Lubricants EU by the seven Lubricants activities, one waiting
// (FR-005, FR-037).
func TestListApprovalWorkflows_SeededThree_FR005_FR037(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodGet, "/api/v1/approval-workflows", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var page approval.WorkflowPage
	response.Decode(t, &page)
	if page.Total != 5 || len(page.Items) != 5 {
		t.Fatalf("page = %+v, want the 5 seeded workflows (032 adds two for purchase requests)", page)
	}
	china, lubricants := page.Items[0], page.Items[1]
	if standard := page.Items[4]; standard.ID != "WF-1003" || standard.Name != "Standard marketing approval" ||
		*standard.ReminderDays != 3 || *standard.EscalationDays != 5 || standard.UsedByCount != 0 {
		t.Errorf("third = %+v, want 017's Standard marketing approval, unused", standard)
	}
	if china.ID != "WF-1002" || china.LevelsText != "Level 1 · Carlos Mendes → Level 2 · Alex Morgan" || !china.Active ||
		china.UsedByCount != 2 || china.WaitingCount != 1 || len(china.Warnings) != 0 {
		t.Errorf("first = %+v, want China BC approval workflow used by 2, 1 waiting", china)
	}
	if lubricants.ID != "WF-1001" || lubricants.LevelsText != "Level 1 · Daniel Okoro" || lubricants.UsedByCount != 9 ||
		lubricants.WaitingCount != 1 {
		t.Errorf("second = %+v, want Lubricants EU approval used by 9 (2 in the GBP plan of 016), 1 waiting", lubricants)
	}
}

// The details of China BC list the Mobility activities that use it, by name, and which one waits for
// approval (FR-005).
func TestGetApprovalWorkflow_UsedBy_FR005(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodGet, "/api/v1/approval-workflows/WF-1002", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var details approval.WorkflowDetails
	response.Decode(t, &details)
	var got []string
	for _, a := range details.Activities {
		line := a.Activity.ID + " " + a.Activity.Name
		if a.Waiting {
			line += " (waiting)"
		}
		got = append(got, line)
	}
	want := []string{fleetCardLaunch + " Fleet card launch (waiting)", "A-1008 Grand opening Shanghai"}
	if !slices.Equal(got, want) || details.Workflow.UsedByCount != 2 || len(details.Activities[0].Path) != 3 {
		t.Errorf("activities = %q, want %q", got, want)
	}
}

func TestUpdateApprovalWorkflow_409(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "administrator", "all")
	renamed := alex.Do(t, http.MethodPatch, "/api/v1/approval-workflows/WF-1001", map[string]any{"name": "Lubricants approval", "version": 1})
	decodeWorkflow(t, renamed, http.StatusOK)

	response := priya.Do(t, http.MethodPatch, "/api/v1/approval-workflows/WF-1001", map[string]any{"name": "Other", "version": 1})

	assertError(t, response, http.StatusConflict, "version_conflict", "Not saved — Alex Morgan changed this approval workflow")
}

func TestDeleteApprovalWorkflow_UnusedUndo_FR008(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created := decodeWorkflow(t, alex.Do(t, http.MethodPost, "/api/v1/approval-workflows", map[string]any{
		"name": "Temp flow", "levels": []map[string]any{{"name": "", "approverIds": []string{"U-1005"}}},
	}), http.StatusCreated)
	path := "/api/v1/approval-workflows/" + created.ID

	deleted := alex.Do(t, http.MethodDelete, path, nil)
	gone := alex.Do(t, http.MethodGet, path, nil)
	inUse := alex.Do(t, http.MethodDelete, "/api/v1/approval-workflows/WF-1002", nil)

	if deleted.Status != http.StatusOK {
		t.Fatalf("delete status = %d %s, want 200", deleted.Status, deleted.Body)
	}
	var result struct {
		RecycleBinEntry struct {
			ID       string `json:"id"`
			RecordID string `json:"recordId"`
		} `json:"recycleBinEntry"`
	}
	deleted.Decode(t, &result)
	if result.RecycleBinEntry.RecordID != created.ID {
		t.Errorf("bin entry = %+v, want %s", result.RecycleBinEntry, created.ID)
	}
	assertError(t, gone, http.StatusNotFound, "not_found", "This approval workflow no longer exists — someone may have deleted it.")
	assertError(t, inUse, http.StatusConflict, "in_use", "Used by 2 activities — set it inactive instead.")
	if undo := alex.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+result.RecycleBinEntry.ID+"/undo", nil); undo.Status != http.StatusOK {
		t.Fatalf("undo status = %d %s, want 200", undo.Status, undo.Body)
	}
	if back := alex.Do(t, http.MethodGet, path, nil); back.Status != http.StatusOK {
		t.Errorf("after undo: status %d, want the workflow back", back.Status)
	}
}

// Set inactive and Set active, each from the version the person saw: the Mobility activity waiting with
// China BC keeps waiting and shows the workflow as inactive; an old version is refused (FR-007).
func TestDeactivateActivate_FR007(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	path := "/api/v1/approval-workflows/WF-1002"

	inactive := decodeWorkflow(t, alex.Do(t, http.MethodPost, path+"/deactivate", map[string]any{"version": 1}), http.StatusOK)
	appr := decodeApproval(t, alex.Do(t, http.MethodGet, "/api/v1/activities/"+fleetCardLaunch+"/approval", nil))
	stale := alex.Do(t, http.MethodPost, path+"/activate", map[string]any{"version": 1})
	active := decodeWorkflow(t, alex.Do(t, http.MethodPost, path+"/activate", map[string]any{"version": 2}), http.StatusOK)

	if inactive.Active || inactive.Version != 2 || inactive.WaitingCount != 1 {
		t.Errorf("deactivated = %+v, want inactive in version 2, 1 waiting", inactive)
	}
	if appr.Rounds[0].Outcome != approval.OutcomeOpen || appr.Workflow == nil || appr.Workflow.Active {
		t.Errorf("approval = %+v, want the round still open and the workflow shown inactive", appr)
	}
	assertError(t, stale, http.StatusConflict, "version_conflict", "Not saved — Alex Morgan changed this approval workflow")
	if !active.Active || active.Version != 3 {
		t.Errorf("activated = %+v, want active in version 3", active)
	}
}

func decodeWorkflow(t *testing.T, response *apitest.Response, status int) approval.WorkflowView {
	t.Helper()
	if response.Status != status {
		t.Fatalf("status = %d %s, want %d", response.Status, response.Body, status)
	}
	var view approval.WorkflowView
	response.Decode(t, &view)
	return view
}
