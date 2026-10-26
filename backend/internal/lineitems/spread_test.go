package lineitems

import (
	"fmt"
	"slices"
	"testing"

	"mapp/internal/platform/money"
)

// dated returns facts with the dates start and end ("" for none) and ongoing.
func dated(start, end string, ongoing bool) ActivityFacts {
	f := ActivityFacts{Ongoing: ongoing, PlanYear: 2026}
	if start != "" {
		f.StartDate = day(start)
	}
	if end != "" {
		f.EndDate = day(end)
	}
	return f
}

// monthTexts writes month amounts as "2026-03 3333.33".
func monthTexts(months []MonthAmount) []string {
	texts := []string{}
	for _, m := range months {
		texts = append(texts, m.Month+" "+m.Amount.String())
	}
	return texts
}

// The activity's months are the calendar months from its start month to its end month, both included
// (data model §3.2, FR-017).
func TestActivityMonths_CalendarMonths_FR017(t *testing.T) {
	t.Parallel()
	tests := []struct {
		start, end string
		want       []string
	}{
		{"2026-03-01", "2026-05-31", []string{"2026-03", "2026-04", "2026-05"}},
		{"2026-12-15", "2027-01-02", []string{"2026-12", "2027-01"}},
		{"2026-06-10", "2026-06-20", []string{"2026-06"}},
	}
	for _, tt := range tests {
		months, hasDates := ActivityMonths(dated(tt.start, tt.end, false))

		if !hasDates || !slices.Equal(months, tt.want) {
			t.Errorf("%s – %s: %v, %v; want %v", tt.start, tt.end, months, hasDates, tt.want)
		}
	}
}

// An ongoing activity has its start month only, for spreading; without a start date, or a start date
// without an end date that is not ongoing, there are no dates yet (FR-021, D-032).
func TestActivityMonths_OngoingAndNoDates_FR021(t *testing.T) {
	t.Parallel()
	if months, has := ActivityMonths(dated("2026-03-15", "", true)); !has || !slices.Equal(months, []string{"2026-03"}) {
		t.Errorf("ongoing: %v, %v; want [2026-03]", months, has)
	}
	for _, f := range []ActivityFacts{dated("", "", false), dated("2026-03-01", "", false), dated("", "2026-05-31", false)} {
		if months, has := ActivityMonths(f); has || months == nil || len(months) != 0 {
			t.Errorf("%+v: %v, %v; want no dates and an empty list", f, months, has)
		}
	}
}

// Evenly gives each month the planned cost ÷ the number of months, rounded down to the cent, and the rest
// to the last month, so the months always add up exactly (FR-017, SC-002, the plan's example table).
func TestSpread_EvenlyRemainderLast_FR017_SC002(t *testing.T) {
	t.Parallel()
	marToMay := dated("2026-03-01", "2026-05-31", false)
	tests := []struct {
		cost string
		want []string
	}{
		{"10000.00", []string{"2026-03 3333.33", "2026-04 3333.33", "2026-05 3333.34"}},
		{"0.00", []string{"2026-03 0.00", "2026-04 0.00", "2026-05 0.00"}},
		{"0.02", []string{"2026-03 0.00", "2026-04 0.00", "2026-05 0.02"}},
	}
	for _, tt := range tests {
		months, spread := SpreadMonths(lineItem(tt.cost, SpreadEvenly), marToMay)

		if got := monthTexts(months); !spread || !slices.Equal(got, tt.want) {
			t.Errorf("%s evenly: %v, want %v", tt.cost, got, tt.want)
		}
	}

	for n := 1; n <= 36; n++ {
		f := dated("2026-01-01", "", false)
		f.EndDate = day(fmt.Sprintf("%04d-%02d-01", 2026+(n-1)/12, (n-1)%12+1))
		for _, cost := range []string{"0.01", "10000.00", "123456.79", "999999999999.99"} {
			months, _ := SpreadMonths(lineItem(cost, SpreadEvenly), f)
			var sum money.Amount
			for _, m := range months {
				sum = sum.Add(m.Amount)
			}
			if len(months) != n || sum != amount(cost) {
				t.Errorf("%s over %d months: %d months adding up to %s", cost, n, len(months), sum)
			}
		}
	}
}

