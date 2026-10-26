package settings_test

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/apitest"
)

// The handler tests of 016's exchange rates and Field help run the real API on the demo data — EUR, GBP, CNY
// and INR with 31 days of simulated GSAP rates up to the seed day, 12 Mar 2026 (data-model §8) — and check
// every request and answer against the contract.

// rateRecord is the part of an ExchangeRateRecord the tests read.
type rateRecord struct {
	ID, Currency, Date, Rate, Source string
	ImportFileName                   *string
	Current, Future                  bool
	Version                          int
}

// A currency's rate history comes newest first with the current rate marked; the current rates come USD
// first with their date and source (FR-001, FR-003, FR-009).
func TestRatesAPI_HistoryAndCurrent_FR003_FR009(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	var history struct {
		Items   []rateRecord
		Total   int
		Current struct{ Currency, Rate, RateDate, Source, ImportFileName string }
	}
	alex.Do(t, http.MethodGet, "/api/v1/currencies/EUR/rates", nil).Decode(t, &history)
	var current struct {
		Today string
		Items []struct {
			Currency, Rate string
			RateDate       *string
			Source         *string
		}
		Total int
	}
	alex.Do(t, http.MethodGet, "/api/v1/exchange-rates/current", nil).Decode(t, &current)
	gone := alex.Do(t, http.MethodGet, "/api/v1/currencies/XXX/rates", nil)
	var usd struct{ Items []rateRecord }
	alex.Do(t, http.MethodGet, "/api/v1/currencies/USD/rates", nil).Decode(t, &usd)

	if history.Total != 31 || len(history.Items) != 31 {
		t.Fatalf("EUR history = %d rates, want 31", history.Total)
	}
	latest, oldest := history.Items[0], history.Items[30]
	if latest.Date != "2026-03-12" || latest.Rate != "0.92" || !latest.Current || latest.Future || latest.Source != "gsapImport" ||
		latest.ImportFileName == nil || *latest.ImportFileName != "gsap_rates_seed.csv" || oldest.Date != "2026-02-10" || oldest.Current {
		t.Errorf("history = %+v … %+v", latest, oldest)
	}
	if history.Current.Rate != "0.92" || history.Current.RateDate != "2026-03-12" || history.Current.ImportFileName != "gsap_rates_seed.csv" {
		t.Errorf("current = %+v", history.Current)
	}
	if current.Today != "2026-03-12" || current.Total != 5 || current.Items[0].Currency != "USD" || current.Items[0].RateDate != nil ||
		current.Items[2].Currency != "EUR" || current.Items[2].Rate != "0.92" || *current.Items[2].Source != "gsapImport" {
		t.Errorf("current rates = %+v", current)
	}
	assertAnswer(t, gone, http.StatusNotFound, currencyGoneBody)
	if len(usd.Items) != 0 {
		t.Errorf("USD history = %+v, want none", usd.Items)
	}
}

