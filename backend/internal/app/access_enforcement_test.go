package app_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
)

// Priya's totals are the Lubricants plan's.
func TestListsAndTotals_OnlyWhatThePersonSees_SC004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	if got := itemIDs(t, priya.Do(t, http.MethodGet, "/api/v1/home/plans?scope=everything", nil)); !slices.Equal(got, []string{"P-1004", "P-1001"}) {
		t.Errorf("Home plans (everything) = %q, want the Lubricants Europe team's plans only", got)
	}
	var grid struct {
		Rows []struct {
			Level, PlanID string
		}
		Summary struct{ Activities int }
	}
	priya.Do(t, http.MethodGet, "/api/v1/all-work?year=2026", nil).Decode(t, &grid)
	for _, row := range grid.Rows {
		if row.PlanID != "P-1001" {
			t.Errorf("All work shows a row of %s", row.PlanID)
		}
	}
	if grid.Summary.Activities != 7 {
		t.Errorf("All work counts %d activities, want the 7 of the Lubricants plan", grid.Summary.Activities)
	}
	export := priya.Do(t, http.MethodGet, "/api/v1/all-work/export?year=2026&format=csv", nil, apitest.SkipValidation())
	if export.Status != http.StatusOK || strings.Contains(string(export.Body), "Mobility") {
		t.Errorf("the export = %d, mentions Mobility: %v", export.Status, strings.Contains(string(export.Body), "Mobility"))
	}

	var open struct {
		Items []struct{ Activity struct{ ID string } }
	}
	priya.Do(t, http.MethodGet, "/api/v1/approvals?tab=open", nil).Decode(t, &open)
	var openIDs []string
	for _, row := range open.Items {
		openIDs = append(openIDs, row.Activity.ID)
	}
	if !slices.Equal(openIDs, []string{"A-1004"}) {
		t.Errorf("All open = %q, want the Lubricants activity A-1004 only (not Mobility's A-1009)", openIDs)
	}
	var counts struct{ AllOpen int }
	priya.Do(t, http.MethodGet, "/api/v1/approvals/summary", nil).Decode(t, &counts)
	if counts.AllOpen != 1 {
		t.Errorf("All open count = %d, want 1", counts.AllOpen)
	}
}

// Settings changes need the whole activity log; restoring needs the level's delete permission (D-238).
func TestLogAndRecycleBin_Filtered_FR007_FR038(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory()) // only the entries of this test
	alex, priya, mei := server.SignIn(t, alexsEmail), server.SignIn(t, priyasEmail), server.SignIn(t, meisEmail)
	var deleted struct{ RecycleBinEntry struct{ ID string } }
	mei.Do(t, http.MethodDelete, "/api/v1/initiatives/I-1006", nil).Decode(t, &deleted)
	alex.Do(t, http.MethodPatch, "/api/v1/models/MOD-1002", map[string]any{"name": "Mobility 1.1", "version": 1}, apitest.SkipRequestValidation())

	var log struct {
		Items []struct{ RecordType, RecordID string }
	}
	priya.Do(t, http.MethodGet, "/api/v1/activity-log", nil).Decode(t, &log)
	for _, e := range log.Items {
		if e.RecordID == "I-1006" || e.RecordType == "model" {
			t.Errorf("Priya's log shows %s %s", e.RecordType, e.RecordID)
		}
	}
	var alexs struct{ Total int }
	alex.Do(t, http.MethodGet, "/api/v1/activity-log", nil).Decode(t, &alexs)
	if alexs.Total < 2 {
		t.Errorf("Alex's log has %d entries, want the delete and the model change", alexs.Total)
	}

	if got := itemIDs(t, priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil)); len(got) != 0 {
		t.Errorf("Priya's recycle bin = %q, want empty", got)
	}
	if got := itemIDs(t, mei.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil)); !slices.Equal(got, []string{deleted.RecycleBinEntry.ID}) {
		t.Errorf("Mei's recycle bin = %q, want her delete", got)
	}
	if restored := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+deleted.RecycleBinEntry.ID+"/restore", nil); restored.Status != http.StatusNotFound ||
		!strings.Contains(string(restored.Body), "This item is not in the recycle bin.") {
		t.Errorf("Priya restoring Mei's initiative = %d %s, want 404 as for a missing entry", restored.Status, restored.Body)
	}
	server.GrantRole(t, sarasEmail, "viewer", "all")
	refused(t, "Sara restoring it", server.SignIn(t, sarasEmail).Do(t, http.MethodPost, "/api/v1/recycle-bin/"+deleted.RecycleBinEntry.ID+"/restore", nil),
		"Deleting campaigns and initiatives isn't part of your role (Viewer). Ask an administrator if you need it.")
	if restored := mei.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+deleted.RecycleBinEntry.ID+"/restore", nil); restored.Status != http.StatusOK {
		t.Errorf("Mei restoring = %d %s, want 200", restored.Status, restored.Body)
	}
}

