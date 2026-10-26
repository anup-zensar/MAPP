package money_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"mapp/internal/platform/money"
)

// rate parses s or stops the test.
func rate(t *testing.T, s string) money.Rate {
	t.Helper()
	r, err := money.ParseRate(s)
	if err != nil {
		t.Fatalf("ParseRate(%q): %v", s, err)
	}
	return r
}

// A rate has at most six decimals, is above 0 and at most 1,000,000; it is written without trailing zeros
// (data model §6, FR-028).
func TestParseRate_Limits_FR028(t *testing.T) {
	t.Parallel()
	valid := map[string]string{
		"0.92":           "0.92",
		"0.920000":       "0.92",
		"150":            "150",
		"1":              "1",
		"7.10":           "7.1",
		"83.50":          "83.5",
		"1.086957":       "1.086957",
		"0.000001":       "0.000001",
		"1000000":        "1000000",
		"1000000.000000": "1000000",
	}
	for text, want := range valid {
		if got := rate(t, text).String(); got != want {
			t.Errorf("ParseRate(%q).String() = %q, want %q", text, got, want)
		}
	}

	invalid := map[string]error{
		"":               money.ErrInvalidRate,
		"0":              money.ErrInvalidRate,
		"0.000000":       money.ErrInvalidRate,
		"-0.92":          money.ErrInvalidRate,
		"0.9200001":      money.ErrInvalidRate, // seven decimals
		"abc":            money.ErrInvalidRate,
		"1,5":            money.ErrInvalidRate,
		"1000000.000001": money.ErrOutOfRange,
		"99999999":       money.ErrOutOfRange,
	}
	for text, want := range invalid {
		if _, err := money.ParseRate(text); !errors.Is(err, want) {
			t.Errorf("ParseRate(%q) error = %v, want %v", text, err, want)
		}
	}

	if !money.One.IsOne() || rate(t, "1.0").Micros() != 1_000_000 || rate(t, "0.92").IsOne() {
		t.Error("One, IsOne or Micros is wrong")
	}
	if got := rate(t, "1000000").Format(); got != "1,000,000" {
		t.Errorf("Format = %q, want 1,000,000", got)
	}
	if got := money.RateFromMicros(920000).String(); got != "0.92" {
		t.Errorf("RateFromMicros(920000) = %q", got)
	}
}

// The inverse rate is 1 ÷ rate, rounded half-up to six decimals (spec US4).
func TestRateInverse_SixDecimals_FR028(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"0.92":    "1.086957",
		"150":     "0.006667",
		"1":       "1",
		"0.79":    "1.265823",
		"7.10":    "0.140845",
		"83.50":   "0.011976",
		"1000000": "0.000001",
		"0.5":     "2",
	}
	for text, want := range cases {
		if got := rate(t, text).Inverse().String(); got != want {
			t.Errorf("Inverse(%s) = %s, want %s", text, got, want)
		}
	}
}

// Converting to USD divides by the rate and rounds half-up to the cent, negative amounts half away from
// zero (domain model §4.3).
func TestToUSD_RoundsHalfUp_FR029(t *testing.T) {
	t.Parallel()
	cases := []struct{ local, rate, want string }{
		{"1000", "0.92", "1086.96"}, // 1,086.9565…
		{"46000", "0.92", "50000.00"},
		{"10000", "0.92", "10869.57"},
		{"15800", "0.79", "20000.00"},
		{"0.01", "2", "0.01"},   // 0.005 rounds up
		{"-0.01", "2", "-0.01"}, // -0.005 rounds away from zero
		{"0.01", "3", "0.00"},   // 0.0033…
		{"-1000", "0.92", "-1086.96"},
		{"0", "0.92", "0.00"},
		{"123.45", "1", "123.45"},
		{"999999999999999.99", "1000000", "1000000000.00"},
	}
	for _, c := range cases {
		if got := money.ToUSD(amount(t, c.local), rate(t, c.rate)).String(); got != c.want {
			t.Errorf("ToUSD(%s, %s) = %s, want %s", c.local, c.rate, got, c.want)
		}
	}
}

// Converting from USD multiplies by the rate and rounds half-up to the cent (display currencies).
func TestFromUSD_RoundsHalfUp_D010(t *testing.T) {
	t.Parallel()
	cases := []struct{ usd, rate, want string }{
		{"1086.96", "0.92", "1000.00"}, // 1,000.0032
		{"100", "0.92", "92.00"},
		{"0.01", "0.5", "0.01"},   // 0.005 rounds up
		{"-0.01", "0.5", "-0.01"}, // half away from zero
		{"1.10", "150", "165.00"},
		{"2500000", "7.10", "17750000.00"},
		{"12.34", "1", "12.34"},
	}
	for _, c := range cases {
		if got := money.FromUSD(amount(t, c.usd), rate(t, c.rate)).String(); got != c.want {
			t.Errorf("FromUSD(%s, %s) = %s, want %s", c.usd, c.rate, got, c.want)
		}
	}
}

