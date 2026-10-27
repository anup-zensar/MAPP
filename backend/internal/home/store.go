package home

import "context"

// Transactor is all Home needs from storage: it reads its Sources and has no store of its own.
type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
}
