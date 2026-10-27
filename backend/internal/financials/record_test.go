package financials

import (
	"net/http"
	"reflect"
	"strconv"
	"testing"

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
func (w want) check(t *testing.T, f Figures) { //nolint:gocyclo // 11: the amounts, then one case per answer of the row's above column
	t.Helper()
	if !reflect.DeepEqual(f, w.withAmounts(t, f)) {
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

// withAmounts returns f with the amounts and the per cent of the row w, so that comparing it with f compares
// only what the row holds.
func (w want) withAmounts(t *testing.T, f Figures) Figures {
	t.Helper()
	planned, notYet := usd(t, w.planned), usd(t, w.notYet)
	percent, err := strconv.Atoi(w.percent)
	if err != nil {
		t.Fatalf("%s: per cent %q: %v", w.id, w.percent, err)
	}
	expected := f
	expected.PlannedCost, expected.NotYetCommitted, expected.CommittedPercent = &planned, &notYet, &percent
	expected.Committed, expected.Invoiced = usd(t, w.committed), usd(t, w.invoiced)
	expected.Paid, expected.Accrued = usd(t, w.paid), usd(t, w.accrued)
	return expected
}

// Every record of data model §6.1 has its figures; every container's row in its parent's breakdown equals
// its own card (FR-011, FR-012, SC-002, US2).
func TestRecord_DemoFiguresAndRollUp_FR012(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for _, w := range demoFigures {
		v := f.record(t, priya, w.id)
		w.check(t, v.Figures)
		f.checkBreakdownRowsAreCards(t, v)
	}
	counts := map[string]int{"A-1002": 7, "A-1003": 3, "L-1004": 4, "P-1001": 10}
	for id, n := range counts {
		if got := f.record(t, priya, id).Figures.TransactionCount; got != n {
			t.Errorf("%s has %d transactions, want %d", id, got, n)
		}
	}
	checkActivityCard(t, f.record(t, priya, "A-1002"))
	f.checkPlanCard(t)
	if v := f.record(t, priya, "L-1003"); len(v.SwbsCodes) != 1 || v.SwbsCodes[0] != "DE03-M26-0003" {
		t.Errorf("Venue hire SWBS codes = %v", v.SwbsCodes)
	}
}

// checkBreakdownRowsAreCards checks that every breakdown row of the card v equals its record's own card.
func (f *fixture) checkBreakdownRowsAreCards(t *testing.T, v RecordView) {
	t.Helper()
	for _, row := range v.Breakdown {
		child := f.record(t, priya, row.Record.ID)
		if !reflect.DeepEqual(row.Figures, child.Figures) {
			t.Errorf("%s's breakdown row %s = %+v, card %+v", v.Record.ID, row.Record.ID, row.Figures, child.Figures)
		}
	}
}

// checkActivityCard checks the latest transactions, path, breakdown and source of the card of Social
// always-on.
func checkActivityCard(t *testing.T, activity RecordView) { //nolint:gocyclo // 13: the latest transactions, then the path, breakdown and source, in one condition each
	t.Helper()
	if len(activity.Latest) != 5 || activity.Latest[0].ID != "FT-1011" || activity.Latest[4].ID != "FT-1008" ||
		activity.Latest[0].Link.State != StateLinked {
		t.Errorf("latest = %v", rowIDs(activity.Latest))
	}
	if s := activity.Source; len(activity.Path) != 3 || activity.BreakdownLevel == nil || *activity.BreakdownLevel != LevelLineItem ||
		activity.Breakdown[0].Record.ID != "L-1003" || s.FeedName == nil || *s.FeedName != "Demo data (simulated GSAP)" ||
		s.AccrualMonth == nil || *s.AccrualMonth != "2026-09" {
		t.Errorf("path %+v, breakdown %v %v, source %+v", activity.Path, activity.BreakdownLevel, activity.Breakdown, s)
	}
}

// checkPlanCard checks that the plan's card breaks down into its campaigns and has no path, and that a line
// item's card has no breakdown.
func (f *fixture) checkPlanCard(t *testing.T) {
	t.Helper()
	if plan := f.record(t, priya, "P-1001"); plan.BreakdownLevel == nil || *plan.BreakdownLevel != LevelCampaign ||
		plan.Breakdown[0].Record.Name != "Helix launch" || len(plan.Path) != 0 || f.record(t, priya, "L-1006").BreakdownLevel != nil {
		t.Errorf("plan breakdown %v %+v, path %+v; want campaigns, Helix launch first, no path", plan.BreakdownLevel, plan.Breakdown, plan.Path)
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	f := newFixture(t)
	_, err := f.service.Record(t.Context(), priya, "C-9999")
	if e := assertCode(t, err, http.StatusNotFound, errs.CodeNotFound); e.Message != "This campaign no longer exists — someone may have deleted it." {
		t.Errorf("message = %q", e.Message)
	}
}

// A deleted line item's transactions stay linked and leave every figure; restoring it brings them back
// (FR-012, D-249).
func TestRecord_LineItemInTheBinLeavesTheFigures_FR012(t *testing.T) {
	t.Parallel()
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
	_ = assertCode(t, err, http.StatusNotFound, errs.CodeNotFound)

	f.planning.restore(removed)
	demoFigures[4].check(t, f.record(t, priya, "A-1003").Figures)
}

// Copies never copy transactions: a copied activity and its line items show 0.00 (FR-030).
func TestRecord_CopiesHaveNoTransactions_FR030(t *testing.T) {
	t.Parallel()
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
func TestRecord_NoTransactionsShowsZero_FR014(t *testing.T) { //nolint:gocyclo // 12: every figure of two records without transactions, in one condition
	t.Parallel()
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
	t.Parallel()
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
