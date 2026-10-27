package lineitems

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/errs"
	"mapp/internal/settings"
)

// Every problem of a form is reported at once, and nothing is saved (FR-009, SC-005, US2 scenario 5).
func TestCreate_AllProblemsAtOnce_FR009_SC005(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.create(t, priya, launchFilm, input("Agency fee", "USD", "50000.00"))
	tests := []struct {
		name string
		in   func(in Input) Input
		want []errs.FieldError
	}{
		{"empty name, negative cost and month", func(in Input) Input {
			in.Name, in.PlannedCost = "  ", text("-50")
			in.Spread = SpreadInput{Method: SpreadCustom, Months: []SpreadMonthInput{{Month: "2026-03", Amount: "-1"}}}
			return in
		}, []errs.FieldError{
			{Field: "name", Message: "Enter a name for the line item."},
			{Field: "plannedCost", Message: "Planned cost can't be negative."},
			{Field: "spread.months[0].amount", Message: "Month amounts can't be negative."},
		}},
		{"name taken and no planned cost", func(in Input) Input {
			in.Name, in.PlannedCost = " agency FEE ", nil
			return in
		}, []errs.FieldError{
			{Field: "name", Message: "'Launch film' already has a line item called 'Agency fee'."},
			{Field: "plannedCost", Message: "Enter the planned cost."},
		}},
		{"everything else", func(in Input) Input {
			in.OwnerID, in.Description, in.Currency = text("U-9999"), strings.Repeat("d", 2001), "XYZ"
			in.PlannedCost, in.ContractedAmount = text("12,500"), text("-1")
			return in
		}, []errs.FieldError{
			{Field: "ownerId", Message: "Choose a person from the list."},
			{Field: "description", Message: "Use 2,000 characters or fewer."},
			{Field: "currency", Message: "Choose a currency."},
			{Field: "plannedCost", Message: "Enter an amount, such as 12500.00."},
			{Field: "contractedAmount", Message: "Contracted amount can't be negative."},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.Create(t.Context(), priya, launchFilm, tt.in(input("Media", "USD", "10.00")))

			assertError(t, err, errs.Validation(tt.want...))
		})
	}
	if stored := f.stored(t); len(stored) != 1 {
		t.Errorf("%d line items stored, want only the first", len(stored))
	}
}

// The finance budget year is the plan's year or a year with active items; the item needs a year first, must
// exist, belong to that year and be active when newly chosen (FR-015, FR-016).
func TestCreate_FinanceBudgetChecks_FR015_FR016(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	year := func(y int) *int { return &y }
	tests := []struct {
		name   string
		year   *int
		itemID string
		want   errs.FieldError
	}{
		{"a year without items", year(2027), "", errs.FieldError{Field: "financeBudgetYear", Message: "Choose 2026 or a year with finance budget items."}},
		{"an item without a year", nil, europeDigital, errs.FieldError{Field: "financeBudgetItemId", Message: "Choose the finance budget year first."}},
		{"an unknown item", year(2026), "FBI-9999", errs.FieldError{Field: "financeBudgetItemId", Message: "Choose a finance budget item from the list."}},
		{"an item of another year", year(2026), lastYearItem, errs.FieldError{Field: "financeBudgetItemId", Message: "Choose a finance budget item of 2026."}},
		{"an inactive item", year(2026), europeLegacy, errs.FieldError{Field: "financeBudgetItemId", Message: "Choose an active finance budget item — 'Europe – Legacy' is no longer active."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := input("Fees", "USD", "1.00")
			in.FinanceBudgetYear, in.FinanceBudgetItemID = tt.year, &tt.itemID

			_, err := f.service.Create(t.Context(), priya, launchFilm, in)

			assertError(t, err, errs.Validation(tt.want))
		})
	}

	in := input("Last year's budget", "USD", "1.00")
	in.FinanceBudgetYear, in.FinanceBudgetItemID = year(2025), text(lastYearItem)
	v := f.create(t, priya, launchFilm, in)
	if v.FinanceBudgetYear == nil || *v.FinanceBudgetYear != 2025 || v.FinanceBudgetItem == nil ||
		*v.FinanceBudgetItem != (FinanceBudgetItemRef{ID: lastYearItem, Year: 2025, BudgetName: "2025 Lubricants Budget", Name: "Europe – Digital", Code: "LUB-EU-DIG", Active: true}) {
		t.Errorf("finance budget = %v / %+v", v.FinanceBudgetYear, v.FinanceBudgetItem)
	}
}

