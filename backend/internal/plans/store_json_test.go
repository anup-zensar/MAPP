package plans_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
	"northstar/internal/rollup"
	"northstar/internal/settings"
)

// A plan is saved in the table "plans" and read back exactly — USD budget as a decimal string, Shell
// attribute values in "attributes" — and the plans of a year are found (data model §1, D-048).
func TestJSONRepository_RoundTrip(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := plans.NewJSONRepository()
	budget := money.FromCents(120_000_000)
	meta := record.NewMeta(actor.System, time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC))
	lubricants := plans.Plan{
		ID: "P-1001", Name: "FY2026 Lubricants Europe", Year: 2026, ModelID: "MOD-1001", TeamIDs: []string{"T-1001"},
		OwnerID: "U-1002", Description: "Lubricants marketing in Europe for 2026.", Budget: &budget,
		DefaultFinanceBudgetID: "FB-1001", DefaultSpread: plans.SpreadEvenly, GoalStage: plans.StageINQ, GoalAmount: "250000",
		Attributes: settings.AttributeValues{"SA-1001": json.RawMessage(`["SO-1002"]`)}, Status: plans.StatusPublished, Meta: meta,
	}
	brand := plans.Plan{
		ID: "P-1003", Name: "FY2027 Global Brand", Year: 2027, ModelID: "MOD-1001", TeamIDs: []string{}, AOP: true,
		DefaultSpread: plans.SpreadEvenly, GoalStage: plans.StageRevenue, GoalAmount: "2000000.00",
		Attributes: settings.AttributeValues{}, Status: plans.StatusDraft, Meta: meta,
	}

	err = db.Write(t.Context(), func(ctx context.Context) error {
		if err := repo.Put(ctx, lubricants); err != nil {
			return err
		}
		return repo.Put(ctx, brand)
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = db.Read(t.Context(), func(ctx context.Context) error {
		got, found, err := repo.Get(ctx, "P-1001")
		if err != nil || !found || !reflect.DeepEqual(got, lubricants.Normalised()) {
			t.Errorf("Get = %+v, %v, %v", got, found, err)
		}
		raw, _, err := store.Get[map[string]json.RawMessage](ctx, "plans", "P-1001")
		if err != nil || string(raw["budget"]) != `"1200000.00"` || string(raw["attributes"]) != `{"SA-1001":["SO-1002"]}` {
			t.Errorf("stored budget %s, attributes %s, %v", raw["budget"], raw["attributes"], err)
		}
		if all, err := repo.List(ctx); err != nil || len(all) != 2 {
			t.Errorf("List = %d plans, %v", len(all), err)
		}
		ofYear, err := repo.PlansOfYear(ctx, 2027)
		if err != nil || len(ofYear) != 1 || ofYear[0].ID != "P-1003" {
			t.Errorf("PlansOfYear(2027) = %+v, %v", ofYear, err)
		}
		if none, err := repo.PlansOfYear(ctx, 2030); err != nil || none == nil || len(none) != 0 {
			t.Errorf("PlansOfYear(2030) = %#v, %v; want an empty list", none, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	err = db.Write(t.Context(), func(ctx context.Context) error { return repo.Delete(ctx, "P-1003") })
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = db.Read(t.Context(), func(ctx context.Context) error {
		if _, found, _ := repo.Get(ctx, "P-1003"); found {
			t.Error("the deleted plan is still stored")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// An attention item's BlocksPublish is for the publish check only, never part of the answer.
func TestAttentionItem_BlocksPublishNotInJSON(t *testing.T) {
	encoded, err := json.Marshal(plans.AttentionItem{BlocksPublish: true, Missing: []settings.MissingField{{Field: "ownerId", Label: "Owner"}}})
	if err != nil || string(encoded) != `{"record":{"type":"","id":"","name":""},"missing":[{"field":"ownerId","label":"Owner"}]}` {
		t.Errorf("JSON = %s, %v", encoded, err)
	}
}

// A plan stored before 016 reads with the defaults of data-model §4.5: currency USD, type Operational, not
// phased; a plan stored with them keeps them (016 T011).
func TestPlanStore_NormalisesOlderRows(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := plans.NewJSONRepository()
	err = db.Write(t.Context(), func(ctx context.Context) error {
		if err := store.Put(ctx, "plans", "P-1001", map[string]any{"id": "P-1001", "name": "Old", "year": 2026, "teamIds": []string{}}); err != nil {
			return err
		}
		gbp := plans.Plan{ID: "P-1004", Name: "New", Year: 2027, TeamIDs: []string{}, Currency: "GBP", PlanType: plans.TypeCrossMarket,
			BudgetPhasing: rollup.Phasing{Method: rollup.MethodEvenly, Months: []rollup.PhasingMonth{}}}
		return repo.Put(ctx, gbp)
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = db.Read(t.Context(), func(ctx context.Context) error {
		old, _, err := repo.Get(ctx, "P-1001")
		if err != nil || old.Currency != "USD" || old.PlanType != plans.TypeOperational || old.BudgetPhasing.Method != rollup.MethodNone ||
			old.BudgetPhasing.Months == nil {
			t.Errorf("older plan = %+v, %v", old, err)
		}
		all, err := repo.List(ctx)
		if err != nil || len(all) != 2 || all[1].Currency != "GBP" || all[1].PlanType != plans.TypeCrossMarket ||
			all[1].BudgetPhasing.Method != rollup.MethodEvenly || all[0].Currency != "USD" {
			t.Errorf("List = %+v, %v", all, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}
