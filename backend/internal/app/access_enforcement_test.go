package app_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
)

// The demo people's e-mails, for the access tests (feature 014). Their seeded roles: Alex Administrator;
// Carlos Global planner (All markets); Daniel Planner and Approver, Priya Planner, Mei Planner, Sara Viewer —
// each for their own teams (Lubricants Europe: Priya, Daniel, Sara; Mobility China: Mei, Carlos, Sara).
const (
	alexsEmail   = "alex.morgan@shell.com"
	priyasEmail  = "priya.raman@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
	meisEmail    = "mei.lin@shell.com"
	carlosEmail  = "carlos.mendes@shell.com"
	sarasEmail   = "sara.novak@shell.com"
)

// errorAnswer is an error answer of the API, inside its envelope.
type errorAnswer struct {
	Error struct {
		Code, Message string
		Fields        []struct{ Field, Message string }
	}
}

// refused checks that the answer is 403 forbidden with message.
func refused(t *testing.T, what string, r *apitest.Response, message string) {
	t.Helper()
	var answer errorAnswer
	r.Decode(t, &answer)
	if e := answer.Error; r.Status != http.StatusForbidden || e.Code != "forbidden" || e.Message != message {
		t.Errorf("%s = %d %s\nwant 403 forbidden %q", what, r.Status, r.Body, message)
	}
}

// hidden checks that the answer is 404 not_found with the not-found message of record, as for a missing one.
func hidden(t *testing.T, what string, r *apitest.Response, record string) {
	t.Helper()
	want := `{"error":{"code":"not_found","message":"This ` + record + ` no longer exists — someone may have deleted it.","fields":[]}}`
	if r.Status != http.StatusNotFound || string(r.Body) != want {
		t.Errorf("%s = %d %s\nwant 404 %s", what, r.Status, r.Body, want)
	}
}

// ids returns the values of the field "id" of every item of a {"items": [...]} answer.
func itemIDs(t *testing.T, r *apitest.Response) []string {
	t.Helper()
	var page struct{ Items []struct{ ID string } }
	r.Decode(t, &page)
	list := []string{}
	for _, item := range page.Items {
		list = append(list, item.ID)
	}
	return list
}

// Every person sees only the plans of their scope, and the plans they own (US1 scenarios 1–3, FR-005 –
// FR-007): Priya and Mei their own market's, Sara both of hers, Carlos every plan; the count is over what
// they see.
func TestPlansList_FilteredByScope_FR006_FR007(t *testing.T) {
	server := apitest.New(t)
	for _, c := range []struct {
		email string
		want  []string
	}{
		{priyasEmail, []string{"P-1004", "P-1001"}},
		{meisEmail, []string{"P-1002"}},
		{sarasEmail, []string{"P-1004", "P-1001", "P-1002"}},
		{carlosEmail, []string{"P-1003", "P-1004", "P-1001", "P-1002"}},
		{alexsEmail, []string{"P-1003", "P-1004", "P-1001", "P-1002"}},
	} {
		response := server.SignIn(t, c.email).Do(t, http.MethodGet, "/api/v1/plans", nil)
		var list struct {
			Items []struct{ ID string }
			Total int
		}
		response.Decode(t, &list)
		var got []string
		for _, p := range list.Items {
			got = append(got, p.ID)
		}
		if !slices.Equal(got, c.want) || list.Total != len(c.want) {
			t.Errorf("%s sees %q of %d, want %q", c.email, got, list.Total, c.want)
		}
	}
}