// The section lists the activity's line items in creation order, each with its currency and rate, local and
// USD amounts, spread, missing fields and SWBS status; the unassigned planned cost; and the grid (FR-001 –
// FR-003, US1 scenario 1).
func TestSection_RowsInCreationOrder_FR001(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee, film := launchFilmWithTwoItems(t, f)
	f.create(t, priya, socialOnAir, func() Input { in := input("Elsewhere", "USD", "1.00"); in.RestartApproval = true; return in }())

	section, err := f.service.Section(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("Section: %v", err)
	}

	if len(section.Items) != 2 || section.Items[0].ID != fee.ID || section.Items[1].ID != film.ID {
		t.Fatalf("items = %+v, want Agency fee then Film production", section.Items)
	}
	v := section.Items[1]
	if v.Currency.Code != "EUR" || v.Currency.RatePerUSD.String() != "0.92" || v.PlannedCost.Format() != "46,000.00 EUR" ||
		v.PlannedCostUSD.Format() != "50,000.00 USD" || v.SwbsStatus != SwbsWaitingForApproval || len(v.Missing) != 3 {
		t.Errorf("Film production = %+v", v)
	}
	var months []string
	for _, m := range v.Spread.Months {
		months = append(months, m.Month+" "+m.Amount.Format()+" = "+m.AmountUSD.Format())
	}
	want := []string{
		"2026-03 15,333.33 EUR = 16,666.66 USD", "2026-04 15,333.33 EUR = 16,666.66 USD", "2026-05 15,333.34 EUR = 16,666.68 USD",
	}
	if !slices.Equal(months, want) || v.Spread.State != StateSpread || !v.Spread.AddsUp || v.Spread.CustomTotal != nil {
		t.Errorf("spread = %q (%+v), want %q", months, v.Spread, want)
	}
	if b := section.Balance; b.State != BalanceLeft || b.Balance.Format() != "20,000.00 USD" || b.Assigned.Format() != "100,000.00 USD" {
		t.Errorf("balance = %+v, want 20,000.00 USD left of 100,000.00 assigned", b)
	}
	if len(section.Grid.Columns) != 3 || len(section.Grid.Rows) != 2 || section.Grid.Total.Format() != "100,000.00 USD" {
		t.Errorf("grid = %+v", section.Grid)
	}
	if section.Activity.ID != launchFilm || section.Activity.Status != "created" {
		t.Errorf("activity = %+v", section.Activity)
	}
}

// For a person who shows amounts in EUR, the section answers the USD column, the balance and the grid in
// EUR, each amount converted once from its USD figure at 0.92, while plannedCostUsd stays the exact USD
// amount; the line-item form's balance stays in USD, the currency the form calculates in, and the details
// answer EUR like the section (FR-005, D-213). The activity carries its plan's Business unit options
// (D-212).
func TestSection_DisplayCurrency_FR005_D213(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	launchFilmWithTwoItems(t, f)
	facts := f.activities.facts[launchFilm]
	facts.BusinessUnit = []string{"SO-1001"}
	f.activities.facts[launchFilm] = facts

	section, err := f.service.Section(t.Context(), lena, launchFilm)
	if err != nil {
		t.Fatalf("Section: %v", err)
	}
	options, err := f.service.FormOptions(t.Context(), lena, launchFilm)
	if err != nil {
		t.Fatalf("FormOptions: %v", err)
	}
	details, err := f.service.Get(t.Context(), lena, section.Items[1].ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	usd, err := f.service.Section(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("Section (USD): %v", err)
	}

	film := section.Items[1]
	if film.PlannedCostUSD.Format() != "50,000.00 USD" || film.PlannedCostDisplay.Format() != "46,000.00 EUR" {
		t.Errorf("Film production = %s, shown %s; want 50,000.00 USD shown as 46,000.00 EUR", film.PlannedCostUSD.Format(), film.PlannedCostDisplay.Format())
	}
	if b := section.Balance; b.State != BalanceLeft || b.PlannedCost.Format() != "110,400.00 EUR" ||
		b.Assigned.Format() != "92,000.00 EUR" || b.Balance.Format() != "18,400.00 EUR" {
		t.Errorf("balance = %+v, want 18,400.00 EUR left of 110,400.00 EUR", b)
	}
	if section.Grid.Total.Format() != "92,000.00 EUR" || section.Grid.Rows[1].Total.Format() != "46,000.00 EUR" ||
		section.Grid.Rows[1].Cells[2].Format() != "15,333.35 EUR" {
		t.Errorf("grid = %+v, want EUR totals and cells", section.Grid)
	}
	if got := options.Balance.Balance.Format(); got != "20,000.00 USD" {
		t.Errorf("form balance = %s, want 20,000.00 USD", got)
	}
	if details.LineItem.PlannedCostDisplay.Format() != "46,000.00 EUR" || details.Balance.Balance.Format() != "18,400.00 EUR" {
		t.Errorf("details = %s, balance %s; want EUR", details.LineItem.PlannedCostDisplay.Format(), details.Balance.Balance.Format())
	}
	if shown := usd.Items[1].PlannedCostDisplay.Format(); shown != "50,000.00 USD" || usd.Balance.Balance.Format() != "20,000.00 USD" {
		t.Errorf("for a USD reader: %s and balance %s, want USD", shown, usd.Balance.Balance.Format())
	}
	if !slices.Equal(section.Activity.BusinessUnitOptionIDs, []string{"SO-1001"}) || usd.Activity.BusinessUnitOptionIDs == nil {
		t.Errorf("business units = %q", section.Activity.BusinessUnitOptionIDs)
	}
	if empty, _ := f.service.Section(t.Context(), priya, tradeAds); empty.Activity.BusinessUnitOptionIDs == nil {
		t.Errorf("an activity whose plan has no business unit answers nil, want []")
	}
}

// An activity without line items has an empty list, its whole planned cost unassigned and its months as
// grid columns (FR-004).
func TestSection_Empty_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)

	section, err := f.service.Section(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("Section: %v", err)
	}

	if section.Items == nil || len(section.Items) != 0 || section.Balance.Balance.Format() != "120,000.00 USD" ||
		section.Grid.Rows == nil || len(section.Grid.Columns) != 3 || section.Grid.Totals[0] != nil {
		t.Errorf("section = %+v", section)
	}
}

