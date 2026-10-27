package activities_test

import (
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"mapp/internal/activities"
	"mapp/internal/history"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/errs"
)

// Split activity (016 US4) on the demo data: A-1011 "UK launch event" of the GBP plan P-1004 in the
// initiative I-1009 "UK" (40,000.00 GBP, 01 Mar – 31 May 2027) with L-1009 "Venue" 15,000.00 GBP and L-1010
// "Agency fee (Ireland)" 23,000.00 EUR = 19,750.00 GBP; I-1010 "Ireland" holds A-1012 "Dealer conference".

const (
	ukLaunchEvent = "A-1011"
	ukInitiative  = "I-1009"
	irelandInit   = "I-1010"
	venueItem     = "L-1009"
	agencyItem    = "L-1010"
)

// splitBody is the body of a split preview or split.
func splitBody(name, initiativeID string, lineItemIDs []string, plannedCost any, restart bool, version int) map[string]any {
	if lineItemIDs == nil {
		lineItemIDs = []string{}
	}
	return map[string]any{
		"name": name, "initiativeId": initiativeID, "lineItemIds": lineItemIDs, "plannedCost": plannedCost,
		"restartApproval": restart, "version": version,
	}
}

// The Split dialog of UK launch event lists its line items with their amounts in GBP, suggests "UK launch
// event (split)" and offers the plan's initiatives, its own first (FR-026).
func TestSplitOptions_Contract_FR026(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/v1/activities/"+ukLaunchEvent+"/split-options", nil)

	assertStatus(t, response, http.StatusOK)
	var options activities.SplitOptions
	response.Decode(t, &options)
	if options.PlanCurrency != "GBP" || options.SuggestedName != "UK launch event (split)" || options.NeedsRestart ||
		options.PlannedCost.Format() != "40,000.00 GBP" || len(options.LineItems) != 2 ||
		options.LineItems[1].PlannedCost.Format() != "23,000.00 EUR" || options.LineItems[1].PlannedCostPlan.Format() != "19,750.00 GBP" {
		t.Errorf("options = %s", response.Body)
	}
	if len(options.Destinations) != 2 || options.Destinations[0].Initiative.ID != ukInitiative || options.Destinations[0].MoveProblem != nil ||
		options.Destinations[1].Initiative.ID != irelandInit || options.Destinations[1].MoveProblem != nil {
		t.Errorf("destinations = %+v", options.Destinations)
	}
}

// The preview shows both activities as the split leaves them (spec US4-1): the original at 20,250.00 GBP
// with 5,250.00 GBP unassigned, the new one at 19,750.00 GBP with 0.00 unassigned; the split then gives the
// same figures, keeps L-1010's ID and links both activities (FR-027, FR-028, FR-031).
func TestSplitPreview_MatchesSplit_FR027(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	body := splitBody("Ireland roadshow", irelandInit, []string{agencyItem}, "19750.00", false, 1)

	previewed := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split-preview", body)

	assertStatus(t, previewed, http.StatusOK)
	var preview activities.SplitPreview
	previewed.Decode(t, &preview)
	if len(preview.Problems) != 0 || preview.Original == nil || preview.Created == nil {
		t.Fatalf("preview = %s", previewed.Body)
	}
	o, c := preview.Original, preview.Created
	if o.PlannedCost.Format() != "20,250.00 GBP" || o.UnassignedPlannedCost.Format() != "5,250.00 GBP" || o.UnassignedState != "left" ||
		!slices.Equal(o.LineItemNames, []string{"Venue"}) || o.Status != activities.StatusCreated || o.InitiativeName != "UK" {
		t.Errorf("original side = %+v", o)
	}
	if c.PlannedCost.Format() != "19,750.00 GBP" || c.UnassignedPlannedCost.Format() != "0.00 GBP" || c.UnassignedState != "allAssigned" ||
		!slices.Equal(c.LineItemNames, []string{"Agency fee (Ireland)"}) || c.Name != "Ireland roadshow" || c.InitiativeName != "Ireland" {
		t.Errorf("created side = %+v", c)
	}

	split := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split", body)

	assertStatus(t, split, http.StatusCreated)
	var result activities.SplitResult
	split.Decode(t, &result)
	original, created := result.Activity, result.NewActivity
	if original.PlannedCost.Format() != "20,250.00 GBP" || original.UnassignedPlannedCost.Format() != "5,250.00 GBP" ||
		original.LineItems.Count != 1 || len(original.SplitInto) != 1 || original.SplitInto[0].ID != created.ID {
		t.Errorf("original = %+v", original)
	}
	if created.ID != "A-1014" || created.Name != "Ireland roadshow" || created.Initiative.ID != irelandInit ||
		created.PlannedCost.Format() != "19,750.00 GBP" || created.UnassignedPlannedCost.Format() != "0.00 GBP" ||
		created.LineItems.Count != 1 || created.SplitFrom == nil ||
		*created.SplitFrom != (activities.SplitLink{ID: ukLaunchEvent, Name: "UK launch event", PlanID: "P-1004"}) {
		t.Errorf("new activity = %+v (split from %+v)", created, created.SplitFrom)
	}
	var moved struct {
		LineItem struct{ ID, ActivityID string } `json:"lineItem"`
	}
	priya.Do(t, http.MethodGet, "/api/v1/line-items/"+agencyItem, nil).Decode(t, &moved)
	if moved.LineItem.ID != agencyItem || moved.LineItem.ActivityID != created.ID {
		t.Errorf("L-1010 after the split = %+v, want it under %s", moved.LineItem, created.ID)
	}
}

