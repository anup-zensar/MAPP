package financials

import (
	"testing"

	"mapp/internal/platform/money"
)

// The figures of data model §5 (FR-010 – FR-014).

// A transaction converts to USD at its currency's current rate, half up to the cent; USD needs no rate; a
// currency no longer in Settings gives no value (FR-010, D-052, D-257).
func TestUSD_HalfUpPerTransaction_FR010(t *testing.T) {
	t.Parallel()
	cases := []struct {
		amount, currency string
		want             string // "" for none
	}{
		{"9800.00", "EUR", "10652.17"}, // 10,652.1739…
		{"15800.00", "GBP", "20000.00"},
		{"5000.00", "USD", "5000.00"},
		{"-9800.00", "EUR", "-10652.17"},
		{"0.01", "GBP", "0.01"}, // 0.012658… rounds down
		{"500.00", "CHF", ""},
	}
	for _, c := range cases {
		tr := Transaction{Fields: Fields{Amount: amountOf(t, c.amount), Currency: c.currency}}
		got := usdOf(&tr, demoRates())
		switch {
		case c.want == "" && got != nil:
			t.Errorf("%s %s = %s, want none", c.amount, c.currency, got.Format())
		case c.want != "" && (got == nil || *got != amountOf(t, c.want)):
			t.Errorf("%s %s = %v, want %s", c.amount, c.currency, got, c.want)
		}
	}
}

// The committed percentage is a whole number, half up, may exceed 100 and is null without a planned cost
// above 0.00 (FR-011).
func TestPercent_HalfUpAndNull_FR011(t *testing.T) {
	t.Parallel()
	cases := []struct {
		committed, planned string
		want               *int
	}{
		{"125.00", "1000.00", intOf(13)}, // 12.5
		{"124.99", "1000.00", intOf(12)},
		{"10652.17", "10000.00", intOf(107)},
		{"20000.00", "20000.00", intOf(100)},
		{"0.00", "120000.00", intOf(0)},
		{"-125.00", "1000.00", intOf(-13)}, // credits: half away from zero
		{"100.00", "0.00", nil},
	}
	for _, c := range cases {
		got := percentOf(amountOf(t, c.committed), amountOf(t, c.planned))
		if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
			t.Errorf("%s of %s = %v, want %v", c.committed, c.planned, deref(got), deref(c.want))
		}
	}
}

// intOf returns a pointer to n.
func intOf(n int) *int { return &n }

// deref shows an optional int.
func deref(n *int) any {
	if n == nil {
		return "null"
	}
	return *n
}

// Each type adds to its figure; credit notes are negative invoices; only accruals of the latest accrual
// month count as Accrued; a transaction without a USD value is counted but left out of the sums (FR-010,
// D-243, D-244, D-257).
func TestFacts_TypesCreditsAccrualMonth_FR010(t *testing.T) {
	t.Parallel()
	list := []*Transaction{
		{Fields: Fields{Type: TypePurchaseOrder, Amount: amountOf(t, "1000.00"), Currency: "USD"}},
		{Fields: Fields{Type: TypeInvoice, Amount: amountOf(t, "600.00"), Currency: "USD"}},
		{Fields: Fields{Type: TypeInvoice, Amount: amountOf(t, "-100.00"), Currency: "USD"}}, // a credit note
		{Fields: Fields{Type: TypePayment, Amount: amountOf(t, "400.00"), Currency: "USD"}},
		{Fields: Fields{Type: TypeAccrual, AccrualMonth: "2026-08", Amount: amountOf(t, "300.00"), Currency: "USD"}},
		{Fields: Fields{Type: TypeAccrual, AccrualMonth: "2026-09", Amount: amountOf(t, "200.00"), Currency: "USD"}},
		{Fields: Fields{Type: TypePurchaseOrder, Amount: amountOf(t, "50.00"), Currency: "CHF"}}, // not in Settings
	}
	month := latestAccrualMonth(list)
	if month != "2026-09" {
		t.Fatalf("accrual month = %q, want 2026-09", month)
	}
	var f Facts
	for _, tr := range list {
		f.add(tr, usdOf(tr, demoRates()), month)
	}
	want := Facts{
		Committed: amountOf(t, "1000.00"), Invoiced: amountOf(t, "500.00"), Paid: amountOf(t, "400.00"),
		Accrued: amountOf(t, "200.00"), Count: 7, POCount: 2, InvoiceCount: 2,
	}
	if f != want {
		t.Errorf("facts = %+v, want %+v", f, want)
	}
	if latestAccrualMonth(list[:4]) != "" {
		t.Errorf("without accruals the month is %q, want none", latestAccrualMonth(list[:4]))
	}
}

