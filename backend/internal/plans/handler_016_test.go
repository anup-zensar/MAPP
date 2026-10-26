package plans_test

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/plans"
	"mapp/internal/platform/apitest"
)

// The handler tests of 016 (plan currency, plan type, Budget by month, Years) on the demo data: P-1004 "FY2027
// Lubricants UK & Ireland" is a Cross-market plan planned in GBP (0.79 per USD), budget 600,000.00 GBP phased
// evenly, with 65,000.00 GBP planned on two activities, a campaign budget and two initiative budgets (5 amounts).

// getPlan reads the plan id as client and stops the test unless it answers 200.
func getPlan(t *testing.T, client *apitest.Client, id string) plans.PlanView {
	t.Helper()
	response := client.Do(t, http.MethodGet, "/api/v1/plans/"+id, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET plan %s = %d %s", id, response.Status, response.Body)
	}
	var plan plans.PlanView
	response.Decode(t, &plan)
	return plan
}

// The plan page answers its amounts in the plan currency (planFigures) and in the display currency beside them
// (figures), with the plan type, the phasing and the currency lock (FR-012, FR-013, FR-020, FR-032).
func TestGetPlan_PlanCurrency_Contract_FR013(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	p := getPlan(t, priya, "P-1004")

	if p.Currency != "GBP" || p.PlanType != "crossMarket" || p.PlanFigures.Budget.Format() != "600,000.00 GBP" ||
		p.PlanFigures.PlannedCost.Format() != "65,000.00 GBP" || p.Figures.Budget.Format() != "759,493.67 USD" ||
		p.Figures.PlannedCost.Format() != "82,278.48 USD" {
		t.Errorf("P-1004: %s %s, plan figures %v / %v, figures %v / %v", p.Currency, p.PlanType, p.PlanFigures.Budget,
			p.PlanFigures.PlannedCost, p.Figures.Budget, p.Figures.PlannedCost)
	}
	if p.BudgetPhasing.Method != "evenly" || len(p.BudgetPhasing.Monthly) != 12 || p.BudgetPhasing.Monthly[0].Format() != "50,000.00 GBP" ||
		p.BudgetPhasing.Problem != nil {
		t.Errorf("phasing = %+v", p.BudgetPhasing)
	}
	want := "The plan currency can't change once campaigns, initiatives or activities have amounts in it (5 amounts)."
	if p.CurrencyLock == nil || *p.CurrencyLock != want {
		t.Errorf("currency lock = %v, want %q", p.CurrencyLock, want)
	}
	if usd := getPlan(t, server.SignIn(t, meisEmail), "P-1002"); usd.Currency != "USD" || usd.PlanType != "operational" || usd.CurrencyLock == nil {
		t.Errorf("P-1002: %s %s, lock %v", usd.Currency, usd.PlanType, usd.CurrencyLock)
	}
}

// A new plan gets the chosen plan currency and type; its budget must be in that currency, and an unknown type
// is refused (FR-011, FR-032).
func TestCreatePlan_CurrencyAndType_Contract_FR011_FR032(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	body := newPlanBody()
	body["currency"], body["planType"] = "GBP", "strategic"
	body["budget"] = map[string]any{"amount": "100000.00", "currency": "GBP"}
	wrong := newPlanBody()
	wrong["name"], wrong["currency"] = "FY2026 Lubricants Wrong", "GBP"
	wrong["budget"] = map[string]any{"amount": "100000.00", "currency": "USD"}

	created := priya.Do(t, http.MethodPost, "/api/v1/plans", body)
	refused := priya.Do(t, http.MethodPost, "/api/v1/plans", wrong)

	if created.Status != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Status, created.Body)
	}
	var p plans.PlanView
	created.Decode(t, &p)
	if p.Currency != "GBP" || p.PlanType != "strategic" || p.PlanFigures.Budget.Format() != "100,000.00 GBP" ||
		p.Figures.Budget.Format() != "126,582.28 USD" || p.CurrencyLock != nil {
		t.Errorf("created = %s %s, %v / %v, lock %v", p.Currency, p.PlanType, p.PlanFigures.Budget, p.Figures.Budget, p.CurrencyLock)
	}
	assertAnswer(t, refused, http.StatusBadRequest, validationBody(`{"field":"budget","message":"Enter the amount in GBP, the plan's currency."}`))
}

