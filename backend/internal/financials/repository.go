package financials

import "context"

// Repository stores GSAP transactions and imports (data model §1–§2). The JSON store implements it now
// (store_json.go) and PostgreSQL can later. Every method runs inside the caller's transaction, which the
// context carries. Lists are ordered by id — the order the records were added in — and never nil.
//
// These are demo-sized tables: the service reads them whole and indexes them in memory per request
// (index.go). A PostgreSQL repository adds the matching SQL indexes instead.
type Repository interface {
	// ListTransactions returns every transaction.
	ListTransactions(ctx context.Context) ([]Transaction, error)
	// GetTransaction returns the transaction with id, and false when there is none.
	GetTransaction(ctx context.Context, id string) (Transaction, bool, error)
	// PutTransaction saves t, replacing the transaction with the same id.
	PutTransaction(ctx context.Context, t Transaction) error
	// DeleteTransaction removes the transaction with id. Only history's Kind uses it (Remove): no feature
	// deletes a transaction (FR-002).
	DeleteTransaction(ctx context.Context, id string) error

	// ListImports returns every import, oldest first.
	ListImports(ctx context.Context) ([]Import, error)
	// GetImport returns the import with id, and false when there is none.
	GetImport(ctx context.Context, id string) (Import, bool, error)
	// PutImport saves i, replacing the import with the same id.
	PutImport(ctx context.Context, i Import) error
	// DeleteImport removes the import with id. Only history's Kind uses it (Remove).
	DeleteImport(ctx context.Context, id string) error
}
