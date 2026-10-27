package settings

import (
	"context"
	"slices"

	"mapp/internal/platform/fake"
)

// fakeStore is settingstest.Store for this package's own tests, which cannot import settingstest.
type fakeStore struct {
	models        *fake.Table[Model]
	activityTypes *fake.Table[ActivityType]
	attributes    *fake.Table[Attribute]
	options       *fake.Table[Option]
	currencies    *fake.Table[Currency]
	budgets       *fake.Table[FinanceBudget]
	items         *fake.Table[FinanceBudgetItem]
	rates         *fake.Table[ExchangeRate]
	imports       *fake.Table[RateImport]
	help          *fake.Table[FieldHelpText]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{
		models:        fake.TableOf(db, "models", func(m Model) string { return m.ID }),
		activityTypes: fake.TableOf(db, "activity_types", func(t ActivityType) string { return t.ID }),
		attributes:    fake.TableOf(db, "shell_attributes", func(a Attribute) string { return a.ID }),
		options:       fake.TableOf(db, "shell_attribute_options", func(o Option) string { return o.ID }),
		currencies:    fake.TableOf(db, "currencies", func(c Currency) string { return c.Code }),
		budgets:       fake.TableOf(db, "finance_budgets", func(b FinanceBudget) string { return b.ID }),
		items:         fake.TableOf(db, "finance_budget_items", func(i FinanceBudgetItem) string { return i.ID }),
		rates:         fake.TableOf(db, "exchange_rates", func(r ExchangeRate) string { return r.ID }),
		imports:       fake.TableOf(db, "rate_imports", func(i RateImport) string { return i.ID }),
		help:          fake.TableOf(db, "field_help", func(h FieldHelpText) string { return h.Key }),
	}
}

func (s fakeStore) GetModel(_ context.Context, id string) (Model, bool, error) {
	return s.models.Get(id)
}

func (s fakeStore) ListModels(context.Context) ([]Model, error) {
	return s.models.List()
}

func (s fakeStore) PutModel(ctx context.Context, m Model) error {
	m.ActivityTypeIDs = nonNil(m.ActivityTypeIDs)
	return s.models.Put(ctx, m)
}

func (s fakeStore) DeleteModel(ctx context.Context, id string) error {
	return s.models.Delete(ctx, id)
}

func (s fakeStore) GetActivityType(_ context.Context, id string) (ActivityType, bool, error) {
	return s.activityTypes.Get(id)
}

func (s fakeStore) ListActivityTypes(context.Context) ([]ActivityType, error) {
	return s.activityTypes.List()
}

func (s fakeStore) PutActivityType(ctx context.Context, t ActivityType) error {
	return s.activityTypes.Put(ctx, t)
}

func (s fakeStore) DeleteActivityType(ctx context.Context, id string) error {
	return s.activityTypes.Delete(ctx, id)
}

func (s fakeStore) GetAttribute(_ context.Context, id string) (Attribute, bool, error) {
	return s.attributes.Get(id)
}

func (s fakeStore) ListAttributes(context.Context) ([]Attribute, error) {
	return s.attributes.List()
}

func (s fakeStore) PutAttribute(ctx context.Context, a Attribute) error {
	a.Condition = nonNil(a.Condition)
	return s.attributes.Put(ctx, a)
}

func (s fakeStore) DeleteAttribute(ctx context.Context, id string) error {
	return s.attributes.Delete(ctx, id)
}

func (s fakeStore) GetOption(_ context.Context, id string) (Option, bool, error) {
	return s.options.Get(id)
}

func (s fakeStore) ListOptions(context.Context) ([]Option, error) {
	return s.options.List()
}

func (s fakeStore) OptionsOf(_ context.Context, attributeID string) ([]Option, error) {
	return s.options.Where(func(o Option) bool { return o.AttributeID == attributeID })
}

func (s fakeStore) PutOption(ctx context.Context, o Option) error {
	o.ParentOptionIDs = nonNil(o.ParentOptionIDs)
	return s.options.Put(ctx, o)
}

func (s fakeStore) AddOptions(ctx context.Context, options []Option) error {
	for _, o := range options {
		if err := s.PutOption(ctx, o); err != nil {
			return err
		}
	}
	return nil
}

