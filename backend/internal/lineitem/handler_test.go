package lineitem_test

import (
	"net/http"
	"testing"

	"mapp/internal/lineitem"
	"mapp/internal/platform/apitest"
)

const (
	priyasEmail = "priya.raman@shell.com"
	launchFilm  = "A-1001" // Created, 120,000.00 USD, Mar–May 2026: Agency fee L-1001 and Film production L-1002
	socialOnAir = "A-1002" // Approved: Venue hire L-1003 and Speaker fees L-1004
	fleetShow   = "A-1005" // Declined, no line items
)

const (
	activityGoneBody = `{"error":{"code":"not_found",` +
		`"message":"This activity was deleted — restore it from the recycle bin to keep working on its line items.","fields":[]}}`
)

func newLineItem(name, currency, cost string) map[string]any {
	return map[string]any{
		"name": name, "ownerId": "U-1002", "description": "", "currency": currency, "plannedCost": cost,
		"spread": map[string]any{"method": "evenly", "months": []any{}}, "attributes": map[string]any{},
		"contractedAmount": nil, "requestSwbs": true, "financeBudgetYear": 2026, "financeBudgetItemId": nil,
	}
}

const lineItemGoneBody = `{"error":{"code":"not_found","message":"This line item no longer exists — someone may have deleted it.","fields":[]}}`

func formOf(v lineitem.View) map[string]any {
	months := []any{}
	if v.Spread.Method == lineitem.SpreadCustom {
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

func lineItem(t *testing.T, client *apitest.Client, id string) lineitem.View {
	t.Helper()
	response := client.Do(t, http.MethodGet, "/api/v1/line-items/"+id, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s", id, response.Status, response.Body)
	}
	var details lineitem.Details
	response.Decode(t, &details)
	return details.LineItem
}

func assertStillApproved(t *testing.T, client *apitest.Client) {
	t.Helper()
	var a struct{ Status string }
	client.Do(t, http.MethodGet, "/api/v1/activities/"+socialOnAir, nil).Decode(t, &a)
	if a.Status != "approved" {
		t.Errorf("Social always-on is %s, want still approved", a.Status)
	}
}

func assertAnswer(t *testing.T, response *apitest.Response, status int, want string) {
	t.Helper()
	if response.Status != status || (want != "" && string(response.Body) != want) {
		t.Errorf("answer = %d %s\nwant     %d %s", response.Status, response.Body, status, want)
	}
}

// An activity without line items has its whole planned cost unassigned; 404 for an activity that is gone.
func TestListActivityLineItems_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/v1/activities/"+launchFilm+"/line-items", nil)
	empty := priya.Do(t, http.MethodGet, "/api/v1/activities/"+fleetShow+"/line-items", nil)
	gone := priya.Do(t, http.MethodGet, "/api/v1/activities/A-9999/line-items", nil)

	assertAnswer(t, response, http.StatusOK, "")
	var section lineitem.Section
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
	var none lineitem.Section
	empty.Decode(t, &none)
	if len(none.Items) != 0 || none.Balance.Balance == nil || none.Balance.Balance.Format() != "60,000.00 USD" {
		t.Errorf("Fleet roadshow: %+v", none)
	}
	assertAnswer(t, gone, http.StatusNotFound, activityGoneBody)
}

// A new line item of Launch film starts with its 20,000.00 USD unassigned planned cost as the planned cost.
func TestGetLineItemFormOptions_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/v1/activities/"+launchFilm+"/line-items/form-options", nil)
	gone := priya.Do(t, http.MethodGet, "/api/v1/activities/A-9999/line-items/form-options", nil)

	assertAnswer(t, response, http.StatusOK, "")
	var options lineitem.FormOptions
	response.Decode(t, &options)
	d := options.Defaults
	if d.Name != "Launch film – line item" || d.Owner == nil || d.Owner.Name != "Priya Raman" || d.Currency != "USD" ||
		d.PlannedCost == nil || d.PlannedCost.Format() != "20,000.00 USD" || d.PlannedCostSource != "unassignedBalance" ||
		d.SpreadMethod != "evenly" || d.FinanceBudgetYear != 2026 || d.FinanceBudgetItemID != nil || !d.RequestSwbs {
		t.Errorf("defaults = %+v", d)
	}
	if len(options.Currencies) != 5 || options.Currencies[0].Code != "USD" || len(options.FinanceBudgetItems) != 8 || // 038 FR-061: and 2027's two
		options.FinanceBudgetItems[0].BudgetName != "2026 Lubricants Budget" {
		t.Errorf("choices = %+v / %+v", options.Currencies, options.FinanceBudgetItems)
	}
	assertAnswer(t, gone, http.StatusNotFound, activityGoneBody)
}

