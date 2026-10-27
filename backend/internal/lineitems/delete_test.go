package lineitems

import (
	"context"
	"reflect"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// The details are the line item with its activity and the balance including it (FR-026). A line item that is
// gone says why (D-154): deleted itself, or deleted with its activity.
func TestGet_GoneMessages_007P15(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee, _ := launchFilmWithTwoItems(t, f)
	gone := f.create(t, priya, launchFilm, input("Gone", "USD", "1.00"))
	if _, err := f.service.Delete(t.Context(), priya, gone.ID, false); err != nil {
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
		_, err := f.history.MoveToBin(ctx, priya, activityKind, launchFilm)
		return err
	})
	if err != nil {
		t.Fatalf("MoveToBin(activity): %v", err)
	}
	_, withActivity := f.service.Get(t.Context(), priya, fee.ID)
	_, saving := f.service.Update(t.Context(), priya, fee.ID, formOf(fee))
	assertError(t, withActivity, errs.NotFound(activityGoneMessage))
	assertError(t, saving, errs.NotFound(activityGoneMessage))
}

// Delete moves the line item to the recycle bin, with its activity as its path, and logs "deleted"; the bin
// entry is the answer, so the web app can offer Undo (FR-030).
func TestDelete_MovesToBin_FR030(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee, film := launchFilmWithTwoItems(t, f)

	entry, err := f.service.Delete(t.Context(), priya, fee.ID, false)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, gone := f.service.Delete(t.Context(), priya, fee.ID, false)

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
	if len(f.activities.restarts) != 0 {
		t.Errorf("a delete on a Created activity restarted its approval")
	}
}

// Deleting a line item of a Submitted or Approved activity needs confirmation and sends the activity back to
// Created (FR-031).
func TestDelete_RestartNeedsConfirmation_FR031(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, tradeAds, confirmed(input("Venue hire", "GBP", "15800")))
	facts := f.activities.facts[tradeAds]
	facts.Status = "submitted"
	f.activities.facts[tradeAds] = facts
	f.activities.restarts = nil

	_, refused := f.service.Delete(t.Context(), priya, venue.ID, false)
	assertError(t, refused, errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage}))
	if len(f.stored(t)) != 1 {
		t.Fatal("an unconfirmed delete removed the line item")
	}

	if _, err := f.service.Delete(t.Context(), priya, venue.ID, true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	want := []restartCall{{activityID: tradeAds, reasons: []string{"Line item 'Venue hire' (L-1001) deleted: 15,800.00 GBP"}}}
	if !reflect.DeepEqual(f.activities.restarts, want) || len(f.stored(t)) != 0 {
		t.Errorf("restarts = %+v, want %+v", f.activities.restarts, want)
	}
}
