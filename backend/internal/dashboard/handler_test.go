package dashboard_test

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/dashboard"
	"mapp/internal/financials"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

const (
	carlos    = "carlos.mendes@shell.com"
	priya     = "priya.raman@shell.com"
	daniel    = "daniel.okoro@shell.com"
	lea       = "lea.santos@shell.com"
	executive = "/api/v1/dashboards/executive"
	regional  = "/api/v1/dashboards/regional"
	pdfPrefix = "%PDF-"
)

func get(t *testing.T, c *apitest.Client, path string, status int) *apitest.Response {
	t.Helper()
	var opts []apitest.RequestOption
	if status == http.StatusBadRequest {
		opts = append(opts, apitest.SkipRequestValidation())
	}
	r := c.Do(t, http.MethodGet, path, nil, opts...)
	if r.Status != status {
		t.Fatalf("GET %s = %d %s, want %d", path, r.Status, r.Body, status)
	}
	return r
}

func view(t *testing.T, c *apitest.Client, path string) dashboard.View {
	t.Helper()
	var v dashboard.View
	get(t, c, path, http.StatusOK).Decode(t, &v)
	return v
}

func amount(m *money.Money) string {
	if m == nil {
		return "—"
	}
	return m.Format()
}

func texts(values ...any) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = fmt.Sprint(v)
	}
	return out
}

func same(t *testing.T, what string, want, got []string) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", what, diff)
	}
}

// Carlos sees every market: both 2026 plans, equal to the plan pages and Financials (US1 1, 3, 8–11, SC-001).
func TestExecutive_EqualsTheRecordPages_SC001(t *testing.T) {
	t.Parallel()
	c := apitest.New(t).SignIn(t, carlos)

	v := view(t, c, executive+"?year=2026")

	var overview financials.Overview
	get(t, c, "/api/v1/financials/overview?scope=everything", http.StatusOK).Decode(t, &overview)
	same(t, "totals", texts(2, "2,000,000.00 USD", overview.Figures.PlannedCost.Format(), overview.Figures.Committed.Format(),
		33, "1,969,347.83 USD", 2), texts(v.PlanCount, amount(v.Totals.Budget), amount(v.Totals.PlannedCost),
		amount(v.Totals.Committed), *v.Totals.BudgetUsedPercent, amount(v.Totals.UnspentBudget), v.Finance.PlanCount))
	same(t, "panels", texts("Lubricants Europe", "1,200,000.00 USD", 2, 5, 1, true, true, true, 3, 12),
		texts(v.Markets[0].Team.Name, amount(v.Markets[0].Figures.Budget), v.Approvals.Waiting, len(v.Completeness),
			len(v.GlobalBudgets), v.CanExport, v.FinanceAsOf != nil, v.PurchaseRequests.Available, len(v.LeftToPlan),
			len(v.Months.Items)))
}

// Priya sees Lubricants Europe only: nothing of Mobility China counts or is named (US1 2, FR-002, SC-002).
func TestExecutive_OnlyThePlansInScope_FR002(t *testing.T) {
	t.Parallel()
	c := apitest.New(t).SignIn(t, priya)

	r := get(t, c, executive+"?year=2026", http.StatusOK)

	var v dashboard.View
	r.Decode(t, &v)
	same(t, "Priya's dashboard", texts(1, "P-1001", "1,200,000.00 USD", 1, 1, true),
		texts(v.PlanCount, v.Plans[0].Plan.ID, amount(v.Totals.Budget), len(v.Markets), v.Approvals.Waiting, v.GlobalBudgets == nil))
	for _, other := range []string{"P-1002", "FY2026 Mobility China", "Retail growth"} {
		if strings.Contains(string(r.Body), other) {
			t.Errorf("the dashboard names %q, outside Priya's scope", other)
		}
	}
}

// Lea may view plans but neither see financial data nor export: those figures are "—" (US1 scenario 4, FR-003).
func TestExecutive_FinanceHidden_FR003(t *testing.T) {
	t.Parallel()
	v := view(t, apitest.New(t).SignIn(t, lea), executive+"?year=2026")

	same(t, "Lea's dashboard", texts(false, "—", "—", true, true, false, "650,000.00 USD"),
		texts(v.Finance.Visible, amount(v.Totals.Committed), amount(v.Totals.Actual), v.Totals.BudgetSpentPercent == nil,
			v.Months.Items[2].Actual == nil, v.CanExport, amount(v.Totals.PlannedCost)))
}

// Currency, plan type and year filters; 2027 holds the demo purchase request (US1 9, 12, 13, FR-006, FR-017).
func TestExecutive_Filters_FR006(t *testing.T) {
	t.Parallel()
	c := apitest.New(t).SignIn(t, carlos)

	euros := view(t, c, executive+"?year=2026&currency=EUR")
	strategic := view(t, c, executive+"?year=2026&planType=strategic")
	empty := view(t, c, executive+"?year=2030")
	purchases := view(t, c, executive+"?year=2027").PurchaseRequests

	same(t, "filters", texts("1,840,000.00 EUR", "EUR", 0, 0, "—", "[2030 2027 2026]", 1, 1, "25,000.00 USD"),
		texts(amount(euros.Totals.Budget), euros.Filters.Currency, strategic.PlanCount, empty.PlanCount,
			amount(empty.Totals.PlannedCost), empty.Choices.Years, purchases.Count, purchases.ByStatus[0].Count,
			purchases.Amount.Format()))
}