// Add rate answers 201; its impact check, Edit (with its impact check and a stale version) and Delete with
// Undo follow the rules of the service (FR-003, FR-005).
func TestRatesAPI_AddUpdateDelete_FR003_FR005(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	impact := alex.Do(t, http.MethodPost, "/api/v1/currencies/EUR/rates/impact", map[string]any{"date": "2027-01-01", "rate": "0.95"})
	invalid := alex.Do(t, http.MethodPost, "/api/v1/currencies/EUR/rates", map[string]any{"date": "2026-03-12", "rate": "0"})
	var added rateRecord
	mustCreate(t, alex, "/api/v1/currencies/EUR/rates", map[string]any{"date": "2027-01-01", "rate": "0.95"}).Decode(t, &added)
	editImpact := alex.Do(t, http.MethodPost, "/api/v1/exchange-rates/"+added.ID+"/impact", map[string]any{"rate": "0.96", "version": 1})
	var edited rateRecord
	alex.Do(t, http.MethodPatch, "/api/v1/exchange-rates/"+added.ID, map[string]any{"rate": "0.96", "version": 1}).Decode(t, &edited)
	stale := alex.Do(t, http.MethodPatch, "/api/v1/exchange-rates/"+added.ID, map[string]any{"rate": "0.97", "version": 1})
	var deleted struct {
		RecycleBinEntry struct{ ID, RecordType, RecordName string }
	}
	alex.Do(t, http.MethodDelete, "/api/v1/exchange-rates/"+added.ID, nil).Decode(t, &deleted)
	gone := alex.Do(t, http.MethodDelete, "/api/v1/exchange-rates/XR-9999", nil)
	undone := alex.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+deleted.RecycleBinEntry.ID+"/undo", nil)

	assertAnswer(t, impact, http.StatusOK, `{"warnings":[]}`)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"date","message":"EUR already has a rate on 12 Mar 2026."},`+
		`{"field":"rate","message":"Enter a rate above 0, with at most six decimals."}`))
	if added.Currency != "EUR" || added.Date != "2027-01-01" || added.Rate != "0.95" || added.Source != "manual" || !added.Future || added.Current {
		t.Errorf("added = %+v", added)
	}
	assertAnswer(t, editImpact, http.StatusOK, `{"warnings":[]}`)
	if edited.Rate != "0.96" || edited.Version != 2 {
		t.Errorf("edited = %+v", edited)
	}
	if stale.Status != http.StatusConflict {
		t.Errorf("stale edit = %d %s", stale.Status, stale.Body)
	}
	if deleted.RecycleBinEntry.RecordType != "exchangeRate" || deleted.RecycleBinEntry.RecordName != "EUR on 01 Jan 2027" {
		t.Errorf("deleted = %+v", deleted)
	}
	assertAnswer(t, gone, http.StatusNotFound,
		`{"error":{"code":"not_found","message":"This exchange rate no longer exists — someone may have deleted it.","fields":[]}}`)
	if undone.Status != http.StatusOK {
		t.Errorf("undo = %d %s", undone.Status, undone.Body)
	}
}

// The sample file is a CSV of today's rates; its review and import answer every row and the import record;
// Recent imports lists the newest first; a second import has nothing to import (FR-006, FR-007).
func TestRatesAPI_ImportCheckAndImport_FR006_FR007(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	server.Advance(24 * time.Hour) // 13 Mar 2026: a day without a rate yet
	alex := server.SignIn(t, alexsEmail)

	sample := alex.Do(t, http.MethodGet, "/api/v1/exchange-rates/import-sample", nil)
	body := map[string]any{"fileName": "gsap_rates_2026-03-13.csv", "content": base64.StdEncoding.EncodeToString(sample.Body)}
	var check struct {
		Feed string
		Rows []struct {
			Row                     int
			Outcome, Currency, Rate string
		}
		Counts struct{ Add, Replace, Present, NotUsed, Problem int }
	}
	alex.Do(t, http.MethodPost, "/api/v1/exchange-rates/import-check", body).Decode(t, &check)
	var imported struct {
		ID, FileName, Feed, FirstDate, LastDate string
		Added                                   int
		ImportedBy                              struct{ Name string }
	}
	mustCreate(t, alex, "/api/v1/exchange-rates/import", body).Decode(t, &imported)
	again := alex.Do(t, http.MethodPost, "/api/v1/exchange-rates/import", body)
	unreadable := alex.Do(t, http.MethodPost, "/api/v1/exchange-rates/import-check",
		map[string]any{"fileName": "x.csv", "content": base64.StdEncoding.EncodeToString([]byte("rate_type,rate\nM,1\n"))})
	var recent struct {
		Items []struct{ ID string }
		Total int
	}
	alex.Do(t, http.MethodGet, "/api/v1/exchange-rates/imports?limit=5", nil).Decode(t, &recent)
	badLimit := alex.Do(t, http.MethodGet, "/api/v1/exchange-rates/imports?limit=0", nil, apitest.SkipRequestValidation())

	wantSample := "rate_type,from_currency,to_currency,valid_from,rate\nM,USD,CNY,2026-03-13,7.1355\nM,USD,EUR,2026-03-13,0.9246\n" +
		"M,USD,GBP,2026-03-13,0.79395\nM,USD,INR,2026-03-13,83.9175\n"
	if sample.Status != http.StatusOK || string(sample.Body) != wantSample || !strings.HasPrefix(sample.Header.Get("Content-Type"), "text/csv") ||
		sample.Header.Get("Content-Disposition") != `attachment; filename="gsap_rates_2026-03-13.csv"` {
		t.Errorf("sample = %d %q %v", sample.Status, sample.Body, sample.Header)
	}
	if check.Feed != "gsapDailyRatesFile" || check.Counts.Add != 4 || len(check.Rows) != 4 || check.Rows[0].Row != 2 || check.Rows[0].Outcome != "add" {
		t.Errorf("check = %+v", check)
	}
	if imported.ID != "RI-1002" || imported.Added != 4 || imported.FirstDate != "2026-03-13" || imported.ImportedBy.Name != "Alex Morgan" {
		t.Errorf("imported = %+v", imported)
	}
	assertAnswer(t, again, http.StatusBadRequest,
		validationBody(`{"field":"content","message":"Nothing to import: every rate is already present or not used."}`))
	assertAnswer(t, unreadable, http.StatusBadRequest, validationBody(`{"field":"content","message":"The file needs the column 'from_currency'."}`))
	if recent.Total != 2 || recent.Items[0].ID != "RI-1002" || recent.Items[1].ID != "RI-1001" {
		t.Errorf("recent imports = %+v", recent)
	}
	if badLimit.Status != http.StatusBadRequest {
		t.Errorf("limit 0 = %d %s", badLimit.Status, badLimit.Body)
	}
}

// Field help lists every entry with its default and, with the demo data, no Shell attribute without help;
// an edit answers the edited entry, a stale version 409, a reset the default again, an unknown key 404
// (FR-045, FR-046, FR-048).
func TestFieldHelpAPI_ListAndUpdate_FR045_FR046_FR048(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	var page struct {
		Items []struct {
			Key, Level, Label, Text, DefaultText string
			IsDefault                            bool
			Version                              int
		}
		Total, ShellAttributesWithoutHelp int
	}
	alex.Do(t, http.MethodGet, "/api/v1/field-help", nil).Decode(t, &page)
	edited := alex.Do(t, http.MethodPatch, "/api/v1/field-help/plan.budget", map[string]any{"text": "The plan's money.", "version": 0})
	stale := alex.Do(t, http.MethodPatch, "/api/v1/field-help/plan.budget", map[string]any{"text": "Daniel's.", "version": 0}) // settings.manage: administrators only (014)
	reset := alex.Do(t, http.MethodPatch, "/api/v1/field-help/plan.budget", map[string]any{"text": nil, "version": 1})
	tooLong := alex.Do(t, http.MethodPatch, "/api/v1/field-help/plan.name", map[string]any{"text": strings.Repeat("a", 301), "version": 0},
		apitest.SkipRequestValidation())
	unknown := alex.Do(t, http.MethodPatch, "/api/v1/field-help/plan.colour", map[string]any{"text": "x", "version": 0})

	if page.Total < 90 || page.Items[0].Key != "plan.name" || page.Items[0].Level != "plan" || !page.Items[0].IsDefault ||
		page.Items[0].Version != 0 || page.ShellAttributesWithoutHelp != 0 {
		t.Errorf("page = %d entries, first %+v, %d attributes without help", page.Total, page.Items[0], page.ShellAttributesWithoutHelp)
	}
	assertAnswer(t, edited, http.StatusOK, `{"key":"plan.budget","level":"plan","label":"Plan budget","text":"The plan's money.",`+
		`"defaultText":"The money this plan may spend in its year, in the plan currency. Campaign budgets are allocated from it.",`+
		`"isDefault":false,"version":1,"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1001","name":"Alex Morgan"}}`)
	assertAnswer(t, stale, http.StatusConflict, `{"error":{"code":"version_conflict","message":"Not saved — Alex Morgan changed this field help",`+
		`"fields":[],"conflict":{"changedBy":"Alex Morgan","changedAt":"2026-03-12T09:00:00Z"}}}`)
	assertAnswer(t, reset, http.StatusOK, `{"key":"plan.budget","level":"plan","label":"Plan budget",`+
		`"text":"The money this plan may spend in its year, in the plan currency. Campaign budgets are allocated from it.",`+
		`"defaultText":"The money this plan may spend in its year, in the plan currency. Campaign budgets are allocated from it.",`+
		`"isDefault":true,"version":0,"updatedAt":null,"updatedBy":null}`)
	assertAnswer(t, tooLong, http.StatusBadRequest, validationBody(`{"field":"text","message":"Use 300 characters or fewer."}`))
	assertAnswer(t, unknown, http.StatusNotFound, `{"error":{"code":"not_found","message":"This help entry doesn't exist.","fields":[]}}`)
}
