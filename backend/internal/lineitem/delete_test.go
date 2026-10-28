package lineitem

import (
	"context"
	"reflect"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// A line item that is gone says why (D-154): deleted itself, or deleted with its activity.
func TestGet_GoneMessages_007P15(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee, _ := launchFilmWithTwoItems(t, f)
	gone := f.create(t, priya, launchFilm, input("Gone", "USD", "1.00"))
	if _, err := f.service.Delete(t.Context(), priya, gone.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	details, err := f.service.Get(t.Context(), priya, fee.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if details.LineItem.ID != fee.ID || details.Activity.Name != "Launch film" || details.Balance.Balance.Format() != "20,000.00 USD" {
		t.Errorf("details = %+v", details)
	}
	_, deleted := f.service.Get(t.Context(), priya, gone.ID)
	_, unknown := f.service.Get(t.Context(), priya, "L-9999")
	assertError(t, deleted, errs.NotFound(lineItemGoneMessage))
	assertError(t, unknown, errs.NotFound(lineItemGoneMessage))

	err = f.db.Write(t.Context(), func(ctx context.Context) error {
		_, moveErr := f.history.MoveToBin(ctx, priya, activityKind, launchFilm)
		return moveErr
	})
	if err != nil {
		t.Fatalf("MoveToBin(activity): %v", err)
	}
	_, withActivity := f.service.Get(t.Context(), priya, fee.ID)
	_, saving := f.service.Update(t.Context(), priya, fee.ID, formOf(fee))
	assertError(t, withActivity, errs.NotFound(activityGoneMessage))
	assertError(t, saving, errs.NotFound(activityGoneMessage))
}

// The bin entry is the answer, so the web app can offer Undo.
func TestDelete_MovesToBin_FR030(t *testing.T) { //nolint:gocyclo // 12: one delete, then the bin entry, the store and the log checked in turn
	t.Parallel()
	f := newTestService(t)
	fee, film := launchFilmWithTwoItems(t, f)

	entry, err := f.service.Delete(t.Context(), priya, fee.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, gone := f.service.Delete(t.Context(), priya, fee.ID)

	want := []history.RecordRef{{Type: "activity", ID: launchFilm, Name: "Launch film"}}
	if entry.RecordType != "lineItem" || entry.RecordID != fee.ID || entry.RecordName != "Agency fee" ||
		!reflect.DeepEqual(entry.Path, want) || len(entry.Snapshots) != 1 {
		t.Errorf("bin entry = %+v", entry)
	}
	if stored := f.stored(t); len(stored) != 1 || stored[0].ID != film.ID {
		t.Errorf("stored = %+v, want only Film production", stored)
	}
	if e := f.log(t)[0]; e.Action != history.ActionDeleted || e.RecordID != fee.ID || !reflect.DeepEqual(e.Path, want) {
		t.Errorf("log entry = %+v", e)
	}
	assertError(t, gone, errs.NotFound(lineItemGoneMessage))
}

// Deleting a line item of a Submitted or Approved activity needs no confirmation and keeps its approval (D-322).
func TestDelete_InApprovalKeepsStatus_034_FR001(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, tradeAds, input("Venue hire", "GBP", "15800"))
	speaker := f.create(t, priya, socialOnAir, input("Speaker fees", "USD", "5000"))

	_, err := f.service.Delete(t.Context(), priya, venue.ID)
	_, err2 := f.service.Delete(t.Context(), priya, speaker.ID)

	if err != nil || err2 != nil || len(f.stored(t)) != 0 {
		t.Fatalf("Delete = %v, %v with %d left; want both in the recycle bin without confirmation", err, err2, len(f.stored(t)))
	}
}
