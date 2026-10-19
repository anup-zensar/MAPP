package money_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"northstar/internal/platform/money"
)

// percent parses s or stops the test.
func percent(t *testing.T, s string) money.Percent {
	t.Helper()
	p, err := money.ParsePercent(s)
	if err != nil {
		t.Fatalf("ParsePercent(%q): %v", s, err)
	}
	return p
}

// A percentage has at most three integer digits and two decimals, never negative; it is written with two
// decimals (data model §1).
func TestParsePercent_Formats_FR009(t *testing.T) {
	valid := map[string]string{"2": "2.00", "2.5": "2.50", "33.33": "33.33", "100.00": "100.00", "0": "0.00", "999.99": "999.99"}
	for text, want := range valid {
		if got := percent(t, text).String(); got != want {
			t.Errorf("ParsePercent(%q) = %q, want %q", text, got, want)
		}
	}
	for _, text := range []string{"", "-1", "2.555", "1000", "abc", "2,5", "50%"} {
		if _, err := money.ParsePercent(text); !errors.Is(err, money.ErrInvalidPercent) {
			t.Errorf("ParsePercent(%q) error = %v, want ErrInvalidPercent", text, err)
		}
	}
	if got := percent(t, "2").Format(); got != "2.00 %" {
		t.Errorf("Format = %q, want 2.00 %%", got)
	}
	if percent(t, "100").Cmp(money.Hundred) != 0 || !percent(t, "0").IsZero() || percent(t, "2").Hundredths() != 200 {
		t.Error("Hundred, Cmp, IsZero or Hundredths is wrong")
	}
	if got := money.PercentFromHundredths(3333).String(); got != "33.33" {
		t.Errorf("PercentFromHundredths(3333) = %q", got)
	}
}

// A percentage of a whole number is rounded half-up to a whole number (domain model §4.5, the waterfall
// examples of data model §2.1).
func TestOfWhole_HalfUp_FR010(t *testing.T) {
	cases := []struct {
		percent string
		n       int64
		want    int64
	}{
		{"2", 250_000, 5_000},
		{"5", 5_000, 250},
		{"3", 250_000, 7_500},
		{"4", 7_500, 300},
		{"2", 25, 1},    // 0.5 rounds up
		{"5", 1, 0},     // 0.05
		{"2", 0, 0},     // nothing
		{"33.33", 3, 1}, // 0.9999
		{"100", 1_000_000_000, 1_000_000_000},
	}
	for _, c := range cases {
		if got := percent(t, c.percent).OfWhole(c.n); got != c.want {
			t.Errorf("%s %% of %d = %d, want %d", c.percent, c.n, got, c.want)
		}
	}
}

// 100 % split into n shares: each rounded down to the hundredth, the remainder in the last (R10).
func TestPercentSplitEvenly_R10(t *testing.T) {
	cases := map[int][]string{
		1: {"100.00"},
		2: {"50.00", "50.00"},
		3: {"33.33", "33.33", "33.34"},
		6: {"16.66", "16.66", "16.66", "16.66", "16.66", "16.70"},
	}
	for n, want := range cases {
		shares := money.PercentSplitEvenly(n)
		var got []string
		for _, s := range shares {
			got = append(got, s.String())
		}
		if !slices.Equal(got, want) {
			t.Errorf("PercentSplitEvenly(%d) = %v, want %v", n, got, want)
		}
		if money.SumPercent(shares...).Cmp(money.Hundred) != 0 {
			t.Errorf("the %d shares do not add up to 100.00", n)
		}
	}
	if got := money.PercentSplitEvenly(0); len(got) != 0 {
		t.Errorf("PercentSplitEvenly(0) = %v, want none", got)
	}
	if got := money.SumPercent(); !got.IsZero() {
		t.Errorf("SumPercent() = %s, want 0.00", got)
	}
}

// Percentages are JSON strings with two decimals both ways.
func TestPercentJSON_String_FR009(t *testing.T) {
	encoded, err := json.Marshal(map[string]money.Percent{"share": percent(t, "60")})
	if err != nil || string(encoded) != `{"share":"60.00"}` {
		t.Errorf("Marshal = %s, %v", encoded, err)
	}
	var decoded struct {
		Share money.Percent `json:"share"`
	}
	if err := json.Unmarshal([]byte(`{"share":"33.3"}`), &decoded); err != nil || decoded.Share.String() != "33.30" {
		t.Errorf("Unmarshal = %+v, %v", decoded, err)
	}
	if err := json.Unmarshal([]byte(`{"share":33.3}`), &decoded); err == nil {
		t.Error("a JSON number was accepted as a percentage")
	}
}
