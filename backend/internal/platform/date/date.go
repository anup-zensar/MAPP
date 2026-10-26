// Package date is MAPP's calendar date: a day without a time or a time zone, written "2026-03-01" in
// JSON (feature 005's design, decision D-198; used by features 004–012). Campaign, initiative and activity
// dates, plan years and "today" are dates; moments such as createdAt stay time.Time.
package date

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrInvalid means a text is not a date in the form YYYY-MM-DD, or is a day that does not exist
// (2026-02-30).
var ErrInvalid = errors.New("date: not a date of the form YYYY-MM-DD")

// layout is the form of a date in JSON, in URLs and in stored records.
const layout = "2006-01-02"

// monthNames are the short month names of every text for people, in English and never "Sept" (D-099).
var monthNames = [...]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// Date is a calendar date. The zero value is "no date" (IsZero); optional dates of records are *Date, nil
// when not set, which JSON writes as null. Dates compare with ==.
type Date struct {
	year  int
	month time.Month
	day   int
}

// New returns the date year-month-day. Values out of range are normalised as time.Date does: New(2026,
// time.February, 30) is 2 March 2026.
func New(year int, month time.Month, day int) Date {
	return fromTime(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

// Parse reads a date in the form YYYY-MM-DD ("2026-03-01"). Anything else, or a day that does not exist,
// is ErrInvalid.
func Parse(s string) (Date, error) {
	t, err := time.Parse(layout, s)
	if err != nil {
		return Date{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	return fromTime(t), nil
}

// MustParse is Parse for dates known to be right, such as those of the seed and of tests; it panics on a
// wrong one.
func MustParse(s string) Date {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Today returns the calendar date of now in UTC (D-091): the "today" of timings and display statuses.
func Today(now time.Time) Date {
	return fromTime(now.UTC())
}

// YearStart returns 1 January of year.
func YearStart(year int) Date {
	return Date{year: year, month: time.January, day: 1}
}

// YearEnd returns 31 December of year.
func YearEnd(year int) Date {
	return Date{year: year, month: time.December, day: 31}
}

// fromTime returns the calendar date of t, read in t's own location.
func fromTime(t time.Time) Date {
	year, month, day := t.Date()
	return Date{year: year, month: month, day: day}
}

// Time returns the start of the date (00:00) in UTC.
func (d Date) Time() time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC)
}

// IsZero reports whether d is the zero Date, "no date".
func (d Date) IsZero() bool {
	return d == Date{}
}

// Year returns the year of the date.
func (d Date) Year() int { return d.year }

// Month returns the month of the date.
func (d Date) Month() time.Month { return d.month }

// Day returns the day of the month.
func (d Date) Day() int { return d.day }

// Compare returns −1 when d is before o, 0 when they are the same day, +1 when d is after o.
func (d Date) Compare(o Date) int {
	switch {
	case d.year != o.year:
		return compareInts(d.year, o.year)
	case d.month != o.month:
		return compareInts(int(d.month), int(o.month))
	}
	return compareInts(d.day, o.day)
}

// compareInts returns −1, 0 or +1.
func compareInts(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// Before reports whether d is before o.
func (d Date) Before(o Date) bool { return d.Compare(o) < 0 }

// After reports whether d is after o.
func (d Date) After(o Date) bool { return d.Compare(o) > 0 }

// AddYears returns the same day n years later (or earlier, for a negative n). 29 February becomes 28
// February when the target year has no 29 February (plan copies to the next year, R13).
func (d Date) AddYears(n int) Date {
	year := d.year + n
	if d.month == time.February && d.day == 29 && !isLeap(year) {
		return Date{year: year, month: time.February, day: 28}
	}
	return Date{year: year, month: d.month, day: d.day}
}

// AddDays returns the date n days later (or earlier, for a negative n).
func (d Date) AddDays(n int) Date {
	return fromTime(d.Time().AddDate(0, 0, n))
}

// isLeap reports whether year has a 29 February.
func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// String writes the date as YYYY-MM-DD ("2026-03-01"); the zero Date is "".
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.Time().Format(layout)
}

// MarshalJSON writes the date as a JSON string, "2026-03-01"; the zero Date as null.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

// UnmarshalJSON reads a JSON string of the form YYYY-MM-DD; anything else is an error, so a request with a
// wrong date is refused as unreadable. JSON null leaves the date as it is (a *Date field becomes nil).
func (d *Date) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("%w: a date is a JSON string such as \"2026-03-01\"", ErrInvalid)
	}
	parsed, err := Parse(text)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Format writes a date as people read it in texts of the API (log entries, messages): two-digit day,
// short English month, year — "01 Mar 2026" (D-099). The zero Date is "".
func Format(d Date) string {
	if d.IsZero() {
		return ""
	}
	return fmt.Sprintf("%02d %s %d", d.day, monthNames[d.month-1], d.year)
}

// FormatRange writes a date range, with the year once when both dates are in the same year: "01 Feb – 30
// Nov 2026", otherwise "01 Dec 2025 – 31 Jan 2026".
func FormatRange(start, end Date) string {
	if start.year == end.year {
		return fmt.Sprintf("%02d %s – %s", start.day, monthNames[start.month-1], Format(end))
	}
	return FormatRangeFull(start, end)
}

// FormatRangeFull writes a date range with both years always: "01 Mar 2026 – 31 Dec 2026" (the R3
// message).
func FormatRangeFull(start, end Date) string {
	return Format(start) + " – " + Format(end)
}
