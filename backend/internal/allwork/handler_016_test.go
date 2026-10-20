package allwork_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"northstar/internal/allwork"
	"northstar/internal/platform/apitest"
)

// The handler tests of 016's All work (FR-034 – FR-043) on the demo data: 2026 holds P-1001 and P-1002 (USD),
// 2027 holds P-1003 (USD, Strategic) and P-1004 "FY2027 Lubricants UK & Ireland" (Cross-market, planned in GBP
// at 0.79 per USD, budget 600,000.00 GBP phased evenly, 65,000.00 GBP planned = 82,278.48 USD).

// allWorkOf reads GET path and stops the test unless it answers 200.
func allWorkOf(t *testing.T, client *apitest.Client, path string) allwork.Result {
	t.Helper()
	response := client.Do(t, http.MethodGet, path, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s", path, response.Status, response.Body)
	}
	var r allwork.Result
	response.Decode(t, &r)
	return r
}

// A range of years reads the plans of every year, each plan row with its year and type; Yearly gives one
// period per year; Compare with phased budget gives the plans their phased budget per period (FR-034, FR-035,
// FR-037, FR-038).
func TestGetAllWork_RangeYearlyCompare_Contract_FR034(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	r := allWorkOf(t, priya, "/api/all-work?yearFrom=2026&yearTo=2027&view=budget&period=year&comparePhased=true")
	gbp := allWorkOf(t, priya, "/api/all-work?yearFrom=2026&yearTo=2027&planCurrency=GBP&planType=crossMarket")
	tooLong := priya.Do(t, http.MethodGet, "/api/all-work?yearFrom=2026&yearTo=2031", nil)

	years := map[string]int{}
	var uk allwork.Row
	for _, row := range r.Rows {
		if row.Level == allwork.LevelPlan && row.Year != nil {
			years[row.RecordID] = *row.Year
		}
		if row.RecordID == "P-1004" && row.Level == allwork.LevelPlan {
			uk = row
		}
	}
	if r.YearFrom != 2026 || r.YearTo != 2027 || r.Year != 2026 || years["P-1001"] != 2026 || years["P-1004"] != 2027 || len(years) != 4 {
		t.Errorf("range %d–%d (year %d), plan years %v", r.YearFrom, r.YearTo, r.Year, years)
	}
	keys := []string{}
	for _, p := range r.Periods {
		keys = append(keys, p.Key)
	}
	if !slices.Equal(keys, []string{"2026", "2027"}) {
		t.Errorf("periods = %q, want 2026 and 2027", keys)
	}
	if uk.PlanType != "crossMarket" || len(uk.PhasedBudget) != 2 || uk.PhasedBudget[0] != nil || uk.PhasedBudget[1] == nil ||
		uk.PhasedBudget[1].Format() != "759,493.67 USD" {
		t.Errorf("P-1004 row: type %q, phased %v", uk.PlanType, uk.PhasedBudget)
	}
	var gbpPlans []string
	for _, row := range gbp.Rows {
		if row.Level == allwork.LevelPlan {
			gbpPlans = append(gbpPlans, row.RecordID)
		}
	}
	if !slices.Equal(gbpPlans, []string{"P-1004"}) || gbp.Summary.PlannedCost == nil || gbp.Summary.PlannedCost.Format() != "82,278.48 USD" {
		t.Errorf("GBP Cross-market plans %v, planned %v", gbpPlans, gbp.Summary.PlannedCost)
	}
	assertAnswer(t, tooLong, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"yearTo","message":"Choose at most 5 years."}]}`)
}

// The Breakdown of a filter set: its groups, its totals equal to the Planned cost card, the plan's own currency
// for one plan, and the refusal without a grouping (FR-040 – FR-042, SC-007).
func TestGetBreakdown_Contract_FR040(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	byYear := priya.Do(t, http.MethodGet, "/api/all-work/breakdown?yearFrom=2026&yearTo=2027&by=year&thenBy=planCurrency", nil)
	card := allWorkOf(t, priya, "/api/all-work?yearFrom=2026&yearTo=2027")
	inGBP := priya.Do(t, http.MethodGet, "/api/all-work/breakdown?year=2027&plan=P-1004&by=planType&amountsIn=plan", nil)
	noGrouping := priya.Do(t, http.MethodGet, "/api/all-work/breakdown?yearFrom=2026", nil)
	several := priya.Do(t, http.MethodGet, "/api/all-work/breakdown?yearFrom=2026&yearTo=2027&by=plan&amountsIn=plan", nil)

	if byYear.Status != http.StatusOK {
		t.Fatalf("breakdown by year = %d %s", byYear.Status, byYear.Body)
	}
	var b allwork.Breakdown
	byYear.Decode(t, &b)
	var labels []string
	for _, g := range b.Groups {
		labels = append(labels, g.Label)
	}
	if !slices.Equal(labels, []string{"2026", "2027"}) || b.Currency != "USD" || card.Summary.PlannedCost == nil ||
		b.Total.PlannedCost != *card.Summary.PlannedCost || b.Total.Budget == nil || len(b.Groups) != 2 ||
		len(b.Groups[1].Children) == 0 || b.Groups[1].Children[0].Label != "GBP" {
		t.Errorf("groups %q, total %v (card %v), budget %v, 2027 by currency %+v", labels, b.Total.PlannedCost, card.Summary.PlannedCost,
			b.Total.Budget, b.Groups[1].Children)
	}
	var plan allwork.Breakdown
	inGBP.Decode(t, &plan)
	if inGBP.Status != http.StatusOK || plan.Currency != "GBP" || len(plan.Groups) != 1 || plan.Groups[0].Label != "Cross-market" ||
		plan.Groups[0].PlannedCost.Format() != "65,000.00 GBP" || plan.Groups[0].Budget == nil || plan.Groups[0].Budget.Format() != "600,000.00 GBP" {
		t.Errorf("P-1004 in GBP = %d %s", inGBP.Status, inGBP.Body)
	}
	assertAnswer(t, noGrouping, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"by","message":"Choose what to break down by."}]}`)
	assertAnswer(t, several, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"amountsIn","message":"Choose one plan to show amounts in its currency."}]}`)
}

// The export of the Breakdown view holds its group table, and "About this export" states the range and the
// rates used (FR-009, FR-043).
func TestExport_Breakdown_FR043(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	csv := priya.Do(t, http.MethodGet, "/api/all-work/export?format=csv&view=breakdown&by=plan&yearFrom=2026&yearTo=2027", nil)
	noGrouping := priya.Do(t, http.MethodGet, "/api/all-work/export?format=csv&view=breakdown", nil)

	body := string(csv.Body)
	if csv.Status != http.StatusOK ||
		!strings.Contains(csv.Header.Get("Content-Disposition"), `filename="NorthStar All work - Breakdown - 2026-2027 - 12 Mar 2026.csv"`) ||
		!strings.Contains(body, "Years,2026 – 2027\n") || !strings.Contains(body, "Break down by,Plan\n") ||
		!strings.Contains(body, "Rates used,") || !strings.Contains(body, "FY2027 Lubricants UK & Ireland,") ||
		!strings.Contains(body, "\nTotal,") {
		t.Errorf("csv = %d %v\n%s", csv.Status, csv.Header, body)
	}
	if noGrouping.Status != http.StatusBadRequest || !strings.Contains(string(noGrouping.Body), `"field":"by"`) {
		t.Errorf("export without a grouping = %d %s, want 400 on by", noGrouping.Status, noGrouping.Body)
	}
}

// The filter bar's choices over a range of years (FR-034, FR-037, FR-039): the range, the plans of every year
// with their year, type and currency, the plan types, the plan currencies of the range's plans, and the
// filterable Shell attributes level by level — a yes/no attribute offers Yes and No. A range above 5 years is
// refused.
func TestGetOptions_RangeTypesCurrenciesAttributes_Contract_FR037_FR039(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/all-work/options?yearFrom=2026&yearTo=2027", nil)
	oneYear := priya.Do(t, http.MethodGet, "/api/all-work/options?year=2026", nil)
	tooLong := priya.Do(t, http.MethodGet, "/api/all-work/options?yearFrom=2026&yearTo=2031", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("GET options = %d %s", response.Status, response.Body)
	}
	var o, first allwork.Options
	response.Decode(t, &o)
	oneYear.Decode(t, &first)
	plans := map[string]allwork.PlanOption{}
	for _, p := range o.Plans {
		plans[p.ID] = p
	}
	uk := plans["P-1004"]
	if o.YearFrom != 2026 || o.YearTo != 2027 || o.Year != 2026 || len(plans) != 4 ||
		uk.Year != 2027 || uk.PlanType != "crossMarket" || uk.Currency != "GBP" || plans["P-1001"].Year != 2026 {
		t.Errorf("range %d–%d (year %d), plans %+v", o.YearFrom, o.YearTo, o.Year, o.Plans)
	}
	if !slices.Equal(o.PlanTypes, []string{"operational", "strategic", "crossMarket"}) || !slices.Equal(o.PlanCurrencies, []string{"GBP", "USD"}) {
		t.Errorf("plan types %q, plan currencies %q", o.PlanTypes, o.PlanCurrencies)
	}
	if first.YearFrom != 2026 || first.YearTo != 2026 || !slices.Equal(first.PlanCurrencies, []string{"USD"}) || len(first.Plans) != 2 {
		t.Errorf("2026 alone: range %d–%d, currencies %q, %d plans", first.YearFrom, first.YearTo, first.PlanCurrencies, len(first.Plans))
	}
	var yesNo *allwork.FilterableAttribute
	levels := []allwork.Level{}
	for i, a := range o.Attributes {
		if len(levels) == 0 || levels[len(levels)-1] != a.Level {
			levels = append(levels, a.Level)
		}
		if a.Label == "Insight session completed" {
			yesNo = &o.Attributes[i]
		}
		if len(a.Values) == 0 {
			t.Errorf("attribute %s %q offers no values", a.ID, a.Label)
		}
	}
	if len(o.Attributes) == 0 || levels[0] != allwork.LevelPlan || slices.Index(levels, allwork.LevelCampaign) < 0 {
		t.Errorf("attribute levels %q (%d attributes)", levels, len(o.Attributes))
	}
	if yesNo == nil || len(yesNo.Values) != 2 || yesNo.Values[0].Label != "Yes" || yesNo.Values[1].Label != "No" {
		t.Errorf("yes/no attribute = %+v", yesNo)
	}
	assertAnswer(t, tooLong, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"yearTo","message":"Choose at most 5 years."}]}`)
}
