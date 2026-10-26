package financials

import (
	"context"
	"encoding/base64"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// The service's rules on the demo data of data model §6, with fakes for the ports (plan, "Tests").

// User story 1 and 2 — the figures of every record (FR-011, FR-012, SC-002).

// want is one row of data model §6.1: "" for a null figure; above is "yes", "no" or the number of line
// items below committed above their planned cost.
type want struct {
	id, planned, committed, invoiced, paid, accrued, notYet string
	percent                                                 string
	above                                                   string
}

// demoFigures is the table of data model §6.1.
var demoFigures = []want{
	{"L-1003", "20000.00", "20000.00", "20000.00", "20000.00", "0.00", "0.00", "100", "no"},
	{"L-1004", "5000.00", "5000.00", "2500.00", "0.00", "2500.00", "0.00", "100", "no"},
	{"L-1006", "10000.00", "10652.17", "10652.17", "10652.17", "0.00", "-652.17", "107", "yes"},
	{"A-1002", "80000.00", "25000.00", "22500.00", "20000.00", "2500.00", "55000.00", "31", "no"},
	{"A-1003", "45000.00", "10652.17", "10652.17", "10652.17", "0.00", "34347.83", "24", "1"},
	{"A-1001", "120000.00", "0.00", "0.00", "0.00", "0.00", "120000.00", "0", "no"},
	{"I-1001", "200000.00", "25000.00", "22500.00", "20000.00", "2500.00", "175000.00", "13", "no"},
	{"I-1002", "75000.00", "10652.17", "10652.17", "10652.17", "0.00", "64347.83", "14", "1"},
	{"C-1001", "275000.00", "35652.17", "33152.17", "30652.17", "2500.00", "239347.83", "13", "1"},
	{"C-1002", "185000.00", "0.00", "0.00", "0.00", "0.00", "185000.00", "0", "no"},
	{"P-1001", "460000.00", "35652.17", "33152.17", "30652.17", "2500.00", "424347.83", "8", "1"},
}

// check compares figures with the row w.
func (w want) check(t *testing.T, f Figures) {
	t.Helper()
	if f.PlannedCost == nil || *f.PlannedCost != usd(t, w.planned) || f.Committed != usd(t, w.committed) ||
		f.Invoiced != usd(t, w.invoiced) || f.Paid != usd(t, w.paid) || f.Accrued != usd(t, w.accrued) ||
		f.NotYetCommitted == nil || *f.NotYetCommitted != usd(t, w.notYet) || f.CommittedPercent == nil ||
		w.percent != strconv.Itoa(*f.CommittedPercent) {
		t.Errorf("%s = %+v, want %+v", w.id, f, w)
	}
	switch w.above {
	case "yes":
		if !f.CommittedAbovePlanned {
			t.Errorf("%s is not above planned", w.id)
		}
	case "no":
		if f.CommittedAbovePlanned || f.LineItemsAbovePlanned != 0 {
			t.Errorf("%s above %v, %d line items", w.id, f.CommittedAbovePlanned, f.LineItemsAbovePlanned)
		}
	default:
		if f.CommittedAbovePlanned || strconv.Itoa(f.LineItemsAbovePlanned) != w.above || f.FirstLineItemAbovePlanned == nil ||
			f.FirstLineItemAbovePlanned.ID != "L-1006" {
			t.Errorf("%s above %v, %d line items, first %+v", w.id, f.CommittedAbovePlanned, f.LineItemsAbovePlanned,
				f.FirstLineItemAbovePlanned)
		}
	}
}

// Every record of data model §6.1 has its figures; every container's row in its parent's breakdown equals
// its own card (FR-011, FR-012, SC-002, US2).
func TestRecord_DemoFiguresAndRollUp_FR012(t *testing.T) {
	f := newFixture(t)
	for _, w := range demoFigures {
		v := f.record(t, priya, w.id)
		w.check(t, v.Figures)
		for _, row := range v.Breakdown {
			child := f.record(t, priya, row.Record.ID)
			if !reflect.DeepEqual(row.Figures, child.Figures) {
				t.Errorf("%s's breakdown row %s = %+v, card %+v", w.id, row.Record.ID, row.Figures, child.Figures)
			}
		}
	}
	counts := map[string]int{"A-1002": 7, "A-1003": 3, "L-1004": 4, "P-1001": 10}
	for id, n := range counts {
		if got := f.record(t, priya, id).Figures.TransactionCount; got != n {
			t.Errorf("%s has %d transactions, want %d", id, got, n)
		}
	}
	activity := f.record(t, priya, "A-1002")
	if len(activity.Latest) != 5 || activity.Latest[0].ID != "FT-1011" || activity.Latest[4].ID != "FT-1008" {
		t.Errorf("latest = %v", rowIDs(activity.Latest))
	}
	if v := f.record(t, priya, "L-1003"); len(v.SwbsCodes) != 1 || v.SwbsCodes[0] != "DE03-M26-0003" {
		t.Errorf("Venue hire SWBS codes = %v", v.SwbsCodes)
	}
}

// rowIDs lists the ids of rows.
func rowIDs(rows []TransactionRow) []string {
	list := []string{}
	for _, r := range rows {
		list = append(list, r.ID)
	}
	return list
}

// A reader with the EUR display currency sees each figure converted once from its USD sum (FR-011, D-061).
func TestRecord_DisplayCurrency_FR011(t *testing.T) {
	f := newFixture(t)
	v := f.record(t, lena, "A-1003")
	eur := func(text string) money.Money { return money.Money{Amount: amountOf(t, text), Currency: "EUR"} }
	if v.Figures.Committed != eur("9800.00") || *v.Figures.PlannedCost != eur("41400.00") {
		t.Errorf("A-1003 in EUR = %+v", v.Figures)
	}
	if row := v.Latest[0]; row.Amount != eur("9800.00") || *row.AmountDisplay != eur("9800.00") {
		t.Errorf("latest = %+v", row)
	}
}

// An activity moved to another initiative takes its figures with it (FR-012).
func TestRecord_MovedActivityTakesItsFigures_FR012(t *testing.T) {
	f := newFixture(t)
	for i := range f.planning.nodes {
		if f.planning.nodes[i].ID == "A-1003" {
			f.planning.nodes[i].ParentID = "I-1001"
		}
	}
	if got := f.record(t, priya, "I-1001").Figures.Committed; got != usd(t, "35652.17") {
		t.Errorf("Digital & social committed = %v, want 35,652.17", got)
	}
	if got := f.record(t, priya, "I-1002").Figures; got.Committed != usd(t, "0.00") || got.LineItemsAbovePlanned != 0 {
		t.Errorf("Print & outdoor = %+v, want nothing", got)
	}
	if got := f.record(t, priya, "C-1001").Figures.Committed; got != usd(t, "35652.17") {
		t.Errorf("the campaign changed: %v", got)
	}
}

// A record that is not live is not found, with the level's sentence.
func TestRecord_NotFound_FR015(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Record(t.Context(), priya, "C-9999")
	if e := assertCode(t, err, http.StatusNotFound, errs.CodeNotFound); e.Message != "This campaign no longer exists — someone may have deleted it." {
		t.Errorf("message = %q", e.Message)
	}
}

// A deleted line item's transactions stay linked and leave every figure; restoring it brings them back
// (FR-012, D-249).
func TestRecord_LineItemInTheBinLeavesTheFigures_FR012(t *testing.T) {
	f := newFixture(t)
	removed := f.planning.moveToBin("L-1006")

	if got := f.record(t, priya, "A-1003").Figures; got.Committed != usd(t, "0.00") || got.LineItemsAbovePlanned != 0 ||
		got.TransactionCount != 0 {
		t.Errorf("Billboard winter wave = %+v, want nothing", got)
	}
	if got := f.record(t, priya, "P-1001").Figures.Committed; got != usd(t, "25000.00") {
		t.Errorf("plan committed = %v, want 25,000.00", got)
	}
	view, err := f.service.Transaction(t.Context(), alex, "FT-1003")
	if err != nil {
		t.Fatalf("Transaction: %v", err)
	}
	if view.Link.State != StateLineItemDeleted || *view.Link.LineItemID != "L-1006" ||
		*view.Link.Reason != "Line item L-1006 is in the recycle bin — its amounts come back when it's restored." {
		t.Errorf("link = %+v", view.Link)
	}
	_, err = f.service.Record(t.Context(), priya, "L-1006")
	assertCode(t, err, http.StatusNotFound, errs.CodeNotFound)

	f.planning.restore(removed)
	demoFigures[4].check(t, f.record(t, priya, "A-1003").Figures)
}

// Copies never copy transactions: a copied activity and its line items show 0.00 (FR-030).
func TestRecord_CopiesHaveNoTransactions_FR030(t *testing.T) {
	f := newFixture(t)
	f.planning.nodes = append(f.planning.nodes,
		node(LevelActivity, "A-2001", "I-1001", "P-1001", "Social always-on (copy)", 9, "80000.00"),
		node(LevelLineItem, "L-2001", "A-2001", "P-1001", "Venue hire", 9, "20000.00"),
		node(LevelLineItem, "L-2002", "A-2001", "P-1001", "Speaker fees", 10, "5000.00"),
	)
	for _, id := range []string{"A-2001", "L-2001", "L-2002"} {
		v := f.record(t, priya, id)
		if v.Figures.Committed != usd(t, "0.00") || v.Figures.TransactionCount != 0 || len(v.Latest) != 0 {
			t.Errorf("%s = %+v", id, v.Figures)
		}
	}
	if n := len(f.transactions(t)); n != 11 {
		t.Errorf("%d transactions, want the 11 of the demo", n)
	}
}

// A record without transactions shows 0.00 for every amount, its whole planned cost as not yet committed and
// no latest transactions, so the card can say that none have come yet (FR-014).
func TestRecord_NoTransactionsShowsZero_FR014(t *testing.T) {
	f := newFixture(t)
	for id, planned := range map[string]string{"A-1001": "120000.00", "C-1002": "185000.00"} {
		v := f.record(t, priya, id)
		zero := usd(t, "0.00")
		got := v.Figures
		if got.Committed != zero || got.Invoiced != zero || got.Paid != zero || got.Accrued != zero ||
			got.TransactionCount != 0 || len(v.Latest) != 0 || got.NotYetCommitted == nil ||
			*got.NotYetCommitted != usd(t, planned) || got.CommittedPercent == nil || *got.CommittedPercent != 0 {
			t.Errorf("%s = %+v, latest %v", id, got, rowIDs(v.Latest))
		}
	}
}

// Every level answers how many transactions are linked under it, the count the delete dialogs show; a line
// item in the recycle bin no longer counts (FR-029).
func TestRecord_TransactionCountForDeleteDialogs_FR029(t *testing.T) {
	f := newFixture(t)
	want := map[string]int{
		"P-1001": 10, "C-1001": 10, "C-1002": 0, "I-1001": 7, "I-1002": 3,
		"A-1001": 0, "A-1002": 7, "A-1003": 3, "L-1003": 3, "L-1004": 4, "L-1006": 3,
	}
	for id, n := range want {
		if got := f.record(t, priya, id).Figures.TransactionCount; got != n {
			t.Errorf("%s has %d transactions, want %d", id, got, n)
		}
	}
	f.planning.moveToBin("L-1004")
	if got := f.record(t, priya, "A-1002").Figures.TransactionCount; got != 3 {
		t.Errorf("Social always-on without Speaker fees has %d transactions, want 3", got)
	}
}

// User story 3 — the Financials page (FR-021).

// list returns the page for q, stopping the test on an error.
func (f *fixture) list(t *testing.T, a actor.Actor, q Query) TransactionPage {
	t.Helper()
	if q.Type == "" {
		q.Type = filterAll
	}
	if q.Link == "" {
		q.Link = filterAll
	}
	if q.Sort == "" {
		q.Sort, q.Desc = sortPostingDate, true
	}
	if q.Page == 0 {
		q.Page, q.PageSize = 1, 50
	}
	page, err := f.service.Transactions(t.Context(), a, q)
	if err != nil {
		t.Fatalf("Transactions(%+v): %v", q, err)
	}
	return page
}

// The tab counts and totals of the seed, and every filter (FR-021, US3 scenario 1).
func TestTransactions_FiltersCountsAndTotals_FR021(t *testing.T) {
	f := newFixture(t)
	all := f.list(t, alex, Query{})
	wantCounts := TypeCounts{All: 11, PurchaseOrder: 4, Invoice: 3, Payment: 2, Accrual: 2, NotLinked: 1}
	if all.Counts != wantCounts || all.Total != 11 || all.Year != 2026 {
		t.Errorf("counts %+v, total %d, year %d", all.Counts, all.Total, all.Year)
	}
	wantTotals := Totals{Committed: usd(t, "47652.17"), Invoiced: usd(t, "33152.17"), Paid: usd(t, "30652.17"), Accrued: usd(t, "2500.00")}
	if all.Totals != wantTotals {
		t.Errorf("totals = %+v", all.Totals)
	}
	if len(all.Choices.Years) != 1 || all.Choices.Years[0] != 2026 || len(all.Choices.Plans) != 2 ||
		all.Choices.Plans[0].Name != "FY2026 Lubricants Europe" {
		t.Errorf("choices = %+v", all.Choices)
	}

	cases := []struct {
		name  string
		q     Query
		total int
		first string
	}{
		{"type", Query{Type: string(TypeInvoice)}, 3, "FT-1006"},
		{"linked", Query{Link: filterLinked}, 10, "FT-1011"},
		{"not linked", Query{Link: filterNotLinked}, 1, "FT-1004"},
		{"plan", Query{PlanID: "P-1001"}, 10, "FT-1011"},
		{"other plan", Query{PlanID: "P-1002"}, 0, ""},
		{"record", Query{RecordID: "A-1003"}, 3, "FT-1009"},
		{"year", Query{Year: 2027}, 0, ""},
		{"search vendor", Query{Search: "brightside"}, 1, "FT-1004"},
		{"search HX key", Query{Search: "hx1003"}, 1, "FT-1001"},
		{"search SWBS code", Query{Search: "GB01-M26"}, 3, "FT-1009"},
		{"search line item", Query{Search: "Venue Hire"}, 3, "FT-1008"},
		{"search line item ID", Query{Search: "L-1004"}, 4, "FT-1011"},
		{"search activity ID", Query{Search: "a-1003"}, 3, "FT-1009"},
		{"sort document number", Query{Sort: sortDocumentNumber}, 11, "FT-1008"},
		{"sort vendor", Query{Sort: sortVendor}, 11, "FT-1004"},
		{"sort type", Query{Sort: sortType}, 11, "FT-1004"},
		{"sort linked to", Query{Sort: sortLinkedTo}, 11, "FT-1004"},
		{"sort amount", Query{Sort: sortAmount, Desc: true}, 11, "FT-1008"},
		{"page 2", Query{Page: 2, PageSize: 50}, 11, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			page := f.list(t, alex, c.q)
			first := ""
			if len(page.Items) > 0 {
				first = page.Items[0].ID
			}
			if page.Total != c.total || first != c.first {
				t.Errorf("total %d, first %q; want %d, %q (%v)", page.Total, first, c.total, c.first, rowIDs(page.Items))
			}
		})
	}

	invoices := f.list(t, alex, Query{Type: string(TypeInvoice)})
	if invoices.Counts != wantCounts || invoices.Totals != wantTotals {
		t.Errorf("the tab changed the counts or totals: %+v %+v", invoices.Counts, invoices.Totals)
	}
	notLinked := f.list(t, alex, Query{Link: filterNotLinked})
	if notLinked.Totals.Committed != usd(t, "12000.00") || notLinked.Counts != wantCounts {
		t.Errorf("not linked: totals %+v, counts %+v", notLinked.Totals, notLinked.Counts)
	}
	record := f.list(t, alex, Query{RecordID: "L-1004"})
	if record.RecordFilter == nil || record.RecordFilter.Name != "Speaker fees" || record.Counts.All != 4 || record.Counts.NotLinked != 0 {
		t.Errorf("record filter %+v, counts %+v", record.RecordFilter, record.Counts)
	}
}