// Month amounts converted to USD add up to the USD value of their sum: the last month takes the
// difference (D-024).
func TestConvertSpread_LastMonthTakesDifference_D024(t *testing.T) {
	t.Parallel()
	parts := amount(t, "10000").SplitEvenly(3)
	converted := money.ConvertSpread(parts, rate(t, "0.92"))
	if got, want := texts(converted), []string{"3623.18", "3623.18", "3623.21"}; !slices.Equal(got, want) {
		t.Errorf("ConvertSpread = %v, want %v", got, want)
	}
	if sum := money.Sum(converted...); sum.String() != "10869.57" {
		t.Errorf("the converted months add up to %s, want 10869.57", sum)
	}
	if got := money.ConvertSpread(nil, money.One); len(got) != 0 {
		t.Errorf("ConvertSpread(nil) = %v, want none", got)
	}
	single := money.ConvertSpread([]money.Amount{amount(t, "46000")}, rate(t, "0.92"))
	if got := texts(single); !slices.Equal(got, []string{"50000.00"}) {
		t.Errorf("one month = %v", got)
	}
}

// Rates convert with each currency's current rate; USD needs no rate; a currency without a rate is
// ErrUnknownCurrency (FR-029).
func TestRates_UnknownCurrency_FR029(t *testing.T) {
	t.Parallel()
	rates := money.Rates{"EUR": rate(t, "0.92"), "GBP": rate(t, "0.79")}

	usd, err := rates.ToUSD(money.Money{Amount: amount(t, "46000"), Currency: "EUR"})
	if err != nil || usd.String() != "50000.00" {
		t.Errorf("ToUSD(46,000.00 EUR) = %s, %v", usd, err)
	}
	usd, err = rates.ToUSD(money.USD(amount(t, "12.34")))
	if err != nil || usd.String() != "12.34" {
		t.Errorf("ToUSD(12.34 USD) = %s, %v", usd, err)
	}
	if _, err := rates.ToUSD(money.Money{Amount: amount(t, "1"), Currency: "JPY"}); !errors.Is(err, money.ErrUnknownCurrency) {
		t.Errorf("ToUSD(JPY) error = %v, want ErrUnknownCurrency", err)
	}

	gbp, err := rates.FromUSD(amount(t, "20000"), "GBP")
	if err != nil || gbp.Amount.String() != "15800.00" || gbp.Currency != "GBP" {
		t.Errorf("FromUSD(20,000.00, GBP) = %+v, %v", gbp, err)
	}
	same, err := rates.FromUSD(amount(t, "5"), "USD")
	if err != nil || same != money.USD(amount(t, "5")) {
		t.Errorf("FromUSD(5, USD) = %+v, %v", same, err)
	}
	if _, err := rates.FromUSD(amount(t, "1"), "CNY"); !errors.Is(err, money.ErrUnknownCurrency) {
		t.Errorf("FromUSD(CNY) error = %v, want ErrUnknownCurrency", err)
	}
	if _, err := (money.Rates{"XXX": {}}).ToUSD(money.Money{Currency: "XXX"}); !errors.Is(err, money.ErrInvalidRate) {
		t.Errorf("a zero rate: error = %v, want ErrInvalidRate", err)
	}
}

// Rates are JSON strings without trailing zeros both ways.
func TestRateJSON_String_FR028(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(map[string]money.Rate{"rate": rate(t, "7.100")})
	if err != nil || string(encoded) != `{"rate":"7.1"}` {
		t.Errorf("Marshal = %s, %v", encoded, err)
	}
	var decoded struct {
		Rate money.Rate `json:"rate"`
	}
	if err := json.Unmarshal([]byte(`{"rate":"0.92"}`), &decoded); err != nil || decoded.Rate.Micros() != 920000 {
		t.Errorf("Unmarshal = %+v, %v", decoded, err)
	}
	for _, body := range []string{`{"rate":0.92}`, `{"rate":"0"}`, `{"rate":"-1"}`} {
		if err := json.Unmarshal([]byte(body), &decoded); err == nil {
			t.Errorf("Unmarshal(%s) succeeded, want an error", body)
		}
	}
}
