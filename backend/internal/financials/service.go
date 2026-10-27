package financials

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
)

// maxListed is the most problems and not-linked rows a check or an import lists; the totals say how many
// there are.
const maxListed = 200

// latestCount is how many latest transactions a Financials card shows (FR-015).
const latestCount = 5

// Deps are the ports of the service (ports.go), which internal/app fills.
type Deps struct {
	Planning Planning
	Rates    Rates
	Displays Displays
	Access   Access
}

// Service holds the rules of feature 015. Every method takes the actor; the reads run in one read
// transaction, imports, links and unlinks in one Write with their activity-log entry.
type Service struct {
	tx       store.Transactor
	repo     Store
	ids      *ids.Generator
	history  *history.Service
	planning Planning
	rates    Rates
	displays Displays
	access   Access
	feed     Feed // the feed of uploaded files: the simulated GSAP extract
	now      clock.Clock
}

// NewService returns the financials service. It keeps transactions and imports in repo, gives out readable
// IDs, logs, works in transactions and reads the time with tools, and reads the other features through
// deps. Uploaded files are read by the simulated GSAP extract (CSVFeed).
func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	return &Service{
		tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, planning: deps.Planning, rates: deps.Rates,
		displays: deps.Displays, access: deps.Access, feed: CSVFeed{}, now: tools.Now,
	}
}

// UseFeed replaces the feed that reads uploaded files — a real GSAP feed later (FR-003, SC-006). Nothing
// else changes: the checks, matching, figures, handler and contract stay the same.
func (s *Service) UseFeed(f Feed) {
	s.feed = f
}

// today is the calendar date of the clock: "the current year" is its year (D-091).
func (s *Service) today() date.Date {
	return date.Today(s.now())
}

// Record returns everything a Financials card shows for the plan, campaign, initiative, activity or line
// item recordID (FR-011 – FR-019): its figures, a row per live child, its 5 latest transactions, the SWBS
// codes of a line item's transactions and where the data comes from. A record that is not live, or is
// outside a's access, is not_found (FR-028).
func (s *Service) Record(ctx context.Context, a actor.Actor, recordID string) (RecordView, error) {
	var view RecordView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		gone := errs.NotFound(fmt.Sprintf(msgRecordGone, levelWordOf(recordID)))
		planID, live, err := s.planning.PlanOf(ctx, recordID)
		if err != nil {
			return err
		}
		if !live {
			return gone
		}
		b, err := s.loadPlans(ctx, a.ID, []string{planID})
		if err != nil {
			return err
		}
		n := b.tree.byID[recordID]
		if n == nil {
			return gone
		}
		if err := s.access.SeeFinanceIn(ctx, a, n.PlanID); err != nil {
			return orGone(err, gone)
		}
		view = b.recordView(n)
		return nil
	})
	if err != nil {
		return RecordView{}, err
	}
	return view, nil
}

// recordView returns the card of n.
func (b *book) recordView(n *tnode) RecordView {
	v := RecordView{
		Record: n.ref(), PlanID: n.PlanID, Path: n.path(), Figures: n.figures(b.display), Breakdown: []BreakdownRow{},
		Latest: []TransactionRow{}, SwbsCodes: []string{}, Source: b.source(),
	}
	if level, found := childLevel[n.Level]; found {
		v.BreakdownLevel = &level
		for _, c := range n.children {
			v.Breakdown = append(v.Breakdown, BreakdownRow{Record: c.ref(), Figures: c.figures(b.display)})
		}
	}
	latest := b.below(n)
	for _, t := range latest[:min(latestCount, len(latest))] {
		v.Latest = append(v.Latest, b.row(t))
	}
	if n.Level == LevelLineItem {
		for _, t := range b.byLineItem[n.ID] {
			if t.SwbsCode != "" && !slices.Contains(v.SwbsCodes, t.SwbsCode) {
				v.SwbsCodes = append(v.SwbsCodes, t.SwbsCode)
			}
		}
		slices.Sort(v.SwbsCodes)
	}
	return v
}

