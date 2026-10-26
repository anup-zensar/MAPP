package settings

import "context"

// Repository stores the Settings records. The JSON store implements it now (store_json.go) and PostgreSQL
// can later, without any change to the service. Every method runs inside the caller's transaction, which
// the context carries. Lists are ordered by id (as store.List: shorter ids first) and never nil; records in
// the recycle bin are not in the tables.
type Repository interface {
	// GetModel returns the model with id, and false when there is none.
	GetModel(ctx context.Context, id string) (Model, bool, error)
	// ListModels returns every model.
	ListModels(ctx context.Context) ([]Model, error)
	// PutModel saves m, replacing the model with the same id.
	PutModel(ctx context.Context, m Model) error
	// DeleteModel removes the model with id; removing one that does not exist changes nothing.
	DeleteModel(ctx context.Context, id string) error

	// GetActivityType returns the activity type with id, and false when there is none.
	GetActivityType(ctx context.Context, id string) (ActivityType, bool, error)
	// ListActivityTypes returns every activity type.
	ListActivityTypes(ctx context.Context) ([]ActivityType, error)
	// PutActivityType saves t, replacing the activity type with the same id.
	PutActivityType(ctx context.Context, t ActivityType) error
	// DeleteActivityType removes the activity type with id.
	DeleteActivityType(ctx context.Context, id string) error

	// GetAttribute returns the Shell attribute with id, and false when there is none.
	GetAttribute(ctx context.Context, id string) (Attribute, bool, error)
	// ListAttributes returns every Shell attribute of every level (display order is the service's).
	ListAttributes(ctx context.Context) ([]Attribute, error)
	// PutAttribute saves a, replacing the Shell attribute with the same id.
	PutAttribute(ctx context.Context, a Attribute) error
	// DeleteAttribute removes the Shell attribute with id (its options are removed separately).
	DeleteAttribute(ctx context.Context, id string) error

	// GetOption returns the option with id, and false when there is none.
	GetOption(ctx context.Context, id string) (Option, bool, error)
	// ListOptions returns every option of every attribute.
	ListOptions(ctx context.Context) ([]Option, error)
	// OptionsOf returns the options of the attribute attributeID, ordered by id.
	OptionsOf(ctx context.Context, attributeID string) ([]Option, error)
	// PutOption saves o, replacing the option with the same id.
	PutOption(ctx context.Context, o Option) error
	// AddOptions saves new options, whose ids are not stored yet, all at once; DeleteOptions removes options.
	AddOptions(ctx context.Context, options []Option) error
	DeleteOptions(ctx context.Context, ids []string) error
	// DeleteOption removes the option with id.
	DeleteOption(ctx context.Context, id string) error

	// GetCurrency returns the currency with the ISO code, and false when there is none.
	GetCurrency(ctx context.Context, code string) (Currency, bool, error)
	// ListCurrencies returns every currency, ordered by code.
	ListCurrencies(ctx context.Context) ([]Currency, error)
	// PutCurrency saves c under its code, replacing the currency with the same code.
	PutCurrency(ctx context.Context, c Currency) error
	// DeleteCurrency removes the currency with the code.
	DeleteCurrency(ctx context.Context, code string) error

	// GetFinanceBudget returns the finance budget with id, and false when there is none.
	GetFinanceBudget(ctx context.Context, id string) (FinanceBudget, bool, error)
	// ListFinanceBudgets returns every finance budget of every year.
	ListFinanceBudgets(ctx context.Context) ([]FinanceBudget, error)
	// PutFinanceBudget saves b, replacing the finance budget with the same id.
	PutFinanceBudget(ctx context.Context, b FinanceBudget) error
	// DeleteFinanceBudget removes the finance budget with id (only a restore's Remove needs it: budgets are
	// never deleted on their own, D-047).
	DeleteFinanceBudget(ctx context.Context, id string) error

	// GetFinanceBudgetItem returns the finance budget item with id, and false when there is none.
	GetFinanceBudgetItem(ctx context.Context, id string) (FinanceBudgetItem, bool, error)
	// ListFinanceBudgetItems returns every finance budget item.
	ListFinanceBudgetItems(ctx context.Context) ([]FinanceBudgetItem, error)
	// ItemsOf returns the items of the finance budget budgetID, ordered by id.
	ItemsOf(ctx context.Context, budgetID string) ([]FinanceBudgetItem, error)
	// PutFinanceBudgetItem saves i, replacing the item with the same id.
	PutFinanceBudgetItem(ctx context.Context, i FinanceBudgetItem) error
	// DeleteFinanceBudgetItem removes the item with id.
	DeleteFinanceBudgetItem(ctx context.Context, id string) error

	// GetRate returns the dated exchange rate with id, and false when there is none (016).
	GetRate(ctx context.Context, id string) (ExchangeRate, bool, error)
	// ListRates returns every dated exchange rate of every currency.
	ListRates(ctx context.Context) ([]ExchangeRate, error)
	// RatesOf returns the dated rates of the currency code, oldest date first.
	RatesOf(ctx context.Context, code string) ([]ExchangeRate, error)
	// PutRate saves r, replacing the rate with the same id.
	PutRate(ctx context.Context, r ExchangeRate) error
	// PutRates saves rates, replacing those with the same id, all at once.
	PutRates(ctx context.Context, rates []ExchangeRate) error
	// DeleteRate removes the rate with id.
	DeleteRate(ctx context.Context, id string) error

	// PutRateImport saves the record of a rate import (016).
	PutRateImport(ctx context.Context, i RateImport) error
	// GetRateImport returns the rate import with id, and false when there is none.
	GetRateImport(ctx context.Context, id string) (RateImport, bool, error)
	// ListRateImports returns every rate import, oldest first.
	ListRateImports(ctx context.Context) ([]RateImport, error)

	// GetFieldHelp returns the edited help text of the catalogue key, and false when it has none (016).
	GetFieldHelp(ctx context.Context, key string) (FieldHelpText, bool, error)
	// ListFieldHelp returns every edited help text.
	ListFieldHelp(ctx context.Context) ([]FieldHelpText, error)
	// PutFieldHelp saves h under its key.
	PutFieldHelp(ctx context.Context, h FieldHelpText) error
	// DeleteFieldHelp removes the edited text of key (Reset to default).
	DeleteFieldHelp(ctx context.Context, key string) error
}
