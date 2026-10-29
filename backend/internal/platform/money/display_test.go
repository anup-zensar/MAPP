package money_test

import (
	"fmt"
	"testing"

	"mapp/internal/platform/money"
)

// A display currency converts US dollars half up to the cent; USD, no code or an unusable rate leave the
// amounts in US dollars (D-061, D-220).
func TestDisplay_ConvertsUSDOnceHalfUp_D220(t *testing.T) {
	t.Parallel()
	usd := money.USD(amount(t, "700000"))
	cases := []struct {
		code, rate     string
		want, currency string
	}{
		{"EUR", "0.92", "644000.00", "EUR"},
		{"INR", "83.5", "58450000.00", "INR"},
		{"USD", "1", "700000.00", "USD"},
		{"", "0.92", "700000.00", "USD"},
		{"EUR", "0", "700000.00", "USD"},
		{"EUR", "not a rate", "700000.00", "USD"},
	}
	for _, c := range cases {
		d := money.DisplayIn(c.code, c.rate)
		got := d.Money(usd)
		if got.Amount.String() != c.want || got.Currency != c.currency || d.Currency() != c.currency {
			t.Errorf("DisplayIn(%q, %q).Money(700,000.00 USD) = %s %s (currency %s), want %s %s",
				c.code, c.rate, got.Amount, got.Currency, d.Currency(), c.want, c.currency)
		}
	}
	if got := money.DisplayIn("EUR", "0.92").Money(money.USD(amount(t, "0.01"))); got.Amount.String() != "0.01" {
		t.Errorf("0.01 USD at 0.92 = %s EUR, want 0.01 (0.0092 rounds half up)", got.Amount)
	}
}

// Optional keeps "no amount" as nil and converts an amount.
func TestDisplay_OptionalKeepsNil_D220(t *testing.T) {
	t.Parallel()
	d := money.DisplayIn("EUR", "0.92")
	if d.Optional(nil) != nil {
		t.Error("Optional(nil) is not nil")
	}
	usd := money.USD(amount(t, "-100"))
	if got := d.Optional(&usd); got == nil || got.Amount.String() != "-92.00" || got.Currency != "EUR" {
		t.Errorf("Optional(-100.00 USD) = %v, want -92.00 EUR", got)
	}
	if got := money.DisplayUSD.Money(usd); got != usd {
		t.Errorf("DisplayUSD changed %v into %v", usd, got)
	}
}

// Parts converts the months of one total so that they add up to the total converted once: 3 × 5,000.00 GBP
// at 0.79 is 18,987.34 USD, not 3 × 6,329.11; in GBP, or without a rate, they stay as they are (038 FR-022, M-1).
func TestDisplayParts_038_FR022_M1(t *testing.T) {
	t.Parallel()
	rates := money.Rates{"GBP": rate(t, "0.79"), "EUR": rate(t, "0.92")}
	gbp := money.Money{Amount: amount(t, "5000.00"), Currency: "GBP"}
	parts := []money.Money{gbp, gbp, gbp}
	cases := []struct {
		d    money.Display
		want string
	}{
		{money.DisplayUSD.WithRates(rates), "[6329.11 USD 6329.11 USD 6329.12 USD]"},
		{money.DisplayIn("EUR", "0.92").WithRates(rates), "[5822.78 EUR 5822.78 EUR 5822.79 EUR]"},
		{money.DisplayIn("GBP", "0.79").WithRates(rates), "[5000.00 GBP 5000.00 GBP 5000.00 GBP]"},
		{money.DisplayIn("EUR", "0.92"), "[5000.00 GBP 5000.00 GBP 5000.00 GBP]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(moneyTexts(c.d.Parts(parts))); got != c.want {
			t.Errorf("Parts(3 × 5,000.00 GBP) in %s = %s, want %s", c.d.Currency(), got, c.want)
		}
	}
	if got := money.DisplayUSD.Parts(nil); len(got) != 0 {
		t.Errorf("Parts(nil) = %v, want empty", got)
	}
}

func moneyTexts(ms []money.Money) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Amount.String() + " " + m.Currency
	}
	return out
}
