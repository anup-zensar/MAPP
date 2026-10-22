package plans

import (
	"reflect"
	"testing"

	"mapp/internal/people"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
)

// gbpInput is the plan form of P-1004 of the 016 demo data: a 2027 Cross-market plan planned in GBP with a
// budget of 600,000.00 GBP.
func gbpInput(name string) PlanInput {
	in := input(name)
	in.Year, in.Currency, in.PlanType = ptr(2027), ptr("GBP"), ptr(string(TypeCrossMarket))
	in.Budget = &money.Input{Amount: "600000", Currency: "GBP"}
	return in
}

// A plan has a plan currency — an active currency, USD when none is sent — and a plan type, Operational when
// none is sent; the budget is typed in the plan currency; every problem comes at once (FR-011, FR-032).
func TestCreatePlan_CurrencyAndType_FR011_FR032(t *testing.T) {
	f := newFixture(t)

	plain := f.create(t, priya, input("FY2026 Plain"))
	gbp := f.create(t, priya, gbpInput("FY2027 Lubricants UK & Ireland"))
	wrong := input("Wrong")
	wrong.Currency, wrong.PlanType = ptr("XXX"), ptr("tactical")
	_, unknown := f.service.Create(t.Context(), priya, wrong)
	inactive := gbpInput("Inactive")
	inactive.Currency = ptr("INR")
	_, inactiveErr := f.service.Create(t.Context(), priya, inactive)
	usdBudget := gbpInput("USD budget")
	usdBudget.Budget = &money.Input{Amount: "759493.67", Currency: "USD"}
	_, budgetErr := f.service.Create(t.Context(), priya, usdBudget)

	if plain.Currency != "USD" || plain.PlanType != TypeOperational || plain.BudgetPhasing.Method != rollup.MethodNone {
		t.Errorf("plain plan = %s %s %s", plain.Currency, plain.PlanType, plain.BudgetPhasing.Method)
	}
	if gbp.Currency != "GBP" || gbp.PlanType != TypeCrossMarket || gbp.Budget == nil || gbp.Budget.Format() != "600,000.00 GBP" {
		t.Errorf("GBP plan = %s %s %v", gbp.Currency, gbp.PlanType, gbp.Budget)
	}
	if stored := f.stored(t, gbp.ID); stored.Currency != "GBP" || stored.PlanType != TypeCrossMarket || stored.Budget.String() != "600000.00" {
		t.Errorf("stored = %+v", stored)
	}
	assertError(t, unknown, problems("planType", "Choose Operational, Strategic or Cross-market.", "currency", "Choose a currency from the list."))
	assertError(t, inactiveErr, problems("currency", "'INR' is inactive — choose an active currency.", "budget",
		"Enter the amount in USD, the plan's currency."))
	assertError(t, budgetErr, problems("budget", "Enter the amount in GBP, the plan's currency."))
}

// The plan currency is locked once campaigns, initiatives or activities have amounts in it, saying how many;
// with only a plan budget it may change and the budget keeps its number (FR-012, US2-5).
func TestUpdatePlan_CurrencyLock_FR012(t *testing.T) {
	f := newFixture(t)
	locked := f.create(t, priya, gbpInput("Locked plan"))
	free := f.create(t, priya, gbpInput("Free plan"))
	f.addLevel(t, "campaign", map[string]Facts{locked.ID: {Campaigns: 1, AmountsSet: 4}})

	edit := updateOf(f.stored(t, locked.ID))
	edit.Currency = ptr("EUR")
	edit.Budget = &money.Input{Amount: "600000.00", Currency: "EUR"}
	_, err := f.service.Update(t.Context(), priya, locked.ID, edit)
	view, getErr := f.service.Get(t.Context(), priya, locked.ID)
	freeEdit := updateOf(f.stored(t, free.ID))
	freeEdit.Currency = ptr("EUR")
	freeEdit.Budget = &money.Input{Amount: "600000.00", Currency: "EUR"}
	changed, freeErr := f.service.Update(t.Context(), priya, free.ID, freeEdit)

	lock := "The plan currency can't change once campaigns, initiatives or activities have amounts in it (4 amounts)."
	assertError(t, err, problems("currency", lock))
	if getErr != nil || view.CurrencyLock == nil || *view.CurrencyLock != lock {
		t.Errorf("currencyLock = %v, %v", view.CurrencyLock, getErr)
	}
	if freeErr != nil || changed.Currency != "EUR" || changed.Budget.Format() != "600,000.00 EUR" || changed.CurrencyLock != nil {
		t.Errorf("free plan = %s %v %v, %v", changed.Currency, changed.Budget, changed.CurrencyLock, freeErr)
	}
	entries := f.entries(t)
	last := entries[0]
	var fields []string
	for _, c := range last.Changes {
		fields = append(fields, c.Field+": "+c.OldText+" → "+c.NewText)
	}
	want := []string{"budget: 600,000.00 GBP → 600,000.00 EUR", "currency: GBP → EUR"}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("log changes = %q, want %q", fields, want)
	}
	if one := currencyLock(Facts{AmountsSet: 1}); one == nil || *one != "The plan currency can't change once campaigns, initiatives or activities have amounts in it (1 amount)." {
		t.Errorf("one amount = %v", one)
	}
}

