package lineitems

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// Only name, currency and planned cost are needed to save (FR-009, D-009); what is needed to submit stays
// empty and is listed: "3 fields needed" (US2 scenario 6).
func TestCreate_OnlyRequired_FR009(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	in := Input{Name: "Agency fee", Currency: "USD", PlannedCost: text("50000"), Spread: SpreadInput{Method: SpreadEvenly}}

	bare := f.create(t, priya, launchFilm, in)
	withOwner := input("Media", "USD", "1000.00")
	defaulted := f.create(t, priya, launchFilm, withOwner)

	if bare.Name != "Agency fee" || bare.PlannedCost != usd("50000.00") || bare.Owner != nil || bare.FinanceBudgetYear != nil {
		t.Errorf("created %+v", bare)
	}
	labels := func(v View) []string {
		var list []string
		for _, m := range v.Missing {
			list = append(list, m.Label)
		}
		return list
	}
	if want := []string{"Owner", "Spend type", "GL code", "Finance budget year", "Finance budget item"}; !slices.Equal(labels(bare), want) {
		t.Errorf("missing = %q, want %q", labels(bare), want)
	}
	if want := []string{"Spend type", "GL code", "Finance budget item"}; !slices.Equal(labels(defaulted), want) {
		t.Errorf("missing with the defaults = %q, want %q (3 fields needed)", labels(defaulted), want)
	}
	if got := f.create(t, priya, launchFilm, complete(input("Complete", "USD", "1.00"))).Missing; len(got) != 0 {
		t.Errorf("complete line item: missing %+v, want nothing (All filled in)", got)
	}
}

// Names are unique within the activity, ignoring letter case and surrounding spaces, and have at most 120
// characters; another activity may use the same name (FR-009).
func TestCreate_NameUnique_FR009(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.create(t, priya, launchFilm, input("Agency fee", "USD", "1.00"))

	saved := f.create(t, priya, launchFilm, input("  "+strings.Repeat("é", 120)+" ", "USD", "1.00"))
	f.create(t, priya, socialOnAir, func() Input { in := input("Agency fee", "USD", "1.00"); in.RestartApproval = true; return in }())
	_, taken := f.service.Create(t.Context(), priya, launchFilm, input(" AGENCY fee", "USD", "1.00"))
	_, long := f.service.Create(t.Context(), priya, launchFilm, input(strings.Repeat("a", 121), "USD", "1.00"))

	if saved.Name != strings.Repeat("é", 120) {
		t.Errorf("saved name = %q, want it without the surrounding spaces", saved.Name)
	}
	assertError(t, taken, errs.Validation(errs.FieldError{Field: "name", Message: "'Launch film' already has a line item called 'Agency fee'."}))
	assertError(t, long, errs.Validation(errs.FieldError{Field: "name", Message: "Use 120 characters or fewer."}))
}

// Typed amounts have at most two decimals: the form rounds and says so, the API refuses more (D-190, spec
// edge case "Rounding"); amounts stay below one trillion.
func TestCreate_AmountsAtMostTwoDecimals_D190(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	tests := []struct {
		cost, message string
	}{
		{"1234.565", "Enter an amount with at most two decimals."},
		{"1000000000000", "Enter an amount below one trillion."},
		{"9999999999999999", "Enter an amount below one trillion."},
		{"twelve", "Enter an amount, such as 12500.00."},
	}
	for _, tt := range tests {
		_, err := f.service.Create(t.Context(), priya, launchFilm, input("Media", "USD", tt.cost))

		assertError(t, err, errs.Validation(errs.FieldError{Field: "plannedCost", Message: tt.message}))
	}

	if v := f.create(t, priya, launchFilm, input("Biggest", "USD", "999999999999.99")); v.PlannedCost.Amount.String() != "999999999999.99" {
		t.Errorf("planned cost = %s", v.PlannedCost.Amount)
	}
	if v := f.create(t, priya, launchFilm, input("Round", "USD", " 1234.5 ")); v.PlannedCost.Amount.String() != "1234.50" {
		t.Errorf("planned cost = %s, want 1234.50", v.PlannedCost.Amount)
	}
}

// A new line item gets the next readable ID L-…, the person and time who created it, and one "created" log
// entry with its activity as its path (FR-010, D-019).
func TestCreate_IDAndLog_FR010(t *testing.T) { //nolint:gocyclo // 14: the IDs, the bookkeeping and the log entry of two creates, field by field
	t.Parallel()
	f := newTestService(t)

	first := f.create(t, priya, launchFilm, input("Agency fee", "USD", "50000.00"))
	second := f.create(t, daniel, launchFilm, input("Film production", "EUR", "46000.00"))

	if first.ID != "L-1001" || second.ID != "L-1002" || first.ActivityID != launchFilm {
		t.Errorf("IDs = %s, %s; want L-1001, L-1002", first.ID, second.ID)
	}
	if first.Meta != record.NewMeta(priya, start) || second.Meta != record.NewMeta(daniel, start) {
		t.Errorf("bookkeeping = %+v / %+v", first.Meta, second.Meta)
	}
	entries := f.log(t)
	if len(entries) != 2 {
		t.Fatalf("%d log entries, want 2", len(entries))
	}
	e := entries[1]
	want := []history.RecordRef{{Type: "activity", ID: launchFilm, Name: "Launch film"}}
	if e.Action != history.ActionCreated || e.RecordType != "lineItem" || e.RecordTypeLabel != "line item" ||
		e.RecordID != "L-1001" || e.RecordName != "Agency fee" || e.Actor != priya || !reflect.DeepEqual(e.Path, want) {
		t.Errorf("log entry = %+v", e)
	}
}

// A new currency choice must be an active currency of Settings (FR-011); the USD amount is calculated at
// its current rate: 10,000.00 EUR at 0.92 → 10,869.57 USD (US2 scenario 2).
func TestCreate_CurrencyActive_FR011(t *testing.T) {
	t.Parallel()
	f := newTestService(t)

	_, inactive := f.service.Create(t.Context(), priya, launchFilm, input("Fees", "INR", "1.00"))
	_, unknown := f.service.Create(t.Context(), priya, launchFilm, input("Fees", "XYZ", "1.00"))
	_, empty := f.service.Create(t.Context(), priya, launchFilm, input("Fees", " ", "1.00"))
	euros := f.create(t, priya, launchFilm, input("Fees", "EUR", "10000"))

	assertError(t, inactive, errs.Validation(errs.FieldError{Field: "currency", Message: "Choose an active currency — INR is no longer active."}))
	assertError(t, unknown, errs.Validation(errs.FieldError{Field: "currency", Message: "Choose a currency."}))
	assertError(t, empty, errs.Validation(errs.FieldError{Field: "currency", Message: "Choose a currency."}))
	if euros.PlannedCost != (money.Money{Amount: amount("10000.00"), Currency: "EUR"}) || euros.PlannedCostUSD != usd("10869.57") ||
		euros.Currency.Code != "EUR" || euros.Currency.RatePerUSD.String() != "0.92" || !euros.Currency.Active {
		t.Errorf("EUR line item: %+v / %+v / %+v", euros.PlannedCost, euros.PlannedCostUSD, euros.Currency)
	}
}

// Shell attribute problems of feature 002 (R11: a GL code that doesn't fit the spend type) are reported with
// the others; the values 002 returns are stored (FR-014).
func TestCreate_AttributeProblems_FR014(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	f.attributes.problems = []errs.FieldError{{Field: "attributes.SA-1033", Message: "Choose one of the options of spend category."}}
	in := input("", "USD", "1.00")
	in.Attributes = settings.AttributeValues{glCode: json.RawMessage(`"SO-1299"`)}

	_, err := f.service.Create(t.Context(), priya, launchFilm, in)

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "name", Message: "Enter a name for the line item."},
		errs.FieldError{Field: "attributes.SA-1033", Message: "Choose one of the options of spend category."},
		errs.FieldError{Field: "attributes." + glCode, Message: glCodeMismatch},
	))

	f.attributes.problems = nil
	in = complete(input("Coded", "USD", "1.00"))
	in.Attributes["SA-1033"] = json.RawMessage(`""`) // an empty value is not stored
	v := f.create(t, priya, launchFilm, in)
	if want := (settings.AttributeValues{spendType: json.RawMessage(`"SO-1101"`), glCode: json.RawMessage(`"SO-1201"`)}); !reflect.DeepEqual(v.Attributes, want) {
		t.Errorf("attributes = %s, want %s", v.Attributes, want)
	}
}

// The contracted amount is optional, 0.00 or more, in the line item's currency (FR-014).
func TestCreate_ContractedAmount_FR014(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	for _, tt := range []struct{ amount, message string }{
		{"-1", "Contracted amount can't be negative."},
		{"12.345", "Enter an amount with at most two decimals."},
		{"lots", "Enter an amount, such as 12500.00."},
	} {
		in := input("Fees", "EUR", "100.00")
		in.ContractedAmount = &tt.amount

		_, err := f.service.Create(t.Context(), priya, launchFilm, in)

		assertError(t, err, errs.Validation(errs.FieldError{Field: "contractedAmount", Message: tt.message}))
	}

	in := input("Fees", "EUR", "100.00")
	in.ContractedAmount = text("80")
	if v := f.create(t, priya, launchFilm, in); v.ContractedAmount == nil || *v.ContractedAmount != (money.Money{Amount: amount("80.00"), Currency: "EUR"}) {
		t.Errorf("contracted amount = %+v, want 80.00 EUR", v.ContractedAmount)
	}
	in = input("Empty", "EUR", "100.00")
	in.ContractedAmount = text("")
	if v := f.create(t, priya, launchFilm, in); v.ContractedAmount != nil {
		t.Errorf("contracted amount = %+v, want none", v.ContractedAmount)
	}
}

// Adding a line item to a Submitted or Approved activity needs the person's confirmation; without it nothing
// is saved. With it, the activity goes back to Created with the reason (FR-031, D-146). A Created activity
// needs no confirmation and keeps its status.
func TestCreate_RestartNeedsConfirmation_FR031(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	for _, activityID := range []string{socialOnAir, tradeAds} {
		_, err := f.service.Create(t.Context(), priya, activityID, input("Venue hire", "GBP", "15800"))

		assertError(t, err, errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage}))
	}
	if len(f.stored(t)) != 0 || len(f.activities.restarts) != 0 {
		t.Fatalf("saved %d line items and restarted %d times, want nothing", len(f.stored(t)), len(f.activities.restarts))
	}

	confirmed := input("Venue hire", "GBP", "15800")
	confirmed.RestartApproval = true
	v := f.create(t, priya, socialOnAir, confirmed)
	f.create(t, priya, launchFilm, input("Agency fee", "USD", "50000.00"))

	want := []restartCall{{activityID: socialOnAir, reasons: []string{"Line item 'Venue hire' (L-1001) added: 15,800.00 GBP"}}}
	if !reflect.DeepEqual(f.activities.restarts, want) {
		t.Errorf("restarts = %+v, want %+v", f.activities.restarts, want)
	}
	if v.SwbsStatus != SwbsWaitingForApproval {
		t.Errorf("SWBS status = %s, want waitingForApproval: the activity is Created again", v.SwbsStatus)
	}
}
