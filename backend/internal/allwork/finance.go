package allwork

import (
	"slices"
	"time"

	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
)

// Financial figures from GSAP (feature 015, FR-023, FR-024): the Financials column group — committed,
// invoiced, paid, accrued and not yet committed — and the Financials filter. Line items carry their figures
// from their transactions (Node.Finance, filled by the source from feature 015's FactsByLineItem); an
// activity adds up its line items, a container its matching activities, as planned cost does (FR-013).
// The figures are worked out by feature 015 with the same rules as its cards, so the totals agree (SC-002).
// A reader who may not see financial data gets neither the group nor the filter (FR-028, Choices.SeesFinance).

// FinanceFacts are a line item's figures from its GSAP transactions, in USD (015 data model §5).
type FinanceFacts struct {
	Committed, Invoiced, Paid, Accrued money.Amount
	// POCount and InvoiceCount count the purchase-order lines and invoices linked: "Has a PO" holds with
	// one, even when credits bring Committed back to 0.00 (FR-024).
	POCount, InvoiceCount int
}

// plus returns f + g.
func (f FinanceFacts) plus(g FinanceFacts) FinanceFacts {
	return FinanceFacts{
		Committed: f.Committed.Add(g.Committed), Invoiced: f.Invoiced.Add(g.Invoiced), Paid: f.Paid.Add(g.Paid),
		Accrued: f.Accrued.Add(g.Accrued), POCount: f.POCount + g.POCount, InvoiceCount: f.InvoiceCount + g.InvoiceCount,
	}
}

// addFacts returns total + f, where a nil f adds nothing and a nil total is "not shown": the reader may not
// see financial data.
func addFacts(total, f *FinanceFacts) *FinanceFacts {
	if f == nil {
		return total
	}
	sum := *f
	if total != nil {
		sum = total.plus(*f)
	}
	return &sum
}

// GroupFinancials is the column group of the financial figures, after the level groups (FR-023).
const GroupFinancials = "financials"

// The keys of the financial columns.
const (
	columnCommitted       = "committed"
	columnInvoiced        = "invoiced"
	columnPaid            = "paid"
	columnAccrued         = "accrued"
	columnNotYetCommitted = "notYetCommitted"
)

// financeLevels are the levels with financial figures: every level but the unassigned rows (FR-023).
var financeLevels = []Level{LevelPlan, LevelCampaign, LevelInitiative, LevelActivity, LevelLineItem}

// financialColumns are the Financials group, in the order of FR-023. Every one is a roll-up column.
var financialColumns = []Column{
	financial(columnCommitted, "Committed"),
	financial(columnInvoiced, "Invoiced"),
	financial(columnPaid, "Paid (actuals)"),
	financial(columnAccrued, "Accrued"),
	financial(columnNotYetCommitted, "Not yet committed"),
}

// financialKeys are the keys of financialColumns.
var financialKeys = []string{columnCommitted, columnInvoiced, columnPaid, columnAccrued, columnNotYetCommitted}

// financial returns a column of the Financials group.
func financial(key, label string) Column {
	return Column{Key: key, Label: label, Group: GroupFinancials, Kind: KindMoney, Levels: financeLevels, Sortable: true}
}

// forReader returns the catalogue as a reader sees it: without the Financials group when the reader may
// not see financial data (FR-028).
func forReader(columns []Column, seesFinance bool) []Column {
	if seesFinance {
		return columns
	}
	return slices.DeleteFunc(slices.Clone(columns), func(c Column) bool { return c.Group == GroupFinancials })
}

// defaultColumnsIn returns the default columns the catalogue has, in order: a reader who may not see
// financial data gets the defaults without Committed and Invoiced, and nothing left out (FR-019, FR-023).
func defaultColumnsIn(byKey map[string]Column) []string {
	keys := []string{}
	for _, key := range DefaultColumns {
		if _, known := byKey[key]; known {
			keys = append(keys, key)
		}
	}
	return keys
}

// withFinance gives every record of the plans the reader may see financial data in (plans; nil for every
// plan) its financial figures: a line item its own (0.00 without transactions), an activity the sum of its
// line items and whether it is committed above planned cost, a container an empty start for its roll-ups.
// The records of other plans keep none: "—" in the columns, and never a match of the filter (015
// consolidation, D-301).
func (t *tree) withFinance(plans []string) {
	for _, p := range t.plans {
		if plans == nil || slices.Contains(plans, p.n.ID) {
			financeOf(p)
		}
	}
}