// Inside a plan, amounts are in the plan currency (planFigures) and in the reader's display currency
// (figures), each converted once from the plan currency (FR-013, data-model §3.1): 600,000.00 GBP is
// 759,493.67 USD at 0.79.
func TestPlanFigures_PlanCurrencyAndDisplay_FR013(t *testing.T) {
	f := newFixture(t)
	plan := f.create(t, priya, gbpInput("FY2027 Lubricants UK & Ireland"))
	planned, budgets := money.FromCents(6_500_000), money.FromCents(40_000_000)
	f.addLevel(t, "activity", map[string]Facts{plan.ID: {Activities: 2, PlannedCost: &planned, CampaignBudgets: budgets, AmountsSet: 2}})
	f.directory.rates[daniel.ID] = people.DisplayRate{Code: "EUR", Rate: "0.92"}

	view, err := f.service.Get(t.Context(), priya, plan.ID)
	inEuro, euroErr := f.service.Get(t.Context(), daniel, plan.ID)

	if err != nil || view.Budget.Format() != "600,000.00 GBP" || view.PlanFigures.Currency != "GBP" ||
		view.PlanFigures.Budget.Format() != "600,000.00 GBP" || view.PlanFigures.PlannedCost.Format() != "65,000.00 GBP" ||
		view.PlanFigures.RemainingBudget.Format() != "200,000.00 GBP" {
		t.Errorf("plan currency: budget %v, planFigures %+v, %v", view.Budget, view.PlanFigures, err)
	}
	if view.Figures.Currency != "USD" || view.Figures.Budget.Format() != "759,493.67 USD" ||
		view.Figures.PlannedCost.Format() != "82,278.48 USD" || view.Figures.CampaignBudgets.Format() != "506,329.11 USD" {
		t.Errorf("display figures = %+v", view.Figures)
	}
	if euroErr != nil || inEuro.Figures.Budget.Format() != "698,734.18 EUR" {
		t.Errorf("EUR reader: %v, %v", inEuro.Figures.Budget, euroErr)
	}
	if view.Figures.BudgetUsedPercent == nil || *view.Figures.BudgetUsedPercent != 11 {
		t.Errorf("budget used = %v, want 11 %%", view.Figures.BudgetUsedPercent)
	}
}

// The Plans list shows each plan's amounts in its plan currency with the display currency beside; the
// summary converts each plan's totals to USD before adding them (FR-014).
func TestPlanList_PlanCurrencyAmounts_FR014(t *testing.T) {
	f := newFixture(t)
	usdPlan := seedPlan(t, "P-1001", "FY2027 USD plan", 2027, teamLubricants, priya.ID, "1000000.00", StatusDraft)
	gbpPlan := seedPlan(t, "P-1002", "FY2027 GBP plan", 2027, teamLubricants, priya.ID, "600000.00", StatusDraft)
	gbpPlan.Currency, gbpPlan.PlanType = "GBP", TypeCrossMarket
	f.put(t, usdPlan, gbpPlan)
	planned := money.FromCents(1_000_000)
	f.addLevel(t, "activity", map[string]Facts{"P-1002": {PlannedCost: &planned}})

	list, err := f.service.List(t.Context(), priya, ListFilter{})

	if err != nil || len(list.Items) != 2 {
		t.Fatalf("List = %+v, %v", list, err)
	}
	gbp := list.Items[0]
	if gbp.ID != "P-1002" || gbp.Currency != "GBP" || gbp.PlanType != TypeCrossMarket || gbp.PlanBudget.Format() != "600,000.00 GBP" ||
		gbp.PlanPlannedCost.Format() != "10,000.00 GBP" || gbp.Figures.Budget.Format() != "759,493.67 USD" ||
		gbp.Figures.PlannedCost.Format() != "12,658.23 USD" {
		t.Errorf("GBP item = %+v", gbp)
	}
	if usd := list.Items[1]; usd.Currency != "USD" || usd.PlanType != TypeOperational || usd.PlanPlannedCost != nil {
		t.Errorf("USD item = %+v", usd)
	}
	if list.Summary.Budget.Format() != "1,759,493.67 USD" || list.Summary.PlannedCost.Format() != "12,658.23 USD" {
		t.Errorf("summary = %+v", list.Summary)
	}
}

