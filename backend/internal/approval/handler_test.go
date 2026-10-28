package approval_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"mapp/internal/approval"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/apitest"
)

const (
	alexsEmail   = "alex.morgan@shell.com"
	priyasEmail  = "priya.raman@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
	meisEmail    = "mei.lin@shell.com"
	carlosEmail  = "carlos.mendes@shell.com"
)

var now = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

const (
	fleetRoadshow     = "A-1005"
	fleetCardLaunch   = "A-1009"
	lunarNewYearPromo = "A-1010"
	tradeMagazineAds  = "A-1004"
)

func TestGetActivityApproval_MobilityWaitingAtLevel2_FR024_FR037(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodGet, "/api/v1/activities/"+fleetCardLaunch+"/approval", nil)
	gone := alex.Do(t, http.MethodGet, "/api/v1/activities/A-9999/approval", nil)

	appr := decodeApproval(t, response)
	if appr.Status != "submitted" || appr.Workflow == nil || appr.Workflow.ID != "WF-1002" || len(appr.Rounds) != 1 {
		t.Fatalf("approval = %+v, want Submitted with WF-1002 and one round", appr)
	}
	round := appr.Rounds[0]
	carlos := actor.Actor{ID: "U-1005", Name: "Carlos Mendes"}
	if round.ID != "AR-1008" || round.Outcome != approval.OutcomeOpen || round.CurrentLevel != 2 || round.LevelCount != 2 {
		t.Errorf("round = %s %s at level %d of %d, want AR-1008 open at level 2 of 2", round.ID, round.Outcome, round.CurrentLevel, round.LevelCount)
	}
	first, second := round.Levels[0], round.Levels[1]
	if first.State != "approved" || first.Decision == nil || first.Decision.Actor != carlos || !first.Decision.At.Equal(now.Add(-2*time.Hour)) {
		t.Errorf("level 1 = %+v, want approved by Carlos Mendes 2 hours ago", first)
	}
	if second.State != "waiting" || second.Decision != nil || len(second.Approvers) != 1 || second.Approvers[0].Name != "Alex Morgan" {
		t.Errorf("level 2 = %+v, want waiting for Alex Morgan", second)
	}
	if actions := historyActions(round); !slices.Equal(actions, []string{"approved", "submitted"}) {
		t.Errorf("history = %q, want approved, submitted (newest first)", actions)
	}
	if !appr.Viewer.IsCurrentApprover || appr.Viewer.IsOwner || len(appr.Viewer.OnBehalfOf) != 0 {
		t.Errorf("viewer = %+v, want Alex as the current approver", appr.Viewer)
	}
	if !appr.Actions.Approve.Available || !appr.Actions.Decline.Available || appr.Actions.Submit.Available {
		t.Errorf("actions = %+v, want approve and decline", appr.Actions)
	}
	assertError(t, gone, http.StatusNotFound, "not_found", "This activity no longer exists — someone may have deleted it.")
}

