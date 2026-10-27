package history

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists log and bin entries oldest first, and never changes or removes a log entry.
type Store interface {
	AppendEntry(ctx context.Context, e Entry) error
	Entries(ctx context.Context) ([]Entry, error)
	PutBinEntry(ctx context.Context, e BinEntry) error
	GetBinEntry(ctx context.Context, id string) (BinEntry, bool, error)
	BinEntries(ctx context.Context) ([]BinEntry, error)
	PurgeBinEntries(ctx context.Context, ids []string, r Resolution) error
}
