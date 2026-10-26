package lineitems

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// PostgresStore is the Repository on PostgreSQL. ListInScope and GetInScope keep only the line items of the
// activities the person acting may see: whoever sees an activity sees its line items.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, id string) (LineItem, bool, error) {
	return firstLineItem(s.list(ctx, sqlcdb.ListLineItemsParams{ID: &id}))
}

func (s *PostgresStore) List(ctx context.Context) ([]LineItem, error) {
	return s.list(ctx, sqlcdb.ListLineItemsParams{})
}

func (s *PostgresStore) ListByActivity(ctx context.Context, activityID string) ([]LineItem, error) {
	return s.list(ctx, sqlcdb.ListLineItemsParams{ActivityIds: []string{activityID}})
}

func (s *PostgresStore) ListByActivities(ctx context.Context, activityIDs []string) ([]LineItem, error) {
	return s.list(ctx, sqlcdb.ListLineItemsParams{ActivityIds: append([]string{}, activityIDs...)})
}

func (s *PostgresStore) ListInScope(ctx context.Context, scope authz.Scope) ([]LineItem, error) {
	return s.listInScope(ctx, scope, nil)
}

func (s *PostgresStore) GetInScope(ctx context.Context, scope authz.Scope, id string) (LineItem, bool, error) {
	return firstLineItem(s.listInScope(ctx, scope, &id))
}

func (s *PostgresStore) Put(ctx context.Context, li LineItem) error {
	attributes, err := li.Attributes.Column()
	if err != nil {
		return fmt.Errorf("encode line item %s: %w", li.ID, err)
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
	months, amounts := make([]string, 0, len(li.Spread.Months)), make([]int64, 0, len(li.Spread.Months))
	for _, m := range li.Spread.Months {
		months, amounts = append(months, m.Month), append(amounts, m.Amount.Cents())
	}
	q := s.db.Queries(ctx)
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

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteLineItem(ctx, id); err != nil {
		return fmt.Errorf("delete line item %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) list(ctx context.Context, filter sqlcdb.ListLineItemsParams) ([]LineItem, error) {
	rows, err := s.db.Queries(ctx).ListLineItems(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("select line items: %w", err)
	}
	return lineItemsFromRows(rows)
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
	return lineItemsFromRows(listRows)
}

func firstLineItem(list []LineItem, err error) (LineItem, bool, error) {
	if err != nil || len(list) == 0 {
		return LineItem{}, false, err
	}
	return list[0], true, nil
}

func lineItemsFromRows(rows []sqlcdb.ListLineItemsRow) ([]LineItem, error) {
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
