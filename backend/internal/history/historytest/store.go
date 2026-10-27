// Package historytest is the in-memory fake of the activity log and recycle bin store, for tests that build
// services or the whole API without a database.
package historytest

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/fake"
)

type Store struct {
	entries *fake.Table[history.Entry]
	bin     *fake.Table[history.BinEntry]
}

func NewStore(db *fake.DB) Store {
	return Store{
		entries: fake.TableOf(db, "activity_log", func(e history.Entry) string { return e.ID }),
		bin:     fake.TableOf(db, "recycle_bin", func(e history.BinEntry) string { return e.ID }),
	}
}

func (s Store) AppendEntry(ctx context.Context, e history.Entry) error {
	return s.entries.Put(ctx, e)
}

// Entries are in the order they were added: their ids come from one counter.
func (s Store) Entries(context.Context) ([]history.Entry, error) {
	return s.entries.List()
}

func (s Store) PutBinEntry(ctx context.Context, e history.BinEntry) error {
	return s.bin.Put(ctx, e)
}

func (s Store) GetBinEntry(_ context.Context, id string) (history.BinEntry, bool, error) {
	return s.bin.Get(id)
}

func (s Store) BinEntries(context.Context) ([]history.BinEntry, error) {
	return s.bin.List()
}

func (s Store) PurgeBinEntries(ctx context.Context, ids []string, r history.Resolution) error {
	for _, id := range ids {
		e, found, err := s.bin.Get(id)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		e.Resolution, e.Snapshots = &r, []history.Snapshot{}
		if err = s.bin.Put(ctx, e); err != nil {
			return err
		}
	}
	return nil
}
