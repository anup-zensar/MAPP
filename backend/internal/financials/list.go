package financials

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// Transactions returns one page of the Financials page (FR-021) for the query q: the transactions a may see
// that match every filter, sorted and paged; the counts of the tabs (every filter but the tab and the link
// filter; Not linked counts the not-linked ones); the totals (every filter but the tab); the record of the
// record filter; the choices of the filters; and where the data comes from.
//
// The year is the linked plan's year; a transaction that is not linked to a live line item counts in its
// posting year (D-256). Without a year, the record filter's plan year is used, otherwise the current year.
func (s *Service) Transactions(ctx context.Context, a actor.Actor, q Query) (TransactionPage, error) {
	var page TransactionPage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		if err := s.access.SeeFinance(ctx, a); err != nil {
			return err
		}
		b, err := s.load(ctx, a.ID)
		if err != nil {
			return err
		}
		sc, err := s.scopeOf(ctx, a)
		if err != nil {
			return err
		}
		var record *tnode
		if q.RecordID != "" {
			gone := fieldProblem("recordId", fmt.Sprintf(msgRecordGone, levelWordOf(q.RecordID)))
			if record = b.tree.byID[q.RecordID]; record == nil {
				return gone
			}
			if err := s.access.SeeFinanceIn(ctx, a, record.PlanID); err != nil {
				return orGone(err, gone)
			}
		}
		if q.PlanID != "" {
			if plan := b.tree.byID[q.PlanID]; plan == nil || plan.Level != LevelPlan {
				return fieldProblem("planId", msgChoosePlan)
			}
			if err := s.access.SeeFinanceIn(ctx, a, q.PlanID); err != nil {
				return orGone(err, fieldProblem("planId", msgChoosePlan))
			}
		}
		year := q.Year
		switch {
		case year != 0:
		case record != nil:
			year = record.PlanYear
		default:
			year = s.today().Year()
		}
		page = b.page(q, year, record, sc, s.today().Year())
		return nil
	})
	if err != nil {
		return TransactionPage{}, err
	}
	return page, nil
}

// placed is a transaction with the live line item it is linked to (nil when none) and its link state.
type placed struct {
	t     *Transaction
	li    *tnode
	state LinkState
}

// page works out the answer of Transactions.
func (b *book) page(q Query, year int, record *tnode, sc scope, currentYear int) TransactionPage {
	search := lower(q.Search)
	var base []placed // every filter but the tab and the link filter
	for _, t := range b.all {
		p := placed{t: t, li: b.lineItemOf(t), state: b.state(t)}
		if !p.inFilters(q, year, record, sc) || (search != "" && !p.holds(search)) {
			continue
		}
		base = append(base, p)
	}
	page := TransactionPage{
		Items: []TransactionRow{}, Page: q.Page, PageSize: q.PageSize, Year: year, Source: b.source(),
		Choices: b.choices(year, sc, currentYear),
	}
	if record != nil {
		ref := record.ref()
		page.RecordFilter = &ref
	}
	var totals Facts
	var shown []placed
	for _, p := range base {
		page.Counts.add(p)
		if !p.linkMatches(q.Link) {
			continue
		}
		totals.add(p.t, b.usd[p.t.ID], b.accrualMonth)
		if q.Type == filterAll || string(p.t.Type) == q.Type {
			shown = append(shown, p)
		}
	}
	page.Totals = Totals{
		Committed: show(b.display, totals.Committed), Invoiced: show(b.display, totals.Invoiced),
		Paid: show(b.display, totals.Paid), Accrued: show(b.display, totals.Accrued),
	}
	b.sort(shown, q.Sort, q.Desc)
	page.Total = len(shown)
	start := (q.Page - 1) * q.PageSize
	for _, p := range shown[min(start, len(shown)):min(start+q.PageSize, len(shown))] {
		page.Items = append(page.Items, b.row(p.t))
	}
	return page
}

// inFilters reports whether the transaction is in a's scope and meets the year, plan and record filters.
// A transaction not linked to a live line item belongs to no plan: it counts in its posting year and only
// for people who may import (D-256, D-301).
func (p placed) inFilters(q Query, year int, record *tnode, sc scope) bool {
	if p.li == nil {
		return sc.unlinked && p.t.PostingDate.Year() == year && q.PlanID == "" && record == nil
	}
	return sc.sees(p.li.PlanID) && p.li.PlanYear == year && (q.PlanID == "" || p.li.PlanID == q.PlanID) &&
		(record == nil || p.li.under(record.ID))
}

