package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
)

// A table that has no file yet reads as empty, and lists are never nil (research R10).
func TestMissingTable_ReadsAsEmpty(t *testing.T) {
	t.Parallel()
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

// Appended records keep their order; a Write sees the records it appended itself; a save only adds lines
// to the end of the file (research R3).
func TestAppendAndScan_KeepTheOrder(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	outside := 0
	for range 2 {
		if _, err := Memo(t.Context(), "store.test:outside", func(context.Context) (int, error) { outside++; return outside, nil }); err != nil {
			t.Errorf("Memo outside a transaction: %v", err)
		}
	}
	if outside != 2 {
		t.Errorf("loads outside a transaction = %d, want 2 (nothing is kept)", outside)
	}
}

// Table names are checked, so that no file outside the data folder can ever be touched (research R3).
func TestTableNames_AreChecked(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	db, _ := openTempStore(t)

	err := db.Write(t.Context(), func(ctx context.Context) error {
		return Put(ctx, "notes", "", note{})
	})
	if err == nil {
		t.Error("Put with an empty id succeeded")
	}
}
