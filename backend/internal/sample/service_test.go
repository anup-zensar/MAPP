package sample

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

func TestCreateFolder_Validation_FR020(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if saved := f.createFolder(t, priya, "  Ideas  "); saved.Name != "Ideas" {
		t.Errorf("saved name = %q, want %q", saved.Name, "Ideas")
	}
	longest := strings.Repeat("é", 80) // 160 bytes: characters are counted, not bytes
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

func TestCreateNote_Validation_FR020(t *testing.T) {
	t.Parallel()
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

// Without a version of at least 1 the version check cannot work (research R5).
func TestUpdateFolder_MissingVersion_FR029(t *testing.T) {
	t.Parallel()
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

func TestUpdateFolder_VersionConflict_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	folder := f.createFolder(t, priya, "Ideas")
	f.clock.Advance(5*time.Hour + 32*time.Minute)
	changedAt := start.Add(5*time.Hour + 32*time.Minute)
	f.rename(t, daniel, folder.ID, FolderUpdate{Name: "Ideas from Daniel", Version: version(1)})
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

// A name in other letters is a change (research R5).
func TestUpdateFolder_NoChangeNoLog_FR022(t *testing.T) {
	t.Parallel()
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

func TestListFolders_SortedByNameNotesOldestFirst(t *testing.T) {
	t.Parallel()
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

func TestGoneRecords_NotFound(t *testing.T) {
	t.Parallel()
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
