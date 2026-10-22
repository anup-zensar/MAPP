package lineitems

import (
	"reflect"
	"testing"

	"mapp/internal/history"
)

// Moving a line item to another activity (a split, 016 FR-028) changes only its activity: its ID, currency,
// planned cost, spread and SWBS request stay as they are.
func TestMoveToActivity_KeepsIDsValuesSpreads_FR028(t *testing.T) {
	f := newTestService(t)
	agency := f.create(t, priya, launchFilm, input("Agency fee", "EUR", "23000.00"))
	before := f.stored(t)[0]

	if err := f.service.MoveToActivity(t.Context(), priya, []string{agency.ID}, socialOnAir); err != nil {
		t.Fatalf("MoveToActivity: %v", err)
	}

	after := f.stored(t)[0]
	if after.ID != agency.ID || after.ActivityID != socialOnAir {
		t.Fatalf("after the move: %s under %s, want %s under %s", after.ID, after.ActivityID, agency.ID, socialOnAir)
	}
	before.ActivityID, before.Meta = after.ActivityID, after.Meta
	if !reflect.DeepEqual(before, after) {
		t.Errorf("values changed by the move:\nbefore %+v\nafter  %+v", before, after)
	}
}

// Each moved line item is logged once, "moved", with its old and its new activity (FR-030).
func TestMoveToActivity_LogsMoved_FR030(t *testing.T) {
	f := newTestService(t)
	agency := f.create(t, priya, launchFilm, input("Agency fee", "EUR", "23000.00"))

	if err := f.service.MoveToActivity(t.Context(), priya, []string{agency.ID}, socialOnAir); err != nil {
		t.Fatalf("MoveToActivity: %v", err)
	}

	entry := f.log(t)[0]
	if entry.Action != history.ActionMoved || entry.RecordID != agency.ID || len(entry.Changes) != 1 ||
		entry.Changes[0].Label != "Activity" || string(entry.Changes[0].Old) != `"`+launchFilm+`"` ||
		string(entry.Changes[0].New) != `"`+socialOnAir+`"` {
		t.Errorf("log entry = %+v", entry)
	}
}

// A moved line item's SWBS request status follows its new activity: from a Created activity (waiting for
// approval) to an Approved one (ready to send) (FR-029).
func TestMoveToActivity_SwbsFollowsNewActivity_FR029(t *testing.T) {
	f := newTestService(t)
	agency := f.create(t, priya, launchFilm, input("Agency fee", "EUR", "23000.00"))
	if agency.SwbsStatus != SwbsWaitingForApproval {
		t.Fatalf("before the move: %s, want waitingForApproval", agency.SwbsStatus)
	}

	if err := f.service.MoveToActivity(t.Context(), priya, []string{agency.ID}, socialOnAir); err != nil {
		t.Fatalf("MoveToActivity: %v", err)
	}

	details, err := f.service.Get(t.Context(), priya, agency.ID)
	if err != nil || details.LineItem.SwbsStatus != SwbsReadyToSend || details.Activity.ID != socialOnAir {
		t.Errorf("after the move: %s under %s, %v; want readyToSend under %s", details.LineItem.SwbsStatus, details.Activity.ID, err, socialOnAir)
	}
}

// A line item that is not stored is refused and nothing moves.
func TestMoveToActivity_UnknownLineItemRefused(t *testing.T) {
	f := newTestService(t)
	agency := f.create(t, priya, launchFilm, input("Agency fee", "EUR", "23000.00"))

	err := f.service.MoveToActivity(t.Context(), priya, []string{agency.ID, "L-9999"}, socialOnAir)

	if err == nil || f.stored(t)[0].ActivityID != launchFilm {
		t.Errorf("err = %v, activity %s; want a refusal and the line item still under %s", err, f.stored(t)[0].ActivityID, launchFilm)
	}
}

// The Split dialog lists the activity's line items with their planned cost in the plan currency (016 FR-026).
func TestBriefs_PlanCurrency_FR026(t *testing.T) {
	f := newTestService(t)
	withGBPActivity(f)
	f.create(t, priya, ukLaunchEvent, input("Venue", "GBP", "15000.00"))
	f.create(t, priya, ukLaunchEvent, input("Agency fee (Ireland)", "EUR", "23000.00"))

	briefs, err := f.service.Briefs(t.Context(), ukLaunchEvent)

	if err != nil || len(briefs) != 2 || briefs[0].Name != "Venue" || briefs[1].PlannedCost.Format() != "23,000.00 EUR" ||
		briefs[1].PlannedCostPlan.Format() != "19,750.00 GBP" {
		t.Errorf("briefs = %+v, %v", briefs, err)
	}
}
