package settings

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists by id (currencies by code, field help by key, RatesOf by date first), never nil; a record in the
// recycle bin is not in its table. AddOptions takes only options whose ids are not stored yet.
type Store interface {
	GetModel(ctx context.Context, id string) (Model, bool, error)
	ListModels(ctx context.Context) ([]Model, error)
	PutModel(ctx context.Context, m Model) error
	DeleteModel(ctx context.Context, id string) error

	GetActivityType(ctx context.Context, id string) (ActivityType, bool, error)
	ListActivityTypes(ctx context.Context) ([]ActivityType, error)
	PutActivityType(ctx context.Context, t ActivityType) error
	DeleteActivityType(ctx context.Context, id string) error

	GetAttribute(ctx context.Context, id string) (Attribute, bool, error)
	ListAttributes(ctx context.Context) ([]Attribute, error)
	PutAttribute(ctx context.Context, a Attribute) error
	DeleteAttribute(ctx context.Context, id string) error

	GetOption(ctx context.Context, id string) (Option, bool, error)
	ListOptions(ctx context.Context) ([]Option, error)
	OptionsOf(ctx context.Context, attributeID string) ([]Option, error)
	PutOption(ctx context.Context, o Option) error
	AddOptions(ctx context.Context, options []Option) error
	DeleteOptions(ctx context.Context, ids []string) error
	DeleteOption(ctx context.Context, id string) error

	GetCurrency(ctx context.Context, code string) (Currency, bool, error)
	ListCurrencies(ctx context.Context) ([]Currency, error)
	PutCurrency(ctx context.Context, c Currency) error
	DeleteCurrency(ctx context.Context, code string) error

	GetFinanceBudget(ctx context.Context, id string) (FinanceBudget, bool, error)
	ListFinanceBudgets(ctx context.Context) ([]FinanceBudget, error)
	PutFinanceBudget(ctx context.Context, b FinanceBudget) error
	DeleteFinanceBudget(ctx context.Context, id string) error

	GetFinanceBudgetItem(ctx context.Context, id string) (FinanceBudgetItem, bool, error)
	ListFinanceBudgetItems(ctx context.Context) ([]FinanceBudgetItem, error)
	ItemsOf(ctx context.Context, budgetID string) ([]FinanceBudgetItem, error)
	PutFinanceBudgetItem(ctx context.Context, i FinanceBudgetItem) error
	DeleteFinanceBudgetItem(ctx context.Context, id string) error

	GetRate(ctx context.Context, id string) (ExchangeRate, bool, error)
	ListRates(ctx context.Context) ([]ExchangeRate, error)
	RatesOf(ctx context.Context, code string) ([]ExchangeRate, error)
	PutRate(ctx context.Context, r ExchangeRate) error
	PutRates(ctx context.Context, rates []ExchangeRate) error
	DeleteRate(ctx context.Context, id string) error

	PutRateImport(ctx context.Context, i RateImport) error
	GetRateImport(ctx context.Context, id string) (RateImport, bool, error)
	ListRateImports(ctx context.Context) ([]RateImport, error)

	GetFieldHelp(ctx context.Context, key string) (FieldHelpText, bool, error)
	ListFieldHelp(ctx context.Context) ([]FieldHelpText, error)
	PutFieldHelp(ctx context.Context, h FieldHelpText) error
	DeleteFieldHelp(ctx context.Context, key string) error
}
