package lineitems

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"northstar/internal/history"
	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/settings"
)

// usd returns the money of a USD amount text.
func usd(s string) money.Money {
	return money.USD(amount(s))
}

// launchFilmWithTwoItems adds the spec's two line items to Launch film: Agency fee 50,000.00 USD and Film
// production 46,000.00 EUR (= 50,000.00 USD at 0.92), leaving 20,000.00 USD unassigned.
func launchFilmWithTwoItems(t *testing.T, f *fixture) (fee, film View) {
	t.Helper()
	fee = f.create(t, priya, launchFilm, complete(input("Agency fee", "USD", "50000.00")))
	film = f.create(t, priya, launchFilm, input("Film production", "EUR", "46000.00"))
	return fee, film
}

// A new line item starts filled in (FR-008): the name "<activity name> – line item", the activity's owner,
// USD, the unassigned planned cost, the plan's default spread and year, no finance budget item when the
// plan's default budget has several, and Request SWBS code from GSAP on; with the activity and its balance.
func TestFormOptions_Defaults_FR008(t *testing.T) {
	f := newTestService(t)
	launchFilmWithTwoItems(t, f)

	options, err := f.service.FormOptions(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("FormOptions: %v", err)
	}

	owner := people.PersonRef{ID: priya.ID, Name: priya.Name, Initials: "PR", JobTitle: "Marketer", Active: true}
	balance := usd("20000.00")
	want := Defaults{
		Name: "Launch film – line item", Owner: &owner, Currency: "USD", PlannedCost: &balance,
		PlannedCostSource: SourceUnassignedBalance, SpreadMethod: SpreadEvenly, FinanceBudgetYear: 2026,
		RequestSwbs: true,
	}
	if !reflect.DeepEqual(options.Defaults, want) {
		t.Errorf("defaults = %+v\nwant       %+v", options.Defaults, want)
	}
	planned := usd("120000.00")
	wantBalance := UnassignedBalance{PlannedCost: &planned, Assigned: usd("100000.00"), Balance: &balance, State: BalanceLeft}
	if !reflect.DeepEqual(options.Balance, wantBalance) {
		t.Errorf("balance = %+v, want %+v", options.Balance, wantBalance)
	}
	if a := options.Activity; a.ID != launchFilm || a.Name != "Launch film" || a.RestartsApproval || a.PlanYear != 2026 ||
		a.DefaultFinanceBudget == nil || *a.DefaultFinanceBudget != "2026 Lubricants Budget" || len(a.Path) != 3 {
		t.Errorf("activity = %+v", a)
	}
}

// A default name that is taken gets " 2", " 3", … (ignoring letter case), and a long activity name is
// shortened so that the default name fits 120 characters (FR-008).
func TestFormOptions_NameTaken_FR008(t *testing.T) {
	f := newTestService(t)
	f.create(t, priya, launchFilm, input("Launch film – line item", "USD", "1.00"))
	f.create(t, priya, launchFilm, input("LAUNCH FILM – LINE ITEM 2", "USD", "1.00"))

	options, err := f.service.FormOptions(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("FormOptions: %v", err)
	}
	if options.Defaults.Name != "Launch film – line item 3" {
		t.Errorf("name = %q, want %q", options.Defaults.Name, "Launch film – line item 3")
	}

	long := f.activities.facts[launchFilm]
	long.Name = strings.Repeat("x", 120)
	f.activities.facts[launchFilm] = long
	options, err = f.service.FormOptions(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("FormOptions: %v", err)
	}
	if want := strings.Repeat("x", 108) + " – line item"; options.Defaults.Name != want {
		t.Errorf("name = %q (%d characters), want %q", options.Defaults.Name, len([]rune(options.Defaults.Name)), want)
	}
}

// When nothing is left to assign, or the activity has no planned cost, the planned cost starts empty and
// says why (spec edge case "Zero amounts", D-144).
func TestFormOptions_NothingLeft_EdgeCase(t *testing.T) {
	f := newTestService(t)
	f.create(t, priya, launchFilm, input("Everything", "USD", "120000.00"))
	f.create(t, priya, retailEvent, input("Too much", "USD", "6000.00"))

	for _, tt := range []struct {
		activityID, source string
	}{
		{launchFilm, SourceNothingLeftToAssign},  // 0.00 left
		{retailEvent, SourceNothingLeftToAssign}, // exceeded by 1,000.00
		{lunarPromo, SourceNoActivityPlannedCost},
	} {
		options, err := f.service.FormOptions(t.Context(), priya, tt.activityID)
		if err != nil {
			t.Fatalf("FormOptions(%s): %v", tt.activityID, err)
		}
		if d := options.Defaults; d.PlannedCost != nil || d.PlannedCostSource != tt.source {
			t.Errorf("%s: planned cost %v from %q, want none from %q", tt.activityID, d.PlannedCost, d.PlannedCostSource, tt.source)
		}
	}
}

