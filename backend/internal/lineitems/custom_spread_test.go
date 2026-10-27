package lineitems

import (
	"reflect"
	"slices"
	"testing"

	"mapp/internal/platform/errs"
	"mapp/internal/settings"
)

// custom returns a custom spread of the months and amounts given as pairs: "2026-02", "2000.00", ….
func custom(pairs ...string) SpreadInput {
	months := []SpreadMonthInput{}
	for i := 0; i+1 < len(pairs); i += 2 {
		months = append(months, SpreadMonthInput{Month: pairs[i], Amount: pairs[i+1]})
	}
	return SpreadInput{Method: SpreadCustom, Months: months}
}

// monthsOf returns the months of a spread view as "2026-02 2,000.00 EUR", with " outside" for a month
// outside the activity's dates.
func monthsOf(v SpreadView) []string {
	list := []string{}
	for _, m := range v.Months {
		text := m.Month + " " + m.Amount.Format()
		if m.OutsideActivityDates {
			text += " outside"
		}
		list = append(list, text)
	}
	return list
}

// Custom months are months from January of the plan's year − 1 to December of the plan's year + 10, each at
// most once, with an amount of 0.00 or more; every problem is reported at once, on the month's own field
// (FR-018, D-152). Months outside the activity's dates are allowed, such as a deposit; the stored months are
// sorted. Only a custom spread keeps months.
func TestCreate_CustomMonthChecks_FR018(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	in := input("Venue", "EUR", "10000.00")
	in.Spread = custom("2024-12", "1.00", "2026-03", "abc", "2026-03", "-1", "2037-01", "1.234", "March", "2", "2026-04", "1000000000000")

	_, err := f.service.Create(t.Context(), priya, launchFilm, in)

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "spread.months[0].month", Message: "Choose a month from Jan 2025 to Dec 2036."},
		errs.FieldError{Field: "spread.months[1].amount", Message: "Enter an amount, such as 12500.00."},
		errs.FieldError{Field: "spread.months[2].month", Message: "Each month can appear only once."},
		errs.FieldError{Field: "spread.months[2].amount", Message: "Month amounts can't be negative."},
		errs.FieldError{Field: "spread.months[3].month", Message: "Choose a month from Jan 2025 to Dec 2036."},
		errs.FieldError{Field: "spread.months[3].amount", Message: "Enter an amount with at most two decimals."},
		errs.FieldError{Field: "spread.months[4].month", Message: "Choose a month from Jan 2025 to Dec 2036."},
		errs.FieldError{Field: "spread.months[5].amount", Message: "Enter an amount below one trillion."},
	))
	in.Spread = SpreadInput{Method: "weekly", Months: []SpreadMonthInput{}}
	_, err = f.service.Create(t.Context(), priya, launchFilm, in)
	assertError(t, err, errs.Validation(errs.FieldError{Field: "spread.method", Message: "Choose how to spread the planned cost over time."}))
	if len(f.stored(t)) != 0 {
		t.Fatal("a form with problems stored a line item")
	}

	in.Spread = custom("2036-12", "0", "2026-04", "4000", "2025-01", "1000.00", "2026-02", "5000")
	v := f.create(t, priya, launchFilm, in)
	evenly := input("Agency fee", "EUR", "10000.00")
	evenly.Spread.Months = custom("2026-02", "10000").Months // months sent with another method are not kept
	plain := f.create(t, priya, launchFilm, evenly)

	want := []string{"2025-01 1,000.00 EUR outside", "2026-02 5,000.00 EUR outside", "2026-04 4,000.00 EUR", "2036-12 0.00 EUR outside"}
	if got := monthsOf(v.Spread); !slices.Equal(got, want) {
		t.Errorf("months = %v\nwant     %v", got, want)
	}
	stored := f.stored(t)
	if months := stored[0].Spread.Months; len(months) != 4 || months[0].Month != "2025-01" || months[3].Month != "2036-12" {
		t.Errorf("stored months = %+v, want sorted by month", months)
	}
	if stored[1].Spread.Method != SpreadEvenly || len(stored[1].Spread.Months) != 0 || len(plain.Spread.Months) != 3 {
		t.Errorf("evenly: stored %+v, shown %v", stored[1].Spread, monthsOf(plain.Spread))
	}
}