// A record without a planned cost has no not-yet-committed figure and no percentage; a planned cost of
// 0.00 gives a figure but no percentage; above planned is an exact comparison (FR-011, FR-013).
func TestFiguresOf_NullCasesAndAbovePlanned_FR013(t *testing.T) {
	t.Parallel()
	facts := Facts{Committed: amountOf(t, "100.00"), Count: 1}
	none := figuresOf(facts, nil, false, 0, nil, money.DisplayUSD)
	if none.PlannedCost != nil || none.NotYetCommitted != nil || none.CommittedPercent != nil || none.CommittedAbovePlanned {
		t.Errorf("without a planned cost = %+v", none)
	}
	zero := amountOf(t, "0.00")
	atZero := figuresOf(facts, &zero, true, 0, nil, money.DisplayUSD)
	if atZero.NotYetCommitted == nil || *atZero.NotYetCommitted != usd(t, "-100.00") || atZero.CommittedPercent != nil {
		t.Errorf("planned 0.00 = %+v", atZero)
	}

	tree := buildTree([]Node{
		node(LevelPlan, "P-1", "", "P-1", "Plan", 0, ""),
		node(LevelCampaign, "C-1", "P-1", "P-1", "Campaign", 0, ""),
		node(LevelInitiative, "I-1", "C-1", "P-1", "Initiative", 0, ""),
		node(LevelActivity, "A-1", "I-1", "P-1", "Activity", 0, "1000.00"),
		node(LevelLineItem, "L-2", "A-1", "P-1", "Exactly planned", 0, "500.00"),
		node(LevelLineItem, "L-3", "A-1", "P-1", "One cent over", 1, "200.00"),
		node(LevelLineItem, "L-4", "A-1", "P-1", "No planned cost", 2, ""),
	}, map[string]Facts{
		"L-2": {Committed: amountOf(t, "500.00")}, "L-3": {Committed: amountOf(t, "200.01")},
		"L-4": {Committed: amountOf(t, "50.00")},
	})
	l2, l3, l4, activity := tree.byID["L-2"], tree.byID["L-3"], tree.byID["L-4"], tree.byID["A-1"]
	if l2.above || !l3.above || l4.above {
		t.Errorf("above: L-2 %v, L-3 %v, L-4 %v; want only L-3", l2.above, l3.above, l4.above)
	}
	if activity.above || activity.lineItemsAbove != 1 || activity.firstAbove != l3 || !activity.activityAbove() {
		t.Errorf("activity above %v, line items above %d, first %v", activity.above, activity.lineItemsAbove, activity.firstAbove)
	}
	plan := tree.byID["P-1"].figures(money.DisplayUSD)
	if plan.LineItemsAbovePlanned != 1 || plan.FirstLineItemAbovePlanned == nil || plan.FirstLineItemAbovePlanned.ID != "L-3" ||
		*plan.PlannedCost != usd(t, "1000.00") || plan.Committed != usd(t, "750.01") {
		t.Errorf("plan = %+v", plan)
	}
	if l3.figures(money.DisplayUSD).LineItemsAbovePlanned != 0 {
		t.Errorf("a line item counts no line items below it")
	}
}

// A container's planned cost is the roll-up of its activities (D-090): a container with no activity
// planned cost has none; an activity's own planned cost wins over its line items'.
func TestTree_PlannedCostRollUp_FR012(t *testing.T) {
	t.Parallel()
	tree := buildTree([]Node{
		node(LevelPlan, "P-1", "", "P-1", "Plan", 0, ""),
		node(LevelCampaign, "C-1", "P-1", "P-1", "With cost", 0, ""),
		node(LevelCampaign, "C-2", "P-1", "P-1", "Without cost", 1, ""),
		node(LevelInitiative, "I-1", "C-1", "P-1", "Initiative", 0, ""),
		node(LevelInitiative, "I-2", "C-2", "P-1", "Empty", 0, ""),
		node(LevelActivity, "A-1", "I-1", "P-1", "Typed", 0, "300.00"),
		node(LevelActivity, "A-2", "I-1", "P-1", "Untyped", 1, ""),
		node(LevelLineItem, "L-1", "A-1", "P-1", "Item", 0, "900.00"),
	}, nil)
	if p := tree.byID["A-1"].planned; p == nil || *p != amountOf(t, "300.00") {
		t.Errorf("activity planned = %v, want its own 300.00", p)
	}
	if p := tree.byID["C-1"].planned; p == nil || *p != amountOf(t, "300.00") {
		t.Errorf("campaign planned = %v, want 300.00", p)
	}
	if tree.byID["C-2"].planned != nil || tree.byID["A-2"].planned != nil {
		t.Errorf("no planned cost should stay none")
	}
	if p := tree.byID["P-1"].planned; p == nil || *p != amountOf(t, "300.00") {
		t.Errorf("plan planned = %v, want 300.00", p)
	}
	if children := tree.byID["I-1"].children; children[0].ID != "A-1" || children[1].ID != "A-2" {
		t.Errorf("children out of order")
	}
}

// Every USD figure is converted once to the display currency; sums are taken in USD before converting
// (data model §5, D-061).
func TestFiguresOf_OneDisplayConversion_FR011(t *testing.T) {
	t.Parallel()
	eur := money.DisplayIn("EUR", "0.92")
	planned := amountOf(t, "10000.00")
	got := figuresOf(Facts{Committed: amountOf(t, "10652.17")}, &planned, true, 0, nil, eur)
	want := func(text string) money.Money { return money.Money{Amount: amountOf(t, text), Currency: "EUR"} }
	if got.Committed != want("9800.00") || *got.PlannedCost != want("9200.00") || *got.NotYetCommitted != want("-600.00") {
		t.Errorf("figures in EUR = %+v", got)
	}
}
