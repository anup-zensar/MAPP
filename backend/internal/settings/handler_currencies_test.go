package settings_test

import (
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"

	"mapp/internal/platform/apitest"
)

// The currencies come USD first, then by code, with the rate both ways and their locks; a new currency
// answers 201 and is listed; every problem of a new currency comes at once (FR-028, US4-1, US4-5).
func TestCurrenciesAPI_ListCreateGet_FR028(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	gone := alex.Do(t, http.MethodGet, "/api/v1/currencies/JPY", nil)
	created := mustCreate(t, alex, "/api/v1/currencies", jpyBody)
	invalid := alex.Do(t, http.MethodPost, "/api/v1/currencies", map[string]any{"code": "EURO", "name": "Euro", "rate": "0"})
	var list struct {
		Items []struct{ Code string }
	}
	alex.Do(t, http.MethodGet, "/api/v1/currencies", nil).Decode(t, &list)
	var euro, dollar struct {
		Rate, InverseRate string
		Locks             []struct{ Field, Reason string }
	}

	alex.Do(t, http.MethodGet, "/api/v1/currencies/EUR", nil).Decode(t, &euro)
	alex.Do(t, http.MethodGet, "/api/v1/currencies/USD", nil).Decode(t, &dollar)

	assertAnswer(t, created, http.StatusCreated, jpyAnswer(1, `"rateChangedAt":"2026-03-12T09:00:00Z","rateChangedBy":{"id":"U-1001","name":"Alex Morgan"}`, true))
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"code","message":"Enter a three-letter code such as JPY."},`+
		`{"field":"name","message":"A currency called 'Euro' already exists."},`+
		`{"field":"rate","message":"Enter a rate above 0, with at most six decimals."}`))
	var codes []string
	for _, c := range list.Items {
		codes = append(codes, c.Code)
	}
	if !slices.Equal(codes, []string{"USD", "CNY", "EUR", "GBP", "INR", "JPY"}) {
		t.Errorf("currencies = %q", codes)
	}
	if euro.Rate != "0.92" || euro.InverseRate != "1.086957" || len(euro.Locks) != 1 || euro.Locks[0].Field != "code" {
		t.Errorf("EUR = %+v", euro)
	}
	if len(dollar.Locks) != 4 || dollar.Locks[1].Reason != "USD is SP&A One's base currency; its rate is always 1." {
		t.Errorf("USD = %+v", dollar)
	}
	assertAnswer(t, gone, http.StatusNotFound, `{"error":{"code":"not_found","message":"This currency no longer exists — someone may have deleted it.","fields":[]}}`)
}

// Bodies of the currency tests.
const (
	currencyGoneBody = `{"error":{"code":"not_found","message":"This currency no longer exists — someone may have deleted it.","fields":[]}}`
)

// jpyBody is the new currency of US4-1.
var jpyBody = map[string]any{"code": "jpy", "name": "Japanese yen", "symbol": "¥", "rate": "150"}

// jpyAnswer is the answer for the currency jpyBody at version, with who changed its rate and its status;
// Alex created it at the start.
func jpyAnswer(version int, rateChanged string, active bool) string {
	return `{"code":"JPY","name":"Japanese yen","symbol":"¥","rate":"150","inverseRate":"0.006667",` + rateChanged +
		`,"rateDate":"2026-03-12","rateSource":"manual","rateImportFileName":null,"active":` + strconv.FormatBool(active) + `,"usage":` + unusedBody +
		`,"locks":[{"field":"code","reason":"The code can't change once the currency is saved."}],"version":` + itoa(version) + `,` + alexAtStart + `}`
}

// A save from an out-of-date copy is refused with who changed the currency and when; a new rate marks who
// changed it (FR-007, FR-028).
func TestCurrenciesAPI_UpdateConflict_FR007(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	daniel := server.SignIn(t, danielsEmail)
	server.GrantRole(t, danielsEmail, "administrator", "all") // 014: changing Settings
	mustCreate(t, alex, "/api/v1/currencies", jpyBody)
	server.Advance(90 * time.Minute)
	var daniels struct {
		Rate          string
		RateChangedBy struct{ Name string }
		Version       int
	}

	daniel.Do(t, http.MethodPatch, "/api/v1/currencies/JPY", map[string]any{"rate": "151", "version": 1}).Decode(t, &daniels)
	alexs := alex.Do(t, http.MethodPatch, "/api/v1/currencies/JPY", map[string]any{"name": "Yen", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/v1/currencies/XXX", map[string]any{"name": "X", "version": 1})

	if daniels.Rate != "151" || daniels.RateChangedBy.Name != "Daniel Okoro" || daniels.Version != 2 {
		t.Errorf("Daniel's update = %+v", daniels)
	}
	assertAnswer(t, alexs, http.StatusConflict, `{"error":{"code":"version_conflict","message":"Not saved — Daniel Okoro changed this currency",`+
		`"fields":[],"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T10:30:00Z"}}}`)
	assertAnswer(t, gone, http.StatusNotFound, currencyGoneBody)
}

// The impact check validates like the update and answers the warnings, saving nothing (FR-030, D-044).
func TestCurrenciesAPI_Impact_FR030(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/v1/currencies", jpyBody)

	unused := alex.Do(t, http.MethodPost, "/api/v1/currencies/JPY/impact", map[string]any{"rate": "151", "version": 1})
	invalid := alex.Do(t, http.MethodPost, "/api/v1/currencies/JPY/impact", map[string]any{"rate": "0", "version": 1})
	gone := alex.Do(t, http.MethodPost, "/api/v1/currencies/XXX/impact", map[string]any{"version": 1})

	assertAnswer(t, unused, http.StatusOK, `{"warnings":[]}`)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"rate","message":"Enter a rate above 0, with at most six decimals."}`))
	assertAnswer(t, gone, http.StatusNotFound, currencyGoneBody)
	assertAnswer(t, alex.Do(t, http.MethodGet, "/api/v1/currencies/JPY", nil), http.StatusOK,
		jpyAnswer(1, `"rateChangedAt":"2026-03-12T09:00:00Z","rateChangedBy":{"id":"U-1001","name":"Alex Morgan"}`, true))
}

