package activities_test

import (
	"reflect"
	"slices"
	"testing"

	"northstar/internal/activities"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// Feature 009's Home reads the activities a person owns, in every plan, as their pages show them: path,
// display status, unassigned planned cost, the fields still needed and the line items needing attention
// (FR-019 – FR-024, 009 FR-011, FR-014).
func TestOwnedBy_AsTheirPagesShowThem_FR019_FR022(t *testing.T) {
	f := newFixture(t)
	f.lineItems.facts["A-1001"] = activities.LineItemFacts{
		Count: 2, PlannedCostUSD: money.FromCents(12_120_000), // 121,200.00 USD: 1,200.00 more than the planned cost
		Problems: []activities.LineItemProblem{{LineItemID: "L-1002", LineItemName: "Film production", Code: "missingField",
			Field: "attributes.SA-1033", Message: "GL code — needed to submit"}},
	}

	owned, err := f.service.OwnedBy(t.Context(), daniel, priya.ID)
	nobody, err2 := f.service.OwnedBy(t.Context(), daniel, "")

	if err != nil || err2 != nil {
		t.Fatalf("OwnedBy: %v, %v", err, err2)
	}
	var ids []string
	for _, o := range owned {
		ids = append(ids, o.ID)
	}
	if want := []string{"A-1001", "A-1002", "A-1003", "A-1004", "A-1005", "A-1006", "A-1007"}; !slices.Equal(ids, want) {
		t.Fatalf("Priya owns %q, want %q", ids, want)
	}
	for _, o := range owned {
		view := f.get(t, o.ID)
		if o.Name != view.Name || o.Plan != view.Plan || o.Campaign != view.Campaign || o.Initiative != view.Initiative ||
			o.Status != view.Status || o.DisplayStatus != view.DisplayStatus || !reflect.DeepEqual(o.StartDate, view.StartDate) ||
			!reflect.DeepEqual(o.EndDate, view.EndDate) || o.Ongoing != view.Ongoing || !reflect.DeepEqual(o.PlannedCost, view.PlannedCost) ||
			!reflect.DeepEqual(o.UnassignedPlannedCost, view.UnassignedPlannedCost) || !o.UpdatedAt.Equal(view.UpdatedAt) ||
			len(o.Missing) != view.Completeness.MissingFields || o.LineItemsNeedingAttention != view.Completeness.LineItemsNeedingAttention {
			t.Errorf("%s = %+v\nwant what its page shows %+v", o.ID, o, view)
		}
	}
	launchFilm := owned[0]
	wantMissing := []string{"Company code", "Cost centre", "PIR"}
	var labels []string
	for _, m := range launchFilm.Missing {
		labels = append(labels, m.Label)
	}
	if !slices.Equal(labels, wantMissing) || launchFilm.UnassignedPlannedCost.Format() != "-1,200.00 USD" ||
		launchFilm.LineItemsNeedingAttention != 1 || launchFilm.Plan != (errs.RecordRef{Type: "plan", ID: "P-1001", Name: "FY2026 Lubricants Europe"}) {
		t.Errorf("Launch film = %+v; want 3 fields needed, -1,200.00 USD unassigned, 1 line item", launchFilm)
	}
	if len(nobody) != 0 || nobody == nil {
		t.Errorf("OwnedBy(\"\") = %v, want an empty list", nobody)
	}
}
