package approvals_test

import (
	"net/http"
	"slices"
	"testing"

	"mapp/internal/activities"
	"mapp/internal/approvals"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/date"
)

// The seeded people of 017's tests (014's seed roles: Sara Novak is a Viewer).
const (
	sarasEmail       = "sara.novak@shell.com"
	standardWorkflow = "WF-1003" // Standard marketing approval (017 data-model §13)
)

// What a level's rule may name: every activity type, the business units in display order and only the roles
// that approve, by name (017 FR-001, FR-005).
func TestRuleOptions_FR001(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/v1/approval-rule-options", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s", response.Status, response.Body)
	}
	var options approvals.RuleOptions
	response.Decode(t, &options)
	var roles, units []string
	for _, r := range options.Roles {
		roles = append(roles, r.Name)
	}
	for _, u := range options.BusinessUnits {
		units = append(units, u.Label)
	}
	if !slices.Equal(roles, []string{"Administrator", "Approver", "Global planner"}) || !slices.Equal(units, []string{"Mobility", "Lubricants"}) ||
		len(options.ActivityTypes) != 6 {
		t.Errorf("options = roles %q, units %q, %d types", roles, units, len(options.ActivityTypes))
	}
}

// A workflow with every rule type answers 201 with the rules in words (017 FR-001 – FR-009).
func TestCreateWorkflow_RuleFields201_FR001(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	var options approvals.RuleOptions
	alex.Do(t, http.MethodGet, "/api/v1/approval-rule-options", nil).Decode(t, &options)
	mobility := options.BusinessUnits[0].ID

	response := alex.Do(t, http.MethodPost, "/api/v1/approval-workflows", map[string]any{
		"name": "Every rule", "reminderDays": 2, "escalationDays": 4,
		"levels": []map[string]any{
			{"name": "", "approverIds": []string{}, "approverType": "manager", "autoApproveWhen": map[string]any{"plannedCostBelow": "5000.00"},
				"backupApproverId": "U-1001"},
			{"name": "", "approverIds": []string{}, "approverType": "role", "roleId": "RL-1004", "mode": "all", "backupApproverId": "U-1001"},
			{"name": "Mobility", "approverIds": []string{"U-1003"}, "appliesWhen": map[string]any{"businessUnitIds": []string{mobility}},
				"backupApproverId": "U-1005"},
		},
	})

	view := decodeWorkflow(t, response, http.StatusCreated)
	want := "Level 1 · Line manager · automatic when planned cost is below $5,000.00 → Level 2 · Role: Approver · all must approve → " +
		"Mobility · Daniel Okoro · only when business unit is Mobility"
	if view.LevelsText != want || *view.ReminderDays != 2 || *view.EscalationDays != 4 || view.Levels[1].Role == nil ||
		view.Levels[1].Role.BuiltIn != "approver" || view.Levels[2].BackupApprover.Name != "Carlos Mendes" {
		t.Errorf("workflow = %+v\nlevels text %q", view, view.LevelsText)
	}
}

// Priya Raman's activity under $10,000.00 in "Standard marketing approval" is approved at submission (017
// FR-014, US1 scenario 1).
func TestSubmit_WF1003Automatic_FR014(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	editActivity(t, priya, fleetRoadshow, func(u *activities.Update) {
		workflow := standardWorkflow
		u.ApprovalWorkflowID, u.PlannedCost = &workflow, usd("8000.00")
	})
	var submission approvals.Submission
	priya.Do(t, http.MethodGet, "/api/v1/activities/"+fleetRoadshow+"/approval/submission", nil).Decode(t, &submission)

	approval := decodeApproval(t, priya.Do(t, http.MethodPost, "/api/v1/activities/"+fleetRoadshow+"/approval/submit",
		map[string]any{"version": submission.ActivityVersion}))

	if !submission.ApprovedAutomatically || submission.Outcomes[0].Outcome != "autoApproved" {
		t.Errorf("submission = %+v, want approved automatically", submission.Outcomes)
	}
	round := approval.Rounds[0]
	if approval.Status != "approved" || !round.AutoApproved || round.Outcome != approvals.OutcomeApproved || round.Facts == nil ||
		round.Facts.PlannedCost.Amount.String() != "8000.00" {
		t.Errorf("approval %s, round %+v; want approved at submission", approval.Status, round)
	}
}

// Segregation of duties is enforced by the server (017 FR-018): Alex Morgan, who may decide on behalf,
// re-submits Fleet roadshow and is refused when he then approves it.
func TestApprove_BySubmitter403_FR018(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	var submission approvals.Submission
	alex.Do(t, http.MethodGet, "/api/v1/activities/"+fleetRoadshow+"/approval/submission", nil).Decode(t, &submission)
	approval := decodeApproval(t, alex.Do(t, http.MethodPost, "/api/v1/activities/"+fleetRoadshow+"/approval/submit",
		map[string]any{"version": submission.ActivityVersion}))

	response := alex.Do(t, http.MethodPost, "/api/v1/activities/"+fleetRoadshow+"/approval/approve",
		map[string]any{"roundId": approval.Rounds[0].ID, "level": 1})

	assertError(t, response, http.StatusForbidden, "forbidden", "You submitted this activity, so someone else must approve it.")
	if approval.Actions.Approve.Available || approval.Actions.Approve.Reason != "You submitted this activity, so someone else must approve it." {
		t.Errorf("approve action = %+v, want unavailable with the reason", approval.Actions.Approve)
	}
}

