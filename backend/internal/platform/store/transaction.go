package store

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sync"
)

// tx is one transaction. A Write keeps its changes here until it saves them, so that a failed Write can
// simply forget them.
type tx struct {
	db       *DB
	writable bool
	changed  map[string]map[string]json.RawMessage // this Write's own copies of the tables it changed
	appended map[string][]json.RawMessage          // records this Write appended, per append-only table

	memoMu sync.Mutex
	memo   map[any]any // Memo's values and a Read's decoded tables (listKey); forgotten on every change
}

// txKey is the context key of the current transaction.
type txKey struct{}

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
