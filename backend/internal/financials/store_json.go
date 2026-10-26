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

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// ListTransactions reads the whole financial_transactions table, in id order.
func (jsonRepository) ListTransactions(ctx context.Context) ([]Transaction, error) {
	return store.List[Transaction](ctx, transactionsTable)
}

// GetTransaction reads one transaction.
func (jsonRepository) GetTransaction(ctx context.Context, id string) (Transaction, bool, error) {
	return store.Get[Transaction](ctx, transactionsTable, id)
}

func (r jsonRepository) AddTransactions(ctx context.Context, list []Transaction) error {
	for _, t := range list {
		if err := r.PutTransaction(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

// PutTransaction saves a transaction under its id.
func (jsonRepository) PutTransaction(ctx context.Context, t Transaction) error {
	return store.Put(ctx, transactionsTable, t.ID, t)
}

// DeleteTransaction removes one transaction.
func (jsonRepository) DeleteTransaction(ctx context.Context, id string) error {
	return store.Delete(ctx, transactionsTable, id)
}

// ListImports reads the whole financial_imports table, in id order: oldest first.
func (jsonRepository) ListImports(ctx context.Context) ([]Import, error) {
	return store.List[Import](ctx, importsTable)
}

// GetImport reads one import.
func (jsonRepository) GetImport(ctx context.Context, id string) (Import, bool, error) {
	return store.Get[Import](ctx, importsTable, id)
}

// PutImport saves an import under its id.
func (jsonRepository) PutImport(ctx context.Context, i Import) error {
	return store.Put(ctx, importsTable, i.ID, i)
}

// DeleteImport removes one import.
func (jsonRepository) DeleteImport(ctx context.Context, id string) error {
	return store.Delete(ctx, importsTable, id)
}
