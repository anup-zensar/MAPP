package purchaserequest

import (
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

func amount(t *testing.T, text string) money.Amount {
	t.Helper()
	a, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("amount %q: %v", text, err)
	}
	return a
}

func TestHXKey(t *testing.T) {
	t.Parallel()
	for id, want := range map[string]string{"L-1006": "HX1006", "L-12": "HX12", "L-1003": "HX1003"} {
		if got := HXKey(id); got != want {
			t.Errorf("032 FR-020 HXKey(%s) = %s, want %s", id, got, want)
		}
	}
}

func TestGSAPPending(t *testing.T) {
	t.Parallel()
	value := money.Money{Amount: amount(t, "9200.00"), Currency: "EUR"}
	other := money.Money{Amount: amount(t, "9200.00"), Currency: "USD"}
	tests := []struct {
		name string
		po   PurchaseOrderLine
		want bool
	}{
		{name: "032 FR-032 no PO line yet", want: true},
		{name: "032 FR-032 PO value matches", po: PurchaseOrderLine{Number: "SIM-1", Line: "10", Value: &value}},
		{name: "032 FR-032 amount changed", po: PurchaseOrderLine{Number: "SIM-1", Line: "10",
			Value: &money.Money{Amount: amount(t, "8000.00"), Currency: "EUR"}}, want: true},
		{name: "032 FR-032 currency changed", po: PurchaseOrderLine{Number: "SIM-1", Line: "10", Value: &other}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l := Line{Currency: "EUR", Amount: amount(t, "9200.00"), PO: tt.po}
			if got := l.gsapPending(); got != tt.want {
				t.Errorf("gsapPending = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChangedSince(t *testing.T) {
	t.Parallel()
	l := Line{ID: "PRL-1", LineItemID: "L-1006", Currency: "EUR", Amount: amount(t, "9200.00"), DeliveryDate: date.New(2026, 11, 30)}
	pr := PurchaseRequest{Lines: []Line{l}}
	v := pr.snapshot(time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC))
	raised := l
	raised.Amount = amount(t, "9500.00")
	added := Line{ID: "PRL-2"}

	if l.changedSince(v) || !raised.changedSince(v) || !added.changedSince(v) {
		t.Errorf("032 FR-036 changed: same %v, raised %v, added %v", l.changedSince(v), raised.changedSince(v), added.changedSince(v))
	}
}

func TestFormFile_ColumnsAndFormulaSafety(t *testing.T) {
	t.Parallel()
	r := Routing{ID: "PRR-1001", Type: RoutingOriginal, RoutedAt: time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC),
		Recipients: []string{"DE_LUBES"}, Form: []FormRow{{PurchaseRequestID: "PR-1002", LineID: "PRL-1002", HXKey: "HX1006",
			ShortText: "=HYPERLINK(1)", Amount: amount(t, "9200.00"), Currency: "EUR", DeliveryDate: date.New(2026, 11, 30)}}}

	csv, err := formFile(PurchaseRequest{ID: "PR-1002", Name: "PR for Stand build"}, r, FormatCSV)
	if err != nil {
		t.Fatalf("formFile: %v", err)
	}

	text := string(csv.Body)
	if csv.Name != "MAPP PR form – PR-1002 – Original – 27 Oct 2026.csv" || csv.ContentType != "text/csv" {
		t.Errorf("032 FR-031 file %q %q", csv.Name, csv.ContentType)
	}
	if !strings.Contains(text, "PR ID,PR name,PR line ID,HX key,Short text,Vendor,Vendor code,Cost centre,SWBS code,GL code,Amount") ||
		!strings.Contains(text, "PRL-1002,HX1006,'=HYPERLINK(1)") || !strings.Contains(text, "Recipients,DE_LUBES") {
		t.Errorf("032 FR-031 columns, recipients and a formula written as text:\n%s", text)
	}
	if _, err := formFile(PurchaseRequest{ID: "PR-1002"}, r, FormatXLSX); err != nil {
		t.Errorf("xlsx: %v", err)
	}
}

func TestOrderText(t *testing.T) {
	t.Parallel()
	order := GSAPOrder{Source: SourceSimulatedGSAP, GSAPPRNumber: "SIM-0010001001", PONumber: "SIM-4500001001",
		Lines: []OrderLine{{POLine: "10"}}}
	first := orderText(order, []Line{{}}, true)
	topUp := orderText(order, []Line{{PO: PurchaseOrderLine{Line: "10"}}, {}}, false)

	if first != "Simulated GSAP created GSAP PR SIM-0010001001 and PO SIM-4500001001 (1 line)." {
		t.Errorf("033 FR-008 first run %q", first)
	}
	if topUp != "Simulated GSAP updated PO SIM-4500001001: 1 changed, 1 added (top-up)." {
		t.Errorf("033 FR-008 top-up %q", topUp)
	}
}
