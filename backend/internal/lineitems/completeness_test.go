package lineitems

import (
	"reflect"
	"testing"

	"mapp/internal/settings"
)

// filledIn is a line item with everything needed to submit: owner, finance budget year and item.
func filledIn() LineItem {
	year := 2026
	return LineItem{OwnerID: priya.ID, FinanceBudgetYear: &year, FinanceBudgetItemID: europeDigital}
}

// The missing list counts the empty needed-to-submit fields in a fixed order: owner, Shell attributes,
// finance budget year, finance budget item; nothing missing is an empty list, "All filled in" (FR-028).
func TestMissing_NeededToSubmit_FR028(t *testing.T) {
	t.Parallel()
	active := budgetItems[0]

	if got := Missing(filledIn(), nil, &active, true); got == nil || len(got) != 0 {
		t.Errorf("filled in: missing = %#v, want an empty list", got)
	}

	got := Missing(LineItem{}, nil, nil, true)

	want := []settings.MissingField{
		{Field: "ownerId", Label: "Owner"},
		{Field: "financeBudgetYear", Label: "Finance budget year"},
		{Field: "financeBudgetItemId", Label: "Finance budget item"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("missing = %+v\nwant %+v", got, want)
	}
}

// The Shell attributes Settings marks needed to submit (Spend type, GL code) come after the owner, in the
// order feature 002 lists them (FR-014, FR-028).
func TestMissing_Attributes_FR014(t *testing.T) {
	t.Parallel()
	attributes := []settings.MissingField{
		{Field: "attributes." + spendType, Label: "Spend type"},
		{Field: "attributes." + glCode, Label: "GL code"},
	}
	li := filledIn()
	li.OwnerID, li.FinanceBudgetItemID = "", ""

	got := Missing(li, attributes, nil, true)

	want := []settings.MissingField{
		{Field: "ownerId", Label: "Owner"}, attributes[0], attributes[1],
		{Field: "financeBudgetItemId", Label: "Finance budget item"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("missing = %+v\nwant %+v", got, want)
	}
}

// A linked finance budget item that became inactive is kept, but counts until another is chosen (FR-016).
func TestMissing_InactiveItem_FR016(t *testing.T) {
	t.Parallel()
	legacy := budgetItems[3]
	li := filledIn()
	li.FinanceBudgetItemID = legacy.ID

	got := Missing(li, nil, &legacy, true)

	want := []settings.MissingField{{Field: "financeBudgetItemId", Label: "Choose an active finance budget item — 'Europe – Legacy' is no longer active."}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("missing = %+v, want %+v", got, want)
	}
}

// A custom spread that does not add up to the planned cost saves, but counts as a problem (FR-019): 900.00
// of 1,000.00 spread.
func TestMissing_CustomNotAddingUp_FR019(t *testing.T) {
	t.Parallel()
	f := newFakeActivities().facts[launchFilm]
	li := lineItem("1000.00", SpreadCustom, "2026-03", "450.00", "2026-04", "450.00")
	li.OwnerID, li.FinanceBudgetYear, li.FinanceBudgetItemID = filledIn().OwnerID, filledIn().FinanceBudgetYear, europeDigital
	spread := spreadView(li, f, rate("1"), rate("1"))

	got := Missing(li, nil, &budgetItems[0], spread.AddsUp)

	if spread.AddsUp || spread.CustomTotal == nil || spread.CustomTotal.Amount.String() != "900.00" {
		t.Errorf("spread = %+v, want 900.00 that doesn't add up", spread)
	}
	want := []settings.MissingField{{Field: "spread", Label: "Spread doesn't add up to the planned cost"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("missing = %+v, want %+v", got, want)
	}
}

// The SWBS request status follows the flag and the activity's status (FR-024, domain model §2.4).
func TestSwbsStatus_FR024(t *testing.T) {
	t.Parallel()
	tests := []struct {
		request bool
		status  string
		want    SwbsStatus
	}{
		{false, "created", SwbsNotRequested},
		{false, "approved", SwbsNotRequested},
		{true, "created", SwbsWaitingForApproval},
		{true, "submitted", SwbsWaitingForApproval},
		{true, "declined", SwbsWaitingForApproval},
		{true, "approved", SwbsReadyToSend},
	}
	for _, tt := range tests {
		got := SwbsStatusOf(LineItem{RequestSwbs: tt.request}, ActivityFacts{Status: tt.status})

		if got != tt.want {
			t.Errorf("request %v, activity %s: %s, want %s", tt.request, tt.status, got, tt.want)
		}
	}
}