// The form offers the plan's year and every year with active finance budget items, and their active items:
// by year, the plan's default finance budget first, then by budget and item name. The item is pre-selected
// only when the default budget has exactly one active item (FR-015, R12).
func TestFormOptions_BudgetChoices_FR015(t *testing.T) {
	f := newTestService(t)

	options, err := f.service.FormOptions(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("FormOptions: %v", err)
	}
	if !slices.Equal(options.FinanceBudgetYears, []int{2025, 2026}) {
		t.Errorf("years = %v, want [2025 2026]", options.FinanceBudgetYears)
	}
	var order []string
	for _, item := range options.FinanceBudgetItems {
		order = append(order, item.ID)
	}
	if want := []string{lastYearItem, europeBrand, europeDigital, europeEvents, onlyItem}; !slices.Equal(order, want) {
		t.Errorf("items = %v, want %v (no inactive item)", order, want)
	}
	if d := options.Defaults; d.FinanceBudgetItemID != nil || d.FinanceBudgetItemPreselected {
		t.Errorf("Launch film: item %v pre-selected %v; want none: its default budget has three items", d.FinanceBudgetItemID, d.FinanceBudgetItemPreselected)
	}

	options, err = f.service.FormOptions(t.Context(), priya, retailEvent)
	if err != nil {
		t.Fatalf("FormOptions: %v", err)
	}
	if d := options.Defaults; d.FinanceBudgetItemID == nil || *d.FinanceBudgetItemID != onlyItem || !d.FinanceBudgetItemPreselected {
		t.Errorf("Retail event: item %v pre-selected %v; want %s, the only item of its default budget", d.FinanceBudgetItemID, d.FinanceBudgetItemPreselected, onlyItem)
	}
	if options.FinanceBudgetItems[1].ID != onlyItem {
		t.Errorf("Retail event: the plan's default budget is not listed first in 2026: %+v", options.FinanceBudgetItems)
	}
}

// The form offers the active currencies, USD first, then by code, each with its current rate (FR-011).
func TestFormOptions_Currencies_FR011(t *testing.T) {
	f := newTestService(t)

	options, err := f.service.FormOptions(t.Context(), priya, launchFilm)
	if err != nil {
		t.Fatalf("FormOptions: %v", err)
	}
	var codes []string
	for _, c := range options.Currencies {
		codes = append(codes, c.Code+" "+c.RatePerUSD.String())
	}
	if want := []string{"USD 1", "CNY 7.1", "EUR 0.92", "GBP 0.79"}; !slices.Equal(codes, want) {
		t.Errorf("currencies = %q, want %q (INR is inactive)", codes, want)
	}
}

// A deleted activity has no form: "This activity was deleted — …" (spec edge case).
func TestFormOptions_ActivityDeleted_EdgeCase(t *testing.T) {
	f := newTestService(t)

	_, err := f.service.FormOptions(t.Context(), priya, "A-9999")

	assertError(t, err, errs.NotFound(activityGoneMessage))
}

// Only name, currency and planned cost are needed to save (FR-009, D-009); what is needed to submit stays
// empty and is listed: "3 fields needed" (US2 scenario 6).
func TestCreate_OnlyRequired_FR009(t *testing.T) {
	f := newTestService(t)
	in := Input{Name: "Agency fee", Currency: "USD", PlannedCost: text("50000"), Spread: SpreadInput{Method: SpreadEvenly}}

	bare := f.create(t, priya, launchFilm, in)
	withOwner := input("Media", "USD", "1000.00")
	defaulted := f.create(t, priya, launchFilm, withOwner)

	if bare.Name != "Agency fee" || bare.PlannedCost != usd("50000.00") || bare.Owner != nil || bare.FinanceBudgetYear != nil {
		t.Errorf("created %+v", bare)
	}
	labels := func(v View) []string {
		var list []string
		for _, m := range v.Missing {
			list = append(list, m.Label)
		}
		return list
	}
	if want := []string{"Owner", "Spend type", "GL code", "Finance budget year", "Finance budget item"}; !slices.Equal(labels(bare), want) {
		t.Errorf("missing = %q, want %q", labels(bare), want)
	}
	if want := []string{"Spend type", "GL code", "Finance budget item"}; !slices.Equal(labels(defaulted), want) {
		t.Errorf("missing with the defaults = %q, want %q (3 fields needed)", labels(defaulted), want)
	}
	if got := f.create(t, priya, launchFilm, complete(input("Complete", "USD", "1.00"))).Missing; len(got) != 0 {
		t.Errorf("complete line item: missing %+v, want nothing (All filled in)", got)
	}
}