// A plan's owner sees it whatever their scope (FR-006): Mei, whose role now applies to Global Marketing Ops
// only, still sees FY2026 Mobility China, which she owns; Priya, neither in its team nor its owner, does not.
func TestPlan_OwnerSeesIt_FR006(t *testing.T) {
	server := apitest.New(t)
	alex, mei, priya := server.SignIn(t, alexsEmail), server.SignIn(t, meisEmail), server.SignIn(t, priyasEmail)
	saved := alex.Do(t, http.MethodPut, "/api/v1/people/U-1004/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1003", "scope": map[string]any{"type": "teams", "teamIds": []string{"T-1003"}}}}})
	if saved.Status != http.StatusOK {
		t.Fatalf("changing Mei's roles = %d %s", saved.Status, saved.Body)
	}

	if opened := mei.Do(t, http.MethodGet, "/api/v1/plans/P-1002", nil); opened.Status != http.StatusOK {
		t.Errorf("Mei, the owner, opening her plan = %d %s, want 200", opened.Status, opened.Body)
	}
	if got := itemIDs(t, mei.Do(t, http.MethodGet, "/api/v1/plans", nil)); !slices.Equal(got, []string{"P-1003", "P-1002"}) {
		t.Errorf("Mei's plans = %q, want Global Marketing Ops's and her own", got)
	}
	hidden(t, "Priya opening it", priya.Do(t, http.MethodGet, "/api/v1/plans/P-1002", nil), "plan")
}

// planForm is the whole form of a new 2026 plan called name for the teams teamIDs, as the web app sends it.
func planForm(name string, teamIDs ...string) map[string]any {
	return map[string]any{
		"name": name, "year": 2026, "modelId": "MOD-1001", "teamIds": teamIDs, "ownerId": nil, "aop": false,
		"description": "", "budget": nil, "defaultFinanceBudgetId": nil, "defaultSpread": "evenly", "bcpir": false,
		"goalStage": "inq", "goalAmount": "", "attributes": map[string]any{},
	}
}

// A record outside a person's access answers exactly as a record that does not exist — the same 404
// not_found and message, naming nothing of it — whatever its level and the operation (022 FR-005, SC-003).
func TestRecordOutsideAccess_AnswersLikeAMissingOne_022_FR005(t *testing.T) {
	server := apitest.New(t)
	mei := server.SignIn(t, meisEmail)
	for _, c := range []struct{ method, outside, missing string }{
		{http.MethodGet, "/api/v1/plans/P-1001", "/api/v1/plans/P-9999"},
		{http.MethodGet, "/api/v1/plans/P-1001/campaigns", "/api/v1/plans/P-9999/campaigns"},
		{http.MethodGet, "/api/v1/campaigns/C-1001", "/api/v1/campaigns/C-9999"},
		{http.MethodGet, "/api/v1/initiatives/I-1001", "/api/v1/initiatives/I-9999"},
		{http.MethodGet, "/api/v1/activities/A-1001", "/api/v1/activities/A-9999"},
		{http.MethodGet, "/api/v1/activities/A-1001/approval", "/api/v1/activities/A-9999/approval"},
		{http.MethodGet, "/api/v1/line-items/L-1001", "/api/v1/line-items/L-9999"},
		{http.MethodGet, "/api/v1/all-work/records/A-1001", "/api/v1/all-work/records/A-9999"},
		{http.MethodDelete, "/api/v1/activities/A-1001", "/api/v1/activities/A-9999"},
		{http.MethodDelete, "/api/v1/line-items/L-1001", "/api/v1/line-items/L-9999"},
	} {
		outside, missing := mei.Do(t, c.method, c.outside, nil), mei.Do(t, c.method, c.missing, nil)
		if outside.Status != http.StatusNotFound || missing.Status != outside.Status || string(missing.Body) != string(outside.Body) {
			t.Errorf("Mei: %s %s = %d %s\nwant as %s: %d %s", c.method, c.outside, outside.Status, outside.Body,
				c.missing, missing.Status, missing.Body)
		}
	}
	hidden(t, "Mei: GET the line items of A-1001", mei.Do(t, http.MethodGet, "/api/v1/activities/A-1001/line-items", nil), "activity")
	if own := mei.Do(t, http.MethodGet, "/api/v1/activities/A-1008", nil); own.Status != http.StatusOK {
		t.Errorf("Mei opening her own market's activity = %d %s, want 200", own.Status, own.Body)
	}
}