// Nobody approves an activity they own or submitted while the rule is on (messages 5, 6a, 6b).
func TestApprovals_WhoMayDecide_FR025_FR026_038_FR056(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex, priya, daniel := server.SignIn(t, alexsEmail), server.SignIn(t, priyasEmail), server.SignIn(t, danielsEmail)
	decision := func(level int) map[string]any { return map[string]any{"roundId": "AR-1004", "level": level} }
	type actions struct {
		Actions struct {
			Approve, Decline, Recall struct {
				Available bool
				Reason    string
			}
		}
	}

	// Priya submitted A-1004 and owns it; Daniel is the approver of its level 1. As a Planner she holds neither
	// approvals permission, which is said first.
	refused(t, "Priya approving her own activity", priya.Do(t, http.MethodPost, "/api/v1/activities/A-1004/approval/approve", decision(1)),
		"Approving activities isn't part of your role (Planner). Ask an administrator if you need it.")
	var priyas actions
	priya.Do(t, http.MethodGet, "/api/v1/activities/A-1004/approval", nil).Decode(t, &priyas)
	if priyas.Actions.Approve.Available || priyas.Actions.Approve.Reason != "" || !priyas.Actions.Recall.Available {
		t.Errorf("Priya's actions = %+v, want Approve hidden and Recall offered", priyas.Actions)
	}

	// As an administrator Priya could decide for Daniel, but the segregation of duties stops her (message 6a).
	server.GrantRole(t, priyasEmail, "administrator", "all")
	ownMessage := "You own this activity, so someone else must approve it."
	priya.Do(t, http.MethodGet, "/api/v1/activities/A-1004/approval", nil).Decode(t, &priyas)
	if priyas.Actions.Approve.Available || priyas.Actions.Approve.Reason != ownMessage {
		t.Errorf("Priya's approve = %+v, want unavailable: %s", priyas.Actions.Approve, ownMessage)
	}
	refused(t, "Priya approving on behalf", priya.Do(t, http.MethodPost, "/api/v1/activities/A-1004/approval/approve", decision(1)), ownMessage)

	// With the rule off, she may, on behalf of the approvers; Daniel, its approver, always may.
	rules := alex.Do(t, http.MethodPatch, "/api/v1/access-rules", map[string]any{"noSelfApproval": false, "version": 1})
	if rules.Status != http.StatusOK {
		t.Fatalf("turning the rule off = %d %s", rules.Status, rules.Body)
	}
	var danielsView actions
	daniel.Do(t, http.MethodGet, "/api/v1/activities/A-1004/approval", nil).Decode(t, &danielsView)
	if !danielsView.Actions.Approve.Available {
		t.Errorf("Daniel's approve = %+v, want available", danielsView.Actions.Approve)
	}
	if approved := priya.Do(t, http.MethodPost, "/api/v1/activities/A-1004/approval/approve", decision(1)); approved.Status != http.StatusOK {
		t.Errorf("Priya approving on behalf with the rule off = %d %s, want 200", approved.Status, approved.Body)
	}
}

// Recalling is for the owner, the submitter, or people who decide on behalf of the approvers.
func TestApprovals_SubmitAndRecall_FR027(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	sara, daniel, alex := server.SignIn(t, sarasEmail), server.SignIn(t, danielsEmail), server.SignIn(t, alexsEmail)

	refused(t, "Sara submitting", sara.Do(t, http.MethodGet, "/api/v1/activities/A-1001/approval/submission", nil),
		"Submitting activities isn't part of your role (Viewer). Ask an administrator if you need it.")
	refused(t, "Daniel recalling Priya's submission",
		daniel.Do(t, http.MethodPost, "/api/v1/activities/A-1004/approval/recall", map[string]any{"roundId": "AR-1004"}),
		"Only the activity's owner or the person who submitted it can recall this activity.")
	if recalled := alex.Do(t, http.MethodPost, "/api/v1/activities/A-1004/approval/recall", map[string]any{"roundId": "AR-1004"}); recalled.Status != http.StatusOK {
		t.Errorf("Alex recalling on behalf = %d %s, want 200", recalled.Status, recalled.Body)
	}
}

// Approvers who no longer hold the permission are flagged (message 8).
func TestWorkflowApprovers_MustApprove_FR028(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	created := alex.Do(t, http.MethodPost, "/api/v1/approval-workflows", map[string]any{"name": "Priya's check",
		"levels": []map[string]any{{"name": "", "approverIds": []string{"U-1002"}}}})
	var answer errorAnswer
	created.Decode(t, &answer)
	if e := answer.Error; created.Status != http.StatusBadRequest || len(e.Fields) != 1 || e.Fields[0].Field != "levels.1.approverIds" ||
		e.Fields[0].Message != "Priya Raman can't approve activities. Give them a role that approves first." {
		t.Errorf("a workflow with Priya as approver = %d %s", created.Status, created.Body)
	}

	if saved := alex.Do(t, http.MethodPut, "/api/v1/people/U-1003/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1003", "scope": map[string]any{"type": "myTeams"}}}}); saved.Status != http.StatusOK {
		t.Fatalf("taking Daniel's Approver role = %d %s", saved.Status, saved.Body)
	}
	var details struct {
		Workflow struct {
			Levels []struct {
				Approvers []struct {
					Name       string
					CanApprove *bool
				}
			}
		}
	}
	alex.Do(t, http.MethodGet, "/api/v1/approval-workflows/WF-1001", nil).Decode(t, &details)
	levels := details.Workflow.Levels
	if len(levels) == 0 || len(levels[0].Approvers) == 0 || levels[0].Approvers[0].Name != "Daniel Okoro" ||
		levels[0].Approvers[0].CanApprove == nil || *levels[0].Approvers[0].CanApprove {
		t.Errorf("WF-1001's approvers = %+v, want Daniel flagged as not able to approve", levels)
	}
}
