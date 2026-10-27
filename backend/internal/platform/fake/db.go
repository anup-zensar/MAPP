// Package fake is the in-memory stand-in of the database for tests: transactions, tables of records and the
// readable-ID counters. Each business area's fake store is built on it; only tests and apitest import it.
package fake

import (
	"context"
	"errors"
	"sync"
)

var ErrReadOnly = errors.New("fake: data cannot be changed inside a Read")

type txKey struct{}

type tx struct {
	db       *DB
	readOnly bool
	mu       sync.Mutex
	undo     []func()
}

// DB runs Writes one at a time, as database.DB does; a failed Write puts back everything it changed.
type DB struct {
	mu       sync.RWMutex
	tablesMu sync.Mutex
	tables   map[string]table
}

type table interface {
	copyTo(db *DB) table
}

func NewDB() *DB {
	return &DB{tables: map[string]table{}}
}

func (db *DB) Read(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, joined := db.current(ctx); joined {
		return fn(ctx)
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	return fn(context.WithValue(ctx, txKey{}, &tx{db: db, readOnly: true}))
}

func (db *DB) Write(ctx context.Context, fn func(ctx context.Context) error) error {
	if current, joined := db.current(ctx); joined {
		if current.readOnly {
			return ErrReadOnly
		}
		return fn(ctx)
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	t := &tx{db: db}
	committed := false
	defer func() {
		if !committed {
			t.rollBack()
		}
	}()
	if err := fn(context.WithValue(ctx, txKey{}, t)); err != nil {
		return err
	}
	committed = true
	return nil
}

// Copy returns a new DB holding the same records: one test changes its copy without touching the original.
func (db *DB) Copy() *DB {
	db.mu.RLock()
	defer db.mu.RUnlock()
	db.tablesMu.Lock()
	defer db.tablesMu.Unlock()
	copied := NewDB()
	for name, t := range db.tables {
		copied.tables[name] = t.copyTo(copied)
	}
	return copied
}

func (db *DB) current(ctx context.Context) (*tx, bool) {
	current, ok := ctx.Value(txKey{}).(*tx)
	return current, ok && current.db == db
}

// changing keeps nothing outside a transaction: such a change stands at once, as on a database connection.
func (db *DB) changing(ctx context.Context) (func(undo func()), error) {
	current, joined := db.current(ctx)
	switch {
	case !joined:
		return func(func()) {}, nil
	case current.readOnly:
		return nil, ErrReadOnly
	}
	return current.keep, nil
}

func (t *tx) keep(undo func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.undo = append(t.undo, undo)
}

func (t *tx) rollBack() {
	for i := len(t.undo) - 1; i >= 0; i-- {
		t.undo[i]()
	}
}
