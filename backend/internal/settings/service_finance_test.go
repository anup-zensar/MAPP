package settings

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/record"
)

// yearLock is the lock every finance budget item carries (data model §8).
var yearLock = []Lock{{Field: "year", Reason: "The year can't change once the item is saved — create a new item for another year."}}

// itemInput is a new finance budget item of the budget year and budget name.
func itemInput(year int, budget, name, code string) ItemInput {
	return ItemInput{Year: &year, BudgetName: budget, Name: name, Code: code}
}

// createItem creates a finance budget item as Alex and stops the test when that fails.
func (f *fixture) createItem(t *testing.T, input ItemInput) FinanceBudgetItemView {
	t.Helper()
	view, err := f.service.CreateFinanceBudgetItem(t.Context(), alex, input)
	if err != nil {
		t.Fatalf("CreateFinanceBudgetItem(%q): %v", input.Name, err)
	}
	return view
}

// unused is the usage of a record nothing uses.
var unused = Usage{Summary: "Not used yet", Parts: []UsagePart{}, Records: []history.RecordRef{}}

// A new item with a budget name not used yet that year creates the budget: one "created" entry for the
// budget, then one for the item, which is listed in its budget (US5-3, FR-014).
func TestCreateItem_CreatesBudget_FR014(t *testing.T) {
	f := newTestService(t)
	f.clock.now = start.Add(time.Hour)

	created, err := f.service.CreateFinanceBudgetItem(t.Context(), daniel, itemInput(2026, " 2026 Lubricants Budget ", " Europe – Retail ", " LUB-EU-RET "))
	if err != nil {
		t.Fatalf("CreateFinanceBudgetItem: %v", err)
	}
	second := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", ""))

	want := FinanceBudgetItemView{
		ID: "FBI-1001", BudgetID: "FB-1001", BudgetName: "2026 Lubricants Budget", Year: 2026, Name: "Europe – Retail",
		Code: "LUB-EU-RET", Active: true, Usage: unused, Locks: yearLock, Meta: record.NewMeta(daniel, f.clock.now),
	}
	if !reflect.DeepEqual(created, want) {
		t.Errorf("created = %+v\nwant      %+v", created, want)
	}
	if second.BudgetID != "FB-1001" || second.Code != "" {
		t.Errorf("second item = %+v, want it in the same budget without a code", second)
	}
	page, err := f.service.ListFinanceBudgets(t.Context(), alex, nil)
	if err != nil || page.Total != 1 || len(page.Items[0].Items) != 2 || page.Items[0].Items[1].Name != "Europe – Retail" ||
		page.Items[0].ActiveItemCount != 2 {
		t.Errorf("ListFinanceBudgets = %+v, %v", page, err)
	}
	entries := f.logEntries(t)
	budget, item := entries[0], entries[1]
	if len(entries) != 3 || budget.Action != history.ActionCreated || budget.RecordType != "financeBudget" || budget.RecordName != "2026 Lubricants Budget" ||
		budget.Actor != daniel {
		t.Errorf("entries = %+v", entries)
	}
	wantPath := []history.RecordRef{{Type: "financeBudget", ID: "FB-1001", Name: "2026 Lubricants Budget"}}
	if item.Action != history.ActionCreated || item.RecordType != "financeBudgetItem" || item.RecordName != "Europe – Retail" ||
		!reflect.DeepEqual(item.Path, wantPath) {
		t.Errorf("item entry = %+v", item)
	}
}

