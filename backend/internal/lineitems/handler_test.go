package lineitems_test

import (
	"net/http"
	"testing"

	"mapp/internal/lineitems"
	"mapp/internal/platform/apitest"
)

// The demo records these tests use (seed: 006 activities, 007 line items).
const (
	priyasEmail = "priya.raman@shell.com"
	launchFilm  = "A-1001" // Created, 120,000.00 USD, Mar–May 2026: Agency fee L-1001 and Film production L-1002
	socialOnAir = "A-1002" // Approved: Venue hire L-1003 and Speaker fees L-1004
	fleetShow   = "A-1005" // Declined, no line items
)

// The answers that do not depend on the data.
const (
	activityGoneBody = `{"code":"not_found",` +
		`"message":"This activity was deleted — restore it from the recycle bin to keep working on its line items.","fields":[]}`
	restartBody = `{"code":"validation_failed","message":"Some fields need your attention.","fields":[` +
		`{"field":"restartApproval","message":"This activity will need approval again. Confirm the change to send it back for approval."}]}`
)

// newLineItem is the body of a create with the fields the form always sends.
func newLineItem(name, currency, cost string) map[string]any {
	return map[string]any{
		"name": name, "ownerId": "U-1002", "description": "", "currency": currency, "plannedCost": cost,
		"spread": map[string]any{"method": "evenly", "months": []any{}}, "attributes": map[string]any{},
		"contractedAmount": nil, "requestSwbs": true, "financeBudgetYear": 2026, "financeBudgetItemId": nil,
	}
}

// lineItemGoneBody is the answer about a line item that no longer exists.
const lineItemGoneBody = `{"code":"not_found","message":"This line item no longer exists — someone may have deleted it.","fields":[]}`

// formOf returns the whole form of a line item as the edit page sends it, with its version.
func formOf(v lineitems.View) map[string]any {
	months := []any{}
	if v.Spread.Method == lineitems.SpreadCustom {
		for _, m := range v.Spread.Months {
			months = append(months, map[string]any{"month": m.Month, "amount": m.Amount.Amount.String()})
		}
	}
	form := map[string]any{
		"name": v.Name, "ownerId": nil, "description": v.Description, "currency": v.Currency.Code,
		"plannedCost": v.PlannedCost.Amount.String(), "spread": map[string]any{"method": v.Spread.Method, "months": months},
		"attributes": v.Attributes, "contractedAmount": nil, "requestSwbs": v.RequestSwbs,
		"financeBudgetYear": v.FinanceBudgetYear, "financeBudgetItemId": nil, "version": v.Version,
	}
	if v.Owner != nil {
		form["ownerId"] = v.Owner.ID
	}
	if v.ContractedAmount != nil {
		form["contractedAmount"] = v.ContractedAmount.Amount.String()
	}
	if v.FinanceBudgetItem != nil {
		form["financeBudgetItemId"] = v.FinanceBudgetItem.ID
	}
	return form
}

// lineItem reads the line item id through GET /api/line-items/{id}.
func lineItem(t *testing.T, client *apitest.Client, id string) lineitems.View {
	t.Helper()
	response := client.Do(t, http.MethodGet, "/api/line-items/"+id, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s", id, response.Status, response.Body)
	}
	var details lineitems.Details
	response.Decode(t, &details)
	return details.LineItem
}

// assertAnswer checks the status and, when want is not empty, the exact body of an answer.
func assertAnswer(t *testing.T, response *apitest.Response, status int, want string) {
	t.Helper()
	if response.Status != status || (want != "" && string(response.Body) != want) {
		t.Errorf("answer = %d %s\nwant     %d %s", response.Status, response.Body, status, want)
	}
}

