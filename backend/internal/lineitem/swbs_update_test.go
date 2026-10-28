package lineitem

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/errs"
)

const venueCode = "SIM-DE03-0001"

func (f *fixture) venueWithCode(t *testing.T) View {
	t.Helper()
	v := f.create(t, priya, socialOnAir, complete(input("Venue hire", "USD", "1000.00")))
	f.setStatus(socialOnAir, "approved")
	if err := f.receive(t, SwbsCodeReceived{LineItemID: v.ID, Code: venueCode}); err != nil {
		t.Fatalf("receive: %v", err)
	}
	details, err := f.service.Get(t.Context(), priya, v.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	return details.LineItem
}

func (f *fixture) storedByID(t *testing.T, id string) LineItem {
	t.Helper()
	for _, li := range f.stored(t) {
		if li.ID == id {
			return li
		}
	}
	t.Fatalf("line item %s is not stored", id)
	return LineItem{}
}

func assertCode(t *testing.T, li LineItem) {
	t.Helper()
	want := Swbs{SwbsCode: venueCode, SwbsRequestedAt: &received, SwbsReceivedAt: &received}
	if diff := cmp.Diff(want, li.Swbs); diff != "" {
		t.Errorf("%s SWBS (-want +got):\n%s", li.ID, diff)
	}
}

func TestUpdate_KeepsSwbsCode_030_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.venueWithCode(t)

	edit := formOf(venue)
	edit.Description = "Berlin venue"
	saved := f.update(t, priya, venue.ID, edit)
	assertCode(t, f.storedByID(t, venue.ID))
	if saved.SwbsStatus != SwbsReceived || saved.HXKey != "HX1001" {
		t.Errorf("after an edit: status %s, HX key %s", saved.SwbsStatus, saved.HXKey)
	}

	off := formOf(saved)
	off.RequestSwbs, off.Name = false, ""
	_, err := f.service.Update(t.Context(), priya, venue.ID, off)
	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "name", Message: enterNameMessage},
		errs.FieldError{Field: "requestSwbs", Message: swbsKeptOnMessage},
	))
	assertCode(t, f.storedByID(t, venue.ID))
}

func TestUpdate_RequestOffWithoutCode_030_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	stand := f.create(t, priya, socialOnAir, complete(input("Stand build", "USD", "1000.00")))
	off := formOf(stand)
	off.RequestSwbs = false
	if got := f.update(t, priya, stand.ID, off); got.SwbsStatus != SwbsNotRequested {
		t.Errorf("status = %s, want notRequested", got.SwbsStatus)
	}
}

func TestDuplicate_DropsSwbsCode_030_FR005(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.venueWithCode(t)

	dup, err := f.service.Duplicate(t.Context(), priya, venue.ID)
	if err != nil {
		t.Fatalf("Duplicate: %v", err)
	}
	if dup.SwbsCode != "" || dup.SwbsRequestedAt != nil || dup.SwbsReceivedAt != nil || !dup.RequestSwbs {
		t.Errorf("copy = %+v, want no SWBS code and the request on", dup.Swbs)
	}
	if dup.SwbsStatus != SwbsReadyToSend {
		t.Errorf("copy status = %s, want readyToSend (the activity stays approved, D-322)", dup.SwbsStatus)
	}
	assertCode(t, f.storedByID(t, venue.ID))
}

func TestCopyLineItems_DropsSwbsCode_030_FR005(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.venueWithCode(t)

	if _, err := f.service.CopyLineItems(t.Context(), priya, socialOnAir, launchFilm, 0); err != nil {
		t.Fatalf("CopyLineItems: %v", err)
	}
	for _, li := range f.stored(t) {
		if li.ActivityID == launchFilm && (li.SwbsCode != "" || li.SwbsReceivedAt != nil || !li.RequestSwbs) {
			t.Errorf("copy %s = %+v, want no SWBS code and the request on", li.ID, li.Swbs)
		}
	}
}

func TestSwbsStatus_ActivityLeavesApproval_030_FR003(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.venueWithCode(t)

	f.setStatus(socialOnAir, "declined")
	details, err := f.service.Get(t.Context(), priya, venue.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if details.LineItem.SwbsStatus != SwbsReceived || details.LineItem.SwbsCode != venueCode {
		t.Errorf("after the activity left approval: %s %q", details.LineItem.SwbsStatus, details.LineItem.SwbsCode)
	}
}

func TestMoveAndRestore_KeepSwbsCode_030_FR003(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.venueWithCode(t)

	if err := f.service.MoveToActivity(t.Context(), priya, []string{venue.ID}, launchFilm); err != nil {
		t.Fatalf("MoveToActivity: %v", err)
	}
	moved := f.storedByID(t, venue.ID)
	assertCode(t, moved)
	if status := SwbsStatusOf(moved, f.activities.facts[launchFilm]); status != SwbsReceived {
		t.Errorf("moved to a created activity: status %s, want received", status)
	}

	entry := f.deleteNow(t, venue.ID)
	f.undo(t, entry.ID)
	assertCode(t, f.storedByID(t, venue.ID))
}
