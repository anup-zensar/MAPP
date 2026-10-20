package money_test

import (
	"errors"
	"slices"
	"testing"

	"northstar/internal/platform/money"
)

// Convert converts between two currencies given their rates per 1 USD: amount × to ÷ from, rounded half up
// to the cent once (016 data-model §3.1, D-262). The examples are the worked examples of 016.
func TestConvert_Table_FR015(t *testing.T) {
	cases := []struct {
		name           string
		amount         string
		from, to, want string
	}{
		{"same rate leaves the amount", "1234.56", "0.79", "0.79", "1234.56"},
		{"EUR line item into a GBP plan", "23000", "0.92", "0.79", "19750.00"},
		{"USD default into a GBP plan", "5000", "1", "0.79", "3950.00"},
		{"GBP plan budget into USD", "600000", "0.79", "1", "759493.67"},
		{"GBP unassigned into USD", "5250", "0.79", "1", "6645.57"},
		{"half up", "0.01", "2", "1", "0.01"},                // 0.005 → 0.01
		{"below half rounds down", "0.01", "3", "1", "0.00"}, // 0.0033 → 0.00
		{"negative half away from zero", "-0.01", "2", "1", "-0.01"},
		{"negative amount", "-23000", "0.92", "0.79", "-19750.00"},
		{"extreme rates do not overflow", "999999999.99", "0.000001", "1", "999999999990000.00"},
		{"extreme rates the other way", "1000000", "1000000", "0.000001", "0.00"},
	}
	for _, c := range cases {
		got := money.Convert(amount(t, c.amount), rate(t, c.from), rate(t, c.to))
		if got.String() != c.want {
			t.Errorf("%s: Convert(%s, %s, %s) = %s, want %s", c.name, c.amount, c.from, c.to, got, c.want)
		}
	}
}

// Convert to and from USD gives exactly what ToUSD and FromUSD give (their results stay unchanged).
func TestConvert_MatchesToUSDAndFromUSD(t *testing.T) {
	for _, text := range []string{"0.01", "1000", "1086.96", "-55.55", "123456789.99"} {
		for _, r := range []string{"0.92", "0.79", "7.1", "83.5", "150", "1"} {
			a, rt := amount(t, text), rate(t, r)
			if got, want := money.Convert(a, rt, money.One), money.ToUSD(a, rt); got != want {
				t.Errorf("Convert(%s, %s, 1) = %s, ToUSD = %s", text, r, got, want)
			}
			if got, want := money.Convert(a, money.One, rt), money.FromUSD(a, rt); got != want {
				t.Errorf("Convert(%s, 1, %s) = %s, FromUSD = %s", text, r, got, want)
			}
		}
	}
}

// ConvertParts converts month amounts so that they add up to the conversion of their sum: every part but
// the last on its own, the last gets the rest (D-024, D-262).
func TestConvertParts_AddsUp_FR015(t *testing.T) {
	parts := []money.Amount{amount(t, "7666.66"), amount(t, "7666.66"), amount(t, "7666.68")} // 23,000.00 EUR evenly
	got := money.ConvertParts(parts, rate(t, "0.92"), rate(t, "0.79"))
	if total := money.Sum(got...); total.String() != "19750.00" {
		t.Errorf("ConvertParts adds up to %s, want 19750.00 (%v)", total, texts(got))
	}
	if got[0] != money.Convert(parts[0], rate(t, "0.92"), rate(t, "0.79")) {
		t.Errorf("the first part is %s, want it converted on its own", got[0])
	}
	if len(money.ConvertParts(nil, money.One, money.One)) != 0 {
		t.Error("ConvertParts(nothing) is not empty")
	}
	spread := []money.Amount{amount(t, "3333.33"), amount(t, "3333.33"), amount(t, "3333.34")}
	if !slices.Equal(money.ConvertParts(spread, rate(t, "0.92"), money.One), money.ConvertSpread(spread, rate(t, "0.92"))) {
		t.Error("ConvertParts(parts, r, 1) differs from ConvertSpread(parts, r)")
	}
}

// Rates.Convert converts a Money into a currency with the rates it holds; USD needs no rate, a currency
// without a rate is ErrUnknownCurrency.
func TestRatesConvert_UnknownCurrency(t *testing.T) {
	rs := money.Rates{"EUR": rate(t, "0.92"), "GBP": rate(t, "0.79")}
	got, err := rs.Convert(money.Money{Amount: amount(t, "23000"), Currency: "EUR"}, "GBP")
	if err != nil || got.Amount.String() != "19750.00" || got.Currency != "GBP" {
		t.Errorf("Convert(23,000.00 EUR, GBP) = %v, %v; want 19,750.00 GBP", got, err)
	}
	if got, err := rs.Convert(money.USD(amount(t, "5000")), "GBP"); err != nil || got.Amount.String() != "3950.00" {
		t.Errorf("Convert(5,000.00 USD, GBP) = %v, %v; want 3,950.00 GBP", got, err)
	}
	if got, err := rs.Convert(money.Money{Amount: amount(t, "10"), Currency: "JPY"}, "JPY"); err != nil || got.Amount.String() != "10.00" {
		t.Errorf("Convert into its own currency = %v, %v; want it unchanged", got, err)
	}
	if _, err := rs.Convert(money.Money{Amount: amount(t, "10"), Currency: "JPY"}, "USD"); !errors.Is(err, money.ErrUnknownCurrency) {
		t.Errorf("Convert(JPY) error = %v, want ErrUnknownCurrency", err)
	}
	if _, err := rs.Convert(money.USD(amount(t, "10")), "CHF"); !errors.Is(err, money.ErrUnknownCurrency) {
		t.Errorf("Convert into CHF error = %v, want ErrUnknownCurrency", err)
	}
}

// A display currency with rates converts an amount of any currency it knows once, half up; an amount in
// the display currency stays as it is (016 FR-013, D-261).
func TestDisplayWithRates_ConvertsFromThePlanCurrency_FR013(t *testing.T) {
	rs := money.Rates{"EUR": rate(t, "0.92"), "GBP": rate(t, "0.79")}
	gbp := money.Money{Amount: amount(t, "5250"), Currency: "GBP"}

	if got := money.DisplayUSD.WithRates(rs).Money(gbp); got.Amount.String() != "6645.57" || got.Currency != "USD" {
		t.Errorf("USD display of 5,250.00 GBP = %v, want 6,645.57 USD", got)
	}
	if got := money.DisplayIn("GBP", "0.79").WithRates(rs).Money(gbp); got != gbp {
		t.Errorf("GBP display of 5,250.00 GBP = %v, want it unchanged", got)
	}
	if got := money.DisplayIn("EUR", "0.92").WithRates(rs).Money(gbp); got.Amount.String() != "6113.92" || got.Currency != "EUR" {
		t.Errorf("EUR display of 5,250.00 GBP = %v, want 6,113.92 EUR", got) // 5,250 × 0.92 ÷ 0.79
	}
	usd := money.USD(amount(t, "700000"))
	if got := money.DisplayIn("EUR", "0.92").WithRates(rs).Money(usd); got.Amount.String() != "644000.00" {
		t.Errorf("EUR display of 700,000.00 USD = %v, want 644,000.00 EUR", got)
	}
	jpy := money.Money{Amount: amount(t, "10"), Currency: "JPY"}
	if got := money.DisplayUSD.WithRates(rs).Money(jpy); got != jpy {
		t.Errorf("an amount without a rate = %v, want it unchanged in its own currency", got)
	}
}