// All in first month and All in last month put everything in that month (FR-017).
func TestSpread_FirstAndLastMonth_FR017(t *testing.T) {
	t.Parallel()
	f := dated("2026-03-01", "2026-05-31", false)

	first, _ := SpreadMonths(lineItem("10000.00", SpreadFirstMonth), f)
	last, _ := SpreadMonths(lineItem("10000.00", SpreadLastMonth), f)

	if got := monthTexts(first); !slices.Equal(got, []string{"2026-03 10000.00"}) {
		t.Errorf("first month: %v", got)
	}
	if got := monthTexts(last); !slices.Equal(got, []string{"2026-05 10000.00"}) {
		t.Errorf("last month: %v", got)
	}
}

// For an ongoing activity, Evenly and All in last month behave like All in first month (FR-021, US3
// scenario 4).
func TestSpread_OngoingLikeFirst_FR021(t *testing.T) {
	t.Parallel()
	f := dated("2026-03-01", "", true)
	for _, method := range []SpreadMethod{SpreadEvenly, SpreadFirstMonth, SpreadLastMonth} {
		months, spread := SpreadMonths(lineItem("10000.00", method), f)

		if got := monthTexts(months); !spread || !slices.Equal(got, []string{"2026-03 10000.00"}) {
			t.Errorf("%s: %v, want everything in Mar 2026", method, got)
		}
	}
}

// Without activity dates, Evenly, first and last are not spread yet; a custom spread keeps its months
// (FR-017, D-032).
func TestSpread_NoDatesNotSpread_FR017(t *testing.T) {
	t.Parallel()
	f := dated("", "", false)
	for _, method := range []SpreadMethod{SpreadEvenly, SpreadFirstMonth, SpreadLastMonth} {
		if months, spread := SpreadMonths(lineItem("100.00", method), f); spread || months == nil || len(months) != 0 {
			t.Errorf("%s without dates: %v, %v; want not spread", method, months, spread)
		}
	}
	months, spread := SpreadMonths(lineItem("100.00", SpreadCustom, "2026-02", "100.00"), f)
	if got := monthTexts(months); !spread || !slices.Equal(got, []string{"2026-02 100.00"}) {
		t.Errorf("custom without dates: %v, %v", got, spread)
	}
	if OutsideActivityDates("2026-02", f) {
		t.Error("a month is outside the dates of an activity that has none")
	}
}

// A custom spread is kept as typed, sorted by month; months before or after the activity's dates are marked
// as outside them, which is allowed (FR-018, US3 scenario 2).
func TestSpread_CustomOutsideMarked_FR018(t *testing.T) {
	t.Parallel()
	f := dated("2026-03-01", "2026-05-31", false)
	li := lineItem("12000.00", SpreadCustom, "2026-04", "3333.33", "2026-02", "2000.00", "2026-03", "3333.33", "2026-06", "3333.34")

	months, _ := SpreadMonths(li, f)

	if got := monthTexts(months); !slices.Equal(got, []string{"2026-02 2000.00", "2026-03 3333.33", "2026-04 3333.33", "2026-06 3333.34"}) {
		t.Errorf("custom months = %v", got)
	}
	var outside []string
	for _, m := range months {
		if OutsideActivityDates(m.Month, f) {
			outside = append(outside, m.Month)
		}
	}
	if !slices.Equal(outside, []string{"2026-02", "2026-06"}) {
		t.Errorf("outside = %v, want Feb and Jun", outside)
	}
	ongoing := dated("2026-03-01", "", true)
	if !OutsideActivityDates("2026-02", ongoing) || OutsideActivityDates("2027-06", ongoing) {
		t.Error("ongoing: only months before the start month are outside")
	}
}