// Every problem of a form is reported at once, and nothing is saved (FR-009, SC-005, US2 scenario 5).
func TestCreate_AllProblemsAtOnce_FR009_SC005(t *testing.T) {
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

// Names are unique within the activity, ignoring letter case and surrounding spaces, and have at most 120
// characters; another activity may use the same name (FR-009).
func TestCreate_NameUnique_FR009(t *testing.T) {
	f := newTestService(t)
	f.create(t, priya, launchFilm, input("Agency fee", "USD", "1.00"))

	saved := f.create(t, priya, launchFilm, input("  "+strings.Repeat("é", 120)+" ", "USD", "1.00"))
	f.create(t, priya, socialOnAir, func() Input { in := input("Agency fee", "USD", "1.00"); in.RestartApproval = true; return in }())
	_, taken := f.service.Create(t.Context(), priya, launchFilm, input(" AGENCY fee", "USD", "1.00"))
	_, long := f.service.Create(t.Context(), priya, launchFilm, input(strings.Repeat("a", 121), "USD", "1.00"))

	if saved.Name != strings.Repeat("é", 120) {
		t.Errorf("saved name = %q, want it without the surrounding spaces", saved.Name)
	}
	assertError(t, taken, errs.Validation(errs.FieldError{Field: "name", Message: "'Launch film' already has a line item called 'Agency fee'."}))
	assertError(t, long, errs.Validation(errs.FieldError{Field: "name", Message: "Use 120 characters or fewer."}))
}

// Typed amounts have at most two decimals: the form rounds and says so, the API refuses more (D-190, spec
// edge case "Rounding"); amounts stay below one trillion.
func TestCreate_AmountsAtMostTwoDecimals_D190(t *testing.T) {
	f := newTestService(t)
	tests := []struct {
		cost, message string
	}{
		{"1234.565", "Enter an amount with at most two decimals."},
		{"1000000000000", "Enter an amount below one trillion."},
		{"9999999999999999", "Enter an amount below one trillion."},
		{"twelve", "Enter an amount, such as 12500.00."},
	}
	for _, tt := range tests {
		_, err := f.service.Create(t.Context(), priya, launchFilm, input("Media", "USD", tt.cost))

		assertError(t, err, errs.Validation(errs.FieldError{Field: "plannedCost", Message: tt.message}))
	}

	if v := f.create(t, priya, launchFilm, input("Biggest", "USD", "999999999999.99")); v.PlannedCost.Amount.String() != "999999999999.99" {
		t.Errorf("planned cost = %s", v.PlannedCost.Amount)
	}
	if v := f.create(t, priya, launchFilm, input("Round", "USD", " 1234.5 ")); v.PlannedCost.Amount.String() != "1234.50" {
		t.Errorf("planned cost = %s, want 1234.50", v.PlannedCost.Amount)
	}
}

// A new line item gets the next readable ID L-…, the person and time who created it, and one "created" log
// entry with its activity as its path (FR-010, D-019).
func TestCreate_IDAndLog_FR010(t *testing.T) {
	f := newTestService(t)

	first := f.create(t, priya, launchFilm, input("Agency fee", "USD", "50000.00"))
	second := f.create(t, daniel, launchFilm, input("Film production", "EUR", "46000.00"))

	if first.ID != "L-1001" || second.ID != "L-1002" || first.ActivityID != launchFilm {
		t.Errorf("IDs = %s, %s; want L-1001, L-1002", first.ID, second.ID)
	}
	if first.Meta != record.NewMeta(priya, start) || second.Meta != record.NewMeta(daniel, start) {
		t.Errorf("bookkeeping = %+v / %+v", first.Meta, second.Meta)
	}
	entries := f.log(t)
	if len(entries) != 2 {
		t.Fatalf("%d log entries, want 2", len(entries))
	}
	e := entries[1]
	want := []history.RecordRef{{Type: "activity", ID: launchFilm, Name: "Launch film"}}
	if e.Action != history.ActionCreated || e.RecordType != "lineItem" || e.RecordTypeLabel != "line item" ||
		e.RecordID != "L-1001" || e.RecordName != "Agency fee" || e.Actor != priya || !reflect.DeepEqual(e.Path, want) {
		t.Errorf("log entry = %+v", e)
	}
}

// A new currency choice must be an active currency of Settings (FR-011); the USD amount is calculated at
// its current rate: 10,000.00 EUR at 0.92 → 10,869.57 USD (US2 scenario 2).
func TestCreate_CurrencyActive_FR011(t *testing.T) {
	f := newTestService(t)

	_, inactive := f.service.Create(t.Context(), priya, launchFilm, input("Fees", "INR", "1.00"))
	_, unknown := f.service.Create(t.Context(), priya, launchFilm, input("Fees", "XYZ", "1.00"))
	_, empty := f.service.Create(t.Context(), priya, launchFilm, input("Fees", " ", "1.00"))
	euros := f.create(t, priya, launchFilm, input("Fees", "EUR", "10000"))

	assertError(t, inactive, errs.Validation(errs.FieldError{Field: "currency", Message: "Choose an active currency — INR is no longer active."}))
	assertError(t, unknown, errs.Validation(errs.FieldError{Field: "currency", Message: "Choose a currency."}))
	assertError(t, empty, errs.Validation(errs.FieldError{Field: "currency", Message: "Choose a currency."}))
	if euros.PlannedCost != (money.Money{Amount: amount("10000.00"), Currency: "EUR"}) || euros.PlannedCostUSD != usd("10869.57") ||
		euros.Currency.Code != "EUR" || euros.Currency.RatePerUSD.String() != "0.92" || !euros.Currency.Active {
		t.Errorf("EUR line item: %+v / %+v / %+v", euros.PlannedCost, euros.PlannedCostUSD, euros.Currency)
	}
}

// Shell attribute problems of feature 002 (R11: a GL code that doesn't fit the spend type) are reported with
// the others; the values 002 returns are stored (FR-014).
func TestCreate_AttributeProblems_FR014(t *testing.T) {
	f := newTestService(t)
	f.attributes.problems = []errs.FieldError{{Field: "attributes.SA-1033", Message: "Choose one of the options of spend category."}}
	in := input("", "USD", "1.00")
	in.Attributes = settings.AttributeValues{glCode: json.RawMessage(`"SO-1299"`)}

	_, err := f.service.Create(t.Context(), priya, launchFilm, in)

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "name", Message: "Enter a name for the line item."},
		errs.FieldError{Field: "attributes.SA-1033", Message: "Choose one of the options of spend category."},
		errs.FieldError{Field: "attributes." + glCode, Message: glCodeMismatch},
	))

	f.attributes.problems = nil
	in = complete(input("Coded", "USD", "1.00"))
	in.Attributes["SA-1033"] = json.RawMessage(`""`) // an empty value is not stored
	v := f.create(t, priya, launchFilm, in)
	if want := (settings.AttributeValues{spendType: json.RawMessage(`"SO-1101"`), glCode: json.RawMessage(`"SO-1201"`)}); !reflect.DeepEqual(v.Attributes, want) {
		t.Errorf("attributes = %s, want %s", v.Attributes, want)
	}
}

