package financials

import (
	"context"
	"slices"

	"mapp/internal/platform/fake"
)

// fakeStore is financialstest.Store for this package's own tests, which cannot import financialstest.
type fakeStore struct {
	transactions *fake.Table[Transaction]
	imports      *fake.Table[Import]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{
		transactions: fake.TableOf(db, "financial_transactions", func(t Transaction) string { return t.ID }),
		imports:      fake.TableOf(db, "financial_imports", func(i Import) string { return i.ID }),
	}
}

func (s fakeStore) ListTransactions(context.Context) ([]Transaction, error) {
	return s.transactions.List()
}

func (s fakeStore) SpendOnLineItems(_ context.Context, lineItemIDs []string) ([]Transaction, error) {
	return s.transactions.Where(func(t Transaction) bool {
		return t.spend() && slices.Contains(lineItemIDs, t.LineItemID)
	})
}

func (s fakeStore) GetTransaction(_ context.Context, id string) (Transaction, bool, error) {
	return s.transactions.Get(id)
}

func (s fakeStore) PutTransaction(ctx context.Context, t Transaction) error {
	return s.transactions.Put(ctx, t)
}

func (s fakeStore) AddTransactions(ctx context.Context, list []Transaction) error {
	return s.transactions.Put(ctx, list...)
}

func (s fakeStore) DeleteTransaction(ctx context.Context, id string) error {
	return s.transactions.Delete(ctx, id)
}

func (s fakeStore) ListImports(context.Context) ([]Import, error) {
	return s.imports.List()
}

func (s fakeStore) GetImport(_ context.Context, id string) (Import, bool, error) {
	return s.imports.Get(id)
}

func (s fakeStore) PutImport(ctx context.Context, i Import) error {
	return s.imports.Put(ctx, i)
}

func (s fakeStore) DeleteImport(ctx context.Context, id string) error {
	return s.imports.Delete(ctx, id)
}
