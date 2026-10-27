package settingstest

import (
	"context"

	"mapp/internal/settings"
)

func (s Store) GetFinanceBudget(_ context.Context, id string) (settings.FinanceBudget, bool, error) {
	return s.budgets.Get(id)
}

func (s Store) ListFinanceBudgets(context.Context) ([]settings.FinanceBudget, error) {
	return s.budgets.List()
}

func (s Store) PutFinanceBudget(ctx context.Context, b settings.FinanceBudget) error {
	return s.budgets.Put(ctx, b)
}

func (s Store) DeleteFinanceBudget(ctx context.Context, id string) error {
	return s.budgets.Delete(ctx, id)
}

func (s Store) GetFinanceBudgetItem(_ context.Context, id string) (settings.FinanceBudgetItem, bool, error) {
	return s.items.Get(id)
}

func (s Store) ListFinanceBudgetItems(context.Context) ([]settings.FinanceBudgetItem, error) {
	return s.items.List()
}

func (s Store) ItemsOf(_ context.Context, budgetID string) ([]settings.FinanceBudgetItem, error) {
	return s.items.Where(func(i settings.FinanceBudgetItem) bool { return i.BudgetID == budgetID })
}

func (s Store) PutFinanceBudgetItem(ctx context.Context, i settings.FinanceBudgetItem) error {
	return s.items.Put(ctx, i)
}

func (s Store) DeleteFinanceBudgetItem(ctx context.Context, id string) error {
	return s.items.Delete(ctx, id)
}

func (s Store) GetFieldHelp(_ context.Context, key string) (settings.FieldHelpText, bool, error) {
	return s.help.Get(key)
}

func (s Store) ListFieldHelp(context.Context) ([]settings.FieldHelpText, error) {
	return s.help.List()
}

func (s Store) PutFieldHelp(ctx context.Context, h settings.FieldHelpText) error {
	return s.help.Put(ctx, h)
}

func (s Store) DeleteFieldHelp(ctx context.Context, key string) error {
	return s.help.Delete(ctx, key)
}
