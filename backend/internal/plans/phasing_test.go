package plans

import (
	"reflect"
	"testing"
	"time"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// phasingOf is a phasing save of method with custom months as pairs of month and amount.
func phasingOf(method string, version int, months ...string) rollup.PhasingInput {
	in := rollup.PhasingInput{Method: method, Version: &version, Months: []rollup.PhasingMonthInput{}}
	for i := 0; i+1 < len(months); i += 2 {
		in.Months = append(in.Months, rollup.PhasingMonthInput{Month: months[i], Amount: months[i+1]})
	}
	return in
}

// twelve returns the 12 months of 2027 with amount each, as pairs, Mar set to march.
func twelve(amount, march string) []string {
	var pairs []string
	for _, key := range rollup.MonthKeys(2027) {
		value := amount
		if key == "2027-03" {
			value = march
		}
		pairs = append(pairs, key, value)
	}
	return pairs
}

// Budget by month is saved with its own operation: Evenly splits the budget over the 12 months; a Custom
// change is logged month by month in the plan currency ("Budget by month: Mar 2027 50,000.00 → 60,000.00
// GBP"), with who changed it (FR-020, FR-023, US3-6).
func TestSetPlanPhasing_SavesAndLogs_FR020_FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, gbpInput("FY2027 Lubricants UK & Ireland"))

	evenly, err := f.service.SetBudgetPhasing(t.Context(), priya, plan.ID, phasingOf("evenly", 1))
	if err != nil {
		t.Fatalf("evenly: %v", err)
	}
	f.clock.Advance(time.Hour)
	custom, err := f.service.SetBudgetPhasing(t.Context(), daniel, plan.ID, phasingOf("custom", 2, twelve("50000", "60000.00")...))
	if err != nil {
		t.Fatalf("custom: %v", err)
	}

	if len(evenly.BudgetPhasing.Monthly) != 12 || evenly.BudgetPhasing.Monthly[0].Format() != "50,000.00 GBP" ||
		evenly.BudgetPhasing.Total.Format() != "600,000.00 GBP" || evenly.Version != 2 {
		t.Errorf("evenly = %+v, version %d", evenly.BudgetPhasing, evenly.Version)
	}
	if custom.BudgetPhasing.Method != rollup.MethodCustom || custom.BudgetPhasing.Monthly[2].Format() != "60,000.00 GBP" ||
		custom.BudgetPhasing.Problem == nil || *custom.BudgetPhasing.Problem != "Adds up to 610,000.00 GBP, not the budget 600,000.00 GBP" {
		t.Errorf("custom = %+v", custom.BudgetPhasing)
	}
	entry := f.entries(t)[0]
	type change struct{ label, old, new string }
	var got []change
	for _, c := range entry.Changes {
		got = append(got, change{c.Label, c.OldText, c.NewText})
	}
	want := []change{{"Budget by month", "Evenly", "Custom"}, {"Budget by month: Mar 2027", "50,000.00 GBP", "60,000.00 GBP"}}
	if entry.Actor != daniel || entry.RecordID != plan.ID || !reflect.DeepEqual(got, want) {
		t.Errorf("log entry by %v: %+v\nwant %+v", entry.Actor, got, want)
	}
	if _, err := f.service.SetBudgetPhasing(t.Context(), daniel, plan.ID, phasingOf("custom", 3, twelve("50000", "60000.00")...)); err != nil ||
		len(f.entries(t)) != 3 {
		t.Errorf("an unchanged save: %v, %d entries", err, len(f.entries(t)))
	}
}

// Phasing never changes a plan's status: a Published plan stays Published (FR-023).
func TestSetPlanPhasing_PublishedStaysPublished_FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, gbpInput("FY2027 Published"))
	published, err := f.service.Publish(t.Context(), priya, plan.ID)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	view, err := f.service.SetBudgetPhasing(t.Context(), priya, plan.ID, phasingOf("evenly", published.Version))

	if err != nil || view.Status != StatusPublished || view.BudgetPhasing.Method != rollup.MethodEvenly {
		t.Errorf("after phasing: %s %s, %v", view.Status, view.BudgetPhasing.Method, err)
	}
}