// Switching a line item away from Custom drops its typed months: the new method spreads the planned cost
// over the activity's months, and the log shows the change (FR-018, US3 scenario 6).
func TestUpdate_LeavingCustomDropsMonths_FR018(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	in := input("Venue", "EUR", "10000.00")
	in.Spread = custom("2026-02", "2000.00", "2026-03", "8000.00")
	venue := f.create(t, priya, launchFilm, in)

	u := formOf(venue) // still sends the custom months, as a form may
	u.Spread.Method = SpreadEvenly
	saved := f.update(t, priya, venue.ID, u)

	if stored := f.stored(t)[0].Spread; stored.Method != SpreadEvenly || len(stored.Months) != 0 {
		t.Errorf("stored spread = %+v, want evenly without months", stored)
	}
	want := []string{"2026-03 3,333.33 EUR", "2026-04 3,333.33 EUR", "2026-05 3,333.34 EUR"}
	if got := monthsOf(saved.Spread); !slices.Equal(got, want) || saved.Spread.CustomTotal != nil || !saved.Spread.AddsUp {
		t.Errorf("months = %v (total %v), want %v", got, saved.Spread.CustomTotal, want)
	}
	entry := f.log(t)[0]
	if len(entry.Changes) != 1 || entry.Changes[0].OldText != "Custom: Feb 2026 2,000.00 · Mar 2026 8,000.00" || entry.Changes[0].NewText != "Evenly" {
		t.Errorf("log changes = %+v", entry.Changes)
	}
}

// When the planned cost or the currency changes, spreads other than Custom follow it, while custom months
// stay as typed (in the line item's currency) and are flagged when they no longer add up (FR-022).
func TestUpdate_CostChangeKeepsCustom_FR022(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	in := complete(input("Venue", "EUR", "10000.00"))
	in.Spread = custom("2026-03", "3000.00", "2026-04", "3000.00", "2026-05", "4000.00")
	venue := f.create(t, priya, launchFilm, in)
	fee := f.create(t, priya, launchFilm, complete(input("Agency fee", "EUR", "9000.00")))

	u := formOf(venue)
	u.PlannedCost = text("12000.00")
	costChanged := f.update(t, priya, venue.ID, u)
	u = formOf(costChanged)
	u.Currency = "GBP"
	currencyChanged := f.update(t, priya, venue.ID, u)
	u = formOf(fee)
	u.PlannedCost = text("12000.00")
	feeChanged := f.update(t, priya, fee.ID, u)

	wantCustom := []string{"2026-03 3,000.00 EUR", "2026-04 3,000.00 EUR", "2026-05 4,000.00 EUR"}
	if got := monthsOf(costChanged.Spread); !slices.Equal(got, wantCustom) || costChanged.Spread.AddsUp {
		t.Errorf("custom after a cost change = %v (adds up %v), want %v, not adding up", got, costChanged.Spread.AddsUp, wantCustom)
	}
	want := []settings.MissingField{{Field: "spread", Label: "Spread doesn't add up to the planned cost"}}
	if !reflect.DeepEqual(costChanged.Missing, want) {
		t.Errorf("missing = %+v, want %+v", costChanged.Missing, want)
	}
	wantGBP := []string{"2026-03 3,000.00 GBP", "2026-04 3,000.00 GBP", "2026-05 4,000.00 GBP"}
	if got := monthsOf(currencyChanged.Spread); !slices.Equal(got, wantGBP) {
		t.Errorf("custom after a currency change = %v, want %v", got, wantGBP)
	}
	wantEvenly := []string{"2026-03 4,000.00 EUR", "2026-04 4,000.00 EUR", "2026-05 4,000.00 EUR"}
	if got := monthsOf(feeChanged.Spread); !slices.Equal(got, wantEvenly) {
		t.Errorf("evenly after a cost change = %v, want %v", got, wantEvenly)
	}
}
