package activities_test

import (
	"fmt"
	"strings"
	"testing"

	"northstar/internal/activities"
	"northstar/internal/plans"
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
// follow every create, edit, delete and Undo (FR-005, D-105; 004's activity level).
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
}
