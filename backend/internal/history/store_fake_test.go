package history

import (
	"context"

	"mapp/internal/platform/fake"
)

// fakeStore is historytest.Store for this package's own tests, which cannot import historytest.
type fakeStore struct {
	entries *fake.Table[Entry]
	bin     *fake.Table[BinEntry]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{
		entries: fake.TableOf(db, "activity_log", func(e Entry) string { return e.ID }),
		bin:     fake.TableOf(db, "recycle_bin", func(e BinEntry) string { return e.ID }),
	}
}

func (s fakeStore) AppendEntry(ctx context.Context, e Entry) error {
	return s.entries.Put(ctx, e)
}

func (s fakeStore) Entries(context.Context) ([]Entry, error) {
	return s.entries.List()
}

func (s fakeStore) PutBinEntry(ctx context.Context, e BinEntry) error {
	return s.bin.Put(ctx, e)
}

func (s fakeStore) GetBinEntry(_ context.Context, id string) (BinEntry, bool, error) {
	return s.bin.Get(id)
}

func (s fakeStore) BinEntries(context.Context) ([]BinEntry, error) {
	return s.bin.List()
}

func (s fakeStore) PurgeBinEntries(ctx context.Context, ids []string, r Resolution) error {
	for _, id := range ids {
		e, found, err := s.bin.Get(id)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		e.Resolution, e.Snapshots = &r, []Snapshot{}
		if err = s.bin.Put(ctx, e); err != nil {
			return err
		}
	}
	return nil
}
