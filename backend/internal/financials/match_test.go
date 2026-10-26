package financials

import (
	"testing"
)

// The matching rules of FR-007 (data model §4.2), on transactions built in memory.

// tx builds a transaction of type typ with document number doc and the changes of with.
func tx(id string, typ Type, doc string, with func(*Transaction)) Transaction {
	t := Transaction{ID: id, Fields: Fields{Type: typ, DocumentNumber: doc, Currency: "USD"}}
	if typ == TypePurchaseOrder {
		t.PONumber = doc
	}
	if with != nil {
		with(&t)
	}
	return t
}

// linkedTo marks a transaction as linked to lineItemID by rule by.
func linkedTo(lineItemID string, by MatchedBy) func(*Transaction) {
	return func(t *Transaction) { t.LineItemID, t.MatchedBy = lineItemID, by }
}

// liveItems are the line items of the tests: L-1003, L-1004, L-1006 and L-1007 are live; L-1005 is in the
// recycle bin, which matching treats as stored.
var liveItems = storedSet{"L-1003": true, "L-1004": true, "L-1005": true, "L-1006": true, "L-1007": true}

// The first rule that gives exactly one line item wins: line-item ID, then HX key, then PO number, then
// SWBS code.
func TestMatch_RuleOrder_FR007(t *testing.T) {
	t.Parallel()
	x := newIndex([]Transaction{
		tx("FT-1001", TypePurchaseOrder, "4500001001", func(t *Transaction) {
			t.LineNumber, t.SwbsCode = "10", "GB01-M26-0006"
			linkedTo("L-1003", MatchedByTrackingKey)(t)
		}),
	})
	cases := []struct {
		name string
		t    Transaction
		want link
	}{
		{"line item ID before HX key", tx("FT-2", TypeInvoice, "5100", func(t *Transaction) {
			t.SentLineItemID, t.TrackingKey, t.PONumber = "L-1006", "HX1004", "4500001001"
		}), link{lineItemID: "L-1006", by: MatchedByLineItemID}},
		{"HX key when the line item ID is unknown", tx("FT-3", TypeInvoice, "5101", func(t *Transaction) {
			t.SentLineItemID, t.TrackingKey = "L-9999", "HX1004"
		}), link{lineItemID: "L-1004", by: MatchedByTrackingKey}},
		{"PO number before SWBS code", tx("FT-4", TypeInvoice, "5102", func(t *Transaction) {
			t.PONumber, t.SwbsCode = "4500001001", "GB01-M26-0099"
		}), link{lineItemID: "L-1003", by: MatchedByPONumber}},
		{"PO number with its PO line", tx("FT-5", TypePayment, "1500", func(t *Transaction) {
			t.PONumber, t.POLineNumber = "4500001001", "10"
		}), link{lineItemID: "L-1003", by: MatchedByPONumber}},
		{"SWBS code", tx("FT-6", TypePayment, "1501", func(t *Transaction) {
			t.SwbsCode = "GB01-M26-0006"
		}), link{lineItemID: "L-1003", by: MatchedBySwbsCode}},
		{"a line item in the recycle bin still matches", tx("FT-7", TypeInvoice, "5103", func(t *Transaction) {
			t.TrackingKey = "HX1005"
		}), link{lineItemID: "L-1005", by: MatchedByTrackingKey}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := match(&c.t, x, liveItems); got != c.want {
				t.Errorf("match = %+v, want %+v", got, c.want)
			}
		})
	}
}

// A purchase order never matches by its own PO number: it is the PO others refer to.
func TestMatch_POsDoNotMatchByPONumber_FR007(t *testing.T) {
	t.Parallel()
	x := newIndex([]Transaction{tx("FT-1001", TypePurchaseOrder, "4500001001", linkedTo("L-1003", MatchedByTrackingKey))})
	po := tx("FT-1002", TypePurchaseOrder, "4500001001", func(t *Transaction) { t.LineNumber = "20" })
	if got := match(&po, x, liveItems); got.lineItemID != "" || got.reason != reasonNoReference {
		t.Errorf("match = %+v, want not linked without reference", got)
	}
}