// levelWordOf names the level of a readable ID mid-sentence: "activity" for A-1002.
func levelWordOf(recordID string) string {
	prefix, _, _ := strings.Cut(recordID, "-")
	for level, p := range map[Level]string{
		LevelPlan: ids.PrefixPlan, LevelCampaign: ids.PrefixCampaign, LevelInitiative: ids.PrefixInitiative,
		LevelActivity: ids.PrefixActivity, LevelLineItem: ids.PrefixLineItem,
	} {
		if p == prefix {
			return levelWords[level]
		}
	}
	return "record"
}

// scope is what a person may see of the transactions (FR-028, 014 FR-007).
type scope struct {
	plans    []string // nil: every plan
	unlinked bool     // not-linked transactions: only for people who may import (D-301)
}

// scopeOf reads a's scope.
func (s *Service) scopeOf(ctx context.Context, a actor.Actor) (scope, error) {
	plans, err := s.access.VisiblePlanIDs(ctx, a)
	if err != nil {
		return scope{}, err
	}
	return scope{plans: plans, unlinked: s.access.Import(ctx, a) == nil}, nil
}

// sees reports whether the plan planID is in the scope.
func (sc scope) sees(planID string) bool {
	return sc.plans == nil || slices.Contains(sc.plans, planID)
}

// Transaction returns one transaction with every field of the details panel (FR-022). A linked
// transaction needs access to its plan's financial data; a not-linked one needs the right to import
// (D-301): otherwise it is forbidden, or not_found when its plan is outside a's access.
func (s *Service) Transaction(ctx context.Context, a actor.Actor, id string) (TransactionView, error) {
	var view TransactionView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		if err := s.access.SeeFinance(ctx, a); err != nil {
			return err
		}
		b, err := s.load(ctx, a.ID)
		if err != nil {
			return err
		}
		t := b.find(id)
		if t == nil {
			return errs.NotFound(msgTransactionGone)
		}
		if err := s.mayOpen(ctx, a, b, t); err != nil {
			return err
		}
		view = s.transactionView(ctx, a, b, t)
		return nil
	})
	if err != nil {
		return TransactionView{}, err
	}
	return view, nil
}

// mayOpen checks that a may open t (D-301).
func (s *Service) mayOpen(ctx context.Context, a actor.Actor, b *book, t *Transaction) error {
	var err error
	if li := b.lineItemOf(t); li != nil {
		err = s.access.SeeFinanceIn(ctx, a, li.PlanID)
	} else {
		err = s.access.Import(ctx, a)
	}
	switch {
	case err == nil:
		return nil
	case isHidden(err):
		return errs.NotFound(msgTransactionGone)
	default:
		return errs.Forbidden(fmt.Sprintf(msgNoAccessRecord, transactionNoun))
	}
}

// isHidden reports whether an access check answered not_found: the record is outside the person's access,
// which must look exactly like a missing record (constitution VII).
func isHidden(err error) bool {
	var apiErr *errs.Error
	return errors.As(err, &apiErr) && apiErr.Code == errs.CodeNotFound
}

// orGone answers gone, the answer for a missing record, when an access check found the record outside the
// person's access, and err otherwise.
func orGone(err, gone error) error {
	if isHidden(err) {
		return gone
	}
	return err
}

// find returns the transaction id, nil when there is none.
func (b *book) find(id string) *Transaction {
	i := slices.IndexFunc(b.all, func(t *Transaction) bool { return t.ID == id })
	if i < 0 {
		return nil
	}
	return b.all[i]
}

// transactionView returns t with every field of the details panel, and what a may do with it.
func (s *Service) transactionView(ctx context.Context, a actor.Actor, b *book, t *Transaction) TransactionView {
	r := b.row(t)
	v := TransactionView{
		ID: r.ID, Type: r.Type, Name: r.Name, DocumentNumber: r.DocumentNumber, LineNumber: r.LineNumber,
		PONumber: r.PONumber, POLineNumber: r.POLineNumber, PostingDate: r.PostingDate, AccrualMonth: r.AccrualMonth,
		Amount: r.Amount, AmountDisplay: r.AmountDisplay, CurrencyMissing: r.CurrencyMissing, Vendor: r.Vendor,
		Description: r.Description, SwbsCode: r.SwbsCode, TrackingKey: t.TrackingKey, SentLineItemID: t.SentLineItemID,
		Link: r.Link, Version: t.Version, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, UpdatedBy: t.UpdatedBy,
	}
	if usd := b.usd[t.ID]; usd != nil {
		amount := money.USD(*usd)
		v.AmountUSD = &amount
	}
	for _, i := range b.imports {
		if i.ID == t.ImportID {
			v.Import = i.ref()
		}
	}
	planID := ""
	if li := b.lineItemOf(t); li != nil {
		planID = li.PlanID
	}
	v.CanLink = !t.linked() && s.access.Link(ctx, a, "") == nil
	v.CanUnlink = t.MatchedBy == MatchedByManual && s.access.Link(ctx, a, planID) == nil
	return v
}

