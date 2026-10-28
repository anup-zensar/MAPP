package financials

import "testing"

func TestFactsByLineItemMonth_FR010_EachTransactionCountsInTheMonth(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	byMonth, err := f.service.FactsByLineItemMonth(t.Context())
	if err != nil {
		t.Fatalf("FactsByLineItemMonth: %v", err)
	}
	whole, err := f.service.FactsByLineItem(t.Context())
	if err != nil {
		t.Fatalf("FactsByLineItem: %v", err)
	}
	l1004 := byMonth["L-1004"]
	if l1004["2026-05"].Committed != amountOf(t, "5000.00") || l1004["2026-09"].Invoiced != amountOf(t, "2500.00") ||
		!l1004["2026-05"].Invoiced.IsZero() {
		t.Errorf("L-1004 by month = %+v", l1004)
	}
	if byMonth["L-1003"]["2026-03"].Paid != whole["L-1003"].Paid || whole["L-1003"].Paid.IsZero() {
		t.Errorf("L-1003 paid in March = %v, whole %v", byMonth["L-1003"]["2026-03"].Paid, whole["L-1003"].Paid)
	}
}

func TestFactsByLineItemMonth_FR010_TheMonthsAddUpToThe(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	byMonth, err := f.service.FactsByLineItemMonth(t.Context())
	if err != nil {
		t.Fatalf("FactsByLineItemMonth: %v", err)
	}
	whole, err := f.service.FactsByLineItem(t.Context())
	if err != nil {
		t.Fatalf("FactsByLineItem: %v", err)
	}
	for id, months := range byMonth {
		var sum Facts
		for _, m := range months {
			sum = sum.plus(m)
		}
		if sum.Committed != whole[id].Committed || sum.Invoiced != whole[id].Invoiced || sum.Paid != whole[id].Paid {
			t.Errorf("%s months %+v, whole %+v", id, sum, whole[id])
		}
	}
	if len(byMonth) != len(whole) {
		t.Errorf("%d line items by month, %d in all", len(byMonth), len(whole))
	}
}
