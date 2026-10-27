package settings

import (
	"context"

	"mapp/internal/platform/store"
)

// The JSON store tables of this package (data model §2–§8).
const (
	modelsTable             = "models"
	activityTypesTable      = "activity_types"
	attributesTable         = "shell_attributes"
	optionsTable            = "shell_attribute_options"
	currenciesTable         = "currencies"
	financeBudgetsTable     = "finance_budgets"
	financeBudgetItemsTable = "finance_budget_items"
)

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// GetModel reads one model.
func (JSONStore) GetModel(ctx context.Context, id string) (Model, bool, error) {
	return store.Get[Model](ctx, modelsTable, id)
}

// ListModels reads the whole models table.
func (JSONStore) ListModels(ctx context.Context) ([]Model, error) {
	return store.List[Model](ctx, modelsTable)
}

// PutModel saves a model under its id.
func (JSONStore) PutModel(ctx context.Context, m Model) error {
	return store.Put(ctx, modelsTable, m.ID, m)
}

// DeleteModel removes one model.
func (JSONStore) DeleteModel(ctx context.Context, id string) error {
	return store.Delete(ctx, modelsTable, id)
}

// GetActivityType reads one activity type.
func (JSONStore) GetActivityType(ctx context.Context, id string) (ActivityType, bool, error) {
	return store.Get[ActivityType](ctx, activityTypesTable, id)
}

// ListActivityTypes reads the whole activity_types table.
func (JSONStore) ListActivityTypes(ctx context.Context) ([]ActivityType, error) {
	return store.List[ActivityType](ctx, activityTypesTable)
}

// PutActivityType saves an activity type under its id.
func (JSONStore) PutActivityType(ctx context.Context, t ActivityType) error {
	return store.Put(ctx, activityTypesTable, t.ID, t)
}

// DeleteActivityType removes one activity type.
func (JSONStore) DeleteActivityType(ctx context.Context, id string) error {
	return store.Delete(ctx, activityTypesTable, id)
}

// GetAttribute reads one Shell attribute.
func (JSONStore) GetAttribute(ctx context.Context, id string) (Attribute, bool, error) {
	return store.Get[Attribute](ctx, attributesTable, id)
}

// ListAttributes reads the whole shell_attributes table.
func (JSONStore) ListAttributes(ctx context.Context) ([]Attribute, error) {
	return store.List[Attribute](ctx, attributesTable)
}

// PutAttribute saves a Shell attribute under its id.
func (JSONStore) PutAttribute(ctx context.Context, a Attribute) error {
	return store.Put(ctx, attributesTable, a.ID, a)
}

// DeleteAttribute removes one Shell attribute.
func (JSONStore) DeleteAttribute(ctx context.Context, id string) error {
	return store.Delete(ctx, attributesTable, id)
}

// GetOption reads one option.
func (JSONStore) GetOption(ctx context.Context, id string) (Option, bool, error) {
	return store.Get[Option](ctx, optionsTable, id)
}

// ListOptions reads the whole shell_attribute_options table.
func (JSONStore) ListOptions(ctx context.Context) ([]Option, error) {
	return store.List[Option](ctx, optionsTable)
}

// OptionsOf reads every option to find those of one attribute: quick enough for the 5,000 options an
// attribute may have. PostgreSQL would use an index.
func (r JSONStore) OptionsOf(ctx context.Context, attributeID string) ([]Option, error) {
	all, err := r.ListOptions(ctx)
	if err != nil {
		return nil, err
	}
	return filter(all, func(o Option) bool { return o.AttributeID == attributeID }), nil
}

// PutOption saves an option under its id.
func (JSONStore) PutOption(ctx context.Context, o Option) error {
	return store.Put(ctx, optionsTable, o.ID, o)
}

// DeleteOption removes one option.
func (JSONStore) DeleteOption(ctx context.Context, id string) error {
	return store.Delete(ctx, optionsTable, id)
}

// GetCurrency reads one currency by its code.
func (JSONStore) GetCurrency(ctx context.Context, code string) (Currency, bool, error) {
	return store.Get[Currency](ctx, currenciesTable, code)
}

// ListCurrencies reads the whole currencies table; the codes all have three letters, so the store's order
// is the order of the codes.
func (JSONStore) ListCurrencies(ctx context.Context) ([]Currency, error) {
	return store.List[Currency](ctx, currenciesTable)
}

// PutCurrency saves a currency under its code.
func (JSONStore) PutCurrency(ctx context.Context, c Currency) error {
	return store.Put(ctx, currenciesTable, c.Code, c)
}

// DeleteCurrency removes one currency.
func (JSONStore) DeleteCurrency(ctx context.Context, code string) error {
	return store.Delete(ctx, currenciesTable, code)
}

// GetFinanceBudget reads one finance budget.
func (JSONStore) GetFinanceBudget(ctx context.Context, id string) (FinanceBudget, bool, error) {
	return store.Get[FinanceBudget](ctx, financeBudgetsTable, id)
}

// ListFinanceBudgets reads the whole finance_budgets table.
func (JSONStore) ListFinanceBudgets(ctx context.Context) ([]FinanceBudget, error) {
	return store.List[FinanceBudget](ctx, financeBudgetsTable)
}

// PutFinanceBudget saves a finance budget under its id.
func (JSONStore) PutFinanceBudget(ctx context.Context, b FinanceBudget) error {
	return store.Put(ctx, financeBudgetsTable, b.ID, b)
}

// DeleteFinanceBudget removes one finance budget.
func (JSONStore) DeleteFinanceBudget(ctx context.Context, id string) error {
	return store.Delete(ctx, financeBudgetsTable, id)
}

// GetFinanceBudgetItem reads one finance budget item.
func (JSONStore) GetFinanceBudgetItem(ctx context.Context, id string) (FinanceBudgetItem, bool, error) {
	return store.Get[FinanceBudgetItem](ctx, financeBudgetItemsTable, id)
}

// ListFinanceBudgetItems reads the whole finance_budget_items table.
func (JSONStore) ListFinanceBudgetItems(ctx context.Context) ([]FinanceBudgetItem, error) {
	return store.List[FinanceBudgetItem](ctx, financeBudgetItemsTable)
}

// ItemsOf reads every item to find those of one budget.
func (r JSONStore) ItemsOf(ctx context.Context, budgetID string) ([]FinanceBudgetItem, error) {
	all, err := r.ListFinanceBudgetItems(ctx)
	if err != nil {
		return nil, err
	}
	return filter(all, func(i FinanceBudgetItem) bool { return i.BudgetID == budgetID }), nil
}

// PutFinanceBudgetItem saves a finance budget item under its id.
func (JSONStore) PutFinanceBudgetItem(ctx context.Context, i FinanceBudgetItem) error {
	return store.Put(ctx, financeBudgetItemsTable, i.ID, i)
}

// DeleteFinanceBudgetItem removes one finance budget item.
func (JSONStore) DeleteFinanceBudgetItem(ctx context.Context, id string) error {
	return store.Delete(ctx, financeBudgetItemsTable, id)
}

// filter returns the elements of list that keep accepts, in order; never nil.
func filter[T any](list []T, keep func(T) bool) []T {
	kept := []T{}
	for _, element := range list {
		if keep(element) {
			kept = append(kept, element)
		}
	}
	return kept
}
