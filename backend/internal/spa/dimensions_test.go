package spa

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func rowTexts(b DimensionBreakdown) []string {
	texts := make([]string, 0, len(b.Rows))
	for _, r := range b.Rows {
		texts = append(texts, r.Label+" "+r.Amount.Amount.String()+" "+r.Share+"%")
	}
	return texts
}

func TestDimension_FR018_ASplitDividesThePlanned(t *testing.T) {
	t.Parallel()
	b := newFixture(t).view(t, Query{CampaignID: "C-1", Dimension: "country"}).Dimension
	want := []string{"China 264000.00 66.00%", "India 136000.00 34.00%"}
	if diff := cmp.Diff(want, rowTexts(b)); diff != "" || b.Total != usd(t, "400000.00") || b.CountsInEach {
		t.Errorf("rows (-want +got):\n%s total %v", diff, b.Total)
	}
	if b.Rows[0].Bar != 100 || b.Rows[1].Bar != 52 {
		t.Errorf("bars %d, %d", b.Rows[0].Bar, b.Rows[1].Bar)
	}
}

func TestDimension_FR018_TheDefaultIsChannelAudience(t *testing.T) {
	t.Parallel()
	b := newFixture(t).view(t, Query{}).Dimension
	want := []string{"External 340000.00 85.00%", "Not filled in 60000.00 15.00%"}
	if diff := cmp.Diff(want, rowTexts(b)); diff != "" || b.Key != "channel" {
		t.Errorf("rows (-want +got):\n%s key %s", diff, b.Key)
	}
}

func TestDimension_FR018_ALineitemDimensionShowsThePlanned(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.record("L-1").PlannedCost = ptr(amt(t, "160000.00"))
	f.record("L-1").Spread = []MonthAmount{{Month: "2026-07", USD: amt(t, "160000.00")}}
	f.record("L-1").Values["spendType"] = []Value{{Label: "Brand"}}
	f.record("L-2").Values["spendType"] = []Value{{Label: "Brand"}}
	f.record("L-3").Values["spendType"] = []Value{{Label: "Media"}}
	b := f.view(t, Query{Dimension: "spendType"}).Dimension
	want := []string{"Brand 300000.00 75.00%", "Media 60000.00 15.00%", "Not yet itemised 40000.00 10.00%"}
	if diff := cmp.Diff(want, rowTexts(b)); diff != "" {
		t.Errorf("rows (-want +got):\n%s", diff)
	}
}

func TestDimension_FR018_SeveralValuesWithoutSharesCountIn(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.record("A-2").Values["channel"] = []Value{{Label: "External"}, {Label: "Partners"}}
	b := f.view(t, Query{}).Dimension
	want := []string{"External 400000.00 100.00%", "Partners 60000.00 15.00%"}
	if diff := cmp.Diff(want, rowTexts(b)); diff != "" || !b.CountsInEach {
		t.Errorf("rows (-want +got):\n%s counts in each %v", diff, b.CountsInEach)
	}
}

func TestDimension_FR018_ADimensionThatIsNotOffered(t *testing.T) {
	t.Parallel()
	_, err := newFixture(t).service().View(t.Context(), planner, "P-1", Query{Dimension: "attr:SA-9999"})
	assertField(t, err, fieldDimension)
}

func TestShareOfAmount(t *testing.T) {
	t.Parallel()
	if got := shareOfAmount(amt(t, "0.05"), *pct(t, "50.00")); got != amt(t, "0.03") {
		t.Errorf("013 FR-018 50%% of 0.05 = %s, want 0.03 (half up)", got)
	}
}
