package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
)

// countersTable is saved first. The readable-ID counters must never go backwards, even when a save
// breaks off half-way, or an ID could be given out twice (research R3, R4).
const countersTable = "counters"

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

// fail puts the store in the failed state after a save broke off half-way. It logs the cause and returns
// an error that carries ErrFailed, so the person sees what to do.
func (db *DB) fail(err error) error {
	db.failed = true
	slog.Error("could not save the data folder; restart the API", "folder", db.dir, "error", err)
	return fmt.Errorf("%w: %w", ErrFailed, err)
}
