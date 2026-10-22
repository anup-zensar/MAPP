package financials

import (
	"cmp"
	"math/big"
	"slices"
	"strings"

	"mapp/internal/platform/money"
)

// The figures of data model §5 (FR-010 – FR-014): worked out on read, for every request, from the live
// planning tree and every transaction. Nothing here is stored (FR-012).

// Facts are figures in USD from the transactions of one or more line items (data model §5).
type Facts struct {
	Committed, Invoiced, Paid, Accrued money.Amount // USD, summed per transaction after conversion
	Count                              int          // transactions linked
	POCount, InvoiceCount              int          // purchase-order lines and invoices among them (All work's filter)
}

// add adds the transaction t, worth usd in US dollars (nil when its currency is no longer in Settings: it
// is counted but left out of every sum, D-257). Only accruals of the accrual month count as Accrued.
func (f *Facts) add(t *Transaction, usd *money.Amount, accrualMonth string) {
	f.Count++
	switch t.Type {
	case TypePurchaseOrder:
		f.POCount++
	case TypeInvoice:
		f.InvoiceCount++
	}
	if usd == nil {
		return
	}
	switch t.Type {
	case TypePurchaseOrder:
		f.Committed = f.Committed.Add(*usd)
	case TypeInvoice:
		f.Invoiced = f.Invoiced.Add(*usd)
	case TypePayment:
		f.Paid = f.Paid.Add(*usd)
	case TypeAccrual:
		if t.AccrualMonth == accrualMonth {
			f.Accrued = f.Accrued.Add(*usd)
		}
	}
}

// plus returns f + g.
func (f Facts) plus(g Facts) Facts {
	return Facts{
		Committed: f.Committed.Add(g.Committed), Invoiced: f.Invoiced.Add(g.Invoiced), Paid: f.Paid.Add(g.Paid),
		Accrued: f.Accrued.Add(g.Accrued), Count: f.Count + g.Count, POCount: f.POCount + g.POCount,
		InvoiceCount: f.InvoiceCount + g.InvoiceCount,
	}
}

// usdOf converts a transaction's amount to USD with its currency's current rate, half up to the cent
// (FR-010, D-052); nil when the currency is no longer in Settings (D-257). USD needs no rate.
func usdOf(t *Transaction, rates money.Rates) *money.Amount {
	usd, err := rates.ToUSD(money.Money{Amount: t.Amount, Currency: t.Currency})
	if err != nil {
		return nil
	}
	return &usd
}

// latestAccrualMonth is the greatest accrual month among all accruals, whatever their link state; "" when
// there are none (D-244).
func latestAccrualMonth(all []*Transaction) string {
	month := ""
	for _, t := range all {
		if t.Type == TypeAccrual && t.AccrualMonth > month {
			month = t.AccrualMonth
		}
	}
	return month
}

// tnode is one live planning record with its figures in USD.
type tnode struct {
	Node
	parent   *tnode
	children []*tnode // in the plan tree's order

	facts          Facts         // over its live line items (a line item: its own transactions)
	planned        *money.Amount // its planned cost; containers: the sum of their activities' (D-090)
	above          bool          // committed above its planned cost
	lineItemsAbove int           // live line items below committed above their own planned cost
	firstAbove     *tnode        // the first of them by line item ID
}

// tree is the live planning tree with every record's figures.
type tree struct {
	plans []*tnode          // by name, ignoring letter case
	byID  map[string]*tnode // every live record
}

// buildTree builds the tree of nodes and works out every record's figures from the transactions linked
// to each line item.
func buildTree(nodes []Node, byLineItem map[string]Facts) *tree {
	t := &tree{byID: make(map[string]*tnode, len(nodes))}
	all := make([]*tnode, len(nodes))
	for i := range nodes {
		all[i] = &tnode{Node: nodes[i]}
		t.byID[nodes[i].ID] = all[i]
	}
	for _, n := range all {
		if n.Level == LevelPlan {
			t.plans = append(t.plans, n)
			continue
		}
		if parent, found := t.byID[n.ParentID]; found {
			n.parent = parent
			parent.children = append(parent.children, n)
		}
	}
	slices.SortFunc(t.plans, func(x, y *tnode) int {
		return cmp.Or(strings.Compare(lower(x.Name), lower(y.Name)), compareIDs(x.ID, y.ID))
	})
	for _, p := range t.plans {
		t.figure(p, byLineItem)
	}
	return t
}

