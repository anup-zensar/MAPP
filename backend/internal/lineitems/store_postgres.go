package lineitems

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/memo"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// keptArea is what the transaction memo keeps of line items.
const keptArea = "lineitems"

// PostgresStore is the Store on PostgreSQL. ListInScope and GetInScope keep only the line items of the
// activities the person acting may see: whoever sees an activity sees its line items.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, id string) (LineItem, bool, error) {
	return firstLineItem(lineItemsFromRows(s.records().ByID(ctx, []string{id})))
}

func (s *PostgresStore) List(ctx context.Context) ([]LineItem, error) {
	return s.list(ctx, sqlcdb.ListLineItemsParams{})
}

func (s *PostgresStore) ListByActivity(ctx context.Context, activityID string) ([]LineItem, error) {
	return lineItemsFromRows(s.records().ByParent(ctx, []string{activityID}))
}

func (s *PostgresStore) ListByActivities(ctx context.Context, activityIDs []string) ([]LineItem, error) {
	return lineItemsFromRows(s.records().ByParent(ctx, activityIDs))
}

func (s *PostgresStore) ListInScope(ctx context.Context, scope authz.Scope) ([]LineItem, error) {
	return s.listInScope(ctx, scope, nil)
}

func (s *PostgresStore) GetInScope(ctx context.Context, scope authz.Scope, id string) (LineItem, bool, error) {
	return firstLineItem(s.listInScope(ctx, scope, &id))
}

func (s *PostgresStore) Put(ctx context.Context, li LineItem) error {
	params, err := lineItemParams(li)
	if err != nil {
		return err
	}
	months, amounts := spreadColumns(li)
	q := s.change(ctx)
	err = q.PutLineItem(ctx, params)
	if err == nil {
		err = q.DeleteLineItemSpreadMonths(ctx, li.ID)
	}
	if err == nil {
		err = q.InsertLineItemSpreadMonths(ctx, sqlcdb.InsertLineItemSpreadMonthsParams{LineItemID: li.ID, Months: months, Amounts: amounts})
	}
	if err != nil {
		return fmt.Errorf("save line item %s: %w", li.ID, err)
	}
	return nil
}

