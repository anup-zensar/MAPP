package approvals_test

import (
	"net/http"
	"slices"
	"testing"

	"northstar/internal/approvals"
	"northstar/internal/platform/apitest"
)

// The two seeded workflows, sorted by name, with their levels text and usage: China BC is used by the two
// Mobility activities, one waiting; Lubricants EU by the seven Lubricants activities, one waiting
// (FR-005, FR-037).
func TestListApprovalWorkflows_SeededThree_FR005_FR037(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodGet, "/api/approval-workflows", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var page approvals.WorkflowPage
	response.Decode(t, &page)
	if page.Total != 3 || len(page.Items) != 3 {
		t.Fatalf("page = %+v, want the 3 seeded workflows", page)
	}
	china, lubricants := page.Items[0], page.Items[1]
	if standard := page.Items[2]; standard.ID != "WF-1003" || standard.Name != "Standard marketing approval" ||
		*standard.ReminderDays != 3 || *standard.EscalationDays != 5 || standard.UsedByCount != 0 {
		t.Errorf("third = %+v, want 017's Standard marketing approval, unused", standard)
	}
	if china.ID != "WF-1002" || china.LevelsText != "Level 1 · Carlos Mendes → Level 2 · Alex Morgan" || !china.Active ||
		china.UsedByCount != 2 || china.WaitingCount != 1 || len(china.Warnings) != 0 {
		t.Errorf("first = %+v, want China BC approval workflow used by 2, 1 waiting", china)
	}
	if lubricants.ID != "WF-1001" || lubricants.LevelsText != "Level 1 · Daniel Okoro" || lubricants.UsedByCount != 7 ||
		lubricants.WaitingCount != 1 {
		t.Errorf("second = %+v, want Lubricants EU approval used by 7, 1 waiting", lubricants)
	}
}

// A new workflow answers 201 with its id, levels and labels (FR-002).
func TestCreateApprovalWorkflow_201_FR002(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodPost, "/api/approval-workflows", map[string]any{
		"name": "Mobility India approval", "description": "India first.",
		"levels": []map[string]any{{"name": "Regional", "approverIds": []string{"U-1005"}}, {"name": "", "approverIds": []string{"U-1001"}}},
	})

	view := decodeWorkflow(t, response, http.StatusCreated)
	if view.ID != "WF-1004" || view.LevelsText != "Regional · Carlos Mendes → Level 2 · Alex Morgan" || !view.Active ||
		view.Version != 1 || view.CreatedBy.Name != "Alex Morgan" || view.Levels[1].Label != "Level 2" {
		t.Errorf("workflow = %+v, want WF-1004 with the two levels, created by Alex Morgan", view)
	}
}

// Every problem of a new workflow comes back at once, each on its field (FR-003).
func TestCreateApprovalWorkflow_AllProblems400_FR003(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodPost, "/api/approval-workflows", map[string]any{
		"name": " lubricants EU APPROVAL ",
		"levels": []map[string]any{
			{"name": "", "approverIds": []string{}},
			{"name": "", "approverIds": []string{"U-1003", "U-1003", "U-9999"}},
		},
	})

	assertError(t, response, http.StatusBadRequest, "validation_failed", "Some fields need your attention.")
	var refusal struct {
		Fields []struct{ Field, Message string } `json:"fields"`
	}
	response.Decode(t, &refusal)
	var got []string
	for _, f := range refusal.Fields {
		got = append(got, f.Field+": "+f.Message)
	}
	want := []string{
		"name: There is already an approval workflow called 'Lubricants EU approval'. Choose another name.",
		"levels.1.approverIds: Level 1: choose at least one approver.",
		"levels.2.approverIds: Level 2: Daniel Okoro is listed twice.",
		"levels.2.approverIds: Level 2: choose approvers from the list.",
	}
	if !slices.Equal(got, want) {
		t.Errorf("problems = %q\nwant %q", got, want)
	}
}

// The details of China BC list the Mobility activities that use it, by name, and which one waits for
// approval (FR-005).
func TestGetApprovalWorkflow_UsedBy_FR005(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodGet, "/api/approval-workflows/WF-1002", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var details approvals.WorkflowDetails
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

// A workflow that is gone answers 404 with its message.
func TestGetApprovalWorkflow_404(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodGet, "/api/approval-workflows/WF-9999", nil)

	assertError(t, response, http.StatusNotFound, "not_found", "This approval workflow no longer exists — someone may have deleted it.")
}

