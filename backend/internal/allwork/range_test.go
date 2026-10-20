package allwork

import (
	"net/url"
	"slices"
	"testing"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// The tests of 016's All work extensions (FR-034 – FR-043) build two years by hand: 2026 with P-1001
// (Operational, USD) and 2027 with P-2001 (Cross-market, GBP), and read them through the in-memory Source.

// Options of the 016 fixtures.
func planType(t string) option     { return func(n *Node) { n.PlanType = t } }
func planCurrency(c string) option { return func(n *Node) { n.PlanCurrency = c } }
func inYear(y int) option          { return func(n *Node) { n.PlanYear = y } }
func attr(id string, v AttrValue) option {
	return func(n *Node) {
		if n.Attrs == nil {
			n.Attrs = map[string]AttrValue{}
		}
		n.Attrs[id] = v
	}
}
func options(ids ...string) AttrValue { return AttrValue{OptionIDs: ids} }
func shares(pairs ...string) AttrValue {
	v := AttrValue{Shares: map[string]money.Percent{}}
	for i := 0; i+1 < len(pairs); i += 2 {
		p, err := money.ParsePercent(pairs[i+1])
		if err != nil {
			panic(err)
		}
		v.OptionIDs = append(v.OptionIDs, pairs[i])
		v.Shares[pairs[i]] = p
	}
	return v
}
func yes(answer bool) AttrValue { return AttrValue{Yes: &answer} }
func phasedUSD(year string, amounts ...string) option {
	return func(n *Node) {
		for i, a := range amounts {
			n.PhasedBudgetUSD = append(n.PhasedBudgetUSD, MonthAmount{Month: year + "-" + twoDigits(i+1), USD: amountOf(a)})
		}
	}
}
func twoDigits(m int) string { return string(rune('0'+m/10)) + string(rune('0'+m%10)) }

// rangeAttributes are the Shell attributes of the 016 fixtures.
var rangeAttributes = []Attribute{
	{ID: "SA-CL", Label: "Cluster", Level: LevelCampaign, Active: true, Type: TypeSingleChoice,
		Options: []Option{{ID: "SO-EU", Label: "Europe"}, {ID: "SO-AS", Label: "Asia"}}},
	{ID: "SA-B", Label: "Brands", Level: LevelActivity, Active: true, Type: TypeMultipleChoice,
		Options: []Option{{ID: "SO-HX", Label: "Helix"}, {ID: "SO-RM", Label: "Rimula"}}},
	{ID: "SA-C", Label: "Country / OU", Level: LevelActivity, Active: true, Type: TypeMultipleChoiceSplit,
		Options: []Option{{ID: "SO-DE", Label: "Germany"}, {ID: "SO-FR", Label: "France"}}},
	{ID: "SA-F", Label: "FIT4 activity", Level: LevelActivity, Active: true, Type: TypeYesNo},
	{ID: "SA-M", Label: "Mobility channel", Level: LevelActivity, Active: true, Type: TypeSingleChoice,
		Condition: []string{"SO-MOB"}, Options: []Option{{ID: "SO-X", Label: "Fleet"}}},
	{ID: "SA-S", Label: "Spend type", Level: LevelLineItem, Active: true, Type: TypeSingleChoice,
		Options: []Option{{ID: "SO-ME", Label: "Media"}, {ID: "SO-EV", Label: "Event"}}},
	{ID: "SA-N", Label: "Expected leads", Level: LevelActivity, Active: true, Number: true, Type: "number"},
}

// twoYears is 2026 and 2027: P-1001 (2026, Operational, USD, budget 100,000.00 USD, Lubricants) with campaign
// C-1 (Europe, 60,000.00) › I-1 › A-1 (10,000.00, Helix, Germany 60 % / France 40 %, FIT4 yes; line items
// L-1 4,000.00 Media and L-2 1,000.00 Event) and A-2 (6,000.00, Helix and Rimula, FIT4 no); P-2001 (2027,
// Cross-market, GBP, budget 50,000.00 USD) with C-2 (Asia, 20,000.00) › I-2 › A-3 (8,000.00, no brand, March
// 2027) and A-4 without a planned cost, with L-3 3,000.00 Media.
func twoYears() []Node {
	b := newYear()
	b.plan("P-1001", "FY2026 Lubricants", budget("100000.00"), planType("operational"), planCurrency("USD"), inYear(2026),
		units("SO-LUB"), teams("T-1001"))
	b.campaign("P-1001", "C-1", "Helix launch", budget("60000.00"), attr("SA-CL", options("SO-EU")))
	b.initiative("C-1", "I-1", "Digital")
	b.activity("I-1", "A-1", "Launch film", planned("10000.00"), dates("2026-03-01", "2026-04-30"), owner(priya),
		activityType("AT-1001"), attr("SA-B", options("SO-HX")), attr("SA-C", shares("SO-DE", "60", "SO-FR", "40")),
		attr("SA-F", yes(true)))
	b.lineItem("A-1", "L-1", "Agency", "4000.00", attr("SA-S", options("SO-ME")))
	b.lineItem("A-1", "L-2", "Venue", "1000.00", attr("SA-S", options("SO-EV")))
	b.activity("I-1", "A-2", "Social", planned("6000.00"), dates("2026-05-01", "2026-05-31"), owner(daniel),
		activityType("AT-1002"), attr("SA-B", options("SO-HX", "SO-RM")), attr("SA-F", yes(false)))
	first := b.data().Nodes

	c := newYear()
	c.plan("P-2001", "FY2027 UK", budget("50000.00"), planType("crossMarket"), planCurrency("GBP"), inYear(2027),
		units("SO-MOB"), teams("T-1002"),
		phasedUSD("2027", "1000.00", "1000.00", "1000.00", "1000.00", "1000.00", "1000.00", "1000.00", "1000.00",
			"1000.00", "1000.00", "1000.00", "1000.00"))
	c.campaign("P-2001", "C-2", "UK events", budget("20000.00"), attr("SA-CL", options("SO-AS")))
	c.initiative("C-2", "I-2", "UK")
	c.activity("I-2", "A-3", "Dealer day", planned("8000.00"), dates("2027-03-01", "2027-03-31"), owner(priya),
		activityType("AT-1001"))
	c.activity("I-2", "A-4", "Roadshow", dates("2027-06-01", "2027-06-30"), owner(priya), activityType("AT-1002"))
	c.lineItem("A-4", "L-3", "Media buy", "3000.00", attr("SA-S", options("SO-ME")))
	return append(first, c.data().Nodes...)
}

// rangeView works out the view of q over nodes for the years from–to.
func rangeView(nodes []Node, q Query, from, to int) built {
	if q.View == "" {
		q.View = ViewGrid
	}
	if q.Strategic == "" {
		q.Strategic = StrategicInclude
	}
	if q.ShowLevels == "" {
		q.ShowLevels = ShowEverything
	}
	if q.Period == "" {
		q.Period = PeriodMonth
	}
	return build(input{
		query: q, year: from, yearFrom: from, yearTo: to, today: fixtureToday, now: fixtureNow, currency: usd,
		data: YearData{Nodes: nodes}, choices: fixtureChoices(priya.ID), catalogue: catalogue(rangeAttributes),
		attributes: rangeAttributes,
	})
}

// levelKeys returns the keys of the rows of levels.
func levelKeys(r Result, levels ...Level) []string {
	keys := []string{}
	for _, row := range r.Rows {
		if slices.Contains(levels, row.Level) {
			keys = append(keys, row.Key)
		}
	}
	return keys
}

// A range is yearFrom to yearTo, at most 5 years; year is the alias of a one-year range; a longer or reversed
// range is refused on yearTo (FR-034, D-269).
func TestQuery_YearRange_FR034(t *testing.T) {
	q, err := ParseQuery(url.Values{"yearFrom": {"2026"}, "yearTo": {"2027"}})
	alias, aliasErr := ParseQuery(url.Values{"year": {"2025"}})
	_, tooLong := ParseQuery(url.Values{"yearFrom": {"2026"}, "yearTo": {"2031"}})
	_, reversed := ParseQuery(url.Values{"yearFrom": {"2027"}, "yearTo": {"2026"}})
	five, fiveErr := ParseQuery(url.Values{"yearFrom": {"2026"}, "yearTo": {"2030"}})

	if err != nil || q.YearFrom != 2026 || q.YearTo != 2027 || q.Year != 2026 {
		t.Errorf("range = %d–%d (year %d), %v", q.YearFrom, q.YearTo, q.Year, err)
	}
	if aliasErr != nil || alias.YearFrom != 2025 || alias.YearTo != 2025 {
		t.Errorf("year alias = %d–%d, %v", alias.YearFrom, alias.YearTo, aliasErr)
	}
	if fiveErr != nil || five.YearTo != 2030 {
		t.Errorf("five years = %d–%d, %v", five.YearFrom, five.YearTo, fiveErr)
	}
	assertProblems(t, tooLong, []errs.FieldError{{Field: "yearTo", Message: "Choose at most 5 years."}})
	assertProblems(t, reversed, []errs.FieldError{{Field: "yearTo", Message: "Choose a last year on or after the first year."}})
}

// The view over a range reads every year of it: the plans of both years, each plan row with its year, the
// count over the range (FR-034).
func TestTree_PlansOfEveryYear_FR034(t *testing.T) {
	nodes := twoYears()
	src := newFakeSource(newYear())
	src.years = map[int]YearData{}
	for _, n := range nodes {
		data := src.years[n.PlanYear]
		if n.PlanYear == 0 {
			continue
		}
		data.Nodes = append(data.Nodes, n)
		src.years[n.PlanYear] = data
	}
	// the nodes below plans carry no year of their own: the service gives them their year's
	for _, n := range nodes {
		if n.PlanYear == 0 {
			year := 2026
			if n.PlanID == "P-2001" {
				year = 2027
			}
			data := src.years[year]
			data.Nodes = append(data.Nodes, n)
			src.years[year] = data
		}
	}
	s, _ := newTestService(t, src)

	r, err := s.Query(t.Context(), actor.Actor{ID: priya.ID}, Query{View: ViewGrid, YearFrom: 2026, YearTo: 2027,
		Strategic: StrategicInclude, ShowLevels: ShowEverything, Period: PeriodMonth})

	if err != nil || !slices.Equal(levelKeys(r, LevelPlan), []string{"P-1001", "P-2001"}) || r.YearFrom != 2026 || r.YearTo != 2027 || r.Year != 2026 {
		t.Fatalf("plans %v, range %d–%d, %v", levelKeys(r, LevelPlan), r.YearFrom, r.YearTo, err)
	}
	if *r.Rows[0].Year != 2026 || r.Count.Total != 4 || r.Count.Shown != 4 {
		t.Errorf("first plan's year %d, count %+v", *r.Rows[0].Year, r.Count)
	}
	for _, row := range r.Rows {
		if row.Key == "P-2001" && (row.Year == nil || *row.Year != 2027 || row.PlanType != "crossMarket") {
			t.Errorf("P-2001 row = %+v", row)
		}
	}
}

// Yearly gives one column per year of the range (FR-035).
func TestPeriods_Yearly_FR035(t *testing.T) {
	r := rangeView(twoYears(), Query{View: ViewBudget, Period: PeriodYear}, 2026, 2027).result

	if got := periodKeys(r); !slices.Equal(got, []string{"2026", "2027"}) {
		t.Fatalf("periods = %q", got)
	}
	if got := amountsText(r, r.Totals.Periods, r.Totals.Total); got != "2026 16000.00 · 2027 11000.00 · = 27000.00 USD" {
		t.Errorf("footer = %s", got)
	}
}

// periodKeys returns the keys of the period columns.
func periodKeys(r Result) []string {
	keys := []string{}
	for _, p := range r.Periods {
		keys = append(keys, p.Key)
	}
	return keys
}

// Quarterly over two years has the keys Q1-2026 … Q4-2027 and the labels "Q1 2026" … (FR-035).
func TestPeriods_QuarterKeysAcrossYears_FR035(t *testing.T) {
	r := rangeView(twoYears(), Query{View: ViewBudget, Period: PeriodQuarter}, 2026, 2027).result

	want := []string{"Q1-2026", "Q2-2026", "Q3-2026", "Q4-2026", "Q1-2027", "Q2-2027", "Q3-2027", "Q4-2027"}
	if got := periodKeys(r); !slices.Equal(got, want) || r.Periods[4].Label != "Q1 2027" {
		t.Errorf("periods = %q (label %q)", got, r.Periods[4].Label)
	}
	monthly := rangeView(twoYears(), Query{View: ViewBudget, Period: PeriodMonth}, 2026, 2027).result
	if len(monthly.Periods) != 24 || monthly.Periods[14].Label != "Mar 2027" || monthly.Periods[14].Key != "2027-03" {
		t.Errorf("monthly = %d periods, the 15th %+v", len(monthly.Periods), monthly.Periods[14])
	}
}

// Amounts outside the range are under Other years (FR-035).
func TestPeriods_OtherYearsOutsideRange_FR035(t *testing.T) {
	nodes := twoYears()
	for i := range nodes {
		if nodes[i].ID == "L-1" { // part of Agency in 2028
			nodes[i].MonthsUSD = []MonthAmount{{Month: "2026-03", USD: amountOf("3000.00")}, {Month: "2028-01", USD: amountOf("1000.00")}}
		}
	}

	r := rangeView(nodes, Query{View: ViewBudget, Period: PeriodYear}, 2026, 2027).result

	if got := periodKeys(r); !slices.Equal(got, []string{"2026", "2027", "other"}) {
		t.Errorf("periods = %q, want the two years and Other years", got)
	}
}

// Plan type and Plan currency are tested on plans (FR-037).
func TestFilters_PlanTypeAndCurrency_FR037(t *testing.T) {
	byType := rangeView(twoYears(), Query{PlanTypes: []string{"crossMarket"}}, 2026, 2027).result
	byCurrency := rangeView(twoYears(), Query{PlanCurrencies: []string{"USD"}}, 2026, 2027).result

	if got := levelKeys(byType, LevelActivity); !slices.Equal(got, []string{"A-3", "A-4"}) {
		t.Errorf("Cross-market activities = %v", got)
	}
	if got := levelKeys(byCurrency, LevelActivity); !slices.Equal(got, []string{"A-1", "A-2"}) {
		t.Errorf("USD plans' activities = %v", got)
	}
}

// The Standard columns Year, Plan type, Plan currency, Plan budget (plan currency) and Planned cost (plan
// currency) (FR-037).
func TestColumns_YearTypeCurrencyPlanAmounts_FR037(t *testing.T) {
	nodes := twoYears()
	for i := range nodes {
		switch nodes[i].ID {
		case "P-2001":
			nodes[i].BudgetPlan = &money.Money{Amount: amountOf("39500.00"), Currency: "GBP"}
		case "A-3":
			nodes[i].PlannedCostPlan = &money.Money{Amount: amountOf("6320.00"), Currency: "GBP"}
		}
	}

	r := rangeView(nodes, Query{Columns: []string{"year", "planType", "planCurrency", "planBudgetPlan", "plannedCostPlan"}}, 2026, 2027).result

	cells := map[string][]string{}
	for _, row := range r.Rows {
		if row.Key == "P-2001" || row.Key == "A-3" {
			for _, c := range row.Cells {
				cells[row.Key] = append(cells[row.Key], cell016(c))
			}
		}
	}
	if want := []string{"2027", "Cross-market", "GBP", "39,500.00 GBP", ""}; !slices.Equal(cells["P-2001"], want) {
		t.Errorf("P-2001 cells = %q, want %q", cells["P-2001"], want)
	}
	if want := []string{"2027", "Cross-market", "GBP", "", "6,320.00 GBP"}; !slices.Equal(cells["A-3"], want) {
		t.Errorf("A-3 cells = %q, want %q", cells["A-3"], want)
	}
}

// cell016 writes a cell for the 016 tests: "" for blank, "—" for empty.
func cell016(c *Cell) string {
	switch {
	case c == nil:
		return ""
	case c.Kind == KindEmpty:
		return "—"
	case c.Money != nil:
		return c.Money.Format()
	case c.Number != nil:
		return money.FromCents(*c.Number * 100).String()[:len(money.FromCents(*c.Number*100).String())-3]
	}
	return c.Text
}

// Compare with phased budget gives plans, campaigns and initiatives their phased budget per period and the
// periods where the planned cost is above it (FR-038): P-2001 is phased at 1,000.00 a month; March holds A-3's
// 8,000.00 and June the 3,000.00 of A-4's line item.
func TestCompare_PhasedBudgetAndOverPeriods_FR038(t *testing.T) {
	var year2027 []Node
	for _, n := range twoYears() {
		if n.PlanID == "P-2001" {
			year2027 = append(year2027, n)
		}
	}
	r := rangeView(year2027, Query{View: ViewBudget, Period: PeriodMonth, ComparePhased: true}, 2027, 2027).result

	for _, row := range r.Rows {
		switch row.Key {
		case "P-2001":
			if len(row.PhasedBudget) != 12 || row.PhasedBudget[0].Format() != "1,000.00 USD" || !slices.Equal(row.OverPeriods, []int{2, 5}) {
				t.Errorf("P-2001 phased %v, over %v; want 1,000.00 a month, March and June over", row.PhasedBudget, row.OverPeriods)
			}
		case "C-2":
			if len(row.PhasedBudget) != 12 || row.PhasedBudget[0] != nil || len(row.OverPeriods) != 0 {
				t.Errorf("C-2, not phased: %v, %v", row.PhasedBudget, row.OverPeriods)
			}
		case "A-3":
			if row.PhasedBudget != nil {
				t.Errorf("activities have no phased budget: %v", row.PhasedBudget)
			}
		}
	}
}

// A saved view of 010 (only year) reads as the range year–year with the new filters empty (FR-043,
// data-model §5.2).
func TestViews_OldYearReadAsRange_FR043(t *testing.T) {
	old := ViewState{View: ViewGrid, Year: 2026, Filters: Filters{Strategic: StrategicInclude, ShowLevels: ShowEverything},
		Expansion: Expansion{Mode: "default"}, Period: PeriodMonth, CalendarLayout: "timeline"}

	var problems errs.Problems
	state := checkState(&problems, old)

	if len(problems) != 0 || state.YearFrom != 2026 || state.YearTo != 2026 || state.Filters.PlanTypes == nil ||
		state.Filters.Attributes == nil || state.ComparePhased || state.Breakdown != nil {
		t.Errorf("state = %+v, problems %v", state, problems)
	}
	tooLong := old
	tooLong.YearFrom, tooLong.YearTo = 2026, 2031
	if checkState(&problems, tooLong); len(problems) != 1 {
		t.Errorf("a range of 6 years is kept: %v", problems)
	}
}

// The Calendar over a range is one timeline (FR-036): the records of every year of the range with their own
// dates — the web app draws the year and quarter bands and "Continues from / into" at the range's edges from
// the range the result states — and, as in one year, no line-item or unassigned rows.
func TestQuery_CalendarOverRange_FR036(t *testing.T) {
	r := rangeView(twoYears(), Query{View: ViewCalendar}, 2026, 2027).result

	dates := map[string][2]string{}
	for _, row := range r.Rows {
		if row.Level == LevelLineItem || row.Level == LevelUnassigned {
			t.Errorf("calendar row %s on level %s", row.Key, row.Level)
		}
		if row.Level == LevelActivity {
			dates[row.RecordID] = [2]string{row.StartDate.String(), row.EndDate.String()}
		}
	}
	if r.YearFrom != 2026 || r.YearTo != 2027 || r.View != ViewCalendar {
		t.Errorf("calendar range %d–%d, view %q", r.YearFrom, r.YearTo, r.View)
	}
	if dates["A-1"] != [2]string{"2026-03-01", "2026-04-30"} || dates["A-3"] != [2]string{"2027-03-01", "2027-03-31"} ||
		dates["A-4"] != [2]string{"2027-06-01", "2027-06-30"} || len(dates) != 4 {
		t.Errorf("activity dates over 2026–2027 = %v", dates)
	}
}
