package store

import (
	"fmt"
	"os"
)

// IsEmpty reports whether the store has no table at all: the data folder had no *.json or *.jsonl table
// file and nothing has been saved since. The seed command loads the demo data only into an empty store.
// Call it outside transactions.
func (db *DB) IsEmpty() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.tables) == 0 && len(db.logs) == 0
}

// Ready returns why the store cannot serve requests now: a save broke off half-way (ErrFailed), or the data
// folder cannot be reached. Call it outside transactions.
func (db *DB) Ready() error {
	db.mu.RLock()
	failed := db.failed
	db.mu.RUnlock()
	if failed {
		return ErrFailed
	}
	if _, err := os.Stat(db.dir); err != nil {
		return fmt.Errorf("store: data folder: %w", err)
	}
	return nil
}
