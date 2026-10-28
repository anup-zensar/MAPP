package swbs

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists runs newest first, at most limit, with how many there are in all.
type Store interface {
	AddRun(ctx context.Context, run Run) error
	GetRun(ctx context.Context, id string) (Run, bool, error)
	DeleteRun(ctx context.Context, id string) error
	ListRuns(ctx context.Context, limit int) ([]Run, int, error)
	Issued
}
