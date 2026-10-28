package purchaserequest

import (
	"testing"

	"mapp/internal/platform/money"
)

func TestAvailable(t *testing.T) {
	t.Parallel()
	eur, err := money.ParseRate("0.92")
	if err != nil {
		t.Fatal(err)
	}
	rates := money.Rates{"USD": money.One, "EUR": eur}
	stand := LineItemFacts{ID: "L-1006", Currency: "EUR", PlannedCost: amount(t, "9200.00")}
	media := LineItemFacts{ID: "L-1005", Currency: "USD", PlannedCost: amount(t, "4000.00")}
	claim := func(pr, li, cur, value string) Claim {
		return Claim{PurchaseRequestID: pr, LineItemID: li, Amount: money.Money{Amount: amount(t, value), Currency: cur}}
	}
	tests := []struct {
		name   string
		li     LineItemFacts
		claims []Claim
		amount money.Money
		except string
		want   string
	}{
		{name: "032 FR-012 nothing claimed, the whole planned cost", li: stand, amount: money.Money{Amount: amount(t, "9200.00"), Currency: "EUR"}},
		{name: "032 FR-012 3,200.00 left after 6,000.00 fits", li: stand, claims: []Claim{claim("PR-1", "L-1006", "EUR", "6000.00")},
			amount: money.Money{Amount: amount(t, "3200.00"), Currency: "EUR"}},
		{name: "032 FR-012 3,200.01 does not", li: stand, claims: []Claim{claim("PR-1", "L-1006", "EUR", "6000.00")},
			amount: money.Money{Amount: amount(t, "3200.01"), Currency: "EUR"},
			want:   "Only 3,200.00 EUR of line item L-1006 is not yet requested."},
		{name: "032 FR-012 another currency converts at today's rate", li: media,
			amount: money.Money{Amount: amount(t, "3700.00"), Currency: "EUR"},
			want:   "Only 4,000.00 USD of line item L-1005 is not yet requested. 3,700.00 EUR is 4,021.74 USD at today's rate."},
		{name: "032 FR-017 the purchase request being saved does not count against itself", li: stand,
			claims: []Claim{claim("PR-1", "L-1006", "EUR", "9200.00")}, amount: money.Money{Amount: amount(t, "9200.00"), Currency: "EUR"},
			except: "PR-1"},
		{name: "032 FR-012 claims on other line items do not count", li: stand, claims: []Claim{claim("PR-1", "L-1005", "USD", "9200.00")},
			amount: money.Money{Amount: amount(t, "9200.00"), Currency: "EUR"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := balance{rates: rates, claims: tt.claims}.claimProblem(tt.li, tt.amount, tt.except)
			if err != nil || got != tt.want {
				t.Errorf("claimProblem = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}