// Every problem of an item is shown at once: the year (current − 1 to + 5), the budget name, the item name
// (unique in its budget) and the code (unique in the year, ignoring case) (FR-014, SC-003).
func TestCreateItem_Validation_FR014(t *testing.T) {
	f := newTestService(t)
	digital := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", "LUB-EU-DIG"))
	entries := len(f.logEntries(t))

	_, first := f.service.CreateFinanceBudgetItem(t.Context(), alex, ItemInput{BudgetName: " ", Code: strings.Repeat("c", 41)})
	_, second := f.service.CreateFinanceBudgetItem(t.Context(), alex, itemInput(2024, strings.Repeat("b", 121), strings.Repeat("n", 121), ""))
	_, third := f.service.CreateFinanceBudgetItem(t.Context(), alex, itemInput(2026, "2026 lubricants budget", " europe – DIGITAL ", "lub-eu-dig"))
	_, otherBudget := f.service.CreateFinanceBudgetItem(t.Context(), alex, itemInput(2026, "2026 Mobility Budget", "China", "LUB-EU-DIG"))
	_, tooLate := f.service.CreateFinanceBudgetItem(t.Context(), alex, itemInput(2032, "2032 Budget", "China", ""))

	assertError(t, first, problems("year", "Choose a year from 2025 to 2031.", "budgetName", "Enter a budget name.",
		"name", "Enter a name.", "code", "Use 40 characters or fewer."))
	assertError(t, second, problems("year", "Choose a year from 2025 to 2031.", "budgetName", "Use 120 characters or fewer.",
		"name", "Use 120 characters or fewer."))
	assertError(t, third, problems("name", "The budget '2026 Lubricants Budget' already has an item called 'Europe – Digital'.",
		"code", "The code 'lub-eu-dig' is already used in 2026 by 'Europe – Digital'."))
	assertError(t, otherBudget, problems("code", "The code 'LUB-EU-DIG' is already used in 2026 by 'Europe – Digital'."))
	assertError(t, tooLate, problems("year", "Choose a year from 2025 to 2031."))
	if len(f.logEntries(t)) != entries {
		t.Errorf("refused creates were logged")
	}

	nextYear, err := f.service.CreateFinanceBudgetItem(t.Context(), alex, itemInput(2031, "2026 Lubricants Budget", "Europe – Digital", "LUB-EU-DIG"))
	if err != nil || nextYear.BudgetID == digital.BudgetID || nextYear.Year != 2031 {
		t.Errorf("the same names and code in another year = %+v, %v", nextYear, err)
	}
}

// An item joins the budget of its year whose name matches ignoring letter case and spaces; the same budget
// name in another year is another budget (FR-014, data model §7).
func TestCreateItem_JoinsBudgetIgnoringCase_FR014(t *testing.T) {
	f := newTestService(t)
	first := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", ""))

	joined := f.createItem(t, itemInput(2026, "  2026 LUBRICANTS budget ", "Europe – Events", ""))
	otherYear := f.createItem(t, itemInput(2027, "2026 Lubricants Budget", "Europe – Digital", ""))

	if joined.BudgetID != first.BudgetID || joined.BudgetName != "2026 Lubricants Budget" {
		t.Errorf("joined = %+v, want the budget of %s", joined, first.BudgetID)
	}
	if otherYear.BudgetID == first.BudgetID || otherYear.Year != 2027 {
		t.Errorf("other year = %+v, want a budget of its own", otherYear)
	}
	created := 0
	for _, e := range f.logEntries(t) {
		if e.RecordType == "financeBudget" && e.Action == history.ActionCreated {
			created++
		}
	}
	if created != 2 {
		t.Errorf("%d budgets created, want 2", created)
	}
}