// GET …/line-items answers Launch film's section (US1 scenario 1): Agency fee and Film production (46,000.00
// EUR = 50,000.00 USD, "3 fields needed"), 20,000.00 USD unassigned and the grid over Mar–May 2026; an
// activity without line items has its whole planned cost unassigned (FR-004); 404 for an activity that is
// gone.
func TestListActivityLineItems_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/activities/"+launchFilm+"/line-items", nil)
	empty := priya.Do(t, http.MethodGet, "/api/activities/"+fleetShow+"/line-items", nil)
	gone := priya.Do(t, http.MethodGet, "/api/activities/A-9999/line-items", nil)

	assertAnswer(t, response, http.StatusOK, "")
	var section lineitems.Section
	response.Decode(t, &section)
	if len(section.Items) != 2 || section.Items[0].Name != "Agency fee" || section.Items[1].Name != "Film production" {
		t.Fatalf("items = %+v", section.Items)
	}
	fee, film := section.Items[0], section.Items[1]
	if fee.ID != "L-1001" || len(fee.Missing) != 0 || fee.SwbsStatus != "waitingForApproval" ||
		film.PlannedCostUSD.Format() != "50,000.00 USD" || len(film.Missing) != 3 {
		t.Errorf("Agency fee missing %+v; Film production %s missing %+v", fee.Missing, film.PlannedCostUSD.Format(), film.Missing)
	}
	if b := section.Balance; b.State != "left" || b.Balance.Format() != "20,000.00 USD" || b.Assigned.Format() != "100,000.00 USD" {
		t.Errorf("balance = %+v", b)
	}
	if g := section.Grid; len(g.Columns) != 3 || g.Columns[0].Key != "2026-03" || g.Columns[2].Key != "2026-05" ||
		g.Total.Format() != "100,000.00 USD" {
		t.Errorf("grid = %+v", g)
	}

	assertAnswer(t, empty, http.StatusOK, "")
	var none lineitems.Section
	empty.Decode(t, &none)
	if len(none.Items) != 0 || none.Balance.Balance == nil || none.Balance.Balance.Format() != "60,000.00 USD" {
		t.Errorf("Fleet roadshow: %+v", none)
	}
	assertAnswer(t, gone, http.StatusNotFound, activityGoneBody)
}

// GET …/line-items/form-options answers the defaults and choices of a new line item of Launch film: its
// 20,000.00 USD unassigned planned cost as the planned cost (FR-008); 404 for an activity that is gone.
func TestGetLineItemFormOptions_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/activities/"+launchFilm+"/line-items/form-options", nil)
	gone := priya.Do(t, http.MethodGet, "/api/activities/A-9999/line-items/form-options", nil)

	assertAnswer(t, response, http.StatusOK, "")
	var options lineitems.FormOptions
	response.Decode(t, &options)
	d := options.Defaults
	if d.Name != "Launch film – line item" || d.Owner == nil || d.Owner.Name != "Priya Raman" || d.Currency != "USD" ||
		d.PlannedCost == nil || d.PlannedCost.Format() != "20,000.00 USD" || d.PlannedCostSource != "unassignedBalance" ||
		d.SpreadMethod != "evenly" || d.FinanceBudgetYear != 2026 || d.FinanceBudgetItemID != nil || !d.RequestSwbs {
		t.Errorf("defaults = %+v", d)
	}
	if len(options.Currencies) != 5 || options.Currencies[0].Code != "USD" || len(options.FinanceBudgetItems) != 6 ||
		options.FinanceBudgetItems[0].BudgetName != "2026 Lubricants Budget" {
		t.Errorf("choices = %+v / %+v", options.Currencies, options.FinanceBudgetItems)
	}
	assertAnswer(t, gone, http.StatusNotFound, activityGoneBody)
}

// POST …/line-items creates a line item with the next ID and answers 201 with it (FR-009, FR-010); every
// problem at once; 404 for an activity that is gone; on an Approved activity only with restartApproval
// (FR-031).
func TestCreateLineItem_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	created := priya.Do(t, http.MethodPost, "/api/activities/"+launchFilm+"/line-items", newLineItem("Music rights", "EUR", "10000"))
	invalid := priya.Do(t, http.MethodPost, "/api/activities/"+launchFilm+"/line-items", newLineItem(" ", "USD", "-50"))
	gone := priya.Do(t, http.MethodPost, "/api/activities/A-9999/line-items", newLineItem("Music rights", "USD", "1"))
	unconfirmed := priya.Do(t, http.MethodPost, "/api/activities/"+socialOnAir+"/line-items", newLineItem("Catering", "USD", "1"))
	confirm := newLineItem("Catering", "USD", "1")
	confirm["restartApproval"] = true
	confirmed := priya.Do(t, http.MethodPost, "/api/activities/"+socialOnAir+"/line-items", confirm)

	assertAnswer(t, created, http.StatusCreated, "")
	var v lineitems.View
	created.Decode(t, &v)
	// L-1011 is the line item of the demo history's deleted initiative, in the recycle bin (011 T019); L-1009 and
	// L-1010 are 016's GBP plan.
	if v.ID != "L-1012" || v.Name != "Music rights" || v.PlannedCost.Format() != "10,000.00 EUR" ||
		v.PlannedCostUSD.Format() != "10,869.57 USD" || v.SwbsStatus != "waitingForApproval" || len(v.Spread.Months) != 3 ||
		v.Owner == nil || v.Version != 1 || v.CreatedBy.Name != "Priya Raman" {
		t.Errorf("created = %+v", v)
	}
	assertAnswer(t, invalid, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.","fields":[`+
		`{"field":"name","message":"Enter a name for the line item."},`+
		`{"field":"plannedCost","message":"Planned cost can't be negative."}]}`)
	assertAnswer(t, gone, http.StatusNotFound, activityGoneBody)
	assertAnswer(t, unconfirmed, http.StatusBadRequest, restartBody)
	assertAnswer(t, confirmed, http.StatusCreated, "")
}

