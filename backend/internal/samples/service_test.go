package samples

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
)

// The people of these tests.
var (
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
)

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// A sample folder needs a name of at most 80 characters that no other sample folder has, ignoring letter
// case and surrounding spaces; the name is saved without its surrounding spaces (data model §8, FR-020).
func TestCreateFolder_Validation_FR020(t *testing.T) {
	f := newFixture(t)
	if saved := f.createFolder(t, priya, "  Ideas  "); saved.Name != "Ideas" {
		t.Errorf("saved name = %q, want %q", saved.Name, "Ideas")
	}
	longest := strings.Repeat("é", 80) // 80 characters in 160 bytes: characters are counted, not bytes
	f.createFolder(t, priya, longest)
	tests := []struct {
		name, input, message string
	}{
		{"empty", "", "Enter a name."},
		{"spaces only", "   ", "Enter a name."},
		{"81 characters", strings.Repeat("a", 81), "Use 80 characters or fewer."},
		{"taken, in other letters and with spaces", " IDEAS ", "There is already a sample folder called 'Ideas'. Choose another name."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.CreateFolder(t.Context(), priya, FolderInput{Name: tt.input})

			assertError(t, err, errs.Validation(errs.FieldError{Field: "name", Message: tt.message}))
		})
	}
	if names := folderNames(f.folders(t)); !slices.Equal(names, []string{"Ideas", longest}) {
		t.Errorf("folders = %q, want only the two valid ones", names)
	}
}

// A sample note needs a title of at most 120 characters, saved without its surrounding spaces, and a
// folder that still exists (data model §9, FR-020).
func TestCreateNote_Validation_FR020(t *testing.T) {
	f := newFixture(t)
	folder := f.createFolder(t, priya, "Ideas")
	note := f.createNote(t, priya, folder.ID, "  First idea  ")
	want := Note{ID: "SN-1001", FolderID: folder.ID, Title: "First idea", Meta: record.NewMeta(priya, start)}
	if note != want {
		t.Errorf("note = %+v, want %+v", note, want)
	}
	f.createNote(t, priya, folder.ID, strings.Repeat("ü", 120))
	tests := []struct {
		name, title, message string
	}{
		{"empty", "", "Enter a title."},
		{"spaces only", "  ", "Enter a title."},
		{"121 characters", strings.Repeat("a", 121), "Use 120 characters or fewer."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.CreateNote(t.Context(), priya, folder.ID, NoteInput{Title: tt.title})

			assertError(t, err, errs.Validation(errs.FieldError{Field: "title", Message: tt.message}))
		})
	}
	_, err := f.service.CreateNote(t.Context(), priya, "SF-9999", NoteInput{Title: "Lost idea"})
	assertError(t, err, errs.NotFound("This sample folder no longer exists — someone may have deleted it."))
	if notes := f.folders(t)[0].Notes; len(notes) != 2 {
		t.Errorf("%d notes, want only the two valid ones", len(notes))
	}
}

// Every update sends the version it started from. Without one, or with one below 1, the version check
// cannot work, so the save is refused next to the field "version", together with every other problem
// (FR-029, FR-020, research R5).
func TestUpdateFolder_MissingVersion_FR029(t *testing.T) {
	f := newFixture(t)
	folder := f.createFolder(t, priya, "Ideas")
	zero := 0
	reload := errs.FieldError{Field: "version", Message: "Reload the page and try again."}
	tests := []struct {
		name   string
		update FolderUpdate
		want   *errs.Error
	}{
		{"no version", FolderUpdate{Name: "Ideas 2"}, errs.Validation(reload)},
		{"version 0", FolderUpdate{Name: "Ideas 2", Version: &zero}, errs.Validation(reload)},
		{"no version and no name", FolderUpdate{Name: " "},
			errs.Validation(errs.FieldError{Field: "name", Message: "Enter a name."}, reload)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.UpdateFolder(t.Context(), priya, folder.ID, tt.update)

			assertError(t, err, tt.want)
		})
	}
	if stored := f.storedFolder(t, folder.ID); stored != folder.Folder {
		t.Errorf("stored folder = %+v, want it unchanged: %+v", stored, folder.Folder)
	}
}

// A save based on an out-of-date copy is refused, naming who changed the folder and when, and nothing is
// overwritten or logged (FR-029). Anyone may change anyone's folder: there are no permission checks
// (FR-008).
func TestUpdateFolder_VersionConflict_FR029(t *testing.T) {
	f := newFixture(t)
	folder := f.createFolder(t, priya, "Ideas") // version 1
	f.clock.Advance(5*time.Hour + 32*time.Minute)
	changedAt := start.Add(5*time.Hour + 32*time.Minute)   // 14:32
	f.rename(t, daniel, folder.ID, "Ideas from Daniel", 1) // Daniel saves first: version 2
	entriesBefore := len(f.entries(t))

	_, err := f.service.UpdateFolder(t.Context(), priya, folder.ID, FolderUpdate{Name: "Ideas from Priya", Version: version(1)})

	assertError(t, err, errs.VersionConflict("sample folder", "Daniel Okoro", changedAt))
	if stored := f.storedFolder(t, folder.ID); stored.Name != "Ideas from Daniel" || stored.Version != 2 {
		t.Errorf("stored folder = %+v, want Daniel's name at version 2", stored)
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}
}