// The year of a saved item is locked; a new budget name moves the item to that budget of the same year,
// found ignoring case or created, and the change is logged with the budget names. The item's name must be
// free in the budget it moves to (FR-014, US5-4, FR-007).
func TestUpdateItem_YearLocked_MovesBudget_FR014(t *testing.T) {
	f := newTestService(t)
	item := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", "LUB-EU-DIG"))
	mobility := f.createItem(t, itemInput(2026, "2026 Mobility Budget", "China – Retail", "MOB-CN-RET"))
	f.clock.now = start.Add(time.Hour)

	moved, err := f.service.UpdateFinanceBudgetItem(t.Context(), daniel, item.ID, ItemUpdate{
		BudgetName: text(" 2026 mobility BUDGET "), Name: text("Europe – Digital media"), Code: text(""), Version: version(1),
	})
	if err != nil {
		t.Fatalf("moving the item: %v", err)
	}
	movedEntry := f.lastEntry(t)
	toNew, err := f.service.UpdateFinanceBudgetItem(t.Context(), daniel, item.ID, ItemUpdate{BudgetName: text("2026 Retail Budget"), Version: version(2)})
	if err != nil {
		t.Fatalf("moving the item to a new budget: %v", err)
	}
	entries := f.logEntries(t)

	if moved.BudgetID != mobility.BudgetID || moved.BudgetName != "2026 Mobility Budget" || moved.Year != 2026 || moved.Code != "" ||
		moved.Name != "Europe – Digital media" || moved.Version != 2 || moved.UpdatedBy != daniel || !reflect.DeepEqual(moved.Locks, yearLock) {
		t.Errorf("moved = %+v", moved)
	}
	wantChanges := []history.Change{
		{Field: "budgetId", Label: "Budget", Old: json.RawMessage(`"FB-1001"`), New: json.RawMessage(`"FB-1002"`),
			OldText: "2026 Lubricants Budget", NewText: "2026 Mobility Budget"},
		{Field: "code", Label: "Code", Old: json.RawMessage(`"LUB-EU-DIG"`), New: json.RawMessage(`""`), OldText: "LUB-EU-DIG", NewText: "Not filled in"},
		{Field: "name", Label: "Item name", Old: json.RawMessage(`"Europe – Digital"`), New: json.RawMessage(`"Europe – Digital media"`),
			OldText: "Europe – Digital", NewText: "Europe – Digital media"},
	}
	if movedEntry.Action != history.ActionChanged || movedEntry.RecordType != "financeBudgetItem" || !reflect.DeepEqual(movedEntry.Changes, wantChanges) {
		t.Errorf("moved entry = %+v", movedEntry)
	}
	budgetEntry, itemEntry := entries[len(entries)-2], entries[len(entries)-1]
	if toNew.BudgetName != "2026 Retail Budget" || toNew.Year != 2026 || toNew.BudgetID == mobility.BudgetID ||
		budgetEntry.RecordType != "financeBudget" || budgetEntry.Action != history.ActionCreated || itemEntry.Changes[0].NewText != "2026 Retail Budget" {
		t.Errorf("to a new budget = %+v; entries %+v, %+v", toNew, budgetEntry, itemEntry)
	}

	_, clash := f.service.UpdateFinanceBudgetItem(t.Context(), alex, mobility.ID, ItemUpdate{BudgetName: text("2026 Retail Budget"),
		Name: text("europe – digital MEDIA"), Version: version(1)})
	_, stale := f.service.UpdateFinanceBudgetItem(t.Context(), alex, item.ID, ItemUpdate{Name: text("X"), Version: version(1)})
	_, noVersion := f.service.UpdateFinanceBudgetItem(t.Context(), alex, item.ID, ItemUpdate{BudgetName: text("")})
	_, gone := f.service.UpdateFinanceBudgetItem(t.Context(), alex, "FBI-9999", ItemUpdate{Name: text("X"), Version: version(1)})
	assertError(t, clash, problems("name", "The budget '2026 Retail Budget' already has an item called 'Europe – Digital media'."))
	assertError(t, stale, errs.VersionConflict("finance budget item", "Daniel Okoro", f.clock.now))
	assertError(t, noVersion, problems("version", "Reload the page and try again.", "budgetName", "Enter a budget name."))
	assertError(t, gone, errs.NotFound("This finance budget item no longer exists — someone may have deleted it."))
}

