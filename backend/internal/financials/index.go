package financials

import (
	"cmp"
	"slices"
	"strings"
)

// index holds the transactions of one request in memory, with the look-ups matching needs (data model §1):
// by key, by PO number (the PO lines) and by SWBS code. The tables are demo-sized, so every request reads
// them whole; PostgreSQL later answers the same look-ups with SQL indexes.
type index struct {
	all   []*Transaction // in id order, then the ones added, in the order they were added
	byKey map[key]*Transaction
	byID  map[string]*Transaction
	pos   map[string][]*Transaction // PO lines by PO number
	swbs  map[string][]*Transaction // every transaction with a SWBS code, by code
}

// newIndex indexes the stored transactions.
func newIndex(stored []Transaction) *index {
	x := &index{
		all: make([]*Transaction, 0, len(stored)), byKey: make(map[key]*Transaction, len(stored)),
		byID: make(map[string]*Transaction, len(stored)), pos: map[string][]*Transaction{}, swbs: map[string][]*Transaction{},
	}
	for i := range stored {
		x.add(&stored[i])
	}
	return x
}

// add indexes the transaction t.
func (x *index) add(t *Transaction) {
	x.all = append(x.all, t)
	x.byKey[t.key()] = t
	x.byID[t.ID] = t
	x.bucket(t)
}

// bucket adds t to the look-ups of its PO number and SWBS code.
func (x *index) bucket(t *Transaction) {
	if t.Type == TypePurchaseOrder {
		x.pos[t.DocumentNumber] = append(x.pos[t.DocumentNumber], t)
	}
	if t.SwbsCode != "" {
		x.swbs[t.SwbsCode] = append(x.swbs[t.SwbsCode], t)
	}
}

// unbucket removes t from the look-ups of its PO number and SWBS code, before its fields change.
func (x *index) unbucket(t *Transaction) {
	isT := func(other *Transaction) bool { return other == t }
	if t.Type == TypePurchaseOrder {
		x.pos[t.DocumentNumber] = slices.DeleteFunc(x.pos[t.DocumentNumber], isT)
	}
	if t.SwbsCode != "" {
		x.swbs[t.SwbsCode] = slices.DeleteFunc(x.swbs[t.SwbsCode], isT)
	}
}

// poLineItems returns the line items that the linked PO lines of the PO number po point to — only the PO
// line line when it is given — each once, in id order (matching rule 3).
func (x *index) poLineItems(po, line string) []string {
	var lineItems []string
	for _, t := range x.pos[po] {
		if t.linked() && (line == "" || t.LineNumber == line) && !slices.Contains(lineItems, t.LineItemID) {
			lineItems = append(lineItems, t.LineItemID)
		}
	}
	slices.SortFunc(lineItems, compareIDs)
	return lineItems
}

// swbsLineItems returns the line items that the linked transactions with the SWBS code, other than self,
// point to, each once, in id order (matching rule 4).
func (x *index) swbsLineItems(code string, self *Transaction) []string {
	var lineItems []string
	for _, t := range x.swbs[code] {
		if t != self && t.linked() && !slices.Contains(lineItems, t.LineItemID) {
			lineItems = append(lineItems, t.LineItemID)
		}
	}
	slices.SortFunc(lineItems, compareIDs)
	return lineItems
}

// compareIDs orders readable IDs as the store does: shorter first, then alphabetically (FT-999 before
// FT-1001).
func compareIDs(x, y string) int {
	return cmp.Or(cmp.Compare(len(x), len(y)), strings.Compare(x, y))
}

// typeRank is the place of a type in the order of application: purchase orders first.
func typeRank(t Type) int {
	return slices.Index(Types, t)
}
