package settings

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// usdLocked is the reason of every USD lock (FR-028).
const usdLocked = "USD is NorthStar's base currency; its rate is always 1."

// jpy is the new currency of US4-1.
func jpy() CurrencyInput {
	return CurrencyInput{Code: " jpy ", Name: "Japanese yen", Symbol: "¥", Rate: "150"}
}

// euroLineItems are the line items of US4-3: 14 in EUR in 6 activities, 48,000.00 EUR in all (13 × 874.00 and
// 36,638.00), so 52,173.91 USD at 0.92 and 50,526.32 USD at 0.95; and, not counted, one EUR line item in
// the recycle bin and one in GBP.
func euroLineItems() []Reference {
	refs := []Reference{}
	for i := range 13 {
		refs = append(refs, lineItemRef(fmt.Sprintf("L-10%02d", i), "P-1001", fmt.Sprintf("A-100%d", i%5), "EUR", "874.00", false))
	}
	return append(refs,
		lineItemRef("L-1013", "P-1001", "A-1005", "EUR", "36638.00", false),
		lineItemRef("L-1014", "P-1001", "A-1006", "EUR", "1000.00", true),
		lineItemRef("L-1015", "P-1001", "A-1007", "GBP", "1000.00", false),
	)
}

// A new currency is listed after USD, active, with its rate both ways and who set the rate (US4-1, FR-028).
func TestCreateCurrency_JPY_FR028(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "EUR", "0.92", true)

	created, err := f.service.CreateCurrency(t.Context(), alex, jpy())
	if err != nil {
		t.Fatalf("CreateCurrency: %v", err)
	}

	if created.Code != "JPY" || created.Name != "Japanese yen" || created.Symbol != "¥" || created.Rate.String() != "150" ||
		created.InverseRate.String() != "0.006667" || !created.Active || !created.RateChangedAt.Equal(start) || created.RateChangedBy != alex ||
		!reflect.DeepEqual(created.Locks, []Lock{{Field: "code", Reason: "The code can't change once the currency is saved."}}) || created.Usage.InUse {
		t.Errorf("created = %+v", created)
	}
	listed, _ := f.service.ListCurrencies(t.Context(), alex)
	if len(listed) != 3 || listed[0].Code != "USD" || listed[1].Code != "EUR" || listed[2].Code != "JPY" {
		t.Errorf("list = %+v", listed)
	}
	if rates, _ := f.service.Rates(t.Context()); rates["JPY"].String() != "150" {
		t.Errorf("rates = %v", rates)
	}
	if entry := f.lastEntry(t); entry.Action != history.ActionCreated || entry.RecordType != "currency" || entry.RecordName != "JPY" {
		t.Errorf("entry = %+v", entry)
	}
}

// Every problem of a new currency is shown at once (US4-5, FR-028).
func TestCreateCurrency_AllProblemsAtOnce_FR028(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "EUR", "0.92", true)

	_, first := f.service.CreateCurrency(t.Context(), alex, CurrencyInput{Code: "EURO", Name: " ", Symbol: "€€€€€€", Rate: "0"})
	_, second := f.service.CreateCurrency(t.Context(), alex, CurrencyInput{Code: " eur ", Name: "eur NAME", Rate: "2000000"})
	_, third := f.service.CreateCurrency(t.Context(), alex, CurrencyInput{Code: "E1R", Name: "Other", Rate: "0.1234567"})

	assertError(t, first, problems("code", "Enter a three-letter code such as JPY.", "name", "Enter a name.",
		"symbol", "Use 5 characters or fewer.", "rate", "Enter a rate above 0, with at most six decimals."))
	assertError(t, second, problems("code", "A currency with the code 'EUR' already exists.", "name", "A currency called 'EUR name' already exists.",
		"rate", "Use a rate of 1,000,000 or less."))
	assertError(t, third, problems("code", "Enter a three-letter code such as JPY.", "rate", "Enter a rate above 0, with at most six decimals."))
	if len(f.logEntries(t)) != 0 {
		t.Errorf("refused creates were logged")
	}
}

// The code of a saved currency is locked: views say so, and an update has no code to change (FR-028).
func TestCurrency_CodeLocked_FR028(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "EUR", "0.92", true)

	view, err := f.service.GetCurrency(t.Context(), alex, "EUR")
	renamed, err2 := f.service.UpdateCurrency(t.Context(), alex, "EUR", CurrencyUpdate{Name: text("Euro"), Symbol: text("€"), Version: version(1)})
	_, gone := f.service.GetCurrency(t.Context(), alex, "JPY")

	if err != nil || len(view.Locks) != 1 || view.Locks[0] != (Lock{Field: "code", Reason: "The code can't change once the currency is saved."}) {
		t.Errorf("EUR = %+v, %v", view, err)
	}
	if err2 != nil || renamed.Code != "EUR" || renamed.Name != "Euro" || renamed.Symbol != "€" || renamed.Version != 2 {
		t.Errorf("renamed = %+v, %v", renamed, err2)
	}
	assertError(t, gone, errs.NotFound("This currency no longer exists — someone may have deleted it."))
}

// USD's rate, status and delete are locked; its name and symbol may change (US4-4, FR-028).
func TestUSD_RateStatusDeleteLocked_FR028(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "USD", "1", true)

	_, rate := f.service.UpdateCurrency(t.Context(), alex, "USD", CurrencyUpdate{Rate: text("2"), Version: version(1)})
	_, status := f.service.SetCurrencyActive(t.Context(), alex, "USD", StatusUpdate{Active: yes(false), Version: version(1)})
	_, deleted := f.service.DeleteCurrency(t.Context(), alex, "USD")
	renamed, err := f.service.UpdateCurrency(t.Context(), alex, "USD", CurrencyUpdate{Name: text("US dollar"), Rate: text("1.000"), Version: version(1)})

	assertError(t, rate, problems("rate", usdLocked))
	assertError(t, status, problems("active", usdLocked))
	assertError(t, deleted, errs.InUse(usdLocked))
	if err != nil || renamed.Name != "US dollar" || !renamed.Rate.IsOne() || len(renamed.Locks) != 4 || renamed.Locks[1] != (Lock{Field: "rate", Reason: usdLocked}) {
		t.Errorf("renamed = %+v, %v", renamed, err)
	}
}

// Before a new rate is saved, the warning says how many live line items in how many activities use the
// currency and their USD total before and after; nothing is saved (US4-3, FR-030).
func TestCurrencyImpact_RateChange_FR030(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "EUR", "0.92", true)
	f.refs = euroLineItems()

	impact, err := f.service.CurrencyImpact(t.Context(), alex, "EUR", CurrencyUpdate{Rate: text("0.95"), Version: version(1)})
	named, err2 := f.service.CurrencyImpact(t.Context(), alex, "EUR", CurrencyUpdate{Name: text("Euro"), Version: version(1)})
	_, invalid := f.service.CurrencyImpact(t.Context(), alex, "EUR", CurrencyUpdate{Rate: text("-1"), Version: version(1)})

	want := []ImpactWarning{{Kind: ImpactAmountsRecalculated, Count: 14, Message: "14 line items in 6 activities use EUR. Their EUR amounts stay " +
		"the same; their USD amounts will be recalculated (52,173.91 → 50,526.32 USD). Unassigned planned cost of these activities will " +
		"change. No activity goes back for approval."}}
	if err != nil || !reflect.DeepEqual(impact.Warnings, want) {
		t.Errorf("impact = %+v, %v", impact, err)
	}
	if err2 != nil || len(named.Warnings) != 0 {
		t.Errorf("a new name = %+v, %v", named, err2)
	}
	assertError(t, invalid, problems("rate", "Enter a rate above 0, with at most six decimals."))
	if rates, _ := f.service.Rates(t.Context()); rates["EUR"].String() != "0.92" {
		t.Errorf("the impact check saved the rate: %v", rates)
	}
}

// A new rate is logged with the old and new rate and how many line items were recalculated, and marks who
// changed the rate and when; a name change does not (FR-006, FR-030).
func TestUpdateCurrency_RateChangeLoggedWithDetails_FR006_FR030(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "EUR", "0.92", true)
	f.refs = euroLineItems()
	f.clock.now = start.Add(time.Hour)

	updated, err := f.service.UpdateCurrency(t.Context(), daniel, "EUR", CurrencyUpdate{Rate: text("0.95"), Version: version(1)})
	if err != nil {
		t.Fatalf("UpdateCurrency: %v", err)
	}
	entry := f.lastEntry(t)
	f.clock.now = start.Add(2 * time.Hour)
	renamed, err := f.service.UpdateCurrency(t.Context(), alex, "EUR", CurrencyUpdate{Name: text("Euro"), Version: version(2)})

	if updated.Rate.String() != "0.95" || !updated.RateChangedAt.Equal(start.Add(time.Hour)) || updated.RateChangedBy != daniel || updated.Version != 2 {
		t.Errorf("updated = %+v", updated)
	}
	wantChanges := []history.Change{{Field: "rate", Label: "Rate (units per 1 USD)", Old: json.RawMessage(`"0.92"`), New: json.RawMessage(`"0.95"`),
		OldText: "0.92", NewText: "0.95"}}
	if entry.Details != "14 line items in 6 activities recalculated: 52,173.91 → 50,526.32 USD." || !reflect.DeepEqual(entry.Changes, wantChanges) {
		t.Errorf("entry = %+v", entry)
	}
	if err != nil || renamed.RateChangedBy != daniel || f.lastEntry(t).Details != "" {
		t.Errorf("renamed = %+v, %v", renamed, err)
	}
	_, stale := f.service.UpdateCurrency(t.Context(), alex, "EUR", CurrencyUpdate{Rate: text("1"), Version: version(1)})
	assertError(t, stale, errs.VersionConflict("currency", "Alex Morgan", start.Add(2*time.Hour)))
}

// The rates are every currency's current rate, active or not; a change applies at once (FR-029).
func TestRates_CurrentRateOfEveryCurrency_FR029(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "EUR", "0.92", true)
	f.addCurrency(t, "GBP", "0.79", false)

	if _, err := f.service.UpdateCurrency(t.Context(), alex, "EUR", CurrencyUpdate{Rate: text("0.95"), Version: version(1)}); err != nil {
		t.Fatalf("UpdateCurrency: %v", err)
	}
	rates, err := f.service.Rates(t.Context())

	if err != nil || len(rates) != 3 || rates["EUR"].String() != "0.95" || rates["GBP"].String() != "0.79" || !rates["USD"].IsOne() {
		t.Errorf("rates = %v, %v", rates, err)
	}
	if usd, _ := rates.ToUSD(money.Money{Amount: money.FromCents(95_00), Currency: "EUR"}); usd.String() != "100.00" {
		t.Errorf("95.00 EUR = %s USD", usd)
	}
}

// A currency counts the line items in it and the people showing amounts in it, those in the recycle bin
// too (US4-2, FR-003).
func TestCurrencyUsage_LineItemsAndPeople_FR003(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "EUR", "0.92", true)
	f.refs = append(euroLineItems(), personRef("U-1001", "EUR"), personRef("U-1002", "EUR"))

	view, err := f.service.GetCurrency(t.Context(), alex, "EUR")

	if err != nil || view.Usage.Summary != "Used by 15 line items (1 in the recycle bin) · Display currency of 2 people" || len(view.Usage.Parts) != 2 {
		t.Errorf("usage = %+v, %v", view.Usage, err)
	}
}

// A currency in use cannot be deleted: the answer says where it is used and to set it inactive instead
// (FR-004).
func TestDeleteCurrency_InUseRefused_FR004(t *testing.T) {
	f := newTestService(t)
	f.addCurrency(t, "EUR", "0.92", true)
	f.addCurrency(t, "GBP", "0.79", true)
	f.refs = []Reference{lineItemRef("L-1001", "P-1001", "A-1001", "EUR", "10.00", false), personRef("U-1001", "GBP")}

	_, lineItem := f.service.DeleteCurrency(t.Context(), alex, "EUR")
	_, person := f.service.DeleteCurrency(t.Context(), alex, "GBP")
	_, gone := f.service.DeleteCurrency(t.Context(), alex, "JPY")

	assertError(t, lineItem, errs.InUse("'EUR' is in use (used by 1 line item) — set it inactive instead."))
	assertError(t, person, errs.InUse("'GBP' is in use (display currency of 1 person) — set it inactive instead."))
	assertError(t, gone, errs.NotFound("This currency no longer exists — someone may have deleted it."))
}

// An unused currency moves to the recycle bin and Undo brings it back exactly; Set inactive and Reactivate
// are logged (FR-004).
func TestDeleteCurrency_Unused_Undo_FR004(t *testing.T) {
	f := newTestService(t)
	created, err := f.service.CreateCurrency(t.Context(), alex, jpy())
	if err != nil {
		t.Fatalf("CreateCurrency: %v", err)
	}
	inactive, err := f.service.SetCurrencyActive(t.Context(), alex, "JPY", StatusUpdate{Active: yes(false), Version: version(1)})
	if err != nil || inactive.Active || f.lastEntry(t).Verb != "deactivated" {
		t.Fatalf("set inactive = %+v, %v", inactive, err)
	}
	stored, _, _ := f.service.Currency(t.Context(), created.Code)

	entry, err := f.service.DeleteCurrency(t.Context(), alex, "JPY")
	if err != nil || entry.RecordType != "currency" || entry.RecordName != "JPY" {
		t.Fatalf("DeleteCurrency = %+v, %v", entry, err)
	}
	if _, found, _ := f.service.Currency(t.Context(), "JPY"); found {
		t.Errorf("JPY is still stored")
	}
	if _, err := f.history.Undo(t.Context(), alex, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if back, found, _ := f.service.Currency(t.Context(), "JPY"); !found || !reflect.DeepEqual(back, stored) {
		t.Errorf("after Undo = %+v", back)
	}
}

// A currency whose code another currency took meanwhile cannot come back: a code takes no suffix (FR-028,
// D-054).
func TestRestoreCurrency_CodeTaken_FR028(t *testing.T) {
	f := newTestService(t)
	if _, err := f.service.CreateCurrency(t.Context(), alex, jpy()); err != nil {
		t.Fatalf("CreateCurrency: %v", err)
	}
	entry, err := f.service.DeleteCurrency(t.Context(), alex, "JPY")
	if err != nil {
		t.Fatalf("DeleteCurrency: %v", err)
	}
	if _, err := f.service.CreateCurrency(t.Context(), alex, CurrencyInput{Code: "JPY", Name: "Yen", Rate: "151"}); err != nil {
		t.Fatalf("the new JPY: %v", err)
	}

	_, undo := f.history.Undo(t.Context(), alex, entry.ID)
	_, restore := f.history.Restore(t.Context(), alex, entry.ID)

	assertError(t, undo, errs.NameTaken("Can't undo — there is now another currency with the code 'JPY'. Restore it from the recycle bin instead."))
	assertError(t, restore, errs.CannotRestore("Can't restore — there is now another currency with the code 'JPY'."))
}
