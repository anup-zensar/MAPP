package simulatedgsap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/purchaserequest"
)

type fakeNumbers map[string]int

func (n fakeNumbers) NextNumber(_ context.Context, prefix string) (int, error) {
	n[prefix]++
	return 1000 + n[prefix], nil
}

type fakeLedger struct {
	booked []BookedOrder
	err    error
}

func (l *fakeLedger) BookPurchaseOrder(_ context.Context, _ actor.Actor, order BookedOrder) error {
	l.booked = append(l.booked, order)
	return l.err
}

var runDay = time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

func line(id, poLine, value string) purchaserequest.RoutedLine {
	amount, _ := money.ParseAmount(value)
	return purchaserequest.RoutedLine{ID: id, LineItemID: "L-1006", HXKey: "HX1006", SwbsCode: "SIM-GB01-0001", Currency: "EUR",
		Amount: amount, POLine: poLine}
}

func TestOrders_NumbersAndAnswer(t *testing.T) {
	t.Parallel()
	ledger := &fakeLedger{}
	orders := NewOrders(fakeNumbers{}, ledger, func() time.Time { return runDay })
	pr := purchaserequest.RoutedPR{ID: "PR-1002", VendorName: "Northlight Events GmbH", VendorCode: "V-100101", ShortText: "Stand",
		Lines: []purchaserequest.RoutedLine{line("PRL-1", "", "9200.00"), line("PRL-2", "", "100.00")}}

	order, err := orders.CreatePurchaseOrder(t.Context(), pr)
	if err != nil {
		t.Fatalf("CreatePurchaseOrder: %v", err)
	}

	today := date.New(2026, 10, 27)
	numbers := []any{order.GSAPPRNumber, order.PONumber, order.Source, order.GSAPPRDate}
	if diff := cmp.Diff([]any{"SIM-0010001001", "SIM-4500001001", "simulatedGsap", today}, numbers, cmp.AllowUnexported(date.Date{})); diff != "" {
		t.Errorf("033 FR-002 FR-003 numbers (-want +got):\n%s", diff)
	}
	want := []purchaserequest.OrderLine{
		{PRLineID: "PRL-1", POLine: "10", Status: "Released", Date: today, Value: pr.Lines[0].Amount},
		{PRLineID: "PRL-2", POLine: "20", Status: "Released", Date: today, Value: pr.Lines[1].Amount},
	}
	if diff := cmp.Diff(want, order.Lines, cmp.AllowUnexported(money.Amount{}, date.Date{})); diff != "" {
		t.Errorf("033 FR-003 PO lines (-want +got):\n%s", diff)
	}
	b := ledger.booked[0]
	booked := []string{b.PurchaseRequestID, b.PONumber, b.Vendor, b.Lines[1].POLine, b.Lines[0].HXKey, b.Lines[0].SwbsCode}
	if diff := cmp.Diff([]string{"PR-1002", "SIM-4500001001", "Northlight Events GmbH", "20", "HX1006", "SIM-GB01-0001"}, booked); diff != "" {
		t.Errorf("033 FR-005 booked (-want +got):\n%s", diff)
	}
}

func TestOrders_TopUpKeepsNumbers_033_FR007(t *testing.T) {
	t.Parallel()
	numbers := fakeNumbers{}
	orders := NewOrders(numbers, &fakeLedger{}, func() time.Time { return runDay })
	pr := purchaserequest.RoutedPR{ID: "PR-1002", GSAPPRNumber: "SIM-0010001001", PONumber: "SIM-4500001001", LastPOLine: 20,
		Lines: []purchaserequest.RoutedLine{line("PRL-1", "10", "9500.00"), line("PRL-3", "", "50.00")}}

	order, err := orders.CreatePurchaseOrder(t.Context(), pr)

	if err != nil || order.GSAPPRNumber != "SIM-0010001001" || order.PONumber != "SIM-4500001001" || len(numbers) != 0 {
		t.Fatalf("top-up = %+v, %v; numbers used %v", order, err, numbers)
	}
	if order.Lines[0].POLine != "10" || order.Lines[1].POLine != "30" {
		t.Errorf("033 FR-003 lines kept and continued: %+v", order.Lines)
	}
}

func TestOrders_LedgerRefusalAnswersNothing_033_FR009(t *testing.T) {
	t.Parallel()
	refused := errors.New("currency not set up")
	orders := NewOrders(fakeNumbers{}, &fakeLedger{err: refused}, func() time.Time { return runDay })

	_, err := orders.CreatePurchaseOrder(t.Context(), purchaserequest.RoutedPR{Lines: []purchaserequest.RoutedLine{line("PRL-1", "", "1.00")}})

	if !errors.Is(err, refused) {
		t.Errorf("error = %v, want the ledger's", err)
	}
}