// A record or plan filter that is not live is refused on its parameter (FR-021).
func TestTransactions_UnknownRecordAndPlan_FR021(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Transactions(t.Context(), alex, Query{Type: filterAll, Link: filterAll, Page: 1, PageSize: 50, RecordID: "L-9999"})
	if e := assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed); e.Fields[0].Field != "recordId" {
		t.Errorf("fields = %+v", e.Fields)
	}
	_, err = f.service.Transactions(t.Context(), alex, Query{Type: filterAll, Link: filterAll, Page: 1, PageSize: 50, PlanID: "P-9999"})
	if e := assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed); e.Fields[0].Field != "planId" {
		t.Errorf("fields = %+v", e.Fields)
	}
}

// ParseQuery takes the defaults and reports every wrong parameter at once (FR-021, D-219).
func TestParseQuery_DefaultsAndEveryProblem_FR021(t *testing.T) {
	q, err := ParseQuery(map[string][]string{})
	if err != nil || q != (Query{Type: filterAll, Link: filterAll, Sort: sortPostingDate, Desc: true, Page: 1, PageSize: 50}) {
		t.Errorf("defaults = %+v, %v", q, err)
	}
	q, err = ParseQuery(map[string][]string{"sort": {"vendor:asc"}, "q": {"  helix "}, "year": {"2025"}, "pageSize": {"200"}})
	if err != nil || q.Sort != sortVendor || q.Desc || q.Search != "helix" || q.Year != 2025 || q.PageSize != 200 {
		t.Errorf("query = %+v, %v", q, err)
	}
	_, err = ParseQuery(map[string][]string{"type": {"x"}, "year": {"abc"}, "page": {"-1"}})
	if e := assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed); len(e.Fields) != 3 {
		t.Errorf("fields = %+v, want type, year and page", e.Fields)
	}
}

