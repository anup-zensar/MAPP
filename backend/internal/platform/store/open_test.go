package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Everything a Write saved is there again after the API restarts.
func TestOpen_ReloadsSavedData(t *testing.T) {
	t.Parallel()
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

func TestOpen_CreatesAMissingFolder(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "backend", "data")

	db := mustOpen(t, dir)

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("data folder: %v, %v; want a new folder", info, err)
	}
	if !db.IsEmpty() {
		t.Error("IsEmpty() = false for a new folder")
	}
}

// Temporary files left by a save that a crash interrupted are deleted; the real files are kept.
func TestOpen_RemovesLeftoverTempFiles(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// IsEmpty is true only while the data folder has no table file (the seed loads only into an empty store).
func TestIsEmpty(t *testing.T) {
	t.Parallel()
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