// A save that changes nothing is not a change: no new version and no log entry. A name in other letters is
// a change (FR-022, research R5).
func TestUpdateFolder_NoChangeNoLog_FR022(t *testing.T) {
	f := newFixture(t)
	folder := f.createFolder(t, priya, "Ideas")

	same, err := f.service.UpdateFolder(t.Context(), priya, folder.ID, FolderUpdate{Name: "  Ideas ", Version: version(1)})

	if err != nil || !reflect.DeepEqual(same, folder) {
		t.Errorf("UpdateFolder with the same name = %+v, %v; want the folder unchanged: %+v", same, err, folder)
	}
	if got := len(f.entries(t)); got != 1 {
		t.Errorf("%d log entries, want only the one of the create", got)
	}

	lower, err := f.service.UpdateFolder(t.Context(), priya, folder.ID, FolderUpdate{Name: "ideas", Version: version(1)})

	if err != nil || lower.Name != "ideas" || lower.Version != 2 {
		t.Errorf("UpdateFolder in lower case = %+v, %v; want the name ideas at version 2", lower, err)
	}
	entries := f.entries(t)
	if want := "Priya Raman changed sample folder 'ideas' (SF-1001): Name Ideas → ideas"; len(entries) != 2 || sentence(entries[1]) != want {
		t.Errorf("log = %q, want a second entry %q", sentences(entries), want)
	}
}

