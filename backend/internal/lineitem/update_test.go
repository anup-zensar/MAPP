package lineitem

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// formOf is the whole form of a stored line item, as the edit page sends it, with its version.
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
	if err := newFakeStore(f.db).Put(t.Context(), li); err != nil {
		t.Fatalf("Put: %v", err)
	}
}

// A missing version is reported with every other problem; an out-of-date one is refused naming who saved.
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

// A save that changes nothing creates no version and no log entry.
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

// One "changed" entry lists every changed field as people read it; each Shell attribute is a change of its own.
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

// An inactive currency, owner or finance budget item stays through any save; only a new choice must be active.
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

// A new planned cost or currency of a line item of a Submitted or Approved activity is saved without a
// confirmation and keeps the activity's approval (D-322); each save logs its change.
func TestUpdate_CostOrCurrencyKeepsStatus_034_FR001(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, input("Venue hire", "GBP", "15800"))
	printRun := f.create(t, priya, tradeAds, input("Print run", "USD", "2000"))

	more := formOf(venue)
	more.PlannedCost = text("17000")
	saved := f.update(t, priya, venue.ID, more)
	euros := formOf(saved)
	euros.Currency = "EUR"
	inEuros := f.update(t, priya, venue.ID, euros)
	dearer := formOf(printRun)
	dearer.PlannedCost = text("2500")
	submitted := f.update(t, priya, printRun.ID, dearer)

	if inEuros.Version != 3 || inEuros.PlannedCost.Format() != "17,000.00 EUR" || inEuros.SwbsStatus != SwbsReadyToSend ||
		submitted.SwbsStatus != SwbsWaitingForApproval {
		t.Errorf("venue v%d %s %s, print run %s; want v3 17,000.00 EUR still ready to send, print run still waiting",
			inEuros.Version, inEuros.PlannedCost.Format(), inEuros.SwbsStatus, submitted.SwbsStatus)
	}
	changed := 0
	for _, e := range f.log(t) {
		if e.Action == history.ActionChanged {
			changed++
		}
	}
	if changed != 3 {
		t.Errorf("%d changed entries, want 3 (one per save)", changed)
	}
}

// Every other change saves without a confirmation and never changes the activity's status.
func TestUpdate_OtherEditsKeepStatus_FR032_SC007(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, input("Venue hire", "GBP", "15800"))
	u := formOf(venue)
	u.Name, u.OwnerID, u.Description, u.ContractedAmount = "Venue", text(daniel.ID), "Hall B", text("15000")
	u.Spread = SpreadInput{Method: SpreadLastMonth, Months: []SpreadMonthInput{}}
	u.Attributes = settings.AttributeValues{glCode: json.RawMessage(`"SO-1202"`)}
	u.RequestSwbs, u.FinanceBudgetItemID = false, text(europeBrand)
	saved := f.update(t, priya, venue.ID, u)

	if saved.Version != 2 || f.activities.facts[socialOnAir].Status != "approved" || saved.SwbsStatus != SwbsNotRequested {
		t.Errorf("version %d, activity %s, SWBS %s; want 2, approved, notRequested",
			saved.Version, f.activities.facts[socialOnAir].Status, saved.SwbsStatus)
	}
}