// User story 4 — load GSAP data (FR-004 – FR-008, SC-003).

// importFile imports req as a, stopping the test on an error.
func (f *fixture) importFile(t *testing.T, a actor.Actor, req ImportRequest) ImportResult {
	t.Helper()
	result, err := f.service.Import(t.Context(), a, req)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	return result
}

// threeRows is the file of US4: a new PO for HX1007, an invoice for it, and a row with a currency not in
// Settings.
var threeRows = file("gsap-2026-10-19.csv", header,
	"PO,4500001010,10,,,2026-10-01,,3000.00,USD,V-1,Signs Ltd,Signage,,HX1007,",
	"INVOICE,5100002010,,4500001010,,2026-10-05,,1000.00,USD,V-1,Signs Ltd,Signage,,,",
	"PO,4500001011,10,,,2026-10-01,,500.00,XYZ,V-1,Signs Ltd,Signage,,HX1007,",
)

// A check says what the import does and saves nothing: no transaction, import, ID or log entry (FR-004,
// SC-003, D-177).
func TestCheck_SavesNothing_FR004(t *testing.T) {
	f := newFixture(t)
	check, err := f.service.Check(t.Context(), alex, threeRows)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if check.Counts != (ImportCounts{Rows: 3, New: 2, Refused: 1}) || check.ProblemsTotal != 1 || check.NotLinkedTotal != 0 ||
		check.FileName != "gsap-2026-10-19.csv" {
		t.Errorf("check = %+v", check)
	}
	if n := len(f.transactions(t)); n != 11 {
		t.Errorf("%d transactions after the check, want 11", n)
	}
	if entries := f.entries(t); len(entries) != 0 {
		t.Errorf("log entries after the check: %+v", entries)
	}
	result := f.importFile(t, alex, threeRows)
	if result.Import.ID != "FIM-1002" || result.Import.Counts != check.Counts {
		t.Errorf("import %s %+v, want FIM-1002 with the check's counts", result.Import.ID, result.Import.Counts)
	}
	if got := f.stored(t, "FT-1012"); got.DocumentNumber != "4500001010" || got.LineItemID != "L-1007" {
		t.Errorf("FT-1012 = %+v", got)
	}
}

