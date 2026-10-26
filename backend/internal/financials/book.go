package financials

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/money"
)

// book is everything one request works out its answers from, read once inside its transaction (plan,
// "Performance Goals"): every transaction with its USD value, the live planning tree with every record's
// figures, the imports and the reader's display currency.
type book struct {
	all          []*Transaction           // every transaction, in id order
	usd          map[string]*money.Amount // transaction id → its USD value; nil when its currency is gone
	accrualMonth string                   // the latest accrual month in the data; "" without accruals
	byLineItem   map[string][]*Transaction
	tree         *tree
	imports      []Import // oldest first
	display      money.Display
}

// load reads the book for the reader personID (an empty personID shows US dollars). It runs inside the
// caller's transaction.
func (s *Service) load(ctx context.Context, personID string) (*book, error) {
	nodes, err := s.planning.Nodes(ctx)
	if err != nil {
		return nil, err
	}
	return s.newBook(ctx, personID, nodes)
}

// loadPlans reads the book with only the plans planIDs: a record's figures never depend on another plan.
func (s *Service) loadPlans(ctx context.Context, personID string, planIDs []string) (*book, error) {
	nodes, err := s.planning.PlanNodes(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	return s.newBook(ctx, personID, nodes)
}

func (s *Service) newBook(ctx context.Context, personID string, nodes []Node) (*book, error) {
	stored, err := s.repo.ListTransactions(ctx)
	if err != nil {
		return nil, err
	}
	rates, err := s.rates.Rates(ctx)
	if err != nil {
		return nil, err
	}
	imports, err := s.repo.ListImports(ctx)
	if err != nil {
		return nil, err
	}
	b := &book{
		all: make([]*Transaction, len(stored)), usd: make(map[string]*money.Amount, len(stored)),
		byLineItem: map[string][]*Transaction{}, imports: imports, display: money.DisplayUSD,
	}
	if personID != "" {
		if b.display, err = s.displays.Display(ctx, personID); err != nil {
			return nil, err
		}
	}
	b.index(stored, rates)
	b.accrualMonth = latestAccrualMonth(b.all)
	facts := make(map[string]Facts, len(b.byLineItem))
	for id, list := range b.byLineItem {
		var f Facts
		for _, t := range list {
			f.add(t, b.usd[t.ID], b.accrualMonth)
		}
		facts[id] = f
	}
	b.tree = buildTree(nodes, facts)
	return b, nil
}

// index keeps every stored transaction with its amount in USD, and the linked ones by line item.
func (b *book) index(stored []Transaction, rates money.Rates) {
	for i := range stored {
		t := &stored[i]
		b.all[i] = t
		b.usd[t.ID] = usdOf(t, rates)
		if t.linked() {
			b.byLineItem[t.LineItemID] = append(b.byLineItem[t.LineItemID], t)
		}
	}
}

// lineItemOf returns the live line item t is linked to, nil when it is not linked or its line item is
// not live (in the recycle bin or purged).
func (b *book) lineItemOf(t *Transaction) *tnode {
	if !t.linked() {
		return nil
	}
	n := b.tree.byID[t.LineItemID]
	if n == nil || n.Level != LevelLineItem {
		return nil
	}
	return n
}

// state returns the link state of t (data model §4.3).
func (b *book) state(t *Transaction) LinkState {
	switch {
	case !t.linked():
		return StateNotLinked
	case b.lineItemOf(t) == nil:
		return StateLineItemDeleted
	}
	return StateLinked
}

// source says where the data comes from (FR-008): the newest import, and the accrual month.
func (b *book) source() SourceView {
	var v SourceView
	if len(b.imports) > 0 {
		newest := b.imports[0]
		for _, i := range b.imports[1:] {
			if i.CreatedAt.After(newest.CreatedAt) || (i.CreatedAt.Equal(newest.CreatedAt) && compareIDs(i.ID, newest.ID) > 0) {
				newest = i
			}
		}
		at, feed := newest.CreatedAt, newest.FeedName
		v.DataAsOf, v.FeedName = &at, &feed
		if newest.FileName != "" {
			name := newest.FileName
			v.FileName = &name
		}
	}
	if b.accrualMonth != "" {
		month := b.accrualMonth
		v.AccrualMonth = &month
	}
	return v
}

// row returns t as lists show it.
func (b *book) row(t *Transaction) TransactionRow {
	r := TransactionRow{
		ID: t.ID, Type: t.Type, Name: t.Name(), DocumentNumber: t.DocumentNumber, LineNumber: t.LineNumber,
		PONumber: t.PONumber, POLineNumber: t.POLineNumber, PostingDate: t.PostingDate,
		Amount: money.Money{Amount: t.Amount, Currency: t.Currency}, Description: t.Description, SwbsCode: t.SwbsCode,
		Link: b.link(t),
	}
	if t.AccrualMonth != "" {
		month := t.AccrualMonth
		r.AccrualMonth = &month
	}
	if usd := b.usd[t.ID]; usd != nil {
		shown := show(b.display, *usd)
		r.AmountDisplay = &shown
	} else {
		r.CurrencyMissing = true
	}
	if t.VendorCode != "" || t.VendorName != "" {
		r.Vendor = &Vendor{Code: t.VendorCode, Name: t.VendorName}
	}
	return r
}

// link returns how t is linked (FinancialLink).
func (b *book) link(t *Transaction) LinkView {
	v := LinkView{State: b.state(t), Path: []RecordRef{}}
	switch v.State {
	case StateNotLinked:
		reason := t.NotLinkedReason
		v.Reason = &reason
		return v
	case StateLineItemDeleted:
		reason := fmt.Sprintf(reasonLineItemInBin, t.LineItemID)
		v.Reason = &reason
	case StateLinked:
		li := b.lineItemOf(t)
		v.Path = append(li.path(), li.ref())
	}
	by, id := t.MatchedBy, t.LineItemID
	v.MatchedBy, v.LineItemID = &by, &id
	if t.ManualLink != nil {
		manualBy, manualAt := t.ManualLink.By, t.ManualLink.At
		v.ManualBy, v.ManualAt = &manualBy, &manualAt
	}
	return v
}

// newestFirst orders transactions by posting date, then id, newest first.
func newestFirst(x, y *Transaction) int {
	return cmp.Or(y.PostingDate.Compare(x.PostingDate), compareIDs(y.ID, x.ID))
}

// below returns the transactions linked to the live line items of n (n itself when it is one).
func (b *book) below(n *tnode) []*Transaction {
	var list []*Transaction
	var visit func(n *tnode)
	visit = func(n *tnode) {
		if n.Level == LevelLineItem {
			list = append(list, b.byLineItem[n.ID]...)
			return
		}
		for _, c := range n.children {
			visit(c)
		}
	}
	visit(n)
	slices.SortFunc(list, newestFirst)
	return list
}