func TestGetSubmission_IncompleteMobilityListsAll_FR010(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	mei := server.SignIn(t, meisEmail)

	response := mei.Do(t, http.MethodGet, "/api/v1/activities/"+lunarNewYearPromo+"/approval/submission", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var submission approval.Submission
	response.Decode(t, &submission)
	if submission.Ready || !submission.Available || submission.Kind != "submit" || submission.RoundNumber != 1 ||
		submission.Workflow != nil || len(submission.Levels) != 0 {
		t.Errorf("submission = %+v, want an available submit that is not ready, without workflow", submission)
	}
	assertListsFields(t, problemFields(submission.Problems), "ownerId", "startDate", "endDate", "plannedCost", "approvalWorkflowId")
}

func TestSubmit_ReadyActivity_FR011(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	submission := priya.Do(t, http.MethodGet, "/api/v1/activities/"+fleetRoadshow+"/approval/submission", nil)
	var before approval.Submission
	submission.Decode(t, &before)
	if !before.Ready || before.Kind != "resubmit" || before.RoundNumber != 2 || before.Workflow == nil || before.Workflow.ID != "WF-1001" {
		t.Fatalf("submission = %+v, want a ready re-submission with WF-1001 as round 2", before)
	}

	response := priya.Do(t, http.MethodPost, "/api/v1/activities/"+fleetRoadshow+"/approval/submit",
		map[string]any{"version": before.ActivityVersion, "comment": "Reduced to fit the Q2 budget."})
	stale := priya.Do(t, http.MethodPost, "/api/v1/activities/"+fleetRoadshow+"/approval/submit", map[string]any{"version": before.ActivityVersion})

	appr := decodeApproval(t, response)
	if appr.Status != "submitted" || appr.ActivityVersion != before.ActivityVersion+1 || appr.LatestDecline != nil {
		t.Errorf("approval: %s, version %d, latest decline %v; want submitted with a new version", appr.Status,
			appr.ActivityVersion, appr.LatestDecline)
	}
	round := appr.Rounds[0]
	if round.Number != 2 || round.Outcome != approval.OutcomeOpen || round.Comment != "Reduced to fit the Q2 budget." ||
		round.Levels[0].State != "waiting" || round.Levels[0].Approvers[0].Name != "Daniel Okoro" {
		t.Errorf("round = %+v, want round 2 waiting for Daniel Okoro with the comment", round)
	}
	if actions := historyActions(round); !slices.Equal(actions, []string{"resubmitted"}) {
		t.Errorf("history of round 2 = %q, want resubmitted", actions)
	}
	assertError(t, stale, http.StatusConflict, "version_conflict", "Not saved — Priya Raman changed this activity")
}

func TestApprove_Level2AsAlex_US2(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodPost, "/api/v1/activities/"+fleetCardLaunch+"/approval/approve",
		map[string]any{"roundId": "AR-1008", "level": 2, "comment": "Approved for launch."})

	appr := decodeApproval(t, response)
	round := appr.Rounds[0]
	if appr.Status != "approved" || round.Outcome != approval.OutcomeApproved || round.EndedAt == nil || !round.EndedAt.Equal(now) {
		t.Errorf("approval %s, round %s ended %v; want approved now", appr.Status, round.Outcome, round.EndedAt)
	}
	if actions := historyActions(round); !slices.Equal(actions, []string{"approved", "approved", "submitted"}) {
		t.Errorf("history = %q, want both approvals and the submission", actions)
	}
	if first, second := round.Levels[0].Decision, round.Levels[1].Decision; first == nil || first.Actor.Name != "Carlos Mendes" ||
		second == nil || second.Actor.Name != "Alex Morgan" || second.Comment != "Approved for launch." {
		t.Errorf("decisions = %+v / %+v, want Carlos Mendes at level 1 and Alex Morgan at level 2", first, second)
	}
}

func TestListApprovals_TabsForDaniel_FR029(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	daniel := server.SignIn(t, danielsEmail)
	server.GrantRole(t, danielsEmail, "approver", "all")

	waiting := decodeRows(t, daniel.Do(t, http.MethodGet, "/api/v1/approvals", nil))
	open := decodeRows(t, daniel.Do(t, http.MethodGet, "/api/v1/approvals?tab=open", nil))
	decided := decodeRows(t, daniel.Do(t, http.MethodGet, "/api/v1/approvals?tab=decided", nil))
	unknown := daniel.Do(t, http.MethodGet, "/api/v1/approvals?tab=mine", nil, apitest.SkipRequestValidation())

	if got := rowIDs(waiting); !slices.Equal(got, []string{tradeMagazineAds}) || waiting.Total != 1 {
		t.Errorf("waiting = %q (total %d), want only %s", got, waiting.Total, tradeMagazineAds)
	}
	row := waiting.Items[0]
	if row.Round.Number != 2 || row.Round.LevelLabel != "Level 1" || row.Round.LevelCount != 1 || row.Owner == nil ||
		row.Owner.Name != "Priya Raman" || len(row.Path) != 3 || row.WaitingSince == nil || !row.WaitingSince.Equal(now.Add(-48*time.Hour)) {
		t.Errorf("waiting row = %+v, want round 2 at level 1 of 1, owned by Priya Raman, waiting for 2 days", row)
	}
	if got := rowIDs(open); !slices.Equal(got, []string{fleetCardLaunch, tradeMagazineAds}) {
		t.Errorf("open = %q, want %s then %s (oldest submission first)", got, fleetCardLaunch, tradeMagazineAds)
	}
	if got := rowIDs(decided); len(got) != 6 || got[0] != fleetRoadshow || got[1] != tradeMagazineAds {
		t.Errorf("decided = %q, want the two declines first, then four approvals", got)
	}
	if first := decided.Items[0]; first.MyDecision == nil || first.MyDecision.Action != "declined" || first.Activity.Status != "declined" {
		t.Errorf("first decided row = %+v, want Daniel's decline of the Declined activity", first)
	}
	assertError(t, unknown, http.StatusBadRequest, "validation_failed", "Some fields need your attention.")
}

