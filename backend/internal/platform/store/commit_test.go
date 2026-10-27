package store

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"mapp/internal/platform/errs"
)

// The counters table is saved first, then the other tables by name (research R3).
func TestCommit_SavesCountersFirstThenByName(t *testing.T) {
	t.Parallel()
	db, _ := openTempStore(t)
	var saved []string
	db.rename = func(from, to string) error {
		saved = append(saved, filepath.Base(to))
		return renameWithRetry(from, to)
	}

	err := db.Write(t.Context(), func(ctx context.Context) error {
		for _, table := range []string{"zebra", "counters", "apple"} {
			if err := Put(ctx, table, "x", note{ID: "x"}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if want := []string{"counters.json", "apple.json", "zebra.json"}; !slices.Equal(saved, want) {
		t.Errorf("files saved in the order %q, want %q", saved, want)
	}
}

// When replacing a data file fails, the Write changes nothing in memory or on disk, and every later
// Write is refused with a message for the user; reads keep working (research R3).
func TestCommit_RenameFailure_NothingChangesAndLaterWritesAreRefused(t *testing.T) {
	t.Parallel()
	db, dir := openTempStore(t)
	putNote(t, db, note{ID: "N-1", Title: "Old"})
	diskFailure := errors.New("the file is being used by another process")
	db.rename = func(_, _ string) error { return diskFailure }

	err := db.Write(t.Context(), func(ctx context.Context) error {
		return Put(ctx, "notes", "N-1", note{ID: "N-1", Title: "New"})
	})
	if !errors.Is(err, ErrFailed) || !errors.Is(err, diskFailure) {
		t.Fatalf("Write = %v, want ErrFailed caused by the disk failure", err)
	}

	if got, _ := getNote(t, db); got.Title != "Old" {
		t.Errorf("in memory: title %q, want Old", got.Title)
	}
	if got, _ := getNote(t, mustOpen(t, dir)); got.Title != "Old" {
		t.Errorf("on disk: title %q, want Old", got.Title)
	}

	db.rename = renameWithRetry // the disk works again, but the store must stay stopped until a restart
	err = db.Write(t.Context(), func(ctx context.Context) error {
		return Put(ctx, "notes", "N-2", note{ID: "N-2"})
	})
	var apiErr *errs.Error
	if !errors.Is(err, ErrFailed) || !errors.As(err, &apiErr) {
		t.Fatalf("later Write = %v, want ErrFailed", err)
	}
	if want := "MAPP could not save its data folder — restart the API."; apiErr.Message != want {
		t.Errorf("message = %q, want %q", apiErr.Message, want)
	}
}

// When a temporary file cannot be written, the temporary files already written are deleted, nothing
// changes, and the store keeps working (research R3).
func TestCommit_TempFileFailure_NothingChangesAndTheStoreKeepsWorking(t *testing.T) { //nolint:gocyclo // 10: a failed save, then the leftover file, memory, disk and a later save checked in turn
	t.Parallel()
	db, dir := openTempStore(t)
	putNote(t, db, note{ID: "N-1", Title: "Old"})
	blocker := filepath.Join(dir, "zebra.json.tmp") // a folder where the temporary file must go
	if err := os.Mkdir(blocker, 0o750); err != nil {
		t.Fatal(err)
	}

	err := db.Write(t.Context(), func(ctx context.Context) error {
		if err := Put(ctx, "notes", "N-1", note{ID: "N-1", Title: "New"}); err != nil {
			return err
		}
		return Put(ctx, "zebra", "z", note{ID: "z"})
	})
	if err == nil || errors.Is(err, ErrFailed) {
		t.Fatalf("Write = %v, want an error that does not stop the store", err)
	}

	if _, statErr := os.Stat(filepath.Join(dir, "notes.json.tmp")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("notes.json.tmp was left behind: %v", statErr)
	}
	if got, _ := getNote(t, db); got.Title != "Old" {
		t.Errorf("in memory: title %q, want Old", got.Title)
	}
	if got, _ := getNote(t, mustOpen(t, dir)); got.Title != "Old" {
		t.Errorf("on disk: title %q, want Old", got.Title)
	}

	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	putNote(t, db, note{ID: "N-1", Title: "New"})
	if got, _ := getNote(t, db); got.Title != "New" {
		t.Errorf("after the problem was gone: title %q, want New", got.Title)
	}
}

// On Windows, antivirus or search indexing can hold a data file open for a moment, which makes replacing
// it fail; the store tries again for up to 2 seconds (research R3). Here the test holds the file open, on the
// fake clock of a synctest bubble.
func TestRenameWithRetry_WaitsForAFileThatIsBrieflyInUse(t *testing.T) {
	t.Parallel()
	synctest.Test(t, renameWhileTheFileIsInUse)
}

func renameWhileTheFileIsInUse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notes.json", `{}`)
	writeFile(t, dir, "notes.json.tmp", `{"N-1": {"id": "N-1"}}`)
	inUse, err := os.Open(filepath.Join(dir, "notes.json")) //nolint:gosec // G304: a file of the test's own temporary folder
	if err != nil {
		t.Fatal(err)
	}
	released := make(chan struct{})
	go func() {
		time.Sleep(300 * time.Millisecond)
		if closeErr := inUse.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		close(released)
	}()

	err = renameWithRetry(filepath.Join(dir, "notes.json.tmp"), filepath.Join(dir, "notes.json"))
	<-released

	if err != nil {
		t.Fatalf("renameWithRetry: %v", err)
	}
	if content, _ := os.ReadFile(filepath.Join(dir, "notes.json")); string(content) != `{"N-1": {"id": "N-1"}}` { //nolint:gosec // G304: a file of the test's own temporary folder
		t.Errorf("notes.json = %s, want the new version", content)
	}
}

// The store keeps no file open between saves. On Windows an open file cannot be deleted, so deleting
// the whole data folder fails if one was left open (research R3).
func TestStore_KeepsNoFileOpen(t *testing.T) {
	t.Parallel()
	db, dir := openTempStore(t)
	putNote(t, db, note{ID: "N-1"})
	appendLog(t, db, entry{Seq: 1})
	reopened := mustOpen(t, dir)
	putNote(t, reopened, note{ID: "N-2"})
	appendLog(t, reopened, entry{Seq: 2})

	// Antivirus or search indexing can hold a new file for a moment, so this tries for up to 2 seconds.
	// A file that the store itself left open would stay locked and still fail the test.
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := os.RemoveAll(dir)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("deleting the data folder: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