// The contracted amount is optional, 0.00 or more, in the line item's currency (FR-014).
func TestCreate_ContractedAmount_FR014(t *testing.T) {
	f := newTestService(t)
	for _, tt := range []struct{ amount, message string }{
		{"-1", "Contracted amount can't be negative."},
		{"12.345", "Enter an amount with at most two decimals."},
		{"lots", "Enter an amount, such as 12500.00."},
	} {
		in := input("Fees", "EUR", "100.00")
		in.ContractedAmount = &tt.amount

		_, err := f.service.Create(t.Context(), priya, launchFilm, in)

		assertError(t, err, errs.Validation(errs.FieldError{Field: "contractedAmount", Message: tt.message}))
	}

	in := input("Fees", "EUR", "100.00")
	in.ContractedAmount = text("80")
	if v := f.create(t, priya, launchFilm, in); v.ContractedAmount == nil || *v.ContractedAmount != (money.Money{Amount: amount("80.00"), Currency: "EUR"}) {
		t.Errorf("contracted amount = %+v, want 80.00 EUR", v.ContractedAmount)
	}
	in = input("Empty", "EUR", "100.00")
	in.ContractedAmount = text("")
	if v := f.create(t, priya, launchFilm, in); v.ContractedAmount != nil {
		t.Errorf("contracted amount = %+v, want none", v.ContractedAmount)
	}
}

// The finance budget year is the plan's year or a year with active items; the item needs a year first, must
// exist, belong to that year and be active when newly chosen (FR-015, FR-016).
func TestCreate_FinanceBudgetChecks_FR015_FR016(t *testing.T) {
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

// Adding a line item to a Submitted or Approved activity needs the person's confirmation; without it nothing
// is saved. With it, the activity goes back to Created with the reason (FR-031, D-146). A Created activity
// needs no confirmation and keeps its status.
func TestCreate_RestartNeedsConfirmation_FR031(t *testing.T) {
	f := newTestService(t)
	for _, activityID := range []string{socialOnAir, tradeAds} {
		_, err := f.service.Create(t.Context(), priya, activityID, input("Venue hire", "GBP", "15800"))

		assertError(t, err, errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage}))
	}
	if len(f.stored(t)) != 0 || len(f.activities.restarts) != 0 {
		t.Fatalf("saved %d line items and restarted %d times, want nothing", len(f.stored(t)), len(f.activities.restarts))
	}

	confirmed := input("Venue hire", "GBP", "15800")
	confirmed.RestartApproval = true
	v := f.create(t, priya, socialOnAir, confirmed)
	f.create(t, priya, launchFilm, input("Agency fee", "USD", "50000.00"))

	want := []restartCall{{activityID: socialOnAir, reasons: []string{"Line item 'Venue hire' (L-1001) added: 15,800.00 GBP"}}}
	if !reflect.DeepEqual(f.activities.restarts, want) {
		t.Errorf("restarts = %+v, want %+v", f.activities.restarts, want)
	}
	if v.SwbsStatus != SwbsWaitingForApproval {
		t.Errorf("SWBS status = %s, want waitingForApproval: the activity is Created again", v.SwbsStatus)
	}
}

