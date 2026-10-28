package lineitem

import (
	"reflect"
	"strings"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

// Duplicate copies every value, the spread included, into the same activity with a new ID and the name
// "<name> (copy)", " (copy 2)", … shortened to fit 120 characters, and logs "copied" (FR-029).
func TestDuplicate_CopiesEverything_FR029(t *testing.T) { //nolint:gocyclo // 11: three copies, then their names, values, bookkeeping and log entry
	t.Parallel()
	f := newTestService(t)
	in := complete(input("Agency fee", "EUR", "1000.00"))
	in.Description, in.ContractedAmount, in.RequestSwbs = "Retainer", text("900"), false
	in.Spread = SpreadInput{Method: SpreadCustom, Months: []SpreadMonthInput{{Month: "2026-02", Amount: "400"}, {Month: "2026-04", Amount: "600"}}}
	fee := f.create(t, priya, launchFilm, in)
	long := f.create(t, priya, launchFilm, input(strings.Repeat("n", 120), "USD", "1.00"))

	first, err := f.service.Duplicate(t.Context(), daniel, fee.ID)
	if err != nil {
		t.Fatalf("Duplicate: %v", err)
	}
	second, _ := f.service.Duplicate(t.Context(), daniel, fee.ID)
	longCopy, _ := f.service.Duplicate(t.Context(), daniel, long.ID)

	if first.ID != "L-1003" || first.Name != "Agency fee (copy)" || second.Name != "Agency fee (copy 2)" ||
		longCopy.Name != strings.Repeat("n", 113)+" (copy)" {
		t.Errorf("copies = %s %q, %q, %q", first.ID, first.Name, second.Name, longCopy.Name)
	}
	copied, original := first, fee
	copied.ID, copied.Name, copied.HXKey, copied.Meta = original.ID, original.Name, original.HXKey, original.Meta
	if !reflect.DeepEqual(copied, original) {
		t.Errorf("copy = %+v\nwant the values of %+v", copied, original)
	}
	if first.Meta != record.NewMeta(daniel, start) {
		t.Errorf("copy bookkeeping = %+v, want new, by Daniel", first.Meta)
	}
	entry := f.log(t)[2]
	if entry.Action != history.ActionCopied || entry.RecordID != first.ID || entry.Details != "Copied from 'Agency fee' (L-1001)" {
		t.Errorf("log entry = %+v", entry)
	}
}

// Duplicating a line item of a Submitted or Approved activity needs no confirmation and keeps the activity's
// approval (D-322); an unknown line item is not found.
func TestDuplicate_InApprovalKeepsStatus_034_FR001(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, input("Venue hire", "GBP", "15800"))
	printRun := f.create(t, priya, tradeAds, input("Print run", "USD", "2000"))

	dup, err := f.service.Duplicate(t.Context(), priya, venue.ID)
	printCopy, err2 := f.service.Duplicate(t.Context(), priya, printRun.ID)
	_, gone := f.service.Duplicate(t.Context(), priya, "L-9999")

	if err != nil || err2 != nil || dup.Name != "Venue hire (copy)" || printCopy.Name != "Print run (copy)" {
		t.Fatalf("Duplicate = %q %v, %q %v; want both copies made without confirmation", dup.Name, err, printCopy.Name, err2)
	}
	if dup.SwbsStatus != SwbsReadyToSend || printCopy.SwbsStatus != SwbsWaitingForApproval {
		t.Errorf("SWBS statuses %s, %s; want readyToSend (still Approved) and waitingForApproval (still Submitted)",
			dup.SwbsStatus, printCopy.SwbsStatus)
	}
	assertError(t, gone, errs.NotFound(lineItemGoneMessage))
}
