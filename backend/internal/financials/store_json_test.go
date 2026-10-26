package financials

import (
	"testing"

	"mapp/internal/platform/actor"
)

// The stored data keeps the invariants of data model §1.
func TestJSONRepository_Invariants_FR001(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	keys := map[key]bool{}
	for _, tr := range f.transactions(t) {
		if keys[tr.key()] {
			t.Errorf("%s repeats key %+v", tr.ID, tr.key())
		}
		keys[tr.key()] = true
		checkInvariants(t, tr)
	}
}

// checkInvariants checks the rules of data model §1 that a transaction keeps on its own.
func checkInvariants(t *testing.T, tr Transaction) {
	t.Helper()
	if tr.Type == TypePurchaseOrder && (tr.PONumber != tr.DocumentNumber || tr.POLineNumber != "") {
		t.Errorf("%s: a PO refers to itself", tr.ID)
	}
	if (tr.AccrualMonth != "") != (tr.Type == TypeAccrual) {
		t.Errorf("%s: accrual month %q on a %s", tr.ID, tr.AccrualMonth, tr.Type)
	}
	if (tr.LineItemID == "") != (tr.MatchedBy == MatchedByNone) || (tr.LineItemID == "") != (tr.NotLinkedReason != "") {
		t.Errorf("%s: line item %q, matched by %q, reason %q", tr.ID, tr.LineItemID, tr.MatchedBy, tr.NotLinkedReason)
	}
	if (tr.ManualLink != nil) != (tr.MatchedBy == MatchedByManual) {
		t.Errorf("%s: manual link %v, matched by %q", tr.ID, tr.ManualLink, tr.MatchedBy)
	}
}

// The demo feed holds the 11 rows of data model §6 in file order, POs first, and loading them gives
// FT-1001 to FT-1011 linked as the table says, and one import FIM-1001 (FR-031).
func TestDemoFeed_LoadsDataModelSix_FR031(t *testing.T) {
	t.Parallel()
	rows, problems, unused, err := DemoFeed{}.Read(t.Context(), Delivery{})
	if err != nil || len(rows) != 11 || len(problems) != 0 || len(unused) != 0 {
		t.Fatalf("Read = %d rows, %v, %v, %v", len(rows), problems, unused, err)
	}
	if rows[0].Row != 2 || rows[10].Row != 12 || rows[3].Type != TypePurchaseOrder || rows[4].Type != TypeInvoice {
		t.Errorf("rows out of order: %+v", rows)
	}

	f := newFixture(t)
	want := []struct {
		id, name, lineItem string
		by                 MatchedBy
	}{
		{"FT-1001", "PO 4500001001 / 10", "L-1003", MatchedByTrackingKey},
		{"FT-1002", "PO 4500001002 / 10", "L-1004", MatchedByTrackingKey},
		{"FT-1003", "PO 4500001003 / 10", "L-1006", MatchedByTrackingKey},
		{"FT-1004", "PO 4500001004 / 10", "", MatchedByNone},
		{"FT-1005", "Invoice 5100002001", "L-1003", MatchedByPONumber},
		{"FT-1006", "Invoice 5100002002", "L-1004", MatchedByPONumber},
		{"FT-1007", "Invoice 5100002003", "L-1006", MatchedByPONumber},
		{"FT-1008", "Payment 1500003001", "L-1003", MatchedByPONumber},
		{"FT-1009", "Payment 1500003002", "L-1006", MatchedBySwbsCode},
		{"FT-1010", "Accrual 9100004001", "L-1004", MatchedByPONumber},
		{"FT-1011", "Accrual 9100004002", "L-1004", MatchedByPONumber},
	}
	stored := f.transactions(t)
	if len(stored) != len(want) {
		t.Fatalf("%d transactions, want 11", len(stored))
	}
	for i, w := range want {
		got := stored[i]
		if got.ID != w.id || got.Name() != w.name || got.LineItemID != w.lineItem || got.MatchedBy != w.by ||
			got.ImportID != "FIM-1001" || got.CreatedBy != actor.System {
			t.Errorf("%s = %+v, want %+v", got.ID, got, w)
		}
	}
	if reason := stored[3].NotLinkedReason; reason != "No line item for HX key HX1099 — check the tracking key in GSAP." {
		t.Errorf("FT-1004 reason = %q", reason)
	}
	page, err := f.service.Imports(t.Context(), alex)
	if err != nil || page.Total != 1 {
		t.Fatalf("Imports = %+v, %v", page, err)
	}
	demo := page.Items[0]
	if demo.ID != "FIM-1001" || demo.Source != SourceDemo || demo.FeedName != "Demo data (simulated GSAP)" || demo.FileName != "" ||
		demo.Counts != (ImportCounts{Rows: 11, New: 11, NotLinked: 1}) {
		t.Errorf("demo import = %+v", demo)
	}
	if entries := f.entries(t); len(entries) != 0 {
		t.Errorf("the demo load wrote %d log entries, want none (D-188)", len(entries))
	}
}