// The section lists the activity's line items in creation order, each with its currency and rate, local and
// USD amounts, spread, missing fields and SWBS status; the unassigned planned cost; and the grid (FR-001 –
// FR-003, US1 scenario 1).
func TestSection_RowsInCreationOrder_FR001(t *testing.T) {
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

// An activity without line items has an empty list, its whole planned cost unassigned and its months as
// grid columns (FR-004).
func TestSection_Empty_FR004(t *testing.T) {
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
	f := newTestService(t)

	_, err := f.service.Section(t.Context(), priya, "A-9999")

	assertError(t, err, errs.NotFound(activityGoneMessage))
}

// USD amounts are worked out with the currency's current rate on every read: a rate change in Settings
// changes the USD amounts and the balance at once, writes nothing and restarts no approval (FR-013, D-141).
func TestSection_RateChange_FR013(t *testing.T) {
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

// confirmed returns in with the person's confirmation that the activity needs approval again.
func confirmed(in Input) Input {
	in.RestartApproval = true
	return in
}

// formOf returns the whole form of a stored line item, as the edit page sends it, with its version.
func formOf(v View) Update {
	months := []SpreadMonthInput{}
	if v.Spread.Method == SpreadCustom {
		for _, m := range v.Spread.Months {
			months = append(months, SpreadMonthInput{Month: m.Month, Amount: m.Amount.Amount.String()})
		}
	}
	u := Update{
		Name: v.Name, Description: v.Description, Currency: v.Currency.Code, PlannedCost: text(v.PlannedCost.Amount.String()),
		Spread: SpreadInput{Method: v.Spread.Method, Months: months}, Attributes: v.Attributes, RequestSwbs: v.RequestSwbs,
		FinanceBudgetYear: v.FinanceBudgetYear, Version: &v.Version,
	}
	if v.Owner != nil {
		u.OwnerID = text(v.Owner.ID)
	}
	if v.ContractedAmount != nil {
		u.ContractedAmount = text(v.ContractedAmount.Amount.String())
	}
	if v.FinanceBudgetItem != nil {
		u.FinanceBudgetItemID = text(v.FinanceBudgetItem.ID)
	}
	return u
}

// update saves u on the line item id and stops the test when that fails.
func (f *fixture) update(t *testing.T, a actor.Actor, id string, u Update) View {
	t.Helper()
	view, err := f.service.Update(t.Context(), a, id, u)
	if err != nil {
		t.Fatalf("Update(%s): %v", id, err)
	}
	return view
}

// put stores li as it is, for situations a form cannot create (values that became inactive since).
func (f *fixture) put(t *testing.T, li LineItem) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error { return NewJSONRepository().Put(ctx, li) })
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
}

// The details are the line item with its activity and the balance including it (FR-026). A line item that is
// gone says why (D-154): deleted itself, or deleted with its activity.
func TestGet_GoneMessages_007P15(t *testing.T) {
	f := newTestService(t)
	fee, _ := launchFilmWithTwoItems(t, f)
	gone := f.create(t, priya, launchFilm, input("Gone", "USD", "1.00"))
	if _, err := f.service.Delete(t.Context(), priya, gone.ID, false); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	details, err := f.service.Get(t.Context(), priya, fee.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if details.LineItem.ID != fee.ID || details.Activity.Name != "Launch film" || details.Balance.Balance.Format() != "20,000.00 USD" {
		t.Errorf("details = %+v", details)
	}
	_, deleted := f.service.Get(t.Context(), priya, gone.ID)
	_, unknown := f.service.Get(t.Context(), priya, "L-9999")
	assertError(t, deleted, errs.NotFound(lineItemGoneMessage))
	assertError(t, unknown, errs.NotFound(lineItemGoneMessage))

	err = f.db.Write(t.Context(), func(ctx context.Context) error {
		_, err := f.history.MoveToBin(ctx, priya, activityKind, launchFilm)
		return err
	})
	if err != nil {
		t.Fatalf("MoveToBin(activity): %v", err)
	}
	_, withActivity := f.service.Get(t.Context(), priya, fee.ID)
	_, saving := f.service.Update(t.Context(), priya, fee.ID, formOf(fee))
	assertError(t, withActivity, errs.NotFound(activityGoneMessage))
	assertError(t, saving, errs.NotFound(activityGoneMessage))
}

// A save needs the version it started from: a missing one is reported with every other problem, an
// out-of-date one is refused naming who changed the line item (FR-027, D-030).
func TestUpdate_Version_FR027(t *testing.T) {
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, input("Agency fee", "USD", "50000.00"))
	zero := 0
	reload := errs.FieldError{Field: "version", Message: reloadMessage}

	noVersion := formOf(fee)
	noVersion.Version, noVersion.Name = nil, ""
	_, missing := f.service.Update(t.Context(), priya, fee.ID, noVersion)
	zeroVersion := formOf(fee)
	zeroVersion.Version = &zero
	_, zeroed := f.service.Update(t.Context(), priya, fee.ID, zeroVersion)
	newer := formOf(fee)
	newer.Description = "Daniel's change"
	f.update(t, daniel, fee.ID, newer)
	stale := formOf(fee)
	stale.Description = "Priya's change"
	_, conflict := f.service.Update(t.Context(), priya, fee.ID, stale)

	assertError(t, missing, errs.Validation(errs.FieldError{Field: "name", Message: enterNameMessage}, reload))
	assertError(t, zeroed, errs.Validation(reload))
	assertError(t, conflict, errs.VersionConflict("line item", "Daniel Okoro", start))
}

