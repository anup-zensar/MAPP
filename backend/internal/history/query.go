package history

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
)

// planKind is the name of the kind of plans (feature 004): the Plan filter offers the plans the log names.
const planKind = "plan"

// An "emptied the recycle bin" entry is about no record: its record type, label and name.
const (
	emptiedBinType  = "recycleBin"
	emptiedBinLabel = "recycle bin"
	emptiedBinName  = "Recycle bin"
)

// Messages of the plan's "Errors and messages (API)" table.
const (
	actionMessage    = "Choose an action from the list."
	endBeforeMessage = "The end date must be on or after the start date."
	dateMessage      = "Choose a valid date."
	reloadMessage    = "Reload the page and try again."
)

// ActionFilters are the values of the Action filter (FR-003), in the order the filter offers them. Each
// matches the entries of its own action; ActionDeletedPermanently also matches an emptied recycle bin.
var ActionFilters = []Action{
	ActionCreated, ActionChanged, ActionStatusChanged, ActionMoved, ActionCopied, ActionDeleted,
	ActionRestored, ActionDeletedPermanently,
}

// problems returns every problem of the query (all at once, FR-003): an action that is not a filter value,
// an end that is not after the start, a negative BeforeSeq and a limit outside 1 to 200.
func (q LogQuery) problems() errs.Problems {
	var problems errs.Problems
	for _, action := range q.Actions {
		if !slices.Contains(ActionFilters, Action(action)) {
			problems.Add("action", actionMessage)
			break // one message for the field is enough
		}
	}
	if q.From != nil && q.To != nil && !q.To.After(*q.From) {
		problems.Add("to", endBeforeMessage)
	}
	if q.BeforeSeq < 0 {
		problems.Add("beforeSeq", reloadMessage)
	}
	if q.Limit < 1 || q.Limit > maxLimit {
		problems.Add("limit", limitMessage)
	}
	return problems
}

// QueryLog returns the newest activity-log entries that match q (FR-003, FR-004, FR-009): every filter
// applies to the whole log, the entries come newest first by sequence number, and Load more passes the
// sequence number of the last entry it has as BeforeSeq. Total counts every matching entry, ignoring
// BeforeSeq and Limit, and UnfilteredTotal the whole log. Each entry says where its record is now (FR-008).
// Nothing is written: entries cannot be changed (FR-010). The log is the part the person acting may see
// (014 FR-007, FR-038, D-238): the entries the access hook hides are left out of the items and of both
// counts.
func (s *Service) QueryLog(ctx context.Context, a actor.Actor, q LogQuery) (LogPage, error) {
	if err := q.problems().Err(); err != nil {
		return LogPage{}, err
	}
	var page LogPage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		entries, err := s.repo.Entries(ctx)
		if err != nil {
			return err
		}
		entries = s.visibleEntries(ctx, entries) // 014: only what the person may see
		matches := newLogMatcher(q, entries)
		// Newest first. The sequence number orders entries, also those saved in the same millisecond. The
		// log is appended in that order, so going through it backwards mostly finds them sorted already;
		// the positions are sorted, not the entries, which are large (SC-006).
		matching := []int{}
		for i := range slices.Backward(entries) {
			if matches(&entries[i]) {
				matching = append(matching, i)
			}
		}
		slices.SortFunc(matching, func(x, y int) int { return cmp.Compare(entries[y].Seq, entries[x].Seq) }) // seq is unique

		page = LogPage{Items: []EntryView{}, Total: len(matching), UnfilteredTotal: len(entries)}
		if q.BeforeSeq > 0 {
			older := slices.IndexFunc(matching, func(i int) bool { return entries[i].Seq < q.BeforeSeq })
			if older < 0 {
				older = len(matching) // nothing older is left
			}
			matching = matching[older:]
		}
		index, err := s.binIndex(ctx)
		if err != nil {
			return err
		}
		for _, i := range matching[:min(q.Limit, len(matching))] {
			e := entries[i]
			now, err := s.recordNow(ctx, e, index)
			if err != nil {
				return err
			}
			page.Items = append(page.Items, EntryView{Entry: e, Items: orEmpty(e.Items), RecordNow: now})
		}
		if q.RecordID != "" {
			page.Record, err = s.recordOf(ctx, q.RecordID, entries)
		}
		return err
	})
	if err != nil {
		return LogPage{}, err
	}
	return page, nil
}

