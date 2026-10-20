package allwork

import (
	"slices"
	"testing"
	"time"

	"northstar/internal/platform/actor"
)

// The financial figures of feature 015 in All work (015 FR-023, FR-024).

// financeYear is the fixture year with the GSAP figures of 015's demo data on Venue hire, Speaker fees and
// Stand build (015 data model §6.1).
func financeYear() *yearBuilder {
	b := seedYear()
	b.node("L-1003").Finance = &FinanceFacts{Committed: amountOf("20000.00"), Invoiced: amountOf("20000.00"),
		Paid: amountOf("20000.00"), POCount: 1, InvoiceCount: 1}
	b.node("L-1004").Finance = &FinanceFacts{Committed: amountOf("5000.00"), Invoiced: amountOf("2500.00"),
		Accrued: amountOf("2500.00"), POCount: 1, InvoiceCount: 1}
	b.node("L-1005").Finance = &FinanceFacts{Committed: amountOf("10652.17"), Invoiced: amountOf("10652.17"),
		Paid: amountOf("10652.17"), POCount: 1, InvoiceCount: 1}
	return b
}

// financeColumnKeys are the five financial columns, in the chooser's order.
var financeColumnKeys = []string{"committed", "invoiced", "paid", "accrued", "notYetCommitted"}

// cellsOf returns the cells of each row by its key, as text: the amount, "—" or "" for blank, with "!" for a
// red cell.
func cellsOf(r Result) map[string][]string {
	rows := map[string][]string{}
	for _, row := range r.Rows {
		rows[row.Key] = cellTexts(row.Cells)
	}
	return rows
}

// cellTexts writes cells as text.
func cellTexts(cells []*Cell) []string {
	texts := make([]string, 0, len(cells))
	for _, c := range cells {
		texts = append(texts, figureText(c))
	}
	return texts
}

// figureText writes one money cell as text.
func figureText(c *Cell) string {
	switch {
	case c == nil:
		return ""
	case c.Kind == KindEmpty:
		return "—"
	case c.Money == nil:
		return "?"
	case c.Tone == ToneProblem:
		return c.Money.Amount.Format() + "!"
	}
	return c.Money.Amount.Format()
}

// Every level shows its figures: a line item its own, an activity the sum of its line items, a container the
// sum of its activities; not yet committed is planned cost − committed; committed above planned cost and a
// negative not yet committed are red; unassigned rows are blank; the totals row adds up the matching
// activities (015 FR-023, US5 scenario 1).
func TestFinance_ColumnsAtEveryLevel_FR023(t *testing.T) {
	r := view(financeYear(), Query{Columns: financeColumnKeys})
	rows := cellsOf(r)
	want := map[string][]string{
		"L-1003":            {"20,000.00", "20,000.00", "20,000.00", "0.00", "0.00"},
		"L-1004":            {"5,000.00", "2,500.00", "0.00", "2,500.00", "0.00"},
		"L-1005":            {"10,652.17!", "10,652.17", "10,652.17", "0.00", "-652.17!"},
		"L-1001":            {"0.00", "0.00", "0.00", "0.00", "50,000.00"},
		"A-1002":            {"25,000.00", "22,500.00", "20,000.00", "2,500.00", "55,000.00"},
		"A-1003":            {"10,652.17", "10,652.17", "10,652.17", "0.00", "34,347.83"},
		"A-1009":            {"0.00", "0.00", "0.00", "0.00", "—"}, // no planned cost
		"I-1001":            {"25,000.00", "22,500.00", "20,000.00", "2,500.00", "175,000.00"},
		"I-1002":            {"10,652.17", "10,652.17", "10,652.17", "0.00", "64,347.83"},
		"I-1007":            {"0.00", "0.00", "0.00", "0.00", "—"}, // no activities
		"C-1001":            {"35,652.17", "33,152.17", "30,652.17", "2,500.00", "239,347.83"},
		"P-1001":            {"35,652.17", "33,152.17", "30,652.17", "2,500.00", "424,347.83"},
		"P-1002":            {"0.00", "0.00", "0.00", "0.00", "190,000.00"},
		"A-1001:unassigned": {"", "", "", "", ""},
	}
	for key, cells := range want {
		if !slices.Equal(rows[key], cells) {
			t.Errorf("%s = %q, want %q", key, rows[key], cells)
		}
	}
	if got := cellTexts(r.Totals.Cells); !slices.Equal(got, []string{"35,652.17", "33,152.17", "30,652.17", "2,500.00", "614,347.83"}) {
		t.Errorf("totals = %q", got)
	}
}

