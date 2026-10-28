package financials

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/date"
)

func simulatedRow(t *testing.T, currency, value string) Fields {
	t.Helper()
	return Fields{Type: TypePurchaseOrder, DocumentNumber: "SIM-4500001001", LineNumber: "10", PONumber: "SIM-4500001001",
		PostingDate: date.New(2026, 10, 20), Amount: amountOf(t, value), Currency: currency, VendorName: "Northlight Events GmbH",
		TrackingKey: "HX1006"}
}

func TestRecordSimulatedGSAP_LinkedAndToppedUp_033_FR005_FR007_FR011(t *testing.T) {
	t.Parallel()
	f := newEmptyFixture(t)
	delivery := SimulatedDelivery{FeedName: "Simulated GSAP purchase order", FileName: "PR-1002",
		Rows: []Fields{simulatedRow(t, "EUR", "9200.00")}}

	first, err := f.service.RecordSimulatedGSAP(t.Context(), alex, delivery)
	if err != nil {
		t.Fatalf("RecordSimulatedGSAP: %v", err)
	}
	delivery.Rows = []Fields{simulatedRow(t, "EUR", "9500.00")}
	second, err := f.service.RecordSimulatedGSAP(t.Context(), alex, delivery)
	if err != nil {
		t.Fatalf("top-up: %v", err)
	}

	got := []any{first.Source, first.FileName, first.New, second.Updated}
	for _, tr := range f.transactions(t) {
		got = append(got, tr.LineItemID+" "+string(tr.MatchedBy)+" "+tr.Amount.String())
	}
	want := []any{SourceSimulatedGSAP, "PR-1002", 1, 1, "L-1006 " + string(MatchedByTrackingKey) + " 9500.00"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("imports and transactions (-want +got):\n%s", diff)
	}
}

func TestRecordSimulatedGSAP_RefusedRowBooksNothing_033_FR009(t *testing.T) {
	t.Parallel()
	f := newEmptyFixture(t)
	_, err := f.service.RecordSimulatedGSAP(t.Context(), alex, SimulatedDelivery{FileName: "PR-1002",
		Rows: []Fields{simulatedRow(t, "XYZ", "1.00")}})
	if err == nil || len(f.transactions(t)) != 0 {
		t.Errorf("error %v, transactions %v", err, f.transactions(t))
	}
}

func TestRecordSimulatedGSAP_WithoutImport_033_FR010(t *testing.T) {
	t.Parallel()
	f := newEmptyFixture(t)
	f.access.noImport = true
	if _, err := f.service.RecordSimulatedGSAP(t.Context(), priya, SimulatedDelivery{}); err == nil {
		t.Errorf("refusal expected")
	}
}
