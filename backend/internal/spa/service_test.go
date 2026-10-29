package spa

import (
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

func assertField(t *testing.T, err error, field string) {
	t.Helper()
	e, ok := errors.AsType[*errs.Error](err)
	if !ok || e.Status != http.StatusBadRequest || len(e.Fields) != 1 || e.Fields[0].Field != field {
		t.Errorf("error = %v, want a problem with %s", err, field)
	}
}

func assertNotFound(t *testing.T, err error) {
	t.Helper()
	if e, ok := errors.AsType[*errs.Error](err); !ok || e.Status != http.StatusNotFound {
		t.Errorf("error = %v, want not found", err)
	}
}

func TestView_FR005_TheWholePlanItsCampaignsTo(t *testing.T) {
	t.Parallel()
	v := newFixture(t).view(t, Query{})
	got := []string{string(v.Scope.Type), v.Scope.ID, v.Plan.Status, v.Campaigns[0].ID, v.Campaigns[1].ID, v.Currency}
	if diff := cmp.Diff([]string{"plan", "P-1", "published", "C-1", "C-2", "USD"}, got); diff != "" || !v.Plan.BCPIR ||
		len(v.Dimensions) != 4 || v.EmptyMessage != nil {
		t.Errorf("view (-want +got):\n%s", diff)
	}
}

func TestView_FR012_APlanThePersonDoesNot(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.planning.found = false
	_, err := f.service().View(t.Context(), planner, "P-1", Query{})
	assertNotFound(t, err)
}

func TestView_FR012_ACampaignThatIsNotIn(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"C-404", "I-1"} {
		_, err := newFixture(t).service().View(t.Context(), planner, "P-1", Query{CampaignID: id})
		assertNotFound(t, err)
	}
}

func TestView_FR005_ACampaignWithoutActivitiesSaysSo(t *testing.T) {
	t.Parallel()
	v := newFixture(t).view(t, Query{CampaignID: "C-2"})
	if textOf(v.EmptyMessage) != msgCampaignEmpty || v.Scope.Name != "Empty campaign" || len(v.Activities) != 0 ||
		v.Headline.PlannedCost != usd(t, "0.00") {
		t.Errorf("view = %+v", v)
	}
}

func TestView_FR008_AmountsInTheDisplayCurrencyEach(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.display = money.DisplayIn("EUR", "0.92")
	v := f.view(t, Query{})
	if v.Currency != "EUR" || v.Headline.PlannedCost.Format() != "368,000.00 EUR" ||
		moneyText(v.Headline.ActualSpend) != "138,000.00 EUR" {
		t.Errorf("headline = %+v", v.Headline)
	}
}

func TestView_FR009_WhenTheFinanceDataWasReceived(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	asOf := time.Date(2026, 10, 21, 14, 5, 0, 0, time.UTC)
	f.finance.asOf = &asOf
	if v := f.view(t, Query{}); v.FinanceDataAsOf == nil || !v.FinanceDataAsOf.Equal(asOf) || v.FinanceNote != nil {
		t.Errorf("as of %v, note %v", v.FinanceDataAsOf, v.FinanceNote)
	}
}

func TestView_FR019_TheActivityRows(t *testing.T) {
	t.Parallel()
	row := newFixture(t).view(t, Query{}).Activities[0]
	got := []string{row.Campaign.Name, row.Initiative.Name, textOf(row.Owner), textOf(row.ActivityType), textOf(row.Channel),
		textOf(row.StartQuarter), row.Status, row.PlannedCost.Format(), moneyText(row.Committed), textOf(row.VariancePercent),
		row.LineItems[0].ID, moneyText(row.LineItems[0].ActualSpend), moneyText(row.LineItems[1].ActualSpend)}
	want := []string{"Brand", "Awareness", "Priya Raman", "Event", "External", "Q3 2026", "approved", "340,000.00 USD",
		"200,000.00 USD", "-55.9", "L-1", "150,000.00 USD", "0.00 USD"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("row (-want +got):\n%s", diff)
	}
}

func TestTraceSums_FR007_SC002_EveryListAddsUpToItsFigure(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.record("A-2").PlannedCost = ptr(amt(t, "61234.57"))
	f.record("L-3").Spread = []MonthAmount{{Month: "2026-11", USD: amt(t, "30000.00")}, {Month: "2027-02", USD: amt(t, "30000.00")}}
	f.finance.spend["L-3"] = map[string]Spend{"2026-12": {Committed: amt(t, "999.99"), Actual: amt(t, "12.34")}}
	v := f.view(t, Query{Dimension: "country"})
	check := func(name string, figure money.Money, trace []Contribution) {
		var sum money.Amount
		for _, c := range trace {
			sum = sum.Add(c.Amount.Amount)
		}
		if sum != figure.Amount {
			t.Errorf("013 FR-007 %s: trace adds up to %s, figure %s", name, sum, figure.Amount)
		}
	}
	check("planned cost", v.Headline.PlannedCost, v.Headline.PlannedCostTrace)
	check("committed", *v.Headline.Committed, v.Headline.CommittedTrace)
	check("actual spend", *v.Headline.ActualSpend, v.Headline.ActualSpendTrace)
	for _, p := range slices.Concat(v.Periods.Quarters, v.Periods.Months) {
		check(p.Label+" planned", p.Planned, p.PlannedTrace)
		check(p.Label+" actual", *p.Actual, p.ActualTrace)
	}
	for _, r := range v.Dimension.Rows {
		check(r.Label, r.Amount, r.Trace)
	}
}

// The plan reader returns live records only; GSAP keeps the transactions of records moved to the recycle bin.
func TestView_FR011_TransactionsOfRecordsInTheRecycleBinAreLeftOut(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.finance.spend["L-4"] = map[string]Spend{"2026-08": {Committed: amt(t, "7000.00"), Actual: amt(t, "7000.00")}}
	f.finance.spend["L-9"] = map[string]Spend{"2026-09": {Committed: amt(t, "9000.00"), Actual: amt(t, "9000.00")}}
	v := f.view(t, Query{})
	got := []string{moneyText(v.Headline.Committed), moneyText(v.Headline.ActualSpend), moneyText(v.Periods.Quarters[2].Actual)}
	want := []string{"200,000.00 USD", "150,000.00 USD", "150,000.00 USD"}
	if diff := cmp.Diff(want, got); diff != "" || len(v.Activities) != 2 || len(v.Headline.ActualSpendTrace) != 1 {
		t.Errorf("figures (-want +got):\n%s activities %d, trace %+v", diff, len(v.Activities), v.Headline.ActualSpendTrace)
	}
}
