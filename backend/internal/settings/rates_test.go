package settings

import (
	"context"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// october20 is the day of the 016 examples: 20 Oct 2026, 09:00 UTC.
var october20 = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

// addRate stores a dated rate of code directly, as the seed and an import do.
func (f *fixture) addRate(t *testing.T, code, day, rate string, source RateSource, importID string) ExchangeRate {
	t.Helper()
	r, err := money.ParseRate(rate)
	if err != nil {
		t.Fatalf("ParseRate(%q): %v", rate, err)
	}
	var x ExchangeRate
	f.write(t, func(ctx context.Context, repo Repository) error {
		x = ExchangeRate{ID: f.next(ctx, ids.PrefixExchangeRate), Currency: code, Date: date.MustParse(day), Rate: r, Source: source,
			ImportID: importID, Meta: record.NewMeta(actor.System, start)}
		return repo.PutRate(ctx, x)
	})
	return x
}

// storedRates returns every dated rate of code, oldest first.
func (f *fixture) storedRates(t *testing.T, code string) []ExchangeRate {
	t.Helper()
	var rates []ExchangeRate
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		rates, err = f.service.repo.RatesOf(ctx, code)
		return err
	})
	if err != nil {
		t.Fatalf("RatesOf: %v", err)
	}
	return rates
}

// The migration gives every currency but USD its stored rate as its first dated rate — dated the day the rate
// was last changed, entered by hand by whoever changed it — and stores the currency without the legacy
// fields; it writes no log entry (data-model §2.2).
func TestMigrateRates_CreatesOneRatePerCurrency(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "EUR", "0.92", true)
	f.addCurrency(t, "JPY", "150", false)
	f.clock.now = october20

	if err := f.service.MigrateRates(t.Context()); err != nil {
		t.Fatalf("MigrateRates: %v", err)
	}

	for code, want := range map[string]string{"EUR": "0.92", "JPY": "150"} {
		rates := f.storedRates(t, code)
		if len(rates) != 1 || rates[0].Rate.String() != want || rates[0].Date != date.MustParse("2026-03-12") ||
			rates[0].Source != RateManual || rates[0].ImportID != "" || rates[0].CreatedBy != actor.System {
			t.Errorf("%s: rates = %+v", code, rates)
		}
	}
	if usd := f.storedRates(t, "USD"); len(usd) != 0 {
		t.Errorf("USD has rates %+v; it is always 1", usd)
	}
	var stored []Currency
	f.write(t, func(ctx context.Context, repo Repository) error {
		var err error
		stored, err = repo.ListCurrencies(ctx)
		return err
	})
	for _, c := range stored {
		if hasLegacy(c) {
			t.Errorf("%s is still stored with its legacy rate: %+v", c.Code, c)
		}
	}
	if rates, err := f.service.Rates(t.Context()); err != nil || rates["EUR"].String() != "0.92" || rates["JPY"].String() != "150" ||
		!rates["USD"].IsOne() {
		t.Errorf("Rates after the migration = %v, %v", rates, err)
	}
	if entries := f.logEntries(t); len(entries) != 0 {
		t.Errorf("the migration logged %d entries", len(entries))
	}
}

// A second start changes nothing (data-model §2.2).
func TestMigrateRates_SecondRunDoesNothing(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.addCurrency(t, "EUR", "0.92", true)
	if err := f.service.MigrateRates(t.Context()); err != nil {
		t.Fatalf("MigrateRates: %v", err)
	}
	first := f.storedRates(t, "EUR")

	if err := f.service.MigrateRates(t.Context()); err != nil {
		t.Fatalf("second MigrateRates: %v", err)
	}

	if again := f.storedRates(t, "EUR"); len(again) != 1 || again[0] != first[0] {
		t.Errorf("after a second start: %+v, want %+v", again, first)
	}
}

// The current rate is the rate with the latest date on or before today (FR-002): a day without a rate uses
// the latest earlier one.
func TestRates_LatestOnOrBeforeToday_FR002(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "EUR", "0.90", true)
	f.addRate(t, "EUR", "2026-10-16", "0.91", RateManual, "")
	f.addRate(t, "EUR", "2026-10-19", "0.92", RateGSAPImport, "RI-1001")
	f.addRate(t, "EUR", "2026-10-18", "0.915", RateGSAPImport, "RI-1001")
	f.clock.now = october20

	rates, err := f.service.Rates(t.Context())

	if err != nil || rates["EUR"].String() != "0.92" || !rates["USD"].IsOne() {
		t.Errorf("rates on 20 Oct = %v, %v; want EUR 0.92 (19 Oct)", rates, err)
	}
	f.clock.now = time.Date(2026, 10, 17, 23, 59, 0, 0, time.UTC)
	if rates, _ := f.service.Rates(t.Context()); rates["EUR"].String() != "0.91" {
		t.Errorf("rate on 17 Oct = %v, want 0.91 (16 Oct, the latest earlier rate)", rates["EUR"])
	}
}

