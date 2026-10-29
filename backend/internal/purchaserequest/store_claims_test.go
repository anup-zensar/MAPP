package purchaserequest_test

import (
	"context"
	"testing"

	"mapp/internal/platform/money"
	"mapp/internal/purchaserequest"
)

func TestPostgresStore_ClaimsKeepTheApprovedVersion_038_FR001_C1(t *testing.T) {
	t.Parallel()
	db := seededDatabase(t)
	store := purchaserequest.NewPostgresStore(db)
	pr, found, err := store.Get(t.Context(), "PR-1001")
	if err != nil || !found || len(pr.Lines) == 0 {
		t.Fatalf("PR-1001 = %v %v", found, err)
	}
	line := pr.Lines[0]
	pr.Status, pr.ChangePending = purchaserequest.StatusInReview, true
	pr.Approved = &purchaserequest.Version{Name: pr.Name, ApprovedAt: seededAt, Lines: []purchaserequest.VersionLine{
		{ID: line.ID, LineItemID: line.LineItemID, Currency: line.Currency, Amount: money.FromCents(1_580_000)},
		{ID: "PRL-9999", LineItemID: "L-1009", Currency: line.Currency, Amount: money.FromCents(50_000)},
	}}
	if err = db.Write(t.Context(), func(ctx context.Context) error { return store.Put(ctx, pr) }); err != nil {
		t.Fatalf("put: %v", err)
	}

	claims, err := store.Claims(t.Context(), []string{line.LineItemID, "L-1009"})
	if err != nil {
		t.Fatalf("claims: %v", err)
	}
	var kept, moved *purchaserequest.Claim
	for i := range claims {
		switch claims[i].LineID {
		case line.ID:
			kept = &claims[i]
		case "PRL-9999":
			moved = &claims[i]
		}
	}
	if kept == nil || kept.Approved == nil || kept.Approved.Amount != money.FromCents(1_580_000) || kept.RemovedByChange {
		t.Errorf("038 FR-001 (C-1) the changed line = %+v, want its approved 15,800.00 kept", kept)
	}
	if moved == nil || !moved.RemovedByChange || moved.LineItemID != "L-1009" || moved.Amount.Amount != money.FromCents(50_000) {
		t.Errorf("038 FR-001 (C-1) the line moved off L-1009 = %+v, want it still claimed", moved)
	}
}