// A save that changes nothing creates no version and no log entry (FR-027).
func TestUpdate_NoChangeNoLog_FR027(t *testing.T) {
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, complete(input("Agency fee", "USD", "50000.00")))
	logged := len(f.log(t))
	same := formOf(fee)
	same.Name, same.PlannedCost = " Agency fee ", text("50000")

	saved := f.update(t, daniel, fee.ID, same)

	if saved.Version != 1 || saved.UpdatedBy != priya || len(f.log(t)) != logged {
		t.Errorf("version %d by %s, %d log entries; want version 1 by Priya and no new entry", saved.Version, saved.UpdatedBy.Name, len(f.log(t))-logged)
	}
}

// A save with changes gets a new version and one "changed" entry listing every changed field with its old
// and new value as people read them; each Shell attribute is a change of its own (FR-027, data model §7.3).
func TestUpdate_LogsOldAndNew_FR027(t *testing.T) {
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, complete(input("Agency fee", "USD", "50000.00")))
	u := formOf(fee)
	u.Name, u.OwnerID, u.PlannedCost = "Agency fees", text(daniel.ID), text("55000")
	u.Spread = SpreadInput{Method: SpreadCustom, Months: []SpreadMonthInput{{Month: "2026-02", Amount: "2000"}, {Month: "2026-03", Amount: "53000"}}}
	u.Attributes = settings.AttributeValues{spendType: json.RawMessage(`"SO-1101"`), glCode: json.RawMessage(`"SO-1202"`)}
	u.FinanceBudgetItemID = text(europeEvents)

	saved := f.update(t, daniel, fee.ID, u)

	if saved.Version != 2 || saved.UpdatedBy != daniel || saved.UpdatedAt != start || saved.CreatedBy != priya {
		t.Errorf("bookkeeping = %+v", saved.Meta)
	}
	entry := f.log(t)[0]
	var got []string
	for _, c := range entry.Changes {
		got = append(got, c.Label+": "+c.OldText+" → "+c.NewText)
	}
	want := []string{
		"Finance budget item: 2026 Lubricants Budget › Europe – Digital → 2026 Lubricants Budget › Europe – Events",
		"Name: Agency fee → Agency fees",
		"Owner: Priya Raman → Daniel Okoro",
		"Planned cost: 50,000.00 → 55,000.00",
		"Spread over time: Evenly → Custom: Feb 2026 2,000.00 · Mar 2026 53,000.00",
		"GL code: 7250080 – Advertising digital → 7250040 – Events and sponsorship",
	}
	if entry.Action != history.ActionChanged || entry.RecordName != "Agency fees" || entry.Actor != daniel || !slices.Equal(got, want) {
		t.Errorf("entry %s %q by %s:\n%s\nwant\n%s", entry.Action, entry.RecordName, entry.Actor.Name, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// A currency, owner or finance budget item that became inactive stays on the line item through any save and
// still converts at its last rate; only a new choice must be active (FR-011, FR-016).
func TestUpdate_KeepsInactiveValues_FR011_FR016(t *testing.T) {
	f := newTestService(t)
	year := 2026
	f.put(t, LineItem{
		ID: "L-1001", ActivityID: launchFilm, Name: "Delhi fees", OwnerID: "U-1009", Currency: "INR",
		PlannedCost: amount("835.00"), Spread: Spread{Method: SpreadEvenly, Months: []MonthAmount{}},
		Attributes: settings.AttributeValues{}, FinanceBudgetYear: &year, FinanceBudgetItemID: europeLegacy,
		Meta: record.NewMeta(priya, start),
	})
	details, err := f.service.Get(t.Context(), priya, "L-1001")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	kept := details.LineItem
	if kept.Currency.Active || kept.PlannedCostUSD.Format() != "10.00 USD" || kept.Owner == nil || kept.Owner.Active ||
		kept.FinanceBudgetItem == nil || kept.FinanceBudgetItem.Active {
		t.Errorf("kept values = %+v / %s / %+v / %+v", kept.Currency, kept.PlannedCostUSD.Format(), kept.Owner, kept.FinanceBudgetItem)
	}

	u := formOf(kept)
	u.Description = "Still in rupees"
	if saved := f.update(t, priya, "L-1001", u); saved.Version != 2 || saved.Currency.Code != "INR" {
		t.Errorf("saved = %+v, want version 2 in INR", saved)
	}

	other := f.create(t, priya, launchFilm, input("Agency fee", "EUR", "10.00"))
	switched := formOf(other)
	switched.Currency, switched.OwnerID, switched.FinanceBudgetItemID = "INR", text("U-1009"), text(europeLegacy)
	_, err = f.service.Update(t.Context(), priya, other.ID, switched)
	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "ownerId", Message: "Choose an active person — Sam Former is inactive."},
		errs.FieldError{Field: "currency", Message: "Choose an active currency — INR is no longer active."},
		errs.FieldError{Field: "financeBudgetItemId", Message: "Choose an active finance budget item — 'Europe – Legacy' is no longer active."},
	))
}