// GET /api/line-items/{id} answers the line item with its activity and the balance (FR-026); 404 for one that
// does not exist.
func TestGetLineItem_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/line-items/L-1001", nil)
	gone := priya.Do(t, http.MethodGet, "/api/line-items/L-9999", nil)

	assertAnswer(t, response, http.StatusOK, "")
	var details lineitems.Details
	response.Decode(t, &details)
	if details.LineItem.Name != "Agency fee" || details.LineItem.PlannedCost.Format() != "50,000.00 USD" ||
		details.LineItem.FinanceBudgetItem == nil || details.Activity.Name != "Launch film" || len(details.Activity.Path) != 3 ||
		details.Balance.Balance.Format() != "20,000.00 USD" {
		t.Errorf("details = %+v", details)
	}
	assertAnswer(t, gone, http.StatusNotFound, lineItemGoneBody)
}

// PATCH /api/line-items/{id} saves the whole form with its version (FR-027): the next version; a missing
// version with the other problems; an out-of-date one 409 version_conflict; a cost change on an Approved
// activity only with restartApproval (FR-031); 404 for one that does not exist.
func TestUpdateLineItem_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	film := formOf(lineItem(t, priya, "L-1002"))
	film["plannedCost"] = "46500.00"

	saved := priya.Do(t, http.MethodPatch, "/api/line-items/L-1002", film)
	stale := priya.Do(t, http.MethodPatch, "/api/line-items/L-1002", film) // still version 1
	delete(film, "version")
	film["name"] = ""
	noVersion := priya.Do(t, http.MethodPatch, "/api/line-items/L-1002", film, apitest.SkipRequestValidation())
	venue := formOf(lineItem(t, priya, "L-1003"))
	venue["plannedCost"] = "17000.00"
	unconfirmed := priya.Do(t, http.MethodPatch, "/api/line-items/L-1003", venue)
	gone := priya.Do(t, http.MethodPatch, "/api/line-items/L-9999", venue)

	assertAnswer(t, saved, http.StatusOK, "")
	var v lineitems.View
	saved.Decode(t, &v)
	if v.Version != 2 || v.PlannedCost.Format() != "46,500.00 EUR" || v.PlannedCostUSD.Format() != "50,543.48 USD" {
		t.Errorf("saved = version %d, %s = %s", v.Version, v.PlannedCost.Format(), v.PlannedCostUSD.Format())
	}
	assertAnswer(t, stale, http.StatusConflict, "")
	var conflict struct{ Code string }
	stale.Decode(t, &conflict)
	if conflict.Code != "version_conflict" {
		t.Errorf("stale save: %s", stale.Body)
	}
	assertAnswer(t, noVersion, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.","fields":[`+
		`{"field":"name","message":"Enter a name for the line item."},`+
		`{"field":"version","message":"Reload the page and try again."}]}`)
	assertAnswer(t, unconfirmed, http.StatusBadRequest, restartBody)
	assertAnswer(t, gone, http.StatusNotFound, lineItemGoneBody)
}

// POST /api/line-items/{id}/duplicate answers 201 with the copy, "Agency fee (copy)" (FR-029); on an Approved
// activity only with restartApproval (FR-031); 404 for one that does not exist.
func TestDuplicateLineItem_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	copied := priya.Do(t, http.MethodPost, "/api/line-items/L-1001/duplicate", map[string]any{})
	unconfirmed := priya.Do(t, http.MethodPost, "/api/line-items/L-1003/duplicate", map[string]any{})
	confirmed := priya.Do(t, http.MethodPost, "/api/line-items/L-1003/duplicate", map[string]any{"restartApproval": true})
	gone := priya.Do(t, http.MethodPost, "/api/line-items/L-9999/duplicate", map[string]any{})

	assertAnswer(t, copied, http.StatusCreated, "")
	var v lineitems.View
	copied.Decode(t, &v)
	if v.ID != "L-1012" || v.Name != "Agency fee (copy)" || v.PlannedCost.Format() != "50,000.00 USD" || len(v.Missing) != 0 {
		t.Errorf("copy = %+v", v)
	}
	assertAnswer(t, unconfirmed, http.StatusBadRequest, restartBody)
	assertAnswer(t, confirmed, http.StatusCreated, "")
	assertAnswer(t, gone, http.StatusNotFound, lineItemGoneBody)
}

