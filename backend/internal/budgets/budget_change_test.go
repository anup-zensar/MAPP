package budgets

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// A new global budget gets the first ID, version 1 and a "created" entry; its page has no allocation yet,
// the year lock, and every action for the global planner (FR-029, FR-037).
func TestCreate_GlobalBudget_FR029(t *testing.T) { //nolint:gocyclo // 11: one create, then its fields, figures, locks, actions and log entry
	t.Parallel()
	f := newFixture(t)

	view, err := f.service.Create(t.Context(), carlos, budgetInput())

	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got := fmt.Sprintf("%s %s %d %s %s %q v%d by %s", view.ID, view.Name, view.Year, view.Amount.Format(), view.Owner.Name,
		view.Description, view.Version, view.CreatedBy.Name)
	want := `GB-1001 FY2026 Global marketing budget 2026 2,500,000.00 USD Carlos Mendes "Shared out to market plans by Global Marketing Ops." v1 by Carlos Mendes`
	if got != want {
		t.Errorf("created = %s\nwant      %s", got, want)
	}
	if figures := figuresText(view.Figures); figures != "2,500,000.00 USD allocated 0.00 USD unallocated 2,500,000.00 USD planned 0.00 USD" {
		t.Errorf("figures = %s", figures)
	}
	if !slices.Equal(view.Locks, []settings.Lock{{Field: "year", Reason: "The year can't change once the global budget is saved."}}) {
		t.Errorf("locks = %+v, want the year lock only", view.Locks)
	}
	if view.Access != (BudgetAccess{Edit: available, Delete: available, Allocate: available}) {
		t.Errorf("access = %+v, want everything available", view.Access)
	}
	entries := f.entries(t, view.ID)
	if len(entries) != 1 || entries[0].Action != history.ActionCreated || entries[0].Actor != carlos ||
		entries[0].RecordName != "FY2026 Global marketing budget" || entries[0].RecordTypeLabel != "global budget" {
		t.Errorf("log = %+v, want one created entry by Carlos", entries)
	}
}

// Every problem of the form is reported at once and nothing is saved (FR-029, messages 24, 26 and 27).
func TestCreate_EveryProblemAtOnce_FR029(t *testing.T) { //nolint:tparallel // the check after the subtests needs them done
	t.Parallel()
	f := newFixture(t)
	f.create(t, budgetInput())
	tooLong := strings.Repeat("x", 121)
	cases := []struct {
		name   string
		change func(in *GlobalBudgetInput)
		fields []errs.FieldError
	}{
		{"everything wrong", func(in *GlobalBudgetInput) {
			in.Name, in.Year, in.Amount, in.OwnerID, in.Description = " ", ptr(2032), nil, "", strings.Repeat("d", 2001)
		}, []errs.FieldError{
			{Field: "name", Message: "Enter a name for the global budget."},
			{Field: "year", Message: "Choose a year from 2025 to 2031."},
			{Field: "amount", Message: "Enter an amount of 0 or more."},
			{Field: "ownerId", Message: "Choose a person who can manage global budgets for all markets."},
			{Field: "description", Message: "Use 2,000 characters or fewer."},
		}},
		{"name taken ignoring case, negative amount, owner who cannot manage all markets", func(in *GlobalBudgetInput) {
			in.Name, in.Amount, in.OwnerID = "fy2026 GLOBAL marketing budget", &money.Input{Amount: "-1.00", Currency: "USD"}, priya.ID
		}, []errs.FieldError{
			{Field: "name", Message: "A global budget called 'FY2026 Global marketing budget' already exists."},
			{Field: "amount", Message: "Enter an amount of 0 or more."},
			{Field: "ownerId", Message: "Choose a person who can manage global budgets for all markets."},
		}},
		{"name too long, three decimals, unknown owner", func(in *GlobalBudgetInput) {
			in.Name, in.Amount, in.OwnerID = tooLong, &money.Input{Amount: "1.005", Currency: "USD"}, "U-9999"
		}, []errs.FieldError{
			{Field: "name", Message: "Use 120 characters or fewer."},
			{Field: "amount", Message: "Use at most 2 decimals."},
			{Field: "ownerId", Message: "Choose a person who can manage global budgets for all markets."},
		}},
		{"amount in another currency, inactive owner", func(in *GlobalBudgetInput) {
			in.Name, in.Amount, in.OwnerID = "Other", &money.Input{Amount: "1.00", Currency: "EUR"}, "U-1006"
		}, []errs.FieldError{
			{Field: "amount", Message: "Enter the amount in USD."},
			{Field: "ownerId", Message: "Choose a person who can manage global budgets for all markets."},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := budgetInput()
			c.change(&in)

			_, err := f.service.Create(t.Context(), carlos, in)

			assertError(t, err, errs.Validation(c.fields...))
		})
	}
	if _, found := f.stored(t, "GB-1002"); found {
		t.Errorf("a refused form saved GB-1002")
	}
}

