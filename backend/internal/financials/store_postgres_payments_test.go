package financials_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/financials"
)

// A dashboard reads the payments of the line items in its scope only: a payment on another market's line item
// (L-1002, plan P-1002 of team T-1002) or an invoice is never read (036 FR-002, D-356).
func TestPostgresStorePaymentsOnLineItemsScopeIsolation_036_FR002(t *testing.T) {
	t.Parallel()
	db, s, _ := pgStore(t)
	payment := func(id, lineItemID string) financials.Transaction {
		p := pgTransaction(id, lineItemID)
		p.Type = financials.TypePayment
		return p
	}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutTransaction(ctx, payment("FT-1005", "L-1001")), s.PutTransaction(ctx, payment("FT-1006", "L-1002")))
	})
	tests := []struct {
		name        string
		lineItemIDs []string
		want        []string
	}{
		{name: "the line item of team T-1001's plan", lineItemIDs: []string{"L-1001"}, want: []string{"FT-1005"}},
		{name: "both markets' line items", lineItemIDs: []string{"L-1001", "L-1002"}, want: []string{"FT-1005", "FT-1006"}},
		{name: "none", lineItemIDs: []string{}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			read, err := s.PaymentsOnLineItems(t.Context(), tt.lineItemIDs)
			if err != nil {
				t.Fatalf("PaymentsOnLineItems() error: %v", err)
			}
			ids := []string{}
			for _, transaction := range read {
				ids = append(ids, transaction.ID)
			}
			if diff := cmp.Diff(tt.want, ids); diff != "" {
				t.Errorf("PaymentsOnLineItems(%v) mismatch (-want +got):\n%s", tt.lineItemIDs, diff)
			}
		})
	}
}