// Rows = new + updated + unchanged + refused, and the import says the same as its check; importing the same
// file twice changes nothing the second time (FR-006, SC-003).
func TestImport_CountsAddUpAndTwiceChangesNothing_FR006(t *testing.T) {
	f := newFixture(t)
	req := file("gsap.csv", header,
		"PO,4500001001,10,,,2026-01-12,,15800.00,GBP,V-100101,Northlight Events GmbH,Venue hire Berlin,DE03-M26-0003,HX1003,",
		"PO,4500001002,10,,,2026-05-04,,5200.00,USD,V-100102,Keynote Speakers Ltd,Keynote speakers,,HX1004,",
		"PAYMENT,1500003010,,4500001002,,2026-10-01,,1000.00,USD,,,,,,",
		"PAYMENT,1500003011,,,,2026-10-01,,1.00,USD,,,,,,",
		"PAYMENT,1500003012,,,,not a date,,1.00,USD,,,,,,",
	)
	check, err := f.service.Check(t.Context(), alex, req)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	first := f.importFile(t, alex, req)
	c := first.Import.Counts
	if c != check.Counts || c != (ImportCounts{Rows: 5, New: 2, Updated: 1, Unchanged: 1, Refused: 1, NotLinked: 1}) ||
		c.Rows != c.New+c.Updated+c.Unchanged+c.Refused {
		t.Errorf("counts = %+v (check %+v)", c, check.Counts)
	}
	if got := f.stored(t, "FT-1002"); got.Amount != amountOf(t, "5200.00") || got.ImportID != "FIM-1002" || got.Version != 2 {
		t.Errorf("updated FT-1002 = %+v", got)
	}
	if got := f.stored(t, "FT-1001"); got.Version != 1 || got.ImportID != "FIM-1001" {
		t.Errorf("unchanged FT-1001 = %+v", got)
	}
	if first.NotLinked[0] != (NotLinkedRow{Row: 5, Name: "Payment 1500003011", Reason: reasonNoReference}) {
		t.Errorf("not linked = %+v", first.NotLinked)
	}

	before := f.transactions(t)
	second := f.importFile(t, alex, req)
	if second.Import.Counts != (ImportCounts{Rows: 5, Unchanged: 4, Refused: 1, NotLinked: 1}) {
		t.Errorf("second import = %+v", second.Import.Counts)
	}
	after := f.transactions(t)
	for i := range before {
		if before[i].Version != after[i].Version {
			t.Errorf("%s changed: version %d → %d", before[i].ID, before[i].Version, after[i].Version)
		}
	}
}