// financeOf sets the financial figures of n and everything below it.
func financeOf(n *tnode) {
	switch n.n.Level {
	case LevelLineItem:
		f := FinanceFacts{}
		if n.n.Finance != nil {
			f = *n.n.Finance
		}
		n.finance = &f
		return
	case LevelActivity:
		f := FinanceFacts{}
		for _, li := range n.children {
			financeOf(li)
			f = f.plus(*li.finance)
			if abovePlanned(li.n.PlannedCost, li.finance) {
				n.abovePlanned = true
			}
		}
		n.finance = &f
		if abovePlanned(n.n.PlannedCost, n.finance) {
			n.abovePlanned = true
		}
		return
	}
	n.finance = &FinanceFacts{}
	for _, c := range n.children {
		financeOf(c)
	}
}

// abovePlanned reports whether committed is above the planned cost, exactly; never without a planned cost
// (015 data model §5).
func abovePlanned(planned *money.Amount, f *FinanceFacts) bool {
	return planned != nil && f != nil && f.Committed.Cmp(*planned) > 0
}

// emptyFinance returns zero figures for a record whose reader sees financial data (marker not nil), nil
// otherwise: the start of a container's roll-ups.
func emptyFinance(marker *FinanceFacts) *FinanceFacts {
	if marker == nil {
		return nil
	}
	return &FinanceFacts{}
}

// financeCell returns the cell of a financial column from the figures f (FR-023): "—" when the reader may
// not see financial data; Committed red when above the planned cost; Not yet committed = planned cost −
// committed, "—" without a planned cost and red when negative. The planned cost is that of the records
// whose figures are shown (figures.financePlanned).
func financeCell(key string, f figures, cur Currency) *Cell {
	if f.finance == nil {
		return emptyCell()
	}
	var amount money.Amount
	switch key {
	case columnCommitted:
		cell := moneyCell(cur, &f.finance.Committed)
		if abovePlanned(f.financePlanned, f.finance) {
			cell.Tone = ToneProblem
		}
		return cell
	case columnInvoiced:
		amount = f.finance.Invoiced
	case columnPaid:
		amount = f.finance.Paid
	case columnAccrued:
		amount = f.finance.Accrued
	case columnNotYetCommitted:
		if f.financePlanned == nil {
			return emptyCell()
		}
		amount = f.financePlanned.Sub(f.finance.Committed)
	}
	return moneyCell(cur, &amount)
}

// The values of the Financials filter (FR-024, contract AllWorkFinance).
const (
	FinanceHasPurchaseOrder      = "hasPurchaseOrder"
	FinanceNoPurchaseOrder       = "noPurchaseOrder"
	FinanceInvoiced              = "invoiced"
	FinanceCommittedAbovePlanned = "committedAbovePlanned"
)

// financeValues are the values of the Financials filter, in the order of the filter menu.
var financeValues = []string{FinanceHasPurchaseOrder, FinanceNoPurchaseOrder, FinanceInvoiced, FinanceCommittedAbovePlanned}

// financeWords are the filter's values as its chips read (plan, "Changes to All work").
var financeWords = map[string]string{
	FinanceHasPurchaseOrder: "Has a PO", FinanceNoPurchaseOrder: "No PO yet", FinanceInvoiced: "Invoiced",
	FinanceCommittedAbovePlanned: "Committed above planned cost",
}

// financeFilter is the Financials filter of values (FR-024): tested on activities, values combined with OR.
// An activity without financial figures — the reader may not see them — never matches.
func financeFilter(values []string, t *tree) filter {
	return filter{levels: []Level{LevelActivity}, meets: func(n *Node) bool {
		a := t.byID[n.ID]
		if a == nil || a.finance == nil {
			return false
		}
		return slices.ContainsFunc(values, func(v string) bool {
			switch v {
			case FinanceHasPurchaseOrder:
				return a.finance.POCount > 0
			case FinanceNoPurchaseOrder:
				return a.finance.POCount == 0
			case FinanceInvoiced:
				return !a.finance.Invoiced.IsZero()
			case FinanceCommittedAbovePlanned:
				return a.abovePlanned
			}
			return false
		})
	}}
}

// showsFinance reports whether the view shows a financial column.
func showsFinance(columns []Column) bool {
	return slices.ContainsFunc(columns, func(c Column) bool { return c.Group == GroupFinancials })
}

// financeNote is the export note of the financial figures (FR-023): "GSAP data as of 19 Oct 2026, 02:00
// UTC", or that none was ever loaded.
func financeNote(asOf *time.Time) [2]string {
	if asOf == nil {
		return [2]string{"Financial figures", "No GSAP data loaded yet"}
	}
	at := asOf.UTC()
	return [2]string{"Financial figures", "GSAP data as of " + date.Format(date.Today(at)) + ", " + at.Format("15:04") + " UTC"}
}
