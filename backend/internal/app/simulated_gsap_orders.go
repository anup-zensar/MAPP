package app

import (
	"context"
	"unicode/utf8"

	"mapp/internal/financials"
	"mapp/internal/platform/actor"
	"mapp/internal/purchaserequest"
	"mapp/internal/simulatedgsap"
)

const (
	simulatedOrderFeed = "Simulated GSAP purchase order"
	maxGSAPDescription = 200
)

// simulatedLedger implements simulatedgsap.Ledger over feature 015's applier (033 data-model §2.4): each PO line
// a purchase-order transaction keyed by PO number and line, linked by the HX key.
type simulatedLedger struct {
	financials *financials.Service
}

func (x simulatedLedger) BookPurchaseOrder(ctx context.Context, a actor.Actor, order simulatedgsap.BookedOrder) error {
	rows := make([]financials.Fields, 0, len(order.Lines))
	description := order.Description
	if utf8.RuneCountInString(description) > maxGSAPDescription {
		description = string([]rune(description)[:maxGSAPDescription])
	}
	for _, l := range order.Lines {
		rows = append(rows, financials.Fields{
			Type: financials.TypePurchaseOrder, DocumentNumber: order.PONumber, LineNumber: l.POLine, PONumber: order.PONumber,
			PostingDate: l.PostingDate, Amount: l.Amount, Currency: l.Currency, VendorCode: order.VendorCode,
			VendorName: order.Vendor, Description: description, SwbsCode: l.SwbsCode, TrackingKey: l.HXKey,
		})
	}
	_, err := x.financials.RecordSimulatedGSAP(ctx, a, financials.SimulatedDelivery{
		FeedName: simulatedOrderFeed, FileName: order.PurchaseRequestID, Rows: rows,
	})
	return err
}

func (w *wiring) simulatedOrders() *simulatedgsap.Orders {
	return simulatedgsap.NewOrders(w.generator, simulatedLedger{financials: w.financials}, w.now)
}

func (w *wiring) simulatedOrdersHandler(prs *purchaserequest.Service) routeRegistrar {
	return simulatedgsap.NewPurchaseOrderHandler(prs)
}