// Refused rows are skipped and listed with their row; a key repeated in the file refuses the later row; the
// rest are imported; lists stop at 200, the totals say how many there are (FR-005).
func TestImport_RefusedRowsAreSkipped_FR005(t *testing.T) {
	f := newFixture(t)
	result := f.importFile(t, alex, file("gsap.csv", header,
		"PO,4500009001,10,,,2026-10-01,,100.00,USD,,,,,HX1007,",
		"PO,4500009002,10,,,2026-10-01,,100.00,CHF,,,,,HX1007,",
		"PO,4500009001,10,,,2026-10-02,,200.00,USD,,,,,HX1007,",
	))
	wantProblems := []ImportProblem{
		{Row: 3, Column: "currency", Message: "Currency 'CHF' isn't set up in Settings — an administrator adds it under Currencies."},
		{Row: 4, Column: "document_number", Message: "Row 4 repeats PO 4500009001 / 10 from row 2 — keep one."},
	}
	if result.Import.Counts != (ImportCounts{Rows: 3, New: 1, Refused: 2}) || len(result.Problems) != 2 ||
		result.Problems[0] != wantProblems[0] || result.Problems[1] != wantProblems[1] {
		t.Errorf("result = %+v", result)
	}
	if got := f.stored(t, "FT-1012"); got.Amount != amountOf(t, "100.00") {
		t.Errorf("the first row was not kept: %+v", got)
	}

	lines := []string{header}
	for i := range 250 {
		lines = append(lines, "PO,46000"+strconv.Itoa(10000+i)+",10,,,2026-10-01,,1.00,XYZ,,,,,,")
	}
	check, err := f.service.Check(t.Context(), alex, file("many.csv", lines...))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if check.ProblemsTotal != 250 || len(check.Problems) != 200 || check.Counts.Refused != 250 {
		t.Errorf("problems %d of %d, refused %d", len(check.Problems), check.ProblemsTotal, check.Counts.Refused)
	}
}

// Purchase orders are applied first, so an invoice finds a PO of the same file and the PO gets the lower
// ID (FR-007, data model §4.1).
func TestImport_POsFirstAndInDeliveryReferences_FR007(t *testing.T) {
	f := newFixture(t)
	f.importFile(t, alex, file("gsap.csv", header,
		"INVOICE,5100009001,,4500009001,,2026-10-02,,100.00,USD,,,,,,",
		"PO,4500009001,10,,,2026-10-01,,400.00,USD,,,,,HX1007,",
	))
	po, invoice := f.stored(t, "FT-1012"), f.stored(t, "FT-1013")
	if po.Type != TypePurchaseOrder || po.LineItemID != "L-1007" || po.MatchedBy != MatchedByTrackingKey {
		t.Errorf("FT-1012 = %+v", po)
	}
	if invoice.Type != TypeInvoice || invoice.LineItemID != "L-1007" || invoice.MatchedBy != MatchedByPONumber {
		t.Errorf("FT-1013 = %+v", invoice)
	}
}