// Rename budget renames the one budget record, so all its items and the plans using it as their default
// show the new name at once; one "changed" entry on the budget (US5-4, FR-015).
func TestRenameBudget_AllItemsFollow_FR015(t *testing.T) {
	f := newTestService(t)
	digital := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", ""))
	f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Events", ""))
	plan := planRef("P-1001", "Lubricants FY2026", "MOD-1001", false, nil)
	plan.FinanceBudgetID = digital.BudgetID
	f.refs = []Reference{plan}
	f.clock.now = start.Add(time.Hour)

	renamed, err := f.service.RenameFinanceBudget(t.Context(), daniel, digital.BudgetID, RenameInput{Name: " 2026 Lubricants Budget (EU) ", Version: version(1)})
	if err != nil {
		t.Fatalf("RenameFinanceBudget: %v", err)
	}
	entry := f.lastEntry(t)
	same, err2 := f.service.RenameFinanceBudget(t.Context(), alex, digital.BudgetID, RenameInput{Name: "2026 Lubricants Budget (EU)", Version: version(2)})

	if renamed.Name != "2026 Lubricants Budget (EU)" || renamed.Version != 2 || renamed.UpdatedBy != daniel || renamed.ActiveItemCount != 2 ||
		renamed.Items[0].BudgetName != "2026 Lubricants Budget (EU)" || renamed.Items[1].BudgetName != "2026 Lubricants Budget (EU)" ||
		renamed.Usage.Summary != "Default finance budget of 1 plan" || len(renamed.Usage.Records) != 1 {
		t.Errorf("renamed = %+v", renamed)
	}
	if item, _ := f.service.GetFinanceBudgetItem(t.Context(), alex, digital.ID); item.BudgetName != "2026 Lubricants Budget (EU)" {
		t.Errorf("the item's budget = %q", item.BudgetName)
	}
	wantChange := history.Change{Field: "name", Label: "Budget name", Old: json.RawMessage(`"2026 Lubricants Budget"`),
		New: json.RawMessage(`"2026 Lubricants Budget (EU)"`), OldText: "2026 Lubricants Budget", NewText: "2026 Lubricants Budget (EU)"}
	if entry.Action != history.ActionChanged || entry.RecordType != "financeBudget" || !reflect.DeepEqual(entry.Changes, []history.Change{wantChange}) {
		t.Errorf("entry = %+v", entry)
	}
	if err2 != nil || same.Version != 2 || f.lastEntry(t).ID != entry.ID {
		t.Errorf("an unchanged rename = %+v, %v", same, err2)
	}
	_, stale := f.service.RenameFinanceBudget(t.Context(), alex, digital.BudgetID, RenameInput{Name: "Other", Version: version(1)})
	_, noVersion := f.service.RenameFinanceBudget(t.Context(), alex, digital.BudgetID, RenameInput{Name: "Other"})
	_, gone := f.service.RenameFinanceBudget(t.Context(), alex, "FB-9999", RenameInput{Name: "Other", Version: version(1)})
	assertError(t, stale, errs.VersionConflict("finance budget", "Daniel Okoro", f.clock.now))
	assertError(t, noVersion, problems("version", "Reload the page and try again."))
	assertError(t, gone, errs.NotFound("This finance budget no longer exists — someone may have deleted it."))
}

// A budget name is unique within its year, ignoring letter case; the same name in another year is free
// (FR-015).
func TestRenameBudget_NameTakenInYear_FR015(t *testing.T) {
	f := newTestService(t)
	f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", ""))
	mobility := f.createItem(t, itemInput(2026, "2026 Mobility Budget", "China – Retail", ""))
	later := f.createItem(t, itemInput(2027, "2027 Budget", "China – Retail", ""))

	_, taken := f.service.RenameFinanceBudget(t.Context(), alex, mobility.BudgetID, RenameInput{Name: " 2026 LUBRICANTS budget", Version: version(1)})
	_, empty := f.service.RenameFinanceBudget(t.Context(), alex, mobility.BudgetID, RenameInput{Name: " ", Version: version(1)})
	_, long := f.service.RenameFinanceBudget(t.Context(), alex, mobility.BudgetID, RenameInput{Name: strings.Repeat("é", 121), Version: version(1)})
	free, err := f.service.RenameFinanceBudget(t.Context(), alex, later.BudgetID, RenameInput{Name: "2026 Lubricants Budget", Version: version(1)})

	assertError(t, taken, problems("name", "2026 already has a budget called '2026 Lubricants Budget'."))
	assertError(t, empty, problems("name", "Enter a budget name."))
	assertError(t, long, problems("name", "Use 120 characters or fewer."))
	if err != nil || free.Name != "2026 Lubricants Budget" || free.Year != 2027 {
		t.Errorf("the name in another year = %+v, %v", free, err)
	}
}

