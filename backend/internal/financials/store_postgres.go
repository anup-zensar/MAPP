package financials

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// PostgresStore is the Store on PostgreSQL. A transaction's line item and currency are plain values,
// as GSAP sent them: the line item may be in the recycle bin and the currency deleted since.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) ListTransactions(ctx context.Context) ([]Transaction, error) {
	rows, err := s.db.Queries(ctx).ListTransactions(ctx, nil)
	return transactionsFromRows(rows, err)
}

func (s *PostgresStore) GetTransaction(ctx context.Context, id string) (Transaction, bool, error) {
	rows, err := s.db.Queries(ctx).ListTransactions(ctx, &id)
	list, err := transactionsFromRows(rows, err)
	if err != nil || len(list) == 0 {
		return Transaction{}, false, err
	}
	return list[0], true, nil
}

func (s *PostgresStore) PaymentsOnLineItems(ctx context.Context, lineItemIDs []string) ([]Transaction, error) {
	rows, err := s.db.Queries(ctx).ListPaymentsOnLineItems(ctx, lineItemIDs)
	return transactionsFromRows(rows, err)
}

// TransactionsInScope keeps what the person may see (FR-028, D-301): every linked transaction with
// finance.view in All markets, else one whose stored line item's plan they see and hold finance.view in; a
// not-linked one only when they may import.
func (s *PostgresStore) TransactionsInScope(ctx context.Context, plans, finance authz.Scope, canImport bool) ([]Transaction, error) {
	rows, err := s.db.Queries(ctx).ListTransactionsInScope(ctx, sqlcdb.ListTransactionsInScopeParams{
		CanImport: canImport, FinanceEverything: finance.Everything, SeesEverything: plans.Everything,
		ViewerID: plans.PersonID, ViewsPlans: plans.Holds, ViewerTeamIds: plans.TeamIDs,
		HoldsFinance: finance.Holds, FinanceTeamIds: finance.TeamIDs,
	})
	return transactionsFromRows(rows, err)
}

func (s *PostgresStore) PutTransaction(ctx context.Context, t Transaction) error {
	if err := s.db.Queries(ctx).PutTransaction(ctx, transactionParams(t)); err != nil {
		return fmt.Errorf("save financial transaction %s: %w", t.ID, err)
	}
	return nil
}

// AddTransactions stores new transactions in one statement whatever their number.
func (s *PostgresStore) AddTransactions(ctx context.Context, list []Transaction) error {
	if len(list) == 0 {
		return nil
	}
	rows := make([]sqlcdb.AddTransactionsParams, 0, len(list))
	for _, t := range list {
		rows = append(rows, sqlcdb.AddTransactionsParams(transactionParams(t)))
	}
	if _, err := s.db.Queries(ctx).AddTransactions(ctx, rows); err != nil {
		return fmt.Errorf("add financial transactions: %w", err)
	}
	return nil
}

func transactionParams(t Transaction) sqlcdb.PutTransactionParams {
	params := sqlcdb.PutTransactionParams{
		ID: t.ID, Type: string(t.Type), DocumentNumber: t.DocumentNumber, LineNumber: t.LineNumber,
		PoNumber: t.PONumber, PoLineNumber: t.POLineNumber, PostingDate: t.PostingDate.Time(),
		AccrualMonth: database.NullIfEmpty(t.AccrualMonth), AmountMinor: t.Amount.Cents(), Currency: t.Currency,
		VendorCode: t.VendorCode, VendorName: t.VendorName, Description: t.Description, SwbsCode: t.SwbsCode,
		TrackingKey: t.TrackingKey, SentLineItemID: t.SentLineItemID, LineItemID: database.NullIfEmpty(t.LineItemID),
		MatchedBy: string(t.MatchedBy), NotLinkedReason: t.NotLinkedReason, ImportID: t.ImportID, Version: t.Version,
		CreatedAt: t.CreatedAt, CreatedByID: t.CreatedBy.ID, CreatedByName: t.CreatedBy.Name, UpdatedAt: t.UpdatedAt,
		UpdatedByID: t.UpdatedBy.ID, UpdatedByName: t.UpdatedBy.Name,
	}
	if link := t.ManualLink; link != nil {
		params.ManualLinkByID, params.ManualLinkByName, params.ManualLinkAt = &link.By.ID, &link.By.Name, &link.At
	}
	return params
}

