package allwork

import (
	"slices"
	"testing"
)

// Yearly gives one column per year of the range (FR-035).
func TestPeriods_Yearly_FR035(t *testing.T) {
	t.Parallel()
	r := rangeView(twoYears(), Query{View: ViewBudget, Period: PeriodYear}, 2026, 2027).result

	if got := periodKeys(r); !slices.Equal(got, []string{"2026", "2027"}) {
		t.Fatalf("periods = %q", got)
	}
	if got := amountsText(r, r.Totals.Periods, r.Totals.Total); got != "2026 16000.00 · 2027 11000.00 · = 27000.00 USD" {
		t.Errorf("footer = %s", got)
	}
}

// periodKeys returns the keys of the period columns.
func periodKeys(r Result) []string {
	keys := []string{}
	for _, p := range r.Periods {
		keys = append(keys, p.Key)
	}
	return keys
}

// Quarterly over two years has the keys Q1-2026 … Q4-2027 and the labels "Q1 2026" … (FR-035).
func TestPeriods_QuarterKeysAcrossYears_FR035(t *testing.T) {
	t.Parallel()
	r := rangeView(twoYears(), Query{View: ViewBudget, Period: PeriodQuarter}, 2026, 2027).result

	want := []string{"Q1-2026", "Q2-2026", "Q3-2026", "Q4-2026", "Q1-2027", "Q2-2027", "Q3-2027", "Q4-2027"}
	if got := periodKeys(r); !slices.Equal(got, want) || r.Periods[4].Label != "Q1 2027" {
		t.Errorf("periods = %q (label %q)", got, r.Periods[4].Label)
	}
	monthly := rangeView(twoYears(), Query{View: ViewBudget, Period: PeriodMonth}, 2026, 2027).result
	if len(monthly.Periods) != 24 || monthly.Periods[14].Label != "Mar 2027" || monthly.Periods[14].Key != "2027-03" {
		t.Errorf("monthly = %d periods, the 15th %+v", len(monthly.Periods), monthly.Periods[14])
	}
}

// Amounts outside the range are under Other years (FR-035).
func TestPeriods_OtherYearsOutsideRange_FR035(t *testing.T) {
	t.Parallel()
	nodes := twoYears()
	for i := range nodes {
		if nodes[i].ID == "L-1" { // part of Agency in 2028
			nodes[i].MonthsUSD = []MonthAmount{{Month: "2026-03", USD: amountOf("3000.00")}, {Month: "2028-01", USD: amountOf("1000.00")}}
		}
	}

	r := rangeView(nodes, Query{View: ViewBudget, Period: PeriodYear}, 2026, 2027).result

	if got := periodKeys(r); !slices.Equal(got, []string{"2026", "2027", "other"}) {
		t.Errorf("periods = %q, want the two years and Other years", got)
	}
}

// Compare with phased budget gives plans, campaigns and initiatives their phased budget per period and the
// periods where the planned cost is above it (FR-038): P-2001 is phased at 1,000.00 a month; March holds A-3's
// 8,000.00 and June the 3,000.00 of A-4's line item.
func TestCompare_PhasedBudgetAndOverPeriods_FR038(t *testing.T) {
	t.Parallel()
	var year2027 []Node
	for _, n := range twoYears() {
		if n.PlanID == "P-2001" {
			year2027 = append(year2027, n)
		}
	}
	r := rangeView(year2027, Query{View: ViewBudget, Period: PeriodMonth, ComparePhased: true}, 2027, 2027).result

	for _, row := range r.Rows {
		switch row.Key {
		case "P-2001":
			if len(row.PhasedBudget) != 12 || row.PhasedBudget[0].Format() != "1,000.00 USD" || !slices.Equal(row.OverPeriods, []int{2, 5}) {
				t.Errorf("P-2001 phased %v, over %v; want 1,000.00 a month, March and June over", row.PhasedBudget, row.OverPeriods)
			}
		case "C-2":
			if len(row.PhasedBudget) != 12 || row.PhasedBudget[0] != nil || len(row.OverPeriods) != 0 {
				t.Errorf("C-2, not phased: %v, %v", row.PhasedBudget, row.OverPeriods)
			}
		case "A-3":
			if row.PhasedBudget != nil {
				t.Errorf("activities have no phased budget: %v", row.PhasedBudget)
			}
		}
	}
}
