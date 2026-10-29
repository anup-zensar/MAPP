package bulk_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/bulk"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/database/dbtest"
)

// A bulk action is read back only for the person who ran it: anyone else gets nothing, as for a missing one.
func TestPostgresStore_GetAction_OnlyTheActor_012FR020(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	store := bulk.NewPostgresStore(db)
	priya := actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	ran := time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	action := bulk.Action{ID: "BLK-1001", Kind: bulk.ActionDelete, Note: "Bulk delete of 2 activities (BLK-1001)", Actor: priya,
		RanAt: ran, BinEntryIDs: []string{"BIN-1004", "BIN-1003"}}
	if err := db.Write(t.Context(), func(ctx context.Context) error { return store.AddAction(ctx, action) }); err != nil {
		t.Fatalf("AddAction() error: %v", err)
	}

	got, found, err := store.GetAction(t.Context(), "BLK-1001", priya.ID)
	if err != nil || !found {
		t.Fatalf("GetAction() by its actor = %v, %v", found, err)
	}
	if diff := cmp.Diff(action, got); diff != "" {
		t.Errorf("GetAction() (-want +got):\n%s", diff)
	}
	if _, found, err := store.GetAction(t.Context(), "BLK-1001", "U-1003"); found || err != nil {
		t.Errorf("GetAction() by another person = %v, %v; want not found", found, err)
	}
}
