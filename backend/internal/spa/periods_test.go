package spa

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/money"
)

func periodTexts(list []Period) []string {
	texts := make([]string, 0, len(list))
	for _, p := range list {
		texts = append(texts, p.Label+" "+p.Planned.Amount.String()+" / "+moneyText(p.Actual))
	}
	return texts
}

func TestPeriods_FR017_PlannedByLineitemSpreadsAndActual(t *testing.T) {
	t.Parallel()
	p := newFixture(t).view(t, Query{CampaignID: "C-1"}).Periods
	want := []string{"Q1 2026 0.00 / 0.00 USD", "Q2 2026 0.00 / 0.00 USD", "Q3 2026 200000.00 / 150,000.00 USD",
		"Q4 2026 200000.00 / 0.00 USD"}
	if diff := cmp.Diff(want, periodTexts(p.Quarters)); diff != "" || p.NotScheduled != nil {
		t.Errorf("quarters (-want +got):\n%s not scheduled %v", diff, p.NotScheduled)
	}
	if p.Quarters[2].PlannedBar != 100 || p.Quarters[2].ActualBar != 75 || p.Quarters[0].PlannedBar != 0 {
		t.Errorf("bars = %+v", p.Quarters[2])
	}
}

func TestPeriods_FR017_TheMonthsAddUpToThe(t *testing.T) {
	t.Parallel()
	p := newFixture(t).view(t, Query{}).Periods
	for q, quarter := range p.Quarters {
		var sum money.Amount
		for _, m := range p.Months[q*3 : q*3+3] {
			sum = sum.Add(m.Planned.Amount)
		}
		if sum != quarter.Planned.Amount {
			t.Errorf("%s: months %s, quarter %s", quarter.Label, sum, quarter.Planned.Amount)
		}
	}
	if p.Months[7].Label != "Aug 2026" || p.Months[7].Key != "2026-08" || moneyText(p.Months[7].Actual) != "150,000.00 USD" {
		t.Errorf("August = %+v", p.Months[7])
	}
}

func TestPeriods_FR017_APositiveBalanceIsSpreadEvenly(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.record("A-2").PlannedCost = ptr(amt(t, "66000.00"))
	p := f.view(t, Query{}).Periods
	if p.Months[6].Planned != usd(t, "201000.00") || p.Months[10].Planned != usd(t, "61000.00") || p.NotScheduled != nil {
		t.Errorf("July %v, November %v, not scheduled %v", p.Months[6].Planned, p.Months[10].Planned, p.NotScheduled)
	}
}

func TestPeriods_FR017_AnActivityWithoutDatesIsNot(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.record("A-2").Months, f.record("A-2").PlannedCost = nil, ptr(amt(t, "66000.00"))
	f.record("L-3").Spread = nil
	p := f.view(t, Query{}).Periods
	if moneyText(p.NotScheduled) != "66,000.00 USD" || len(p.NotScheduledTrace) != 1 || p.NotScheduledTrace[0].ActivityID != "A-2" {
		t.Errorf("not scheduled %v, trace %+v", p.NotScheduled, p.NotScheduledTrace)
	}
}

func TestPeriods_FR017_LineItemsAboveThePlannedCost(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.record("A-2").PlannedCost = ptr(amt(t, "50000.00"))
	v := f.view(t, Query{})
	if moneyText(v.Periods.NotScheduled) != "-10,000.00 USD" || v.Headline.PlannedCost != usd(t, "390000.00") {
		t.Errorf("not scheduled %v, planned %v", v.Periods.NotScheduled, v.Headline.PlannedCost)
	}
}

func TestYear_FR010_PlannedCostSpreadToAnotherYear(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.record("L-3").Spread = []MonthAmount{{Month: "2026-12", USD: amt(t, "45000.00")}, {Month: "2027-01", USD: amt(t, "15000.00")}}
	v := f.view(t, Query{})
	want := []string{"15,000.00 USD of planned cost falls outside 2026 and is not counted."}
	if diff := cmp.Diff(want, v.Notes); diff != "" || v.Headline.PlannedCost != usd(t, "385000.00") {
		t.Errorf("notes (-want +got):\n%s planned %v", diff, v.Headline.PlannedCost)
	}
	if v.Activities[1].LineItems[0].PlannedCost != usd(t, "45000.00") {
		t.Errorf("L-3 in the year = %v", v.Activities[1].LineItems[0].PlannedCost)
	}
}

func TestYear_FR010_TransactionsPostedInAnotherYearAre(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.finance.spend["L-2"] = map[string]Spend{"2025-12": {Committed: amt(t, "1000.00"), Actual: amt(t, "500.00")}}
	v := f.view(t, Query{})
	want := []string{"Transactions posted outside 2026 are not counted: committed 1,000.00 USD, actual spend 500.00 USD."}
	if diff := cmp.Diff(want, v.Notes); diff != "" || moneyText(v.Headline.ActualSpend) != "150,000.00 USD" {
		t.Errorf("notes (-want +got):\n%s actual %v", diff, v.Headline.ActualSpend)
	}
}

func TestMonthIndex(t *testing.T) {
	t.Parallel()
	for month, want := range map[string]int{"2026-01": 0, "2026-12": 11, "2027-01": -1, "2026-13": -1, "2026": -1, "x-03": -1} {
		i, in := monthIndex(2026, month)
		if (want < 0 && in) || (want >= 0 && (!in || i != want)) {
			t.Errorf("013 FR-010 monthIndex(2026, %q) = %d, %v; want %d", month, i, in, want)
		}
	}
}
