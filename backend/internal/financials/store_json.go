package financials

import (
	"context"

	"mapp/internal/platform/store"
)

// The JSON store tables of this package (data model §1–§2).
const (
	transactionsTable = "financial_transactions"
	importsTable      = "financial_imports"
)

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// ListTransactions reads the whole financial_transactions table, in id order.
func (JSONStore) ListTransactions(ctx context.Context) ([]Transaction, error) {
	return store.List[Transaction](ctx, transactionsTable)
}

// GetTransaction reads one transaction.
func (JSONStore) GetTransaction(ctx context.Context, id string) (Transaction, bool, error) {
	return store.Get[Transaction](ctx, transactionsTable, id)
}

func (r JSONStore) AddTransactions(ctx context.Context, list []Transaction) error {
	for _, t := range list {
		if err := r.PutTransaction(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

// PutTransaction saves a transaction under its id.
func (JSONStore) PutTransaction(ctx context.Context, t Transaction) error {
	return store.Put(ctx, transactionsTable, t.ID, t)
}

// DeleteTransaction removes one transaction.
func (JSONStore) DeleteTransaction(ctx context.Context, id string) error {
	return store.Delete(ctx, transactionsTable, id)
}

// ListImports reads the whole financial_imports table, in id order: oldest first.
func (JSONStore) ListImports(ctx context.Context) ([]Import, error) {
	return store.List[Import](ctx, importsTable)
}

// GetImport reads one import.
func (JSONStore) GetImport(ctx context.Context, id string) (Import, bool, error) {
	return store.Get[Import](ctx, importsTable, id)
}

// PutImport saves an import under its id.
func (JSONStore) PutImport(ctx context.Context, i Import) error {
	return store.Put(ctx, importsTable, i.ID, i)
}

// DeleteImport removes one import.
func (JSONStore) DeleteImport(ctx context.Context, id string) error {
	return store.Delete(ctx, importsTable, id)
}