// newLogMatcher returns whether an entry matches every filter of q. The scopes of the Plan filter and of
// a record with the items inside it are worked out once, from the whole log.
func newLogMatcher(q LogQuery, entries []Entry) func(*Entry) bool {
	actors, types := setOf(q.ActorIDs), setOf(q.RecordTypes)
	actions := map[Action]bool{}
	for _, action := range q.Actions {
		actions[Action(action)] = true
	}
	if actions[ActionDeletedPermanently] {
		actions[ActionEmptiedRecycleBin] = true // both show the chip Permanently deleted (FR-003)
	}
	var planScope, recordScope map[string]bool
	if len(q.PlanIDs) > 0 {
		planScope = scopeOf(q.PlanIDs, entries)
	}
	if q.RecordID != "" && q.IncludeInside {
		recordScope = scopeOf([]string{q.RecordID}, entries)
	}
	search := strings.ToLower(strings.TrimSpace(q.Search))

	return func(e *Entry) bool {
		switch {
		case len(actors) > 0 && !actors[e.Actor.ID],
			len(types) > 0 && !types[e.RecordType],
			len(actions) > 0 && !actions[e.Action],
			planScope != nil && !inScope(e, planScope),
			recordScope != nil && !inScope(e, recordScope),
			q.RecordID != "" && recordScope == nil && e.RecordID != q.RecordID,
			q.From != nil && e.At.Before(*q.From),
			q.To != nil && !e.At.Before(*q.To):
			return false
		}
		return search == "" ||
			strings.Contains(strings.ToLower(e.RecordName), search) ||
			strings.Contains(strings.ToLower(e.RecordID), search) ||
			strings.Contains(strings.ToLower(e.Actor.Name), search)
	}
}

// scopeOf returns the IDs of the chosen records and of every record that was moved in below one of them
// (D-178, data model §8): a "moved" entry is logged once the record is in its new place, so its path names
// the new place. A record moved in below a record that was itself moved in later counts too. With
// inScope, an entry is in the scope when it is about one of these records or sits below one of them: the
// records that are inside now and those that were inside once, deleted and permanently deleted ones
// included, because entries keep their path at the time.
func scopeOf(ids []string, entries []Entry) map[string]bool {
	scope := setOf(ids)
	var moves []Entry
	for _, e := range entries {
		if e.Action == ActionMoved {
			moves = append(moves, e)
		}
	}
	for grown := true; grown; { // until no move adds a record
		grown = false
		for _, e := range moves {
			if !scope[e.RecordID] && inScope(&e, scope) {
				scope[e.RecordID] = true
				grown = true
			}
		}
	}
	return scope
}

// inScope reports whether the entry is about a record of scope or sits below one of them.
func inScope(e *Entry, scope map[string]bool) bool {
	if scope[e.RecordID] {
		return true
	}
	return slices.ContainsFunc(e.Path, func(r RecordRef) bool { return scope[r.ID] })
}

// recordNow says where the record of the entry is now (FR-008, data model §4): stored, with its current
// name and place; in the recycle bin, with the entry holding it; or gone for good. An emptied recycle bin
// is about no record, and a kind this API does not know cannot be looked up.
func (s *Service) recordNow(ctx context.Context, e Entry, index binIndex) (RecordNow, error) {
	gone := RecordNow{State: RecordNone, Path: []RecordRef{}}
	k, found := s.findKind(e.RecordType)
	if e.Action == ActionEmptiedRecycleBin || !found {
		return gone, nil
	}
	snapshot, stored, err := k.Snapshot(ctx, e.RecordID)
	if err != nil {
		return RecordNow{}, err
	}
	if stored {
		path, err := s.ancestors(ctx, k.Parent, snapshot.ParentID)
		if err != nil {
			return RecordNow{}, err
		}
		return RecordNow{State: RecordExists, Name: snapshot.Name, Path: path}, nil
	}
	if held, inBin := index.held[recordKey{e.RecordType, e.RecordID}]; inBin {
		return RecordNow{State: RecordInRecycleBin, Name: held.name, Path: []RecordRef{}, BinEntryID: held.entry.ID}, nil
	}
	gone.State = RecordDeletedPermanently
	return gone, nil
}

// recordOf returns the record id for the Record filter's chip (FR-013): its kind and latest name in the
// log, as the record of an entry or in an entry's path, and its current name when it is stored. It is nil
// when the log does not name it.
func (s *Service) recordOf(ctx context.Context, id string, entries []Entry) (*RecordRef, error) {
	var ref *RecordRef
	for _, e := range entries { // oldest first: the latest name wins
		if e.RecordID == id {
			ref = &RecordRef{Type: e.RecordType, ID: id, Name: e.RecordName}
		}
		for _, p := range e.Path {
			if p.ID == id {
				ref = &RecordRef{Type: p.Type, ID: id, Name: p.Name}
			}
		}
	}
	if ref == nil {
		return nil, nil
	}
	if k, found := s.findKind(ref.Type); found {
		snapshot, stored, err := k.Snapshot(ctx, id)
		if err != nil {
			return nil, err
		}
		if stored {
			ref.Name = snapshot.Name
		}
	}
	return ref, nil
}

