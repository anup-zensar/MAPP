package spa

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHeadline_FR015_TheCampaignsPlannedCommittedActualRemaining(t *testing.T) {
	t.Parallel()
	h := newFixture(t).view(t, Query{CampaignID: "C-1"}).Headline
	got := []string{h.PlannedCost.Format(), moneyText(h.Committed), moneyText(h.ActualSpend), moneyText(h.Remaining),
		textOf(h.UtilisationPercent), textOf(h.VariancePercent)}
	want := []string{"400,000.00 USD", "200,000.00 USD", "150,000.00 USD", "250,000.00 USD", "37.5", "-62.5"}
	if diff := cmp.Diff(want, got); diff != "" || h.Overspend {
		t.Errorf("headline (-want +got):\n%s overspend %v", diff, h.Overspend)
	}
}

func TestHeadline_FR016_ActualAbovePlannedIsOverspend(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.finance.spend["L-1"]["2026-08"] = Spend{Committed: amt(t, "200000.00"), Actual: amt(t, "450000.00")}
	h := f.view(t, Query{CampaignID: "C-1"}).Headline
	got := []string{moneyText(h.Remaining), textOf(h.UtilisationPercent), textOf(h.VariancePercent)}
	if diff := cmp.Diff([]string{"-50,000.00 USD", "112.5", "12.5"}, got); diff != "" || !h.Overspend {
		t.Errorf("headline (-want +got):\n%s overspend %v", diff, h.Overspend)
	}
}

func TestHeadline_FR015_WithoutPlannedCostUtilisationAndVariance(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.record("A-1").PlannedCost, f.record("A-2").PlannedCost = nil, nil
	f.finance.spend = map[string]map[string]Spend{}
	h := f.view(t, Query{CampaignID: "C-1"}).Headline
	if h.UtilisationPercent != nil || h.VariancePercent != nil || textOf(h.NoPlannedCostReason) != msgNoPlannedCost {
		t.Errorf("headline = %+v", h)
	}
}

func TestHeadline_FR012_WithoutFinanceviewTheFinancialFiguresAre(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.finance.sees = false
	v := f.view(t, Query{})
	h := v.Headline
	if h.Committed != nil || h.ActualSpend != nil || h.Remaining != nil || h.VariancePercent != nil ||
		textOf(v.FinanceNote) != msgNoFinanceAccess || h.PlannedCost != usd(t, "400000.00") || v.Activities[0].ActualSpend != nil {
		t.Errorf("view = %+v", v)
	}
}

func TestPercentOf(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, part, whole, want string }{
		{"013 FR-015 one decimal", "150000.00", "400000.00", "37.5"},
		{"013 FR-015 half away from zero", "1.00", "6.00", "16.7"},
		{"013 FR-015 negative half away from zero", "-1.00", "6.00", "-16.7"},
		{"013 FR-015 above 100", "450000.00", "400000.00", "112.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := textOf(percentOf(amt(t, tt.part), amt(t, tt.whole), 1)); got != tt.want {
				t.Errorf("percentOf(%s, %s) = %s, want %s", tt.part, tt.whole, got, tt.want)
			}
		})
	}
	if percentOf(amt(t, "1.00"), amt(t, "0.00"), 1) != nil {
		t.Error("percentOf of 0.00 is not nil")
	}
}
