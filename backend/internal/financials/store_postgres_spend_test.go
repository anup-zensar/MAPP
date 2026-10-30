package financials_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/financials"
)

// A dashboard reads the purchase orders and payments of the line items in its scope only: those on another
// market's line item (L-1002, plan P-1002 of team T-1002) and invoices are never read (D-356, D-358).
func TestPostgresStoreSpendOnLineItemsScopeIsolation_036_FR002(t *testing.T) {
	t.Parallel()
	db, s, _ := pgStore(t)
	spend := func(id, lineItemID string, typ financials.Type) financials.Transaction {
		p := pgTransaction(id, lineItemID)
		p.Type = typ
		return p
	}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutTransaction(ctx, spend("FT-1005", "L-1001", financials.TypePayment)),
			s.PutTransaction(ctx, spend("FT-1006", "L-1002", financials.TypePayment)),
			s.PutTransaction(ctx, spend("FT-1007", "L-1001", financials.TypePurchaseOrder)))
	})
	tests := []struct {
		name        string
		lineItemIDs []string
		want        []string
	}{
		{name: "the line item of team T-1001's plan", lineItemIDs: []string{"L-1001"}, want: []string{"FT-1005", "FT-1007"}},
		{name: "both markets' line items", lineItemIDs: []string{"L-1001", "L-1002"},
			want: []string{"FT-1005", "FT-1006", "FT-1007"}},
		{name: "none", lineItemIDs: []string{}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			read, err := s.SpendOnLineItems(t.Context(), tt.lineItemIDs)
			if err != nil {
				t.Fatalf("SpendOnLineItems() error: %v", err)
			}
			ids := []string{}
			for _, transaction := range read {
				ids = append(ids, transaction.ID)
			}
			if diff := cmp.Diff(tt.want, ids); diff != "" {
				t.Errorf("SpendOnLineItems(%v) mismatch (-want +got):\n%s", tt.lineItemIDs, diff)
			}
		})
	}
}
