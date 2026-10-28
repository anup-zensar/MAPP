package lineitem

import (
	"context"
	"reflect"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

func (f *fixture) undo(t *testing.T, entryID string) history.RestoreResult {
	t.Helper()
	result, err := f.history.Undo(t.Context(), priya, entryID)
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}
	return result
}

func (f *fixture) deleteNow(t *testing.T, id string) history.BinEntry {
	t.Helper()
	entry, err := f.service.Delete(t.Context(), priya, id)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	return entry
}

func (f *fixture) setStatus(id, status string) {
	facts := f.activities.facts[id]
	facts.Status = status
	f.activities.facts[id] = facts
}

func TestUndo_RestoresExactly_FR030(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, complete(input("Agency fee", "EUR", "46000.00")))
	before := f.stored(t)

	result := f.undo(t, f.deleteNow(t, fee.ID).ID)

	if !reflect.DeepEqual(f.stored(t), before) {
		t.Errorf("after Undo: %+v\nwant %+v", f.stored(t), before)
	}
	if result.RecordID != fee.ID || result.RecordName != "Agency fee" || result.Details != "" {
		t.Errorf("result = %+v", result)
	}
}

// Deleting and undoing a line item of an Approved activity leave it Approved, with no note about approval (D-322).
func TestUndo_InApprovalKeepsStatus_034_FR002(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, input("Venue hire", "GBP", "15800"))
	before := f.stored(t)

	result := f.undo(t, f.deleteNow(t, venue.ID).ID)

	if !reflect.DeepEqual(f.stored(t), before) || result.Details != "" {
		t.Errorf("after Undo: %+v with details %q; want the line item back exactly, no note", f.stored(t), result.Details)
	}
}

// A restore into an activity approved meanwhile keeps it Approved, with no approval note (D-322, replacing D-147).
func TestRestore_IntoApprovedKeepsStatus_034_FR002(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, input("Agency fee", "USD", "50000.00"))
	entry := f.deleteNow(t, fee.ID)
	f.setStatus(launchFilm, "approved")

	result, err := f.history.Restore(t.Context(), priya, entry.ID)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if result.Details != "" || len(f.stored(t)) != 1 {
		t.Errorf("details %q with %d stored; want the line item back and no note", result.Details, len(f.stored(t)))
	}
	if e := f.log(t)[0]; e.Action != history.ActionRestored || e.Details != "Restored from the recycle bin" {
		t.Errorf("log entry = %+v, want restored without a note", e)
	}
}

// Undo must put the line item back exactly, so it is refused when another line item of the activity has
// meanwhile taken the name; the line item stays in the recycle bin (D-037).
func TestUndo_NameTaken_D037(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, input("Agency fee", "USD", "1.00"))
	entry := f.deleteNow(t, fee.ID)
	f.create(t, priya, launchFilm, input("agency FEE", "USD", "2.00"))

	_, err := f.history.Undo(t.Context(), priya, entry.ID)

	assertError(t, err, errs.NameTaken("Can't undo — there is now another line item called 'agency FEE'. Restore it from the recycle bin instead."))
	if bin, _ := f.history.ListBin(t.Context(), priya); len(bin) != 1 || bin[0].ID != entry.ID {
		t.Errorf("bin = %+v, want the line item still there", bin)
	}
}

// A Restore renames a line item whose name is taken, with a note and a new version.
func TestRestore_NameTakenSuffix_FR030(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, input("Agency fee", "USD", "1.00"))
	entry := f.deleteNow(t, fee.ID)
	f.create(t, priya, launchFilm, input("Agency fee", "USD", "2.00"))
	f.create(t, priya, launchFilm, input("Agency fee (restored)", "USD", "3.00"))

	result, err := f.history.Restore(t.Context(), priya, entry.ID)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if result.RecordName != "Agency fee (restored 2)" || result.Details != "Renamed to 'Agency fee (restored 2)' because the name was taken." {
		t.Errorf("result = %+v", result)
	}
	stored := f.stored(t)[0]
	if stored.ID != fee.ID || stored.Name != "Agency fee (restored 2)" || stored.Version != 2 {
		t.Errorf("stored = %+v", stored)
	}
}

