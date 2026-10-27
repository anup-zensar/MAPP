package lineitems

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"mapp/internal/people"
	"mapp/internal/platform/errs"
)

// A new line item starts filled in (FR-008): the name "<activity name> – line item", the activity's owner,
// USD, the unassigned planned cost, the plan's default spread and year, no finance budget item when the
// plan's default budget has several, and Request SWBS code from GSAP on; with the activity and its balance.
func TestFormOptions_Defaults_FR008(t *testing.T) { //nolint:gocyclo // 11: the defaults, the balance and the activity of one form, field by field
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
func TestFormOptions_BudgetChoices_FR015(t *testing.T) { //nolint:gocyclo // 12: the years, the items and the pre-selected item of two forms
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	f := newTestService(t)

	_, err := f.service.FormOptions(t.Context(), priya, "A-9999")

	assertError(t, err, errs.NotFound(activityGoneMessage))
}
