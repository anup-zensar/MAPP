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

	first, err := f.service.Duplicate(t.Context(), daniel, fee.ID, DuplicateInput{})
	if err != nil {
		t.Fatalf("Duplicate: %v", err)
	}
	second, _ := f.service.Duplicate(t.Context(), daniel, fee.ID, DuplicateInput{})
	longCopy, _ := f.service.Duplicate(t.Context(), daniel, long.ID, DuplicateInput{})

	if first.ID != "L-1003" || first.Name != "Agency fee (copy)" || second.Name != "Agency fee (copy 2)" ||
		longCopy.Name != strings.Repeat("n", 113)+" (copy)" {
		t.Errorf("copies = %s %q, %q, %q", first.ID, first.Name, second.Name, longCopy.Name)
	}
	copied, original := first, fee
	copied.ID, copied.Name, copied.Meta = original.ID, original.Name, original.Meta
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

// Duplicating counts as adding a line item: on a Submitted or Approved activity only with confirmation,
// which sends it back to Created (FR-031, US4 scenario 3).
func TestDuplicate_RestartNeedsConfirmation_FR031(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, confirmed(input("Venue hire", "GBP", "15800")))
	facts := f.activities.facts[socialOnAir]
	facts.Status = "approved"
	f.activities.facts[socialOnAir] = facts
	f.activities.restarts = nil

	_, refused := f.service.Duplicate(t.Context(), priya, venue.ID, DuplicateInput{})
	dup, err := f.service.Duplicate(t.Context(), priya, venue.ID, DuplicateInput{RestartApproval: true})
	_, gone := f.service.Duplicate(t.Context(), priya, "L-9999", DuplicateInput{RestartApproval: true})

	assertError(t, refused, errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage}))
	assertError(t, gone, errs.NotFound(lineItemGoneMessage))
	if err != nil || dup.ID != "L-1002" {
		t.Fatalf("Duplicate = %+v, %v", dup, err)
	}
	want := []restartCall{{activityID: socialOnAir, reasons: []string{"Line item 'Venue hire (copy)' (L-1002) added: 15,800.00 GBP"}}}
	if !reflect.DeepEqual(f.activities.restarts, want) {
		t.Errorf("restarts = %+v, want %+v", f.activities.restarts, want)
	}
}
