package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"northstar/internal/platform/errs"
)

// note is the record type these tests store in the table "notes".
type note struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

// entry is the record type these tests append to the append-only table "log".
type entry struct {
	Seq int `json:"seq"`
}

func TestWrite_ReadsItsOwnWrites(t *testing.T) {
	db, _ := openTempStore(t)
	putNote(t, db, note{ID: "N-1", Title: "Kept until deleted"})

	err := db.Write(t.Context(), func(ctx context.Context) error {
		if err := Put(ctx, "notes", "N-2", note{ID: "N-2", Title: "New"}); err != nil {
			return err
		}
		got, found, err := Get[note](ctx, "notes", "N-2")
		if err != nil || !found || got.Title != "New" {
			t.Errorf("Get after Put in the same Write = %+v, %v, %v; want the new note", got, found, err)
		}

		if err := Delete(ctx, "notes", "N-1"); err != nil {
			return err
		}
		list, err := List[note](ctx, "notes")
		if err != nil {
			return err
		}
		if ids := noteIDs(list); !slices.Equal(ids, []string{"N-2"}) {
			t.Errorf("List after Delete in the same Write = %q, want [N-2]", ids)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

// When the function of a Write fails, nothing it changed is kept: not in memory, not on disk.
func TestWrite_FailedWriteKeepsTheOldValue(t *testing.T) {
	db, dir := openTempStore(t)
	putNote(t, db, note{ID: "N-1", Title: "Old"})
	failure := errors.New("the name is already taken")

	err := db.Write(t.Context(), func(ctx context.Context) error {
		if err := Put(ctx, "notes", "N-1", note{ID: "N-1", Title: "New"}); err != nil {
			return err
		}
		if err := Append(ctx, "log", entry{Seq: 1}); err != nil {
			return err
		}
		return failure
	})
	if err != failure {
		t.Fatalf("Write = %v, want the function's error", err)
	}

	if got, _ := getNote(t, db, "N-1"); got.Title != "Old" {
		t.Errorf("in memory: title %q, want Old", got.Title)
	}
	reopened := mustOpen(t, dir)
	if got, _ := getNote(t, reopened, "N-1"); got.Title != "Old" {
		t.Errorf("on disk: title %q, want Old", got.Title)
	}
	if log := scanLog(t, reopened); len(log) != 0 {
		t.Errorf("on disk: log %v, want no entries", log)
	}
}

// Read and Write return the function's error unchanged, so an API error keeps its status (research R9).
func TestReadAndWrite_ReturnTheFunctionsErrorUnchanged(t *testing.T) {
	db, _ := openTempStore(t)
	notFound := errs.NotFound("This sample folder no longer exists — someone may have deleted it.")

	if err := db.Write(t.Context(), func(context.Context) error { return notFound }); err != notFound {
		t.Errorf("Write = %v, want the function's error itself", err)
	}
	if err := db.Read(t.Context(), func(context.Context) error { return notFound }); err != notFound {
		t.Errorf("Read = %v, want the function's error itself", err)
	}
}

// Callers never share memory with the store: changing a value after Put, or a value returned by Get or
// List, does not change what is stored.
func TestGetAndList_ReturnCopies(t *testing.T) {
	db, _ := openTempStore(t)
	saved := note{ID: "N-1", Title: "Ideas", Tags: []string{"original"}}
	putNote(t, db, saved)

	saved.Tags[0] = "changed after Put"
	got, _ := getNote(t, db, "N-1")
	got.Tags[0] = "changed after Get"
	err := db.Read(t.Context(), func(ctx context.Context) error {
		list, err := List[note](ctx, "notes")
		if err == nil {
			list[0].Tags[0] = "changed after List"
		}
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if again, _ := getNote(t, db, "N-1"); again.Tags[0] != "original" {
		t.Errorf("stored tag = %q, want original", again.Tags[0])
	}
}

// Everything a Write saved is there again after the API restarts.
func TestOpen_ReloadsSavedData(t *testing.T) {
	db, dir := openTempStore(t)
	saved := note{ID: "N-1", Title: "Ideas", Tags: []string{"a", "b"}}
	putNote(t, db, saved)
	appendLog(t, db, entry{Seq: 1}, entry{Seq: 2})

	reopened := mustOpen(t, dir)

	got, found := getNote(t, reopened, "N-1")
	if !found || got.Title != saved.Title || !slices.Equal(got.Tags, saved.Tags) {
		t.Errorf("after reopening: %+v, %v; want %+v", got, found, saved)
	}
	if log := scanLog(t, reopened); !slices.Equal(log, []int{1, 2}) {
		t.Errorf("after reopening: log %v, want [1 2]", log)
	}
}

// A Write or Read inside a Write joins it (one service can call another without waiting forever); the
// joined changes are kept or thrown away together with the outer Write.
func TestWrite_JoinsTheCallersTransaction(t *testing.T) {
	db, _ := openTempStore(t)
	failure := errors.New("the outer service failed")

	err := db.Write(t.Context(), func(ctx context.Context) error {
		err := db.Write(ctx, func(ctx context.Context) error {
			return Put(ctx, "notes", "N-1", note{ID: "N-1", Title: "From the inner Write"})
		})
		if err != nil {
			return err
		}
		err = db.Read(ctx, func(ctx context.Context) error {
			if _, found, err := Get[note](ctx, "notes", "N-1"); err != nil || !found {
				t.Errorf("a Read inside the Write did not see the inner change: %v, %v", found, err)
			}
			return nil
		})
		if err != nil {
			return err
		}
		return failure
	})
	if err != failure {
		t.Fatalf("Write = %v, want the outer function's error", err)
	}
	if _, found := getNote(t, db, "N-1"); found {
		t.Error("the inner change was saved although the outer Write failed")
	}
}

func TestRead_RefusesChanges(t *testing.T) {
	db, _ := openTempStore(t)

	err := db.Read(t.Context(), func(ctx context.Context) error {
		if err := db.Write(ctx, func(context.Context) error { return nil }); !errors.Is(err, ErrReadOnly) {
			t.Errorf("Write inside Read = %v, want ErrReadOnly", err)
		}
		if err := Put(ctx, "notes", "N-1", note{ID: "N-1"}); !errors.Is(err, ErrReadOnly) {
			t.Errorf("Put inside Read = %v, want ErrReadOnly", err)
		}
		if err := Delete(ctx, "notes", "N-1"); !errors.Is(err, ErrReadOnly) {
			t.Errorf("Delete inside Read = %v, want ErrReadOnly", err)
		}
		if err := Append(ctx, "log", entry{Seq: 1}); !errors.Is(err, ErrReadOnly) {
			t.Errorf("Append inside Read = %v, want ErrReadOnly", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// The helpers work only inside Read or Write, where the context carries the transaction.
func TestHelpers_NeedATransaction(t *testing.T) {
	ctx := t.Context()

	if _, _, err := Get[note](ctx, "notes", "N-1"); !errors.Is(err, ErrNoTransaction) {
		t.Errorf("Get = %v, want ErrNoTransaction", err)
	}
	if _, err := List[note](ctx, "notes"); !errors.Is(err, ErrNoTransaction) {
		t.Errorf("List = %v, want ErrNoTransaction", err)
	}
	if err := Put(ctx, "notes", "N-1", note{ID: "N-1"}); !errors.Is(err, ErrNoTransaction) {
		t.Errorf("Put = %v, want ErrNoTransaction", err)
	}
	if err := Delete(ctx, "notes", "N-1"); !errors.Is(err, ErrNoTransaction) {
		t.Errorf("Delete = %v, want ErrNoTransaction", err)
	}
	if err := Append(ctx, "log", entry{Seq: 1}); !errors.Is(err, ErrNoTransaction) {
		t.Errorf("Append = %v, want ErrNoTransaction", err)
	}
	if _, err := Scan[entry](ctx, "log"); !errors.Is(err, ErrNoTransaction) {
		t.Errorf("Scan = %v, want ErrNoTransaction", err)
	}
}

// Writes run one at a time, so no change is lost when requests arrive together.
func TestWrite_ParallelWritesAreSerialised(t *testing.T) {
	db, _ := openTempStore(t)
	type count struct {
		N int `json:"n"`
	}
	const writers = 20

	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			err := db.Write(t.Context(), func(ctx context.Context) error {
				c, _, err := Get[count](ctx, "counts", "total")
				if err != nil {
					return err
				}
				c.N++
				return Put(ctx, "counts", "total", c)
			})
			if err != nil {
				t.Errorf("Write: %v", err)
			}
		})
	}
	wg.Wait()

	err := db.Read(t.Context(), func(ctx context.Context) error {
		c, _, err := Get[count](ctx, "counts", "total")
		if c.N != writers {
			t.Errorf("total = %d, want %d", c.N, writers)
		}
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// A panic in the function of a Write does not leave the store locked.
func TestWrite_PanicDoesNotLockTheStore(t *testing.T) {
	db, _ := openTempStore(t)

	func() {
		defer func() { _ = recover() }()
		_ = db.Write(t.Context(), func(context.Context) error { panic("boom") })
	}()

	putNote(t, db, note{ID: "N-1", Title: "Saved after the panic"})
	if _, found := getNote(t, db, "N-1"); !found {
		t.Error("the Write after the panic saved nothing")
	}
}

// The counters table is saved first, then the other tables by name (research R3).
func TestCommit_SavesCountersFirstThenByName(t *testing.T) {
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
	db, dir := openTempStore(t)
	putNote(t, db, note{ID: "N-1", Title: "Old"})
	diskFailure := errors.New("the file is being used by another process")
	db.rename = func(from, to string) error { return diskFailure }

	err := db.Write(t.Context(), func(ctx context.Context) error {
		return Put(ctx, "notes", "N-1", note{ID: "N-1", Title: "New"})
	})
	if !errors.Is(err, ErrFailed) || !errors.Is(err, diskFailure) {
		t.Fatalf("Write = %v, want ErrFailed caused by the disk failure", err)
	}

	if got, _ := getNote(t, db, "N-1"); got.Title != "Old" {
		t.Errorf("in memory: title %q, want Old", got.Title)
	}
	if got, _ := getNote(t, mustOpen(t, dir), "N-1"); got.Title != "Old" {
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
	if want := "NorthStar could not save its data folder — restart the API."; apiErr.Message != want {
		t.Errorf("message = %q, want %q", apiErr.Message, want)
	}
}

// When a temporary file cannot be written, the temporary files already written are deleted, nothing
// changes, and the store keeps working (research R3).
func TestCommit_TempFileFailure_NothingChangesAndTheStoreKeepsWorking(t *testing.T) {
	db, dir := openTempStore(t)
	putNote(t, db, note{ID: "N-1", Title: "Old"})
	blocker := filepath.Join(dir, "zebra.json.tmp") // a folder where the temporary file must go
	if err := os.Mkdir(blocker, 0o755); err != nil {
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
	if got, _ := getNote(t, db, "N-1"); got.Title != "Old" {
		t.Errorf("in memory: title %q, want Old", got.Title)
	}
	if got, _ := getNote(t, mustOpen(t, dir), "N-1"); got.Title != "Old" {
		t.Errorf("on disk: title %q, want Old", got.Title)
	}

	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	putNote(t, db, note{ID: "N-1", Title: "New"})
	if got, _ := getNote(t, db, "N-1"); got.Title != "New" {
		t.Errorf("after the problem was gone: title %q, want New", got.Title)
	}
}

// On Windows, antivirus or search indexing can hold a data file open for a moment, which makes replacing
// it fail; the store tries again for up to 2 seconds (research R3). Here the test holds the file open.
func TestRenameWithRetry_WaitsForAFileThatIsBrieflyInUse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notes.json", `{}`)
	writeFile(t, dir, "notes.json.tmp", `{"N-1": {"id": "N-1"}}`)
	inUse, err := os.Open(filepath.Join(dir, "notes.json"))
	if err != nil {
		t.Fatal(err)
	}
	released := make(chan struct{})
	go func() {
		time.Sleep(300 * time.Millisecond)
		inUse.Close()
		close(released)
	}()

	err = renameWithRetry(filepath.Join(dir, "notes.json.tmp"), filepath.Join(dir, "notes.json"))
	<-released

	if err != nil {
		t.Fatalf("renameWithRetry: %v", err)
	}
	if content, _ := os.ReadFile(filepath.Join(dir, "notes.json")); string(content) != `{"N-1": {"id": "N-1"}}` {
		t.Errorf("notes.json = %s, want the new version", content)
	}
}

func TestOpen_CreatesAMissingFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "backend", "data")

	db := mustOpen(t, dir)

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("data folder: %v, %v; want a new folder", info, err)
	}
	if !db.IsEmpty() {
		t.Error("IsEmpty() = false for a new folder")
	}
}

// A table that has no file yet reads as empty, and lists are never nil (research R10).
func TestMissingTable_ReadsAsEmpty(t *testing.T) {
	db, _ := openTempStore(t)

	err := db.Read(t.Context(), func(ctx context.Context) error {
		if _, found, err := Get[note](ctx, "notes", "N-1"); err != nil || found {
			t.Errorf("Get = %v, %v; want not found and no error", found, err)
		}
		if list, err := List[note](ctx, "notes"); err != nil || list == nil || len(list) != 0 {
			t.Errorf("List = %#v, %v; want an empty, non-nil list", list, err)
		}
		if log, err := Scan[entry](ctx, "log"); err != nil || log == nil || len(log) != 0 {
			t.Errorf("Scan = %#v, %v; want an empty, non-nil list", log, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// Temporary files left by a save that a crash interrupted are deleted; the real files are kept.
func TestOpen_RemovesLeftoverTempFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notes.json", `{"N-1": {"id": "N-1", "title": "Kept", "tags": []}}`)
	writeFile(t, dir, "notes.json.tmp", `{"N-1": {"id": "N-1", "title": "Half-saved"`)
	writeFile(t, dir, "log.jsonl.tmp", `{"seq": 1}`)

	db := mustOpen(t, dir)

	for _, name := range []string{"notes.json.tmp", "log.jsonl.tmp"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s still exists: %v", name, err)
		}
	}
	if got, _ := getNote(t, db, "N-1"); got.Title != "Kept" {
		t.Errorf("title %q, want Kept", got.Title)
	}
}

// A data file that is not valid JSON stops the start-up with a message that names the file and the
// line, and says how to fix it.
func TestOpen_InvalidJSONStopsWithAMessage(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		content string
		line    int
	}{
		{"broken table", "people.json", "{\n  \"U-1001\": {\"name\": \"Alex Morgan\"},\n  \"U-1002\": {\"name\": \"Priya Raman\",}\n}\n", 3},
		{"table that is not an object", "people.json", "[]\n", 1},
		{"broken line of an append-only table", "log.jsonl", "{\"seq\": 1}\nnot JSON\n{\"seq\": 3}\n", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, tt.file, tt.content)

			_, err := Open(dir)

			path := filepath.ToSlash(filepath.Join(dir, tt.file))
			want := fmt.Sprintf("%s is not valid JSON (line %d) — fix it or run go run ./cmd/seed -reset.", path, tt.line)
			if err == nil || err.Error() != want {
				t.Errorf("Open = %v\nwant %s", err, want)
			}
		})
	}
}

// A half-written last line of an append-only table (a crash during an append) is ignored and cut off
// the file, so the next append starts on a fresh line (research R3).
func TestOpen_PartialLastLineOfAnAppendOnlyTableIsIgnored(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "log.jsonl", "{\"seq\":1}\n{\"seq\":2}\n{\"seq\":3,\"wri")

	db := mustOpen(t, dir)
	if log := scanLog(t, db); !slices.Equal(log, []int{1, 2}) {
		t.Fatalf("log %v, want [1 2]", log)
	}

	appendLog(t, db, entry{Seq: 3})
	if log := scanLog(t, mustOpen(t, dir)); !slices.Equal(log, []int{1, 2, 3}) {
		t.Errorf("after an append and a reopen: log %v, want [1 2 3]", log)
	}
}

// Appended records keep their order; a Write sees the records it appended itself; a save only adds lines
// to the end of the file (research R3).
func TestAppendAndScan_KeepTheOrder(t *testing.T) {
	db, dir := openTempStore(t)

	err := db.Write(t.Context(), func(ctx context.Context) error {
		for _, seq := range []int{1, 2} {
			if err := Append(ctx, "log", entry{Seq: seq}); err != nil {
				return err
			}
		}
		log, err := Scan[entry](ctx, "log")
		if len(log) != 2 {
			t.Errorf("Scan inside the Write = %v, want its own 2 entries", log)
		}
		return err
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	appendLog(t, db, entry{Seq: 3})

	if log := scanLog(t, db); !slices.Equal(log, []int{1, 2, 3}) {
		t.Errorf("log %v, want [1 2 3]", log)
	}
	content, err := os.ReadFile(filepath.Join(dir, "log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\"seq\":1}\n{\"seq\":2}\n{\"seq\":3}\n"; string(content) != want {
		t.Errorf("log.jsonl =\n%s\nwant\n%s", content, want)
	}
}

// List sorts by id as people expect: shorter ids first, then alphabetically (research R3).
func TestList_SortsShorterIDsFirst(t *testing.T) {
	db, _ := openTempStore(t)
	for _, id := range []string{"P-1001", "P-999", "P-1000", "P-10"} {
		putNote(t, db, note{ID: id})
	}

	var ids []string
	err := db.Read(t.Context(), func(ctx context.Context) error {
		list, err := List[note](ctx, "notes")
		ids = noteIDs(list)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if want := []string{"P-10", "P-999", "P-1000", "P-1001"}; !slices.Equal(ids, want) {
		t.Errorf("List order %q, want %q", ids, want)
	}
}

// decodes counts the records decoded as countedNote, so the tests can see how often List decodes.
var decodes atomic.Int64

// countedNote is a note that counts its decodes.
type countedNote note

func (n *countedNote) UnmarshalJSON(data []byte) error {
	decodes.Add(1)
	return json.Unmarshal(data, (*note)(n))
}

// A Read decodes a table once however often List reads it, and every List gets a list of its own; a Write
// decodes afresh, so it sees its own changes.
func TestList_ARead_DecodesEachTableOnce(t *testing.T) {
	db, _ := openTempStore(t)
	putNote(t, db, note{ID: "N-1", Title: "One"})
	putNote(t, db, note{ID: "N-2", Title: "Two"})

	decodes.Store(0)
	err := db.Read(t.Context(), func(ctx context.Context) error {
		first, err := List[countedNote](ctx, "notes")
		if err != nil {
			return err
		}
		slices.Reverse(first)
		first = append(first, countedNote{ID: "N-3"})
		second, err := List[countedNote](ctx, "notes")
		if err != nil {
			return err
		}
		ids := make([]string, 0, len(second))
		for _, n := range second {
			ids = append(ids, n.ID)
		}
		if !slices.Equal(ids, []string{"N-1", "N-2"}) {
			t.Errorf("second List = %+v, want N-1 and N-2 in order, untouched by the changes to the first", second)
		}
		plain, err := List[note](ctx, "notes") // another record type decodes on its own
		if err != nil || len(plain) != 2 {
			t.Errorf("List[note] = %+v, %v; want the two notes", plain, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := decodes.Load(); got != 2 {
		t.Errorf("records decoded in the Read = %d, want 2 (the table once)", got)
	}

	decodes.Store(0)
	err = db.Write(t.Context(), func(ctx context.Context) error {
		if _, err := List[countedNote](ctx, "notes"); err != nil {
			return err
		}
		if err := Put(ctx, "notes", "N-3", note{ID: "N-3"}); err != nil {
			return err
		}
		list, err := List[countedNote](ctx, "notes")
		if err == nil && len(list) != 3 {
			t.Errorf("List after Put in the same Write has %d notes, want 3", len(list))
		}
		return err
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := decodes.Load(); got != 5 {
		t.Errorf("records decoded in the Write = %d, want 5 (2, then 3 after the Put)", got)
	}
}

// Memo works a value out once per transaction, forgets it on every change, and keeps no error.
func TestMemo_OncePerTransaction_ForgottenOnChange(t *testing.T) {
	db, _ := openTempStore(t)
	loads := 0
	count := func(ctx context.Context) (int, error) {
		loads++
		list, err := List[note](ctx, "notes")
		return len(list), err
	}
	err := db.Write(t.Context(), func(ctx context.Context) error {
		for range 2 {
			if n, err := Memo(ctx, "store.test:count", count); err != nil || n != 0 {
				t.Errorf("Memo = %d, %v; want 0", n, err)
			}
		}
		if loads != 1 {
			t.Errorf("loads before the change = %d, want 1", loads)
		}
		if err := Put(ctx, "notes", "N-1", note{ID: "N-1"}); err != nil {
			return err
		}
		if n, err := Memo(ctx, "store.test:count", count); err != nil || n != 1 {
			t.Errorf("Memo after Put = %d, %v; want 1", n, err)
		}
		if loads != 2 {
			t.Errorf("loads after the change = %d, want 2", loads)
		}
		if _, err := Memo(ctx, "store.test:count", func(context.Context) (string, error) { return "", nil }); err == nil {
			t.Error("Memo with another type for the same key: no error, want one")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	failing := errors.New("not now")
	err = db.Read(t.Context(), func(ctx context.Context) error {
		if _, err := Memo(ctx, "store.test:fails", func(context.Context) (int, error) { return 0, failing }); !errors.Is(err, failing) {
			t.Errorf("Memo error = %v, want the load's error", err)
		}
		n, err := Memo(ctx, "store.test:fails", func(context.Context) (int, error) { return 7, nil })
		if err != nil || n != 7 {
			t.Errorf("Memo after a failed load = %d, %v; want 7 (the error is not kept)", n, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if _, err := Memo(t.Context(), "store.test:outside", count); !errors.Is(err, ErrNoTransaction) {
		t.Errorf("Memo outside a transaction = %v, want ErrNoTransaction", err)
	}
}

// Table names are checked, so that no file outside the data folder can ever be touched (research R3).
func TestTableNames_AreChecked(t *testing.T) {
	db, _ := openTempStore(t)

	err := db.Write(t.Context(), func(ctx context.Context) error {
		for _, table := range []string{"", "People", "../people", "people.json", "1people", "peo-ple", `data\people`} {
			if err := Put(ctx, table, "x", note{ID: "x"}); err == nil {
				t.Errorf("Put accepted the table name %q", table)
			}
			if _, _, err := Get[note](ctx, table, "x"); err == nil {
				t.Errorf("Get accepted the table name %q", table)
			}
			if err := Append(ctx, table, entry{Seq: 1}); err == nil {
				t.Errorf("Append accepted the table name %q", table)
			}
		}
		for _, table := range []string{"people", "sample_folders", "a1"} {
			if err := Put(ctx, table, "x", note{ID: "x"}); err != nil {
				t.Errorf("Put refused the table name %q: %v", table, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func TestPut_RefusesAnEmptyID(t *testing.T) {
	db, _ := openTempStore(t)

	err := db.Write(t.Context(), func(ctx context.Context) error {
		return Put(ctx, "notes", "", note{})
	})
	if err == nil {
		t.Error("Put with an empty id succeeded")
	}
}

// A Write that changed nothing writes no file; deleting a record that does not exist changes nothing.
func TestWrite_WithoutChangesWritesNoFile(t *testing.T) {
	db, dir := openTempStore(t)

	err := db.Write(t.Context(), func(ctx context.Context) error {
		if _, _, err := Get[note](ctx, "notes", "N-1"); err != nil {
			return err
		}
		return Delete(ctx, "notes", "N-1")
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if files, _ := os.ReadDir(dir); len(files) != 0 {
		t.Errorf("the data folder has %d files, want none", len(files))
	}
	if !db.IsEmpty() {
		t.Error("IsEmpty() = false after a Write that changed nothing")
	}
}

// The store keeps no file open between saves. On Windows an open file cannot be deleted, so deleting
// the whole data folder fails if one was left open (research R3).
func TestStore_KeepsNoFileOpen(t *testing.T) {
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

// IsEmpty is true only while the data folder has no table file (the seed loads only into an empty store).
func TestIsEmpty(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "README.txt", "not a table")

	db := mustOpen(t, dir)
	if !db.IsEmpty() {
		t.Error("IsEmpty() = false for a folder without table files")
	}

	putNote(t, db, note{ID: "N-1"})
	if db.IsEmpty() {
		t.Error("IsEmpty() = true after a saved record")
	}
	if mustOpen(t, dir).IsEmpty() {
		t.Error("IsEmpty() = true after reopening a folder with a table file")
	}

	logOnly := t.TempDir()
	writeFile(t, logOnly, "log.jsonl", "{\"seq\":1}\n")
	if mustOpen(t, logOnly).IsEmpty() {
		t.Error("IsEmpty() = true for a folder with an append-only table")
	}
}

// openTempStore opens a store on a new temporary folder.
func openTempStore(t *testing.T) (*DB, string) {
	t.Helper()
	dir := t.TempDir()
	return mustOpen(t, dir), dir
}

func mustOpen(t *testing.T, dir string) *DB {
	t.Helper()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return db
}

// putNote saves n in a Write of its own.
func putNote(t *testing.T, db *DB, n note) {
	t.Helper()
	err := db.Write(t.Context(), func(ctx context.Context) error {
		return Put(ctx, "notes", n.ID, n)
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

// getNote reads a note in a Read of its own.
func getNote(t *testing.T, db *DB, id string) (note, bool) {
	t.Helper()
	var n note
	var found bool
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		n, found, err = Get[note](ctx, "notes", id)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return n, found
}

// appendLog appends entries to the table "log" in a Write of its own.
func appendLog(t *testing.T, db *DB, entries ...entry) {
	t.Helper()
	err := db.Write(t.Context(), func(ctx context.Context) error {
		for _, e := range entries {
			if err := Append(ctx, "log", e); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

// scanLog returns the sequence numbers in the table "log".
func scanLog(t *testing.T, db *DB) []int {
	t.Helper()
	var seqs []int
	err := db.Read(t.Context(), func(ctx context.Context) error {
		log, err := Scan[entry](ctx, "log")
		for _, e := range log {
			seqs = append(seqs, e.Seq)
		}
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return seqs
}

func noteIDs(notes []note) []string {
	ids := make([]string, 0, len(notes))
	for _, n := range notes {
		ids = append(ids, n.ID)
	}
	return ids
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