// Changing the planned cost or the currency of a line item of a Submitted or Approved activity needs the
// person's confirmation and sends the activity back to Created with the change (FR-031, US4 scenario 3).
func TestUpdate_CostOrCurrencyRestart_FR031(t *testing.T) {
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, confirmed(input("Venue hire", "GBP", "15800")))
	approve := func() {
		facts := f.activities.facts[socialOnAir]
		facts.Status = "approved"
		f.activities.facts[socialOnAir] = facts
	}
	approve()
	f.activities.restarts = nil

	more := formOf(venue)
	more.PlannedCost = text("17000")
	_, err := f.service.Update(t.Context(), priya, venue.ID, more)
	assertError(t, err, errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage}))
	if stored := f.stored(t)[0]; stored.Version != 1 || len(f.activities.restarts) != 0 {
		t.Fatalf("unconfirmed save stored version %d and restarted %d times", stored.Version, len(f.activities.restarts))
	}

	more.RestartApproval = true
	saved := f.update(t, priya, venue.ID, more)
	approve()
	euros := formOf(saved)
	euros.Currency, euros.RestartApproval = "EUR", true
	f.update(t, priya, venue.ID, euros)

	want := []restartCall{
		{activityID: socialOnAir, reasons: []string{"Planned cost of line item 'Venue hire' (L-1001): 15,800.00 GBP → 17,000.00 GBP"}},
		{activityID: socialOnAir, reasons: []string{"Planned cost of line item 'Venue hire' (L-1001): 17,000.00 GBP → 17,000.00 EUR"}},
	}
	if !reflect.DeepEqual(f.activities.restarts, want) {
		t.Errorf("restarts = %+v\nwant %+v", f.activities.restarts, want)
	}
}

// Every other change — name, owner, description, spread, Shell attributes, contracted amount, SWBS request,
// finance budget link — saves without a confirmation and never changes the activity's status (FR-032,
// SC-007, US4 scenario 2).
func TestUpdate_OtherEditsKeepStatus_FR032_SC007(t *testing.T) {
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, confirmed(input("Venue hire", "GBP", "15800")))
	facts := f.activities.facts[socialOnAir]
	facts.Status = "approved"
	f.activities.facts[socialOnAir] = facts
	f.activities.restarts = nil

	u := formOf(venue)
	u.Name, u.OwnerID, u.Description, u.ContractedAmount = "Venue", text(daniel.ID), "Hall B", text("15000")
	u.Spread = SpreadInput{Method: SpreadLastMonth, Months: []SpreadMonthInput{}}
	u.Attributes = settings.AttributeValues{glCode: json.RawMessage(`"SO-1202"`)}
	u.RequestSwbs, u.FinanceBudgetItemID = false, text(europeBrand)
	saved := f.update(t, priya, venue.ID, u)

	if saved.Version != 2 || len(f.activities.restarts) != 0 || f.activities.facts[socialOnAir].Status != "approved" ||
		saved.SwbsStatus != SwbsNotRequested {
		t.Errorf("version %d, %d restarts, activity %s, SWBS %s; want 2, none, approved, notRequested",
			saved.Version, len(f.activities.restarts), f.activities.facts[socialOnAir].Status, saved.SwbsStatus)
	}
}

// Duplicate copies every value, the spread included, into the same activity with a new ID and the name
// "<name> (copy)", " (copy 2)", … shortened to fit 120 characters, and logs "copied" (FR-029).
func TestDuplicate_CopiesEverything_FR029(t *testing.T) {
	f := newTestService(t)
	in := complete(input("Agency fee", "EUR", "1000.00"))
	in.Description, in.ContractedAmount, in.RequestSwbs = "Retainer", text("900"), false
	in.Spread = SpreadInput{Method: SpreadCustom, Months: []SpreadMonthInput{{Month: "2026-02", Amount: "400"}, {Month: "2026-04", Amount: "600"}}}
	fee := f.create(t, priya, launchFilm, in)
	long := f.create(t, priya, launchFilm, input(strings.Repeat("n", 120), "USD", "1.00"))

	first, err := f.service.Duplicate(t.Context(), daniel, fee.ID, DuplicateInput{})
	if err != nil {
		t.Fatalf("Duplicate: %v", err)
	}
	second, _ := f.service.Duplicate(t.Context(), daniel, fee.ID, DuplicateInput{})
	longCopy, _ := f.service.Duplicate(t.Context(), daniel, long.ID, DuplicateInput{})

	if first.ID != "L-1003" || first.Name != "Agency fee (copy)" || second.Name != "Agency fee (copy 2)" ||
		longCopy.Name != strings.Repeat("n", 113)+" (copy)" {
		t.Errorf("copies = %s %q, %q, %q", first.ID, first.Name, second.Name, longCopy.Name)
	}
	copied, original := first, fee
	copied.ID, copied.Name, copied.Meta = original.ID, original.Name, original.Meta
	if !reflect.DeepEqual(copied, original) {
		t.Errorf("copy = %+v\nwant the values of %+v", copied, original)
	}
	if first.Meta != record.NewMeta(daniel, start) {
		t.Errorf("copy bookkeeping = %+v, want new, by Daniel", first.Meta)
	}
	entry := f.log(t)[2]
	if entry.Action != history.ActionCopied || entry.RecordID != first.ID || entry.Details != "Copied from 'Agency fee' (L-1001)" {
		t.Errorf("log entry = %+v", entry)
	}
}

