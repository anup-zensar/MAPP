package purchaserequest

import (
	"slices"
	"testing"

	"mapp/internal/platform/money"
)

func TestGates_LineItemsAboveTheActivityPlannedCost_037_FR019(t *testing.T) {
	t.Parallel()
	over := money.Money{Amount: money.FromCents(800000), Currency: "USD"}
	facts := map[string]LineItemFacts{
		"L-1": {ID: "L-1", ActivityID: "A-1", ActivityName: "Launch film", ActivityOverBy: &over},
		"L-2": {ID: "L-2", ActivityID: "A-1", ActivityName: "Launch film", ActivityOverBy: &over},
		"L-3": {ID: "L-3", ActivityID: "A-2", ActivityName: "Trade fair"},
	}
	tests := []struct {
		name  string
		lines []string
		want  []string
	}{
		{name: "037 FR-019 line items within the planned cost pass", lines: []string{"L-3"}},
		{name: "037 FR-019 line items above it stop the submission, the activity named once", lines: []string{"L-1", "L-2", "L-3"},
			want: []string{"Activity 'Launch film': line items exceed its planned cost by 8,000.00 USD — lower them, or " +
				"raise the activity's planned cost and get it approved again."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pr := PurchaseRequest{ID: "PR-1"}
			for _, id := range tt.lines {
				pr.Lines = append(pr.Lines, Line{LineItemID: id})
			}
			var got []string
			for _, p := range overspentActivityGates(pr, facts) {
				got = append(got, p.Message)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("gates = %q, want %q", got, tt.want)
			}
		})
	}
}
