package lineitems

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
	"northstar/internal/settings"
)

// Messages of the plan's "User-visible messages" and "Validation messages" tables.
const (
	activityGoneMessage = "This activity was deleted — restore it from the recycle bin to keep working on its line items."
	lineItemGoneMessage = "This line item no longer exists — someone may have deleted it."
	restartMessage      = "This activity will need approval again. Confirm the change to send it back for approval."
	reloadMessage       = "Reload the page and try again."
)

// The restart reasons sent to feature 006 (plan, "Restart reasons").
const (
	addedReason   = "Line item '%s' (%s) added: %s %s"
	deletedReason = "Line item '%s' (%s) deleted: %s %s"
	changedReason = "Planned cost of line item '%s' (%s): %s %s → %s %s"
)

// Service holds the rules of line items (feature 007).
type Service struct {
	tx      store.Transactor
	repo    Repository
	ids     *ids.Generator
	history *history.Service
	deps    Deps
	now     clock.Clock
}

// NewService returns the line-items service. It keeps line items in repo, gives them readable IDs with
// generator, logs their changes and deletes them through hist, reads the other features through deps, works
// in transactions of tx and reads the time from now.
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator, hist *history.Service, deps Deps, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, ids: generator, history: hist, deps: deps, now: now}
}

// Section returns the Line items section of the activity activityID (FR-001 – FR-004, FR-023): its line
// items in creation order, each with its USD amount at the currency's current rate, its spread, what is
// missing and its SWBS request status; the unassigned planned cost; and the spread grid. Nothing in USD is
// stored, so a rate change in Settings shows at once and changes no status (FR-013, D-141). The amounts the
// section shows in USD are answered in a's display currency (FR-005, D-213); every signed-in person may see
// it (D-004).
func (s *Service) Section(ctx context.Context, a actor.Actor, activityID string) (Section, error) {
	var section Section
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.facts(ctx, activityID)
		if err != nil {
			return err
		}
		items, err := s.repo.ListByActivity(ctx, activityID)
		if err != nil {
			return err
		}
		views, err := s.views(ctx, f, items)
		if err != nil {
			return err
		}
		d, err := s.displayOf(ctx, a)
		if err != nil {
			return err
		}
		section = Section{
			Activity: activityView(f), Items: d.views(views), Balance: d.balance(balanceOf(f, views)), Grid: d.grid(Grid(f, views)),
			Access: authz.LineItemAccessOf(ctx, f.Access), // 014 FR-011
		}
		return nil
	})
	if err != nil {
		return Section{}, err
	}
	return section, nil
}

// FormOptions returns what a new line item of the activity activityID starts with and the choices the
// line-item form offers (FR-008, FR-011, FR-015): the defaults, each with its source; the active currencies,
// USD first; the plan's year and every year with active finance budget items; their active items, the plan's
// default finance budget first. Its balance stays in USD, the currency the form calculates in (D-213).
// Every signed-in person may add line items (D-004), so a is not used yet.
func (s *Service) FormOptions(ctx context.Context, a actor.Actor, activityID string) (FormOptions, error) {
	var options FormOptions
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.facts(ctx, activityID)
		if err != nil {
			return err
		}
		items, err := s.repo.ListByActivity(ctx, activityID)
		if err != nil {
			return err
		}
		views, err := s.views(ctx, f, items)
		if err != nil {
			return err
		}
		options = FormOptions{Activity: activityView(f), Balance: balanceOf(f, views)}
		if options.Defaults, err = s.defaults(ctx, f, items, options.Balance); err != nil {
			return err
		}
		if options.Currencies, err = s.activeCurrencies(ctx); err != nil {
			return err
		}
		budgetItems, err := s.deps.FinanceBudgets.Items(ctx)
		if err != nil {
			return err
		}
		options.FinanceBudgetYears, options.FinanceBudgetItems = budgetChoices(f, budgetItems)
		options.Defaults.FinanceBudgetItemID, options.Defaults.FinanceBudgetItemPreselected = preselectedItem(f, budgetItems)
		return nil
	})
	if err != nil {
		return FormOptions{}, err
	}
	return options, nil
}

