package activities_test

import (
	"slices"
	"testing"

	"mapp/internal/activities"
	"mapp/internal/platform/money"
)

// An activity without a planned cost can still be split: its line items move, no amount does, and both
// activities stay without a planned cost; an amount to move is refused (016 data-model §3.6, FR-026, FR-028).
func TestSplit_NoPlannedCostMovesLineItemsOnly(t *testing.T) {
	f := newFixture(t)
	view := f.create(t, ukInitiativeID, activities.Input{Name: "Dealer dinner", ActivityTypeID: eventType})
	gbp := func(text string) money.Money { return money.Money{Amount: amount(t, text), Currency: "GBP"} }
	f.lineItems.items = map[string][]activities.LineItemBrief{view.ID: {
		{ID: "L-9001", Name: "Catering", PlannedCost: gbp("800.00"), PlannedCostPlan: gbp("800.00")},
		{ID: "L-9002", Name: "Room", PlannedCost: gbp("200.00"), PlannedCostPlan: gbp("200.00")},
	}}
	f.lineItems.facts[view.ID] = activities.LineItemFacts{Count: 2, PlannedCostPlan: amount(t, "1000.00"), Currency: "GBP"}
	version := view.Version
	withAmount := "100.00"

	_, refused := f.service.Split(t.Context(), priya, view.ID, activities.SplitRequest{
		Name: "Dealer dinner (split)", InitiativeID: ukInitiativeID, LineItemIDs: []string{"L-9001"}, PlannedCost: &withAmount, Version: &version,
	})
	result, err := f.service.Split(t.Context(), priya, view.ID, activities.SplitRequest{
		Name: "Dealer dinner (split)", InitiativeID: ukInitiativeID, LineItemIDs: []string{"L-9001"}, Version: &version,
	})

	assertError(t, refused, problems("plannedCost", "This activity has no planned cost."))
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if result.Activity.PlannedCost != nil || result.NewActivity.PlannedCost != nil || result.NewActivity.Name != "Dealer dinner (split)" {
		t.Errorf("planned costs after the split: original %v, new %v; want none", result.Activity.PlannedCost, result.NewActivity.PlannedCost)
	}
	if want := []string{"[L-9001] → " + result.NewActivity.ID}; !slices.Equal(f.lineItems.moves, want) {
		t.Errorf("moves = %q, want %q", f.lineItems.moves, want)
	}
	if result.Activity.LineItems.Count != 1 || result.NewActivity.LineItems.Count != 1 {
		t.Errorf("line items: original %d, new %d; want one each", result.Activity.LineItems.Count, result.NewActivity.LineItems.Count)
	}
}