// Set inactive and Reactivate answer the currency with its new status; an out-of-date version is refused
// (FR-004).
func TestCurrenciesAPI_Status_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/v1/currencies", jpyBody)
	rateChanged := `"rateChangedAt":"2026-03-12T09:00:00Z","rateChangedBy":{"id":"U-1001","name":"Alex Morgan"}`

	inactive := alex.Do(t, http.MethodPost, "/api/v1/currencies/JPY/status", map[string]any{"active": false, "version": 1})
	stale := alex.Do(t, http.MethodPost, "/api/v1/currencies/JPY/status", map[string]any{"active": true, "version": 1})
	active := alex.Do(t, http.MethodPost, "/api/v1/currencies/JPY/status", map[string]any{"active": true, "version": 2})

	assertAnswer(t, inactive, http.StatusOK, jpyAnswer(2, rateChanged, false))
	if stale.Status != http.StatusConflict || active.Status != http.StatusOK {
		t.Errorf("stale %d %s; reactivate %d %s", stale.Status, stale.Body, active.Status, active.Body)
	}
}

// An unused currency moves to the recycle bin, listed by its code, and Undo brings it back (FR-004).
func TestCurrenciesAPI_DeleteAndUndo_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/v1/currencies", jpyBody)

	deleted := alex.Do(t, http.MethodDelete, "/api/v1/currencies/JPY", nil)
	afterDelete := alex.Do(t, http.MethodGet, "/api/v1/currencies/JPY", nil)
	undone := alex.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/undo", nil)
	gone := alex.Do(t, http.MethodDelete, "/api/v1/currencies/XXX", nil)

	assertAnswer(t, deleted, http.StatusOK, `{"recycleBinEntry":{"id":"BIN-1001","recordType":"currency","recordTypeLabel":"currency",`+
		`"recordId":"JPY","recordName":"JPY","path":[],"parent":null,"deletedAt":"2026-03-12T09:00:00Z",`+
		`"deletedBy":{"id":"U-1001","name":"Alex Morgan"},"contents":[{"recordType":"exchangeRate","label":"exchange rate","count":1}]}}`)
	assertAnswer(t, afterDelete, http.StatusNotFound, currencyGoneBody)
	assertAnswer(t, undone, http.StatusOK, `{"recordType":"currency","recordTypeLabel":"currency","recordId":"JPY","recordName":"JPY","details":""}`)
	assertAnswer(t, gone, http.StatusNotFound, currencyGoneBody)
}
