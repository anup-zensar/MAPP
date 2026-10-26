package history

import (
	"context"

	"mapp/internal/platform/store"
)

// The JSON store tables of this package.
const (
	logTable = "activity_log" // append-only: saved as activity_log.jsonl, one entry per line (research R3)
	binTable = "recycle_bin"
)

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// AppendEntry adds an entry at the end of the append-only activity_log table.
func (jsonRepository) AppendEntry(ctx context.Context, e Entry) error {
	return store.Append(ctx, logTable, e)
}

// Entries reads the activity_log table, in the order the entries were added.
func (jsonRepository) Entries(ctx context.Context) ([]Entry, error) {
	return store.Scan[Entry](ctx, logTable)
}

// PutBinEntry saves an entry in the recycle_bin table under its id.
func (jsonRepository) PutBinEntry(ctx context.Context, e BinEntry) error {
	return store.Put(ctx, binTable, e.ID, e)
}

// GetBinEntry reads one entry from the recycle_bin table.
func (jsonRepository) GetBinEntry(ctx context.Context, id string) (BinEntry, bool, error) {
	return store.Get[BinEntry](ctx, binTable, id)
}

// BinEntries reads the whole recycle_bin table. The store lists records by id (BIN-999 before BIN-1000),
// and the ids are given out in order, so this is the order the entries were made in.
func (jsonRepository) BinEntries(ctx context.Context) ([]BinEntry, error) {
	return store.List[BinEntry](ctx, binTable)
}

func (r jsonRepository) PurgeBinEntries(ctx context.Context, ids []string, resolution Resolution) error {
	for _, id := range ids {
		e, found, err := r.GetBinEntry(ctx, id)
		if err != nil || !found {
			return err
		}
		e.Resolution, e.Snapshots = &resolution, []Snapshot{}
		if err := r.PutBinEntry(ctx, e); err != nil {
			return err
		}
	}
	return nil
}