// On an Approved activity a create saves without confirmation and the activity stays Approved; the
// deprecated restartApproval is accepted and ignored (D-322).
func TestCreateLineItem_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	created := priya.Do(t, http.MethodPost, "/api/v1/activities/"+launchFilm+"/line-items", newLineItem("Music rights", "EUR", "10000"))
	invalid := priya.Do(t, http.MethodPost, "/api/v1/activities/"+launchFilm+"/line-items", newLineItem(" ", "USD", "-50"))
	gone := priya.Do(t, http.MethodPost, "/api/v1/activities/A-9999/line-items", newLineItem("Music rights", "USD", "1"))
	onApproved := priya.Do(t, http.MethodPost, "/api/v1/activities/"+socialOnAir+"/line-items", newLineItem("Catering", "USD", "1"))
	oldClient := newLineItem("Drinks", "USD", "1")
	oldClient["restartApproval"] = true
	withOldFlag := priya.Do(t, http.MethodPost, "/api/v1/activities/"+socialOnAir+"/line-items", oldClient)

	assertAnswer(t, created, http.StatusCreated, "")
	var v lineitem.View
	created.Decode(t, &v)
	// L-1009 to L-1011 belong to the GBP plan and to the deleted initiative in the demo recycle bin.
	if v.ID != "L-1012" || v.Name != "Music rights" || v.PlannedCost.Format() != "10,000.00 EUR" ||
		v.PlannedCostUSD.Format() != "10,869.57 USD" || v.SwbsStatus != "waitingForApproval" || len(v.Spread.Months) != 3 ||
		v.Owner == nil || v.Version != 1 || v.CreatedBy.Name != "Priya Raman" {
		t.Errorf("created = %+v", v)
	}
	assertAnswer(t, invalid, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.","fields":[`+
		`{"field":"name","message":"Enter a name for the line item."},`+
		`{"field":"plannedCost","message":"Planned cost can't be negative."}]}}`)
	assertAnswer(t, gone, http.StatusNotFound, activityGoneBody)
	assertAnswer(t, onApproved, http.StatusCreated, "")
	assertAnswer(t, withOldFlag, http.StatusCreated, "")
	assertStillApproved(t, priya)
}

func TestGetLineItem_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/v1/line-items/L-1001", nil)
	gone := priya.Do(t, http.MethodGet, "/api/v1/line-items/L-9999", nil)

	assertAnswer(t, response, http.StatusOK, "")
	var details lineitem.Details
	response.Decode(t, &details)
	if details.LineItem.Name != "Agency fee" || details.LineItem.PlannedCost.Format() != "50,000.00 USD" ||
		details.LineItem.FinanceBudgetItem == nil || details.Activity.Name != "Launch film" || len(details.Activity.Path) != 3 ||
		details.Balance.Balance.Format() != "20,000.00 USD" {
		t.Errorf("details = %+v", details)
	}
	assertAnswer(t, gone, http.StatusNotFound, lineItemGoneBody)
}