// Every change of a sample folder or note writes exactly one log entry naming the person (FR-022, SC-004),
// as data model §9 lists: create, rename, add notes, delete with the notes, and Undo, which brings
// everything back with the same IDs.
func TestFolderLifecycle_OneLogEntryEach_FR022_SC004(t *testing.T) {
	f := newFixture(t)

	folder := f.createFolder(t, priya, "Ideas")
	f.rename(t, priya, folder.ID, "Ideas 2", 1)
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

// Undo puts a folder back exactly, so when someone has meanwhile created a folder with its name, Undo is
// refused naming the clash, and nothing changes: the folder stays in the recycle bin, from where Restore can
// bring it back (spec edge case, D-037).
func TestUndo_NameTakenRefused_FR026(t *testing.T) {
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

// Restore from the recycle bin brings a folder back even when its name is taken: it becomes "<name>
// (restored)", then "<name> (restored 2)" and so on, gets a new version, and the log and the answer say why
// (FR-025, feature 011 FR-022).
func TestRestore_NameClashGetsSuffix_FR025(t *testing.T) {
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

// The developer page lists the folders by name, ignoring letter case, each with its notes oldest first; a
// folder without notes has an empty list.
func TestListFolders_SortedByNameNotesOldestFirst(t *testing.T) {
	f := newFixture(t)
	banana := f.createFolder(t, priya, "banana")
	f.createFolder(t, priya, "Apple")
	f.createFolder(t, priya, "cherry")
	f.createNote(t, priya, banana.ID, "Ripe")
	f.createNote(t, priya, banana.ID, "Peeled")

	got := f.folders(t)

	if names := folderNames(got); !slices.Equal(names, []string{"Apple", "banana", "cherry"}) {
		t.Fatalf("folders = %q, want Apple, banana, cherry", names)
	}
	var titles []string
	for _, n := range got[1].Notes {
		titles = append(titles, n.Title)
	}
	if !slices.Equal(titles, []string{"Ripe", "Peeled"}) {
		t.Errorf("banana's notes = %q, want Ripe then Peeled", titles)
	}
	if got[0].Notes == nil || len(got[0].Notes) != 0 {
		t.Errorf("Apple's notes = %#v, want an empty list", got[0].Notes)
	}
}

// A folder or note that no longer exists answers not_found with the page's message, whatever is asked of
// it (plan, User-visible messages).
func TestGoneRecords_NotFound(t *testing.T) {
	f := newFixture(t)
	folderGone := errs.NotFound("This sample folder no longer exists — someone may have deleted it.")
	noteGone := errs.NotFound("This sample note no longer exists — someone may have deleted it.")

	_, updateErr := f.service.UpdateFolder(t.Context(), priya, "SF-9999", FolderUpdate{Name: "Ideas", Version: version(1)})
	_, deleteFolderErr := f.service.DeleteFolder(t.Context(), priya, "SF-9999")
	_, deleteNoteErr := f.service.DeleteNote(t.Context(), priya, "SN-9999")

	assertError(t, updateErr, folderGone)
	assertError(t, deleteFolderErr, folderGone)
	assertError(t, deleteNoteErr, noteGone)
}

// fixture is the samples service and the history service it logs to, on a temporary data folder, with the
// sample kinds registered as internal/app registers them, and a clock that moves only when the test says
// so.
type fixture struct {
	service *Service
	history *history.Service
	db      *store.DB
	clock   *testClock
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	clock := &testClock{now: start}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	hist := history.NewService(db, history.NewJSONRepository(), generator, clock.Now)
	service := NewService(db, NewJSONRepository(), generator, hist, clock.Now)
	for _, kind := range service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	return &fixture{service: service, history: hist, db: db, clock: clock}
}

// createFolder creates a folder as a and stops the test when that fails.
func (f *fixture) createFolder(t *testing.T, a actor.Actor, name string) FolderView {
	t.Helper()
	folder, err := f.service.CreateFolder(t.Context(), a, FolderInput{Name: name})
	if err != nil {
		t.Fatalf("CreateFolder(%q): %v", name, err)
	}
	return folder
}

// rename renames a folder as a, from the version it started at, and stops the test when that fails.
func (f *fixture) rename(t *testing.T, a actor.Actor, id, name string, from int) FolderView {
	t.Helper()
	folder, err := f.service.UpdateFolder(t.Context(), a, id, FolderUpdate{Name: name, Version: version(from)})
	if err != nil {
		t.Fatalf("UpdateFolder(%s, %q): %v", id, name, err)
	}
	return folder
}

// deleteFolder deletes a folder as a and stops the test when that fails.
func (f *fixture) deleteFolder(t *testing.T, a actor.Actor, id string) history.BinEntry {
	t.Helper()
	entry, err := f.service.DeleteFolder(t.Context(), a, id)
	if err != nil {
		t.Fatalf("DeleteFolder(%s): %v", id, err)
	}
	return entry
}

// createNote adds a note as a and stops the test when that fails.
func (f *fixture) createNote(t *testing.T, a actor.Actor, folderID, title string) Note {
	t.Helper()
	note, err := f.service.CreateNote(t.Context(), a, folderID, NoteInput{Title: title})
	if err != nil {
		t.Fatalf("CreateNote(%s, %q): %v", folderID, title, err)
	}
	return note
}

// folders lists the folders as the developer page does.
func (f *fixture) folders(t *testing.T) []FolderView {
	t.Helper()
	folders, err := f.service.ListFolders(t.Context(), priya)
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	return folders
}

// storedFolder returns the stored folder id and stops the test when there is none.
func (f *fixture) storedFolder(t *testing.T, id string) Folder {
	t.Helper()
	var folder Folder
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		folder, found, err = NewJSONRepository().GetFolder(ctx, id)
		return err
	})
	if err != nil || !found {
		t.Fatalf("GetFolder(%s) = found %v, %v", id, found, err)
	}
	return folder
}

// entries returns the whole activity log, oldest first.
func (f *fixture) entries(t *testing.T) []history.Entry {
	t.Helper()
	var entries []history.Entry
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entries, err = history.NewJSONRepository().Entries(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return entries
}

// binEntry returns the stored recycle-bin entry id, with its snapshots.
func (f *fixture) binEntry(t *testing.T, id string) history.BinEntry {
	t.Helper()
	var entry history.BinEntry
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entry, found, err = history.NewJSONRepository().GetBinEntry(ctx, id)
		return err
	})
	if err != nil || !found {
		t.Fatalf("GetBinEntry(%s) = found %v, %v", id, found, err)
	}
	return entry
}

// sentences writes each log entry as a sentence: see sentence.
func sentences(entries []history.Entry) []string {
	list := make([]string, 0, len(entries))
	for _, e := range entries {
		list = append(list, sentence(e))
	}
	return list
}

// sentence writes a log entry the way feature 011 will show it: "<person> <verb> <record type> '<name>'
// (<id>)", then where the record was, its changes and the details.
func sentence(e history.Entry) string {
	var text strings.Builder
	fmt.Fprintf(&text, "%s %s %s '%s' (%s)", e.Actor.Name, e.Verb, e.RecordTypeLabel, e.RecordName, e.RecordID)
	for i, place := range e.Path {
		separator := " › "
		if i == 0 {
			separator = " in "
		}
		text.WriteString(separator + place.Name)
	}
	for _, c := range e.Changes {
		fmt.Fprintf(&text, ": %s %s → %s", c.Label, c.OldText, c.NewText)
	}
	if e.Details != "" {
		text.WriteString(" — " + e.Details)
	}
	return text.String()
}

func folderNames(folders []FolderView) []string {
	names := make([]string, 0, len(folders))
	for _, folder := range folders {
		names = append(names, folder.Name)
	}
	return names
}

// version returns a pointer to v, for FolderUpdate.Version.
func version(v int) *int {
	return &v
}

// assertError checks that err is exactly want: the same status, code, message and details.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v, want %#v", err, want)
	}
}

// testClock is a clock that shows a fixed time until Advance moves it.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }
