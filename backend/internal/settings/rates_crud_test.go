package settings

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// euroPlan is a plan planned in EUR with a budget of 1,200,000.00 EUR (FR-005's example), as 004 reports it.
func euroPlan(id string, inBin bool) Reference {
	budget := money.FromCents(120_000_000)
	return Reference{Record: history.RecordRef{Type: "plan", ID: id, Name: "Plan " + id}, InBin: inBin, Level: LevelPlan,
		PlanID: id, Currency: "EUR", Budget: &budget}
}

// rateFixture is EUR with a rate on 18 and 19 Oct 2026, on 20 Oct 2026.
func rateFixture(t *testing.T) (*fixture, ExchangeRate, ExchangeRate) {
	t.Helper()
	f := newTestService(t)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "EUR", "0.90", true)
	older := f.addRate(t, "EUR", "2026-10-18", "0.91", RateManual, "")
	latest := f.addRate(t, "EUR", "2026-10-19", "0.92", RateGSAPImport, "RI-1001")
	f.clock.now = october20
	return f, older, latest
}

// A rate needs a date the currency has no rate on yet and a rate above 0 with at most six decimals; every
// problem comes at once; USD has no rates. A new rate is entered by hand and logged with who added it
// (FR-001, FR-003, FR-049).
func TestAddRate_Rules_FR001(t *testing.T) {
	f, _, _ := rateFixture(t)

	_, taken := f.service.AddRate(t.Context(), alex, "EUR", RateInput{Date: "2026-10-19", Rate: "0"})
	_, unreadable := f.service.AddRate(t.Context(), alex, "EUR", RateInput{Date: "19/10/2026", Rate: "2000000"})
	_, usd := f.service.AddRate(t.Context(), alex, "USD", RateInput{Date: "2026-10-20", Rate: "1"})
	_, gone := f.service.AddRate(t.Context(), alex, "XXX", RateInput{Date: "2026-10-20", Rate: "1"})
	added, err := f.service.AddRate(t.Context(), daniel, "EUR", RateInput{Date: " 2027-01-01 ", Rate: "0.95"})

	assertError(t, taken, problems("date", "EUR already has a rate on 19 Oct 2026.", "rate", "Enter a rate above 0, with at most six decimals."))
	assertError(t, unreadable, problems("date", "Enter a date such as 2026-10-19.", "rate", "Use a rate of 1,000,000 or less."))
	assertError(t, usd, problems("code", "USD is always 1."))
	var notFound *errs.Error
	if !errors.As(gone, &notFound) || notFound.Code != errs.CodeNotFound {
		t.Errorf("unknown currency: %v", gone)
	}
	if err != nil || added.Currency != "EUR" || added.Date != date.MustParse("2027-01-01") || added.Rate.String() != "0.95" ||
		added.Source != RateManual || added.ImportID != nil || added.Current || !added.Future || added.CreatedBy != daniel {
		t.Fatalf("added = %+v, %v", added, err)
	}
	entry := f.lastEntry(t)
	if entry.RecordType != exchangeRateKind || entry.RecordID != added.ID || entry.RecordName != "EUR on 01 Jan 2027" ||
		entry.Action != history.ActionCreated || entry.Actor != daniel || entry.Details != "" {
		t.Errorf("log entry = %+v", entry)
	}
}

// Editing a rate changes its value only, makes an imported rate "Entered by hand" and logs both changes with
// who made them; an out-of-date version is refused (FR-003, FR-049).
func TestUpdateRate_SetsManual_FR003(t *testing.T) {
	f, _, latest := rateFixture(t)

	updated, err := f.service.UpdateRate(t.Context(), daniel, latest.ID, RateUpdate{Rate: "0.9230", Version: version(1)})
	_, stale := f.service.UpdateRate(t.Context(), alex, latest.ID, RateUpdate{Rate: "0.93", Version: version(1)})
	_, invalid := f.service.UpdateRate(t.Context(), alex, latest.ID, RateUpdate{Rate: "-1"})

	if err != nil || updated.Rate.String() != "0.923" || updated.Source != RateManual || updated.ImportID != nil ||
		updated.Date != latest.Date || !updated.Current || updated.Version != 2 || updated.UpdatedBy != daniel {
		t.Fatalf("updated = %+v, %v", updated, err)
	}
	assertError(t, stale, errs.VersionConflict("exchange rate", "Daniel Okoro", october20))
	assertError(t, invalid, problems("version", "Reload the page and try again.", "rate", "Enter a rate above 0, with at most six decimals."))
	entry := f.lastEntry(t)
	want := []history.Change{
		{Field: "rate", Label: "Rate (units per 1 USD)", Old: json.RawMessage(`"0.92"`), New: json.RawMessage(`"0.923"`), OldText: "0.92", NewText: "0.923"},
		{Field: "source", Label: "Source", Old: json.RawMessage(`"gsapImport"`), New: json.RawMessage(`"manual"`), OldText: "GSAP import",
			NewText: "Entered by hand"},
	}
	if entry.Action != history.ActionChanged || entry.Actor != daniel || entry.RecordName != "EUR on 19 Oct 2026" || !reflect.DeepEqual(entry.Changes, want) {
		t.Errorf("log entry = %+v", entry)
	}
	if rates, _ := f.service.Rates(t.Context()); rates["EUR"].String() != "0.923" {
		t.Errorf("current EUR rate = %v, want 0.923", rates["EUR"])
	}
}

// The currency's last rate dated today or earlier can't be deleted, even with a future rate (US1-5).
func TestDeleteRate_LastCurrentRefused_FR003(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "EUR", "0.90", true)
	only := f.addRate(t, "EUR", "2026-10-19", "0.92", RateManual, "")
	f.addRate(t, "EUR", "2027-01-01", "0.95", RateManual, "")
	f.clock.now = october20

	_, err := f.service.DeleteRate(t.Context(), alex, only.ID)

	assertError(t, err, problems("id", "EUR needs a rate dated today or earlier."))
	if rates := f.storedRates(t, "EUR"); len(rates) != 2 {
		t.Errorf("%d rates left, want 2", len(rates))
	}
}

// Other rates — a future one, or an older one — move to the recycle bin and come back with Undo (US1-5).
func TestDeleteRate_FutureAllowed(t *testing.T) {
	f, older, _ := rateFixture(t)
	future := f.addRate(t, "EUR", "2027-01-01", "0.95", RateManual, "")

	entry, err := f.service.DeleteRate(t.Context(), alex, future.ID)
	_, olderErr := f.service.DeleteRate(t.Context(), alex, older.ID)

	if err != nil || entry.RecordType != exchangeRateKind || entry.RecordName != "EUR on 01 Jan 2027" || entry.Parent == nil ||
		entry.Parent.ID != "EUR" {
		t.Fatalf("DeleteRate = %+v, %v", entry, err)
	}
	if olderErr != nil {
		t.Errorf("deleting the older rate: %v", olderErr)
	}
	if rates := f.storedRates(t, "EUR"); len(rates) != 1 {
		t.Errorf("%d rates left, want 1", len(rates))
	}
	if _, err := f.history.Undo(t.Context(), alex, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if rates := f.storedRates(t, "EUR"); len(rates) != 2 || rates[1].ID != future.ID {
		t.Errorf("after Undo: %+v", rates)
	}
}

// A deleted rate is not restored over a rate added on its date meanwhile (data-model §2.1, edge cases).
func TestRestoreRate_DateTakenRefused(t *testing.T) {
	f, _, latest := rateFixture(t)
	entry, err := f.service.DeleteRate(t.Context(), alex, latest.ID)
	if err != nil {
		t.Fatalf("DeleteRate: %v", err)
	}
	if _, err := f.service.AddRate(t.Context(), alex, "EUR", RateInput{Date: "2026-10-19", Rate: "0.93"}); err != nil {
		t.Fatalf("AddRate: %v", err)
	}

	_, err = f.history.Restore(t.Context(), daniel, entry.ID)

	var apiErr *errs.Error
	if !errors.As(err, &apiErr) || apiErr.Code != errs.CodeCannotRestore || apiErr.Message != "EUR already has a rate on 19 Oct 2026." {
		t.Errorf("Restore = %v", err)
	}
}

// A new currency gets its rate as its first dated rate, dated today and entered by hand (FR-003).
func TestCreateCurrency_FirstRateToday_FR003(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "USD", "1", true)
	f.clock.now = october20

	created, err := f.service.CreateCurrency(t.Context(), alex, jpy())

	rates := f.storedRates(t, "JPY")
	if err != nil || len(rates) != 1 || rates[0].Date != date.MustParse("2026-10-20") || rates[0].Rate.String() != "150" ||
		rates[0].Source != RateManual || rates[0].CreatedBy != alex {
		t.Fatalf("rates = %+v, %v", rates, err)
	}
	if created.Rate.String() != "150" || created.RateDate != date.MustParse("2026-10-20") || created.RateSource != RateManual ||
		created.RateImportFileName != nil || created.RateChangedBy != alex {
		t.Errorf("created = %+v", created)
	}
}

// A new rate in the currency form adds today's rate, or replaces it when today has one (FR-004); the change
// is logged on the rate with what it recalculated.
func TestUpdateCurrencyRate_UpsertsToday_FR004(t *testing.T) {
	f, _, _ := rateFixture(t)
	f.refs = []Reference{euroPlan("P-1004", false)}

	first, err := f.service.UpdateCurrency(t.Context(), daniel, "EUR", CurrencyUpdate{Rate: text("0.95"), Version: version(1)})
	if err != nil {
		t.Fatalf("first UpdateCurrency: %v", err)
	}
	added := f.lastEntry(t)
	second, err := f.service.UpdateCurrency(t.Context(), alex, "EUR", CurrencyUpdate{Rate: text("0.96"), Version: version(first.Version)})
	if err != nil {
		t.Fatalf("second UpdateCurrency: %v", err)
	}
	replaced := f.lastEntry(t)

	rates := f.storedRates(t, "EUR")
	if len(rates) != 3 || rates[2].Date != date.MustParse("2026-10-20") || rates[2].Rate.String() != "0.96" || rates[2].Source != RateManual {
		t.Fatalf("rates = %+v", rates)
	}
	if second.Rate.String() != "0.96" || second.RateDate != date.MustParse("2026-10-20") || second.RateChangedBy != alex {
		t.Errorf("currency = %+v", second)
	}
	if added.RecordType != exchangeRateKind || added.Action != history.ActionCreated || added.Actor != daniel ||
		added.Details != "1 plan planned in EUR: 1,304,347.83 → 1,263,157.89 USD." {
		t.Errorf("first entry = %+v", added)
	}
	if replaced.RecordType != exchangeRateKind || replaced.Action != history.ActionChanged || replaced.Actor != alex ||
		len(replaced.Changes) != 1 || replaced.Changes[0].OldText != "0.95" || replaced.Changes[0].NewText != "0.96" {
		t.Errorf("second entry = %+v", replaced)
	}
}

// Before a rate that changes the current rate is saved, the impact names the line items and the plans planned
// in the currency with their USD values before and after (FR-005, US1-4).
func TestRateImpact_LineItemsAndPlans_FR005(t *testing.T) {
	f, _, latest := rateFixture(t)
	f.write(t, func(ctx context.Context, repo Repository) error {
		r := latest
		r.Rate = money.RateFromMicros(920_000) // 0.92, as in US4-3
		return repo.PutRate(ctx, r)
	})
	f.refs = append(euroLineItems(), euroPlan("P-1004", false), euroPlan("P-1005", true))

	added, err := f.service.AddRateImpact(t.Context(), alex, "EUR", RateInput{Date: "2026-10-20", Rate: "0.95"})
	edited, editErr := f.service.UpdateRateImpact(t.Context(), alex, latest.ID, RateUpdate{Rate: "0.95", Version: version(1)})
	_, invalid := f.service.AddRateImpact(t.Context(), alex, "EUR", RateInput{Date: "2026-10-19", Rate: "0.95"})

	want := []ImpactWarning{
		{Kind: ImpactAmountsRecalculated, Count: 14, Message: "14 line items in 6 activities use EUR. Their EUR amounts stay the same; their USD amounts will be recalculated (52,173.91 → 50,526.32 USD). Unassigned planned cost of these activities will change. No activity goes back for approval."},
		{Kind: ImpactAmountsRecalculated, Count: 1, Message: "1 plan is planned in EUR. Its EUR amounts stay the same; its USD values change (1,304,347.83 → 1,263,157.89 USD)."},
	}
	if err != nil || !reflect.DeepEqual(added.Warnings, want) {
		t.Errorf("AddRateImpact = %+v, %v", added.Warnings, err)
	}
	if editErr != nil || !reflect.DeepEqual(edited.Warnings, want) {
		t.Errorf("UpdateRateImpact = %+v, %v", edited.Warnings, editErr)
	}
	assertError(t, invalid, problems("date", "EUR already has a rate on 19 Oct 2026."))
	if rates := f.storedRates(t, "EUR"); len(rates) != 2 {
		t.Errorf("the impact check saved a rate: %+v", rates)
	}
}

// A future-dated rate changes no conversion before its date: no warning. Editing a rate that is not the
// current one warns of nothing either (FR-005).
func TestRateImpact_FutureRateNoWarning(t *testing.T) {
	f, older, _ := rateFixture(t)
	f.refs = append(euroLineItems(), euroPlan("P-1004", false))

	future, err := f.service.AddRateImpact(t.Context(), alex, "EUR", RateInput{Date: "2027-01-01", Rate: "0.95"})
	past, pastErr := f.service.UpdateRateImpact(t.Context(), alex, older.ID, RateUpdate{Rate: "0.80", Version: version(1)})

	if err != nil || len(future.Warnings) != 0 {
		t.Errorf("future rate impact = %+v, %v", future, err)
	}
	if pastErr != nil || len(past.Warnings) != 0 {
		t.Errorf("older rate impact = %+v, %v", past, pastErr)
	}
}

// The rate history lists the newest date first, marks the current rate and future rates, and names the
// import of an imported rate (FR-003).
func TestListRates_History_FR003(t *testing.T) {
	f, older, latest := rateFixture(t)
	future := f.addRate(t, "EUR", "2027-01-01", "0.95", RateManual, "")
	f.clock.now = october20.Add(time.Hour)

	page, err := f.service.ListRates(t.Context(), alex, "EUR")

	if err != nil || page.Total != 3 || len(page.Items) != 3 {
		t.Fatalf("ListRates = %+v, %v", page, err)
	}
	ids := []string{page.Items[0].ID, page.Items[1].ID, page.Items[2].ID}
	if !reflect.DeepEqual(ids, []string{future.ID, latest.ID, older.ID}) {
		t.Errorf("order = %q", ids)
	}
	if !page.Items[0].Future || page.Items[0].Current || !page.Items[1].Current || page.Items[2].Current ||
		page.Items[1].ImportID == nil || *page.Items[1].ImportID != "RI-1001" || page.Current.Rate.String() != "0.92" {
		t.Errorf("page = %+v", page)
	}
}