// A split moves amounts, never creates or loses them: the plan's planned cost and line items stay the same
// (SC-005).
func TestSplit_SumsUnchanged_SC005(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	type plan struct {
		PlanFigures struct {
			PlannedCost struct{ Amount, Currency string } `json:"plannedCost"`
		} `json:"planFigures"`
		Contents struct{ Activities, LineItems int } `json:"contents"`
	}
	var before, after plan
	priya.Do(t, http.MethodGet, "/api/v1/plans/P-1004", nil).Decode(t, &before)

	split := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split",
		splitBody("Ireland roadshow", irelandInit, []string{agencyItem}, "12000.00", false, 1))

	assertStatus(t, split, http.StatusCreated)
	priya.Do(t, http.MethodGet, "/api/v1/plans/P-1004", nil).Decode(t, &after)
	if before.PlanFigures.PlannedCost != after.PlanFigures.PlannedCost || after.PlanFigures.PlannedCost.Amount != "65000.00" ||
		before.Contents.LineItems != after.Contents.LineItems || after.Contents.Activities != before.Contents.Activities+1 {
		t.Errorf("before %+v, after %+v", before, after)
	}
}

// Nothing ticked and 0.00 to move: the preview lists the problem and the split refuses it, changing nothing
// (FR-027, US4-4).
func TestSplit_NothingChosen_FR027(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	body := splitBody("Ireland roadshow", irelandInit, nil, "0", false, 1)

	previewed := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split-preview", body)
	refused := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split", body)

	var preview activities.SplitPreview
	previewed.Decode(t, &preview)
	want := []errs.FieldError{{Field: "lineItemIds", Message: "Choose line items or an amount to move."}}
	if !reflect.DeepEqual(preview.Problems, want) {
		t.Errorf("preview problems = %+v, want %+v", preview.Problems, want)
	}
	assertAnswer(t, refused, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"lineItemIds","message":"Choose line items or an amount to move."}]}}`)
}

// At most the activity's planned cost moves (FR-026); a line item of another activity is refused too.
func TestSplit_AmountTooHigh_FR026(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	refused := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split",
		splitBody("Ireland roadshow", irelandInit, []string{"L-1001"}, "50000", false, 1))

	assertAnswer(t, refused, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"lineItemIds","message":"L-1001 is not a line item of this activity."},`+
		`{"field":"plannedCost","message":"Move at most 40,000.00 GBP, the activity's planned cost."}]}}`)
}