func (s *PostgresStore) DeleteTransaction(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteTransaction(ctx, id); err != nil {
		return fmt.Errorf("delete financial transaction %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) ListImports(ctx context.Context) ([]Import, error) {
	rows, err := s.db.Queries(ctx).ListFinancialImports(ctx, nil)
	return importsFromRows(rows, err)
}

func (s *PostgresStore) GetImport(ctx context.Context, id string) (Import, bool, error) {
	rows, err := s.db.Queries(ctx).ListFinancialImports(ctx, &id)
	list, err := importsFromRows(rows, err)
	if err != nil || len(list) == 0 {
		return Import{}, false, err
	}
	return list[0], true, nil
}

func (s *PostgresStore) PutImport(ctx context.Context, i Import) error {
	err := s.db.Queries(ctx).PutFinancialImport(ctx, sqlcdb.PutFinancialImportParams{
		ID: i.ID, Source: string(i.Source), FeedName: i.FeedName, FileName: i.FileName, Rows: i.Rows, New: i.New,
		Updated: i.Updated, Unchanged: i.Unchanged, Refused: i.Refused, NotLinked: i.NotLinked, Relinked: i.Relinked,
		Version: i.Version, CreatedAt: i.CreatedAt, CreatedByID: i.CreatedBy.ID, CreatedByName: i.CreatedBy.Name,
		UpdatedAt: i.UpdatedAt, UpdatedByID: i.UpdatedBy.ID, UpdatedByName: i.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save financial import %s: %w", i.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteImport(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteFinancialImport(ctx, id); err != nil {
		return fmt.Errorf("delete financial import %s: %w", id, err)
	}
	return nil
}

func transactionsFromRows(rows []sqlcdb.FinancialTransaction, err error) ([]Transaction, error) {
	if err != nil {
		return nil, fmt.Errorf("select financial transactions: %w", err)
	}
	list := make([]Transaction, 0, len(rows))
	for _, r := range rows {
		t := Transaction{
			ID: r.ID,
			Fields: Fields{
				Type: Type(r.Type), DocumentNumber: r.DocumentNumber, LineNumber: r.LineNumber, PONumber: r.PoNumber,
				POLineNumber: r.PoLineNumber, PostingDate: date.Today(r.PostingDate),
				AccrualMonth: database.EmptyIfNull(r.AccrualMonth), Amount: money.FromCents(r.AmountMinor),
				Currency: r.Currency, VendorCode: r.VendorCode, VendorName: r.VendorName, Description: r.Description,
				SwbsCode: r.SwbsCode, TrackingKey: r.TrackingKey, SentLineItemID: r.SentLineItemID,
			},
			LineItemID: database.EmptyIfNull(r.LineItemID), MatchedBy: MatchedBy(r.MatchedBy),
			NotLinkedReason: r.NotLinkedReason, ImportID: r.ImportID,
			Meta: record.Meta{
				Version:   r.Version,
				CreatedAt: r.CreatedAt,
				CreatedBy: actor.Actor{ID: r.CreatedByID, Name: r.CreatedByName},
				UpdatedAt: r.UpdatedAt,
				UpdatedBy: actor.Actor{ID: r.UpdatedByID, Name: r.UpdatedByName},
			},
		}
		if r.ManualLinkAt != nil && r.ManualLinkByID != nil && r.ManualLinkByName != nil {
			t.ManualLink = &ManualLink{By: actor.Actor{ID: *r.ManualLinkByID, Name: *r.ManualLinkByName}, At: *r.ManualLinkAt}
		}
		list = append(list, t)
	}
	return list, nil
}

func importsFromRows(rows []sqlcdb.FinancialImport, err error) ([]Import, error) {
	if err != nil {
		return nil, fmt.Errorf("select financial imports: %w", err)
	}
	list := make([]Import, 0, len(rows))
	for _, r := range rows {
		list = append(list, Import{
			ID: r.ID, Source: Source(r.Source), FeedName: r.FeedName, FileName: r.FileName, Rows: r.Rows, New: r.New,
			Updated: r.Updated, Unchanged: r.Unchanged, Refused: r.Refused, NotLinked: r.NotLinked, Relinked: r.Relinked,
			Meta: record.Meta{
				Version:   r.Version,
				CreatedAt: r.CreatedAt,
				CreatedBy: actor.Actor{ID: r.CreatedByID, Name: r.CreatedByName},
				UpdatedAt: r.UpdatedAt,
				UpdatedBy: actor.Actor{ID: r.UpdatedByID, Name: r.UpdatedByName},
			},
		})
	}
	return list, nil
}
