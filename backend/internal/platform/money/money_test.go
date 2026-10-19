package money_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"northstar/internal/platform/money"
)

// amount parses s or stops the test.
func amount(t *testing.T, s string) money.Amount {
	t.Helper()
	a, err := money.ParseAmount(s)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", s, err)
	}
	return a
}

// texts writes each amount with String.
func texts(amounts []money.Amount) []string {
	list := make([]string, 0, len(amounts))
	for _, a := range amounts {
		list = append(list, a.String())
	}
	return list
}

// An amount has at most fifteen integer digits and two decimals; a leading minus is allowed; nothing else
// is an amount, and the errors tell "too many decimals" and "too large" apart (data model §1, D-010, D-190).
func TestParseAmount_FormatsAndLimits_D010(t *testing.T) {
	valid := map[string]string{
		"0":                      "0.00",
		"1234":                   "1234.00",
		"1234.5":                 "1234.50",
		"1234.50":                "1234.50",
		"-1234.5":                "-1234.50",
		"-0.00":                  "0.00",
		"007.10":                 "7.10",
		"999999999999999.99":     "999999999999999.99",
		"-999999999999999.99":    "-999999999999999.99",
		"0000999999999999999.99": "999999999999999.99", // leading zeros are not digits that count
	}
	for text, want := range valid {
		got, err := money.ParseAmount(text)
		if err != nil || got.String() != want {
			t.Errorf("ParseAmount(%q) = %q, %v; want %q", text, got.String(), err, want)
		}
	}

	invalid := map[string]error{
		"":                    money.ErrInvalidAmount,
		"abc":                 money.ErrInvalidAmount,
		"1,234.50":            money.ErrInvalidAmount,
		" 12":                 money.ErrInvalidAmount,
		"+12":                 money.ErrInvalidAmount,
		"12.":                 money.ErrInvalidAmount,
		".5":                  money.ErrInvalidAmount,
		"1e3":                 money.ErrInvalidAmount,
		"12.345":              money.ErrTooManyDecimals,
		"1000000000000000":    money.ErrOutOfRange, // sixteen integer digits
		"-1000000000000000.5": money.ErrOutOfRange,
	}
	for text, want := range invalid {
		if _, err := money.ParseAmount(text); !errors.Is(err, want) {
			t.Errorf("ParseAmount(%q) error = %v, want %v", text, err, want)
		}
	}
	if _, err := money.ParseAmount("12.345"); !errors.Is(err, money.ErrInvalidAmount) {
		t.Errorf("ErrTooManyDecimals is not an ErrInvalidAmount: %v", err)
	}
}

