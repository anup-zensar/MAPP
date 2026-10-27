package plans_test

import (
	"context"
	"testing"

	"mapp/internal/plans"
	"mapp/internal/platform/store"
	"mapp/internal/rollup"
)

// A plan stored before 016 reads with the defaults of data-model §4.5: currency USD, type Operational, not
// phased; a plan stored with them keeps them (016 T011).
func TestPlanStore_NormalisesOlderRows(t *testing.T) {
	t.Parallel()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := plans.NewJSONStore()
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
