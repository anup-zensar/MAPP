package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