// Containers add up their matching activities and keep the unfiltered sum when filters hide some, as planned
// cost does (015 FR-023, 010 FR-013); the columns sort (FR-015).
func TestFinance_FilteredRollUpAndSort_FR023(t *testing.T) {
	filtered := view(financeYear(), Query{Columns: []string{"committed"}, Owners: []string{priya.ID},
		Finance: []string{FinanceCommittedAbovePlanned}})
	for _, row := range filtered.Rows {
		if row.Key == "C-1001" && (figureText(row.Cells[0]) != "10,652.17" || figureText(row.Cells[0].Unfiltered) != "35,652.17") {
			t.Errorf("Helix launch = %s, unfiltered %s", figureText(row.Cells[0]), figureText(row.Cells[0].Unfiltered))
		}
	}

	sorted := view(financeYear(), Query{Columns: []string{"committed"}, Sort: &Sort{Column: "committed", Direction: SortDescending}})
	var activities []string
	for _, row := range sorted.Rows {
		if row.Level == LevelActivity && row.PlanID == "P-1001" {
			activities = append(activities, row.Key)
		}
	}
	if activities[0] != "A-1002" || activities[1] != "A-1001" || activities[2] != "A-1003" {
		t.Errorf("activities by committed, descending within their initiatives = %v", activities)
	}
}

// The Financials filter is tested on activities, its values combined with OR; the records above a matching
// activity hold by it, as for every activity filter (015 FR-024, US5 scenario 2, 010 D-113).
func TestFinance_Filter_FR024(t *testing.T) {
	cases := []struct {
		values []string
		want   []string
	}{
		{[]string{FinanceCommittedAbovePlanned}, []string{"A-1003"}},
		{[]string{FinanceHasPurchaseOrder}, []string{"A-1002", "A-1003"}},
		{[]string{FinanceInvoiced}, []string{"A-1002", "A-1003"}},
		{[]string{FinanceNoPurchaseOrder}, []string{"A-1001", "A-1004", "A-1005", "A-1006", "A-1007", "A-1008", "A-1009", "A-1010"}},
		{[]string{FinanceCommittedAbovePlanned, FinanceNoPurchaseOrder}, []string{"A-1001", "A-1003", "A-1004", "A-1005", "A-1006",
			"A-1007", "A-1008", "A-1009", "A-1010"}},
	}
	for _, c := range cases {
		r := view(financeYear(), Query{Finance: c.values})
		var matches []string
		for _, row := range r.Rows {
			if row.Level == LevelActivity && row.Role == RoleMatch {
				matches = append(matches, row.Key)
			}
		}
		slices.Sort(matches)
		if !slices.Equal(matches, c.want) {
			t.Errorf("%v matches %v, want %v", c.values, matches, c.want)
		}
	}
	// A purchase order credited back to 0.00 still counts as a PO.
	b := financeYear()
	b.node("L-1003").Finance = &FinanceFacts{POCount: 2}
	b.node("L-1004").Finance = nil
	r := view(b, Query{Finance: []string{FinanceHasPurchaseOrder}})
	if !slices.ContainsFunc(r.Rows, func(row Row) bool { return row.Key == "A-1002" && row.Role == RoleMatch }) {
		t.Errorf("Social always-on has a PO credited to 0.00 and should match")
	}
}

// A reader who may not see financial data gets neither the group nor the default financial columns; asked
// for anyway, the figures show "—" and the filter matches nothing (015 FR-028, consolidation).
func TestFinance_NoAccess_FR028(t *testing.T) {
	columns := forReader(catalogue(fixtureAttributes), false)
	if slices.ContainsFunc(columns, func(c Column) bool { return c.Group == GroupFinancials }) {
		t.Errorf("the Financials group is offered without access")
	}
	choices := fixtureChoices(priya.ID)
	choices.SeesFinance = false
	o := optionsOf(2026, 2026, nil, financeYear().data(), choices, offered(columns))
	if slices.Contains(o.DefaultColumns, "committed") || len(o.DefaultColumns) != 9 ||
		slices.ContainsFunc(o.ColumnGroups, func(g ColumnGroup) bool { return g.Group == GroupFinancials }) {
		t.Errorf("options = %v, %d groups", o.DefaultColumns, len(o.ColumnGroups))
	}

	in := input{query: Query{View: ViewGrid, Strategic: StrategicInclude, ShowLevels: ShowEverything, Period: PeriodMonth},
		year: 2026, today: fixtureToday, now: fixtureNow, currency: usd, data: financeYear().data(), choices: choices,
		catalogue: columns}
	r := build(in).result
	if len(r.Columns) != 9 || len(r.LeftOut) != 0 {
		t.Errorf("default columns without access = %d, left out %+v", len(r.Columns), r.LeftOut)
	}
	in.catalogue = catalogue(fixtureAttributes)
	in.query.Columns = []string{"committed"}
	in.query.Finance = []string{FinanceNoPurchaseOrder}
	r = build(in).result
	for _, row := range r.Rows {
		if row.Role == RoleMatch {
			t.Errorf("%s matches the Financials filter without access", row.Key)
		}
	}
	in.query.Finance = nil
	for _, row := range build(in).result.Rows {
		if row.Level != LevelUnassigned && figureText(row.Cells[0]) != "—" {
			t.Errorf("%s shows %s without access", row.Key, figureText(row.Cells[0]))
		}
	}
}