// Wrong filters are refused all at once, and a currency that is not in Settings too (FR-006).
func TestExecutive_Validation_FR006(t *testing.T) {
	t.Parallel()
	c := apitest.New(t).SignIn(t, carlos)

	wrong := get(t, c, executive+"?year=1999&planType=other&currency=eur", http.StatusBadRequest)

	same(t, "refusals", texts(true, true, true), texts(strings.Contains(string(wrong.Body), `"year"`),
		strings.Contains(string(wrong.Body), `"planType"`), strings.Contains(string(wrong.Body), `"currency"`)))
}

func TestUnknownFilters_FallBack_038_FR012_H1(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	c, d := server.SignIn(t, carlos), server.SignIn(t, daniel)

	currency := view(t, c, executive+"?year=2026&currency=XYZ")
	team := view(t, d, regional+"?year=2026&teamId=T-9999")
	own := view(t, d, regional+"?year=2026")

	same(t, "fallback", texts("USD", *own.Filters.TeamID, own.Team.Name),
		texts(currency.Filters.Currency, *team.Filters.TeamID, team.Team.Name))
	get(t, c, executive+"/export?format=pdf&year=2026&currency=XYZ", http.StatusOK)
	get(t, d, regional+"/export?format=pptx&year=2026&teamId=T-9999", http.StatusOK)
}

func TestMonths_NegativeBalanceCount_038_FR024_L2(t *testing.T) {
	t.Parallel()
	v := view(t, apitest.New(t).SignIn(t, carlos), executive+"?year=2026")

	same(t, "negative balance count", texts(1), texts(v.Months.NegativeBalanceCount))
}

// Own market by default, a plan's campaigns, a market row's equal, scope kept.
func TestRegional_FR020_FR022(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	d, c := server.SignIn(t, daniel), server.SignIn(t, carlos)

	own := view(t, d, regional+"?year=2026&planId=P-1001")
	row := view(t, c, executive+"?year=2026").Markets[1]
	mobility := view(t, c, regional+"?year=2026&teamId=T-1002")
	outside := view(t, server.SignIn(t, priya), regional+"?year=2026&teamId=T-1002")

	same(t, "regional", texts("Lubricants Europe", 1, 0, 2, "Helix launch", amount(row.Figures.PlannedCost), 0),
		texts(own.Team.Name, own.PlanCount, len(own.Markets), len(own.PlanCampaigns.Items),
			own.PlanCampaigns.Items[0].Campaign.Name, amount(mobility.Totals.PlannedCost), outside.PlanCount))
}

// A plan as .pptx and .pdf; refused without data.export, not found out of scope (US2, FR-023, FR-025).
func TestPlanExport_FR023_FR025(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	p := server.SignIn(t, priya)

	pptx := get(t, p, "/api/v1/plans/P-1001/export?format=pptx", http.StatusOK)
	pdf := get(t, p, "/api/v1/plans/P-1001/export?format=pdf&currency=EUR", http.StatusOK)

	disposition := pptx.Header.Get("Content-Disposition")
	same(t, "files", texts(true, true, true, true, true), texts(bytes.HasPrefix(pptx.Body, []byte("PK\x03\x04")),
		strings.Contains(disposition, "FY2026%20Lubricants%20Europe"), strings.HasSuffix(disposition, ".pptx"),
		bytes.HasPrefix(pdf.Body, []byte(pdfPrefix)), bytes.Contains(pdf.Body, []byte("1,104,000.00 EUR"))))
	get(t, p, "/api/v1/plans/P-1002/export?format=pdf", http.StatusNotFound)
	get(t, server.SignIn(t, lea), "/api/v1/plans/P-1001/export?format=pdf", http.StatusForbidden)
	get(t, p, "/api/v1/plans/P-1001/export?format=docx", http.StatusBadRequest)
}

// Both dashboards download in both formats for people who may export, with the filters in the name (US4, FR-024).
func TestDashboardExport_FR024(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	c := server.SignIn(t, carlos)

	exec := get(t, c, executive+"/export?format=pptx&year=2026", http.StatusOK)
	region := get(t, c, regional+"/export?format=pdf&year=2026&teamId=T-1001", http.StatusOK)

	same(t, "files", texts(true, true, true, true), texts(bytes.HasPrefix(exec.Body, []byte("PK")),
		strings.Contains(exec.Header.Get("Content-Disposition"), "Executive%20dashboard%20%E2%80%93%202026"),
		bytes.HasPrefix(region.Body, []byte(pdfPrefix)), strings.Contains(region.Header.Get("Content-Disposition"), "Lubricants%20Europe")))
	get(t, server.SignIn(t, lea), executive+"/export?format=pdf", http.StatusForbidden)
}
