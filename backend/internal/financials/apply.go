package financials

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// applier applies one delivery of GSAP data (data model §4.1). Check, Import and the demo load share it,
// so a check says exactly what the import does (SC-003) and the demo data goes through the import's rules.
// It runs inside the caller's Write.
type applier struct {
	repo  Repository
	ids   *ids.Generator
	items LineItemStates // which line items are stored, live or in the recycle bin
	rates money.Rates    // the currencies of Settings
	actor actor.Actor    // who imports
	now   time.Time      // when
	feed  Feed           // where the delivery comes from
}

// applied is the outcome of one delivery: the stored import, every row problem and every accepted row left
// not linked, by row, and the columns the feed did not read.
type applied struct {
	imp       Import
	problems  []RowProblem
	notLinked []NotLinkedRow
	unused    []string
}

// apply reads the delivery d with the applier's feed and applies it (data model §4.1):
//
//  1. Every row is checked: the feed's column rules, then the currency (set up in Settings, active or
//     not) and the key (a key already in an earlier row refuses the later row). Rows with a problem are
//     refused and skipped.
//  2. The accepted rows are applied in the order purchase orders, invoices, payments, accruals, keeping file
//     order within a type, so new transactions get their IDs in that order and later rows find the POs of
//     earlier ones.
//  3. A row with a new key adds a transaction; a known key with different GSAP fields updates it (its link
//     is matched again unless a person made it); otherwise it is unchanged. Imports never delete.
//  4. Every stored not-linked transaction this delivery did not touch is matched again; those that become
//     linked are counted as relinked.
//
// A problem with the whole delivery is returned as the feed's error, before anything is written. apply
// stores the transactions and one import record; it writes no activity-log entry (the caller does).
func (ap applier) apply(ctx context.Context, d Delivery) (applied, error) {
	rows, problems, unused, err := ap.feed.Read(ctx, d)
	if err != nil {
		return applied{}, err
	}
	accepted, problems := ap.check(rows, problems)
	importID, err := ap.ids.Next(ctx, ids.PrefixFinancialImport)
	if err != nil {
		return applied{}, err
	}
	stored, err := ap.repo.ListTransactions(ctx)
	if err != nil {
		return applied{}, err
	}
	x := newIndex(stored)
	ordered := inApplyOrder(accepted)
	run, err := ap.startRun(ctx, x, ordered, importID)
	if err != nil {
		return applied{}, err
	}
	imp := Import{
		ID: importID, Source: ap.feed.Source(), FeedName: ap.feed.Name(), FileName: d.FileName,
		Rows: len(accepted) + refusedRows(problems), Refused: refusedRows(problems), Meta: record.NewMeta(ap.actor, ap.now),
	}
	if imp.Source == SourceDemo {
		imp.FileName = ""
	}
	var notLinked []NotLinkedRow
	touched := map[*Transaction]bool{}
	for _, in := range ordered {
		t, outcome, err := ap.applyRow(ctx, run, in)
		if err != nil {
			return applied{}, err
		}
		switch outcome {
		case rowNew:
			imp.New++
		case rowUpdated:
			imp.Updated++
		default:
			imp.Unchanged++
		}
		touched[t] = true
		if !t.linked() {
			imp.NotLinked++
			notLinked = append(notLinked, NotLinkedRow{Row: in.Row, Name: t.Name(), Reason: t.NotLinkedReason})
		}
	}
	if err := ap.repo.AddTransactions(ctx, run.added); err != nil {
		return applied{}, err
	}
	for _, t := range inRetryOrder(x.all) {
		if touched[t] || t.MatchedBy != MatchedByNone || !relink(t, x, ap.items) {
			continue
		}
		if t.linked() {
			imp.Relinked++
		}
		t.Touch(ap.actor, ap.now)
		if err := ap.repo.PutTransaction(ctx, *t); err != nil {
			return applied{}, err
		}
	}
	if err := ap.repo.PutImport(ctx, imp); err != nil {
		return applied{}, err
	}
	slices.SortStableFunc(notLinked, func(a, b NotLinkedRow) int { return cmp.Compare(a.Row, b.Row) })
	return applied{imp: imp, problems: problems, notLinked: orEmpty(notLinked), unused: orEmpty(unused)}, nil
}

// The outcomes of an accepted row (FR-006).
const (
	rowNew = iota
	rowUpdated
	rowUnchanged
)

// rowRun is what applying the rows of one delivery shares: the index, the import, the IDs taken for the new
// transactions in apply order, and the new transactions, stored together after the last row.
type rowRun struct {
	x        *index
	importID string
	newIDs   []string
	added    []Transaction
}