// LogSummary returns the summary cards of the activity log and the choices of its filters (FR-002,
// FR-003), over the whole log, whatever the filters: todayStart is midnight today and weekStart midnight
// of this week's Monday, both in the person's time zone, which only the browser knows (D-179). Both are
// required, and the week cannot start after today. Like QueryLog, it covers only the entries the person
// acting may see (014 FR-007, D-238).
func (s *Service) LogSummary(ctx context.Context, a actor.Actor, todayStart, weekStart time.Time) (LogSummary, error) {
	var problems errs.Problems
	if todayStart.IsZero() {
		problems.Add("todayStart", reloadMessage)
	}
	if weekStart.IsZero() || (!todayStart.IsZero() && weekStart.After(todayStart)) {
		problems.Add("weekStart", reloadMessage)
	}
	if err := problems.Err(); err != nil {
		return LogSummary{}, err
	}
	var entries []Entry
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		all, err := s.repo.Entries(ctx)
		entries = s.visibleEntries(ctx, all) // 014: only what the person may see
		return err
	})
	if err != nil {
		return LogSummary{}, err
	}

	summary := LogSummary{Total: len(entries)}
	activeThisWeek := map[string]bool{}
	people := map[string]actor.Actor{}    // by id, with the latest name
	types := map[string]RecordTypeCount{} // by type, with the latest label
	plans := map[string]RecordRef{}       // by id, with the latest name
	for _, e := range entries {
		if !e.At.Before(todayStart) {
			summary.ChangesToday++
		}
		if !e.At.Before(weekStart) {
			summary.ChangesThisWeek++
			activeThisWeek[e.Actor.ID] = true
			if e.Action == ActionDeleted || e.Action == ActionDeletedPermanently || e.Action == ActionEmptiedRecycleBin {
				summary.DeletionsThisWeek++
			}
		}
		people[e.Actor.ID] = e.Actor
		if e.RecordType != emptiedBinType {
			types[e.RecordType] = RecordTypeCount{Type: e.RecordType, Label: e.RecordTypeLabel, Count: types[e.RecordType].Count + 1}
		}
		if len(e.Path) > 0 && e.Path[0].Type == planKind {
			plans[e.Path[0].ID] = e.Path[0]
		}
		if e.RecordType == planKind {
			plans[e.RecordID] = RecordRef{Type: planKind, ID: e.RecordID, Name: e.RecordName}
		}
	}
	summary.PeopleActiveThisWeek = len(activeThisWeek)

	summary.People = sortedValues(people, func(x, y actor.Actor) int { return compareNames(x.Name, y.Name, x.ID, y.ID) })
	summary.RecordTypes = sortedValues(types, func(x, y RecordTypeCount) int { return compareNames(x.Label, y.Label, x.Type, y.Type) })
	summary.Plans = sortedValues(plans, func(x, y RecordRef) int { return compareNames(x.Name, y.Name, x.ID, y.ID) })
	return summary, nil
}

// compareNames orders by name the way people sort names, ignoring letter case, and then by id, so that the
// order never depends on a map.
func compareNames(nameX, nameY, idX, idY string) int {
	return cmp.Or(cmp.Compare(strings.ToLower(nameX), strings.ToLower(nameY)), cmp.Compare(idX, idY))
}

// sortedValues returns the values of m sorted with compare. The list is never nil.
func sortedValues[K comparable, V any](m map[K]V, compare func(x, y V) int) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	slices.SortFunc(values, compare)
	return values
}

// setOf returns the values that are not empty as a set.
func setOf(values []string) map[string]bool {
	set := map[string]bool{}
	for _, v := range values {
		if v != "" {
			set[v] = true
		}
	}
	return set
}

// orEmpty returns list, or an empty list instead of nil, so the JSON has [] rather than null.
func orEmpty[T any](list []T) []T {
	if list == nil {
		return []T{}
	}
	return list
}

// EntryViews returns entries as the API returns them (ActivityLogEntry), each with where its record is now,
// in the same order: for other features that select entries of their own, such as the recent access changes
// of feature 014. It joins the caller's transaction.
func (s *Service) EntryViews(ctx context.Context, entries []Entry) ([]EntryView, error) {
	views := make([]EntryView, 0, len(entries))
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		index, err := s.binIndex(ctx)
		if err != nil {
			return err
		}
		for _, e := range entries {
			now, err := s.recordNow(ctx, e, index)
			if err != nil {
				return err
			}
			views = append(views, EntryView{Entry: e, Items: orEmpty(e.Items), RecordNow: now})
		}
		return nil
	})
	return views, err
}
