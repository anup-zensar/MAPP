package settings

import (
	"strings"
	"testing"
)

// Every problem of an item is shown at once: the year (current − 1 to + 5), the budget name, the item name
// (unique in its budget) and the code (unique in the year, ignoring case) (FR-014, SC-003).
func TestCreateItem_Validation_FR014(t *testing.T) {
	t.Parallel()
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

// A budget name is unique within its year, ignoring letter case; the same name in another year is free
// (FR-015).
func TestRenameBudget_NameTakenInYear_FR015(t *testing.T) {
	t.Parallel()
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