// Overview returns Home's Financials card (FR-025): the figures of the current year's live plans in the
// Home scope (My teams: the plans with one of a's teams; Everything: every plan) that a may see, the number
// of their activities committed above planned cost and the number of not-linked transactions (for people
// who may import). The year is the API clock's (D-091).
func (s *Service) Overview(ctx context.Context, a actor.Actor, scopeText string) (Overview, error) {
	homeScope, parseErr := ParseScope(scopeText)
	if parseErr != nil {
		return Overview{}, parseErr
	}
	var overview Overview
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		if err := s.access.SeeFinance(ctx, a); err != nil {
			return err
		}
		sc, err := s.scopeOf(ctx, a)
		if err != nil {
			return err
		}
		planIDs, year, err := s.overviewPlans(ctx, a, homeScope, sc)
		if err != nil {
			return err
		}
		b, err := s.loadPlans(ctx, a.ID, planIDs)
		if err != nil {
			return err
		}
		overview = b.overview(b.tree.plans, sc)
		overview.Year, overview.Scope = year, homeScope
		return nil
	})
	if err != nil {
		return Overview{}, err
	}
	return overview, nil
}

// overviewPlans returns the plans of the current year the overview adds up, those sc sees, with My teams
// a's teams' only, and the year.
func (s *Service) overviewPlans(ctx context.Context, a actor.Actor, homeScope Scope, sc scope) ([]string, int, error) {
	year := s.today().Year()
	var mine []string
	if homeScope == ScopeMyTeams {
		var err error
		if mine, err = s.planning.MyTeamPlanIDs(ctx, a.ID); err != nil {
			return nil, year, err
		}
	}
	ofYear, err := s.planning.PlanIDsOfYear(ctx, year)
	if err != nil {
		return nil, year, err
	}
	planIDs := []string{}
	for _, id := range ofYear {
		if sc.sees(id) && (homeScope == ScopeEverything || slices.Contains(mine, id)) {
			planIDs = append(planIDs, id)
		}
	}
	return planIDs, year, nil
}

// overview adds up the plans as one record.
func (b *book) overview(plans []*tnode, sc scope) Overview {
	var facts Facts
	var planned *money.Amount
	var first *tnode
	lineItemsAbove, activitiesAbove := 0, 0
	for _, p := range plans {
		facts = facts.plus(p.facts)
		planned = addAmount(planned, p.planned)
		lineItemsAbove += p.lineItemsAbove
		first = firstByID(first, p.firstAbove)
		activitiesAbove += countActivitiesAbove(p)
	}
	above := planned != nil && facts.Committed.Cmp(*planned) > 0
	o := Overview{
		Figures:                figuresOf(facts, planned, above, lineItemsAbove, first, b.display),
		ActivitiesAbovePlanned: activitiesAbove, Source: b.source(),
	}
	if sc.unlinked {
		for _, t := range b.all {
			if !t.linked() {
				o.NotLinked++
			}
		}
	}
	return o
}

// countActivitiesAbove counts the activities below n committed above planned cost (data model §5).
func countActivitiesAbove(n *tnode) int {
	if n.Level == LevelActivity {
		if n.activityAbove() {
			return 1
		}
		return 0
	}
	count := 0
	for _, c := range n.children {
		count += countActivitiesAbove(c)
	}
	return count
}

// Imports returns every import, newest first (FR-008).
func (s *Service) Imports(ctx context.Context, a actor.Actor) (ImportPage, error) {
	page := ImportPage{Items: []ImportView{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		if err := s.access.SeeFinance(ctx, a); err != nil {
			return err
		}
		imports, err := s.repo.ListImports(ctx)
		if err != nil {
			return err
		}
		for _, i := range slices.Backward(imports) {
			page.Items = append(page.Items, i.view())
		}
		return nil
	})
	if err != nil {
		return ImportPage{}, err
	}
	page.Total = len(page.Items)
	return page, nil
}