func (s fakeStore) DeleteOptions(ctx context.Context, ids []string) error {
	return s.options.Delete(ctx, ids...)
}

func (s fakeStore) DeleteOption(ctx context.Context, id string) error {
	return s.options.Delete(ctx, id)
}

func (s fakeStore) GetCurrency(_ context.Context, code string) (Currency, bool, error) {
	return s.currencies.Get(code)
}

func (s fakeStore) ListCurrencies(context.Context) ([]Currency, error) {
	return s.currencies.List()
}

func (s fakeStore) PutCurrency(ctx context.Context, c Currency) error {
	return s.currencies.Put(ctx, c)
}

func (s fakeStore) DeleteCurrency(ctx context.Context, code string) error {
	return s.currencies.Delete(ctx, code)
}

func (s fakeStore) GetFinanceBudget(_ context.Context, id string) (FinanceBudget, bool, error) {
	return s.budgets.Get(id)
}

func (s fakeStore) ListFinanceBudgets(context.Context) ([]FinanceBudget, error) {
	return s.budgets.List()
}

func (s fakeStore) PutFinanceBudget(ctx context.Context, b FinanceBudget) error {
	return s.budgets.Put(ctx, b)
}

func (s fakeStore) DeleteFinanceBudget(ctx context.Context, id string) error {
	return s.budgets.Delete(ctx, id)
}

func (s fakeStore) GetFinanceBudgetItem(_ context.Context, id string) (FinanceBudgetItem, bool, error) {
	return s.items.Get(id)
}

func (s fakeStore) ListFinanceBudgetItems(context.Context) ([]FinanceBudgetItem, error) {
	return s.items.List()
}

func (s fakeStore) ItemsOf(_ context.Context, budgetID string) ([]FinanceBudgetItem, error) {
	return s.items.Where(func(i FinanceBudgetItem) bool { return i.BudgetID == budgetID })
}

func (s fakeStore) PutFinanceBudgetItem(ctx context.Context, i FinanceBudgetItem) error {
	return s.items.Put(ctx, i)
}

func (s fakeStore) DeleteFinanceBudgetItem(ctx context.Context, id string) error {
	return s.items.Delete(ctx, id)
}

func (s fakeStore) GetRate(_ context.Context, id string) (ExchangeRate, bool, error) {
	return s.rates.Get(id)
}

func (s fakeStore) ListRates(context.Context) ([]ExchangeRate, error) {
	return s.rates.List()
}

func (s fakeStore) RatesOf(_ context.Context, code string) ([]ExchangeRate, error) {
	rates, err := s.rates.Where(func(r ExchangeRate) bool { return r.Currency == code })
	slices.SortStableFunc(rates, func(a, b ExchangeRate) int { return a.Date.Compare(b.Date) })
	return rates, err
}

func (s fakeStore) PutRate(ctx context.Context, r ExchangeRate) error {
	return s.rates.Put(ctx, r)
}

func (s fakeStore) PutRates(ctx context.Context, rates []ExchangeRate) error {
	return s.rates.Put(ctx, rates...)
}

func (s fakeStore) DeleteRate(ctx context.Context, id string) error {
	return s.rates.Delete(ctx, id)
}

func (s fakeStore) PutRateImport(ctx context.Context, i RateImport) error {
	return s.imports.Put(ctx, i)
}

func (s fakeStore) GetRateImport(_ context.Context, id string) (RateImport, bool, error) {
	return s.imports.Get(id)
}

func (s fakeStore) ListRateImports(context.Context) ([]RateImport, error) {
	return s.imports.List()
}

func (s fakeStore) GetFieldHelp(_ context.Context, key string) (FieldHelpText, bool, error) {
	return s.help.Get(key)
}

func (s fakeStore) ListFieldHelp(context.Context) ([]FieldHelpText, error) {
	return s.help.List()
}

func (s fakeStore) PutFieldHelp(ctx context.Context, h FieldHelpText) error {
	return s.help.Put(ctx, h)
}

func (s fakeStore) DeleteFieldHelp(ctx context.Context, key string) error {
	return s.help.Delete(ctx, key)
}