// The new activity's name is unique in the chosen initiative, ignoring letter case (FR-026).
func TestSplit_NameUnique_FR026(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	refused := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split",
		splitBody("dealer CONFERENCE", irelandInit, []string{agencyItem}, nil, false, 1))

	assertAnswer(t, refused, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"name","message":"'Ireland' already has an activity called 'Dealer conference'."}]}}`)
}

// The new activity's initiative follows Move's rules: another plan's initiative is refused, and so is one
// whose dates do not contain the activity's (FR-026, US4-5).
func TestSplit_DestinationRules_FR026(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	otherPlan := priya.Do(t, http.MethodPost, "/api/v1/activities/A-1001/split",
		splitBody("Launch film part 2", ukInitiative, nil, "1000", false, 1))
	outside := priya.Do(t, http.MethodPost, "/api/v1/activities/A-1001/split",
		splitBody("Launch film part 2", "I-1004", nil, "1000", false, 1))
	options := priya.Do(t, http.MethodGet, "/api/v1/activities/A-1001/split-options", nil)

	assertAnswer(t, otherPlan, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"initiativeId","message":"Choose another initiative of this plan."}]}}`)
	assertAnswer(t, outside, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"initiativeId","message":"'Launch film' runs 01 Mar – 31 May 2026, but 'Retail events' runs `+
		`01 Jun – 31 Dec 2026. Change the activity's dates first, or pick another initiative."}]}}`)
	var page activities.SplitOptions
	options.Decode(t, &page)
	for _, d := range page.Destinations {
		if d.Initiative.ID == "I-1004" && d.MoveProblem == nil {
			t.Errorf("Retail events is offered without its dates reason: %+v", d)
		}
	}
}

// The new activity copies the original's type, owner, dates, workflow, description, external name and Shell
// attributes, with no goals ("Goals stay on …") and status Created (FR-028).
func TestSplit_CreatesCopyOfFieldsWithoutGoals_FR028(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	var original activities.View
	priya.Do(t, http.MethodGet, "/api/v1/activities/A-1001", nil).Decode(t, &original)

	split := priya.Do(t, http.MethodPost, "/api/v1/activities/A-1001/split",
		splitBody("Launch film (split)", "I-1001", nil, "20000.00", false, original.Version))

	assertStatus(t, split, http.StatusCreated)
	var result activities.SplitResult
	split.Decode(t, &result)
	n := result.NewActivity
	if n.ActivityType.ID != original.ActivityType.ID || !reflect.DeepEqual(n.Owner, original.Owner) ||
		!reflect.DeepEqual(n.StartDate, original.StartDate) || !reflect.DeepEqual(n.EndDate, original.EndDate) ||
		!reflect.DeepEqual(n.ApprovalWorkflow, original.ApprovalWorkflow) || n.Description != original.Description ||
		n.ExternalName != original.ExternalName || !reflect.DeepEqual(n.Attributes, original.Attributes) {
		t.Errorf("new activity %+v does not copy %+v", n, original)
	}
	if n.Goals.INQ != nil || n.Goals.MQL != nil || n.Goals.Revenue != nil || n.Status != activities.StatusCreated ||
		n.PlannedCost.Format() != "20,000.00 USD" || result.Activity.PlannedCost.Format() != "100,000.00 USD" {
		t.Errorf("new activity goals %+v, status %s, planned %v; original planned %v", n.Goals, n.Status, n.PlannedCost, result.Activity.PlannedCost)
	}
	if !reflect.DeepEqual(result.Activity.Goals, original.Goals) {
		t.Errorf("the original's goals changed: %+v, want %+v", result.Activity.Goals, original.Goals)
	}
}