// Every import matches the stored not-linked transactions again and counts those it links; the log entry
// says so (FR-007, FR-027, US4 scenario 8).
func TestImport_RetryRelinksAndLogs_FR007(t *testing.T) {
	f := newFixture(t)
	first := f.importFile(t, alex, file("invoice.csv", header, "INVOICE,5100009002,,4500009002,,2026-10-02,,100.00,USD,,,,,,"))
	if first.NotLinked[0].Reason != "PO 4500009002 isn't linked to a line item yet." {
		t.Errorf("not linked = %+v", first.NotLinked)
	}
	f.now = f.now.Add(1)
	second := f.importFile(t, alex, file("po.csv", header, "PO,4500009002,10,,,2026-10-01,,400.00,USD,,,,,HX1007,"))
	if second.Import.Counts != (ImportCounts{Rows: 1, New: 1, Relinked: 1}) {
		t.Errorf("second import = %+v", second.Import.Counts)
	}
	if got := f.stored(t, "FT-1012"); got.LineItemID != "L-1007" || got.MatchedBy != MatchedByPONumber || got.NotLinkedReason != "" {
		t.Errorf("the invoice was not relinked: %+v", got)
	}

	entries := f.entries(t)
	if len(entries) != 2 {
		t.Fatalf("%d log entries, want one per import", len(entries))
	}
	e := entries[1]
	if e.RecordType != "financialImport" || e.Verb != "imported" || e.RecordID != "FIM-1003" || e.RecordName != "po.csv" ||
		e.Actor.Name != "Alex Morgan" || e.RecordTypeLabel != "GSAP extract" ||
		e.Details != "1 row: 1 new, 0 updated, 0 unchanged, 0 refused · 0 not linked · 1 earlier transaction is now linked" {
		t.Errorf("entry = %+v", e)
	}
	if entries[0].Details != "1 row: 1 new, 0 updated, 0 unchanged, 0 refused · 1 not linked" {
		t.Errorf("first entry = %q", entries[0].Details)
	}
}

// An import never changes a manual link, even when it updates the transaction (FR-007, FR-026).
func TestImport_KeepsManualLinks_FR007(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.Link(t.Context(), alex, "FT-1004", LinkRequest{LineItemID: "L-1007", Version: version(1)}); err != nil {
		t.Fatalf("Link: %v", err)
	}
	result := f.importFile(t, alex, file("gsap.csv", header,
		"PO,4500001004,10,,,2026-09-15,,12500.00,USD,V-100104,Brightside Media BV,Retail display production,,HX1099,"))
	if result.Import.Counts.Updated != 1 || result.Import.Counts.NotLinked != 0 {
		t.Errorf("counts = %+v", result.Import.Counts)
	}
	got := f.stored(t, "FT-1004")
	if got.LineItemID != "L-1007" || got.MatchedBy != MatchedByManual || got.ManualLink == nil || got.Amount != amountOf(t, "12500.00") {
		t.Errorf("FT-1004 = %+v", got)
	}
}

// A whole-file problem saves nothing; unknown columns are named in the check (FR-004).
func TestImport_WholeFileRefusalSavesNothing_FR004(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Import(t.Context(), alex, ImportRequest{FileName: "gsap.csv", Content: "not base64!"})
	if e := assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed); e.Fields[0] != (errs.FieldError{Field: "content",
		Message: "The file couldn't be read — choose it again."}) {
		t.Errorf("fields = %+v", e.Fields)
	}
	_, err = f.service.Import(t.Context(), alex, file("gsap.csv", "type,amount", "PO,1"))
	assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed)
	page, err := f.service.Imports(t.Context(), alex)
	if err != nil || page.Total != 1 || len(f.entries(t)) != 0 {
		t.Errorf("imports %+v, %v; log %d", page, err, len(f.entries(t)))
	}
	check, err := f.service.Check(t.Context(), alex, file("gsap.csv", header+",cost_centre", "PO,1,,,,2026-01-01,,1,USD,,,,,,,CC-1"))
	if err != nil || len(check.UnusedColumns) != 1 || check.UnusedColumns[0] != "cost_centre" {
		t.Errorf("unused = %v, %v", check.UnusedColumns, err)
	}
}

// The import history is newest first; GSAP data as of is the newest import (FR-008).
func TestImports_NewestFirstAndDataAsOf_FR008(t *testing.T) {
	f := newFixture(t)
	asOf, err := f.service.DataAsOf(t.Context())
	if err != nil || asOf == nil || !asOf.Equal(start.Add(-24*time.Hour)) {
		t.Errorf("data as of = %v, %v", asOf, err)
	}
	f.importFile(t, alex, threeRows)
	page, err := f.service.Imports(t.Context(), alex)
	if err != nil || page.Total != 2 || page.Items[0].ID != "FIM-1002" || page.Items[1].Source != SourceDemo {
		t.Errorf("imports = %+v, %v", page, err)
	}
	if asOf, _ = f.service.DataAsOf(t.Context()); !asOf.Equal(start) {
		t.Errorf("data as of = %v, want the import's time", asOf)
	}
	empty := newEmptyFixture(t)
	if asOf, _ := empty.service.DataAsOf(t.Context()); asOf != nil {
		t.Errorf("without data: %v", asOf)
	}
	if v := empty.record(t, alex, "A-1002"); v.Source.DataAsOf != nil || v.Figures.Committed != usd(t, "0.00") {
		t.Errorf("empty card = %+v", v)
	}
}

// fakeFeed is a second source of GSAP data, as a real GSAP feed later (SC-006).
type fakeFeed struct{ rows []Incoming }

func (fakeFeed) Name() string   { return "GSAP PGSI 941" }
func (fakeFeed) Source() Source { return SourceFile }
func (f fakeFeed) Read(context.Context, Delivery) ([]Incoming, []RowProblem, []string, error) {
	return f.rows, nil, nil, nil
}

