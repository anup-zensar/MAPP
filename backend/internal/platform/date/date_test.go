package date_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"mapp/internal/platform/date"
)

// A date is read only in the form YYYY-MM-DD and only when the day exists (D-101).
func TestParse_StrictForm_D101(t *testing.T) {
	t.Parallel()
	for _, text := range []string{"2026-03-01", "2028-02-29", "2026-12-31"} {
		d, err := date.Parse(text)
		if err != nil || d.String() != text {
			t.Errorf("Parse(%q) = %q, %v", text, d, err)
		}
	}
	for _, text := range []string{"", "2026-3-1", "01/03/2026", "2026-02-29", "2026-13-01", "2026-03-01T00:00:00Z", " 2026-03-01"} {
		if _, err := date.Parse(text); !errors.Is(err, date.ErrInvalid) {
			t.Errorf("Parse(%q) error = %v, want ErrInvalid", text, err)
		}
	}
}

// Dates are JSON strings; a missing date (*Date nil, or the zero Date) is null; a wrong date is refused
// (D-101).
func TestJSON_StringOrNull_D101(t *testing.T) {
	t.Parallel()
	start := date.MustParse("2026-03-01")
	type record struct {
		StartDate *date.Date `json:"startDate"`
		EndDate   *date.Date `json:"endDate"`
		Latest    date.Date  `json:"latest"`
	}
	encoded, err := json.Marshal(record{StartDate: &start})
	if err != nil || string(encoded) != `{"startDate":"2026-03-01","endDate":null,"latest":null}` {
		t.Fatalf("Marshal = %s, %v", encoded, err)
	}

	var decoded record
	if err := json.Unmarshal([]byte(`{"startDate":"2026-03-01","endDate":null,"latest":"2036-12-31"}`), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.StartDate == nil || *decoded.StartDate != start || decoded.EndDate != nil || decoded.Latest != date.YearEnd(2036) {
		t.Errorf("decoded %+v", decoded)
	}
	for _, body := range []string{`{"startDate":"2026-02-30"}`, `{"startDate":20260301}`, `{"startDate":"1 Mar 2026"}`} {
		if err := json.Unmarshal([]byte(body), &decoded); err == nil {
			t.Errorf("Unmarshal(%s) succeeded, want an error", body)
		}
	}
}

// Dates compare by day; today is the UTC calendar date of the clock (D-091).
func TestCompareAndToday_D091(t *testing.T) {
	t.Parallel()
	a, b := date.MustParse("2026-03-01"), date.MustParse("2026-03-02")
	if !a.Before(b) || !b.After(a) || a.Compare(b) != -1 || b.Compare(a) != 1 || a.Compare(a) != 0 {
		t.Error("Before, After or Compare is wrong")
	}
	if date.MustParse("2025-12-31").Compare(date.MustParse("2026-01-01")) != -1 || date.MustParse("2026-02-28").Compare(date.MustParse("2026-01-31")) != 1 {
		t.Error("Compare across months or years is wrong")
	}
	lateEvening := time.Date(2026, 3, 12, 23, 30, 0, 0, time.FixedZone("UTC-2", -2*60*60)) // 13 March in UTC
	if got := date.Today(lateEvening); got != date.New(2026, time.March, 13) {
		t.Errorf("Today = %s, want 2026-03-13", got)
	}
	if date.YearStart(2026).String() != "2026-01-01" || date.YearEnd(2026).String() != "2026-12-31" {
		t.Error("YearStart or YearEnd is wrong")
	}
	if !(date.Date{}).IsZero() || a.IsZero() || (date.Date{}).String() != "" {
		t.Error("the zero Date is wrong")
	}
	if got := date.New(2026, time.February, 30); got.String() != "2026-03-02" {
		t.Errorf("New(2026, 2, 30) = %s, want 2026-03-02", got)
	}
	if got := a.AddDays(-1); got.String() != "2026-02-28" {
		t.Errorf("AddDays(-1) = %s", got)
	}
}

// Moving a date by years keeps the day; 29 February becomes 28 February in a year without it (R13).
func TestAddYears_LeapDay_R13(t *testing.T) {
	t.Parallel()
	cases := []struct {
		from  string
		years int
		want  string
	}{
		{"2026-03-01", 1, "2027-03-01"},
		{"2028-02-29", 1, "2029-02-28"},
		{"2028-02-29", 4, "2032-02-29"},
		{"2027-02-28", 1, "2028-02-28"},
		{"2026-12-31", 0, "2026-12-31"},
		{"2026-06-15", -1, "2025-06-15"},
	}
	for _, c := range cases {
		if got := date.MustParse(c.from).AddYears(c.years); got.String() != c.want {
			t.Errorf("%s + %d years = %s, want %s", c.from, c.years, got, c.want)
		}
	}
}

// Texts for people: "01 Mar 2026"; a range shows the year once when it is the same (D-099).
func TestFormat_DayMonthYear_D099(t *testing.T) {
	t.Parallel()
	if got := date.Format(date.MustParse("2026-03-01")); got != "01 Mar 2026" {
		t.Errorf("Format = %q", got)
	}
	if got := date.Format(date.MustParse("2026-09-30")); got != "30 Sep 2026" {
		t.Errorf("Format(September) = %q, want 30 Sep 2026", got)
	}
	if got := date.Format(date.Date{}); got != "" {
		t.Errorf("Format(zero) = %q", got)
	}
	if got := date.FormatRange(date.MustParse("2026-02-01"), date.MustParse("2026-11-30")); got != "01 Feb – 30 Nov 2026" {
		t.Errorf("FormatRange(same year) = %q", got)
	}
	if got := date.FormatRange(date.MustParse("2025-12-01"), date.MustParse("2026-01-31")); got != "01 Dec 2025 – 31 Jan 2026" {
		t.Errorf("FormatRange(two years) = %q", got)
	}
	if got := date.FormatRangeFull(date.MustParse("2026-03-01"), date.MustParse("2026-12-31")); got != "01 Mar 2026 – 31 Dec 2026" {
		t.Errorf("FormatRangeFull = %q", got)
	}
}

// MustParse panics on a wrong date, so seed and test data cannot hide a typing error.
func TestMustParse_PanicsOnWrongDate_D101(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Error("MustParse(2026-02-30) did not panic")
		}
	}()
	date.MustParse("2026-02-30")
}
