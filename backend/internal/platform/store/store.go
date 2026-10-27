// Package store is MAPP's JSON file store, since feature 023 only the stand-in of service, handler and
// contract tests; feature 025 replaces it with fakes and deletes it (ADR-0013). Transactor stays until then.
//
// Every record type is a table. A normal table is kept in memory and saved as <data folder>/<table>.json:
// an object of records keyed by id, indented so that people can read it. An append-only table (the
// activity log) is saved as <table>.jsonl, one record per line, and a save only adds lines at its end.
//
// All work happens in transactions. Write runs a function with exclusive access and saves everything the
// function changed together, or nothing when it returns an error; Read gives a consistent view. The
// transaction travels in the context passed to the function, where the helpers Get, List, Put, Delete,
// Append and Scan find it:
//
//	err := db.Write(ctx, func(ctx context.Context) error {
//		folder, found, err := store.Get[Folder](ctx, "sample_folders", id)
//		...
//		return store.Put(ctx, "sample_folders", id, folder)
//	})
//
// Services depend only on the Transactor interface and on their own repository interfaces, never on DB,
// so a PostgreSQL implementation can replace this store without changing them.
//
// Memo keeps a value worked out from the data for the rest of the transaction, so that what many records
// need, such as the Shell attribute forms, is decoded once per request instead of once per record. For
// the same reason a Read decodes each table once, however many times List reads it: pages such as Home
// and the Plans list build the tree of every plan, and every tree reads the same tables.
package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"mapp/internal/platform/errs"
)

// Transactor runs work in transactions. Services depend on this interface, never on the JSON store.
type Transactor interface {
	// Read runs fn with a consistent, read-only view of the data.
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	// Write runs fn with exclusive access and saves all of its changes together, or none when fn fails.
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

var (
	// ErrNoTransaction means a helper was called outside Read and Write.
	ErrNoTransaction = errors.New("store: no transaction in the context; call this inside db.Read or db.Write")

	// ErrReadOnly means something tried to change data inside a Read.
	ErrReadOnly = errors.New("store: data cannot be changed inside a Read")

	// ErrFailed refuses every Write after a save broke off half-way (memory and the files may then
	// differ) until the API is restarted. It is an *errs.Error, so the person sees what to do.
	ErrFailed = &errs.Error{
		Status:  http.StatusInternalServerError,
		Code:    errs.CodeInternalError,
		Message: "MAPP could not save its data folder — restart the API.",
		Fields:  []errs.FieldError{},
	}
)

// tableName is the form every table name must have, so that no file outside the data folder can ever be
// touched.
var tableName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// The compiler checks here that DB implements Transactor.
var _ Transactor = (*DB)(nil)

// DB is the JSON file store of one data folder. It implements Transactor.
type DB struct {
	dir    string
	mu     sync.RWMutex                          // a Write holds it alone; Reads share it
	tables map[string]map[string]json.RawMessage // normal tables: table → record id → encoded record
	logs   map[string][]json.RawMessage          // append-only tables: table → encoded records in order
	failed bool                                  // a save broke off half-way: Writes are refused

	// rename replaces a data file with its new version. Tests replace it to simulate a failing disk.
	rename func(from, to string) error
}

// Open loads the data folder dir, creating it when it is missing; a table without a file is empty.
// Temporary files left by an interrupted save are deleted, and a half-written last line of an
// append-only table (a crash during an append) is cut off. A file that is not valid JSON stops the
// start-up with a message that says how to fix it. No file stays open afterwards.
func Open(dir string) (*DB, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	db := &DB{
		dir:    dir,
		tables: map[string]map[string]json.RawMessage{},
		logs:   map[string][]json.RawMessage{},
		rename: renameWithRetry,
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		path := filepath.Join(dir, name)
		if strings.HasSuffix(name, ".tmp") {
			err = os.Remove(path)
		} else if table, ok := strings.CutSuffix(name, ".json"); ok && tableName.MatchString(table) {
			err = db.loadTable(path, table)
		} else if table, ok := strings.CutSuffix(name, ".jsonl"); ok && tableName.MatchString(table) {
			err = db.loadLog(path, table)
		}
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}

// loadTable reads a normal table: a JSON object of records keyed by id.
func (db *DB) loadTable(path, table string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	records := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &records); err != nil {
		return invalidJSON(path, lineOfError(data, err))
	}
	db.tables[table] = records
	return nil
}

// loadLog reads an append-only table: one record per line. A line was saved completely only when it ends
// with a newline. A half-written last line is cut off the file, so that the next append starts on a fresh
// line instead of making that line invalid.
func (db *DB) loadLog(path, table string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	complete := data[:bytes.LastIndexByte(data, '\n')+1]
	if len(complete) < len(data) {
		slog.Warn("cut off a half-written last line", "file", path)
		if err := os.Truncate(path, int64(len(complete))); err != nil {
			return err
		}
	}

	records := []json.RawMessage{}
	lineNumber := 0
	for line := range bytes.Lines(complete) {
		lineNumber++
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue // an empty line holds no record
		}
		if !json.Valid(line) {
			return invalidJSON(path, lineNumber)
		}
		records = append(records, line)
	}
	db.logs[table] = records
	return nil
}

// invalidJSON is the start-up message for a data file that is not valid JSON.
func invalidJSON(path string, line int) error {
	return fmt.Errorf("%s is not valid JSON (line %d) — fix it or run go run ./cmd/seed -reset.",
		filepath.ToSlash(path), line)
}

// lineOfError returns the line of data on which JSON decoding failed.
func lineOfError(data []byte, err error) int {
	var offset int64
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) {
		offset = syntaxErr.Offset
	} else if errors.As(err, &typeErr) {
		offset = typeErr.Offset
	}
	offset = min(offset, int64(len(data)))
	return 1 + bytes.Count(data[:offset], []byte("\n"))
}

// writeAndSync writes data to a new file at path and flushes it to disk. On an error the file is removed.
func writeAndSync(path string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		_ = os.Remove(path) // best effort: Open deletes leftover temporary files anyway
		return err
	}
	return nil
}

// appendLines adds one line per record at the end of the file at path and flushes it to disk.
func appendLines(path string, records []json.RawMessage) error {
	var data []byte
	for _, record := range records {
		data = append(data, record...)
		data = append(data, '\n')
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	return errors.Join(writeErr, syncErr, closeErr)
}

// renameWithRetry replaces the file to with the file from. On Windows, antivirus or search indexing can
// hold a file open for a moment, which makes a rename fail, so it tries again for up to 2 seconds.
func renameWithRetry(from, to string) error {
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := os.Rename(from, to)
		if err == nil || time.Now().After(deadline) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// tablePath is the file of a normal table.
func (db *DB) tablePath(table string) string {
	return filepath.Join(db.dir, table+".json")
}

// logPath is the file of an append-only table.
func (db *DB) logPath(table string) string {
	return filepath.Join(db.dir, table+".jsonl")
}