// A save from an out-of-date version is refused naming who changed the plan; the rules of a save come at
// once (FR-020, data-model §3.3).
func TestSetPlanPhasing_VersionConflict(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, gbpInput("FY2027 Conflict"))
	noBudget := f.create(t, priya, input("FY2026 No budget"))
	f.clock.Advance(time.Hour)
	if _, err := f.service.SetBudgetPhasing(t.Context(), daniel, plan.ID, phasingOf("evenly", 1)); err != nil {
		t.Fatalf("SetBudgetPhasing: %v", err)
	}

	_, stale := f.service.SetBudgetPhasing(t.Context(), priya, plan.ID, phasingOf("none", 1))
	_, invalid := f.service.SetBudgetPhasing(t.Context(), priya, plan.ID, phasingOf("custom", 2, "2026-03", "1", "2027-04", "-1"))
	_, unbudgeted := f.service.SetBudgetPhasing(t.Context(), priya, noBudget.ID, phasingOf("evenly", 1))
	_, gone := f.service.SetBudgetPhasing(t.Context(), priya, "P-9999", phasingOf("evenly", 1))

	assertError(t, stale, errs.VersionConflict("plan", "Daniel Okoro", start.Add(time.Hour)))
	assertError(t, invalid, problems("months.2027-04", "An amount can't be negative.", "months", "Months must be in 2027."))
	assertError(t, unbudgeted, problems("method", "Set a budget before phasing it."))
	assertError(t, gone, errs.NotFound(planGoneMessage))
}

// Removing the plan budget removes its phasing in the same save, and the log says so (FR-022, data-model §3.3).
func TestPlanBudgetRemoved_ResetsPhasing_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, gbpInput("FY2027 Removed"))
	if _, err := f.service.SetBudgetPhasing(t.Context(), priya, plan.ID, phasingOf("evenly", 1)); err != nil {
		t.Fatalf("SetBudgetPhasing: %v", err)
	}
	edit := updateOf(f.stored(t, plan.ID))
	edit.Budget = nil

	view, err := f.service.Update(t.Context(), priya, plan.ID, edit)

	if err != nil || view.Budget != nil || view.BudgetPhasing.Method != rollup.MethodNone || len(view.BudgetPhasing.Monthly) != 0 {
		t.Fatalf("after removing the budget: %v %+v, %v", view.Budget, view.BudgetPhasing, err)
	}
	changes := f.entries(t)[0].Changes
	if changes[0].Field != "budget" || changes[0].OldText != "600,000.00 GBP" || changes[1].Label != "Budget by month" ||
		changes[1].OldText != "Evenly" || changes[1].NewText != "Not phased" || len(changes) != 14 {
		t.Errorf("changes = %+v", changes)
	}
}

// A Custom phasing that does not add up blocks publishing the plan, listed with the plan's own fields
// (FR-022, US3-3).
func TestPublish_PhasingMismatchBlocks_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, gbpInput("FY2027 Mismatch"))
	if _, err := f.service.SetBudgetPhasing(t.Context(), priya, plan.ID, phasingOf("custom", 1, "2027-01", "590000")); err != nil {
		t.Fatalf("SetBudgetPhasing: %v", err)
	}

	_, err := f.service.Publish(t.Context(), priya, plan.ID)

	assertError(t, err, errs.NotReady("Publish needs 1 field", []errs.ReadinessGroup{
		{Record: errs.RecordRef{Type: "plan", ID: plan.ID, Name: "FY2027 Mismatch"}, Fields: []settings.MissingField{
			{Field: "budgetPhasing", Label: "Budget by month", Reason: "Adds up to 590,000.00 GBP, not the budget 600,000.00 GBP"}}},
	}))
}

// The phasing problem is part of the plan's completeness; a changed budget keeps Custom months and flags them
// (FR-022).
func TestPlanCompleteness_PhasingProblem_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, gbpInput("FY2027 Complete"))
	if _, err := f.service.SetBudgetPhasing(t.Context(), priya, plan.ID, phasingOf("custom", 1, "2027-06", "600000")); err != nil {
		t.Fatalf("SetBudgetPhasing: %v", err)
	}
	fine, _ := f.service.Get(t.Context(), priya, plan.ID)
	edit := updateOf(f.stored(t, plan.ID))
	edit.Budget = &money.Input{Amount: "612000", Currency: "GBP"}

	changed, err := f.service.Update(t.Context(), priya, plan.ID, edit)

	if len(fine.Completeness.Missing) != 0 {
		t.Errorf("before: missing %+v", fine.Completeness.Missing)
	}
	want := []settings.MissingField{{Field: "budgetPhasing", Label: "Budget by month", Reason: "Adds up to 600,000.00 GBP, not the budget 612,000.00 GBP"}}
	if err != nil || !reflect.DeepEqual(changed.Completeness.Missing, want) || changed.BudgetPhasing.Method != rollup.MethodCustom {
		t.Errorf("after the budget change: %+v, %v", changed.Completeness.Missing, err)
	}
}
