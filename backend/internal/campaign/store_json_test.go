package campaign_test

import (
	"context"
	"testing"

	"mapp/internal/campaign"
	"mapp/internal/platform/store"
	"mapp/internal/rollup"
)

// A campaign or initiative stored before 016 reads as not phased (016 data-model §4.5, T012).
func TestJSONRepository_PhasingOfOlderRows(t *testing.T) {
	t.Parallel()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := campaign.NewJSONStore()
	err = db.Write(t.Context(), func(ctx context.Context) error {
		if err := store.Put(ctx, "campaigns", "C-1001", map[string]any{"id": "C-1001", "planId": "P-1001", "name": "Old"}); err != nil {
			return err
		}
		return store.Put(ctx, "initiatives", "I-1001", map[string]any{"id": "I-1001", "campaignId": "C-1001", "name": "Old"})
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = db.Read(t.Context(), func(ctx context.Context) error {
		c, _, err := repo.GetCampaign(ctx, "C-1001")
		if err != nil || c.BudgetPhasing.Method != rollup.MethodNone || c.BudgetPhasing.Months == nil {
			t.Errorf("campaign phasing = %+v, %v", c.BudgetPhasing, err)
		}
		list, err := repo.ListInitiatives(ctx)
		if err != nil || len(list) != 1 || list[0].BudgetPhasing.Method != rollup.MethodNone {
			t.Errorf("initiatives = %+v, %v", list, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}