// Changing China BC's levels applies to later submissions only: the Mobility activity waiting at level 2
// keeps the levels it was submitted with (FR-006).
func TestUpdateApprovalWorkflow_OpenRoundUnchanged_FR006(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodPatch, "/api/approval-workflows/WF-1002", map[string]any{
		"levels": []map[string]any{{"name": "", "approverIds": []string{"U-1003"}, "backupApproverId": "U-1001"}}, "version": 1,
	})

	view := decodeWorkflow(t, response, http.StatusOK)
	if view.LevelsText != "Level 1 · Daniel Okoro" || view.Version != 2 || view.WaitingCount != 1 {
		t.Errorf("workflow = %+v, want the new level in version 2, 1 waiting", view)
	}
	approval := decodeApproval(t, alex.Do(t, http.MethodGet, "/api/activities/"+fleetCardLaunch+"/approval", nil))
	round := approval.Rounds[0]
	if round.LevelCount != 2 || round.CurrentLevel != 2 || round.Levels[1].Approvers[0].Name != "Alex Morgan" || !approval.Viewer.IsCurrentApprover {
		t.Errorf("round = %+v, want still waiting for Alex Morgan at level 2 of 2", round)
	}
}

// An update from an old version is refused with version_conflict naming who changed the workflow (FR-003).
func TestUpdateApprovalWorkflow_409(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	priya := server.SignIn(t, priyasEmail)
	renamed := alex.Do(t, http.MethodPatch, "/api/approval-workflows/WF-1001", map[string]any{"name": "Lubricants approval", "version": 1})
	decodeWorkflow(t, renamed, http.StatusOK)

	response := priya.Do(t, http.MethodPatch, "/api/approval-workflows/WF-1001", map[string]any{"name": "Other", "version": 1})

	assertError(t, response, http.StatusConflict, "version_conflict", "Not saved — Alex Morgan changed this approval workflow")
}

// A workflow that activities use cannot be deleted: 409 in_use with the number of activities (FR-008).
func TestDeleteApprovalWorkflow_InUse409_FR008(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodDelete, "/api/approval-workflows/WF-1002", nil)

	assertError(t, response, http.StatusConflict, "in_use", "Used by 2 activities — set it inactive instead.")
}

// An unused workflow moves to the recycle bin; Undo brings it back (FR-008).
func TestDeleteApprovalWorkflow_UnusedUndo_FR008(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created := decodeWorkflow(t, alex.Do(t, http.MethodPost, "/api/approval-workflows", map[string]any{
		"name": "Temp flow", "levels": []map[string]any{{"name": "", "approverIds": []string{"U-1005"}}},
	}), http.StatusCreated)
	path := "/api/approval-workflows/" + created.ID

	deleted := alex.Do(t, http.MethodDelete, path, nil)
	gone := alex.Do(t, http.MethodGet, path, nil)

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
	if undo := alex.Do(t, http.MethodPost, "/api/recycle-bin/"+result.RecycleBinEntry.ID+"/undo", nil); undo.Status != http.StatusOK {
		t.Fatalf("undo status = %d %s, want 200", undo.Status, undo.Body)
	}
	if back := alex.Do(t, http.MethodGet, path, nil); back.Status != http.StatusOK {
		t.Errorf("after undo: status %d, want the workflow back", back.Status)
	}
}

// Set inactive and Set active, each from the version the person saw: the Mobility activity waiting with
// China BC keeps waiting and shows the workflow as inactive; an old version is refused (FR-007).
func TestDeactivateActivate_FR007(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	path := "/api/approval-workflows/WF-1002"

	inactive := decodeWorkflow(t, alex.Do(t, http.MethodPost, path+"/deactivate", map[string]any{"version": 1}), http.StatusOK)
	approval := decodeApproval(t, alex.Do(t, http.MethodGet, "/api/activities/"+fleetCardLaunch+"/approval", nil))
	stale := alex.Do(t, http.MethodPost, path+"/activate", map[string]any{"version": 1})
	active := decodeWorkflow(t, alex.Do(t, http.MethodPost, path+"/activate", map[string]any{"version": 2}), http.StatusOK)

	if inactive.Active || inactive.Version != 2 || inactive.WaitingCount != 1 {
		t.Errorf("deactivated = %+v, want inactive in version 2, 1 waiting", inactive)
	}
	if approval.Rounds[0].Outcome != approvals.OutcomeOpen || approval.Workflow == nil || approval.Workflow.Active {
		t.Errorf("approval = %+v, want the round still open and the workflow shown inactive", approval)
	}
	assertError(t, stale, http.StatusConflict, "version_conflict", "Not saved — Alex Morgan changed this approval workflow")
	if !active.Active || active.Version != 3 {
		t.Errorf("activated = %+v, want active in version 3", active)
	}
}

// decodeWorkflow reads an ApprovalWorkflow answered with status.
func decodeWorkflow(t *testing.T, response *apitest.Response, status int) approvals.WorkflowView {
	t.Helper()
	if response.Status != status {
		t.Fatalf("status = %d %s, want %d", response.Status, response.Body, status)
	}
	var view approvals.WorkflowView
	response.Decode(t, &view)
	return view
}
