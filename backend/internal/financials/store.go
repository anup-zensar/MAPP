package financials

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists by id, the order of adding, never nil; AddTransactions takes new ids only; only history deletes.
type Store interface {
	ListTransactions(ctx context.Context) ([]Transaction, error)
	GetTransaction(ctx context.Context, id string) (Transaction, bool, error)
	PutTransaction(ctx context.Context, t Transaction) error
	AddTransactions(ctx context.Context, list []Transaction) error
	DeleteTransaction(ctx context.Context, id string) error

	ListImports(ctx context.Context) ([]Import, error)
	GetImport(ctx context.Context, id string) (Import, bool, error)
	PutImport(ctx context.Context, i Import) error
	DeleteImport(ctx context.Context, id string) error
}