// defaults are the values of a new line item (FR-008): the name "<activity name> – line item" (" 2", " 3", …
// while taken), the activity's owner when active, USD, the unassigned planned cost when it is above 0.00, the
// plan's default spread and year, and Request SWBS code from GSAP on. The finance budget item is filled in by
// the caller.
func (s *Service) defaults(ctx context.Context, f ActivityFacts, items []LineItem, balance UnassignedBalance) (Defaults, error) {
	d := Defaults{
		Name:              uniqueName(f.Name, defaultNameSuffix, items, maxNameLength),
		Currency:          money.BaseCurrency,
		PlannedCostSource: SourceNoActivityPlannedCost,
		SpreadMethod:      defaultSpread(f),
		FinanceBudgetYear: f.PlanYear,
		RequestSwbs:       true,
	}
	if f.OwnerID != "" {
		owner, found, err := s.deps.People.Person(ctx, f.OwnerID)
		if err != nil {
			return Defaults{}, err
		}
		if found && owner.Active { // a new choice must be active, so an inactive owner is no default
			d.Owner = &owner
		}
	}
	switch balance.State {
	case BalanceLeft:
		d.PlannedCost, d.PlannedCostSource = balance.Balance, SourceUnassignedBalance
	case BalanceAllAssigned, BalanceExceeded:
		d.PlannedCostSource = SourceNothingLeftToAssign
	}
	return d, nil
}

// defaultNameSuffix is the suffix of the n-th default name: " – line item", " – line item 2", …
func defaultNameSuffix(n int) string {
	if n == 1 {
		return " – line item"
	}
	return fmt.Sprintf(" – line item %d", n)
}

// activeCurrencies returns the currencies a new choice may use: the active ones, USD first, then by code.
func (s *Service) activeCurrencies(ctx context.Context) ([]CurrencyView, error) {
	all, err := s.deps.Currencies.Currencies(ctx)
	if err != nil {
		return nil, err
	}
	views := []CurrencyView{}
	for _, c := range all {
		if c.Active {
			views = append(views, currencyView(c))
		}
	}
	slices.SortFunc(views, func(x, y CurrencyView) int {
		return cmp.Or(cmp.Compare(rankUSD(x.Code), rankUSD(y.Code)), strings.Compare(x.Code, y.Code))
	})
	return views, nil
}

// rankUSD puts USD before every other currency.
func rankUSD(code string) int {
	if code == money.BaseCurrency {
		return 0
	}
	return 1
}

// budgetChoices returns the finance budget years the form offers — the plan's year and every year with
// active items, ascending — and the active items of those years: by year, then the plan's default finance
// budget first in the plan's year, then by budget name and item name (FR-015).
func budgetChoices(f ActivityFacts, items []BudgetItem) ([]int, []FinanceBudgetItemRef) {
	years := yearsWithActiveItems(items)
	if !slices.Contains(years, f.PlanYear) {
		years = append(years, f.PlanYear)
		slices.Sort(years)
	}
	choices := []FinanceBudgetItemRef{}
	for _, item := range items {
		if item.Active {
			choices = append(choices, itemRef(item))
		}
	}
	isDefault := func(item FinanceBudgetItemRef) int {
		if item.Year == f.PlanYear && f.DefaultFinanceBudget != "" && item.BudgetName == f.DefaultFinanceBudget {
			return 0
		}
		return 1
	}
	slices.SortFunc(choices, func(x, y FinanceBudgetItemRef) int {
		return cmp.Or(cmp.Compare(x.Year, y.Year), cmp.Compare(isDefault(x), isDefault(y)),
			strings.Compare(x.BudgetName, y.BudgetName), strings.Compare(x.Name, y.Name))
	})
	return years, choices
}