// errDiscard ends the Write of a check, so that nothing it did is saved (D-177).
var errDiscard = errors.New("financials: check only, discard the changes")

// Check says what importing the file would do, without saving anything (FR-004): it runs the import inside
// a Write and throws it away, so the check and the import cannot disagree (SC-003). Only people who may
// import may check (FR-028).
func (s *Service) Check(ctx context.Context, a actor.Actor, req ImportRequest) (ImportCheck, error) {
	var check ImportCheck
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		result, err := s.apply(ctx, a, req)
		if err != nil {
			return err
		}
		problems, notLinked := capped(result.problems), capped(result.notLinked)
		check = ImportCheck{
			FileName: result.imp.FileName, Counts: result.imp.counts(), Problems: importProblems(problems),
			ProblemsTotal: len(result.problems), NotLinked: notLinked, NotLinkedTotal: len(result.notLinked),
			UnusedColumns: result.unused,
		}
		return errDiscard
	})
	if !errors.Is(err, errDiscard) {
		return ImportCheck{}, err
	}
	return check, nil
}

// Import imports the file (FR-004 – FR-007): its valid rows are applied, refused rows skipped and listed,
// every not-linked transaction matched again; one import record and one activity-log entry are written
// with the counts (FR-027). A problem with the whole file saves nothing.
func (s *Service) Import(ctx context.Context, a actor.Actor, req ImportRequest) (ImportResult, error) {
	var result ImportResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		done, err := s.apply(ctx, a, req)
		if err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: importKind, RecordID: done.imp.ID, RecordName: done.imp.FileName, Action: history.ActionCreated,
			Verb: importedVerb, Details: countsText(done.imp.counts()),
		})
		if err != nil {
			return err
		}
		result = ImportResult{
			Import: done.imp.view(), Problems: importProblems(capped(done.problems)), ProblemsTotal: len(done.problems),
			NotLinked: capped(done.notLinked), NotLinkedTotal: len(done.notLinked),
		}
		return nil
	})
	if err != nil {
		return ImportResult{}, err
	}
	return result, nil
}

// apply checks a's right to import, decodes the file and applies it with the file feed.
func (s *Service) apply(ctx context.Context, a actor.Actor, req ImportRequest) (applied, error) {
	if err := s.access.Import(ctx, a); err != nil {
		return applied{}, err
	}
	content, err := base64.StdEncoding.DecodeString(req.Content)
	if err != nil {
		return applied{}, contentProblem(msgFileUnreadable)
	}
	ap, err := s.applier(ctx, a, s.feed)
	if err != nil {
		return applied{}, err
	}
	return ap.apply(ctx, Delivery{FileName: strings.TrimSpace(req.FileName), Content: content})
}

// applier returns the applier of a delivery by a with feed.
func (s *Service) applier(ctx context.Context, a actor.Actor, feed Feed) (applier, error) {
	nodes, err := s.planning.Nodes(ctx)
	if err != nil {
		return applier{}, err
	}
	inBin, err := s.planning.LineItemsInBin(ctx)
	if err != nil {
		return applier{}, err
	}
	items := storedSet{}
	for _, n := range nodes {
		if n.Level == LevelLineItem {
			items[n.ID] = true
		}
	}
	for _, id := range inBin {
		items[id] = true
	}
	rates, err := s.rates.Rates(ctx)
	if err != nil {
		return applier{}, err
	}
	return applier{repo: s.repo, ids: s.ids, items: items, rates: rates, actor: a, now: s.now(), feed: feed}, nil
}

// DemoLoad is what LoadDemo matches and dates the demo data with: the line items the seed stored, the
// rates of its currencies and the seed time.
type DemoLoad struct {
	Items LineItemStates
	Rates money.Rates
	At    time.Time
}