// The Plans list takes a range of years and plan types; its rows carry the plan type, the plan currency and
// the amounts in it beside the display amounts (FR-014, FR-033).
func TestListPlans_YearRangeAndType_Contract_FR033(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	crossMarket := listOf(t, priya.Do(t, http.MethodGet, "/api/v1/plans?yearFrom=2027&yearTo=2027&type=crossMarket", nil))
	both := listOf(t, priya.Do(t, http.MethodGet, "/api/v1/plans?yearFrom=2026&yearTo=2027&type=crossMarket&type=strategic", nil))
	reversed := priya.Do(t, http.MethodGet, "/api/v1/plans?yearFrom=2027&yearTo=2026", nil)

	if ids := listIDs(crossMarket); !reflect.DeepEqual(ids, []string{"P-1004"}) {
		t.Fatalf("Cross-market 2027 = %q, want P-1004", ids)
	}
	row := crossMarket.Items[0]
	if row.PlanType != "crossMarket" || row.Currency != "GBP" || row.PlanBudget.Format() != "600,000.00 GBP" ||
		row.PlanPlannedCost.Format() != "65,000.00 GBP" || row.Figures.Budget.Format() != "759,493.67 USD" ||
		crossMarket.Summary.Budget.Format() != "759,493.67 USD" {
		t.Errorf("row = %+v, summary %+v", row, crossMarket.Summary)
	}
	if ids := listIDs(both); !reflect.DeepEqual(ids, []string{"P-1004"}) { // P-1003 is not Priya's team's plan (014)
		t.Errorf("Cross-market or Strategic 2026–2027 = %q, want P-1004", ids)
	}
	if reversed.Status != http.StatusBadRequest || !strings.Contains(string(reversed.Body), `"field":"yearTo"`) {
		t.Errorf("reversed range = %d %s, want 400 on yearTo", reversed.Status, reversed.Body)
	}
}

// Budget by month is saved with PUT …/budget-phasing whatever the plan's status: a Custom phasing that does not
// add up is saved and becomes its problem; a month outside the plan's year is refused; an old version
// conflicts (FR-020 – FR-023).
func TestSetPlanPhasing_Contract_FR020(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	before := getPlan(t, priya, "P-1004")
	months := []map[string]any{}
	for m := 1; m <= 12; m++ {
		amount := "50000.00"
		if m == 3 {
			amount = "60000.00"
		}
		months = append(months, map[string]any{"month": fmt.Sprintf("2027-%02d", m), "amount": amount})
	}
	body := map[string]any{"method": "custom", "months": months, "version": before.Version}
	outside := map[string]any{"method": "custom", "months": []map[string]any{{"month": "2026-03", "amount": "10.00"}},
		"version": before.Version}

	refused := priya.Do(t, http.MethodPut, "/api/v1/plans/P-1004/budget-phasing", outside)
	saved := priya.Do(t, http.MethodPut, "/api/v1/plans/P-1004/budget-phasing", body)
	stale := priya.Do(t, http.MethodPut, "/api/v1/plans/P-1004/budget-phasing", body)
	gone := priya.Do(t, http.MethodPut, "/api/v1/plans/P-9999/budget-phasing", body)

	assertAnswer(t, refused, http.StatusBadRequest, validationBody(`{"field":"months","message":"Months must be in 2027."}`))
	if saved.Status != http.StatusOK {
		t.Fatalf("save = %d %s", saved.Status, saved.Body)
	}
	var p plans.PlanView
	saved.Decode(t, &p)
	if p.BudgetPhasing.Method != "custom" || p.BudgetPhasing.Total.Format() != "610,000.00 GBP" || p.BudgetPhasing.Problem == nil ||
		*p.BudgetPhasing.Problem != "Adds up to 610,000.00 GBP, not the budget 600,000.00 GBP" || p.Version != before.Version+1 {
		t.Errorf("saved phasing = %+v (version %d)", p.BudgetPhasing, p.Version)
	}
	if stale.Status != http.StatusConflict || gone.Status != http.StatusNotFound {
		t.Errorf("stale = %d, unknown plan = %d; want 409 and 404", stale.Status, gone.Status)
	}
}