// The list shows one year — the current one unless another is chosen — with its budgets by name and their
// items by name; a budget without items is not listed; the years offered run from the current year − 1 to
// + 5, plus any other year with items (FR-015, US5-3).
func TestListBudgets_YearFilterAndGrouping_FR015(t *testing.T) {
	f := newTestService(t)
	china := f.createItem(t, itemInput(2026, "2026 Mobility Budget", "China – Retail", ""))
	f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Events", ""))
	brand := f.createItem(t, itemInput(2026, "2026 lubricants budget", "europe – Brand", ""))
	f.createItem(t, itemInput(2027, "2027 Lubricants Budget", "Europe – Events", ""))
	f.addItem(t, 2020, "2020 Budget", "Old item", "", true)
	if _, err := f.service.SetFinanceBudgetItemActive(t.Context(), alex, brand.ID, StatusUpdate{Active: yes(false), Version: version(1)}); err != nil {
		t.Fatalf("SetFinanceBudgetItemActive: %v", err)
	}

	current, err := f.service.ListFinanceBudgets(t.Context(), alex, nil)
	if err != nil {
		t.Fatalf("ListFinanceBudgets: %v", err)
	}
	if _, err := f.service.DeleteFinanceBudgetItem(t.Context(), alex, china.ID); err != nil {
		t.Fatalf("DeleteFinanceBudgetItem: %v", err)
	}
	afterDelete, _ := f.service.ListFinanceBudgets(t.Context(), alex, nil)
	empty, _ := f.service.ListFinanceBudgets(t.Context(), alex, ptr(2030))

	var names []string
	for _, b := range current.Items {
		names = append(names, b.Name)
		for _, i := range b.Items {
			names = append(names, "  "+i.Name)
		}
	}
	want := []string{"2026 Lubricants Budget", "  europe – Brand", "  Europe – Events", "2026 Mobility Budget", "  China – Retail"}
	if !reflect.DeepEqual(names, want) || current.Year != 2026 || current.Total != 2 || current.Items[0].ActiveItemCount != 1 {
		t.Errorf("2026 = %q (year %d, total %d)", names, current.Year, current.Total)
	}
	if !reflect.DeepEqual(current.Years, []int{2020, 2025, 2026, 2027, 2028, 2029, 2030, 2031}) {
		t.Errorf("years = %v", current.Years)
	}
	if afterDelete.Total != 1 || afterDelete.Items[0].Name != "2026 Lubricants Budget" {
		t.Errorf("after deleting the only Mobility item = %+v", afterDelete)
	}
	if empty.Year != 2030 || empty.Total != 0 || empty.Items == nil {
		t.Errorf("2030 = %+v", empty)
	}
}

// ptr returns a pointer to n.
func ptr(n int) *int { return &n }

// An item counts the line items linked to it, those in the recycle bin too; its budget shows the plans
// using it as their default finance budget (US5-4, FR-003).
func TestItemUsage_LineItemsAndPlans_FR003(t *testing.T) {
	f := newTestService(t)
	item := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", ""))
	plan := planRef("P-1001", "Lubricants FY2026", "MOD-1001", false, nil)
	plan.FinanceBudgetID = item.BudgetID
	f.refs = []Reference{plan}
	for i, id := range []string{"L-1001", "L-1002", "L-1003", "L-1004", "L-1005", "L-1006"} {
		ref := lineItemRef(id, "P-1001", "A-1001", "USD", "100.00", i == 5)
		ref.FinanceBudgetItemID = item.ID
		f.refs = append(f.refs, ref)
	}

	view, err := f.service.GetFinanceBudgetItem(t.Context(), alex, item.ID)
	page, _ := f.service.ListFinanceBudgets(t.Context(), alex, nil)

	wantItem := Usage{InUse: true, Summary: "Linked to 6 line items (1 in the recycle bin)",
		Parts:   []UsagePart{{Kind: "lineItem", Text: "Linked to 6 line items (1 in the recycle bin)", Count: 6, InRecycleBin: 1}},
		Records: []history.RecordRef{}}
	if err != nil || !reflect.DeepEqual(view.Usage, wantItem) {
		t.Errorf("item usage = %+v, %v\nwant %+v", view.Usage, err, wantItem)
	}
	wantBudget := Usage{InUse: true, Summary: "Default finance budget of 1 plan",
		Parts:   []UsagePart{{Kind: "plan", Text: "Default finance budget of 1 plan", Count: 1}},
		Records: []history.RecordRef{{Type: "plan", ID: "P-1001", Name: "Lubricants FY2026"}}}
	if !reflect.DeepEqual(page.Items[0].Usage, wantBudget) {
		t.Errorf("budget usage = %+v\nwant %+v", page.Items[0].Usage, wantBudget)
	}
}