// Another feed replaces the simulated extract with no other change: the same checks, matching and figures
// (FR-003, SC-006).
func TestImport_AnotherFeed_FR003(t *testing.T) {
	f := newFixture(t)
	f.service.UseFeed(fakeFeed{rows: []Incoming{{Row: 2, Fields: Fields{
		Type: TypePurchaseOrder, DocumentNumber: "4500007001", PONumber: "4500007001", PostingDate: date.MustParse("2026-10-01"),
		Amount: amountOf(t, "2000.00"), Currency: "USD", TrackingKey: "HX1007",
	}}}})
	result := f.importFile(t, alex, ImportRequest{FileName: "pgsi941.dat", Content: base64.StdEncoding.EncodeToString([]byte("x"))})
	if result.Import.FeedName != "GSAP PGSI 941" || result.Import.Counts.New != 1 {
		t.Errorf("import = %+v", result.Import)
	}
	if got := f.record(t, alex, "L-1007").Figures.Committed; got != usd(t, "2000.00") {
		t.Errorf("Station signage committed = %v", got)
	}
}

// The sample file has the header and three rows that import cleanly (FR-009).
func TestTemplate_ImportsCleanly_FR009(t *testing.T) {
	f := newFixture(t)
	body, err := f.service.Template(t.Context(), alex)
	if err != nil {
		t.Fatalf("Template: %v", err)
	}
	check, err := f.service.Check(t.Context(), alex, ImportRequest{FileName: TemplateName, Content: base64.StdEncoding.EncodeToString(body)})
	if err != nil || check.Counts != (ImportCounts{Rows: 3, New: 3}) || check.ProblemsTotal != 0 {
		t.Errorf("check of the sample = %+v, %v", check, err)
	}
}

// User story 6 — Home's card (FR-025).

// The overview of the current year: Priya's team plan, everything, and the activities above planned cost
// (FR-025, data model §6.1).
func TestOverview_MyTeamsAndEverything_FR025(t *testing.T) {
	f := newFixture(t)
	mine, err := f.service.Overview(t.Context(), priya, "myTeams")
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if mine.Year != 2026 || mine.Figures.Committed != usd(t, "35652.17") || *mine.Figures.PlannedCost != usd(t, "460000.00") ||
		*mine.Figures.CommittedPercent != 8 || mine.ActivitiesAbovePlanned != 1 || mine.NotLinked != 1 ||
		mine.Figures.FirstLineItemAbovePlanned.ID != "L-1006" {
		t.Errorf("my teams = %+v", mine)
	}
	everything, err := f.service.Overview(t.Context(), alex, "everything")
	if err != nil || *everything.Figures.PlannedCost != usd(t, "500000.00") || everything.Scope != ScopeEverything {
		t.Errorf("everything = %+v, %v", everything, err)
	}
	none, err := f.service.Overview(t.Context(), alex, "myTeams")
	if err != nil || none.Figures.PlannedCost != nil || none.Figures.Committed != usd(t, "0.00") {
		t.Errorf("Alex's teams have no plan: %+v, %v", none.Figures, err)
	}
	_, err = f.service.Overview(t.Context(), alex, "all")
	assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed)
}

// User story 7 — link a transaction by hand (FR-026, FR-027).

// A manual link is logged with the line item before and after; unlinking matches again and is logged too
// (FR-026, FR-027).
func TestLinkAndUnlink_Logged_FR027(t *testing.T) {
	f := newFixture(t)
	view, err := f.service.Link(t.Context(), alex, "FT-1004", LinkRequest{LineItemID: "L-1007", Version: version(1)})
	if err != nil {
		t.Fatalf("Link: %v", err)
	}
	if view.Link.State != StateLinked || *view.Link.MatchedBy != MatchedByManual || !view.Link.ManualAt.Equal(start) ||
		!view.CanUnlink || view.CanLink {
		t.Errorf("view = %+v", view)
	}
	if _, err := f.service.Unlink(t.Context(), alex, "FT-1004", UnlinkRequest{Version: version(2)}); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	got := f.stored(t, "FT-1004")
	if got.LineItemID != "" || got.ManualLink != nil || got.NotLinkedReason != "No line item for HX key HX1099 — check the tracking key in GSAP." {
		t.Errorf("after unlink = %+v", got)
	}
	entries := f.entries(t)
	if len(entries) != 2 {
		t.Fatalf("%d entries, want 2", len(entries))
	}
	linked, unlinked := entries[0], entries[1]
	if linked.Verb != "linked" || linked.RecordType != "financialTransaction" || linked.RecordName != "PO 4500001004 / 10" ||
		len(linked.Changes) != 1 || linked.Changes[0].Label != "Line item" || linked.Changes[0].OldText != "Not linked" ||
		linked.Changes[0].NewText != "Station signage (L-1007)" {
		t.Errorf("linked entry = %+v", linked)
	}
	if unlinked.Verb != "unlinked" || unlinked.Changes[0].OldText != "Station signage (L-1007)" || unlinked.Changes[0].NewText != "Not linked" {
		t.Errorf("unlinked entry = %+v", unlinked)
	}
}

// Linking refuses a line item in the recycle bin, an unknown one and a wrong ID form, with the field's
// message (FR-026).
func TestLink_RefusesLineItemsNotLive_FR026(t *testing.T) {
	f := newFixture(t)
	f.planning.moveToBin("L-1005")
	cases := map[string]string{
		"L-1005": "Line item L-1005 is in the recycle bin — restore it first.",
		"L-9999": "No line item L-9999.",
		"A-1002": "Enter a line item ID like L-1007.",
	}
	for id, message := range cases {
		_, err := f.service.Link(t.Context(), alex, "FT-1004", LinkRequest{LineItemID: id, Version: version(1)})
		e := assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed)
		if e.Fields[0] != (errs.FieldError{Field: "lineItemId", Message: message}) {
			t.Errorf("%s: fields = %+v", id, e.Fields)
		}
	}
	if got := f.stored(t, "FT-1004"); got.Version != 1 || got.LineItemID != "" {
		t.Errorf("FT-1004 changed: %+v", got)
	}
}