// figure orders the children of n and works out the figures of n and everything below it.
func (t *tree) figure(n *tnode, byLineItem map[string]Facts) {
	slices.SortStableFunc(n.children, func(x, y *tnode) int { return cmp.Or(cmp.Compare(x.Order, y.Order), compareIDs(x.ID, y.ID)) })
	if n.Level == LevelLineItem {
		n.facts, n.planned = byLineItem[n.ID], n.PlannedCost
		n.above = n.planned != nil && n.facts.Committed.Cmp(*n.planned) > 0
		return
	}
	n.planned = n.PlannedCost // activities as typed; containers start with none
	for _, c := range n.children {
		t.figure(c, byLineItem)
		n.facts = n.facts.plus(c.facts)
		if n.Level != LevelActivity {
			n.planned = addAmount(n.planned, c.planned)
		}
		if c.Level == LevelLineItem {
			if c.above {
				n.lineItemsAbove++
				n.firstAbove = firstByID(n.firstAbove, c)
			}
			continue
		}
		n.lineItemsAbove += c.lineItemsAbove
		n.firstAbove = firstByID(n.firstAbove, c.firstAbove)
	}
	n.above = n.planned != nil && n.facts.Committed.Cmp(*n.planned) > 0
}

// activityAbove reports whether an activity is committed above planned cost (data model §5): itself, or
// one of its line items. This drives All work's filter and Home's count.
func (n *tnode) activityAbove() bool {
	return n.above || n.lineItemsAbove > 0
}

// firstByID returns whichever of a and b has the lower line item ID; nil only when both are.
func firstByID(a, b *tnode) *tnode {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	case compareIDs(b.ID, a.ID) < 0:
		return b
	}
	return a
}

// addAmount returns total + a, where a nil a adds nothing and a nil total is "none yet", as rollup.Sum:
// a sum of only missing planned costs stays missing.
func addAmount(total, a *money.Amount) *money.Amount {
	if a == nil {
		return total
	}
	sum := *a
	if total != nil {
		sum = total.Add(*a)
	}
	return &sum
}

// path returns the ancestors of n, plan first.
func (n *tnode) path() []RecordRef {
	path := []RecordRef{}
	for a := n.parent; a != nil; a = a.parent {
		path = append(path, a.ref())
	}
	slices.Reverse(path)
	return path
}

// ref returns n as a record reference.
func (n *tnode) ref() RecordRef {
	return RecordRef{Type: n.Level, ID: n.ID, Name: n.Name}
}

// under reports whether n is the record id or below it.
func (n *tnode) under(id string) bool {
	for a := n; a != nil; a = a.parent {
		if a.ID == id {
			return true
		}
	}
	return false
}

// ancestor returns the ancestor of n (or n itself) at level, nil when there is none.
func (n *tnode) ancestor(level Level) *tnode {
	for a := n; a != nil; a = a.parent {
		if a.Level == level {
			return a
		}
	}
	return nil
}

// figuresOf returns a record's figures in the display currency d (FR-011, FR-013): each USD figure is
// converted once.
func figuresOf(f Facts, planned *money.Amount, above bool, lineItemsAbove int, firstAbove *tnode, d money.Display) Figures {
	figures := Figures{
		Committed: show(d, f.Committed), Invoiced: show(d, f.Invoiced), Paid: show(d, f.Paid), Accrued: show(d, f.Accrued),
		CommittedAbovePlanned: above, LineItemsAbovePlanned: lineItemsAbove, TransactionCount: f.Count,
	}
	if planned != nil {
		p := show(d, *planned)
		left := show(d, planned.Sub(f.Committed))
		figures.PlannedCost, figures.NotYetCommitted = &p, &left
		figures.CommittedPercent = percentOf(f.Committed, *planned)
	}
	if firstAbove != nil {
		ref := firstAbove.ref()
		figures.FirstLineItemAbovePlanned = &ref
	}
	return figures
}

// figures returns the figures of the record n.
func (n *tnode) figures(d money.Display) Figures {
	lineItemsAbove := n.lineItemsAbove
	if n.Level == LevelLineItem {
		lineItemsAbove = 0
	}
	return figuresOf(n.facts, n.planned, n.above, lineItemsAbove, n.firstAbove, d)
}

// show converts an amount in USD to the display currency d.
func show(d money.Display, usd money.Amount) money.Money {
	return d.Money(money.USD(usd))
}

// percentOf is committed × 100 ÷ planned as a whole per cent, half up (half away from zero for negative
// values); nil when planned is 0.00 (FR-011). 12.5 % gives 13; it may exceed 100.
func percentOf(committed, planned money.Amount) *int {
	if planned.IsZero() {
		return nil
	}
	num := new(big.Int).Mul(big.NewInt(committed.Cents()), big.NewInt(100))
	den := big.NewInt(planned.Cents())
	quotient, remainder := new(big.Int).QuoRem(num, den, new(big.Int))
	twice := new(big.Int).Mul(new(big.Int).Abs(remainder), big.NewInt(2))
	if twice.Cmp(new(big.Int).Abs(den)) >= 0 {
		if num.Sign()*den.Sign() < 0 {
			quotient.Sub(quotient, big.NewInt(1))
		} else {
			quotient.Add(quotient, big.NewInt(1))
		}
	}
	percent := int(quotient.Int64())
	return &percent
}
