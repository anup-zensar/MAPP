package history

import "context"

// Repository stores the activity log and the recycle bin. The JSON store implements it now (store_json.go)
// and PostgreSQL will later, without any change to the service. Every method runs inside the caller's
// transaction, which the context carries.
//
// There is no method to change or remove a log entry: entries are only ever added (FR-023).
type Repository interface {
	// AppendEntry adds e at the end of the activity log.
	AppendEntry(ctx context.Context, e Entry) error
	// Entries returns every activity-log entry, in the order they were added.
	Entries(ctx context.Context) ([]Entry, error)
	// PutBinEntry saves e, replacing the recycle-bin entry with the same id.
	PutBinEntry(ctx context.Context, e BinEntry) error
	// GetBinEntry returns the recycle-bin entry with id, and false when there is none.
	GetBinEntry(ctx context.Context, id string) (BinEntry, bool, error)
	// BinEntries returns every recycle-bin entry, resolved or not, in the order they were made.
	BinEntries(ctx context.Context) ([]BinEntry, error)
}