// preselectedItem returns the only active item of the plan's default finance budget in the plan's year, when
// it has exactly one (R12).
func preselectedItem(f ActivityFacts, items []BudgetItem) (*string, bool) {
	var only []string
	for _, item := range items {
		if item.Active && item.Year == f.PlanYear && f.DefaultFinanceBudget != "" && item.BudgetName == f.DefaultFinanceBudget {
			only = append(only, item.ID)
		}
	}
	if len(only) != 1 {
		return nil, false
	}
	return &only[0], true
}

// Create adds a line item to the activity activityID (FR-009, FR-010): every problem of the form at once;
// on a Submitted or Approved activity only with in.RestartApproval, which then sends the activity back to
// Created (FR-031); a new ID L-…; one "created" log entry; all in one Write.
func (s *Service) Create(ctx context.Context, a actor.Actor, activityID string, in Input) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, err := s.facts(ctx, activityID)
		if err != nil {
			return err
		}
		siblings, err := s.repo.ListByActivity(ctx, activityID)
		if err != nil {
			return err
		}
		li, problems, err := s.check(ctx, f, siblings, in, nil)
		if err != nil {
			return err
		}
		if err := confirmRestart(problems, f, in.RestartApproval); err != nil {
			return err
		}
		if li.ID, err = s.ids.Next(ctx, ids.PrefixLineItem); err != nil {
			return err
		}
		li.ActivityID, li.Meta = f.ID, record.NewMeta(a, s.now())
		if err := s.repo.Put(ctx, li); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: lineItemKind, RecordID: li.ID, RecordName: li.Name, Action: history.ActionCreated,
		}); err != nil {
			return err
		}
		if f, err = s.restart(ctx, a, f, fmt.Sprintf(addedReason, li.Name, li.ID, li.PlannedCost.Format(), li.Currency)); err != nil {
			return err
		}
		view, err = s.shownView(ctx, a, f, li)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return view, nil
}

// Get returns the line item id with its activity and the activity's unassigned planned cost now (FR-026),
// its amounts in USD answered in a's display currency (FR-005, D-213).
// A line item that is gone answers 404 with why (D-154): its activity was deleted (it went to the recycle bin
// with it), or the line item itself no longer exists.
func (s *Service) Get(ctx context.Context, a actor.Actor, id string) (Details, error) {
	var details Details
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		li, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return s.gone(ctx, a, id)
		}
		f, err := s.facts(ctx, li.ActivityID)
		if err != nil {
			return err
		}
		items, err := s.repo.ListByActivity(ctx, li.ActivityID)
		if err != nil {
			return err
		}
		views, err := s.views(ctx, f, items)
		if err != nil {
			return err
		}
		d, err := s.displayOf(ctx, a)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(views, func(v View) bool { return v.ID == id })
		details = Details{LineItem: d.view(views[i]), Activity: activityView(f), Balance: d.balance(balanceOf(f, views)),
			Access: authz.LineItemAccessOf(ctx, f.Access)} // 014 FR-011
		return nil
	})
	if err != nil {
		return Details{}, err
	}
	return details, nil
}

// gone answers a request about the line item id that is not stored (D-154): "This activity was deleted — …"
// when it is in the recycle bin inside its activity's (or a level above's) entry, otherwise "This line item
// no longer exists — …".
func (s *Service) gone(ctx context.Context, a actor.Actor, id string) error {
	entries, err := s.history.ListBin(ctx, a)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		withActivity := entry.RecordType != lineItemKind && slices.ContainsFunc(entry.Snapshots, func(snapshot history.Snapshot) bool {
			return snapshot.Kind == lineItemKind && snapshot.ID == id
		})
		if withActivity {
			return errs.NotFound(activityGoneMessage)
		}
	}
	return errs.NotFound(lineItemGoneMessage)
}

