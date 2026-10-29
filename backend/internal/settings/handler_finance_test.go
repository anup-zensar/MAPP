package settings_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
)

// The finance budgets of the current year come by name with their items; the years run from the current
// year − 1 to + 5; one item answers with its budget and its year locked (FR-014, FR-015).
func TestFinanceAPI_ListYear_FR015(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	var current, next budgetPage
	var item struct {
		BudgetName string
		Year       int
		Locks      []struct{ Field string }
	}

	alex.Do(t, http.MethodGet, "/api/v1/finance-budgets", nil).Decode(t, &current)
	alex.Do(t, http.MethodGet, "/api/v1/finance-budgets?year=2027", nil).Decode(t, &next)
	alex.Do(t, http.MethodGet, "/api/v1/finance-budget-items/FBI-1002", nil).Decode(t, &item)
	notANumber := alex.Do(t, http.MethodGet, "/api/v1/finance-budgets?year=next", nil, apitest.SkipRequestValidation())
	gone := alex.Do(t, http.MethodGet, "/api/v1/finance-budget-items/FBI-9999", nil)

	assertBudgets2026(t, current)
	if next.Year != 2027 || next.Total != 1 || len(next.Items) != 1 || next.Items[0].Name != "2027 Lubricants Budget" { // 038 L-30 seed
		t.Errorf("2027 = %+v", next)
	}
	if item.BudgetName != "2026 Lubricants Budget" || item.Year != 2026 || len(item.Locks) != 1 || item.Locks[0].Field != "year" {
		t.Errorf("item = %+v", item)
	}
	if notANumber.Status != http.StatusBadRequest {
		t.Errorf("year=next = %d %s", notANumber.Status, notANumber.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, itemGoneBody)
}

// budgetPage is the part of a finance budget list these tests read.
type budgetPage struct {
	Items []struct {
		Name            string
		ActiveItemCount int
		Items           []struct{ Name string }
	}
	Total, Year int
	Years       []int
}

// assertBudgets2026 checks the demo's budgets of 2026: two, the Lubricants budget first with its three
// active items, and the years 2025 to 2031.
func assertBudgets2026(t *testing.T, current budgetPage) {
	t.Helper()
	if current.Year != 2026 || current.Total != 2 || current.Items[0].Name != "2026 Lubricants Budget" ||
		current.Items[0].ActiveItemCount != 3 || current.Items[0].Items[0].Name != "Europe – Brand" ||
		!slices.Equal(current.Years, []int{2025, 2026, 2027, 2028, 2029, 2030, 2031}) {
		t.Errorf("2026 = %+v", current)
	}
}

// Bodies of the finance tests.
const (
	itemGoneBody   = `{"error":{"code":"not_found","message":"This finance budget item no longer exists — someone may have deleted it.","fields":[]}}`
	budgetGoneBody = `{"error":{"code":"not_found","message":"This finance budget no longer exists — someone may have deleted it.","fields":[]}}`
	yearLockBody   = `[{"field":"year","reason":"The year can't change once the item is saved — create a new item for another year."}]`
)

// retailBody is the new item of US5-3: "Europe – Retail" in "2026 Lubricants Budget", code "LUB-EU-RET".
var retailBody = map[string]any{"year": 2026, "budgetName": "2026 Lubricants Budget", "name": "Europe – Retail", "code": "LUB-EU-RET"}

// retailAnswer is the answer for the active item retailBody (FBI-1009) in the budget budget (id and name) at
// version, with its code, created by Alex.
func retailAnswer(budgetID, budgetName, code string, version int) string {
	return `{"id":"FBI-1009","budgetId":"` + budgetID + `","budgetName":"` + budgetName + `","year":2026,"name":"Europe – Retail","code":"` + code +
		`","active":true,"usage":` + unusedBody + `,"locks":` + yearLockBody + `,"version":` + itoa(version) + `,` + alexAtStart + `}`
}

// A new item answers 201 in the budget of its year and name and is read back; a new budget name moves it to
// a new budget of the same year; every problem comes at once (FR-014, US5-3, US5-4).
func TestFinanceAPI_CreateGetUpdate_FR014(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	created := alex.Do(t, http.MethodPost, "/api/v1/finance-budget-items", retailBody)
	got := alex.Do(t, http.MethodGet, "/api/v1/finance-budget-items/FBI-1009", nil)
	moved := alex.Do(t, http.MethodPatch, "/api/v1/finance-budget-items/FBI-1009", map[string]any{"budgetName": "2026 Retail Budget", "code": "", "version": 1})
	invalid := alex.Do(t, http.MethodPost, "/api/v1/finance-budget-items", map[string]any{"year": 2040, "budgetName": " ", "name": ""})
	stale := alex.Do(t, http.MethodPatch, "/api/v1/finance-budget-items/FBI-1009", map[string]any{"name": "X", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/v1/finance-budget-items/FBI-9999", map[string]any{"name": "X", "version": 1})

	assertAnswer(t, created, http.StatusCreated, retailAnswer("FB-1001", "2026 Lubricants Budget", "LUB-EU-RET", 1))
	assertAnswer(t, got, http.StatusOK, retailAnswer("FB-1001", "2026 Lubricants Budget", "LUB-EU-RET", 1))
	assertAnswer(t, moved, http.StatusOK, retailAnswer("FB-1004", "2026 Retail Budget", "", 2))
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"year","message":"Choose a year from 2025 to 2031."},`+
		`{"field":"budgetName","message":"Enter a budget name."},{"field":"name","message":"Enter a name."}`))
	if stale.Status != http.StatusConflict {
		t.Errorf("stale update = %d %s", stale.Status, stale.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, itemGoneBody)
}

// Rename budget answers the budget with all its items under the new name; the name must be free in its
// year; an out-of-date version is refused (FR-015, US5-4).
func TestFinanceAPI_Rename_FR015(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	var renamed struct {
		Name    string
		Version int
		Items   []struct{ BudgetName string }
		Usage   struct{ Summary string }
	}
	var item struct{ BudgetName string }

	alex.Do(t, http.MethodPatch, "/api/v1/finance-budgets/FB-1002", map[string]any{"name": "2026 Mobility Budget (China)", "version": 1}).Decode(t, &renamed)
	alex.Do(t, http.MethodGet, "/api/v1/finance-budget-items/FBI-1004", nil).Decode(t, &item)
	taken := alex.Do(t, http.MethodPatch, "/api/v1/finance-budgets/FB-1002", map[string]any{"name": "2026 lubricants budget", "version": 2})
	stale := alex.Do(t, http.MethodPatch, "/api/v1/finance-budgets/FB-1002", map[string]any{"name": "Other", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/v1/finance-budgets/FB-9999", map[string]any{"name": "Other", "version": 1})

	if renamed.Name != "2026 Mobility Budget (China)" || renamed.Version != 2 || len(renamed.Items) != 3 ||
		renamed.Items[0].BudgetName != "2026 Mobility Budget (China)" || !strings.HasPrefix(renamed.Usage.Summary, "Default finance budget of ") {
		t.Errorf("renamed = %+v", renamed)
	}
	if item.BudgetName != "2026 Mobility Budget (China)" {
		t.Errorf("China – Retail is in %q", item.BudgetName)
	}
	assertAnswer(t, taken, http.StatusBadRequest, validationBody(`{"field":"name","message":"2026 already has a budget called '2026 Lubricants Budget'."}`))
	assertAnswer(t, stale, http.StatusConflict, `{"error":{"code":"version_conflict","message":"Not saved — Alex Morgan changed this finance budget",`+
		`"fields":[],"conflict":{"changedBy":"Alex Morgan","changedAt":"2026-03-12T09:00:00Z"}}}`)
	assertAnswer(t, gone, http.StatusNotFound, budgetGoneBody)
}

// Set inactive and Reactivate answer the item with its new status; an out-of-date version is refused
// (FR-004).
func TestFinanceAPI_Status_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/v1/finance-budget-items", retailBody)

	inactive := alex.Do(t, http.MethodPost, "/api/v1/finance-budget-items/FBI-1009/status", map[string]any{"active": false, "version": 1})
	stale := alex.Do(t, http.MethodPost, "/api/v1/finance-budget-items/FBI-1009/status", map[string]any{"active": true, "version": 1})
	active := alex.Do(t, http.MethodPost, "/api/v1/finance-budget-items/FBI-1009/status", map[string]any{"active": true, "version": 2})
	gone := alex.Do(t, http.MethodPost, "/api/v1/finance-budget-items/FBI-9999/status", map[string]any{"active": true, "version": 1})

	assertAnswer(t, inactive, http.StatusOK, replace(retailAnswer("FB-1001", "2026 Lubricants Budget", "LUB-EU-RET", 2), `"active":true`, `"active":false`))
	if stale.Status != http.StatusConflict || active.Status != http.StatusOK {
		t.Errorf("stale %d %s; reactivate %d %s", stale.Status, stale.Body, active.Status, active.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, itemGoneBody)
}

// An unused item moves to the recycle bin under its budget and Undo brings it back; an item linked to line
// items is refused with in_use (FR-004, US5-5).
func TestFinanceAPI_Delete_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/v1/finance-budget-items", retailBody)

	deleted := alex.Do(t, http.MethodDelete, "/api/v1/finance-budget-items/FBI-1009", nil)
	afterDelete := alex.Do(t, http.MethodGet, "/api/v1/finance-budget-items/FBI-1009", nil)
	undone := alex.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/undo", nil)
	inUse := alex.Do(t, http.MethodDelete, "/api/v1/finance-budget-items/FBI-1001", nil)
	gone := alex.Do(t, http.MethodDelete, "/api/v1/finance-budget-items/FBI-9999", nil)
	var refused struct {
		Error struct{ Code, Message string }
	}
	inUse.Decode(t, &refused)

	budget := `{"type":"financeBudget","id":"FB-1001","name":"2026 Lubricants Budget"}`
	assertAnswer(t, deleted, http.StatusOK, `{"recycleBinEntry":{"id":"BIN-1001","recordType":"financeBudgetItem","recordTypeLabel":"finance budget item",`+
		`"recordId":"FBI-1009","recordName":"Europe – Retail","path":[`+budget+`],"parent":`+budget+`,"deletedAt":"2026-03-12T09:00:00Z",`+
		`"deletedBy":{"id":"U-1001","name":"Alex Morgan"},"contents":[]}}`)
	assertAnswer(t, afterDelete, http.StatusNotFound, itemGoneBody)
	assertAnswer(t, undone, http.StatusOK, `{"recordType":"financeBudgetItem","recordTypeLabel":"finance budget item","recordId":"FBI-1009",`+
		`"recordName":"Europe – Retail","details":""}`)
	assertAnswer(t, alex.Do(t, http.MethodGet, "/api/v1/finance-budget-items/FBI-1009", nil), http.StatusOK,
		retailAnswer("FB-1001", "2026 Lubricants Budget", "LUB-EU-RET", 1))
	if inUse.Status != http.StatusConflict || refused.Error.Code != "in_use" ||
		!strings.HasPrefix(refused.Error.Message, "'Europe – Digital' is in use (linked to ") {
		t.Errorf("deleting Europe – Digital = %d %s", inUse.Status, inUse.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, itemGoneBody)
}
