package settings

import (
	"context"

	"northstar/internal/platform/store"
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

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// GetModel reads one model.
func (jsonRepository) GetModel(ctx context.Context, id string) (Model, bool, error) {
	return store.Get[Model](ctx, modelsTable, id)
}

// ListModels reads the whole models table.
func (jsonRepository) ListModels(ctx context.Context) ([]Model, error) {
	return store.List[Model](ctx, modelsTable)
}

// PutModel saves a model under its id.
func (jsonRepository) PutModel(ctx context.Context, m Model) error {
	return store.Put(ctx, modelsTable, m.ID, m)
}

// DeleteModel removes one model.
func (jsonRepository) DeleteModel(ctx context.Context, id string) error {
	return store.Delete(ctx, modelsTable, id)
}

// GetActivityType reads one activity type.
func (jsonRepository) GetActivityType(ctx context.Context, id string) (ActivityType, bool, error) {
	return store.Get[ActivityType](ctx, activityTypesTable, id)
}

// ListActivityTypes reads the whole activity_types table.
func (jsonRepository) ListActivityTypes(ctx context.Context) ([]ActivityType, error) {
	return store.List[ActivityType](ctx, activityTypesTable)
}

// PutActivityType saves an activity type under its id.
func (jsonRepository) PutActivityType(ctx context.Context, t ActivityType) error {
	return store.Put(ctx, activityTypesTable, t.ID, t)
}

// DeleteActivityType removes one activity type.
func (jsonRepository) DeleteActivityType(ctx context.Context, id string) error {
	return store.Delete(ctx, activityTypesTable, id)
}

// GetAttribute reads one Shell attribute.
func (jsonRepository) GetAttribute(ctx context.Context, id string) (Attribute, bool, error) {
	return store.Get[Attribute](ctx, attributesTable, id)
}

// ListAttributes reads the whole shell_attributes table.
func (jsonRepository) ListAttributes(ctx context.Context) ([]Attribute, error) {
	return store.List[Attribute](ctx, attributesTable)
}

// PutAttribute saves a Shell attribute under its id.
func (jsonRepository) PutAttribute(ctx context.Context, a Attribute) error {
	return store.Put(ctx, attributesTable, a.ID, a)
}

// DeleteAttribute removes one Shell attribute.
func (jsonRepository) DeleteAttribute(ctx context.Context, id string) error {
	return store.Delete(ctx, attributesTable, id)
}

// GetOption reads one option.
func (jsonRepository) GetOption(ctx context.Context, id string) (Option, bool, error) {
	return store.Get[Option](ctx, optionsTable, id)
}

// ListOptions reads the whole shell_attribute_options table.
func (jsonRepository) ListOptions(ctx context.Context) ([]Option, error) {
	return store.List[Option](ctx, optionsTable)
}

// OptionsOf reads every option to find those of one attribute: quick enough for the 5,000 options an
// attribute may have. PostgreSQL would use an index.
func (r jsonRepository) OptionsOf(ctx context.Context, attributeID string) ([]Option, error) {
	all, err := r.ListOptions(ctx)
	if err != nil {
		return nil, err
	}
	return filter(all, func(o Option) bool { return o.AttributeID == attributeID }), nil
}

// PutOption saves an option under its id.
func (jsonRepository) PutOption(ctx context.Context, o Option) error {
	return store.Put(ctx, optionsTable, o.ID, o)
}

// DeleteOption removes one option.
func (jsonRepository) DeleteOption(ctx context.Context, id string) error {
	return store.Delete(ctx, optionsTable, id)
}

// GetCurrency reads one currency by its code.
func (jsonRepository) GetCurrency(ctx context.Context, code string) (Currency, bool, error) {
	return store.Get[Currency](ctx, currenciesTable, code)
}

// ListCurrencies reads the whole currencies table; the codes all have three letters, so the store's order
// is the order of the codes.
func (jsonRepository) ListCurrencies(ctx context.Context) ([]Currency, error) {
	return store.List[Currency](ctx, currenciesTable)
}

// PutCurrency saves a currency under its code.
func (jsonRepository) PutCurrency(ctx context.Context, c Currency) error {
	return store.Put(ctx, currenciesTable, c.Code, c)
}

// DeleteCurrency removes one currency.
func (jsonRepository) DeleteCurrency(ctx context.Context, code string) error {
	return store.Delete(ctx, currenciesTable, code)
}

// GetFinanceBudget reads one finance budget.
func (jsonRepository) GetFinanceBudget(ctx context.Context, id string) (FinanceBudget, bool, error) {
	return store.Get[FinanceBudget](ctx, financeBudgetsTable, id)
}

// ListFinanceBudgets reads the whole finance_budgets table.
func (jsonRepository) ListFinanceBudgets(ctx context.Context) ([]FinanceBudget, error) {
	return store.List[FinanceBudget](ctx, financeBudgetsTable)
}

// PutFinanceBudget saves a finance budget under its id.
func (jsonRepository) PutFinanceBudget(ctx context.Context, b FinanceBudget) error {
	return store.Put(ctx, financeBudgetsTable, b.ID, b)
}

// DeleteFinanceBudget removes one finance budget.
func (jsonRepository) DeleteFinanceBudget(ctx context.Context, id string) error {
	return store.Delete(ctx, financeBudgetsTable, id)
}

// GetFinanceBudgetItem reads one finance budget item.
func (jsonRepository) GetFinanceBudgetItem(ctx context.Context, id string) (FinanceBudgetItem, bool, error) {
	return store.Get[FinanceBudgetItem](ctx, financeBudgetItemsTable, id)
}

// ListFinanceBudgetItems reads the whole finance_budget_items table.
func (jsonRepository) ListFinanceBudgetItems(ctx context.Context) ([]FinanceBudgetItem, error) {
	return store.List[FinanceBudgetItem](ctx, financeBudgetItemsTable)
}

// ItemsOf reads every item to find those of one budget.
func (r jsonRepository) ItemsOf(ctx context.Context, budgetID string) ([]FinanceBudgetItem, error) {
	all, err := r.ListFinanceBudgetItems(ctx)
	if err != nil {
		return nil, err
	}
	return filter(all, func(i FinanceBudgetItem) bool { return i.BudgetID == budgetID }), nil
}

// PutFinanceBudgetItem saves a finance budget item under its id.
func (jsonRepository) PutFinanceBudgetItem(ctx context.Context, i FinanceBudgetItem) error {
	return store.Put(ctx, financeBudgetItemsTable, i.ID, i)
}

// DeleteFinanceBudgetItem removes one finance budget item.
func (jsonRepository) DeleteFinanceBudgetItem(ctx context.Context, id string) error {
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