// Evenly, first and last are worked out from the activity's current dates and the line item's current planned
// cost, so they follow every change; custom months are kept, marked when they fall outside the new dates
// (FR-022, D-140, US3 scenario 5).
func TestSpread_FollowsDatesAndCost_FR022(t *testing.T) {
	t.Parallel()
	moved := dated("2026-04-01", "2026-06-30", false)
	tests := []struct {
		li   LineItem
		want []string
	}{
		{lineItem("10000.00", SpreadEvenly), []string{"2026-04 3333.33", "2026-05 3333.33", "2026-06 3333.34"}},
		{lineItem("10000.00", SpreadFirstMonth), []string{"2026-04 10000.00"}},
		{lineItem("10000.00", SpreadLastMonth), []string{"2026-06 10000.00"}},
		{lineItem("12000.00", SpreadEvenly), []string{"2026-04 4000.00", "2026-05 4000.00", "2026-06 4000.00"}},
		{lineItem("9000.00", SpreadCustom, "2026-03", "3000.00", "2026-04", "6000.00"), []string{"2026-03 3000.00", "2026-04 6000.00"}},
	}
	for _, tt := range tests {
		months, _ := SpreadMonths(tt.li, moved)

		if got := monthTexts(months); !slices.Equal(got, tt.want) {
			t.Errorf("%s %s: %v, want %v", tt.li.Spread.Method, tt.li.PlannedCost, got, tt.want)
		}
	}
	if !OutsideActivityDates("2026-03", moved) {
		t.Error("Mar 2026 is now outside the activity's dates")
	}
}

// The USD months of every amount, rate and number of months add up exactly to the USD planned cost (FR-020,
// SC-002, D-024).
func TestUSDMonths_AddUp_FR020_SC002(t *testing.T) {
	t.Parallel()
	for _, r := range []string{"1", "0.92", "0.79", "7.10", "83.50"} {
		for n := 1; n <= 36; n++ {
			f := dated("2026-01-01", fmt.Sprintf("%04d-%02d-28", 2026+(n-1)/12, (n-1)%12+1), false)
			for _, cost := range []string{"0.01", "0.07", "10000.00", "46000.00", "123456.78"} {
				li := lineItem(cost, SpreadEvenly)
				months, _ := SpreadMonths(li, f)

				usd := USDMonths(months, rate(r))

				if sum := money.Sum(usd...); sum != PlannedCostUSD(li, rate(r)) || len(usd) != n {
					t.Errorf("%s at %s over %d months: USD months add up to %s, want %s", cost, r, n, sum, PlannedCostUSD(li, rate(r)))
				}
			}
		}
	}
}

// The rounding cent goes into the last month that has an amount (FR-020): 10,000.00 EUR evenly over Mar–May
// at 0.92 → 3,623.18 · 3,623.18 · 3,623.21 USD; a custom spread ending with 0.00 months keeps them at 0.00.
func TestUSDMonths_CentInLastNonZeroMonth_FR020(t *testing.T) {
	t.Parallel()
	even, _ := SpreadMonths(lineItem("10000.00", SpreadEvenly), dated("2026-03-01", "2026-05-31", false))
	if got := USDMonths(even, rate("0.92")); !slices.Equal(got, amounts("3623.18", "3623.18", "3623.21")) {
		t.Errorf("evenly at 0.92 = %v", got)
	}

	custom := lineItem("0.05", SpreadCustom, "2026-03", "0.01", "2026-04", "0.01", "2026-05", "0.01", "2026-06", "0.01", "2026-07", "0.01", "2026-08", "0.00")
	got := USDMonths(custom.Spread.Months, rate("7.10")) // 0.05 CNY = 0.01 USD; each 0.01 CNY = 0.00 USD
	if !slices.Equal(got, amounts("0.00", "0.00", "0.00", "0.00", "0.01", "0.00")) {
		t.Errorf("custom at 7.10 = %v, want the cent in Jul, the last month with an amount", got)
	}
	if got := USDMonths([]MonthAmount{{Month: "2026-03"}}, rate("0.92")); !slices.Equal(got, amounts("0.00")) {
		t.Errorf("0.00 = %v", got)
	}
}
