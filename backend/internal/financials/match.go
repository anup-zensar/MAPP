package financials

import (
	"fmt"
	"regexp"
)

// hxKey reads the line-item number of an HX key: "HX1003" is line item L-1003 (D-019, workshop D5).
var hxKey = regexp.MustCompile(`^HX([0-9]+)$`)

// link is the outcome of matching: the line item, the rule, or the reason it is not linked.
type link struct {
	lineItemID string
	by         MatchedBy
	reason     string
}

// match links the transaction t to one line item by the first rule that gives exactly one (FR-007, data
// model §4.2):
//
//  1. its line-item ID, when that line item is stored (live or in the recycle bin);
//  2. its HX key, "HX" + the line item's number, when that line item is stored;
//  3. its PO number (with its PO line, when given), when not a PO itself: the one line item of the linked
//     PO lines it refers to;
//  4. its SWBS code: the one line item of the other linked transactions with that code.
//
// Otherwise t is not linked, with the reason of the first key it carries. x holds every transaction known
// so far, those of the current delivery included, so an invoice finds a PO of the same file.
func match(t *Transaction, x *index, items LineItemStates) link {
	if lineItemIDForm.MatchString(t.SentLineItemID) && items.Stored(t.SentLineItemID) {
		return link{lineItemID: t.SentLineItemID, by: MatchedByLineItemID}
	}
	if m := hxKey.FindStringSubmatch(t.TrackingKey); m != nil && items.Stored("L-"+m[1]) {
		return link{lineItemID: "L-" + m[1], by: MatchedByTrackingKey}
	}
	refersToPO := t.Type != TypePurchaseOrder && t.PONumber != ""
	if refersToPO {
		if found := x.poLineItems(t.PONumber, t.POLineNumber); len(found) == 1 {
			return link{lineItemID: found[0], by: MatchedByPONumber}
		}
	}
	if t.SwbsCode != "" {
		if found := x.swbsLineItems(t.SwbsCode, t); len(found) == 1 {
			return link{lineItemID: found[0], by: MatchedBySwbsCode}
		}
	}
	return link{reason: notLinkedReason(t, x, refersToPO)}
}

// notLinkedReason is the reason of the first key t carries, in the order of the rules (data model §4.2).
func notLinkedReason(t *Transaction, x *index, refersToPO bool) string {
	switch {
	case t.SentLineItemID != "":
		return fmt.Sprintf(reasonLineItemID, t.SentLineItemID)
	case t.TrackingKey != "":
		return fmt.Sprintf(reasonTrackingKey, t.TrackingKey)
	case refersToPO:
		if len(x.poLineItems(t.PONumber, t.POLineNumber)) > 1 {
			return fmt.Sprintf(reasonPOSeveral, t.PONumber)
		}
		return fmt.Sprintf(reasonPONotLinked, t.PONumber)
	case t.SwbsCode != "":
		if len(x.swbsLineItems(t.SwbsCode, t)) > 1 {
			return fmt.Sprintf(reasonSwbsSeveral, t.SwbsCode)
		}
		return fmt.Sprintf(reasonSwbsNone, t.SwbsCode)
	}
	return reasonNoReference
}

// relink matches t again, unless a person linked it (a manual link is never changed by an import, FR-007),
// and stores the outcome in t. It reports whether the link or its reason changed.
func relink(t *Transaction, x *index, items LineItemStates) bool {
	if t.MatchedBy == MatchedByManual {
		return false
	}
	l := match(t, x, items)
	if l.lineItemID == t.LineItemID && l.by == t.MatchedBy && l.reason == t.NotLinkedReason {
		return false
	}
	t.LineItemID, t.MatchedBy, t.NotLinkedReason, t.ManualLink = l.lineItemID, l.by, l.reason, nil
	return true
}
