package sample

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

// The changes of data model §9: create, rename, add notes, delete with the notes, and Undo with the same IDs.
func TestFolderLifecycle_OneLogEntryEach_FR022_SC004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	folder := f.createFolder(t, priya, "Ideas")
	f.rename(t, priya, folder.ID, FolderUpdate{Name: "Ideas 2", Version: version(1)})
	f.createNote(t, priya, folder.ID, "First idea")
	f.createNote(t, priya, folder.ID, "Second idea")
	before := f.folders(t)
	deleted, err := f.service.DeleteFolder(t.Context(), priya, folder.ID)
	if err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}
	if snapshots := len(f.binEntry(t, deleted.ID).Snapshots); snapshots != 3 {
		t.Errorf("the recycle-bin entry holds %d records, want the folder and its 2 notes", snapshots)
	}
	if folders := f.folders(t); len(folders) != 0 {
		t.Errorf("folders after the delete = %+v, want none", folders)
	}
	if _, err := f.history.Undo(t.Context(), priya, deleted.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}

	want := []string{
		"Priya Raman created sample folder 'Ideas' (SF-1001)",
		"Priya Raman changed sample folder 'Ideas 2' (SF-1001): Name Ideas → Ideas 2",
		"Priya Raman created sample note 'First idea' (SN-1001) in Ideas 2",
		"Priya Raman created sample note 'Second idea' (SN-1002) in Ideas 2",
		"Priya Raman deleted sample folder 'Ideas 2' (SF-1001) — with 2 sample notes",
		"Priya Raman restored sample folder 'Ideas 2' (SF-1001) — Undo, with 2 sample notes",
	}
	if got := sentences(f.entries(t)); !slices.Equal(got, want) {
		t.Errorf("log =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if after := f.folders(t); !reflect.DeepEqual(after, before) {
		t.Errorf("after Undo = %+v, want exactly %+v", after, before)
	}
}

// Undo puts a folder back exactly, so a name taken meanwhile refuses it and the folder stays in the bin.
func TestUndo_NameTakenRefused_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	folder := f.createFolder(t, priya, "Ideas")
	deleted := f.deleteFolder(t, priya, folder.ID)
	f.createFolder(t, daniel, "IDEAS")
	entriesBefore := len(f.entries(t))

	_, err := f.history.Undo(t.Context(), priya, deleted.ID)

	assertError(t, err, errs.NameTaken(
		"Can't undo — there is now another sample folder called 'IDEAS'. Restore it from the recycle bin instead."))
	if names := folderNames(f.folders(t)); !slices.Equal(names, []string{"IDEAS"}) {
		t.Errorf("folders = %q, want only IDEAS", names)
	}
	bin, err := f.history.ListBin(t.Context(), priya)
	if err != nil || len(bin) != 1 || bin[0].ID != deleted.ID {
		t.Errorf("recycle bin = %+v, %v; want only %s", bin, err, deleted.ID)
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}
}

func TestRestore_NameClashGetsSuffix_FR025(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	first := f.createFolder(t, priya, "Ideas")
	firstDeleted := f.deleteFolder(t, priya, first.ID)
	second := f.createFolder(t, priya, "Ideas")
	secondDeleted := f.deleteFolder(t, priya, second.ID)
	f.createFolder(t, priya, "ideas")
	f.clock.Advance(24 * time.Hour)
	restoredAt := start.Add(24 * time.Hour)

	got, err := f.history.Restore(t.Context(), daniel, firstDeleted.ID)

	note := "Renamed to 'Ideas (restored)' because the name was taken."
	want := history.RestoreResult{RecordType: "sampleFolder", RecordTypeLabel: "sample folder", RecordID: first.ID,
		RecordName: "Ideas (restored)", Details: note}
	if err != nil || got != want {
		t.Fatalf("Restore = %+v, %v; want %+v", got, err, want)
	}
	wantFolder := Folder{ID: first.ID, Name: "Ideas (restored)", Meta: record.NewMeta(priya, start)}
	wantFolder.Touch(daniel, restoredAt)
	if stored := f.storedFolder(t, first.ID); stored != wantFolder {
		t.Errorf("stored folder = %+v, want %+v", stored, wantFolder)
	}
	entries := f.entries(t)
	wantSentence := "Daniel Okoro restored sample folder 'Ideas (restored)' (SF-1001): Name Ideas → Ideas (restored)" +
		" — Restored from the recycle bin. " + note
	if last := sentence(entries[len(entries)-1]); last != wantSentence {
		t.Errorf("last log entry = %q\nwant %q", last, wantSentence)
	}

	got, err = f.history.Restore(t.Context(), daniel, secondDeleted.ID)

	if err != nil || got.RecordName != "Ideas (restored 2)" || got.Details != "Renamed to 'Ideas (restored 2)' because the name was taken." {
		t.Errorf("second Restore = %+v, %v; want the name Ideas (restored 2) and its note", got, err)
	}
}
