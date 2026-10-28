package apibench

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mapp/internal/platform/database"
	"mapp/internal/platform/ids"
	"mapp/internal/purchaserequest"
	"mapp/internal/stores"
)

// PurchaseRequests is the demo data with 10,000 purchase-request lines: 2,000 purchase requests of five lines
// each on the Lubricants plan, from 40 vendors (032 SC-007).
const PurchaseRequests Data = "purchaseRequests"

const (
	purchaseRequestsWanted = 2000
	vendorsWanted          = 40
)

func loadPurchaseRequests(tb testing.TB, db *database.DB) {
	tb.Helper()
	loadDemo(tb, db)
	v := &volume{tb: tb, ctx: tb.Context(), set: stores.Postgres(db), started: time.Now()}
	template, found, err := v.set.PurchaseRequests.Get(v.ctx, "PR-1001")
	if err != nil || !found {
		tb.Fatalf("apibench: the demo purchase request PR-1001: found %v, %v", found, err)
	}
	v.write(func(ctx context.Context, next func(string) string) error {
		for n := range purchaseRequestsWanted {
			if err := v.set.PurchaseRequests.Put(ctx, volumePurchaseRequest(template, n, next)); err != nil {
				return err
			}
		}
		return nil
	})
	v.progress("%d purchase requests", purchaseRequestsWanted)
}

func volumePurchaseRequest(template purchaserequest.PurchaseRequest, n int, next func(string) string) purchaserequest.PurchaseRequest {
	pr := template
	pr.ID, pr.PlanID = next(ids.PrefixPurchaseRequest), "P-1001"
	pr.Name, pr.Vendor = fmt.Sprintf("Volume PR %04d", n), fmt.Sprintf("Volume vendor %02d", n%vendorsWanted)
	pr.Lines = nil
	for i, lineItemID := range []string{"L-1001", "L-1002", "L-1003", "L-1004", "L-1005"} {
		l := template.Lines[0]
		l.ID, l.PurchaseRequestID, l.Position, l.LineItemID, l.Currency = next(ids.PrefixPurchaseRequestLine), pr.ID, i+1, lineItemID, usd
		pr.Lines = append(pr.Lines, l)
	}
	return pr
}
