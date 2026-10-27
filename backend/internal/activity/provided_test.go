package activity_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/activity"
	"mapp/internal/activity/activitytest"
	"mapp/internal/plan"
	"mapp/internal/platform/date"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

const (
	lubricantsPlan = "P-1001"
	mobilityPlan   = "P-1002"
	globalPlan     = "P-1003" // no activities
)

// planFigures writes the activity level's facts of planIDs, each from zero Facts, as one line per plan.
func (f *fixture) planFigures(t *testing.T, planIDs ...string) map[string]string {
	t.Helper()
	facts := map[string]*plan.Facts{}
	for _, id := range planIDs {
		facts[id] = &plan.Facts{}
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

// The plan's figures are worked out from the activities when read: they follow every create, edit, delete, Undo and copy.
func TestAddPlanFacts_FollowEveryChange_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	got := f.planFigures(t, lubricantsPlan, mobilityPlan, globalPlan)

	want := map[string]string{
		// A-1002, A-1003, A-1006 and A-1007 are approved; goals calculated with Lubricants 1.0.
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

	created := f.create(t, digitalAndSocial, activity.Input{Name: "Launch video", ActivityTypeID: eventType,
		OwnerID: text("U-1002"), StartDate: text("2026-03-01"), EndDate: text("2026-03-31"), PlannedCost: usdInput("25000"),
		ApprovalWorkflowID: text(lubricantsFlow)})
	f.mustEdit(t, created.ID, func(u *activity.Update) { u.PlannedCost = usdInput("30000.00") })
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

	if _, err := f.service.Copy(t.Context(), priya, "A-1001", activity.CopyInput{InitiativeID: digitalAndSocial}); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if got := f.planFigures(t, lubricantsPlan)[lubricantsPlan]; !strings.HasPrefix(got, "9 activities, 4 approved, planned 610000.00,") {
		t.Errorf("after copying Launch film (120,000.00) = %s", got)
	}
}

// CopyAll keeps names and values, moves dates by the year shift (29 February → 28 February), gives new IDs,
// the status Created and the line items, and writes no log entries of its own.
func TestCopyAll_ShiftsYears(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	const events = "I-1003" // Partner co-marketing fund: Approved, ongoing from 1 Mar 2026
	leap := date.MustParse("2028-02-29")
	f.change(t, func(ctx context.Context) error {
		return activitytest.NewStore(f.db).Put(ctx, activity.Activity{
			ID: "A-9001", InitiativeID: events, Name: "Leap day stunt", ActivityTypeID: eventType, StartDate: &leap,
			EndDate: &leap, Status: activity.StatusSubmitted, Attributes: settings.AttributeValues{}, Meta: record.NewMeta(priya, start),
		})
	})
	partner, _ := f.stored(t, "A-1007")
	f.lineItems.facts["A-1007"] = activity.LineItemFacts{Count: 2}
	entriesBefore := len(f.entries(t))

	var counts activity.CopyCounts
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		counts, err = f.service.CopyAll(ctx, daniel, events, retailEvents, 1)
		return err
	})

	if err != nil || counts != (activity.CopyCounts{Activities: 2, LineItems: 2}) {
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
	if partnerCopy.Status != activity.StatusCreated || partnerCopy.Meta != record.NewMeta(daniel, start) {
		t.Errorf("copy status %s, bookkeeping %+v; want Created, new, by Daniel", partnerCopy.Status, partnerCopy.Meta)
	}
	want := partner
	want.ID, want.InitiativeID, want.Status, want.Meta = partnerCopy.ID, retailEvents, activity.StatusCreated, partnerCopy.Meta
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

// A plan copy's activity level leaves line items to the line-item level, so each line item is copied once.
func TestCopyForInitiatives_LeavesLineItemsToTheirLevel_D195(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	var newIDs map[string]string
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		newIDs, err = f.service.CopyForInitiatives(ctx, priya, map[string]string{
			lubricantsPlan: "P-2001", digitalAndSocial: printAndOutdoor,
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

func (f *fixture) byName(t *testing.T, initiativeID string) map[string]activity.Activity {
	t.Helper()
	list, err := activitytest.NewStore(f.db).ListByInitiative(t.Context(), initiativeID)
	if err != nil {
		t.Fatalf("ListByInitiative(%s): %v", initiativeID, err)
	}
	named := map[string]activity.Activity{}
	for _, a := range list {
		named[a.Name] = a
	}
	return named
}