// On an "all must approve" level an approver's second decision is refused, naming who is still waiting
// (017 FR-020).
func TestApprove_AlreadyApprovedAllLevel403_FR020(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	decodeWorkflow(t, alex.Do(t, http.MethodPatch, "/api/v1/approval-workflows/WF-1001", map[string]any{
		"levels":  []map[string]any{{"name": "", "approverIds": []string{"U-1003", "U-1005"}, "mode": "all", "backupApproverId": "U-1001"}},
		"version": 1,
	}), http.StatusOK)
	var submission approvals.Submission
	priya.Do(t, http.MethodGet, "/api/v1/activities/"+fleetRoadshow+"/approval/submission", nil).Decode(t, &submission)
	round := decodeApproval(t, priya.Do(t, http.MethodPost, "/api/v1/activities/"+fleetRoadshow+"/approval/submit",
		map[string]any{"version": submission.ActivityVersion})).Rounds[0]
	decision := map[string]any{"roundId": round.ID, "level": 1}

	first := decodeApproval(t, daniel.Do(t, http.MethodPost, "/api/v1/activities/"+fleetRoadshow+"/approval/approve", decision))
	again := daniel.Do(t, http.MethodPost, "/api/v1/activities/"+fleetRoadshow+"/approval/approve", decision)

	if level := first.Rounds[0].Levels[0]; first.Status != "submitted" || level.ApprovedCount != 1 || level.RequiredCount != 2 {
		t.Errorf("after Daniel: %s, level %+v; want 1 of 2 approved", first.Status, level)
	}
	assertError(t, again, http.StatusForbidden, "forbidden", "You already approved level 1. Waiting for Carlos Mendes.")
}

// Delegations through the API (017 FR-022, FR-024): Daniel Okoro sees his seeded planned delegation, adds a
// current one, changes its note, cancels it; a cancelled one answers 409, an unknown one 404.
func TestDelegations_CRUD_FR022_FR024(t *testing.T) {
	server := apitest.New(t)
	daniel := server.SignIn(t, danielsEmail)
	carlos := server.SignIn(t, carlosEmail)
	today := date.Today(now)

	var before approvals.DelegationPage
	daniel.Do(t, http.MethodGet, "/api/v1/approval-delegations", nil).Decode(t, &before)
	created := daniel.Do(t, http.MethodPost, "/api/v1/approval-delegations", map[string]any{
		"delegateId": "U-1005", "startDate": today.String(), "endDate": today.AddDays(2).String(), "note": "Conference"})
	refused := daniel.Do(t, http.MethodPost, "/api/v1/approval-delegations", map[string]any{
		"delegateId": "U-1006", "startDate": today.AddDays(20).String(), "endDate": today.AddDays(20).String()})

	if before.Total != 1 || before.Items[0].ID != "DLG-1001" || before.Items[0].State != "planned" || before.CanManageOthers {
		t.Errorf("before = %+v, want the seeded planned DLG-1001", before)
	}
	if created.Status != http.StatusCreated {
		t.Fatalf("POST = %d %s", created.Status, created.Body)
	}
	var view approvals.DelegationView
	created.Decode(t, &view)
	assertValidation(t, refused, `{"field":"delegateId","message":"Sara Novak can't approve activities. Choose someone with a role that approves."}`)

	var asDelegate approvals.DelegationPage
	carlos.Do(t, http.MethodGet, "/api/v1/approval-delegations?scope=mine", nil).Decode(t, &asDelegate)
	if asDelegate.Total != 2 || asDelegate.Items[0].ID != view.ID || asDelegate.Items[0].CanChange {
		t.Errorf("Carlos's list = %+v, want both of Daniel's, read-only", asDelegate)
	}

	changed := daniel.Do(t, http.MethodPatch, "/api/v1/approval-delegations/"+view.ID, map[string]any{"note": "Conference in Lisbon", "version": 1})
	cancelled := daniel.Do(t, http.MethodPost, "/api/v1/approval-delegations/"+view.ID+"/cancel", map[string]any{"version": 2})
	late := daniel.Do(t, http.MethodPost, "/api/v1/approval-delegations/"+view.ID+"/cancel", map[string]any{"version": 3})
	gone := daniel.Do(t, http.MethodPatch, "/api/v1/approval-delegations/DLG-9999", map[string]any{"version": 1})

	if changed.Status != http.StatusOK || cancelled.Status != http.StatusOK {
		t.Fatalf("PATCH %d %s / cancel %d %s", changed.Status, changed.Body, cancelled.Status, cancelled.Body)
	}
	cancelled.Decode(t, &view)
	if view.State != "cancelled" || view.Note != "Conference in Lisbon" || view.CancelledBy == nil {
		t.Errorf("cancelled = %+v", view)
	}
	assertError(t, late, http.StatusConflict, "not_allowed", "Ended and cancelled delegations can't be changed. They are kept for the audit trail.")
	assertError(t, gone, http.StatusNotFound, "not_found", "This delegation no longer exists.")
}