// LoadDemo loads the demo data of FR-031 inside the caller's Write: the demo feed's 11 transactions
// through the same apply code as an import, as MAPP (actor.System) at the time demo.At, and one import
// record "Demo data (simulated GSAP)". It writes no activity-log entry (D-188). The seed calls it.
func LoadDemo(ctx context.Context, repo Store, generator *ids.Generator, demo DemoLoad) (Import, error) {
	ap := applier{
		repo: repo, ids: generator, items: demo.Items, rates: demo.Rates, actor: actor.System, now: demo.At,
		feed: DemoFeed{},
	}
	done, err := ap.apply(ctx, Delivery{})
	if err != nil {
		return Import{}, err
	}
	return done.imp, nil
}

// capped returns at most the first maxListed of list.
func capped[T any](list []T) []T {
	return list[:min(maxListed, len(list))]
}

// importProblems returns row problems as the API answers them.
func importProblems(problems []RowProblem) []ImportProblem {
	list := make([]ImportProblem, 0, len(problems))
	for _, p := range problems {
		list = append(list, ImportProblem{Row: p.Row, Column: p.Column, Message: p.Message})
	}
	return list
}

// Link links the not-linked transaction id to the line item req.LineItemID by hand (FR-026, D-253) and
// logs it (FR-027). It refuses: a missing version or a line item ID of the wrong form (validation); a
// transaction that does not exist (not_found); a version that is out of date (version_conflict); a
// transaction already linked, by the rules — corrected in GSAP — or by a person (not_allowed); a line item
// that does not exist or is in the recycle bin (validation on lineItemId); a person who may not link in the
// line item's plan (forbidden).
func (s *Service) Link(ctx context.Context, a actor.Actor, id string, req LinkRequest) (TransactionView, error) {
	var problems errs.Problems
	if req.Version == nil {
		problems.Add("version", msgEnterVersion)
	}
	if !lineItemIDForm.MatchString(req.LineItemID) {
		problems.Add("lineItemId", msgLineItemIDForm)
	}
	if err := problems.Err(); err != nil {
		return TransactionView{}, err
	}
	var view TransactionView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if err := s.access.Link(ctx, a, ""); err != nil {
			return err
		}
		b, err := s.load(ctx, a.ID)
		if err != nil {
			return err
		}
		t := b.find(id)
		if t == nil {
			return errs.NotFound(msgTransactionGone)
		}
		if err := t.CheckVersion(*req.Version, transactionNoun); err != nil {
			return err
		}
		switch {
		case t.MatchedBy == MatchedByManual:
			return errs.NotAllowed(fmt.Sprintf(msgLinkedByPerson, t.ManualLink.By.Name, dayText(t.ManualLink.At)))
		case t.linked():
			return errs.NotAllowed(fmt.Sprintf(msgLinkedByGSAP, matchedWords(*t)))
		}
		target := b.tree.byID[req.LineItemID]
		if target == nil || target.Level != LevelLineItem {
			return s.missingLineItem(ctx, req.LineItemID)
		}
		if err := s.access.Link(ctx, a, target.PlanID); err != nil {
			return err
		}
		t.LineItemID, t.MatchedBy, t.NotLinkedReason = target.ID, MatchedByManual, ""
		t.ManualLink = &ManualLink{By: a, At: s.now()}
		if err := s.save(ctx, a, t, linkedVerb, "", target.ID); err != nil {
			return err
		}
		view = s.transactionView(ctx, a, b, t)
		return nil
	})
	if err != nil {
		return TransactionView{}, err
	}
	return view, nil
}

// missingLineItem is the refusal of a line item ID that is not live: in the recycle bin, or unknown.
func (s *Service) missingLineItem(ctx context.Context, id string) error {
	inBin, err := s.planning.LineItemsInBin(ctx)
	if err != nil {
		return err
	}
	if slices.Contains(inBin, id) {
		return fieldProblem("lineItemId", fmt.Sprintf(msgLineItemInBin, id))
	}
	return fieldProblem("lineItemId", fmt.Sprintf(msgNoLineItem, id))
}