// A reader who may see financial data in some plans only sees the figures of those: the rows of the other
// plans show "—", never match the Financials filter and add nothing to the totals (015 FR-028,
// consolidation "All work", D-301).
func TestFinance_OnlyPlansWithAccess_FR028(t *testing.T) {
	choices := fixtureChoices(priya.ID)
	choices.FinancePlans = []string{"P-1001"}
	in := input{query: Query{View: ViewGrid, Strategic: StrategicInclude, ShowLevels: ShowEverything, Period: PeriodMonth,
		Columns: financeColumnKeys}, year: 2026, today: fixtureToday, now: fixtureNow, currency: usd,
		data: financeYear().data(), choices: choices, catalogue: catalogue(fixtureAttributes)}
	r := build(in).result
	seen := map[string]bool{}
	for _, row := range r.Rows {
		if row.Level == LevelUnassigned {
			continue
		}
		seen[row.PlanID] = true
		texts := cellTexts(row.Cells)
		dashes := !slices.ContainsFunc(texts, func(s string) bool { return s != "—" })
		if row.PlanID == "P-1002" && !dashes {
			t.Errorf("%s of a plan without access shows %q", row.Key, texts)
		}
		if row.PlanID == "P-1001" && row.Key != "A-1009" && row.Key != "I-1007" && dashes {
			t.Errorf("%s of a plan with access shows only dashes", row.Key)
		}
	}
	if !seen["P-1001"] || !seen["P-1002"] {
		t.Fatalf("the fixture should have rows of both plans: %v", seen)
	}
	if got := cellTexts(r.Totals.Cells); !slices.Equal(got, []string{"35,652.17", "33,152.17", "30,652.17", "2,500.00", "424,347.83"}) {
		t.Errorf("totals = %q, want the figures of P-1001 only", got)
	}

	in.query.Finance = []string{FinanceNoPurchaseOrder}
	matched := 0
	for _, row := range build(in).result.Rows {
		if row.Level == LevelActivity && row.Role == RoleMatch {
			matched++
			if row.PlanID != "P-1001" {
				t.Errorf("%s of a plan without access matches the Financials filter", row.Key)
			}
		}
	}
	if matched == 0 {
		t.Errorf("no activity of P-1001 matches No PO yet")
	}
}

// An export with a financial column says when the GSAP data was loaded, and the Financials filter is among
// its filter notes (015 FR-023, US5 scenario 3).
func TestFinance_ExportNotes_FR023(t *testing.T) {
	b := financeYear()
	data := b.data()
	asOf := time.Date(2026, 10, 19, 2, 0, 0, 0, time.UTC)
	data.FinanceAsOf = &asOf
	q := Query{Finance: []string{FinanceCommittedAbovePlanned}}
	built := viewOf(data, q)
	in := input{query: built.query, year: 2026, today: fixtureToday, now: fixtureNow, currency: usd, data: data,
		choices: fixtureChoices(priya.ID), catalogue: catalogue(fixtureAttributes)}
	file, err := exportFile(in, built, actor.Actor{ID: priya.ID, Name: priya.Name}, FormatCSV, true)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	notes, table := csvLines(t, file)
	if !slices.ContainsFunc(notes, func(n []string) bool {
		return n[0] == "Financial figures" && n[1] == "GSAP data as of 19 Oct 2026, 02:00 UTC"
	}) || !slices.ContainsFunc(notes, func(n []string) bool { return n[1] == "Financials: Committed above planned cost" }) {
		t.Errorf("notes = %q", notes)
	}
	if !slices.Contains(table[0], "Committed (USD)") || rowNamed(t, table, "Billboard winter wave")[12] != "10652.17" {
		t.Errorf("heading %q, Billboard %q", table[0], rowNamed(t, table, "Billboard winter wave"))
	}

	plain := viewOf(data, Query{Columns: []string{"id"}})
	in.query = plain.query
	file, _ = exportFile(in, plain, actor.Actor{ID: priya.ID, Name: priya.Name}, FormatCSV, true)
	notes, _ = csvLines(t, file)
	if slices.ContainsFunc(notes, func(n []string) bool { return n[0] == "Financial figures" }) {
		t.Errorf("a view without financial columns has the note: %q", notes)
	}
}

// The parameter finance takes the four values, and refuses others on its name (015 FR-024).
func TestFinance_ParseQuery_FR024(t *testing.T) {
	q, err := ParseQuery(map[string][]string{"finance": {"committedAbovePlanned", "invoiced"}})
	if err != nil || !slices.Equal(q.Finance, []string{"committedAbovePlanned", "invoiced"}) {
		t.Errorf("query = %v, %v", q.Finance, err)
	}
	if _, err := ParseQuery(map[string][]string{"finance": {"overBudget"}}); err == nil {
		t.Errorf("an unknown value was accepted")
	}
}
