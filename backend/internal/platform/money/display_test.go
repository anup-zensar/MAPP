package money_test

import (
	"testing"

	"mapp/internal/platform/money"
)

// A display currency converts US dollars half up to the cent; USD, no code or an unusable rate leave the
// amounts in US dollars (D-061, D-220).
func TestDisplay_ConvertsUSDOnceHalfUp_D220(t *testing.T) {
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