// Amounts are JSON strings with two decimals both ways; reading accepts 0 to 2 decimals and refuses
// numbers and more decimals (D-010).
func TestAmountJSON_StringWithTwoDecimals_D010(t *testing.T) {
	type record struct {
		Budget  money.Amount  `json:"budget"`
		Planned *money.Amount `json:"planned"`
	}
	encoded, err := json.Marshal(record{Budget: money.FromCents(120000050)})
	if err != nil || string(encoded) != `{"budget":"1200000.50","planned":null}` {
		t.Fatalf("Marshal = %s, %v", encoded, err)
	}

	var decoded record
	if err := json.Unmarshal([]byte(`{"budget":"12.5","planned":"7"}`), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Budget.Cents() != 1250 || decoded.Planned == nil || decoded.Planned.Cents() != 700 {
		t.Errorf("decoded %+v", decoded)
	}

	for _, body := range []string{`{"budget":12.5}`, `{"budget":"12.555"}`, `{"budget":"x"}`} {
		if err := json.Unmarshal([]byte(body), &decoded); err == nil {
			t.Errorf("Unmarshal(%s) succeeded, want an error", body)
		}
	}

	var money2 money.Money
	if err := json.Unmarshal([]byte(`{"amount":"1234.50","currency":"EUR"}`), &money2); err != nil {
		t.Fatalf("Unmarshal Money: %v", err)
	}
	if again, _ := json.Marshal(money2); string(again) != `{"amount":"1234.50","currency":"EUR"}` {
		t.Errorf("Money round trip = %s", again)
	}
}

// Texts for people group thousands; Money adds its currency (data model §1, the texts of the activity log).
func TestFormat_ThousandsAndCurrency_D010(t *testing.T) {
	cases := map[string]string{
		"0":                   "0.00",
		"0.05":                "0.05",
		"999":                 "999.00",
		"1234.5":              "1,234.50",
		"1200000":             "1,200,000.00",
		"-1234567.89":         "-1,234,567.89",
		"-0.05":               "-0.05",
		"999999999999999.99":  "999,999,999,999,999.99",
		"-100000000000000.00": "-100,000,000,000,000.00",
	}
	for text, want := range cases {
		if got := amount(t, text).Format(); got != want {
			t.Errorf("Format(%s) = %q, want %q", text, got, want)
		}
	}
	if got := (money.Money{Amount: amount(t, "1234.5"), Currency: "EUR"}).Format(); got != "1,234.50 EUR" {
		t.Errorf("Money.Format = %q", got)
	}
	if got := money.USD(amount(t, "25000")).Format(); got != "25,000.00 USD" {
		t.Errorf("USD(...).Format = %q", got)
	}
}

// Add, Sub, Neg, Cmp, Sign, IsZero and Sum are exact; the sum of nothing is 0.00.
func TestArithmetic_Exact_D010(t *testing.T) {
	a, b := amount(t, "0.10"), amount(t, "0.20")
	if got := a.Add(b).String(); got != "0.30" {
		t.Errorf("0.10 + 0.20 = %s", got)
	}
	if got := a.Sub(b).String(); got != "-0.10" {
		t.Errorf("0.10 - 0.20 = %s", got)
	}
	if got := a.Neg().String(); got != "-0.10" {
		t.Errorf("-0.10 = %s", got)
	}
	if a.Cmp(b) != -1 || b.Cmp(a) != 1 || a.Cmp(a) != 0 {
		t.Error("Cmp is wrong")
	}
	if a.Sign() != 1 || a.Neg().Sign() != -1 || (money.Amount{}).Sign() != 0 || !(money.Amount{}).IsZero() {
		t.Error("Sign or IsZero is wrong")
	}
	if got := money.Sum(); !got.IsZero() {
		t.Errorf("Sum() = %s, want 0.00", got)
	}
	if got := money.Sum(a, b, amount(t, "-0.05")).String(); got != "0.25" {
		t.Errorf("Sum = %s, want 0.25", got)
	}
}

// Deals × deal size is exact; beyond the largest amount it is ErrOutOfRange (data model §1).
func TestMulInt_OutOfRange_D010(t *testing.T) {
	got, err := amount(t, "10000").MulInt(250)
	if err != nil || got.String() != "2500000.00" {
		t.Errorf("10,000.00 × 250 = %s, %v; want 2500000.00", got, err)
	}
	if got, err := amount(t, "-1.25").MulInt(3); err != nil || got.String() != "-3.75" {
		t.Errorf("-1.25 × 3 = %s, %v", got, err)
	}
	if _, err := amount(t, "100000000").MulInt(10_000_000_000); !errors.Is(err, money.ErrOutOfRange) {
		t.Errorf("100,000,000.00 × 10^10: error = %v, want ErrOutOfRange", err)
	}
	if _, err := amount(t, "999999999999999.99").MulInt(9_223_372_036_854_775_807); !errors.Is(err, money.ErrOutOfRange) {
		t.Errorf("a product beyond int64: error = %v, want ErrOutOfRange", err)
	}
}

// An even split rounds every part down to the cent and puts the remainder in the last part (domain model
// §4.4 E, D-024).
func TestSplitEvenly_RemainderLast_D024(t *testing.T) {
	cases := []struct {
		total string
		n     int
		want  []string
	}{
		{"10000", 3, []string{"3333.33", "3333.33", "3333.34"}},
		{"0.02", 3, []string{"0.00", "0.00", "0.02"}},
		{"0", 3, []string{"0.00", "0.00", "0.00"}},
		{"100", 1, []string{"100.00"}},
		{"120000", 4, []string{"30000.00", "30000.00", "30000.00", "30000.00"}},
	}
	for _, c := range cases {
		parts := amount(t, c.total).SplitEvenly(c.n)
		if got := texts(parts); !slices.Equal(got, c.want) {
			t.Errorf("SplitEvenly(%s, %d) = %v, want %v", c.total, c.n, got, c.want)
		}
		if sum := money.Sum(parts...); sum.Cmp(amount(t, c.total)) != 0 {
			t.Errorf("the parts of %s add up to %s", c.total, sum)
		}
	}
	if got := amount(t, "5").SplitEvenly(0); len(got) != 0 {
		t.Errorf("SplitEvenly(0) = %v, want none", got)
	}
}

// A request's amount is parsed with its currency code; the amount's problem comes first (MoneyInput).
func TestParseInput_AmountAndCurrency_D010(t *testing.T) {
	got, err := money.ParseInput(money.Input{Amount: "25000.5", Currency: "USD"})
	if err != nil || got.Amount.String() != "25000.50" || got.Currency != "USD" {
		t.Errorf("ParseInput = %+v, %v", got, err)
	}
	cases := []struct {
		in   money.Input
		want error
	}{
		{money.Input{Amount: "", Currency: "USD"}, money.ErrInvalidAmount},
		{money.Input{Amount: "1.234", Currency: "USD"}, money.ErrTooManyDecimals},
		{money.Input{Amount: "12", Currency: "usd"}, money.ErrInvalidCurrency},
		{money.Input{Amount: "12", Currency: ""}, money.ErrInvalidCurrency},
		{money.Input{Amount: "12", Currency: "EURO"}, money.ErrInvalidCurrency},
		{money.Input{Amount: "x", Currency: "x"}, money.ErrInvalidAmount},
	}
	for _, c := range cases {
		if _, err := money.ParseInput(c.in); !errors.Is(err, c.want) {
			t.Errorf("ParseInput(%+v) error = %v, want %v", c.in, err, c.want)
		}
	}
}