// Duplicating counts as adding a line item: on a Submitted or Approved activity only with confirmation,
// which sends it back to Created (FR-031, US4 scenario 3).
func TestDuplicate_RestartNeedsConfirmation_FR031(t *testing.T) {
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, confirmed(input("Venue hire", "GBP", "15800")))
	facts := f.activities.facts[socialOnAir]
	facts.Status = "approved"
	f.activities.facts[socialOnAir] = facts
	f.activities.restarts = nil

	_, refused := f.service.Duplicate(t.Context(), priya, venue.ID, DuplicateInput{})
	dup, err := f.service.Duplicate(t.Context(), priya, venue.ID, DuplicateInput{RestartApproval: true})
	_, gone := f.service.Duplicate(t.Context(), priya, "L-9999", DuplicateInput{RestartApproval: true})

	assertError(t, refused, errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage}))
	assertError(t, gone, errs.NotFound(lineItemGoneMessage))
	if err != nil || dup.ID != "L-1002" {
		t.Fatalf("Duplicate = %+v, %v", dup, err)
	}
	want := []restartCall{{activityID: socialOnAir, reasons: []string{"Line item 'Venue hire (copy)' (L-1002) added: 15,800.00 GBP"}}}
	if !reflect.DeepEqual(f.activities.restarts, want) {
		t.Errorf("restarts = %+v, want %+v", f.activities.restarts, want)
	}
}

// Delete moves the line item to the recycle bin, with its activity as its path, and logs "deleted"; the bin
// entry is the answer, so the web app can offer Undo (FR-030).
func TestDelete_MovesToBin_FR030(t *testing.T) {
	f := newTestService(t)
	fee, film := launchFilmWithTwoItems(t, f)

	entry, err := f.service.Delete(t.Context(), priya, fee.ID, false)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, gone := f.service.Delete(t.Context(), priya, fee.ID, false)

	want := []history.RecordRef{{Type: "activity", ID: launchFilm, Name: "Launch film"}}
	if entry.RecordType != "lineItem" || entry.RecordID != fee.ID || entry.RecordName != "Agency fee" ||
		!reflect.DeepEqual(entry.Path, want) || len(entry.Snapshots) != 1 {
		t.Errorf("bin entry = %+v", entry)
	}
	if stored := f.stored(t); len(stored) != 1 || stored[0].ID != film.ID {
		t.Errorf("stored = %+v, want only Film production", stored)
	}
	if e := f.log(t)[0]; e.Action != history.ActionDeleted || e.RecordID != fee.ID || !reflect.DeepEqual(e.Path, want) {
		t.Errorf("log entry = %+v", e)
	}
	assertError(t, gone, errs.NotFound(lineItemGoneMessage))
	if len(f.activities.restarts) != 0 {
		t.Errorf("a delete on a Created activity restarted its approval")
	}
}

// Deleting a line item of a Submitted or Approved activity needs confirmation and sends the activity back to
// Created (FR-031).
func TestDelete_RestartNeedsConfirmation_FR031(t *testing.T) {
	f := newTestService(t)
	venue := f.create(t, priya, tradeAds, confirmed(input("Venue hire", "GBP", "15800")))
	facts := f.activities.facts[tradeAds]
	facts.Status = "submitted"
	f.activities.facts[tradeAds] = facts
	f.activities.restarts = nil

	_, refused := f.service.Delete(t.Context(), priya, venue.ID, false)
	assertError(t, refused, errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage}))
	if len(f.stored(t)) != 1 {
		t.Fatal("an unconfirmed delete removed the line item")
	}

	if _, err := f.service.Delete(t.Context(), priya, venue.ID, true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	want := []restartCall{{activityID: tradeAds, reasons: []string{"Line item 'Venue hire' (L-1001) deleted: 15,800.00 GBP"}}}
	if !reflect.DeepEqual(f.activities.restarts, want) || len(f.stored(t)) != 0 {
		t.Errorf("restarts = %+v, want %+v", f.activities.restarts, want)
	}
}

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

// A custom spread that does not add up still saves; its total shows how far off it is, and the line item's
// missing list (which blocks submitting the activity) says "Spread doesn't add up to the planned cost"
// (FR-019, US3 scenarios 2 and 3). The activity's grid shows the month outside its dates.
func TestCreate_CustomNotAddingUpSaves_FR019(t *testing.T) {
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

// Switching a line item away from Custom drops its typed months: the new method spreads the planned cost
// over the activity's months, and the log shows the change (FR-018, US3 scenario 6).
func TestUpdate_LeavingCustomDropsMonths_FR018(t *testing.T) {
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