// A copy keeps the plan currency, the plan type and the phasing; a next-year copy moves the phasing months
// one year later (FR-018, FR-025).
func TestCopyPlan_KeepsCurrencyTypePhasing_FR018_FR025(t *testing.T) {
	f := newFixture(t)
	in := gbpInput("FY2027 Phased")
	plan := f.create(t, priya, in)
	if _, err := f.service.SetBudgetPhasing(t.Context(), priya, plan.ID, rollup.PhasingInput{Method: "custom", Version: ptr(1),
		Months: []rollup.PhasingMonthInput{{Month: "2027-03", Amount: "300000"}, {Month: "2027-09", Amount: "300000"}}}); err != nil {
		t.Fatalf("SetBudgetPhasing: %v", err)
	}

	same, err := f.service.Copy(t.Context(), priya, plan.ID, CopyRequest{Name: "Same year", Year: ptr(2027)})
	next, nextErr := f.service.Copy(t.Context(), priya, plan.ID, CopyRequest{Name: "Next year", Year: ptr(2028)})

	if err != nil || same.Plan.Currency != "GBP" || same.Plan.PlanType != TypeCrossMarket || same.Plan.BudgetPhasing.Method != rollup.MethodCustom ||
		same.Plan.BudgetPhasing.Months[0].Month != "2027-03" {
		t.Errorf("same-year copy = %+v, %v", same.Plan.BudgetPhasing, err)
	}
	if nextErr != nil || next.Plan.Currency != "GBP" || next.Plan.BudgetPhasing.Months[1].Month != "2028-09" ||
		next.Plan.BudgetPhasing.Total.Format() != "600,000.00 GBP" || next.Plan.BudgetPhasing.Problem != nil {
		t.Errorf("next-year copy = %+v, %v", next.Plan.BudgetPhasing, nextErr)
	}
	if source := f.stored(t, plan.ID); source.BudgetPhasing.Months[0].Month != "2027-03" {
		t.Errorf("the source's phasing moved: %+v", source.BudgetPhasing)
	}
}

// A new plan starts in USD, Operational (FR-011, FR-032).
func TestPlanDefaults_CurrencyType(t *testing.T) {
	f := newFixture(t)

	defaults, err := f.service.Defaults(t.Context(), priya)

	if err != nil || defaults.Currency != "USD" || defaults.PlanType != TypeOperational {
		t.Errorf("Defaults = %+v, %v", defaults, err)
	}
}

// A change of the plan type is logged in the plan's ordinary change entry, with the actor and the types as
// people read them (FR-032, FR-049, data-model §7).
func TestUpdatePlan_PlanTypeLogged_FR032_FR049(t *testing.T) {
	f := newFixture(t)
	plan := f.create(t, priya, input("FY2026 Lubricants Europe"))

	edit := updateOf(f.stored(t, plan.ID))
	edit.PlanType = ptr(string(TypeStrategic))
	changed, err := f.service.Update(t.Context(), priya, plan.ID, edit)
	wrong := updateOf(f.stored(t, plan.ID))
	wrong.PlanType = ptr("tactical")
	_, refused := f.service.Update(t.Context(), priya, plan.ID, wrong)

	if err != nil || changed.PlanType != TypeStrategic {
		t.Fatalf("Update = %s, %v", changed.PlanType, err)
	}
	entry := f.entries(t)[0]
	var got []string
	for _, c := range entry.Changes {
		got = append(got, c.Label+": "+c.OldText+" → "+c.NewText)
	}
	if entry.Actor != priya || entry.RecordID != plan.ID || !reflect.DeepEqual(got, []string{"Plan type: Operational → Strategic"}) {
		t.Errorf("log entry by %v on %s: %q", entry.Actor, entry.RecordID, got)
	}
	assertError(t, refused, problems("planType", "Choose Operational, Strategic or Cross-market."))
}
