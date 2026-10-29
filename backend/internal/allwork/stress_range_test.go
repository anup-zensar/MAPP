package allwork

import (
	"fmt"
	"testing"
	"time"

	"mapp/internal/platform/race"
)

const (
	rangeFrom = 2026
	rangeTo   = 2030
)

// stressRangeYear is one year of the 016 SC-006 range: 20 plans of 5 campaigns of 5 initiatives of 10
// activities (5,000 activities) with two line items each; IDs carry the year so the five years never meet.
func stressRangeYear(year int) YearData {
	b := newYear()
	owners := []*Owner{priya, daniel, mei, sara}
	inYear := func(n *Node) { n.PlanYear = year }
	whole := dates(fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-12-31", year))
	for p := range 20 {
		plan := b.plan(fmt.Sprintf("P-%d%03d", year, p), fmt.Sprintf("FY%d Plan %02d", year, p), budget("5000000.00"),
			owner(owners[p%4]), teams("T-1001"), units("SO-1002"), inYear)
		for c := range 5 {
			campaign := b.campaign(plan, fmt.Sprintf("C-%d%04d", year, p*5+c), fmt.Sprintf("Campaign %d", c), whole,
				budget("900000.00"), timing("running"))
			for i := range 5 {
				n := (p*5+c)*5 + i
				initiative := b.initiative(campaign, fmt.Sprintf("I-%d%04d", year, n), fmt.Sprintf("Initiative %d", i), whole,
					budget("150000.00"), timing("running"))
				for a := range 10 {
					m := n*10 + a
					activity := b.activity(initiative, fmt.Sprintf("A-%d%05d", year, m), fmt.Sprintf("Activity %d", m),
						owner(owners[m%4]), whole, planned("12000.00"), status("approved"), activityType("AT-1002"))
					b.lineItem(activity, fmt.Sprintf("L-%d%05d-1", year, m), "Media", "6000.00")
					b.lineItem(activity, fmt.Sprintf("L-%d%05d-2", year, m), "Agency", "6000.00")
				}
			}
		}
	}
	return b.data()
}

// A five-year range with 25,000 activities shows its first grid page within the second 010 SC-004 allows for
// one year (016 SC-006). Timed three times, the fastest counts, as in TestQuery_StressYear_SC004.
func TestQuery_StressFiveYears_016_SC006(t *testing.T) {
	if testing.Short() || race.Enabled {
		t.Skip("a timing test: it runs without -short and without the race detector, in CI's Timing tests step")
	}
	src := newFakeSource(newYear())
	src.years = map[int]YearData{}
	for year := rangeFrom; year <= rangeTo; year++ {
		src.years[year] = stressRangeYear(year)
	}
	s, _ := newTestService(t, src)
	query := Query{View: ViewGrid, YearFrom: rangeFrom, YearTo: rangeTo, Strategic: StrategicInclude,
		ShowLevels: ShowEverything, Period: PeriodYear}

	var r Result
	elapsed := time.Hour
	for range 3 {
		start := time.Now()
		var err error
		r, err = s.Query(t.Context(), priyaActs, query)
		elapsed = min(elapsed, time.Since(start))
		if err != nil {
			t.Fatalf("query: %v", err)
		}
	}

	if r.Count.Total != 25000 || len(r.Rows) == 0 {
		t.Errorf("count %+v, %d rows", r.Count, len(r.Rows))
	}
	if elapsed > time.Second {
		t.Errorf("five years took %s, want under 1 second", elapsed)
	}
	t.Logf("five years: %d rows in %s", len(r.Rows), elapsed)
}