// Everyone's delegations need "Create and change approval workflows" (017 FR-024, D-296).
func TestDelegationsAll_NeedsWorkflowsManage_FR024(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	alex := server.SignIn(t, alexsEmail)

	refused := priya.Do(t, http.MethodGet, "/api/v1/approval-delegations?scope=all", nil)
	allowed := alex.Do(t, http.MethodGet, "/api/v1/approval-delegations?scope=all", nil)

	assertError(t, refused, http.StatusForbidden, "forbidden",
		"Changing approval workflows isn't part of your role (Planner). Ask an administrator if you need it.")
	var page approvals.DelegationPage
	allowed.Decode(t, &page)
	if allowed.Status != http.StatusOK || !page.CanManageOthers || page.Total != 1 || !page.Items[0].CanChange {
		t.Errorf("Alex's = %d %+v, want everyone's, changeable", allowed.Status, page)
	}
}

// Approvals › Monitoring on the seed (017 FR-030, US6): Trade magazine ads has waited its 2 escalation days
// and is the overdue row; the ageing bars add up to All open; only Alex may run escalations.
func TestMonitoring_Seeded_TradeMagazineOverdue_FR030(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	priya := server.SignIn(t, priyasEmail)

	var m, priyas approvals.Monitoring
	response := alex.Do(t, http.MethodGet, "/api/v1/approvals/monitoring", nil)
	response.Decode(t, &m)
	priya.Do(t, http.MethodGet, "/api/v1/approvals/monitoring", nil).Decode(t, &priyas)

	if response.Status != http.StatusOK || m.Cards.Overdue != 1 || len(m.Overdue) != 1 || m.Overdue[0].Activity.ID != tradeMagazineAds ||
		*m.Overdue[0].WaitingDays != 2 || !m.CanRunEscalations || priyas.CanRunEscalations {
		t.Errorf("monitoring = %+v", m)
	}
	total := 0
	for _, b := range m.Ageing {
		total += b.Count
	}
	if open := decodeRows(t, alex.Do(t, http.MethodGet, "/api/v1/approvals?tab=open", nil)); total != open.Total || len(m.ByLevel) != 2 {
		t.Errorf("ageing adds up to %d, All open %d; by level %+v", total, open.Total, m.ByLevel)
	}
	bad := alex.Do(t, http.MethodGet, "/api/v1/approvals/monitoring?monthStart=2026-03-01T00:00:00Z", nil)
	assertError(t, bad, http.StatusBadRequest, "validation_failed", "Some fields need your attention.")
}

// Run reminders and escalations on the seed (017 FR-027, US5): Trade magazine ads is escalated to Alex
// Morgan, who then approves it as backup approver; a second run sends nothing.
func TestRunEscalations_SeededEscalatesToAlex_FR027(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodPost, "/api/v1/approvals/escalations/run", nil)
	var run, again approvals.EscalationRun
	response.Decode(t, &run)
	alex.Do(t, http.MethodPost, "/api/v1/approvals/escalations/run", nil).Decode(t, &again)

	if response.Status != http.StatusOK || run.Escalated != 1 || run.Reminded != 0 || len(run.Events) != 1 ||
		run.Events[0].Activity.ID != tradeMagazineAds || run.Events[0].To[0].Name != "Alex Morgan" || len(run.NotEscalated) != 0 {
		t.Errorf("run = %d %+v, want Trade magazine ads escalated to Alex Morgan", response.Status, run)
	}
	if again.Escalated != 0 || again.Reminded != 0 {
		t.Errorf("second run = %+v, want nothing", again)
	}
	approval := decodeApproval(t, alex.Do(t, http.MethodPost, "/api/v1/activities/"+tradeMagazineAds+"/approval/approve",
		map[string]any{"roundId": "AR-1004", "level": 1}))
	if entry := approval.Rounds[0].History[0]; approval.Status != "approved" || entry.Capacity != approvals.CapacityBackup {
		t.Errorf("approval %s, entry %+v; want approved by Alex Morgan as backup approver", approval.Status, entry)
	}
}

// Running reminders and escalations needs "Create and change approval workflows" (017 FR-028).
func TestRunEscalations_WithoutPermission403_FR028(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/v1/approvals/escalations/run", nil)

	assertError(t, response, http.StatusForbidden, "forbidden",
		"Changing approval workflows isn't part of your role (Planner). Ask an administrator if you need it.")
}

// assertValidation checks a 400 validation_failed answer with exactly the field problems given as JSON.
func assertValidation(t *testing.T, response *apitest.Response, fields string) {
	t.Helper()
	want := `{"error":{"code":"validation_failed","message":"Some fields need your attention.","fields":[` + fields + `]}}`
	if response.Status != http.StatusBadRequest || string(response.Body) != want {
		t.Errorf("answer = %d %s\nwant 400 %s", response.Status, response.Body, want)
	}
}