// Splitting the Approved Social always-on needs the person's confirmation and returns it to Created (FR-029,
// US4-3); the moved line item is ready to send no more, because its new activity is Created.
func TestSplit_ApprovedNeedsRestart_FR029(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	var options activities.SplitOptions
	priya.Do(t, http.MethodGet, "/api/v1/activities/A-1002/split-options", nil).Decode(t, &options)
	if !options.NeedsRestart {
		t.Fatalf("options = %+v, want needsRestart", options)
	}

	unconfirmed := priya.Do(t, http.MethodPost, "/api/v1/activities/A-1002/split",
		splitBody("Speakers", "I-1001", []string{"L-1004"}, "5000.00", false, options.Version))
	confirmed := priya.Do(t, http.MethodPost, "/api/v1/activities/A-1002/split",
		splitBody("Speakers", "I-1001", []string{"L-1004"}, "5000.00", true, options.Version))

	assertAnswer(t, unconfirmed, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"restartApproval","message":"This activity will need approval again — confirm the change to save it."}]}}`)
	assertStatus(t, confirmed, http.StatusCreated)
	var result activities.SplitResult
	confirmed.Decode(t, &result)
	if result.Activity.Status != activities.StatusCreated || result.NewActivity.Status != activities.StatusCreated ||
		result.Activity.PlannedCost.Format() != "75,000.00 USD" {
		t.Errorf("after the split: original %s %v, new %s", result.Activity.Status, result.Activity.PlannedCost, result.NewActivity.Status)
	}
	var moved struct {
		LineItem struct{ SwbsStatus string } `json:"lineItem"`
	}
	priya.Do(t, http.MethodGet, "/api/v1/line-items/L-1004", nil).Decode(t, &moved)
	if moved.LineItem.SwbsStatus == "readyToSend" {
		t.Errorf("L-1004 SWBS = %s, want it to follow its new Created activity", moved.LineItem.SwbsStatus)
	}
}

// A split is logged on the original ("split", what moved, planned cost old → new), on the new activity
// (created, "Split from …") and on each moved line item (moved) (FR-030).
func TestSplit_Logs_FR030(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	split := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split",
		splitBody("Ireland roadshow", irelandInit, []string{agencyItem}, "19750.00", false, 1))
	assertStatus(t, split, http.StatusCreated)

	var page history.LogPage
	priya.Do(t, http.MethodGet, "/api/v1/activity-log?limit=10", nil).Decode(t, &page)
	var lines []string
	for _, e := range page.Items {
		line := strings.TrimSpace(string(e.Action) + " " + e.Verb + " " + e.RecordID + " " + e.Details)
		for _, c := range e.Changes {
			line += " | " + c.Label + ": " + c.OldText + " → " + c.NewText
		}
		lines = append(lines, strings.TrimSpace(line))
	}
	want := []string{
		"changed split A-1011 Into 'Ireland roadshow' (A-1014): 1 line item and 19,750.00 GBP | Planned cost: 40,000.00 GBP → 20,250.00 GBP",
		"moved moved L-1010 | Activity: 'UK launch event' (A-1011) → 'Ireland roadshow' (A-1014)",
		"created created A-1014 Split from 'UK launch event' (A-1011)",
	}
	if len(lines) < 3 || !slices.Equal(lines[:3], want) {
		t.Errorf("log =\n%s\nwant first\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

// When the original goes to the recycle bin, the new activity's link says so ("deleted"); a copy of the new
// activity carries no link (FR-031).
func TestSplitLinks_DeletedOriginal_FR031(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	split := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split",
		splitBody("Ireland roadshow", irelandInit, []string{agencyItem}, "19750.00", false, 1))
	var result activities.SplitResult
	split.Decode(t, &result)

	deleted := priya.Do(t, http.MethodDelete, "/api/v1/activities/"+ukLaunchEvent, nil)
	assertStatus(t, deleted, http.StatusOK)
	var created activities.View
	priya.Do(t, http.MethodGet, "/api/v1/activities/"+result.NewActivity.ID, nil).Decode(t, &created)

	if created.SplitFrom == nil || *created.SplitFrom != (activities.SplitLink{ID: ukLaunchEvent, Name: "UK launch event", PlanID: "P-1004", Deleted: true}) {
		t.Errorf("split from = %+v, want UK launch event, deleted", created.SplitFrom)
	}
}

// A copy never carries the split link (FR-031, R13).
func TestCopy_DoesNotCarrySplitFrom_FR031(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	split := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split",
		splitBody("Ireland roadshow", irelandInit, []string{agencyItem}, "19750.00", false, 1))
	var result activities.SplitResult
	split.Decode(t, &result)

	copied := priya.Do(t, http.MethodPost, "/api/v1/activities/"+result.NewActivity.ID+"/copy", map[string]any{"initiativeId": irelandInit})

	assertStatus(t, copied, http.StatusCreated)
	var copy activities.View
	copied.Decode(t, &copy)
	if copy.SplitFrom != nil || len(copy.SplitInto) != 0 {
		t.Errorf("copy links: from %+v, into %+v; want none", copy.SplitFrom, copy.SplitInto)
	}
}
