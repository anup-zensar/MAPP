// Package settingstest is the in-memory fake of the Settings store, for tests that build services or the
// whole API without a database.
package settingstest

import (
	"context"

	"mapp/internal/platform/fake"
	"mapp/internal/settings"
)

type Store struct {
	models        *fake.Table[settings.Model]
	activityTypes *fake.Table[settings.ActivityType]
	attributes    *fake.Table[settings.Attribute]
	options       *fake.Table[settings.Option]
	currencies    *fake.Table[settings.Currency]
	budgets       *fake.Table[settings.FinanceBudget]
	items         *fake.Table[settings.FinanceBudgetItem]
	rates         *fake.Table[settings.ExchangeRate]
	imports       *fake.Table[settings.RateImport]
	help          *fake.Table[settings.FieldHelpText]
}

func NewStore(db *fake.DB) Store {
	return Store{
		models:        fake.TableOf(db, "models", func(m settings.Model) string { return m.ID }),
		activityTypes: fake.TableOf(db, "activity_types", func(t settings.ActivityType) string { return t.ID }),
		attributes:    fake.TableOf(db, "shell_attributes", func(a settings.Attribute) string { return a.ID }),
		options:       fake.TableOf(db, "shell_attribute_options", func(o settings.Option) string { return o.ID }),
		currencies:    fake.TableOf(db, "currencies", func(c settings.Currency) string { return c.Code }),
		budgets:       fake.TableOf(db, "finance_budgets", func(b settings.FinanceBudget) string { return b.ID }),
		items:         fake.TableOf(db, "finance_budget_items", func(i settings.FinanceBudgetItem) string { return i.ID }),
		rates:         fake.TableOf(db, "exchange_rates", func(r settings.ExchangeRate) string { return r.ID }),
		imports:       fake.TableOf(db, "rate_imports", func(i settings.RateImport) string { return i.ID }),
		help:          fake.TableOf(db, "field_help", func(h settings.FieldHelpText) string { return h.Key }),
	}
}

func (s Store) GetModel(_ context.Context, id string) (settings.Model, bool, error) {
	return s.models.Get(id)
}

func (s Store) ListModels(context.Context) ([]settings.Model, error) {
	return s.models.List()
}

func (s Store) PutModel(ctx context.Context, m settings.Model) error {
	m.ActivityTypeIDs = orEmpty(m.ActivityTypeIDs)
	return s.models.Put(ctx, m)
}

func (s Store) DeleteModel(ctx context.Context, id string) error {
	return s.models.Delete(ctx, id)
}

func (s Store) GetActivityType(_ context.Context, id string) (settings.ActivityType, bool, error) {
	return s.activityTypes.Get(id)
}

func (s Store) ListActivityTypes(context.Context) ([]settings.ActivityType, error) {
	return s.activityTypes.List()
}

func (s Store) PutActivityType(ctx context.Context, t settings.ActivityType) error {
	return s.activityTypes.Put(ctx, t)
}

func (s Store) DeleteActivityType(ctx context.Context, id string) error {
	return s.activityTypes.Delete(ctx, id)
}

// orEmpty keeps a record's list as PostgreSQL reads its child rows back: empty, never nil.
func orEmpty(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
}