// An item in use — linked to line items, also in the recycle bin — cannot be deleted: the answer says
// where it is used and to set it inactive instead (FR-004, SC-004).
func TestDeleteItem_InUseRefused_FR004(t *testing.T) {
	f := newTestService(t)
	item := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", ""))
	ref := lineItemRef("L-1001", "P-1001", "A-1001", "USD", "100.00", true)
	ref.FinanceBudgetItemID = item.ID
	f.refs = []Reference{ref}

	_, used := f.service.DeleteFinanceBudgetItem(t.Context(), alex, item.ID)
	_, gone := f.service.DeleteFinanceBudgetItem(t.Context(), alex, "FBI-9999")

	assertError(t, used, errs.InUse("'Europe – Digital' is in use (linked to 1 line item (1 in the recycle bin)) — set it inactive instead."))
	assertError(t, gone, errs.NotFound("This finance budget item no longer exists — someone may have deleted it."))
}

// An unused item can be set inactive and reactivated, each logged; deleted, it moves to the recycle bin
// under its budget and Undo brings it back exactly (US5-5, FR-004, FR-006).
func TestDeleteItem_Unused_Undo_FR004(t *testing.T) {
	f := newTestService(t)
	item := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", "LUB-EU-DIG"))

	inactive, err := f.service.SetFinanceBudgetItemActive(t.Context(), alex, item.ID, StatusUpdate{Active: yes(false), Version: version(1)})
	if err != nil || inactive.Active || inactive.Version != 2 || f.lastEntry(t).Verb != "deactivated" {
		t.Fatalf("set inactive = %+v, %v", inactive, err)
	}
	_, stale := f.service.SetFinanceBudgetItemActive(t.Context(), alex, item.ID, StatusUpdate{Active: yes(true), Version: version(1)})
	assertError(t, stale, errs.VersionConflict("finance budget item", "Alex Morgan", start))
	if active, err := f.service.SetFinanceBudgetItemActive(t.Context(), alex, item.ID, StatusUpdate{Active: yes(true), Version: version(2)}); err != nil ||
		!active.Active || f.lastEntry(t).Verb != "reactivated" {
		t.Fatalf("reactivate = %+v, %v", active, err)
	}
	stored, _, _ := f.service.FinanceBudgetItem(t.Context(), item.ID)

	entry, err := f.service.DeleteFinanceBudgetItem(t.Context(), alex, item.ID)
	if err != nil || entry.RecordType != "financeBudgetItem" || entry.RecordName != "Europe – Digital" ||
		entry.Parent == nil || entry.Parent.Name != "2026 Lubricants Budget" {
		t.Fatalf("DeleteFinanceBudgetItem = %+v, %v", entry, err)
	}
	if _, found, _ := f.service.FinanceBudgetItem(t.Context(), item.ID); found {
		t.Errorf("the item is still stored")
	}
	if _, err := f.history.Undo(t.Context(), alex, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if back, found, _ := f.service.FinanceBudgetItem(t.Context(), item.ID); !found || !reflect.DeepEqual(back, stored) {
		t.Errorf("after Undo = %+v, want %+v", back, stored)
	}
}

// An item whose code another item of its year took meanwhile: Undo is refused, and Restore brings it back
// without the code and says so (FR-014, D-054).
func TestRestoreItem_CodeTaken_FR014(t *testing.T) {
	f := newTestService(t)
	item := f.createItem(t, itemInput(2026, "2026 Lubricants Budget", "Europe – Digital", "LUB-EU-DIG"))
	entry, err := f.service.DeleteFinanceBudgetItem(t.Context(), alex, item.ID)
	if err != nil {
		t.Fatalf("DeleteFinanceBudgetItem: %v", err)
	}
	f.createItem(t, itemInput(2026, "2026 Mobility Budget", "China", "lub-eu-dig"))

	_, undo := f.history.Undo(t.Context(), alex, entry.ID)
	restored, err := f.history.Restore(t.Context(), daniel, entry.ID)

	assertError(t, undo, errs.NameTaken("Can't undo — the code 'LUB-EU-DIG' is now used by 'China'. Restore it from the recycle bin instead."))
	if err != nil || restored.Details != "Code 'LUB-EU-DIG' removed because another item now uses it." {
		t.Errorf("Restore = %+v, %v", restored, err)
	}
	if back, _, _ := f.service.FinanceBudgetItem(t.Context(), item.ID); back.Code != "" || back.Name != "Europe – Digital" || back.UpdatedBy != daniel {
		t.Errorf("restored item = %+v", back)
	}
}