// Creating, changing and deleting a global budget needs "Create global budgets and allocate them to plans"
// for All markets (FR-029): a local planner is refused with the access rules' message and nothing changes.
func TestCreateUpdateDelete_NeedManageForAllMarkets_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	b := f.create(t, budgetInput())
	f.as(priyasAccess())

	_, errCreate := f.service.Create(t.Context(), priya, budgetInput())
	_, errUpdate := f.service.Update(t.Context(), priya, b.ID, GlobalBudgetUpdate{Name: ptr("Mine"), Version: ptr(1)})
	_, errDelete := f.service.Delete(t.Context(), priya, b.ID)

	for _, err := range []error{errCreate, errUpdate, errDelete} {
		assertError(t, err, refusal(refusedAll))
	}
	if stored, found := f.stored(t, b.ID); !found || stored.Name != b.Name || stored.Version != 1 {
		t.Errorf("stored = %+v, %v; want it unchanged", stored, found)
	}
}

// Saving changed fields gives a new version and one "changed" entry with every field in words; the year
// cannot be sent; an out-of-date version is refused naming who changed it; a missing version is a field
// problem; a save that changes nothing changes nothing (FR-029, FR-037, data model §3).
func TestUpdate_ChangedFieldsAndVersion_FR029_FR037(t *testing.T) { //nolint:gocyclo // 11: one save, a stale, a missing and an unchanged one, then the log
	t.Parallel()
	f := newFixture(t)
	b := f.create(t, budgetInput())
	f.clock.Advance(time.Hour)

	saved, err := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{
		Name: ptr(" FY2026 Global budget "), Amount: &money.Input{Amount: "3000000.00", Currency: "USD"}, Version: ptr(1),
	})
	_, stale := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{Description: ptr("Old"), Version: ptr(1)})
	_, missing := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{Description: ptr("New")})
	same, errSame := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{Name: ptr("FY2026 Global budget"), Version: ptr(2)})

	if err != nil || saved.Name != "FY2026 Global budget" || saved.Amount.Format() != "3,000,000.00 USD" || saved.Version != 2 ||
		!saved.UpdatedAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	assertError(t, stale, errs.VersionConflict("global budget", "Carlos Mendes", start.Add(time.Hour)))
	assertError(t, missing, errs.Validation(errs.FieldError{Field: "version", Message: "Reload the page and try again."}))
	if errSame != nil || same.Version != 2 {
		t.Errorf("unchanged save = version %d, %v; want version 2", same.Version, errSame)
	}
	entries := f.entries(t, b.ID)
	if len(entries) != 2 || entries[0].Action != history.ActionChanged {
		t.Fatalf("log = %+v, want created and one changed entry", entries)
	}
	if got := changesText(entries[0].Changes); got != "Amount: 2,500,000.00 USD → 3,000,000.00 USD; Name: FY2026 Global marketing budget → FY2026 Global budget" {
		t.Errorf("changes = %s", got)
	}
}

