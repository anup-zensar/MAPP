package allwork_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"northstar/internal/allwork"
	"northstar/internal/platform/apitest"
)

// All work's financial columns and filter on the demo GSAP data of feature 015 (015 FR-023, FR-024, US5).

// The default columns end with Committed and Invoiced; FY2026 Lubricants Europe has committed 35,652.17, as
// its Financials card; the totals row adds up the matching activities (015 US5 scenario 1, SC-002).
func TestGetAllWork_FinancialColumns_FR023(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/all-work?year=2026&column=committed&column=invoiced&column=paid&column=accrued"+
		"&column=notYetCommitted", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/all-work = %d %s", response.Status, response.Body)
	}
	var r allwork.Result
	response.Decode(t, &r)
	figures := map[string][]string{}
	for _, row := range r.Rows {
		var texts []string
		for _, c := range row.Cells {
			if c == nil || c.Money == nil {
				texts = append(texts, "")
				continue
			}
			texts = append(texts, c.Money.Amount.Format())
		}
		figures[row.Name] = texts
	}
	want := map[string][]string{
		"FY2026 Lubricants Europe": {"35,652.17", "33,152.17", "30,652.17", "2,500.00", "424,347.83"},
		"Helix launch":             {"35,652.17", "33,152.17", "30,652.17", "2,500.00", "239,347.83"},
		"Social always-on":         {"25,000.00", "22,500.00", "20,000.00", "2,500.00", "55,000.00"},
		"Stand build":              {"10,652.17", "10,652.17", "10,652.17", "0.00", "-652.17"},
		"Station signage":          {"0.00", "0.00", "0.00", "0.00", "1,000.00"},
	}
	for name, cells := range want {
		if got := figures[name]; got == nil || !slices.Equal(got[:5], cells) {
			t.Errorf("%s = %q, want %q", name, got, cells)
		}
	}
	if total := r.Totals.Cells[0]; total == nil || total.Money.Amount.Format() != "35,652.17" {
		t.Errorf("totals committed = %+v", total)
	}

	defaults := priya.Do(t, http.MethodGet, "/api/all-work", nil)
	var d allwork.Result
	defaults.Decode(t, &d)
	keys := []string{}
	for _, c := range d.Columns {
		keys = append(keys, c.Key)
	}
	if !slices.Equal(keys[len(keys)-3:], []string{"unassigned", "committed", "invoiced"}) {
		t.Errorf("default columns = %v", keys)
	}
}

// The filter "Committed above planned cost" keeps Billboard winter wave only; the export of that view holds
// the financial columns and says when the GSAP data was loaded (015 FR-024, US5 scenarios 2 and 3).
func TestGetAllWork_FinancialsFilterAndExport_FR024(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/all-work?year=2026&finance=committedAbovePlanned", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/all-work = %d %s", response.Status, response.Body)
	}
	var r allwork.Result
	response.Decode(t, &r)
	var activities []string
	for _, row := range r.Rows {
		if row.Level == allwork.LevelActivity && row.Role == allwork.RoleMatch {
			activities = append(activities, row.Name)
		}
	}
	if !slices.Equal(activities, []string{"Billboard winter wave"}) {
		t.Errorf("matching activities = %v", activities)
	}

	export := priya.Do(t, http.MethodGet, "/api/all-work/export?format=csv&finance=committedAbovePlanned", nil)
	body := string(export.Body)
	if export.Status != http.StatusOK || !strings.Contains(body, `Financial figures,"GSAP data as of 12 Mar 2026, 09:00 UTC"`) ||
		!strings.Contains(body, "Financials: Committed above planned cost") || !strings.Contains(body, "Committed (USD)") {
		t.Errorf("export = %d %s", export.Status, body)
	}

	refused := priya.Do(t, http.MethodGet, "/api/all-work?finance=overBudget", nil, apitest.SkipRequestValidation())
	if refused.Status != http.StatusBadRequest || !strings.Contains(string(refused.Body), `"field":"finance"`) {
		t.Errorf("an unknown value = %d %s", refused.Status, refused.Body)
	}
}
