package lineitem

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// The spread rules of domain model §4.4 and data model §3.2–§3.4, as pure functions. Only custom month
// amounts are stored; the months of the other methods are calculated on every read from the planned cost and
// the activity's current dates, so a date or cost change re-spreads them without a write (D-140, FR-022).
// Features 010 and 013 use these functions too.

// monthText is the form of a month: "2026-03".
var monthText = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])$`)

// monthOf returns the month of d: "2026-03".
func monthOf(d date.Date) string {
	return fmt.Sprintf("%04d-%02d", d.Year(), int(d.Month()))
}

// nextMonth returns the month after month ("2026-12" → "2027-01"); month has the form of monthText.
func nextMonth(month string) string {
	year, _ := strconv.Atoi(month[:4])
	m, _ := strconv.Atoi(month[5:])
	if m == 12 {
		return fmt.Sprintf("%04d-01", year+1)
	}
	return fmt.Sprintf("%04d-%02d", year, m+1)
}

// monthLabel writes a month as people read it: "2026-02" → "Feb 2026".
func monthLabel(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.Format("Jan 2006")
}

// ActivityMonths returns the months of the activity's dates, "YYYY-MM" (data model §3.2): every calendar
// month from the start month to the end month, both included; for an ongoing activity only its start month.
// hasDates is false, and months empty, when the activity has no start date, or a start date without an end
// date while not ongoing ("No dates yet", D-032).
func ActivityMonths(f ActivityFacts) (months []string, hasDates bool) {
	switch {
	case f.StartDate == nil:
		return []string{}, false
	case f.Ongoing:
		return []string{monthOf(*f.StartDate)}, true
	case f.EndDate == nil:
		return []string{}, false
	}
	last := monthOf(*f.EndDate)
	months = []string{}
	for month := monthOf(*f.StartDate); month <= last; month = nextMonth(month) {
		months = append(months, month)
	}
	return months, true
}

// OutsideActivityDates reports whether month lies outside the activity's dates: the activity has dates and
// the month is before its start month or after its end month (for an ongoing activity: before its start
// month). Only custom months can be outside (FR-018).
func OutsideActivityDates(month string, f ActivityFacts) bool {
	if _, hasDates := ActivityMonths(f); !hasDates {
		return false
	}
	if month < monthOf(*f.StartDate) {
		return true
	}
	return !f.Ongoing && month > monthOf(*f.EndDate)
}

// SpreadMonths returns the month amounts of li in its currency, sorted by month (data model §3.3), and
// whether it is spread:
//
//   - Evenly: the planned cost ÷ the number of activity months, rounded down to the cent, the rest in the
//     last month (10,000.00 over 3 → 3,333.33 · 3,333.33 · 3,333.34);
//   - All in first month / All in last month: everything in that month;
//   - Custom: the stored months, as typed;
//   - an ongoing activity: Evenly and All in last month behave like All in first month (FR-021);
//   - an activity without dates: Evenly, first and last are not spread (spread false, no months).
func SpreadMonths(li LineItem, f ActivityFacts) (months []MonthAmount, spread bool) {
	if li.Spread.Method == SpreadCustom {
		months = slices.Clone(li.Spread.Months)
		slices.SortFunc(months, func(x, y MonthAmount) int { return strings.Compare(x.Month, y.Month) })
		return orEmpty(months), true
	}
	activityMonths, hasDates := ActivityMonths(f)
	if !hasDates {
		return []MonthAmount{}, false
	}
	switch {
	case f.Ongoing || li.Spread.Method == SpreadFirstMonth:
		return []MonthAmount{{Month: activityMonths[0], Amount: li.PlannedCost}}, true
	case li.Spread.Method == SpreadLastMonth:
		return []MonthAmount{{Month: activityMonths[len(activityMonths)-1], Amount: li.PlannedCost}}, true
	}
	parts := li.PlannedCost.SplitEvenly(len(activityMonths))
	months = make([]MonthAmount, len(parts))
	for i, part := range parts {
		months[i] = MonthAmount{Month: activityMonths[i], Amount: part}
	}
	return months, true
}

// USDMonths converts month amounts in a currency to USD with its rate (data model §3.4, D-024, FR-020):
// each month half-up to the cent, and the difference between the USD value of their sum and the sum of the
// converted months in the last month that has a non-zero amount, so they add up exactly. 10,000.00 EUR
// evenly over three months at 0.92 → 3,623.18 · 3,623.18 · 3,623.21 USD = 10,869.57 USD.
func USDMonths(months []MonthAmount, rate money.Rate) []money.Amount {
	return ConvertMonths(months, rate, money.One)
}

// ConvertMonths converts month amounts between two currencies with their rates per 1 USD, as USDMonths does
// into USD (016 FR-015: into the plan currency). The same rate keeps the amounts as they are.
func ConvertMonths(months []MonthAmount, from, to money.Rate) []money.Amount {
	usd := make([]money.Amount, len(months))
	last := -1 // the last month with a non-zero amount
	for i, m := range months {
		if !m.Amount.IsZero() {
			last = i
		}
	}
	if last < 0 {
		return usd // every month is 0.00, in USD too
	}
	parts := make([]money.Amount, last+1)
	for i := range parts {
		parts[i] = months[i].Amount
	}
	copy(usd, money.ConvertParts(parts, from, to)) // the months after the last non-zero one stay 0.00
	return usd
}

// PlannedCostUSD converts the line item's planned cost to USD with its currency's rate, half-up to the cent
// (domain model §4.3): 46,000.00 EUR at 0.92 → 50,000.00 USD.
func PlannedCostUSD(li LineItem, rate money.Rate) money.Amount {
	return money.ToUSD(li.PlannedCost, rate)
}

// orEmpty returns list, or an empty list instead of nil, so the JSON has [] and never null.
func orEmpty[T any](list []T) []T {
	if list == nil {
		return []T{}
	}
	return list
}
