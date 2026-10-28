// Package simulatedgsap is the simulated GSAP's purchasing side (feature 033, ADR-0016): it answers a purchase
// request routed to the requisitioner with a GSAP PR number and a purchase order, and books the PO lines.
package simulatedgsap

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/purchaserequest"
)

type Numbers interface {
	NextNumber(ctx context.Context, prefix string) (int, error)
}

type Ledger interface {
	BookPurchaseOrder(ctx context.Context, a actor.Actor, order BookedOrder) error
}

type BookedOrder struct {
	PurchaseRequestID  string
	PONumber           string
	VendorCode, Vendor string
	Description        string
	Lines              []BookedLine
}

type BookedLine struct {
	POLine, Currency, SwbsCode, HXKey string
	Amount                            money.Amount
	PostingDate                       date.Date
}

type PurchaseRequests interface {
	CreateGSAPOrder(ctx context.Context, a actor.Actor, id string, version int) (purchaserequest.View, error)
}