// startRun takes the IDs of the new transactions among rows at once, in apply order.
func (ap applier) startRun(ctx context.Context, x *index, rows []Incoming, importID string) (*rowRun, error) {
	newIDs, err := ap.ids.NextIDs(ctx, ids.PrefixFinancialTransaction, newRows(x, rows))
	if err != nil {
		return nil, err
	}
	return &rowRun{x: x, importID: importID, newIDs: newIDs}, nil
}

// newRows counts the rows whose key no stored transaction has: the delivery's keys are unique (check).
func newRows(x *index, rows []Incoming) int {
	n := 0
	for _, in := range rows {
		if _, known := x.byKey[in.key()]; !known {
			n++
		}
	}
	return n
}

// applyRow applies one accepted row and returns its transaction and outcome.
func (ap applier) applyRow(ctx context.Context, run *rowRun, in Incoming) (*Transaction, int, error) {
	x, importID := run.x, run.importID
	t, known := x.byKey[in.key()]
	switch {
	case !known:
		t = &Transaction{ID: run.newIDs[len(run.added)], Fields: in.Fields, ImportID: importID, Meta: record.NewMeta(ap.actor, ap.now)}
		relink(t, x, ap.items)
		x.add(t)
		run.added = append(run.added, *t)
		return t, rowNew, nil
	case t.Fields != in.Fields:
		x.unbucket(t)
		t.Fields, t.ImportID = in.Fields, importID
		x.bucket(t)
		relink(t, x, ap.items) // keeps a manual link
		t.Touch(ap.actor, ap.now)
		return t, rowUpdated, ap.repo.PutTransaction(ctx, *t)
	}
	// An unchanged row keeps its GSAP fields; when it is still not linked, it is matched again like every
	// not-linked transaction, so importing the same file twice changes nothing the second time.
	if t.MatchedBy == MatchedByNone && relink(t, x, ap.items) {
		t.Touch(ap.actor, ap.now)
		if err := ap.repo.PutTransaction(ctx, *t); err != nil {
			return nil, 0, err
		}
	}
	return t, rowUnchanged, nil
}

// check adds the problems a feed cannot know to its problems (data model §3): a currency that is not set
// up in Settings, and a key already used by an earlier row of the delivery. It returns the rows without a
// problem, in file order, and every problem ordered by row.
func (ap applier) check(rows []Incoming, problems []RowProblem) ([]Incoming, []RowProblem) {
	problems = slices.Clone(problems)
	firstRow := map[key]int{}
	accepted := make([]Incoming, 0, len(rows))
	for _, in := range rows {
		refused := false
		if _, setUp := ap.rates[in.Currency]; !setUp && in.Currency != money.BaseCurrency {
			problems = append(problems, RowProblem{Row: in.Row, Column: colCurrency, Message: fmt.Sprintf(msgCurrency, in.Currency)})
			refused = true
		}
		if first, seen := firstRow[in.key()]; seen {
			problems = append(problems, RowProblem{
				Row: in.Row, Column: colDocumentNumber, Message: fmt.Sprintf(msgRepeatedKey, in.Row, in.Name(), first),
			})
			refused = true
		} else {
			firstRow[in.key()] = in.Row
		}
		if !refused {
			accepted = append(accepted, in)
		}
	}
	slices.SortStableFunc(problems, func(a, b RowProblem) int { return cmp.Compare(a.Row, b.Row) })
	return accepted, orEmpty(problems)
}

// refusedRows counts the rows with at least one problem.
func refusedRows(problems []RowProblem) int {
	rows := map[int]bool{}
	for _, p := range problems {
		rows[p.Row] = true
	}
	return len(rows)
}

// inApplyOrder returns the rows in the order of application: by type, purchase orders first, keeping file
// order within a type.
func inApplyOrder(rows []Incoming) []Incoming {
	ordered := slices.Clone(rows)
	slices.SortStableFunc(ordered, func(a, b Incoming) int { return cmp.Compare(typeRank(a.Type), typeRank(b.Type)) })
	return ordered
}

// inRetryOrder returns the transactions in the order of the retry: by type, then by id.
func inRetryOrder(all []*Transaction) []*Transaction {
	ordered := slices.Clone(all)
	slices.SortStableFunc(ordered, func(a, b *Transaction) int {
		return cmp.Or(cmp.Compare(typeRank(a.Type), typeRank(b.Type)), compareIDs(a.ID, b.ID))
	})
	return ordered
}

// orEmpty returns list, or an empty list instead of nil, so the JSON has [] and never null.
func orEmpty[T any](list []T) []T {
	if list == nil {
		return []T{}
	}
	return list
}
