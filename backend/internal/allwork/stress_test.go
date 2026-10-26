package allwork

import (
	"fmt"
	"testing"
	"time"

	"mapp/internal/platform/race"
)

// stressYear is a year of the size of SC-004: 20 plans, each with 5 campaigns of 5 initiatives of 10
// activities — 5,000 activities — and 4 line items per activity — 20,000 line items.
func stressYear() *yearBuilder {
	b := newYear()
	statuses := []string{"created", "submitted", "declined", "approved", "inProgress", "complete"}
	owners := []*Owner{priya, daniel, mei, sara}
	for p := range 20 {
		plan := b.plan(fmt.Sprintf("P-%d", 1001+p), fmt.Sprintf("FY2026 Plan %02d", p), budget("5000000.00"), owner(owners[p%4]),
			teams("T-1001"), units("SO-1002"))
		for c := range 5 {
			campaign := b.campaign(plan, fmt.Sprintf("C-%d", 1001+p*5+c), fmt.Sprintf("Campaign %d", c), owner(owners[c%4]),
				dates("2026-01-01", "2026-12-31"), budget("900000.00"), progress("inProgress", 3, 50), timing("running"))
			for i := range 5 {
				n := (p*5+c)*5 + i
				initiative := b.initiative(campaign, fmt.Sprintf("I-%d", 1001+n), fmt.Sprintf("Initiative %d", i),
					owner(owners[i%4]), dates("2026-01-01", "2026-12-31"), budget("150000.00"), progress("inProgress", 1, 10),
					timing("running"))
				for a := range 10 {
					m := n*10 + a
					start := fmt.Sprintf("2026-%02d-01", m%12+1)
					activity := b.activity(initiative, fmt.Sprintf("A-%d", 1001+m), fmt.Sprintf("Activity %d launch", m),
						owner(owners[m%4]), dates(start, "2026-12-31"), planned("12000.00"), status(statuses[m%6]),
						activityType(fmt.Sprintf("AT-%d", 1001+m%6)), describe("A piece of planned marketing work"),
						goals(1000, 20, "200000.00"))
					for l := range 4 {
						b.lineItem(activity, fmt.Sprintf("L-%d", 1001+m*4+l), fmt.Sprintf("Line item %d", l), "2500.00", owner(owners[l%4]))
					}
				}
			}
		}
	}
	return b
}

// With 5,000 activities and 20,000 line items in one year, the grid with every Standard column, Budget &
// cost, the Calendar, and a filtered, searched and sorted grid each answer in under 1 second (SC-004). Each
// is timed three times and the fastest counts, so that other test binaries running at the same time (go test
// ./... runs packages in parallel) do not make it fail.
func TestQuery_StressYear_SC004(t *testing.T) {
	if testing.Short() || race.Enabled {
		t.Skip("a timing test: it runs without -short and without the race detector, in CI's Timing tests step")
	}
	s, _ := newTestService(t, newFakeSource(stressYear()))
	standard := Query{View: ViewGrid, Strategic: StrategicInclude, ShowLevels: ShowEverything, Period: PeriodMonth}
	queries := map[string]func(q Query) Query{
		"grid":     func(q Query) Query { q.Columns = allColumns(); return q },
		"budget":   func(q Query) Query { q.View = ViewBudget; return q },
		"calendar": func(q Query) Query { q.View = ViewCalendar; return q },
		"filtered, searched and sorted": func(q Query) Query {
			q.ActivityStatuses, q.Owners, q.Search = []string{"declined", "approved"}, []string{daniel.ID}, "LAUNCH"
			q.Sort, q.ShowLevels = &Sort{Column: "plannedCost", Direction: SortDescending}, ShowCompleteBranches
			return q
		},
	}
	for name, change := range queries {
		var r Result
		elapsed := time.Hour
		for range 3 {
			start := time.Now()
			var err error
			r, err = s.Query(t.Context(), priyaActs, change(standard))
			elapsed = min(elapsed, time.Since(start))
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		}

		if r.Count.Total != 5000 || len(r.Rows) == 0 {
			t.Errorf("%s: count %+v, %d rows", name, r.Count, len(r.Rows))
		}
		if elapsed > time.Second {
			t.Errorf("%s took %s, want under 1 second", name, elapsed)
		}
		t.Logf("%s: %d rows in %s", name, len(r.Rows), elapsed)
	}
}
