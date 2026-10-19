package activities_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"northstar/internal/activities"
	"northstar/internal/plans"
	"northstar/internal/platform/date"
	"northstar/internal/platform/record"
	"northstar/internal/settings"
)

// The demo plans of these tests.
const (
	lubricantsPlan = "P-1001"
	mobilityPlan   = "P-1002"
	globalPlan     = "P-1003" // no activities
)

// planFigures asks for the activity level's facts of planIDs, each starting from zero Facts, and writes
// them as one line per plan.
func (f *fixture) planFigures(t *testing.T, planIDs ...string) map[string]string {
	t.Helper()
	facts := map[string]*plans.Facts{}
	for _, id := range planIDs {
		facts[id] = &plans.Facts{}
	}
	if err := f.service.AddPlanFacts(t.Context(), append(planIDs, "P-9999"), facts); err != nil {
		t.Fatalf("AddPlanFacts: %v", err)
	}
	figures := map[string]string{}
	for id, fact := range facts {
		planned := "—"
		if fact.PlannedCost != nil {
			planned = fact.PlannedCost.String()
		}
		var attention []string
		for _, item := range fact.NeedsAttention {
			var fields []string
			for _, m := range item.Missing {
				fields = append(fields, m.Label)
			}
			attention = append(attention, fmt.Sprintf("%s %s (%s)", item.Record.Type, item.Record.ID, strings.Join(fields, ", ")))
		}
		figures[id] = fmt.Sprintf("%d activities, %d approved, planned %s, goals %d/%d/%s; attention: %s",
			fact.Activities, fact.ApprovedActivities, planned, fact.ActivityGoals.INQ, fact.ActivityGoals.MQL,
			fact.ActivityGoals.Revenue.Amount, strings.Join(attention, "; "))
	}
	return figures
}