// A transaction no rule links carries the reason of the first key it has (data model §4.2).
func TestMatch_NotLinkedReasons_FR007(t *testing.T) {
	t.Parallel()
	x := newIndex([]Transaction{
		tx("FT-1001", TypePurchaseOrder, "4500001003", func(t *Transaction) {
			t.LineNumber, t.SwbsCode = "10", "GB01-M26-0006"
			linkedTo("L-1006", MatchedByTrackingKey)(t)
		}),
		tx("FT-1002", TypePurchaseOrder, "4500001003", func(t *Transaction) {
			t.LineNumber, t.SwbsCode = "20", "GB01-M26-0006"
			linkedTo("L-1003", MatchedByTrackingKey)(t)
		}),
		tx("FT-1003", TypePurchaseOrder, "4500001009", func(t *Transaction) { t.LineNumber = "10" }),
	})
	cases := []struct {
		name string
		t    Transaction
		want string
	}{
		{"unknown line item ID", tx("FT-2", TypeInvoice, "5100", func(t *Transaction) {
			t.SentLineItemID, t.TrackingKey = "L-1099", "HX1099"
		}), "No line item L-1099 — check the line item ID in GSAP."},
		{"unknown HX key", tx("FT-3", TypePurchaseOrder, "4500001004", func(t *Transaction) { t.TrackingKey = "HX1099" }),
			"No line item for HX key HX1099 — check the tracking key in GSAP."},
		{"PO not linked", tx("FT-4", TypeInvoice, "5101", func(t *Transaction) { t.PONumber = "4500001009" }),
			"PO 4500001009 isn't linked to a line item yet."},
		{"unknown PO", tx("FT-5", TypeInvoice, "5102", func(t *Transaction) { t.PONumber = "4500009999" }),
			"PO 4500009999 isn't linked to a line item yet."},
		{"PO on several line items", tx("FT-6", TypeInvoice, "5103", func(t *Transaction) { t.PONumber = "4500001003" }),
			"PO 4500001003 has lines on several line items — the row needs its PO line."},
		{"SWBS code on no linked transaction", tx("FT-7", TypePayment, "1500", func(t *Transaction) { t.SwbsCode = "GB01-M26-0099" }),
			"SWBS code GB01-M26-0099 isn't on any linked transaction yet."},
		{"SWBS code on several line items", tx("FT-8", TypePayment, "1501", func(t *Transaction) { t.SwbsCode = "GB01-M26-0006" }),
			"SWBS code GB01-M26-0006 is on transactions of several line items."},
		{"no reference", tx("FT-9", TypePayment, "1502", nil),
			"No line item reference (line item ID, HX key, PO number or SWBS code)."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := match(&c.t, x, liveItems)
			if got.lineItemID != "" || got.by != MatchedByNone || got.reason != c.want {
				t.Errorf("match = %+v, want not linked: %q", got, c.want)
			}
		})
	}
}

// The PO line settles a PO with lines on several line items; a purged line item (neither live nor in the
// bin) does not match.
func TestMatch_AmbiguousPOWithItsLineAndPurgedLineItem_FR007(t *testing.T) {
	t.Parallel()
	x := newIndex([]Transaction{
		tx("FT-1001", TypePurchaseOrder, "4500001003", func(t *Transaction) {
			t.LineNumber = "10"
			linkedTo("L-1006", MatchedByTrackingKey)(t)
		}),
		tx("FT-1002", TypePurchaseOrder, "4500001003", func(t *Transaction) {
			t.LineNumber = "20"
			linkedTo("L-1003", MatchedByTrackingKey)(t)
		}),
	})
	invoice := tx("FT-2", TypeInvoice, "5100", func(t *Transaction) { t.PONumber, t.POLineNumber = "4500001003", "20" })
	if got := match(&invoice, x, liveItems); got != (link{lineItemID: "L-1003", by: MatchedByPONumber}) {
		t.Errorf("match = %+v, want L-1003 by PO number", got)
	}
	purged := tx("FT-3", TypeInvoice, "5101", func(t *Transaction) { t.TrackingKey = "HX1001" })
	if got := match(&purged, x, liveItems); got.lineItemID != "" {
		t.Errorf("match = %+v, want not linked", got)
	}
}

// relink never changes a manual link, and reports whether anything changed.
func TestRelink_KeepsManualLinks_FR007(t *testing.T) {
	t.Parallel()
	x := newIndex(nil)
	manual := tx("FT-1", TypePurchaseOrder, "4500001004", func(t *Transaction) {
		t.TrackingKey = "HX1004"
		linkedTo("L-1007", MatchedByManual)(t)
		t.ManualLink = &ManualLink{By: alex, At: start}
	})
	if relink(&manual, x, liveItems) || manual.LineItemID != "L-1007" || manual.ManualLink == nil {
		t.Errorf("manual link changed: %+v", manual)
	}
	rule := tx("FT-2", TypePurchaseOrder, "4500001005", func(t *Transaction) { t.TrackingKey = "HX1004" })
	if !relink(&rule, x, liveItems) || rule.LineItemID != "L-1004" || relink(&rule, x, liveItems) {
		t.Errorf("relink = %+v, want linked once, then unchanged", rule)
	}
}
