package store

import (
	"context"
	"errors"
	"os"
	"slices"
	"sync"
	"testing"

	"mapp/internal/platform/errs"
)

func TestWrite_ReadsItsOwnWrites(t *testing.T) {
	t.Parallel()
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

		if deleteErr := Delete(ctx, "notes", "N-1"); deleteErr != nil {
			return deleteErr
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
	t.Parallel()
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
	if !errors.Is(err, failure) {
		t.Fatalf("Write = %v, want the function's error", err)
	}

	if got, _ := getNote(t, db); got.Title != "Old" {
		t.Errorf("in memory: title %q, want Old", got.Title)
	}
	reopened := mustOpen(t, dir)
	if got, _ := getNote(t, reopened); got.Title != "Old" {
		t.Errorf("on disk: title %q, want Old", got.Title)
	}
	if log := scanLog(t, reopened); len(log) != 0 {
		t.Errorf("on disk: log %v, want no entries", log)
	}
}

// Read and Write return the function's error unchanged, so an API error keeps its status (research R9).
func TestReadAndWrite_ReturnTheFunctionsErrorUnchanged(t *testing.T) {
	t.Parallel()
	db, _ := openTempStore(t)
	notFound := errs.NotFound("This sample folder no longer exists — someone may have deleted it.")

	if err := db.Write(t.Context(), func(context.Context) error { return notFound }); err != notFound { //nolint:errorlint // the error itself, not one that wraps it
		t.Errorf("Write = %v, want the function's error itself", err)
	}
	if err := db.Read(t.Context(), func(context.Context) error { return notFound }); err != notFound { //nolint:errorlint // the error itself, not one that wraps it
		t.Errorf("Read = %v, want the function's error itself", err)
	}
}

// A Write or Read inside a Write joins it (one service can call another without waiting forever); the
// joined changes are kept or thrown away together with the outer Write.
func TestWrite_JoinsTheCallersTransaction(t *testing.T) {
	t.Parallel()
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
			if _, found, getErr := Get[note](ctx, "notes", "N-1"); getErr != nil || !found {
				t.Errorf("a Read inside the Write did not see the inner change: %v, %v", found, getErr)
			}
			return nil
		})
		if err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("Write = %v, want the outer function's error", err)
	}
	if _, found := getNote(t, db); found {
		t.Error("the inner change was saved although the outer Write failed")
	}
}

func TestRead_RefusesChanges(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	db, _ := openTempStore(t)

	func() {
		defer func() { _ = recover() }()
		_ = db.Write(t.Context(), func(context.Context) error { panic("boom") })
	}()

	putNote(t, db, note{ID: "N-1", Title: "Saved after the panic"})
	if _, found := getNote(t, db); !found {
		t.Error("the Write after the panic saved nothing")
	}
}

// A Write that changed nothing writes no file; deleting a record that does not exist changes nothing.
func TestWrite_WithoutChangesWritesNoFile(t *testing.T) {
	t.Parallel()
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