// DELETE /api/line-items/{id} moves the line item to the recycle bin and answers the entry, and Undo brings
// it back with the same ID (FR-030); on an Approved activity only with ?restartApproval=true (FR-031); a
// query value other than true or false is refused; 404 for one that does not exist.
func TestDeleteLineItem_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	deleted := priya.Do(t, http.MethodDelete, "/api/line-items/L-1001", nil)
	gone := priya.Do(t, http.MethodGet, "/api/line-items/L-1001", nil)
	var result struct {
		RecycleBinEntry struct {
			ID, RecordType, RecordID string
		} `json:"recycleBinEntry"`
	}
	deleted.Decode(t, &result)
	undone := priya.Do(t, http.MethodPost, "/api/recycle-bin/"+result.RecycleBinEntry.ID+"/undo", nil)
	unconfirmed := priya.Do(t, http.MethodDelete, "/api/line-items/L-1003", nil)
	unreadable := priya.Do(t, http.MethodDelete, "/api/line-items/L-1003?restartApproval=maybe", nil, apitest.SkipRequestValidation())
	confirmed := priya.Do(t, http.MethodDelete, "/api/line-items/L-1003?restartApproval=true", nil)
	unknown := priya.Do(t, http.MethodDelete, "/api/line-items/L-9999", nil)

	assertAnswer(t, deleted, http.StatusOK, "")
	if result.RecycleBinEntry.RecordType != "lineItem" || result.RecycleBinEntry.RecordID != "L-1001" {
		t.Errorf("entry = %+v", result.RecycleBinEntry)
	}
	assertAnswer(t, gone, http.StatusNotFound, lineItemGoneBody)
	assertAnswer(t, undone, http.StatusOK, "")
	if back := lineItem(t, priya, "L-1001"); back.Name != "Agency fee" || back.Version != 1 {
		t.Errorf("after Undo: %+v", back)
	}
	assertAnswer(t, unconfirmed, http.StatusBadRequest, restartBody)
	assertAnswer(t, unreadable, http.StatusBadRequest, `{"code":"bad_request",`+
		`"message":"MAPP couldn't read this request. Reload the page and try again.","fields":[]}`)
	assertAnswer(t, confirmed, http.StatusOK, "")
	assertAnswer(t, unknown, http.StatusNotFound, lineItemGoneBody)
}

// The line items of A-1011 "UK launch event" in the GBP plan P-1004 (016 FR-015): each keeps its own currency
// and shows its planned cost in the plan currency; the unassigned planned cost and the spread grid are worked
// out in GBP — 40,000.00 − (15,000.00 + 19,750.00) = 5,250.00 GBP — and answered in Priya's display currency,
// each amount converted once from GBP (contract UnassignedBalance and SpreadGrid, D-213): 5,250.00 GBP =
// 6,645.57 USD, 34,750.00 GBP = 43,987.34 USD, 40,000.00 GBP = 50,632.91 USD.
func TestListActivityLineItems_PlanCurrency_Contract_FR015(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/activities/A-1011/line-items", nil)

	assertAnswer(t, response, http.StatusOK, "")
	var section lineitems.Section
	response.Decode(t, &section)
	if len(section.Items) != 2 {
		t.Fatalf("items = %+v", section.Items)
	}
	venue, agency := section.Items[0], section.Items[1]
	if venue.PlannedCostPlan.Format() != "15,000.00 GBP" || agency.PlannedCost.Format() != "23,000.00 EUR" ||
		agency.PlannedCostPlan.Format() != "19,750.00 GBP" || agency.PlannedCostUSD.Format() != "25,000.00 USD" {
		t.Errorf("venue %v; agency %v = %v = %v", venue.PlannedCostPlan, agency.PlannedCost, agency.PlannedCostPlan, agency.PlannedCostUSD)
	}
	if b := section.Balance; b.State != "left" || b.Balance.Format() != "6,645.57 USD" || b.Assigned.Format() != "43,987.34 USD" ||
		b.PlannedCost.Format() != "50,632.91 USD" {
		t.Errorf("balance = %v of %v, assigned %v, %s", b.Balance, b.PlannedCost, b.Assigned, b.State)
	}
	if g := section.Grid; g.Total.Format() != "43,987.34 USD" || len(g.Columns) != 3 || g.Columns[0].Key != "2027-03" {
		t.Errorf("grid total %v, columns %+v", g.Total, g.Columns)
	}
}