// The plan's figures — number of activities, approved ones, planned cost, goals as shown and the
// activities with fields still needed — are worked out from the activities when they are read, so they
// follow every create, edit, delete, Undo and copy (FR-005, D-105; 004's activity level).
func TestAddPlanFacts_FollowEveryChange_FR005(t *testing.T) {
	f := newFixture(t)

	got := f.planFigures(t, lubricantsPlan, mobilityPlan, globalPlan)

	want := map[string]string{
		// A-1002, A-1003, A-1006 and A-1007 are approved; goals 250,000 + 40,000 + 20,000 INQ, calculated
		// with Lubricants 1.0.
		lubricantsPlan: "7 activities, 4 approved, planned 460000.00, goals 310000/6200/3100000.00; " +
			"attention: activity A-1001 (Company code, Cost centre, PIR)",
		// A-1009's goals are typed; A-1010 has no planned cost.
		mobilityPlan: "3 activities, 0 approved, planned 190000.00, goals 60000/2000/400000.00; " +
			"attention: activity A-1008 (Country / OU, Cost centre); " +
			"activity A-1010 (Owner, Start date, End date, Planned cost, Approval workflow, Channel / audience, Country / OU, Company code, Cost centre, Global campaign, Growth ambition, PIR, FIT4 initiative, FIT4 ID)",
		globalPlan: "0 activities, 0 approved, planned —, goals 0/0/0.00; attention: ",
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s before:\n got %s\nwant %s", id, got[id], w)
		}
	}

	created := f.create(t, digitalAndSocial, activities.Input{Name: "Launch video", ActivityTypeID: eventType,
		OwnerID: text("U-1002"), StartDate: text("2026-03-01"), EndDate: text("2026-03-31"), PlannedCost: usdInput("25000"),
		ApprovalWorkflowID: text(lubricantsFlow)})
	f.mustEdit(t, created.ID, func(u *activities.Update) { u.PlannedCost = usdInput("30000.00") })
	after := f.planFigures(t, lubricantsPlan)[lubricantsPlan]
	if !strings.HasPrefix(after, "8 activities, 4 approved, planned 490000.00,") || !strings.Contains(after, "activity "+created.ID+" (") {
		t.Errorf("after create and edit = %s", after)
	}

	entry, err := f.service.Delete(t.Context(), priya, created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := f.planFigures(t, lubricantsPlan)[lubricantsPlan]; got != want[lubricantsPlan] {
		t.Errorf("after delete = %s", got)
	}
	if _, err := f.history.Undo(t.Context(), priya, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if got := f.planFigures(t, lubricantsPlan)[lubricantsPlan]; got != after {
		t.Errorf("after Undo = %s, want %s", got, after)
	}

	if _, err := f.service.Copy(t.Context(), priya, "A-1001", activities.CopyInput{InitiativeID: digitalAndSocial}); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if got := f.planFigures(t, lubricantsPlan)[lubricantsPlan]; !strings.HasPrefix(got, "9 activities, 4 approved, planned 610000.00,") {
		t.Errorf("after copying Launch film (120,000.00) = %s", got)
	}
}

// CopyAll copies every activity of an initiative into another for the copies of features 004 and 005
// (R13, D-195): the same names and values, every date moved by the year shift (29 February → 28 February,
// an ongoing activity keeps no end date), new IDs, the status Created, line items through feature 007, and
// no log entries of their own.
func TestCopyAll_ShiftsYears(t *testing.T) {
	f := newFixture(t)
	const events = "I-1003" // Partner co-marketing fund: Approved, ongoing from 1 Mar 2026
	leap := date.MustParse("2028-02-29")
	f.change(t, func(ctx context.Context) error {
		return activities.NewJSONRepository().Put(ctx, activities.Activity{
			ID: "A-9001", InitiativeID: events, Name: "Leap day stunt", ActivityTypeID: eventType, StartDate: &leap,
			EndDate: &leap, Status: activities.StatusSubmitted, Attributes: settings.AttributeValues{}, Meta: record.NewMeta(priya, start),
		})
	})
	partner, _ := f.stored(t, "A-1007")
	f.lineItems.facts["A-1007"] = activities.LineItemFacts{Count: 2}
	entriesBefore := len(f.entries(t))

	var counts activities.CopyCounts
	err := f.db.Write(t.Context(), func(ctx context.Context) error { // joins the caller's Write
		var err error
		counts, err = f.service.CopyAll(ctx, daniel, events, retailEvents, 1)
		return err
	})

	if err != nil || counts != (activities.CopyCounts{Activities: 2, LineItems: 2}) {
		t.Fatalf("CopyAll = %+v, %v; want 2 activities and 2 line items", counts, err)
	}
	copies := f.byName(t, retailEvents)
	if len(copies) != 4 { // Fleet roadshow, Christmas trade gifting and the two copies
		t.Fatalf("%s holds %d activities, want 4", retailEvents, len(copies))
	}
	partnerCopy := copies["Partner co-marketing fund"]
	if partnerCopy.StartDate.String() != "2027-03-01" || partnerCopy.EndDate != nil || !partnerCopy.Ongoing {
		t.Errorf("ongoing copy runs %v – %v (ongoing %v), want from 2027-03-01, still ongoing", partnerCopy.StartDate, partnerCopy.EndDate, partnerCopy.Ongoing)
	}
	if partnerCopy.Status != activities.StatusCreated || partnerCopy.Meta != record.NewMeta(daniel, start) {
		t.Errorf("copy status %s, bookkeeping %+v; want Created, new, by Daniel", partnerCopy.Status, partnerCopy.Meta)
	}
	want := partner
	want.ID, want.InitiativeID, want.Status, want.Meta = partnerCopy.ID, retailEvents, activities.StatusCreated, partnerCopy.Meta
	want.StartDate = partnerCopy.StartDate
	if !reflect.DeepEqual(partnerCopy, want) {
		t.Errorf("copy =\n%+v\nwant every value of the original\n%+v", partnerCopy, want)
	}
	if leapCopy := copies["Leap day stunt"]; leapCopy.StartDate.String() != "2029-02-28" || leapCopy.EndDate.String() != "2029-02-28" {
		t.Errorf("29 February copied to %v – %v, want 2029-02-28", leapCopy.StartDate, leapCopy.EndDate)
	}
	wantCalls := []string{"A-1007 → " + partnerCopy.ID + " +1", "A-9001 → " + copies["Leap day stunt"].ID + " +1"}
	if !reflect.DeepEqual(f.lineItems.copies, wantCalls) {
		t.Errorf("line-item copies = %q, want %q", f.lineItems.copies, wantCalls)
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("log entries = %d, want %d: the copy above writes the one entry", got, entriesBefore)
	}
}

// The activity level of a plan copy copies the activities of every copied initiative and leaves their line
// items to the line-item level, so each line item is copied once (D-195).
func TestCopyForInitiatives_LeavesLineItemsToTheirLevel_D195(t *testing.T) {
	f := newFixture(t)

	var newIDs map[string]string
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		newIDs, err = f.service.CopyForInitiatives(ctx, priya, map[string]string{
			lubricantsPlan: "P-2001", digitalAndSocial: printAndOutdoor, // a plan id matches no activity
		}, 0)
		return err
	})

	if err != nil || len(newIDs) != 2 || newIDs["A-1001"] == "" || newIDs["A-1002"] == "" {
		t.Fatalf("CopyForInitiatives = %v, %v; want the copies of A-1001 and A-1002", newIDs, err)
	}
	if launch, _ := f.stored(t, newIDs["A-1001"]); launch.Name != "Launch film" || launch.InitiativeID != printAndOutdoor {
		t.Errorf("copy of A-1001 = %q in %s, want Launch film in %s", launch.Name, launch.InitiativeID, printAndOutdoor)
	}
	if len(f.lineItems.copies) != 0 {
		t.Errorf("line-item copies = %q, want none", f.lineItems.copies)
	}
}

// byName returns the stored activities of the initiative initiativeID by name.
func (f *fixture) byName(t *testing.T, initiativeID string) map[string]activities.Activity {
	t.Helper()
	named := map[string]activities.Activity{}
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		list, err := activities.NewJSONRepository().ListByInitiative(ctx, initiativeID)
		for _, a := range list {
			named[a.Name] = a
		}
		return err
	})
	if err != nil {
		t.Fatalf("ListByInitiative(%s): %v", initiativeID, err)
	}
	return named
}