// Update saves the whole form of the line item id (FR-027, D-153): the version it started from is needed
// (missing: reported with every other problem; out of date: version_conflict naming who changed it); an
// inactive owner, currency or finance budget item already stored may stay (FR-011, FR-016); the custom months
// are kept only for a custom spread (FR-018) and stay when the cost or currency changes (FR-022). A save that
// changes nothing creates no version and no log entry. Changing the planned cost or the currency of a line
// item of a Submitted or Approved activity needs u.RestartApproval and sends the activity back to Created
// (FR-031); other changes never change its status (FR-032). One "changed" entry lists every change, old and
// new.
func (s *Service) Update(ctx context.Context, a actor.Actor, id string, u Update) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		before, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return s.gone(ctx, a, id)
		}
		f, err := s.facts(ctx, before.ActivityID)
		if err != nil {
			return err
		}
		siblings, err := s.siblings(ctx, before)
		if err != nil {
			return err
		}
		after, problems, err := s.check(ctx, f, siblings, u.input(), &before)
		if err != nil {
			return err
		}
		if u.Version == nil || *u.Version < 1 {
			problems.Add("version", reloadMessage)
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := before.CheckVersion(*u.Version, lineItemSingular); err != nil {
			return err
		}
		after.ID, after.ActivityID, after.Meta = before.ID, before.ActivityID, before.Meta
		changes, err := s.changes(before, after)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			view, err = s.shownView(ctx, a, f, before)
			return err
		}
		costChanged := after.PlannedCost != before.PlannedCost || after.Currency != before.Currency
		if costChanged {
			if err := confirmRestart(nil, f, u.RestartApproval); err != nil {
				return err
			}
		}
		after.Touch(a, s.now())
		if err := s.repo.Put(ctx, after); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: lineItemKind, RecordID: after.ID, RecordName: after.Name, Action: history.ActionChanged, Changes: changes,
		}); err != nil {
			return err
		}
		if costChanged {
			reason := fmt.Sprintf(changedReason, after.Name, after.ID, before.PlannedCost.Format(), before.Currency,
				after.PlannedCost.Format(), after.Currency)
			if f, err = s.restart(ctx, a, f, reason); err != nil {
				return err
			}
		}
		view, err = s.shownView(ctx, a, f, after)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return view, nil
}

// siblings returns the other line items of li's activity.
func (s *Service) siblings(ctx context.Context, li LineItem) ([]LineItem, error) {
	items, err := s.repo.ListByActivity(ctx, li.ActivityID)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(items, func(other LineItem) bool { return other.ID == li.ID }), nil
}

// changes lists what differs between two versions of a line item, for the log (data model §7.3): every
// changed field, and each Shell attribute as a field of its own, "attributes.<id>" (as 002's Diff).
func (s *Service) changes(before, after LineItem) ([]history.Change, error) {
	b, x := before, after
	b.Attributes, x.Attributes = nil, nil
	changes, err := history.Diff(b, x)
	if err != nil {
		return nil, err
	}
	ids := slices.Sorted(maps.Keys(before.Attributes))
	for id := range after.Attributes {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		old, value := before.Attributes[id], after.Attributes[id]
		if !sameValue(old, value) {
			changes = append(changes, history.Change{Field: "attributes." + id, Old: old, New: value})
		}
	}
	return changes, nil
}

// sameValue reports whether two stored attribute values are the same: equal JSON, ignoring spaces, or both
// empty.
func sameValue(x, y json.RawMessage) bool {
	compact := func(raw json.RawMessage) string {
		var b bytes.Buffer
		if json.Compact(&b, raw) != nil {
			return string(raw)
		}
		switch b.String() {
		case "", "null", `""`, "[]", "{}":
			return ""
		}
		return b.String()
	}
	return compact(x) == compact(y)
}

