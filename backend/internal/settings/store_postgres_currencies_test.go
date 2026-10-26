package settings_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

func pgCurrency(code, name string) settings.Currency {
	return settings.Currency{Code: code, Name: name, Symbol: "", Active: true, Meta: pgMeta}
}

func pgRate(id, currency, day string) settings.ExchangeRate {
	return settings.ExchangeRate{ID: id, Currency: currency, Date: date.MustParse(day),
		Rate: money.RateFromMicros(920000), Source: settings.RateManual, Meta: pgMeta}
}

func TestPostgresStoreCurrenciesAndRatesFR008(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	legacy := pgCurrency("EUR", "Euro")
	legacy.Rate, legacy.RateChangedAt = money.RateFromMicros(920000), time.Date(2026, time.January, 5, 8, 0, 0, 0, time.UTC)
	legacy.RateChangedBy = actor.Actor{ID: "U-1001", Name: "Priya Raman"}
	first, last := date.MustParse("2026-03-01"), date.MustParse("2026-03-02")
	imports := []settings.RateImport{
		{ID: "RI-1001", FileName: "rates.csv", Feed: "gsapDailyRatesFile", FirstDate: &first, LastDate: &last, Added: 2, Meta: pgMeta},
		{ID: "RI-1002", FileName: "empty.csv", Feed: "gsapDailyRatesFile", Problems: 1, Meta: pgMeta},
	}
	later := pgRate("XR-1001", "EUR", "2026-03-02")
	earlier := pgRate("XR-1002", "EUR", "2026-03-01")
	earlier.Source, earlier.ImportID = settings.RateGSAPImport, "RI-1001"
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutCurrency(ctx, pgCurrency("USD", "US dollar")), s.PutCurrency(ctx, legacy),
			s.PutCurrency(ctx, pgCurrency("JPY", "Japanese yen")), s.PutRateImport(ctx, imports[0]),
			s.PutRateImport(ctx, imports[1]), s.PutRate(ctx, later), s.PutRate(ctx, earlier),
			s.PutRate(ctx, pgRate("XR-1003", "JPY", "2026-03-01")))
	})

	currencies, currenciesErr := s.ListCurrencies(t.Context())
	ofEuro, ratesErr := s.RatesOf(t.Context(), "EUR")
	gotImports, importsErr := s.ListRateImports(t.Context())

	if err := errors.Join(currenciesErr, ratesErr, importsErr); err != nil {
		t.Fatalf("ListCurrencies(), RatesOf(), ListRateImports() error: %v", err)
	}
	wantCurrencies := []settings.Currency{legacy, pgCurrency("JPY", "Japanese yen"), pgCurrency("USD", "US dollar")}
	if diff := cmp.Diff(wantCurrencies, currencies, pgExact); diff != "" {
		t.Errorf("ListCurrencies() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]settings.ExchangeRate{earlier, later}, ofEuro, pgExact); diff != "" {
		t.Errorf("RatesOf() oldest date first mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(imports, gotImports, pgExact); diff != "" {
		t.Errorf("ListRateImports() mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreFinanceBudgetsAndFieldHelpFR008(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	budgets := []settings.FinanceBudget{{ID: "FB-1001", Year: 2026, Name: "Brand", Meta: pgMeta},
		{ID: "FB-1002", Year: 2027, Name: "Brand", Meta: pgMeta}}
	items := []settings.FinanceBudgetItem{
		{ID: "FBI-1001", BudgetID: "FB-1001", Name: "Media", Code: "MED-1", Active: true, Meta: pgMeta},
		{ID: "FBI-1002", BudgetID: "FB-1001", Name: "Events", Active: false, Meta: pgMeta},
		{ID: "FBI-1003", BudgetID: "FB-1002", Name: "Media", Code: "MED-1", Active: true, Meta: pgMeta},
	}
	help := []settings.FieldHelpText{{Key: "plan.budget", Text: "The plan's budget.", Meta: pgMeta},
		{Key: "activity.name", Text: "", Meta: pgMeta}}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutFinanceBudget(ctx, budgets[0]), s.PutFinanceBudget(ctx, budgets[1]),
			s.PutFinanceBudgetItem(ctx, items[0]), s.PutFinanceBudgetItem(ctx, items[1]),
			s.PutFinanceBudgetItem(ctx, items[2]), s.PutFieldHelp(ctx, help[1]), s.PutFieldHelp(ctx, help[0]),
			s.PutFieldHelp(ctx, settings.FieldHelpText{Key: "line.cost", Meta: pgMeta}))
	})
	pgWrite(t, db, func(ctx context.Context) error { return s.DeleteFieldHelp(ctx, "line.cost") })

	gotBudgets, budgetsErr := s.ListFinanceBudgets(t.Context())
	ofFirst, itemsErr := s.ItemsOf(t.Context(), "FB-1001")
	gotHelp, helpErr := s.ListFieldHelp(t.Context())

	if err := errors.Join(budgetsErr, itemsErr, helpErr); err != nil {
		t.Fatalf("ListFinanceBudgets(), ItemsOf(), ListFieldHelp() error: %v", err)
	}
	if diff := cmp.Diff(budgets, gotBudgets, pgExact); diff != "" {
		t.Errorf("ListFinanceBudgets() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(items[:2], ofFirst, pgExact); diff != "" {
		t.Errorf("ItemsOf() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(help, gotHelp, pgExact); diff != "" {
		t.Errorf("ListFieldHelp() shorter keys first mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreRefusesBrokenRatesFR004(t *testing.T) {
	t.Parallel()
	usdInactive := pgCurrency("USD", "US dollar")
	usdInactive.Active = false
	tests := []struct {
		name  string
		write func(ctx context.Context, s *settings.PostgresStore) error
	}{
		{name: "a second rate for the same currency and day", write: func(ctx context.Context, s *settings.PostgresStore) error {
			return s.PutRate(ctx, pgRate("XR-1002", "EUR", "2026-03-01"))
		}},
		{name: "a rate for US dollars", write: func(ctx context.Context, s *settings.PostgresStore) error {
			return s.PutRate(ctx, pgRate("XR-1002", "USD", "2026-03-02"))
		}},
		{name: "a rate of a currency that does not exist", write: func(ctx context.Context, s *settings.PostgresStore) error {
			return s.PutRate(ctx, pgRate("XR-1002", "GBP", "2026-03-02"))
		}},
		{name: "US dollars set inactive", write: func(ctx context.Context, s *settings.PostgresStore) error {
			return s.PutCurrency(ctx, usdInactive)
		}},
		{name: "a currency deleted while it has rates", write: func(ctx context.Context, s *settings.PostgresStore) error {
			return s.DeleteCurrency(ctx, "EUR")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s := pgStore(t)
			pgWrite(t, db, func(ctx context.Context) error {
				return errors.Join(s.PutCurrency(ctx, pgCurrency("USD", "US dollar")), s.PutCurrency(ctx, pgCurrency("EUR", "Euro")),
					s.PutRate(ctx, pgRate("XR-1001", "EUR", "2026-03-01")))
			})

			err := db.Write(t.Context(), func(ctx context.Context) error { return tt.write(ctx, s) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}
