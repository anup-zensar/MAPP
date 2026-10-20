package lineitems

import (
	"context"
	"testing"

	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// ukLaunchEvent is A-1012 of the 016 demo data: "UK launch event" in the GBP plan P-1004, 40,000.00 GBP,
// 01 Mar – 31 May 2027.
const ukLaunchEvent = "A-1012"

// withGBPActivity adds A-1012 to the fakes.
func withGBPActivity(f *fixture) {
	planned := amount("40000.00")
	f.activities.facts[ukLaunchEvent] = ActivityFacts{ID: ukLaunchEvent, Name: "UK launch event", OwnerID: priya.ID, Status: "created",
		StartDate: day("2027-03-01"), EndDate: day("2027-05-31"), PlannedCost: &planned, PlanYear: 2027, DefaultSpread: SpreadEvenly,
		PlanCurrency: "GBP", Path: []errs.RecordRef{
			{Type: "plan", ID: "P-1004", Name: "FY2027 Lubricants UK & Ireland"},
			{Type: "campaign", ID: "C-1005", Name: "Helix UK & Ireland"},
			{Type: "initiative", ID: "I-1009", Name: "UK"},
		}}
}

// A line item in another currency than the plan's is converted into the plan currency once, half up:
// 23,000.00 EUR at 0.92 is 19,750.00 GBP at 0.79 (FR-015, data-model §3.1); one in the plan currency is exact.
func TestPlannedCostPlan_FR015(t *testing.T) {
	f := newTestService(t)
	withGBPActivity(f)

	venue := f.create(t, priya, ukLaunchEvent, input("Venue", "GBP", "15000.00"))
	agency := f.create(t, priya, ukLaunchEvent, input("Agency fee (Ireland)", "EUR", "23000.00"))

	if venue.PlannedCostPlan.Format() != "15,000.00 GBP" || agency.PlannedCostPlan.Format() != "19,750.00 GBP" ||
		agency.PlannedCostUSD.Format() != "25,000.00 USD" {
		t.Errorf("plan currency: venue %v, agency %v (USD %v)", venue.PlannedCostPlan, agency.PlannedCostPlan, agency.PlannedCostUSD)
	}
}

// The unassigned planned cost is worked out in the plan currency: 40,000.00 − (15,000.00 + 19,750.00) =
// 5,250.00 GBP, shown in the reader's display currency from it (FR-015, US2-3).
func TestBalance_PlanCurrency_FR015(t *testing.T) {
	f := newTestService(t)
	withGBPActivity(f)
	f.create(t, priya, ukLaunchEvent, input("Venue", "GBP", "15000.00"))
	f.create(t, priya, ukLaunchEvent, input("Agency fee (Ireland)", "EUR", "23000.00"))

	section, err := f.service.Section(t.Context(), priya, ukLaunchEvent)
	options, optionsErr := f.service.FormOptions(t.Context(), priya, ukLaunchEvent)

	if err != nil || section.Balance.Balance.Format() != "6,645.57 USD" || section.Balance.State != BalanceLeft ||
		section.Activity.PlannedCost.Format() != "40,000.00 GBP" {
		t.Errorf("section balance = %+v (activity %v), %v", section.Balance, section.Activity.PlannedCost, err)
	}
	if optionsErr != nil || options.Balance.Balance.Format() != "5,250.00 GBP" || options.Balance.Assigned.Format() != "34,750.00 GBP" ||
		options.Defaults.Currency != "GBP" || options.Defaults.PlannedCost.Format() != "5,250.00 GBP" {
		t.Errorf("form balance = %+v, defaults %+v, %v", options.Balance, options.Defaults, optionsErr)
	}
}

// The spread grid is worked out in the plan currency: a EUR line item's months converted so they add up to its
// plan-currency value (FR-015).
func TestGrid_PlanCurrency_FR015(t *testing.T) {
	f := newTestService(t)
	withGBPActivity(f)
	f.create(t, priya, ukLaunchEvent, input("Venue", "GBP", "15000.00"))
	f.create(t, priya, ukLaunchEvent, input("Agency fee (Ireland)", "EUR", "23000.00"))

	var grid SpreadGrid
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		facts, err := f.service.facts(ctx, ukLaunchEvent)
		if err != nil {
			return err
		}
		items, err := f.service.repo.ListByActivity(ctx, ukLaunchEvent)
		if err != nil {
			return err
		}
		views, err := f.service.views(ctx, facts, items)
		grid = Grid(facts, views)
		return err
	})

	if err != nil || len(grid.Rows) != 2 || grid.Rows[1].Total.Format() != "19,750.00 GBP" || grid.Total.Format() != "34,750.00 GBP" ||
		grid.Rows[0].Cells[0].Format() != "5,000.00 GBP" {
		t.Fatalf("grid = %+v, %v", grid, err)
	}
	var sum money.Amount
	for _, cell := range grid.Rows[1].Cells {
		if cell != nil {
			sum = sum.Add(cell.Amount)
		}
	}
	if sum.Format() != "19,750.00" {
		t.Errorf("the EUR line item's months add up to %s GBP, want 19,750.00", sum.Format())
	}
}

// The rounding hint is measured in the plan currency: a few cents left while a line item uses another
// currency than the plan's (D-145, data-model §3.1).
func TestRoundingHint_PlanCurrency(t *testing.T) {
	planned := amount("100.00")
	f := ActivityFacts{PlanCurrency: "GBP", PlannedCost: &planned}

	foreign := Balance(f, []money.Amount{amount("99.97")}, []string{"EUR"})
	own := Balance(f, []money.Amount{amount("99.97")}, []string{"GBP"})

	if !foreign.RoundingHint || foreign.Balance.Format() != "0.03 GBP" || own.RoundingHint {
		t.Errorf("EUR line item: %+v; GBP line item: %+v", foreign, own)
	}
}

// A line item's currency comes with the date and source of its current rate (FR-009).
func TestLineItemCurrency_RateDate_FR009(t *testing.T) {
	f := newTestService(t)
	withGBPActivity(f)

	agency := f.create(t, priya, ukLaunchEvent, input("Agency fee (Ireland)", "EUR", "23000.00"))
	options, err := f.service.FormOptions(t.Context(), priya, ukLaunchEvent)

	if agency.Currency.RateDate == nil || agency.Currency.RateDate.String() != "2026-10-19" || agency.Currency.RateSource == nil ||
		*agency.Currency.RateSource != "gsapImport" {
		t.Errorf("currency = %+v", agency.Currency)
	}
	if err != nil || options.Currencies[0].Code != "USD" || options.Currencies[0].RateDate != nil {
		t.Errorf("form currencies = %+v, %v", options.Currencies, err)
	}
}