// A future-dated rate is not used before its date and becomes current on its date without anyone saving
// anything (FR-002, US1-1).
func TestRates_FutureRateNotUsed_FR002(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.addCurrency(t, "EUR", "0.90", true)
	f.addRate(t, "EUR", "2026-10-19", "0.92", RateManual, "")
	f.addRate(t, "EUR", "2027-01-01", "0.95", RateManual, "")
	f.clock.now = october20

	before, _ := f.service.Rates(t.Context())
	f.clock.now = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	after, _ := f.service.Rates(t.Context())

	if before["EUR"].String() != "0.92" || after["EUR"].String() != "0.95" {
		t.Errorf("EUR on 20 Oct 2026 = %v, on 01 Jan 2027 = %v; want 0.92 and 0.95", before["EUR"], after["EUR"])
	}
}

// RatesOn converts at any day: the rate current on that day (D-275).
func TestRatesOn(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "EUR", "0.90", true)
	f.addRate(t, "EUR", "2026-10-18", "0.91", RateManual, "")
	f.addRate(t, "EUR", "2026-10-19", "0.92", RateManual, "")
	f.addRate(t, "EUR", "2027-01-01", "0.95", RateManual, "")
	f.clock.now = october20

	for day, want := range map[string]string{"2026-10-18": "0.91", "2026-12-31": "0.92", "2027-01-02": "0.95"} {
		rates, err := f.service.RatesOn(t.Context(), date.MustParse(day))
		if err != nil || rates["EUR"].String() != want || !rates["USD"].IsOne() {
			t.Errorf("RatesOn(%s) = %v, %v; want EUR %s", day, rates, err, want)
		}
	}
}

// Every currency's current rate comes with its date and source, and an imported rate with its file: what the
// web app shows beside converted amounts (FR-009). USD first, without a date or source.
func TestCurrentRates_DateAndSource_FR009(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "GBP", "0.80", true)
	f.addCurrency(t, "EUR", "0.90", true)
	f.write(t, func(ctx context.Context, repo Repository) error {
		return repo.PutRateImport(ctx, RateImport{ID: "RI-1001", FileName: "gsap_rates_2026-10-19.csv", Feed: gsapFeedName,
			Meta: record.NewMeta(alex, start)})
	})
	f.addRate(t, "GBP", "2026-10-19", "0.79", RateGSAPImport, "RI-1001")
	f.addRate(t, "EUR", "2026-10-18", "0.92", RateManual, "")
	f.clock.now = october20

	infos, err := f.service.CurrentRates(t.Context())

	if err != nil || len(infos) != 3 {
		t.Fatalf("CurrentRates = %+v, %v", infos, err)
	}
	usd, eur, gbp := infos[0], infos[1], infos[2]
	if usd.Currency != "USD" || !usd.Rate.IsOne() || usd.RateDate != nil || usd.Source != nil || usd.ImportFileName != nil {
		t.Errorf("USD = %+v", usd)
	}
	if eur.Currency != "EUR" || eur.Rate.String() != "0.92" || eur.InverseRate.String() != "1.086957" || *eur.RateDate != date.MustParse("2026-10-18") ||
		*eur.Source != RateManual || eur.ImportFileName != nil {
		t.Errorf("EUR = %+v", eur)
	}
	if gbp.Currency != "GBP" || gbp.Rate.String() != "0.79" || *gbp.RateDate != date.MustParse("2026-10-19") || *gbp.Source != RateGSAPImport ||
		gbp.ImportFileName == nil || *gbp.ImportFileName != "gsap_rates_2026-10-19.csv" {
		t.Errorf("GBP = %+v", gbp)
	}
	if info, found, err := f.service.CurrentRate(t.Context(), "GBP"); err != nil || !found || info.Rate.String() != "0.79" {
		t.Errorf("CurrentRate(GBP) = %+v, %v, %v", info, found, err)
	}
}
