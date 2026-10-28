package simulatedgsap

import (
	"context"
	"fmt"
	"strconv"

	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/purchaserequest"
)

const (
	poLineStatus  = "Released"
	poLineStep    = 10
	gsapPRBase    = 10_000_000
	purchaseBase  = 4_500_000_000
	numberPattern = "SIM-%010d"
)

// Orders answers a routed purchase request as GSAP would once the requisitioner typed it:
// a GSAP PR number and a PO number once, a Released PO line per line sent dated today, booked as GSAP data.
type Orders struct {
	numbers Numbers
	ledger  Ledger
	now     clock.Clock
}

var _ purchaserequest.GSAP = (*Orders)(nil)

func NewOrders(numbers Numbers, ledger Ledger, now clock.Clock) *Orders {
	return &Orders{numbers: numbers, ledger: ledger, now: now}
}

func (o *Orders) CreatePurchaseOrder(ctx context.Context, pr purchaserequest.RoutedPR) (purchaserequest.GSAPOrder, error) {
	today := date.Today(o.now())
	order := purchaserequest.GSAPOrder{Source: purchaserequest.SourceSimulatedGSAP, GSAPPRNumber: pr.GSAPPRNumber,
		GSAPPRDate: today, PONumber: pr.PONumber}
	var err error
	if order.GSAPPRNumber == "" {
		if order.GSAPPRNumber, err = o.number(ctx, ids.PrefixSimulatedGSAPPR, gsapPRBase); err != nil {
			return purchaserequest.GSAPOrder{}, err
		}
	}
	if order.PONumber == "" {
		if order.PONumber, err = o.number(ctx, ids.PrefixSimulatedPO, purchaseBase); err != nil {
			return purchaserequest.GSAPOrder{}, err
		}
	}
	order.Lines = answers(pr, today)
	if err = o.ledger.BookPurchaseOrder(ctx, pr.RequestedBy, booked(pr, order)); err != nil {
		return purchaserequest.GSAPOrder{}, err
	}
	return order, nil
}

func (o *Orders) number(ctx context.Context, prefix string, base int) (string, error) {
	n, err := o.numbers.NextNumber(ctx, prefix)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(numberPattern, base+n), nil
}

// answers gives each line sent its PO line: the one it has, or the next after the highest (10, 20, 30 …).
func answers(pr purchaserequest.RoutedPR, today date.Date) []purchaserequest.OrderLine {
	next := pr.LastPOLine
	lines := make([]purchaserequest.OrderLine, 0, len(pr.Lines))
	for _, l := range pr.Lines {
		poLine := l.POLine
		if poLine == "" {
			next = (next/poLineStep + 1) * poLineStep
			poLine = strconv.Itoa(next)
		}
		lines = append(lines, purchaserequest.OrderLine{PRLineID: l.ID, POLine: poLine, Status: poLineStatus, Date: today,
			Value: l.Amount})
	}
	return lines
}

func booked(pr purchaserequest.RoutedPR, order purchaserequest.GSAPOrder) BookedOrder {
	b := BookedOrder{PurchaseRequestID: pr.ID, PONumber: order.PONumber, VendorCode: pr.VendorCode, Vendor: pr.VendorName,
		Description: pr.ShortText, Lines: make([]BookedLine, 0, len(pr.Lines))}
	for i, l := range pr.Lines {
		b.Lines = append(b.Lines, BookedLine{POLine: order.Lines[i].POLine, Currency: l.Currency, SwbsCode: l.SwbsCode,
			HXKey: l.HXKey, Amount: l.Amount, PostingDate: order.Lines[i].Date})
	}
	return b
}