// Deleting an activity takes its line items into the same bin entry; undoing it brings them back without
// restarting its approval.
func TestDeleteActivity_TakesLineItems(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.create(t, priya, socialOnAir, input("Venue hire", "GBP", "15800"))
	f.create(t, priya, socialOnAir, input("Speaker fees", "USD", "5000"))
	f.setStatus(socialOnAir, "approved")

	var entry history.BinEntry
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		entry, err = f.history.MoveToBin(ctx, priya, activityKind, socialOnAir)
		return err
	})
	if err != nil {
		t.Fatalf("MoveToBin: %v", err)
	}
	if len(entry.Snapshots) != 3 || len(f.stored(t)) != 0 || f.log(t)[0].Details != "with 2 line items" {
		t.Fatalf("bin entry %+v, %d stored, details %q", entry.Contents, len(f.stored(t)), f.log(t)[0].Details)
	}

	f.undo(t, entry.ID)

	if len(f.stored(t)) != 2 || f.activities.facts[socialOnAir].Status != "approved" {
		t.Errorf("%d stored, activity %s; want both back, still approved", len(f.stored(t)), f.activities.facts[socialOnAir].Status)
	}
}

// The log reads like the screens: names, formatted amounts, the spread in words, labels of Shell attributes.
func TestDescribe_LogTexts_FR027(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tests := []struct {
		field       string
		value       any
		label, text string
	}{
		{"ownerId", "U-1002", "Owner", "Priya Raman"},
		{"ownerId", "", "Owner", ""},
		{"plannedCost", "46000.00", "Planned cost", "46,000.00"},
		{"contractedAmount", nil, "Contracted amount", ""},
		{"spread", map[string]any{"method": "firstMonth", "months": []any{}}, "Spread over time", "All in first month"},
		{"spread", map[string]any{"method": "custom", "months": []any{
			map[string]any{"month": "2026-02", "amount": "2000.00"}, map[string]any{"month": "2026-03", "amount": "3333.33"},
		}}, "Spread over time", "Custom: Feb 2026 2,000.00 · Mar 2026 3,333.33"},
		{"requestSwbs", true, "Request SWBS code from GSAP", ""},
		{"financeBudgetYear", nil, "Finance budget year", ""},
		{"financeBudgetItemId", europeDigital, "Finance budget item", "2026 Lubricants Budget › Europe – Digital"},
		{"attributes." + glCode, "SO-1201", "GL code", "7250080 – Advertising digital"},
		{"name", "Agency fee", "", ""},
	}
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		for _, tt := range tests {
			label, text, err := f.service.describe(ctx, tt.field, tt.value)
			if err != nil || label != tt.label || text != tt.text {
				t.Errorf("describe(%s, %v) = %q, %q, %v; want %q, %q", tt.field, tt.value, label, text, err, tt.label, tt.text)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The kind keeps the stored line item as its snapshot, with the activity as its parent.
func TestKind_SnapshotAndChildren(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, input("Agency fee", "USD", "1.00"))
	f.create(t, priya, socialOnAir, input("Elsewhere", "USD", "1.00"))
	kind := f.service.Kinds()[0]

	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		snapshot, found, err := kind.Snapshot(ctx, fee.ID)
		if err != nil || !found || snapshot.Name != "Agency fee" || snapshot.ParentID != launchFilm {
			t.Errorf("snapshot = %+v, %v, %v", snapshot, found, err)
		}
		children, err := kind.ChildIDs(ctx, launchFilm)
		if err != nil || !reflect.DeepEqual(children, []string{fee.ID}) {
			t.Errorf("children = %v, %v", children, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if kind.Name != "lineItem" || kind.Parent != "activity" || fee.Meta != record.NewMeta(priya, start) {
		t.Errorf("kind %s under %s", kind.Name, kind.Parent)
	}
}