// A new owner must manage global budgets for all markets (message 27) and is logged by name; an owner the
// global budget already has stays when other fields change, even after losing that role (FR-029, FR-037).
func TestUpdate_Owner_FR029_FR037(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	b := f.create(t, budgetInput())

	_, refused := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{OwnerID: ptr(priya.ID), Version: ptr(1)})
	handed, err := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{OwnerID: ptr(alex.ID), Version: ptr(1)})
	f.access.managers = map[string]bool{} // Alex loses the role
	kept, errKept := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{OwnerID: ptr(alex.ID), Description: ptr("New"), Version: ptr(2)})

	assertError(t, refused, errs.Validation(errs.FieldError{Field: "ownerId", Message: "Choose a person who can manage global budgets for all markets."}))
	if err != nil || handed.Owner.Name != "Alex Morgan" {
		t.Fatalf("handed = %+v, %v", handed.Owner, err)
	}
	if errKept != nil || kept.Description != "New" || kept.Owner.Name != "Alex Morgan" {
		t.Errorf("kept = %+v, %v", kept, errKept)
	}
	entries := f.entries(t, b.ID)
	if got := changesText(entries[1].Changes); got != "Owner: Carlos Mendes → Alex Morgan" {
		t.Errorf("owner change = %s", got)
	}
}

// A global budget with allocations cannot be deleted (409 in_use, message 30, one plan and several plans);
// one without goes to the recycle bin and comes back with Undo (FR-035).
func TestDelete_InUseOrToTheBin_FR035(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	demo := f.demo(t)
	single := f.create(t, GlobalBudgetInput{Name: "Single", Year: ptr(2026), Amount: &money.Input{Amount: "0.00", Currency: "USD"}, OwnerID: carlos.ID})
	f.allocate(t, single.ID, planBrazil, "10.00")
	empty := f.create(t, GlobalBudgetInput{Name: "Empty", Year: ptr(2027), Amount: &money.Input{Amount: "0.00", Currency: "USD"}, OwnerID: carlos.ID})

	_, errMany := f.service.Delete(t.Context(), carlos, demo.ID)
	_, errOne := f.service.Delete(t.Context(), carlos, single.ID)
	entry, err := f.service.Delete(t.Context(), carlos, empty.ID)
	_, errGone := f.service.Delete(t.Context(), carlos, empty.ID)

	assertError(t, errMany, errs.InUse("'FY2026 Global marketing budget' has allocations to 2 plans. Remove them first."))
	assertError(t, errOne, errs.InUse("'Single' has an allocation to 1 plan. Remove it first."))
	assertError(t, errGone, errs.NotFound("This global budget no longer exists — someone may have deleted it."))
	if err != nil || entry.RecordType != kindBudget || entry.RecordName != "Empty" {
		t.Fatalf("delete = %+v, %v", entry, err)
	}
	if _, err := f.history.Undo(t.Context(), carlos, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if stored, found := f.stored(t, empty.ID); !found || stored.Name != "Empty" {
		t.Errorf("after Undo = %+v, %v; want it back", stored, found)
	}
}

// A global budget restored from the bin after another took its name is renamed; its Undo is refused
// (001's name rule, FR-035).
func TestRestore_NameRule_FR035(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	b := f.create(t, budgetInput())
	entry, err := f.service.Delete(t.Context(), carlos, b.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	f.create(t, budgetInput())

	_, errUndo := f.history.Undo(t.Context(), carlos, entry.ID)
	result, err := f.history.Restore(t.Context(), carlos, entry.ID)

	assertError(t, errUndo, errs.NameTaken("Can't undo — there is now another global budget called 'FY2026 Global marketing budget'. Restore it from the recycle bin instead."))
	if err != nil || result.RecordName != "FY2026 Global marketing budget (restored)" ||
		result.Details != "Renamed to 'FY2026 Global marketing budget (restored)' because the name was taken." {
		t.Errorf("restore = %+v, %v", result, err)
	}
}