// A deleted activity has no section: "This activity was deleted — …" (spec edge case).
func TestSection_ActivityDeleted_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newTestService(t)

	_, err := f.service.Section(t.Context(), priya, "A-9999")

	assertError(t, err, errs.NotFound(activityGoneMessage))
}

// USD amounts are worked out with the currency's current rate on every read: a rate change in Settings
// changes the USD amounts and the balance at once, writes nothing and restarts no approval (FR-013, D-141).
func TestSection_RateChange_FR013(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	launchFilmWithTwoItems(t, f)
	stored, logged := f.stored(t), len(f.log(t))

	f.currencies.setRate("EUR", "0.80")
	section, err := f.service.Section(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("Section: %v", err)
	}
	if got := section.Items[1].PlannedCostUSD.Format(); got != "57,500.00 USD" || section.Balance.Balance.Format() != "12,500.00 USD" {
		t.Errorf("at 0.80: %s and balance %s, want 57,500.00 USD and 12,500.00 USD", got, section.Balance.Balance.Format())
	}

	f.currencies.setRate("EUR", "0.50")
	section, err = f.service.Section(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("Section: %v", err)
	}
	if b := section.Balance; b.State != BalanceExceeded || b.Balance.Format() != "-22,000.00 USD" {
		t.Errorf("at 0.50: balance %+v, want exceeded by 22,000.00 USD", b)
	}
	if !reflect.DeepEqual(f.stored(t), stored) || len(f.log(t)) != logged || len(f.activities.restarts) != 0 {
		t.Errorf("a rate change wrote line items, log entries or restarts")
	}
}

// A custom spread that does not add up still saves; its total shows how far off it is, and the line item's
// missing list (which blocks submitting the activity) says "Spread doesn't add up to the planned cost"
// (FR-019, US3 scenarios 2 and 3). The activity's grid shows the month outside its dates.
func TestCreate_CustomNotAddingUpSaves_FR019(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	in := complete(input("Venue", "EUR", "10000.00"))
	in.Spread = custom("2026-02", "2000.00", "2026-03", "3333.33", "2026-04", "3333.33", "2026-05", "3333.34")

	v, err := f.service.Create(t.Context(), priya, launchFilm, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if v.Spread.AddsUp || v.Spread.CustomTotal == nil || v.Spread.CustomTotal.Format() != "12,000.00 EUR" {
		t.Errorf("spread adds up %v, total %v; want 12,000.00 EUR, not adding up", v.Spread.AddsUp, v.Spread.CustomTotal)
	}
	want := []settings.MissingField{{Field: "spread", Label: "Spread doesn't add up to the planned cost"}}
	if !reflect.DeepEqual(v.Missing, want) {
		t.Errorf("missing = %+v, want %+v", v.Missing, want)
	}
	if got := monthsOf(v.Spread); len(got) != 4 || got[0] != "2026-02 2,000.00 EUR outside" {
		t.Errorf("months = %v", got)
	}

	section, err := f.service.Section(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("Section: %v", err)
	}
	if columns := section.Grid.Columns; len(columns) != 4 || columns[0].Key != "2026-02" || !columns[0].OutsideActivityDates ||
		columns[1].OutsideActivityDates {
		t.Errorf("grid columns = %+v, want Feb (outside) to May", columns)
	}

	in.Name, in.PlannedCost = "Venue deposit", text("12000.00")
	if adds := f.create(t, priya, launchFilm, in); !adds.Spread.AddsUp || len(adds.Missing) != 0 {
		t.Errorf("a custom spread that adds up: adds up %v, missing %+v", adds.Spread.AddsUp, adds.Missing)
	}
}
