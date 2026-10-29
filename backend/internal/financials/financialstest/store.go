// Package financialstest is the in-memory fake of the GSAP transactions and imports store, for tests that
// build services or the whole API without a database.
package financialstest

import (
	"context"
	"slices"

	"mapp/internal/financials"
	"mapp/internal/platform/fake"
)

type Store struct {
	transactions *fake.Table[financials.Transaction]
	imports      *fake.Table[financials.Import]
}

func NewStore(db *fake.DB) Store {
	return Store{
		transactions: fake.TableOf(db, "financial_transactions", func(t financials.Transaction) string { return t.ID }),
		imports:      fake.TableOf(db, "financial_imports", func(i financials.Import) string { return i.ID }),
	}
}

func (s Store) ListTransactions(context.Context) ([]financials.Transaction, error) {
	return s.transactions.List()
}

func (s Store) PaymentsOnLineItems(_ context.Context, lineItemIDs []string) ([]financials.Transaction, error) {
	return s.transactions.Where(func(t financials.Transaction) bool {
		return t.Type == financials.TypePayment && slices.Contains(lineItemIDs, t.LineItemID)
	})
}

func (s Store) GetTransaction(_ context.Context, id string) (financials.Transaction, bool, error) {
	return s.transactions.Get(id)
}

func (s Store) PutTransaction(ctx context.Context, t financials.Transaction) error {
	return s.transactions.Put(ctx, t)
}

func (s Store) AddTransactions(ctx context.Context, list []financials.Transaction) error {
	return s.transactions.Put(ctx, list...)
}

func (s Store) DeleteTransaction(ctx context.Context, id string) error {
	return s.transactions.Delete(ctx, id)
}

func (s Store) ListImports(context.Context) ([]financials.Import, error) {
	return s.imports.List()
}

func (s Store) GetImport(_ context.Context, id string) (financials.Import, bool, error) {
	return s.imports.Get(id)
}

func (s Store) PutImport(ctx context.Context, i financials.Import) error {
	return s.imports.Put(ctx, i)
}

func (s Store) DeleteImport(ctx context.Context, id string) error {
	return s.imports.Delete(ctx, id)
}