// Changes need the level's permission in the record's plan (FR-009, FR-010): Sara, a Viewer, is told her
// role does not include it; Priya may change only her market's records; nothing is changed.
func TestChanges_NeedThePermissionInThePlan_FR009_FR010(t *testing.T) {
	server := apitest.New(t)
	sara, priya := server.SignIn(t, sarasEmail), server.SignIn(t, priyasEmail)
	rename := map[string]any{"name": "Renamed", "version": 1}

	refused(t, "Sara renaming a campaign", sara.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1001", rename, apitest.SkipRequestValidation()),
		"Changing campaigns and initiatives isn't part of your role (Viewer). Ask an administrator if you need it.")
	refused(t, "Sara deleting an activity", sara.Do(t, http.MethodDelete, "/api/v1/activities/A-1001", nil),
		"Deleting activities and line items isn't part of your role (Viewer). Ask an administrator if you need it.")
	refused(t, "Priya deleting a plan", priya.Do(t, http.MethodDelete, "/api/v1/plans/P-1001", nil),
		"Deleting plans isn't part of your role (Planner). Ask an administrator if you need it.")
	hidden(t, "Priya adding a campaign to the Mobility plan",
		priya.Do(t, http.MethodPost, "/api/v1/plans/P-1002/campaigns", map[string]any{"name": "Spring"}, apitest.SkipRequestValidation()),
		"plan")

	// Given the Planner role for Mobility China too, Priya sees the Mobility plan, and still may not delete there.
	server.GrantRole(t, priyasEmail, "viewer", "teams", "Mobility China")
	refused(t, "Priya, a Viewer of Mobility China, changing its campaign",
		priya.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1003", rename, apitest.SkipRequestValidation()),
		"Changing campaigns and initiatives is allowed for you only in plans of Lubricants Europe. Ask an administrator if you need it here.")

	var campaign struct {
		Name    string
		Version int
	}
	sara.Do(t, http.MethodGet, "/api/v1/campaigns/C-1001", nil).Decode(t, &campaign)
	if campaign.Name != "Helix launch" || campaign.Version != 1 {
		t.Errorf("after the refusals the campaign is %q version %d, want unchanged", campaign.Name, campaign.Version)
	}
}

// A person whose plan permissions are for their teams may give a plan only their own teams, and must keep
// one of them on it (FR-013, messages 20 and 21); Carlos, for All markets, may give any team.
func TestPlanTeams_OwnTeamsOnly_FR013(t *testing.T) {
	server := apitest.New(t)
	priya, carlos := server.SignIn(t, priyasEmail), server.SignIn(t, carlosEmail)

	other := priya.Do(t, http.MethodPost, "/api/v1/plans", planForm("FY2026 Mobility extra", "T-1002"))
	mine := priya.Do(t, http.MethodPost, "/api/v1/plans", planForm("FY2026 Lubricants extra", "T-1001"))
	carlosPlan := carlos.Do(t, http.MethodPost, "/api/v1/plans", planForm("FY2026 Global extra", "T-1001", "T-1002"))
	var shared struct{ ID string }
	carlosPlan.Decode(t, &shared)
	edit := planForm("FY2026 Global extra", "T-1002") // Priya takes Lubricants Europe off Carlos's plan
	delete(edit, "year")
	edit["version"] = 1
	dropped := priya.Do(t, http.MethodPatch, "/api/v1/plans/"+shared.ID, edit)

	assertProblem := func(what string, r *apitest.Response, message string) {
		t.Helper()
		var e errorAnswer
		r.Decode(t, &e)
		if r.Status != http.StatusBadRequest || !slices.ContainsFunc(e.Error.Fields, func(f struct{ Field, Message string }) bool {
			return f.Field == "teamIds" && f.Message == message
		}) {
			t.Errorf("%s = %d %s\nwant teamIds: %s", what, r.Status, r.Body, message)
		}
	}
	assertProblem("Priya's plan for Mobility China", other, "You can only choose your own teams: Lubricants Europe.")
	assertProblem("Priya dropping her team", dropped,
		"Keep at least one of your teams (Lubricants Europe) on the plan, or you won't be able to see it.")
	if mine.Status != http.StatusCreated || carlosPlan.Status != http.StatusCreated {
		t.Errorf("Priya's own plan = %d %s; Carlos's = %d %s; want both created", mine.Status, mine.Body, carlosPlan.Status, carlosPlan.Body)
	}
}

