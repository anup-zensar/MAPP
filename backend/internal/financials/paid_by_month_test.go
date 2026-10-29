package financials

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// Only payments of the line items asked for count, by posting month, at the current rate; a currency no longer in
// Settings is left out, as Paid leaves it out (036 FR-014, D-257, D-356).
func TestPaidByMonth_036_FR014(t *testing.T) {
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
			transaction("FT-3", TypePayment, "L-1003", "2026-04-01", "10.00", "USD"),
			transaction("FT-4", TypeInvoice, "L-1003", "2026-03-06", "500.00", "USD"),
			transaction("FT-5", TypePayment, "", "2026-03-06", "700.00", "USD"),
			transaction("FT-6", TypePayment, "L-1004", "2026-03-06", "9.00", "XYZ"),
			transaction("FT-7", TypePayment, "L-1005", "2026-03-06", "40.00", "USD"),
		})
	})

	got, err := f.service.PaidByMonth(t.Context(), []string{"L-1003", "L-1004"})

	if err != nil {
		t.Fatalf("PaidByMonth: %v", err)
	}
	want := map[string]map[string]money.Amount{
		"L-1003": {"2026-03": amountOf(t, "200.00"), "2026-04": amountOf(t, "10.00")},
	}
	if diff := cmp.Diff(want, got, cmp.Comparer(func(x, y money.Amount) bool { return x.Cmp(y) == 0 })); diff != "" {
		t.Errorf("PaidByMonth mismatch (-want +got):\n%s", diff)
	}
}