func TestApprovalSummary_FR028(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	daniel := server.SignIn(t, danielsEmail)
	server.GrantRole(t, danielsEmail, "approver", "all")

	march := daniel.Do(t, http.MethodGet, "/api/v1/approvals/summary", nil)
	february := daniel.Do(t, http.MethodGet, "/api/v1/approvals/summary?monthStart=2026-02-01T00:00:00Z&monthEnd=2026-03-01T00:00:00Z", nil)
	broken := daniel.Do(t, http.MethodGet, "/api/v1/approvals/summary?monthStart=yesterday&monthEnd=2026-03-01T00:00:00Z", nil,
		apitest.SkipRequestValidation())

	if got, want := decodeCounts(t, march), (approval.Counts{WaitingForMe: 1, AllOpen: 2, DeclinedThisMonth: 2}); got != want {
		t.Errorf("this month = %+v, want %+v", got, want)
	}
	if got, want := decodeCounts(t, february), (approval.Counts{WaitingForMe: 1, AllOpen: 2, ApprovedThisMonth: 4}); got != want {
		t.Errorf("February = %+v, want %+v", got, want)
	}
	assertError(t, broken, http.StatusBadRequest, "validation_failed", "Some fields need your attention.")
}

func TestApprovalPreview_FR032(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	daniel := server.SignIn(t, danielsEmail)

	response := daniel.Do(t, http.MethodGet, "/api/v1/approvals/activities/"+tradeMagazineAds, nil)
	gone := daniel.Do(t, http.MethodGet, "/api/v1/approvals/activities/A-9999", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var preview approval.Preview
	response.Decode(t, &preview)
	activity := preview.Activity
	if activity.ID != tradeMagazineAds || activity.Status != "submitted" || len(activity.Path) != 3 || activity.Owner == nil ||
		activity.Owner.Name != "Priya Raman" || activity.PlannedCost == nil || activity.TypeName == "" {
		t.Errorf("activity = %+v, want the Submitted Trade magazine ads of Priya Raman", activity)
	}
	if len(preview.LineItems) != 1 || preview.LineItems[0].Name != "Media booking" || preview.LineItems[0].PlannedCostDisplay == nil ||
		preview.LineItems[0].PlannedCost.Currency != "USD" {
		t.Errorf("line items = %+v, want Media booking in USD", preview.LineItems)
	}
	if preview.UnassignedPlannedCost == nil {
		t.Error("unassigned planned cost is null, want the balance")
	}
	if len(preview.Approval.Rounds) != 2 || !preview.Approval.Viewer.IsCurrentApprover || !preview.Approval.Actions.Decline.Available {
		t.Errorf("approval = %+v, want two rounds, the second waiting for Daniel", preview.Approval)
	}
	assertError(t, gone, http.StatusNotFound, "not_found", "This activity no longer exists — someone may have deleted it.")
}

func decodeRows(t *testing.T, response *apitest.Response) approval.RowPage {
	t.Helper()
	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var page approval.RowPage
	response.Decode(t, &page)
	return page
}

func rowIDs(page approval.RowPage) []string {
	ids := []string{}
	for _, row := range page.Items {
		ids = append(ids, row.Activity.ID)
	}
	return ids
}

func decodeCounts(t *testing.T, response *apitest.Response) approval.Counts {
	t.Helper()
	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var counts approval.Counts
	response.Decode(t, &counts)
	return counts
}

func decodeApproval(t *testing.T, response *apitest.Response) approval.ActivityApproval {
	t.Helper()
	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var appr approval.ActivityApproval
	response.Decode(t, &appr)
	return appr
}

func historyActions(round approval.RoundView) []string {
	var actions []string
	for _, e := range round.History {
		actions = append(actions, e.Action)
	}
	return actions
}

func problemFields(problems []approval.SubmitProblem) []string {
	var fields []string
	for _, p := range problems {
		fields = append(fields, p.Field)
	}
	return fields
}

func assertListsFields(t *testing.T, fields []string, want ...string) {
	t.Helper()
	for _, field := range want {
		if !slices.Contains(fields, field) {
			t.Errorf("problems %q do not list %s", fields, field)
		}
	}
}

func assertError(t *testing.T, response *apitest.Response, status int, code, message string) {
	t.Helper()
	var answer struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	response.Decode(t, &answer)
	if response.Status != status || answer.Error.Code != code || answer.Error.Message != message {
		t.Errorf("answer = %d %s\nwant %d %s %q", response.Status, response.Body, status, code, message)
	}
}
