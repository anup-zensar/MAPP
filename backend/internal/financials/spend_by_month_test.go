package financials

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/date"
)

// Only purchase orders and payments of the line items asked for count, by posting month, at the current rate; a
// currency no longer in Settings is left out, as Committed and Paid leave it out (D-257, D-358).
func TestSpendByMonth_036_FR011_FR014(t *testing.T) {
	t.Parallel()
	f := newEmptyFixture(t)
	transaction := func(id string, typ Type, lineItemID, day, amount, currency string) Transaction {
		return Transaction{ID: id, LineItemID: lineItemID, Fields: Fields{Type: typ, PostingDate: date.MustParse(day),
			Amount: amountOf(t, amount), Currency: currency}}
	}
	f.write(t, func(ctx context.Context) error {
		return f.service.repo.AddTransactions(ctx, []Transaction{
			transaction("FT-1", TypePayment, "L-1003", "2026-03-05", "100.00", "USD"),
			transaction("FT-2", TypePayment, "L-1003", "2026-03-20", "92.00", "EUR"),
			transaction("FT-3", TypePurchaseOrder, "L-1003", "2025-12-01", "300.00", "USD"),
			transaction("FT-4", TypeInvoice, "L-1003", "2026-03-06", "500.00", "USD"),
			transaction("FT-5", TypePayment, "", "2026-03-06", "700.00", "USD"),
			transaction("FT-6", TypePayment, "L-1004", "2026-03-06", "9.00", "XYZ"),
			transaction("FT-7", TypePayment, "L-1005", "2026-03-06", "40.00", "USD"),
		})
	})

	got, err := f.service.SpendByMonth(t.Context(), priya, []string{"L-1003", "L-1004"})

	if err != nil {
		t.Fatalf("SpendByMonth: %v", err)
	}
	texts := map[string]string{}
	for lineItem, months := range got {
		for month, facts := range months {
			texts[lineItem+" "+month] = "committed " + facts.Committed.String() + ", paid " + facts.Paid.String()
		}
	}
	want := map[string]string{
		"L-1003 2025-12": "committed 300.00, paid 0.00",
		"L-1003 2026-03": "committed 0.00, paid 200.00",
	}
	if diff := cmp.Diff(want, texts); diff != "" {
		t.Errorf("SpendByMonth mismatch (-want +got):\n%s", diff)
	}
}