// Unlink undoes a manual link (FR-026): the transaction is matched again by the rules, and usually is Not
// linked again with its import reason. Only links made by a person can be undone (not_allowed otherwise).
func (s *Service) Unlink(ctx context.Context, a actor.Actor, id string, req UnlinkRequest) (TransactionView, error) {
	if req.Version == nil {
		return TransactionView{}, fieldProblem("version", msgEnterVersion)
	}
	var view TransactionView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if err := s.access.Link(ctx, a, ""); err != nil {
			return err
		}
		b, err := s.load(ctx, a.ID)
		if err != nil {
			return err
		}
		t := b.find(id)
		if t == nil {
			return errs.NotFound(msgTransactionGone)
		}
		if err := t.CheckVersion(*req.Version, transactionNoun); err != nil {
			return err
		}
		if t.MatchedBy != MatchedByManual {
			return errs.NotAllowed(msgOnlyManualUnlink)
		}
		planID := ""
		if li := b.lineItemOf(t); li != nil {
			planID = li.PlanID
		}
		if err := s.access.Link(ctx, a, planID); err != nil {
			return err
		}
		before := t.LineItemID
		t.LineItemID, t.MatchedBy, t.ManualLink = "", MatchedByNone, nil
		if err := s.rematch(ctx, b, t); err != nil {
			return err
		}
		if err := s.save(ctx, a, t, unlinkedVerb, before, t.LineItemID); err != nil {
			return err
		}
		view = s.transactionView(ctx, a, b, t)
		return nil
	})
	if err != nil {
		return TransactionView{}, err
	}
	return view, nil
}

// rematch matches t again by the rules, against every other stored transaction.
func (s *Service) rematch(ctx context.Context, b *book, t *Transaction) error {
	inBin, err := s.planning.LineItemsInBin(ctx)
	if err != nil {
		return err
	}
	items := storedSet{}
	for id, n := range b.tree.byID {
		if n.Level == LevelLineItem {
			items[id] = true
		}
	}
	for _, id := range inBin {
		items[id] = true
	}
	stored := make([]Transaction, 0, len(b.all))
	for _, other := range b.all {
		if other.ID != t.ID {
			stored = append(stored, *other)
		}
	}
	x := newIndex(stored)
	l := match(t, x, items)
	t.LineItemID, t.MatchedBy, t.NotLinkedReason = l.lineItemID, l.by, l.reason
	return nil
}

// save stores t after a link change by a and writes its one activity-log entry: the line item before and
// after ("" for Not linked).
func (s *Service) save(ctx context.Context, a actor.Actor, t *Transaction, verb, before, after string) error {
	t.Touch(a, s.now())
	if err := s.repo.PutTransaction(ctx, *t); err != nil {
		return err
	}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: transactionKind, RecordID: t.ID, RecordName: t.Name(), Action: history.ActionChanged, Verb: verb,
		Changes: []history.Change{{Field: "lineItemId", Old: idValue(before), New: idValue(after)}},
	})
	return err
}

// idValue is a line item id as a stored JSON value: null for none.
func idValue(id string) json.RawMessage {
	if id == "" {
		return json.RawMessage("null")
	}
	encoded, _ := json.Marshal(id) // a string always encodes
	return encoded
}

// dayText writes a moment's UTC date as people read it: "20 Oct 2026".
func dayText(t time.Time) string {
	return t.UTC().Format("2 Jan 2006")
}

// FactsByLineItem returns the figures in USD of every line item with linked transactions, by line item
// id, for All work (010 FR-023): the same rules as the cards — current rates, the latest accrual month,
// currencies no longer in Settings left out (SC-002). It joins the caller's transaction and checks no
// access: the caller does.
func (s *Service) FactsByLineItem(ctx context.Context) (map[string]Facts, error) {
	var facts map[string]Facts
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.repo.ListTransactions(ctx)
		if err != nil {
			return err
		}
		rates, err := s.rates.Rates(ctx)
		if err != nil {
			return err
		}
		all := make([]*Transaction, len(stored))
		for i := range stored {
			all[i] = &stored[i]
		}
		month := latestAccrualMonth(all)
		facts = map[string]Facts{}
		for _, t := range all {
			if t.linked() {
				f := facts[t.LineItemID]
				f.add(t, usdOf(t, rates), month)
				facts[t.LineItemID] = f
			}
		}
		return nil
	})
	return facts, err
}

// DataAsOf returns when the newest GSAP data was loaded (FR-008), nil when none ever was. It joins the
// caller's transaction.
func (s *Service) DataAsOf(ctx context.Context) (*time.Time, error) {
	var asOf *time.Time
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		imports, err := s.repo.ListImports(ctx)
		if err != nil {
			return err
		}
		asOf = (&book{imports: imports}).source().DataAsOf
		return nil
	})
	return asOf, err
}
