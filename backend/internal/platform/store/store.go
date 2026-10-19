// Package store is NorthStar's JSON file store: the storage behind every repository until PostgreSQL
// replaces it (ADR-0004, research R3).
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
package store

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"northstar/internal/platform/errs"
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
		Message: "NorthStar could not save its data folder — restart the API.",
		Fields:  []errs.FieldError{},
	}
)

// countersTable is saved first. The readable-ID counters must never go backwards, even when a save
// breaks off half-way, or an ID could be given out twice (research R3, R4).
const countersTable = "counters"

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

// tx is one transaction. A Write keeps its changes here until it saves them, so that a failed Write can
// simply forget them.
type tx struct {
	db       *DB
	writable bool
	changed  map[string]map[string]json.RawMessage // this Write's own copies of the tables it changed
	appended map[string][]json.RawMessage          // records this Write appended, per append-only table
}

// txKey is the context key of the current transaction.
type txKey struct{}

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

// IsEmpty reports whether the store has no table at all: the data folder had no *.json or *.jsonl table
// file and nothing has been saved since. The seed command loads the demo data only into an empty store.
// Call it outside transactions.
func (db *DB) IsEmpty() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.tables) == 0 && len(db.logs) == 0
}

// Write runs fn with exclusive access to the store. When fn returns nil, everything it changed is saved
// to the data folder together; when fn returns an error, nothing changes and that error is returned
// unchanged. Called inside another Write (its context carries that transaction), Write joins it: its
// changes are saved or thrown away together with the outer Write's, so one service can call another.
// Called inside a Read, it returns ErrReadOnly.
//
// fn must pass on the context it receives. A transaction started inside fn with a fresh context, such as
// context.Background(), would wait forever for the one that is running.
func (db *DB) Write(ctx context.Context, fn func(ctx context.Context) error) error {
	if current, ok := ctx.Value(txKey{}).(*tx); ok {
		if !current.writable {
			return ErrReadOnly
		}
		return fn(ctx)
	}

	db.mu.Lock()
	defer db.mu.Unlock() // deferred, so that a panic in fn never leaves the store locked
	if db.failed {
		return ErrFailed
	}

	t := &tx{
		db:       db,
		writable: true,
		changed:  map[string]map[string]json.RawMessage{},
		appended: map[string][]json.RawMessage{},
	}
	if err := fn(context.WithValue(ctx, txKey{}, t)); err != nil {
		return err // the changes stay in t, which is forgotten
	}
	return db.commit(t)
}

// Read runs fn with a consistent view of the store: no Write changes the data while fn runs. Called
// inside another transaction, it joins it.
func (db *DB) Read(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*tx); ok {
		return fn(ctx)
	}

	db.mu.RLock()
	defer db.mu.RUnlock()
	return fn(context.WithValue(ctx, txKey{}, &tx{db: db}))
}

// Get returns the record id of table, and false when there is none.
func Get[T any](ctx context.Context, table, id string) (T, bool, error) {
	var record T
	t, err := transaction(ctx, table)
	if err != nil {
		return record, false, err
	}
	encoded, found := t.records(table)[id]
	if !found {
		return record, false, nil
	}
	if err := json.Unmarshal(encoded, &record); err != nil {
		return record, false, fmt.Errorf("store: read %s %s: %w", table, id, err)
	}
	return record, true, nil
}

// List returns every record of table, ordered by id: shorter ids first, then alphabetically, so that
// P-999 comes before P-1001. The list is never nil.
func List[T any](ctx context.Context, table string) ([]T, error) {
	t, err := transaction(ctx, table)
	if err != nil {
		return nil, err
	}
	records := t.records(table)
	list := make([]T, 0, len(records))
	for _, id := range slices.SortedFunc(maps.Keys(records), compareIDs) {
		var record T
		if err := json.Unmarshal(records[id], &record); err != nil {
			return nil, fmt.Errorf("store: read %s %s: %w", table, id, err)
		}
		list = append(list, record)
	}
	return list, nil
}

// Put saves record under id in table, replacing the record that had that id.
func Put(ctx context.Context, table, id string, record any) error {
	t, err := writableTransaction(ctx, table)
	if err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("store: put in %s: the id is empty", table)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("store: put %s %s: %w", table, id, err)
	}
	t.ownCopy(table)[id] = encoded
	return nil
}

// Delete removes the record id from table. Deleting a record that does not exist changes nothing.
func Delete(ctx context.Context, table, id string) error {
	t, err := writableTransaction(ctx, table)
	if err != nil {
		return err
	}
	if _, found := t.records(table)[id]; found {
		delete(t.ownCopy(table), id)
	}
	return nil
}

// Append adds record at the end of the append-only table.
func Append(ctx context.Context, table string, record any) error {
	t, err := writableTransaction(ctx, table)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(record) // one line: JSON escapes the line breaks inside strings
	if err != nil {
		return fmt.Errorf("store: append to %s: %w", table, err)
	}
	t.appended[table] = append(t.appended[table], encoded)
	return nil
}