// Duplicate copies the line item id into its activity with every value, including the spread, a new ID and
// the name "<name> (copy)", " (copy 2)", … (FR-029, R13: there are no GSAP or finance values to leave out in
// this phase). It counts as adding a line item: on a Submitted or Approved activity only with
// in.RestartApproval (FR-031). One "copied" log entry names the original.
func (s *Service) Duplicate(ctx context.Context, a actor.Actor, id string, in DuplicateInput) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		li, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return s.gone(ctx, a, id)
		}
		f, err := s.facts(ctx, li.ActivityID)
		if err != nil {
			return err
		}
		if err := confirmRestart(nil, f, in.RestartApproval); err != nil {
			return err
		}
		siblings, err := s.repo.ListByActivity(ctx, li.ActivityID)
		if err != nil {
			return err
		}
		dup := copyOf(li)
		dup.Name = uniqueName(li.Name, copyNameSuffix, siblings, maxNameLength)
		if dup.ID, err = s.ids.Next(ctx, ids.PrefixLineItem); err != nil {
			return err
		}
		dup.Meta = record.NewMeta(a, s.now())
		if err := s.repo.Put(ctx, dup); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: lineItemKind, RecordID: dup.ID, RecordName: dup.Name, Action: history.ActionCopied,
			Details: fmt.Sprintf(copiedDetails, li.Name, li.ID),
		}); err != nil {
			return err
		}
		if f, err = s.restart(ctx, a, f, fmt.Sprintf(addedReason, dup.Name, dup.ID, dup.PlannedCost.Format(), dup.Currency)); err != nil {
			return err
		}
		view, err = s.shownView(ctx, a, f, dup)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return view, nil
}

// copiedDetails are the details of the "copied" log entry of a duplicate.
const copiedDetails = "Copied from '%s' (%s)"

// copyNameSuffix is the suffix of the n-th copy's name: " (copy)", " (copy 2)", …
func copyNameSuffix(n int) string {
	if n == 1 {
		return " (copy)"
	}
	return fmt.Sprintf(" (copy %d)", n)
}

// copyOf returns a copy of li that shares no list, map or pointer with it.
func copyOf(li LineItem) LineItem {
	dup := li
	dup.Spread.Months = slices.Clone(li.Spread.Months)
	dup.Attributes = maps.Clone(li.Attributes)
	if li.ContractedAmount != nil {
		contracted := *li.ContractedAmount
		dup.ContractedAmount = &contracted
	}
	if li.FinanceBudgetYear != nil {
		year := *li.FinanceBudgetYear
		dup.FinanceBudgetYear = &year
	}
	return dup
}

// Delete moves the line item id to the recycle bin and returns the bin entry, so the web app can offer Undo
// (FR-030). On a Submitted or Approved activity it needs restartApproval, and sends the activity back to
// Created (FR-031).
func (s *Service) Delete(ctx context.Context, a actor.Actor, id string, restartApproval bool) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		li, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return s.gone(ctx, a, id)
		}
		f, err := s.facts(ctx, li.ActivityID)
		if err != nil {
			return err
		}
		if err := confirmRestart(nil, f, restartApproval); err != nil {
			return err
		}
		if entry, err = s.history.MoveToBin(ctx, a, lineItemKind, id); err != nil {
			return err
		}
		_, err = s.restart(ctx, a, f, fmt.Sprintf(deletedReason, li.Name, li.ID, li.PlannedCost.Format(), li.Currency))
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// confirmRestart returns the problems of a form, or, when there are none, refuses a change that would send a
// Submitted or Approved activity back for approval without the person's confirmation (FR-031, D-146, D-189).
func confirmRestart(problems errs.Problems, f ActivityFacts, confirmed bool) error {
	if err := problems.Err(); err != nil {
		return err
	}
	if f.restartsApproval() && !confirmed {
		return errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartMessage})
	}
	return nil
}

// restart sends the activity back to Created with reason when it is Submitted or Approved (FR-031), and
// returns its facts as they are afterwards.
func (s *Service) restart(ctx context.Context, a actor.Actor, f ActivityFacts, reason string) (ActivityFacts, error) {
	if !f.restartsApproval() {
		return f, nil
	}
	if err := s.deps.Activities.RestartApproval(ctx, a, f.ID, []string{reason}); err != nil {
		return ActivityFacts{}, err
	}
	return s.facts(ctx, f.ID)
}