// AddMany stores new line items, each with its spread months, in two statements whatever their number.
func (s *PostgresStore) AddMany(ctx context.Context, list []LineItem) error {
	if len(list) == 0 {
		return nil
	}
	rows := make([]sqlcdb.AddLineItemsParams, 0, len(list))
	var spread []sqlcdb.AddLineItemSpreadMonthsParams
	for _, li := range list {
		params, err := lineItemParams(li)
		if err != nil {
			return err
		}
		rows = append(rows, sqlcdb.AddLineItemsParams(params))
		for _, m := range li.Spread.Months {
			spread = append(spread, sqlcdb.AddLineItemSpreadMonthsParams{LineItemID: li.ID, Month: m.Month, AmountMinor: m.Amount.Cents()})
		}
	}
	q := s.change(ctx)
	if _, err := q.AddLineItems(ctx, rows); err != nil {
		return fmt.Errorf("add line items: %w", err)
	}
	if len(spread) == 0 {
		return nil
	}
	if _, err := q.AddLineItemSpreadMonths(ctx, spread); err != nil {
		return fmt.Errorf("add line item spreads: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteMany(ctx context.Context, ids []string) error {
	if err := s.change(ctx).DeleteLineItems(ctx, ids); err != nil {
		return fmt.Errorf("delete line items: %w", err)
	}
	return nil
}

func lineItemParams(li LineItem) (sqlcdb.PutLineItemParams, error) {
	attributes, err := li.Attributes.Column()
	if err != nil {
		return sqlcdb.PutLineItemParams{}, fmt.Errorf("encode line item %s: %w", li.ID, err)
	}
	params := sqlcdb.PutLineItemParams{
		ID: li.ID, ActivityID: li.ActivityID, Name: li.Name, OwnerID: database.NullIfEmpty(li.OwnerID),
		Description: li.Description, Currency: li.Currency, PlannedCostMinor: li.PlannedCost.Cents(),
		SpreadMethod: string(li.Spread.Method), Attributes: attributes, RequestSwbs: li.RequestSwbs,
		FinanceBudgetYear: li.FinanceBudgetYear, FinanceBudgetItemID: database.NullIfEmpty(li.FinanceBudgetItemID),
		Version: li.Version, CreatedAt: li.CreatedAt, CreatedByID: li.CreatedBy.ID, CreatedByName: li.CreatedBy.Name,
		UpdatedAt: li.UpdatedAt, UpdatedByID: li.UpdatedBy.ID, UpdatedByName: li.UpdatedBy.Name,
	}
	if li.ContractedAmount != nil {
		cents := li.ContractedAmount.Cents()
		params.ContractedAmountMinor = &cents
	}
	return params, nil
}

func spreadColumns(li LineItem) (months []string, amounts []int64) {
	months, amounts = make([]string, 0, len(li.Spread.Months)), make([]int64, 0, len(li.Spread.Months))
	for _, m := range li.Spread.Months {
		months, amounts = append(months, m.Month), append(amounts, m.Amount.Cents())
	}
	return months, amounts
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteLineItem(ctx, id); err != nil {
		return fmt.Errorf("delete line item %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) list(ctx context.Context, filter sqlcdb.ListLineItemsParams) ([]LineItem, error) {
	rows, err := s.db.Queries(ctx).ListLineItems(ctx, filter)
	s.records().Keep(ctx, rows)
	return lineItemsFromRows(rows, err)
}

// records keeps the line items a transaction reads, by id and by activity (memo). A change forgets them.
func (s *PostgresStore) records() memo.Records[sqlcdb.ListLineItemsRow] {
	return memo.Records[sqlcdb.ListLineItemsRow]{
		Area: keptArea, ID: func(r sqlcdb.ListLineItemsRow) string { return r.LineItem.ID },
		Parent: func(r sqlcdb.ListLineItemsRow) string { return r.LineItem.ActivityID },
		Read: func(ctx context.Context, ids, activityIDs []string) ([]sqlcdb.ListLineItemsRow, error) {
			return s.db.Queries(ctx).ListLineItems(ctx, sqlcdb.ListLineItemsParams{Ids: ids, ActivityIds: activityIDs})
		},
	}
}

func (s *PostgresStore) change(ctx context.Context) *sqlcdb.Queries {
	memo.Forget(ctx, keptArea)
	return s.db.Queries(ctx)
}

func (s *PostgresStore) listInScope(ctx context.Context, scope authz.Scope, id *string) ([]LineItem, error) {
	rows, err := s.db.Queries(ctx).ListLineItemsInScope(ctx, sqlcdb.ListLineItemsInScopeParams{
		ID: id, SeesEverything: scope.Everything, ViewerID: scope.PersonID, ViewsPlans: scope.Holds,
		ViewerTeamIds: scope.TeamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("select line items in scope: %w", err)
	}
	listRows := make([]sqlcdb.ListLineItemsRow, 0, len(rows))
	for _, row := range rows {
		listRows = append(listRows, sqlcdb.ListLineItemsRow(row))
	}
	s.records().Keep(ctx, listRows)
	return lineItemsFromRows(listRows, nil)
}

func firstLineItem(list []LineItem, err error) (LineItem, bool, error) {
	if err != nil || len(list) == 0 {
		return LineItem{}, false, err
	}
	return list[0], true, nil
}

func lineItemsFromRows(rows []sqlcdb.ListLineItemsRow, err error) ([]LineItem, error) {
	if err != nil {
		return nil, fmt.Errorf("select line items: %w", err)
	}
	list := make([]LineItem, 0, len(rows))
	for _, row := range rows {
		r := row.LineItem
		attributes, err := settings.AttributeValuesOf(r.Attributes)
		if err != nil {
			return nil, fmt.Errorf("decode line item %s: %w", r.ID, err)
		}
		li := LineItem{
			ID: r.ID, ActivityID: r.ActivityID, Name: r.Name, OwnerID: database.EmptyIfNull(r.OwnerID),
			Description: r.Description, Currency: r.Currency, PlannedCost: money.FromCents(r.PlannedCostMinor),
			Spread:     Spread{Method: SpreadMethod(r.SpreadMethod), Months: make([]MonthAmount, 0, len(row.SpreadMonths))},
			Attributes: attributes, RequestSwbs: r.RequestSwbs, FinanceBudgetYear: r.FinanceBudgetYear,
			FinanceBudgetItemID: database.EmptyIfNull(r.FinanceBudgetItemID),
			Meta: record.Meta{
				Version:   r.Version,
				CreatedAt: r.CreatedAt,
				CreatedBy: actor.Actor{ID: r.CreatedByID, Name: r.CreatedByName},
				UpdatedAt: r.UpdatedAt,
				UpdatedBy: actor.Actor{ID: r.UpdatedByID, Name: r.UpdatedByName},
			},
		}
		for i, month := range row.SpreadMonths {
			li.Spread.Months = append(li.Spread.Months, MonthAmount{Month: month, Amount: money.FromCents(row.SpreadAmounts[i])})
		}
		if r.ContractedAmountMinor != nil {
			contracted := money.FromCents(*r.ContractedAmountMinor)
			li.ContractedAmount = &contracted
		}
		list = append(list, li)
	}
	return list, nil
}
