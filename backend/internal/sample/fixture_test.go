package sample

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
)

var (
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
)

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// fixture registers the sample kinds with history as internal/app does.
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
	hist := history.NewService(db, history.NewJSONStore(), generator, clock.Now)
	service := NewService(history.Tools{Tx: db, IDs: generator, Log: hist, Now: clock.Now}, NewJSONStore())
	for _, kind := range service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	return &fixture{service: service, history: hist, db: db, clock: clock}
}

func (f *fixture) createFolder(t *testing.T, a actor.Actor, name string) FolderView {
	t.Helper()
	folder, err := f.service.CreateFolder(t.Context(), a, FolderInput{Name: name})
	if err != nil {
		t.Fatalf("CreateFolder(%q): %v", name, err)
	}
	return folder
}

func (f *fixture) rename(t *testing.T, a actor.Actor, id string, to FolderUpdate) FolderView {
	t.Helper()
	folder, err := f.service.UpdateFolder(t.Context(), a, id, to)
	if err != nil {
		t.Fatalf("UpdateFolder(%s, %q): %v", id, to.Name, err)
	}
	return folder
}

func (f *fixture) deleteFolder(t *testing.T, a actor.Actor, id string) history.BinEntry {
	t.Helper()
	entry, err := f.service.DeleteFolder(t.Context(), a, id)
	if err != nil {
		t.Fatalf("DeleteFolder(%s): %v", id, err)
	}
	return entry
}

func (f *fixture) createNote(t *testing.T, a actor.Actor, folderID, title string) Note {
	t.Helper()
	note, err := f.service.CreateNote(t.Context(), a, folderID, NoteInput{Title: title})
	if err != nil {
		t.Fatalf("CreateNote(%s, %q): %v", folderID, title, err)
	}
	return note
}

func (f *fixture) folders(t *testing.T) []FolderView {
	t.Helper()
	folders, err := f.service.ListFolders(t.Context(), priya)
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	return folders
}

func (f *fixture) storedFolder(t *testing.T, id string) Folder {
	t.Helper()
	var folder Folder
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		folder, found, err = NewJSONStore().GetFolder(ctx, id)
		return err
	})
	if err != nil || !found {
		t.Fatalf("GetFolder(%s) = found %v, %v", id, found, err)
	}
	return folder
}

func (f *fixture) entries(t *testing.T) []history.Entry {
	t.Helper()
	var entries []history.Entry
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entries, err = history.NewJSONStore().Entries(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return entries
}

func (f *fixture) binEntry(t *testing.T, id string) history.BinEntry {
	t.Helper()
	var entry history.BinEntry
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entry, found, err = history.NewJSONStore().GetBinEntry(ctx, id)
		return err
	})
	if err != nil || !found {
		t.Fatalf("GetBinEntry(%s) = found %v, %v", id, found, err)
	}
	return entry
}

func sentences(entries []history.Entry) []string {
	list := make([]string, 0, len(entries))
	for _, e := range entries {
		list = append(list, sentence(e))
	}
	return list
}

// sentence writes a log entry the way feature 011 shows it.
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

func version(v int) *int {
	return &v
}

func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v, want %#v", err, want)
	}
}

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time { return c.now }

func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }
