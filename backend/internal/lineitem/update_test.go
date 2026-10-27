package lineitem

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// formOf returns the whole form of a stored line item, as the edit page sends it, with its version.
func formOf(v View) Update {
	months := []SpreadMonthInput{}
	if v.Spread.Method == SpreadCustom {
		for _, m := range v.Spread.Months {
			months = append(months, SpreadMonthInput{Month: m.Month, Amount: m.Amount.Amount.String()})
		}
	}
	u := Update{
		Name: v.Name, Description: v.Description, Currency: v.Currency.Code, PlannedCost: text(v.PlannedCost.Amount.String()),
		Spread: SpreadInput{Method: v.Spread.Method, Months: months}, Attributes: v.Attributes, RequestSwbs: v.RequestSwbs,
		FinanceBudgetYear: v.FinanceBudgetYear, Version: &v.Version,
	}
	if v.Owner != nil {
		u.OwnerID = text(v.Owner.ID)
	}
	if v.ContractedAmount != nil {
		u.ContractedAmount = text(v.ContractedAmount.Amount.String())
	}
	if v.FinanceBudgetItem != nil {
		u.FinanceBudgetItemID = text(v.FinanceBudgetItem.ID)
	}
	return u
}

// update saves u on the line item id and stops the test when that fails.
func (f *fixture) update(t *testing.T, a actor.Actor, id string, u Update) View {
	t.Helper()
	view, err := f.service.Update(t.Context(), a, id, u)
	if err != nil {
		t.Fatalf("Update(%s): %v", id, err)
	}
	return view
}

// put stores li as it is, for situations a form cannot create (values that became inactive since).
func (f *fixture) put(t *testing.T, li LineItem) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error { return NewJSONStore().Put(ctx, li) })
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
}

// A save needs the version it started from: a missing one is reported with every other problem, an
// out-of-date one is refused naming who changed the line item (FR-027, D-030).
func TestUpdate_Version_FR027(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, input("Agency fee", "USD", "50000.00"))
	zero := 0
	reload := errs.FieldError{Field: "version", Message: reloadMessage}

	noVersion := formOf(fee)
	noVersion.Version, noVersion.Name = nil, ""
	_, missing := f.service.Update(t.Context(), priya, fee.ID, noVersion)
	zeroVersion := formOf(fee)
	zeroVersion.Version = &zero
	_, zeroed := f.service.Update(t.Context(), priya, fee.ID, zeroVersion)
	newer := formOf(fee)
	newer.Description = "Daniel's change"
	f.update(t, daniel, fee.ID, newer)
	stale := formOf(fee)
	stale.Description = "Priya's change"
	_, conflict := f.service.Update(t.Context(), priya, fee.ID, stale)

	assertError(t, missing, errs.Validation(errs.FieldError{Field: "name", Message: enterNameMessage}, reload))
	assertError(t, zeroed, errs.Validation(reload))
	assertError(t, conflict, errs.VersionConflict("line item", "Daniel Okoro", start))
}

// A save that changes nothing creates no version and no log entry (FR-027).
func TestUpdate_NoChangeNoLog_FR027(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, complete(input("Agency fee", "USD", "50000.00")))
	logged := len(f.log(t))
	same := formOf(fee)
	same.Name, same.PlannedCost = " Agency fee ", text("50000")

	saved := f.update(t, daniel, fee.ID, same)

	if saved.Version != 1 || saved.UpdatedBy != priya || len(f.log(t)) != logged {
		t.Errorf("version %d by %s, %d log entries; want version 1 by Priya and no new entry", saved.Version, saved.UpdatedBy.Name, len(f.log(t))-logged)
	}
}

// A save with changes gets a new version and one "changed" entry listing every changed field with its old
// and new value as people read them; each Shell attribute is a change of its own (FR-027, data model §7.3).
func TestUpdate_LogsOldAndNew_FR027(t *testing.T) { //nolint:gocyclo // 10: one save, then its bookkeeping and every change of its entry
	t.Parallel()
	f := newTestService(t)
	fee := f.create(t, priya, launchFilm, complete(input("Agency fee", "USD", "50000.00")))
	u := formOf(fee)
	u.Name, u.OwnerID, u.PlannedCost = "Agency fees", text(daniel.ID), text("55000")
	u.Spread = SpreadInput{Method: SpreadCustom, Months: []SpreadMonthInput{{Month: "2026-02", Amount: "2000"}, {Month: "2026-03", Amount: "53000"}}}
	u.Attributes = settings.AttributeValues{spendType: json.RawMessage(`"SO-1101"`), glCode: json.RawMessage(`"SO-1202"`)}
	u.FinanceBudgetItemID = text(europeEvents)

	saved := f.update(t, daniel, fee.ID, u)

	if saved.Version != 2 || saved.UpdatedBy != daniel || saved.UpdatedAt != start || saved.CreatedBy != priya {
		t.Errorf("bookkeeping = %+v", saved.Meta)
	}
	entry := f.log(t)[0]
	var got []string
	for _, c := range entry.Changes {
		got = append(got, c.Label+": "+c.OldText+" → "+c.NewText)
	}
	want := []string{
		"Finance budget item: 2026 Lubricants Budget › Europe – Digital → 2026 Lubricants Budget › Europe – Events",
		"Name: Agency fee → Agency fees",
		"Owner: Priya Raman → Daniel Okoro",
		"Planned cost: 50,000.00 → 55,000.00",
		"Spread over time: Evenly → Custom: Feb 2026 2,000.00 · Mar 2026 53,000.00",
		"GL code: 7250080 – Advertising digital → 7250040 – Events and sponsorship",
	}
	if entry.Action != history.ActionChanged || entry.RecordName != "Agency fees" || entry.Actor != daniel || !slices.Equal(got, want) {
		t.Errorf("entry %s %q by %s:\n%s\nwant\n%s", entry.Action, entry.RecordName, entry.Actor.Name, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// A currency, owner or finance budget item that became inactive stays on the line item through any save and
// still converts at its last rate; only a new choice must be active (FR-011, FR-016).
func TestUpdate_KeepsInactiveValues_FR011_FR016(t *testing.T) { //nolint:gocyclo // 10: the kept values read, saved again, then refused as a new choice
	t.Parallel()
	f := newTestService(t)
	year := 2026
	f.put(t, LineItem{
		ID: "L-1001", ActivityID: launchFilm, Name: "Delhi fees", OwnerID: "U-1009", Currency: "INR",
		PlannedCost: amount("835.00"), Spread: Spread{Method: SpreadEvenly, Months: []MonthAmount{}},
		Attributes: settings.AttributeValues{}, FinanceBudgetYear: &year, FinanceBudgetItemID: europeLegacy,
		Meta: record.NewMeta(priya, start),
	})
	details, err := f.service.Get(t.Context(), priya, "L-1001")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	kept := details.LineItem
	if kept.Currency.Active || kept.PlannedCostUSD.Format() != "10.00 USD" || kept.Owner == nil || kept.Owner.Active ||
		kept.FinanceBudgetItem == nil || kept.FinanceBudgetItem.Active {
		t.Errorf("kept values = %+v / %s / %+v / %+v", kept.Currency, kept.PlannedCostUSD.Format(), kept.Owner, kept.FinanceBudgetItem)
	}

	u := formOf(kept)
	u.Description = "Still in rupees"
	if saved := f.update(t, priya, "L-1001", u); saved.Version != 2 || saved.Currency.Code != "INR" {
		t.Errorf("saved = %+v, want version 2 in INR", saved)
	}

	other := f.create(t, priya, launchFilm, input("Agency fee", "EUR", "10.00"))
	switched := formOf(other)
	switched.Currency, switched.OwnerID, switched.FinanceBudgetItemID = "INR", text("U-1009"), text(europeLegacy)
	_, err = f.service.Update(t.Context(), priya, other.ID, switched)
	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "ownerId", Message: "Choose an active person — Sam Former is inactive."},
		errs.FieldError{Field: "currency", Message: "Choose an active currency — INR is no longer active."},
		errs.FieldError{Field: "financeBudgetItemId", Message: "Choose an active finance budget item — 'Europe – Legacy' is no longer active."},
	))
}

// Changing the planned cost or the currency of a line item of a Submitted or Approved activity needs the
// person's confirmation and sends the activity back to Created with the change (FR-031, US4 scenario 3).
func TestUpdate_CostOrCurrencyRestart_FR031(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, confirmed(input("Venue hire", "GBP", "15800")))
	approve := func() {
		facts := f.activities.facts[socialOnAir]
		facts.Status = "approved"
		f.activities.facts[socialOnAir] = facts
	}
	approve()
	f.activities.restarts = nil

	more := formOf(venue)
	more.PlannedCost = text("17000")
	_, err := f.service.Update(t.Context(), priya, venue.ID, more)
	assertError(t, err, errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage}))
	if stored := f.stored(t)[0]; stored.Version != 1 || len(f.activities.restarts) != 0 {
		t.Fatalf("unconfirmed save stored version %d and restarted %d times", stored.Version, len(f.activities.restarts))
	}

	more.RestartApproval = true
	saved := f.update(t, priya, venue.ID, more)
	approve()
	euros := formOf(saved)
	euros.Currency, euros.RestartApproval = "EUR", true
	f.update(t, priya, venue.ID, euros)

	want := []restartCall{
		{activityID: socialOnAir, reasons: []string{"Planned cost of line item 'Venue hire' (L-1001): 15,800.00 GBP → 17,000.00 GBP"}},
		{activityID: socialOnAir, reasons: []string{"Planned cost of line item 'Venue hire' (L-1001): 17,000.00 GBP → 17,000.00 EUR"}},
	}
	if !reflect.DeepEqual(f.activities.restarts, want) {
		t.Errorf("restarts = %+v\nwant %+v", f.activities.restarts, want)
	}
}

// Every other change — name, owner, description, spread, Shell attributes, contracted amount, SWBS request,
// finance budget link — saves without a confirmation and never changes the activity's status (FR-032,
// SC-007, US4 scenario 2).
func TestUpdate_OtherEditsKeepStatus_FR032_SC007(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, confirmed(input("Venue hire", "GBP", "15800")))
	facts := f.activities.facts[socialOnAir]
	facts.Status = "approved"
	f.activities.facts[socialOnAir] = facts
	f.activities.restarts = nil

	u := formOf(venue)
	u.Name, u.OwnerID, u.Description, u.ContractedAmount = "Venue", text(daniel.ID), "Hall B", text("15000")
	u.Spread = SpreadInput{Method: SpreadLastMonth, Months: []SpreadMonthInput{}}
	u.Attributes = settings.AttributeValues{glCode: json.RawMessage(`"SO-1202"`)}
	u.RequestSwbs, u.FinanceBudgetItemID = false, text(europeBrand)
	saved := f.update(t, priya, venue.ID, u)

	if saved.Version != 2 || len(f.activities.restarts) != 0 || f.activities.facts[socialOnAir].Status != "approved" ||
		saved.SwbsStatus != SwbsNotRequested {
		t.Errorf("version %d, %d restarts, activity %s, SWBS %s; want 2, none, approved, notRequested",
			saved.Version, len(f.activities.restarts), f.activities.facts[socialOnAir].Status, saved.SwbsStatus)
	}
}