// Moves and copies check the destination too, and destination lists offer only plans where the person may
// change campaigns (FR-014).
func TestCampaignMoves_DestinationInScope_FR014(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "viewer", "teams", "Mobility China") // she sees the Mobility plan, may not change it

	var destinations struct{ Items []struct{ ID string } }
	priya.Do(t, http.MethodGet, "/api/v1/campaigns/C-1001/destinations?purpose=copy", nil).Decode(t, &destinations)
	moved := priya.Do(t, http.MethodPost, "/api/v1/campaigns/C-1001/move", map[string]any{"planId": "P-1002", "version": 1})
	copied := priya.Do(t, http.MethodPost, "/api/v1/campaigns/C-1001/copy", map[string]any{"planId": "P-1002", "name": "Helix copy"})

	if len(destinations.Items) != 1 || destinations.Items[0].ID != "P-1001" {
		t.Errorf("copy destinations = %+v, want her own plan only", destinations.Items)
	}
	message := "Changing campaigns and initiatives is allowed for you only in plans of Lubricants Europe. Ask an administrator if you need it here."
	refused(t, "moving to the Mobility plan", moved, message)
	refused(t, "copying to the Mobility plan", copied, message)
}

// Home, All work and the Approvals page count and list only what the person sees (FR-007, SC-004): Priya's
// totals are the Lubricants plan's.
func TestListsAndTotals_OnlyWhatThePersonSees_SC004(t *testing.T) {
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

// The activity log and the recycle bin show only entries about what the person sees; Settings changes need
// "See the whole activity log"; restoring needs the level's delete permission in the plan (FR-007, FR-038,
// D-238).
func TestLogAndRecycleBin_Filtered_FR007_FR038(t *testing.T) {
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

// Approving needs to be a named approver who may approve, or to decide on behalf of the approvers; nobody
// approves an activity they own or submitted while the rule is on; the Approval section offers only what the
// person may do (US5, FR-025 – FR-027, messages 5, 6a, 6b).
func TestApprovals_WhoMayDecide_FR025_FR026(t *testing.T) {
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

	// Priya submitted A-1004 and owns it; Daniel is the approver of its level 1.
	refused(t, "Priya approving her own activity", priya.Do(t, http.MethodPost, "/api/v1/activities/A-1004/approval/approve", decision(1)),
		"Only the approvers of level 1 (Daniel Okoro) can approve or decline this activity.")
	var priyas actions
	priya.Do(t, http.MethodGet, "/api/v1/activities/A-1004/approval", nil).Decode(t, &priyas)
	if priyas.Actions.Approve.Available || priyas.Actions.Approve.Reason != "" || !priyas.Actions.Recall.Available {
		t.Errorf("Priya's actions = %+v, want Approve hidden and Recall offered", priyas.Actions)
	}

	// Given the Administrator role, Priya could decide on behalf of Daniel — but she owns and submitted the
	// activity: the segregation of duties stops her, with the reason on screen (message 6a).
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

// Submitting needs "Submit activities" in the plan, and recalling is for the owner or the submitter (with
// it) or for people who decide on behalf of the approvers (FR-027).
func TestApprovals_SubmitAndRecall_FR027(t *testing.T) {
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

// New approvers of a workflow must hold "Approve or decline when named as an approver", and approvers are
// flagged when they no longer do (FR-028, message 8).
func TestWorkflowApprovers_MustApprove_FR028(t *testing.T) {
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