// Scan returns every record of the append-only table in the order they were appended, including the
// records appended earlier in the same Write. The list is never nil.
func Scan[T any](ctx context.Context, table string) ([]T, error) {
	t, err := transaction(ctx, table)
	if err != nil {
		return nil, err
	}
	encoded := slices.Concat(t.db.logs[table], t.appended[table])
	list := make([]T, 0, len(encoded))
	for i, line := range encoded {
		var record T
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("store: read %s record %d: %w", table, i+1, err)
		}
		list = append(list, record)
	}
	return list, nil
}

// transaction returns the transaction that ctx carries, after checking the table name.
func transaction(ctx context.Context, table string) (*tx, error) {
	if !tableName.MatchString(table) {
		return nil, fmt.Errorf("store: %q is not a valid table name", table)
	}
	t, ok := ctx.Value(txKey{}).(*tx)
	if !ok {
		return nil, ErrNoTransaction
	}
	return t, nil
}

// writableTransaction is transaction for the helpers that change data, which need a Write.
func writableTransaction(ctx context.Context, table string) (*tx, error) {
	t, err := transaction(ctx, table)
	if err != nil {
		return nil, err
	}
	if !t.writable {
		return nil, ErrReadOnly
	}
	return t, nil
}

// records returns table as this transaction sees it: its own changed copy, or else the saved table.
func (t *tx) records(table string) map[string]json.RawMessage {
	if records, ok := t.changed[table]; ok {
		return records
	}
	return t.db.tables[table] // nil for a table that does not exist yet, which reads as empty
}

// ownCopy returns this transaction's own copy of table, made the first time the table changes. Changes
// go only to the copy, so the saved table stays as it was until the commit.
func (t *tx) ownCopy(table string) map[string]json.RawMessage {
	records, ok := t.changed[table]
	if !ok {
		records = maps.Clone(t.db.tables[table])
		if records == nil {
			records = map[string]json.RawMessage{}
		}
		t.changed[table] = records
	}
	return records
}

// compareIDs orders ids as people expect: shorter ids first, then alphabetically.
func compareIDs(a, b string) int {
	return cmp.Or(cmp.Compare(len(a), len(b)), strings.Compare(a, b))
}

// commit saves a Write's changes (research R3), as safely as a laptop allows:
//
//  1. Write every changed table to <table>.json.tmp and flush it to disk. If that fails, the temporary
//     files are deleted and nothing has changed.
//  2. Replace each table's file with its temporary file: counters.json first, then the others by name.
//  3. Append the new lines of the append-only tables and flush them to disk.
//  4. Only then apply the changes to memory.
//
// From step 2 on, files on disk are being changed. If anything fails then, memory and disk may no longer
// agree, so the store refuses every later Write until the API is restarted; reads keep working.
func (db *DB) commit(t *tx) error {
	tables := tablesInSaveOrder(t)
	if err := db.writeTempFiles(t, tables); err != nil {
		return err
	}
	for _, table := range tables {
		if err := db.rename(db.tablePath(table)+".tmp", db.tablePath(table)); err != nil {
			return db.fail(err)
		}
	}
	for _, table := range slices.Sorted(maps.Keys(t.appended)) {
		if err := appendLines(db.logPath(table), t.appended[table]); err != nil {
			return db.fail(err)
		}
	}

	maps.Copy(db.tables, t.changed)
	for table, records := range t.appended {
		db.logs[table] = append(db.logs[table], records...)
	}
	return nil
}

// tablesInSaveOrder returns the tables a transaction changed: counters first, then the others by name.
func tablesInSaveOrder(t *tx) []string {
	var tables []string
	if _, ok := t.changed[countersTable]; ok {
		tables = append(tables, countersTable)
	}
	for _, table := range slices.Sorted(maps.Keys(t.changed)) {
		if table != countersTable {
			tables = append(tables, table)
		}
	}
	return tables
}

// writeTempFiles writes each table to its temporary file. On an error it deletes the temporary files it
// wrote, so nothing has changed.
func (db *DB) writeTempFiles(t *tx, tables []string) error {
	var written []string
	for _, table := range tables {
		path := db.tablePath(table) + ".tmp"
		data, err := json.MarshalIndent(t.changed[table], "", "  ")
		if err == nil {
			err = writeAndSync(path, append(data, '\n'))
		}
		if err != nil {
			for _, done := range written {
				_ = os.Remove(done) // best effort: Open deletes leftover temporary files anyway
			}
			return fmt.Errorf("store: save %s: %w", table, err)
		}
		written = append(written, path)
	}
	return nil
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

// fail puts the store in the failed state after a save broke off half-way. It logs the cause and returns
// an error that carries ErrFailed, so the person sees what to do.
func (db *DB) fail(err error) error {
	db.failed = true
	slog.Error("could not save the data folder; restart the API", "folder", db.dir, "error", err)
	return fmt.Errorf("%w: %w", ErrFailed, err)
}

// tablePath is the file of a normal table.
func (db *DB) tablePath(table string) string {
	return filepath.Join(db.dir, table+".json")
}

// logPath is the file of an append-only table.
func (db *DB) logPath(table string) string {
	return filepath.Join(db.dir, table+".jsonl")
}