// Polish — the access seam (FR-028, D-252, D-301).

// Without finance.view every read is forbidden (FR-028).
func TestAccess_NoViewRefusesEveryRead_FR028(t *testing.T) {
	f := newFixture(t)
	f.access.noView = true
	ctx := t.Context()
	_, err := f.service.Record(ctx, priya, "A-1002")
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Transactions(ctx, priya, Query{Type: filterAll, Link: filterAll, Page: 1, PageSize: 50})
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Transaction(ctx, priya, "FT-1001")
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Overview(ctx, priya, "myTeams")
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Imports(ctx, priya)
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
}

// Without finance.import a person cannot check, import or get the sample file, and does not see the
// transactions that are not linked: not in the lists, counts, totals or the overview (FR-028, D-301).
func TestAccess_NoImportHidesNotLinked_FR028(t *testing.T) {
	f := newFixture(t)
	f.access.noImport = true
	ctx := t.Context()
	_, err := f.service.Check(ctx, priya, threeRows)
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Import(ctx, priya, threeRows)
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Template(ctx, priya)
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Transaction(ctx, priya, "FT-1004")
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)

	page := f.list(t, priya, Query{})
	if page.Counts.All != 10 || page.Counts.NotLinked != 0 || page.Totals.Committed != usd(t, "35652.17") {
		t.Errorf("counts %+v, totals %+v", page.Counts, page.Totals)
	}
	overview, err := f.service.Overview(ctx, priya, "everything")
	if err != nil || overview.NotLinked != 0 {
		t.Errorf("overview = %+v, %v", overview, err)
	}
}

// Without finance.link a person can neither link nor unlink, and the panel offers neither (FR-028).
func TestAccess_NoLinkRefusesLinking_FR028(t *testing.T) {
	f := newFixture(t)
	f.access.noLink = true
	_, err := f.service.Link(t.Context(), priya, "FT-1004", LinkRequest{LineItemID: "L-1007", Version: version(1)})
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Unlink(t.Context(), priya, "FT-1004", UnlinkRequest{Version: version(1)})
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	view, err := f.service.Transaction(t.Context(), priya, "FT-1004")
	if err != nil || view.CanLink || view.CanUnlink {
		t.Errorf("view = %+v, %v", view, err)
	}
}

// A person who sees only some plans gets only their transactions; a record of another plan, or a link into
// one, is forbidden (FR-028, 014 FR-007).
func TestAccess_VisiblePlansScopeEverything_FR028(t *testing.T) {
	f := newFixture(t)
	f.access.plans = []string{"P-1002"}
	ctx := t.Context()
	_, err := f.service.Record(ctx, priya, "A-1002")
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Transaction(ctx, priya, "FT-1001")
	e := assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	if e.Message != "You don't have access to this GSAP transaction. Ask an administrator if you need it." {
		t.Errorf("message = %q", e.Message)
	}
	if _, err := f.service.Record(ctx, priya, "A-1008"); err != nil {
		t.Errorf("Record(A-1008): %v", err)
	}
	page := f.list(t, priya, Query{})
	if page.Counts.All != 1 || page.Items[0].ID != "FT-1004" || len(page.Choices.Plans) != 1 || page.Choices.Plans[0].ID != "P-1002" {
		t.Errorf("page = %+v", page)
	}
	overview, err := f.service.Overview(ctx, priya, "everything")
	if err != nil || overview.Figures.Committed != usd(t, "0.00") || *overview.Figures.PlannedCost != usd(t, "40000.00") ||
		overview.ActivitiesAbovePlanned != 0 {
		t.Errorf("overview = %+v, %v", overview.Figures, err)
	}
	_, err = f.service.Link(ctx, priya, "FT-1004", LinkRequest{LineItemID: "L-1003", Version: version(1)})
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Transactions(ctx, priya, Query{Type: filterAll, Link: filterAll, Page: 1, PageSize: 50, PlanID: "P-1001"})
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
}

// FactsByLineItem gives All work the same figures as the cards (FR-023, SC-002).
func TestFactsByLineItem_SameAsTheCards_SC002(t *testing.T) {
	f := newFixture(t)
	facts, err := f.service.FactsByLineItem(t.Context())
	if err != nil {
		t.Fatalf("FactsByLineItem: %v", err)
	}
	if len(facts) != 3 {
		t.Errorf("%d line items with facts, want 3", len(facts))
	}
	for _, id := range []string{"L-1003", "L-1004", "L-1006"} {
		card := f.record(t, alex, id).Figures
		got := facts[id]
		if money.USD(got.Committed) != card.Committed || money.USD(got.Invoiced) != card.Invoiced ||
			money.USD(got.Paid) != card.Paid || money.USD(got.Accrued) != card.Accrued || got.Count != card.TransactionCount {
			t.Errorf("%s facts %+v, card %+v", id, got, card)
		}
	}
	if facts["L-1004"].POCount != 1 || facts["L-1004"].InvoiceCount != 1 {
		t.Errorf("L-1004 counts = %+v", facts["L-1004"])
	}
}
