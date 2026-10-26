package activities_test

import (
	"testing"

	"mapp/internal/activities"
	"mapp/internal/platform/money"
)

// Activities of a plan planned in another currency than USD (016 US2) on the demo data: P-1004 "FY2027
// Lubricants UK & Ireland" is planned in GBP (0.79 per USD); its initiative I-1009 "UK" holds A-1011 "UK launch
// event", 40,000.00 GBP.

const (
	ukInitiativeID = "I-1009"
	ukLaunchID     = "A-1011"
)

// gbpInput is an amount in pounds as a request sends it.
func gbpInput(amount string) *money.Input {
	return &money.Input{Amount: amount, Currency: "GBP"}
}

// The planned cost is typed and kept in the plan currency; another currency is refused with the plan's (FR-011).
func TestPlannedCost_MustBePlanCurrency_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, refused := f.service.Create(t.Context(), priya, ukInitiativeID, activities.Input{
		Name: "Dealer dinner", ActivityTypeID: eventType, PlannedCost: usdInput("1000.00"),
	})
	view := f.create(t, ukInitiativeID, activities.Input{Name: "Dealer dinner", ActivityTypeID: eventType, PlannedCost: gbpInput("1000.00")})

	assertError(t, refused, problems("plannedCost", "Enter the amount in GBP, the plan's currency."))
	stored, _ := f.stored(t, view.ID)
	if view.PlannedCost == nil || view.PlannedCost.Format() != "1,000.00 GBP" || stored.PlannedCost.Currency != "GBP" {
		t.Errorf("planned cost = %v, stored %v; want 1,000.00 GBP", view.PlannedCost, stored.PlannedCost)
	}
	_, refusedEdit := f.edit(t, priya, view.ID, func(u *activities.Update) { u.PlannedCost = usdInput("2000.00") })
	assertError(t, refusedEdit, problems("plannedCost", "Enter the amount in GBP, the plan's currency."))
}

// The activity's amounts are in the plan currency, and the display amounts are converted from it (FR-013):
// 40,000.00 GBP → 50,632.91 USD; with 34,750.00 GBP on line items the unassigned 5,250.00 GBP → 6,645.57 USD.
func TestActivityDisplay_FromPlanCurrency_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.lineItems.facts[ukLaunchID] = activities.LineItemFacts{Count: 2, PlannedCostPlan: amount(t, "34750.00"), Currency: "GBP"}

	view := f.get(t, ukLaunchID)
	rows, err := f.service.ListForPlan(t.Context(), priya, "P-1004")
	if err != nil {
		t.Fatalf("ListForPlan: %v", err)
	}

	if view.PlannedCost.Format() != "40,000.00 GBP" || view.UnassignedPlannedCost.Format() != "5,250.00 GBP" ||
		view.LineItems.PlannedCost.Format() != "34,750.00 GBP" {
		t.Errorf("planned %v, unassigned %v, line items %v; want GBP", view.PlannedCost, view.UnassignedPlannedCost, view.LineItems.PlannedCost)
	}
	if view.Display.PlannedCost.Format() != "50,632.91 USD" || view.Display.UnassignedPlannedCost.Format() != "6,645.57 USD" {
		t.Errorf("display = %v / %v, want 50,632.91 USD / 6,645.57 USD", view.Display.PlannedCost, view.Display.UnassignedPlannedCost)
	}
	var row *activities.Summary
	for i := range rows {
		if rows[i].ID == ukLaunchID {
			row = &rows[i]
		}
	}
	if row == nil || row.PlannedCost.Format() != "40,000.00 GBP" || row.Display.PlannedCost.Format() != "50,632.91 USD" {
		t.Errorf("plan-tree row = %+v", row)
	}
}

// An activity type's default planned cost, kept in USD, is offered converted into the plan currency at the
// current rate, with the USD value beside it for the note (FR-016): Event 25,000.00 USD = 19,750.00 GBP.
func TestActivityForm_TypeDefaultConverted_FR016(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	gbp, err := f.service.NewForm(t.Context(), priya, ukInitiativeID)
	if err != nil {
		t.Fatalf("NewForm(UK): %v", err)
	}
	usd, err := f.service.NewForm(t.Context(), priya, digitalAndSocial)
	if err != nil {
		t.Fatalf("NewForm(Digital & social): %v", err)
	}

	event := typeOption(gbp, eventType)
	if event == nil || event.DefaultPlannedCost.Format() != "19,750.00 GBP" || event.DefaultPlannedCostUSD.Format() != "25,000.00 USD" {
		t.Errorf("Event in the GBP plan = %+v", event)
	}
	if usdEvent := typeOption(usd, eventType); usdEvent == nil || usdEvent.DefaultPlannedCost.Format() != "25,000.00 USD" ||
		usdEvent.DefaultPlannedCostUSD.Format() != "25,000.00 USD" {
		t.Errorf("Event in a USD plan = %+v", usdEvent)
	}
	if content := typeOption(gbp, contentType); content == nil || content.DefaultPlannedCost != nil || content.DefaultPlannedCostUSD != nil {
		t.Errorf("a type without a default = %+v", content)
	}
}

// typeOption returns the form's activity type id; nil when the form does not offer it.
func typeOption(form activities.FormView, id string) *activities.TypeOption {
	for i := range form.ActivityTypes {
		if form.ActivityTypes[i].ID == id {
			return &form.ActivityTypes[i]
		}
	}
	return nil
}

// Goals stay in USD in a GBP plan (FR-019): the calculated revenue comes from the model's USD average deal
// size, and a typed revenue must be USD.
func TestGoals_RevenueStaysUSD_FR019(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	inq := int64(1_000)

	view := f.create(t, ukInitiativeID, activities.Input{Name: "Dealer dinner", ActivityTypeID: eventType,
		Goals: activities.GoalsInput{INQ: &inq}})
	_, refused := f.service.Create(t.Context(), priya, ukInitiativeID, activities.Input{Name: "Fleet day", ActivityTypeID: eventType,
		Goals: activities.GoalsInput{INQ: &inq, SetLaterStagesMyself: true, Revenue: gbpInput("1000.00")}})

	if view.Goals.Revenue == nil || view.Goals.Revenue.Format() != "10,000.00 USD" {
		t.Errorf("calculated revenue = %v, want 10,000.00 USD (1 deal of 10,000.00 USD)", view.Goals.Revenue)
	}
	assertError(t, refused, problems("goals.revenue", "Enter the revenue in USD."))
}

// amount parses a decimal amount.
func amount(t *testing.T, text string) money.Amount {
	t.Helper()
	a, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return a
}