// holds reports whether the search text (lower case) is in the transaction's numbers, vendor, SWBS code,
// HX key or description, or in the name or ID of its line item or activity (FR-021).
func (p placed) holds(search string) bool {
	t := p.t
	texts := []string{t.DocumentNumber, t.PONumber, t.VendorCode, t.VendorName, t.SwbsCode, t.TrackingKey, t.Description}
	if p.li != nil {
		texts = append(texts, p.li.Name, p.li.ID)
		if activity := p.li.parent; activity != nil {
			texts = append(texts, activity.Name, activity.ID)
		}
	}
	return slices.ContainsFunc(texts, func(text string) bool { return strings.Contains(lower(text), search) })
}

// linkMatches reports whether the transaction meets the link filter: linked means linked to a line item,
// live or in the recycle bin.
func (p placed) linkMatches(filter string) bool {
	switch filter {
	case filterLinked:
		return p.state != StateNotLinked
	case filterNotLinked:
		return p.state == StateNotLinked
	}
	return true
}

// add counts p in its tabs.
func (c *TypeCounts) add(p placed) {
	c.All++
	switch p.t.Type {
	case TypePurchaseOrder:
		c.PurchaseOrder++
	case TypeInvoice:
		c.Invoice++
	case TypePayment:
		c.Payment++
	case TypeAccrual:
		c.Accrual++
	}
	if p.state == StateNotLinked {
		c.NotLinked++
	}
}

// sort orders the transactions by the column, ascending or descending; ties are newest first (posting
// date, then ID). The amount sorts by the USD value, a currency no longer in Settings lowest.
func (b *book) sort(list []placed, column string, desc bool) {
	slices.SortStableFunc(list, func(x, y placed) int {
		var c int
		switch column {
		case sortDocumentNumber:
			c = cmp.Or(strings.Compare(lower(x.t.DocumentNumber), lower(y.t.DocumentNumber)), compareIDs(x.t.LineNumber, y.t.LineNumber))
		case sortType:
			c = cmp.Compare(typeRank(x.t.Type), typeRank(y.t.Type))
		case sortVendor:
			c = strings.Compare(lower(x.t.VendorName), lower(y.t.VendorName))
		case sortAmount:
			c = compareUSD(b.usd[x.t.ID], b.usd[y.t.ID])
		case sortLinkedTo:
			c = strings.Compare(lower(x.linkedName()), lower(y.linkedName()))
		default: // posting date, ties by ID
			c = cmp.Or(x.t.PostingDate.Compare(y.t.PostingDate), compareIDs(x.t.ID, y.t.ID))
		}
		if desc {
			c = -c
		}
		return cmp.Or(c, newestFirst(x.t, y.t))
	})
}

// compareUSD compares two USD values; nil (currency no longer in Settings) is the lowest.
func compareUSD(x, y *money.Amount) int {
	switch {
	case x == nil && y == nil:
		return 0
	case x == nil:
		return -1
	case y == nil:
		return 1
	}
	return x.Cmp(*y)
}

// linkedName is the name of the live line item the transaction is linked to; "" otherwise.
func (p placed) linkedName() string {
	if p.li == nil {
		return ""
	}
	return p.li.Name
}

// choices returns the values of the filters (FinancialChoices): the years of the plans a may see and, for
// people who may import, the posting years of transactions not linked to a live line item, plus the
// current year, newest first; and the live plans of year a may see, by name.
func (b *book) choices(year int, sc scope, currentYear int) Choices {
	c := Choices{Years: []int{currentYear}, CurrentYear: currentYear, Plans: []PlanChoice{}}
	addYear := func(y int) {
		if !slices.Contains(c.Years, y) {
			c.Years = append(c.Years, y)
		}
	}
	for _, p := range b.tree.plans { // by name
		if !sc.sees(p.ID) {
			continue
		}
		addYear(p.PlanYear)
		if p.PlanYear == year {
			c.Plans = append(c.Plans, PlanChoice{ID: p.ID, Name: p.Name, Year: p.PlanYear})
		}
	}
	if sc.unlinked {
		for _, t := range b.all {
			if b.lineItemOf(t) == nil {
				addYear(t.PostingDate.Year())
			}
		}
	}
	slices.SortFunc(c.Years, func(x, y int) int { return cmp.Compare(y, x) })
	return c
}