// facts returns the facts of the activity activityID, or the "activity was deleted" answer (404) when it is
// not stored.
func (s *Service) facts(ctx context.Context, activityID string) (ActivityFacts, error) {
	f, found, err := s.deps.Activities.Facts(ctx, activityID)
	if err != nil {
		return ActivityFacts{}, err
	}
	if !found {
		return ActivityFacts{}, errs.NotFound(activityGoneMessage)
	}
	return f, nil
}

// shownView returns li of the activity f as an answer shows it to a: its view, with its USD planned cost
// also in a's display currency (D-213).
func (s *Service) shownView(ctx context.Context, a actor.Actor, f ActivityFacts, li LineItem) (View, error) {
	v, err := s.view(ctx, f, li)
	if err != nil {
		return View{}, err
	}
	d, err := s.displayOf(ctx, a)
	if err != nil {
		return View{}, err
	}
	return d.view(v), nil
}

// views returns the views of the activity's line items, in the order given.
func (s *Service) views(ctx context.Context, f ActivityFacts, items []LineItem) ([]View, error) {
	views := make([]View, 0, len(items))
	for _, li := range items {
		v, err := s.view(ctx, f, li)
		if err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, nil
}

// view returns li of the activity f as the API shows it, with everything derived on read (D-140, D-141):
// its currency with the current rate, its USD planned cost, its spread in the currency and in USD, its owner,
// its finance budget item, what is missing and its SWBS request status.
func (s *Service) view(ctx context.Context, f ActivityFacts, li LineItem) (View, error) {
	currency, found, err := s.deps.Currencies.Currency(ctx, li.Currency)
	if err != nil {
		return View{}, err
	}
	if !found { // Settings never deletes a currency in use (domain model §6)
		return View{}, fmt.Errorf("lineitems: line item %s has the unknown currency %q", li.ID, li.Currency)
	}
	spread := spreadView(li, f, currency.Rate)
	v := View{
		ID: li.ID, ActivityID: li.ActivityID, Name: li.Name, Description: li.Description,
		Currency:          currencyView(currency),
		PlannedCost:       money.Money{Amount: li.PlannedCost, Currency: li.Currency},
		PlannedCostUSD:    money.USD(PlannedCostUSD(li, currency.Rate)),
		Spread:            spread,
		Attributes:        orNoValues(li.Attributes),
		RequestSwbs:       li.RequestSwbs,
		SwbsStatus:        SwbsStatusOf(li, f),
		FinanceBudgetYear: li.FinanceBudgetYear,
		Meta:              li.Meta,
	}
	v.PlannedCostDisplay = v.PlannedCostUSD // in USD until an answer converts it (display.view)
	if li.ContractedAmount != nil {
		v.ContractedAmount = &money.Money{Amount: *li.ContractedAmount, Currency: li.Currency}
	}
	if li.OwnerID != "" {
		owner, found, err := s.deps.People.Person(ctx, li.OwnerID)
		if err != nil {
			return View{}, err
		}
		if found {
			v.Owner = &owner
		}
	}
	var item *BudgetItem
	if li.FinanceBudgetItemID != "" {
		stored, found, err := s.deps.FinanceBudgets.Item(ctx, li.FinanceBudgetItemID)
		if err != nil {
			return View{}, err
		}
		if found {
			ref := itemRef(stored)
			item, v.FinanceBudgetItem = &stored, &ref
		}
	}
	attributeMissing, err := s.deps.Attributes.Missing(ctx, li.Attributes, f.BusinessUnit)
	if err != nil {
		return View{}, err
	}
	v.Missing = Missing(li, attributeMissing, item, spread.AddsUp)
	return v, nil
}

// spreadView returns the spread of li as the API shows it: the months in its currency and in USD at rate,
// each marked when it lies outside the activity's dates, and for a custom spread its total and whether it
// adds up to the planned cost (FR-017 – FR-020).
func spreadView(li LineItem, f ActivityFacts, rate money.Rate) SpreadView {
	months, spread := SpreadMonths(li, f)
	usd := USDMonths(months, rate)
	view := SpreadView{Method: li.Spread.Method, State: StateSpread, Months: []SpreadMonth{}, AddsUp: true}
	if !spread {
		view.State = StateNotSpread
	}
	var total money.Amount
	for i, m := range months {
		view.Months = append(view.Months, SpreadMonth{
			Month:                m.Month,
			Amount:               money.Money{Amount: m.Amount, Currency: li.Currency},
			AmountUSD:            money.USD(usd[i]),
			OutsideActivityDates: OutsideActivityDates(m.Month, f),
		})
		total = total.Add(m.Amount)
	}
	if li.Spread.Method == SpreadCustom {
		view.CustomTotal = &money.Money{Amount: total, Currency: li.Currency}
		view.AddsUp = total.Cmp(li.PlannedCost) == 0
	}
	return view
}

// balanceOf returns the unassigned planned cost of the activity f whose line items are views.
func balanceOf(f ActivityFacts, views []View) UnassignedBalance {
	usd := make([]money.Amount, 0, len(views))
	currencies := make([]string, 0, len(views))
	for _, v := range views {
		usd = append(usd, v.PlannedCostUSD.Amount)
		currencies = append(currencies, v.Currency.Code)
	}
	return Balance(f, usd, currencies)
}

// activityView returns what the line-item screens show of the activity f (LineItemActivity).
func activityView(f ActivityFacts) ActivityView {
	v := ActivityView{
		ID: f.ID, Name: f.Name, Status: f.Status, RestartsApproval: f.restartsApproval(),
		StartDate: f.StartDate, EndDate: f.EndDate, Ongoing: f.Ongoing,
		Path: orEmpty(f.Path), PlanYear: f.PlanYear, DefaultSpreadMethod: defaultSpread(f),
		BusinessUnitOptionIDs: orEmpty(f.BusinessUnit),
	}
	if f.PlannedCost != nil {
		planned := money.USD(*f.PlannedCost)
		v.PlannedCost = &planned
	}
	if f.DefaultFinanceBudget != "" {
		name := f.DefaultFinanceBudget
		v.DefaultFinanceBudget = &name
	}
	return v
}

// defaultSpread returns the plan's default line-item spread, Evenly when the plan has none.
func defaultSpread(f ActivityFacts) SpreadMethod {
	return cmp.Or(f.DefaultSpread, SpreadEvenly)
}

// currencyView returns a currency as the line-item screens show it.
func currencyView(c settings.Currency) CurrencyView {
	return CurrencyView{Code: c.Code, Name: c.Name, Symbol: c.Symbol, RatePerUSD: c.Rate, Active: c.Active}
}

// itemRef returns a finance budget item as the line-item screens show it.
func itemRef(item BudgetItem) FinanceBudgetItemRef {
	return FinanceBudgetItemRef{
		ID: item.ID, Year: item.Year, BudgetName: item.BudgetName, Name: item.Name, Code: item.Code, Active: item.Active,
	}
}

// uniqueName returns the first of base + suffix(1), base + suffix(2), … that no line item among items has
// (ignoring letter case). With a limit above 0, base is shortened so that the name fits limit characters.
func uniqueName(base string, suffix func(n int) string, items []LineItem, limit int) string {
	for n := 1; ; n++ {
		end := suffix(n)
		start := base
		if limit > 0 {
			start = shorten(base, limit-utf8.RuneCountInString(end))
		}
		if _, taken := named(items, start+end); !taken {
			return start + end
		}
	}
}

// shorten returns the first n characters of text, without spaces at their end.
func shorten(text string, n int) string {
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	return strings.TrimRight(string([]rune(text)[:max(n, 0)]), " ")
}
