package settings

import (
	"context"
	"fmt"

	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/memo"
)

// The store reads each Settings table once per transaction and keeps its rows (memo): pages read Settings
// for every record they show, and Settings change only through this store, whose changes forget them.
const keptArea = "settings"

func (s *PostgresStore) change(ctx context.Context) *sqlcdb.Queries {
	memo.Forget(ctx, keptArea)
	return s.db.Queries(ctx)
}

func keptTable[R any](ctx context.Context, name string, read func(ctx context.Context) ([]R, error),
	idOf func(R) string) (memo.Table[R], error) {
	table, err := memo.GetTable(ctx, keptArea+":"+name, read, idOf)
	if err != nil {
		return memo.Table[R]{}, fmt.Errorf("select %s: %w", name, err)
	}
	return table, nil
}

func (s *PostgresStore) models(ctx context.Context) (memo.Table[sqlcdb.ListModelsRow], error) {
	return keptTable(ctx, "models", func(ctx context.Context) ([]sqlcdb.ListModelsRow, error) {
		return s.db.Queries(ctx).ListModels(ctx)
	}, func(r sqlcdb.ListModelsRow) string { return r.Model.ID })
}

func (s *PostgresStore) activityTypes(ctx context.Context) (memo.Table[sqlcdb.ActivityType], error) {
	return keptTable(ctx, "activity types", func(ctx context.Context) ([]sqlcdb.ActivityType, error) {
		return s.db.Queries(ctx).ListActivityTypes(ctx)
	}, func(r sqlcdb.ActivityType) string { return r.ID })
}

func (s *PostgresStore) currencies(ctx context.Context) (memo.Table[sqlcdb.Currency], error) {
	return keptTable(ctx, "currencies", func(ctx context.Context) ([]sqlcdb.Currency, error) {
		return s.db.Queries(ctx).ListCurrencies(ctx)
	}, func(r sqlcdb.Currency) string { return r.Code })
}

func (s *PostgresStore) rates(ctx context.Context) (memo.Table[sqlcdb.ExchangeRate], error) {
	return keptTable(ctx, "exchange rates", func(ctx context.Context) ([]sqlcdb.ExchangeRate, error) {
		return s.db.Queries(ctx).ListRates(ctx)
	}, func(r sqlcdb.ExchangeRate) string { return r.ID })
}

func (s *PostgresStore) rateImports(ctx context.Context) (memo.Table[sqlcdb.RateImport], error) {
	return keptTable(ctx, "rate imports", func(ctx context.Context) ([]sqlcdb.RateImport, error) {
		return s.db.Queries(ctx).ListRateImports(ctx)
	}, func(r sqlcdb.RateImport) string { return r.ID })
}

func (s *PostgresStore) financeBudgets(ctx context.Context) (memo.Table[sqlcdb.FinanceBudget], error) {
	return keptTable(ctx, "finance budgets", func(ctx context.Context) ([]sqlcdb.FinanceBudget, error) {
		return s.db.Queries(ctx).ListFinanceBudgets(ctx)
	}, func(r sqlcdb.FinanceBudget) string { return r.ID })
}

func (s *PostgresStore) financeBudgetItemRows(ctx context.Context) (memo.Table[sqlcdb.FinanceBudgetItem], error) {
	return keptTable(ctx, "finance budget items", func(ctx context.Context) ([]sqlcdb.FinanceBudgetItem, error) {
		return s.db.Queries(ctx).ListFinanceBudgetItems(ctx, nil)
	}, func(r sqlcdb.FinanceBudgetItem) string { return r.ID })
}

func (s *PostgresStore) attributes(ctx context.Context) (memo.Table[sqlcdb.ListAttributesRow], error) {
	return keptTable(ctx, "Shell attributes", func(ctx context.Context) ([]sqlcdb.ListAttributesRow, error) {
		return s.db.Queries(ctx).ListAttributes(ctx)
	}, func(r sqlcdb.ListAttributesRow) string { return r.ShellAttribute.ID })
}

func (s *PostgresStore) optionRows(ctx context.Context) (memo.Table[sqlcdb.ListOptionsRow], error) {
	return keptTable(ctx, "options", func(ctx context.Context) ([]sqlcdb.ListOptionsRow, error) {
		return s.db.Queries(ctx).ListOptions(ctx, nil)
	}, func(r sqlcdb.ListOptionsRow) string { return r.ShellAttributeOption.ID })
}
