package lineitem

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// Copies for features 004–006 (data model §5): every stored field and the same names, new IDs, the actor as
// creator, and no log entries of their own.
func TestCopyLineItems_SameYear(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	agency := f.create(t, priya, launchFilm, complete(input("Agency fee", "EUR", "46000.00")))
	custom := input("Venue hire", "USD", "12000.00")
	custom.Spread = SpreadInput{Method: SpreadCustom, Months: []SpreadMonthInput{{Month: "2026-03", Amount: "12000.00"}}}
	venue := f.create(t, priya, launchFilm, custom)
	f.create(t, priya, retailEvent, input("Not copied", "USD", "1.00"))
	entriesBefore := len(f.log(t))

	var copied int
	err := f.db.Write(t.Context(), func(ctx context.Context) error { // joins the caller's Write
		var err error
		copied, err = f.service.CopyLineItems(ctx, daniel, launchFilm, "A-2001", 0)
		return err
	})

	if err != nil || copied != 2 {
		t.Fatalf("CopyLineItems = %d, %v; want 2 line items", copied, err)
	}
	stored := f.stored(t)
	if len(stored) != 5 {
		t.Fatalf("stored %d line items, want 5", len(stored))
	}
	for _, source := range []string{agency.ID, venue.ID} {
		original := storedByID(t, stored, source)
		dup := lineItemNamed(t, stored, "A-2001", original.Name)
		if slices.Contains([]string{agency.ID, venue.ID}, dup.ID) {
			t.Errorf("copy of %s has the ID %s, want a new one", original.ID, dup.ID)
		}
		if dup.Meta != record.NewMeta(daniel, start) {
			t.Errorf("copy of %s: bookkeeping = %+v, want new, by Daniel", original.ID, dup.Meta)
		}
		dup.ID, dup.ActivityID, dup.Meta = original.ID, original.ActivityID, original.Meta
		if !reflect.DeepEqual(dup, original) {
			t.Errorf("copy of %s =\n%+v\nwant the values of\n%+v", original.ID, dup, original)
		}
	}
	if got := len(f.log(t)); got != entriesBefore {
		t.Errorf("log entries = %d, want %d: a copy writes none of its own", got, entriesBefore)
	}
}

// A copy into the next year's plan moves the finance budget year and the custom months one year on, and
// links the copy to the new year's only active item with the same code, otherwise to none (007-P9, D-148).
func TestCopyLineItems_NextYear_007P9(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	year := func(y int) *int { return &y }
	items := []LineItem{
		{ // 2025 Europe – Digital has its 2026 namesake by code
			ID: "L-9001", ActivityID: launchFilm, Name: "Agency fee", Currency: "USD", PlannedCost: amount("300.00"),
			Spread: Spread{Method: SpreadCustom, Months: []MonthAmount{{Month: "2025-12", Amount: amount("100.00")},
				{Month: "2026-01", Amount: amount("200.00")}}},
			FinanceBudgetYear: year(2025), FinanceBudgetItemID: lastYearItem,
		},
		{ // 2027 has no finance budget items
			ID: "L-9002", ActivityID: launchFilm, Name: "Venue hire", Currency: "USD", PlannedCost: amount("50.00"),
			Spread: Spread{Method: SpreadEvenly, Months: []MonthAmount{}}, FinanceBudgetYear: year(2026), FinanceBudgetItemID: europeEvents,
		},
		{ // nothing to move
			ID: "L-9003", ActivityID: launchFilm, Name: "Printing", Currency: "USD", PlannedCost: amount("10.00"),
			Spread: Spread{Method: SpreadEvenly, Months: []MonthAmount{}},
		},
	}
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		for _, li := range items {
			li.Attributes, li.Meta = settings.AttributeValues{"SA-1031": json.RawMessage(`"SO-1101"`)}, record.NewMeta(priya, start)
			if err := f.service.repo.Put(ctx, li); err != nil {
				return err
			}
		}
		_, err := f.service.CopyLineItems(ctx, priya, launchFilm, "A-2001", 1)
		return err
	})
	if err != nil {
		t.Fatalf("CopyLineItems: %v", err)
	}

	stored := f.stored(t)
	if len(stored) != 6 {
		t.Fatalf("stored %d line items, want 6", len(stored))
	}
	agency, venue, printing := lineItemNamed(t, stored, "A-2001", "Agency fee"), lineItemNamed(t, stored, "A-2001", "Venue hire"),
		lineItemNamed(t, stored, "A-2001", "Printing")
	if *agency.FinanceBudgetYear != 2026 || agency.FinanceBudgetItemID != europeDigital {
		t.Errorf("Agency fee: year %d, item %q; want 2026, %s (same code)", *agency.FinanceBudgetYear, agency.FinanceBudgetItemID, europeDigital)
	}
	wantMonths := []MonthAmount{{Month: "2026-12", Amount: amount("100.00")}, {Month: "2027-01", Amount: amount("200.00")}}
	if !reflect.DeepEqual(agency.Spread.Months, wantMonths) {
		t.Errorf("Agency fee months = %+v, want %+v", agency.Spread.Months, wantMonths)
	}
	if *venue.FinanceBudgetYear != 2027 || venue.FinanceBudgetItemID != "" {
		t.Errorf("Venue hire: year %d, item %q; want 2027 and no item", *venue.FinanceBudgetYear, venue.FinanceBudgetItemID)
	}
	if printing.FinanceBudgetYear != nil || printing.FinanceBudgetItemID != "" {
		t.Errorf("Printing: year %v, item %q; want neither", printing.FinanceBudgetYear, printing.FinanceBudgetItemID)
	}
	if original := storedByID(t, stored, "L-9001"); original.Spread.Months[0].Month != "2025-12" || *original.FinanceBudgetYear != 2025 {
		t.Errorf("the original changed: %+v", original)
	}
}

// storedByID returns the line item id among stored.
func storedByID(t *testing.T, stored []LineItem, id string) LineItem {
	t.Helper()
	for _, li := range stored {
		if li.ID == id {
			return li
		}
	}
	t.Fatalf("line item %s is not stored", id)
	return LineItem{}
}

// lineItemNamed returns the line item called name of the activity activityID among stored.
func lineItemNamed(t *testing.T, stored []LineItem, activityID, name string) LineItem {
	t.Helper()
	for _, li := range stored {
		if li.ActivityID == activityID && li.Name == name {
			return li
		}
	}
	t.Fatalf("no line item %q in %s", name, activityID)
	return LineItem{}
}