// A missing version is reported with the other problems, an out-of-date one is a 409; a cost and currency
// change on an Approved activity saves without confirmation and the activity stays Approved (D-322).
func TestUpdateLineItem_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	film := formOf(lineItem(t, priya, "L-1002"))
	film["plannedCost"] = "46500.00"

	saved := priya.Do(t, http.MethodPatch, "/api/v1/line-items/L-1002", film)
	stale := priya.Do(t, http.MethodPatch, "/api/v1/line-items/L-1002", film) // still version 1
	delete(film, "version")
	film["name"] = ""
	noVersion := priya.Do(t, http.MethodPatch, "/api/v1/line-items/L-1002", film, apitest.SkipRequestValidation())
	venue := formOf(lineItem(t, priya, "L-1003"))
	venue["plannedCost"], venue["currency"] = "17000.00", "EUR"
	onApproved := priya.Do(t, http.MethodPatch, "/api/v1/line-items/L-1003", venue)
	gone := priya.Do(t, http.MethodPatch, "/api/v1/line-items/L-9999", venue)

	assertAnswer(t, saved, http.StatusOK, "")
	var v lineitem.View
	saved.Decode(t, &v)
	if v.Version != 2 || v.PlannedCost.Format() != "46,500.00 EUR" || v.PlannedCostUSD.Format() != "50,543.48 USD" {
		t.Errorf("saved = version %d, %s = %s", v.Version, v.PlannedCost.Format(), v.PlannedCostUSD.Format())
	}
	assertAnswer(t, stale, http.StatusConflict, "")
	var conflict struct{ Error struct{ Code string } }
	stale.Decode(t, &conflict)
	if conflict.Error.Code != "version_conflict" {
		t.Errorf("stale save: %s", stale.Body)
	}
	assertAnswer(t, noVersion, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.","fields":[`+
		`{"field":"name","message":"Enter a name for the line item."},`+
		`{"field":"version","message":"Reload the page and try again."}]}}`)
	assertAnswer(t, onApproved, http.StatusOK, "")
	assertStillApproved(t, priya)
	assertAnswer(t, gone, http.StatusNotFound, lineItemGoneBody)
}

// On an Approved activity a copy needs no confirmation; the deprecated restartApproval is accepted and ignored.
func TestDuplicateLineItem_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	copied := priya.Do(t, http.MethodPost, "/api/v1/line-items/L-1001/duplicate", map[string]any{})
	onApproved := priya.Do(t, http.MethodPost, "/api/v1/line-items/L-1003/duplicate", map[string]any{})
	withOldFlag := priya.Do(t, http.MethodPost, "/api/v1/line-items/L-1003/duplicate", map[string]any{"restartApproval": true})
	gone := priya.Do(t, http.MethodPost, "/api/v1/line-items/L-9999/duplicate", map[string]any{})

	assertAnswer(t, copied, http.StatusCreated, "")
	var v lineitem.View
	copied.Decode(t, &v)
	if v.ID != "L-1012" || v.Name != "Agency fee (copy)" || v.PlannedCost.Format() != "50,000.00 USD" || len(v.Missing) != 0 {
		t.Errorf("copy = %+v", v)
	}
	assertAnswer(t, onApproved, http.StatusCreated, "")
	assertAnswer(t, withOldFlag, http.StatusCreated, "")
	assertStillApproved(t, priya)
	assertAnswer(t, gone, http.StatusNotFound, lineItemGoneBody)
}

// On an Approved activity a delete needs no confirmation; the deprecated ?restartApproval is ignored.
func TestDeleteLineItem_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	deleted := priya.Do(t, http.MethodDelete, "/api/v1/line-items/L-1001", nil)
	gone := priya.Do(t, http.MethodGet, "/api/v1/line-items/L-1001", nil)
	var result struct {
		RecycleBinEntry struct {
			ID, RecordType, RecordID string
		} `json:"recycleBinEntry"`
	}
	deleted.Decode(t, &result)
	undone := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+result.RecycleBinEntry.ID+"/undo", nil)
	onApproved := priya.Do(t, http.MethodDelete, "/api/v1/line-items/L-1003", nil)
	withOldFlag := priya.Do(t, http.MethodDelete, "/api/v1/line-items/L-1004?restartApproval=true", nil)
	unknown := priya.Do(t, http.MethodDelete, "/api/v1/line-items/L-9999", nil)

	assertAnswer(t, deleted, http.StatusOK, "")
	if result.RecycleBinEntry.RecordType != "lineItem" || result.RecycleBinEntry.RecordID != "L-1001" {
		t.Errorf("entry = %+v", result.RecycleBinEntry)
	}
	assertAnswer(t, gone, http.StatusNotFound, lineItemGoneBody)
	assertAnswer(t, undone, http.StatusOK, "")
	if back := lineItem(t, priya, "L-1001"); back.Name != "Agency fee" || back.Version != 1 {
		t.Errorf("after Undo: %+v", back)
	}
	assertAnswer(t, onApproved, http.StatusOK, "")
	assertAnswer(t, withOldFlag, http.StatusOK, "")
	assertStillApproved(t, priya)
	assertAnswer(t, unknown, http.StatusNotFound, lineItemGoneBody)
}
