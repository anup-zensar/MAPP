package lineitem

import (
	"testing"

	"mapp/internal/platform/money"
)

// rate returns the rate of a text known to be right, such as "0.92".
func rate(s string) money.Rate {
	r, err := money.ParseRate(s)
	if err != nil {
		panic(err)
	}
	return r
}

// amounts returns the amounts of texts known to be right.
func amounts(texts ...string) []money.Amount {
	list := make([]money.Amount, 0, len(texts))
	for _, s := range texts {
		list = append(list, amount(s))
	}
	return list
}

// withPlannedCost returns Launch film's facts with the planned cost cost ("" for none).
func withPlannedCost(cost string) ActivityFacts {
	f := newFakeActivities().facts[launchFilm]
	f.PlannedCost = nil
	if cost != "" {
		a := amount(cost)
		f.PlannedCost = &a
	}
	return f
}

// The USD planned cost is the planned cost ÷ the currency's rate, rounded half-up to the cent (domain model
// §4.3, the plan's shared example table).
func TestPlannedCostUSD_HalfUp_FR011_SC003(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cost, rate, want string
	}{
		{"46000.00", "0.92", "50000.00"},
		{"10000.00", "0.92", "10869.57"},
		{"15800.00", "0.79", "20000.00"},
		{"1000.00", "0.92", "1086.96"},
		{"50000.00", "1", "50000.00"},
		{"0.00", "7.10", "0.00"},
		{"0.01", "2", "0.01"}, // 0.005 rounds up
		{"0.03", "2", "0.02"}, // 0.015 rounds up
		{"0.01", "7.10", "0.00"},
	}
	for _, tt := range tests {
		got := PlannedCostUSD(LineItem{PlannedCost: amount(tt.cost)}, rate(tt.rate))

		if got.String() != tt.want {
			t.Errorf("%s at %s = %s USD, want %s", tt.cost, tt.rate, got, tt.want)
		}
	}
}

// Unassigned planned cost = the activity's planned cost − the sum of its line items' USD planned costs, to
// the cent (FR-002, SC-004): 120,000.00 − (50,000.00 + 50,000.00) = 20,000.00 USD.
func TestBalance_Formula_FR002_SC004(t *testing.T) {
	t.Parallel()
	tests := []struct {
		planned string
		usd     []money.Amount
		want    string
	}{
		{"120000.00", amounts("50000.00", "50000.00"), "20000.00"},
		{"120000.00", amounts(), "120000.00"},
		{"0.10", amounts("0.03", "0.03", "0.03"), "0.01"},
		{"30000.00", amounts("10869.57", "9130.43"), "10000.00"},
	}
	for _, tt := range tests {
		b := Balance(withPlannedCost(tt.planned), tt.usd, []string{})

		if b.Balance == nil || b.Balance.Amount.String() != tt.want || b.PlannedCost.Amount.String() != amount(tt.planned).String() ||
			b.Assigned.Amount != money.Sum(tt.usd...) || b.Balance.Currency != "USD" {
			t.Errorf("Balance(%s, %v) = %+v, want %s", tt.planned, tt.usd, b, tt.want)
		}
	}
}

// The balance reads as left, all assigned or exceeded (FR-002, D-016): exceeded is negative.
func TestBalance_States_FR002(t *testing.T) {
	t.Parallel()
	tests := []struct {
		usd   []money.Amount
		state string
		want  string
	}{
		{amounts("100000.00"), BalanceLeft, "20000.00"},
		{amounts("50000.00", "70000.00"), BalanceAllAssigned, "0.00"},
		{amounts("50000.00", "70500.00"), BalanceExceeded, "-500.00"},
	}
	for _, tt := range tests {
		b := Balance(withPlannedCost("120000.00"), tt.usd, []string{"USD"})

		if b.State != tt.state || b.Balance.Amount.String() != tt.want {
			t.Errorf("Balance(%v) = %s %s, want %s %s", tt.usd, b.State, b.Balance.Amount, tt.state, tt.want)
		}
	}
}

// An activity without a planned cost has no balance: no amount, no "exceeded" (D-144, 007-P5); what is
// assigned is still added up.
func TestBalance_NoPlannedCost_007P5(t *testing.T) {
	t.Parallel()
	b := Balance(withPlannedCost(""), amounts("1000.00", "250.50"), []string{"USD", "EUR"})

	if b.State != BalanceNoPlannedCost || b.Balance != nil || b.PlannedCost != nil || b.RoundingHint ||
		b.Assigned != money.USD(amount("1250.50")) {
		t.Errorf("balance = %+v, want noPlannedCost with 1,250.50 USD assigned", b)
	}
}

// The rounding hint shows when a few cents (at most 0.05 USD either way) are left and a line item uses
// another currency than USD (spec edge case, D-145).
func TestBalance_RoundingHint_EdgeCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		assigned   string
		currencies []string
		hint       bool
	}{
		{"119999.99", []string{"USD", "EUR"}, true},
		{"119999.95", []string{"EUR"}, true},
		{"120000.05", []string{"EUR"}, true},
		{"119999.94", []string{"EUR"}, false},
		{"120000.06", []string{"EUR"}, false},
		{"120000.00", []string{"EUR"}, false},
		{"119999.99", []string{"USD", "USD"}, false},
	}
	for _, tt := range tests {
		b := Balance(withPlannedCost("120000.00"), amounts(tt.assigned), tt.currencies)

		if b.RoundingHint != tt.hint {
			t.Errorf("assigned %s in %v: hint %v, want %v", tt.assigned, tt.currencies, b.RoundingHint, tt.hint)
		}
	}
}
