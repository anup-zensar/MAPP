package lineitems_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"northstar/internal/lineitems"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
	"northstar/internal/settings"
)

// A line item is saved in the table "line_items" and read back exactly — amounts as decimal strings in its
// currency, the custom months, the finance budget link — and the line items of activities are found in
// creation order (data model §1).
func TestJSONRepository_RoundTrip(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := lineitems.NewJSONRepository()
	meta := record.NewMeta(actor.System, time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC))
	year := 2026
	fee := lineitems.LineItem{
		ID: "L-1001", ActivityID: "A-1001", Name: "Agency fee", OwnerID: "U-1002", Currency: "USD",
		PlannedCost: money.FromCents(5_000_000), Spread: lineitems.Spread{Method: lineitems.SpreadEvenly, Months: []lineitems.MonthAmount{}},
		Attributes: settings.AttributeValues{"SA-1032": json.RawMessage(`"SO-1101"`)}, RequestSwbs: true,
		FinanceBudgetYear: &year, FinanceBudgetItemID: "FBI-1001", Meta: meta,
	}
	signage := lineitems.LineItem{
		ID: "L-1007", ActivityID: "A-1008", Name: "Station signage", Currency: "USD", PlannedCost: money.FromCents(100_000),
		Spread: lineitems.Spread{Method: lineitems.SpreadCustom, Months: []lineitems.MonthAmount{
			{Month: "2026-04", Amount: money.FromCents(45_000)}, {Month: "2026-05", Amount: money.FromCents(45_000)},
		}},
		Attributes: settings.AttributeValues{}, Meta: meta,
	}

	err = db.Write(t.Context(), func(ctx context.Context) error {
		if err := repo.Put(ctx, signage); err != nil {
			return err
		}
		return repo.Put(ctx, fee)
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = db.Read(t.Context(), func(ctx context.Context) error {
		if got, found, err := repo.Get(ctx, "L-1007"); err != nil || !found || !reflect.DeepEqual(got, signage) {
			t.Errorf("Get = %+v, %v, %v", got, found, err)
		}
		raw, _, _ := store.Get[map[string]json.RawMessage](ctx, "line_items", "L-1007")
		if string(raw["plannedCost"]) != `"1000.00"` || string(raw["spread"]) != `{"method":"custom","months":[{"month":"2026-04","amount":"450.00"},{"month":"2026-05","amount":"450.00"}]}` ||
			string(raw["financeBudgetYear"]) != "null" || string(raw["contractedAmount"]) != "null" {
			t.Errorf("stored as %s / %s / %s / %s", raw["plannedCost"], raw["spread"], raw["financeBudgetYear"], raw["contractedAmount"])
		}
		all, err := repo.List(ctx)
		if err != nil || len(all) != 2 || all[0].ID != "L-1001" {
			t.Errorf("List = %+v, %v; want L-1001 first", all, err)
		}
		if of, err := repo.ListByActivity(ctx, "A-1001"); err != nil || !reflect.DeepEqual(of, []lineitems.LineItem{fee}) {
			t.Errorf("ListByActivity = %+v, %v", of, err)
		}
		if of, err := repo.ListByActivities(ctx, []string{"A-1001", "A-1008"}); err != nil || len(of) != 2 {
			t.Errorf("ListByActivities = %d, %v", len(of), err)
		}
		if none, err := repo.ListByActivity(ctx, "A-1005"); err != nil || none == nil || len(none) != 0 {
			t.Errorf("ListByActivity(none) = %#v, %v; want an empty list", none, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	err = db.Write(t.Context(), func(ctx context.Context) error { return repo.Delete(ctx, "L-1001") })
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = db.Read(t.Context(), func(ctx context.Context) error {
		if _, found, _ := repo.Get(ctx, "L-1001"); found {
			t.Error("the deleted line item is still stored")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}
