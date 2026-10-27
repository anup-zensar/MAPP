package history

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/race"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/store"
)

// The Person filter keeps the entries by the chosen people only, and Total counts them (011 FR-003).
func TestQueryLog_FilterPerson_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	byPriya := f.logAs(t, priya, parentOne(ActionCreated, ""))
	byDaniel := f.logAs(t, daniel, parentOne(ActionChanged, ""))
	byMei := f.logAs(t, mei, parentOne(ActionChanged, ""))

	assertPage(t, f.query(t, LogQuery{ActorIDs: []string{daniel.ID}, Limit: 50}), []Entry{byDaniel}, 1)
	assertPage(t, f.query(t, LogQuery{ActorIDs: []string{priya.ID, mei.ID}, Limit: 50}), []Entry{byMei, byPriya}, 2)
}

// The Record type filter keeps the entries about the chosen kinds only (011 FR-003).
func TestQueryLog_FilterRecordType_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	parent := f.logAs(t, priya, parentOne(ActionCreated, ""))
	child := f.logAs(t, priya, childOne(ActionCreated))
	grandchild := f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-1", RecordName: "Grandchild one", Action: ActionCreated})

	assertPage(t, f.query(t, LogQuery{RecordTypes: []string{"child"}, Limit: 50}), []Entry{child}, 1)
	assertPage(t, f.query(t, LogQuery{RecordTypes: []string{"parent", "grandchild"}, Limit: 50}), []Entry{grandchild, parent}, 2)
}

// The Action filter keeps the entries of the chosen actions. "Permanently deleted" also finds the emptied
// recycle bins, which show the same chip (011 FR-003, data model §1).
func TestQueryLog_FilterAction_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	created := f.logAs(t, priya, parentOne(ActionCreated, ""))
	published := f.logAs(t, priya, parentOne(ActionStatusChanged, "published"))
	deleted := f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-9", RecordName: "Gone", ParentID: "C-1", Action: ActionDeleted})
	purged := f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-9", RecordName: "Gone", ParentID: "C-1", Action: ActionDeletedPermanently})
	emptied := f.appendRaw(t, daniel, Entry{Action: ActionEmptiedRecycleBin, Verb: verbEmptiedRecycleBin,
		RecordType: emptiedBinType, RecordTypeLabel: "recycle bin", RecordName: "Recycle bin", Details: "1 item"})
	tests := []struct {
		actions []string
		want    []Entry
	}{
		{[]string{"created"}, []Entry{created}},
		{[]string{"statusChanged", "deleted"}, []Entry{deleted, published}},
		{[]string{"deletedPermanently"}, []Entry{emptied, purged}},
		{nil, []Entry{emptied, purged, deleted, published, created}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.actions), func(t *testing.T) {
			assertPage(t, f.query(t, LogQuery{Actions: tt.actions, Limit: 50}), tt.want, len(tt.want))
		})
	}
}

// The date range keeps the entries from From, included, up to To, left out (011 FR-003): the browser sends
// local midnights as moments (D-179).
func TestQueryLog_FilterDateRange_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	dayBefore := f.logAs(t, priya, parentOne(ActionCreated, "")) // 12 Mar 09:00
	f.clock.Advance(15 * time.Hour)                              // 13 Mar 00:00
	midnight := f.logAs(t, priya, parentOne(ActionChanged, ""))
	f.clock.Advance(23*time.Hour + 59*time.Minute) // 13 Mar 23:59
	lateInTheDay := f.logAs(t, priya, parentOne(ActionChanged, ""))
	f.clock.Advance(time.Minute) // 14 Mar 00:00
	nextDay := f.logAs(t, priya, parentOne(ActionChanged, ""))
	from, to := start.Add(15*time.Hour), start.Add(39*time.Hour) // 13 Mar 00:00 and 14 Mar 00:00
	tests := []struct {
		name     string
		from, to *time.Time
		want     []Entry
	}{
		{"from and to", &from, &to, []Entry{lateInTheDay, midnight}},
		{"from only", &from, nil, []Entry{nextDay, lateInTheDay, midnight}},
		{"to only", nil, &from, []Entry{dayBefore}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPage(t, f.query(t, LogQuery{From: tt.from, To: tt.to, Limit: 50}), tt.want, len(tt.want))
		})
	}
}

// The search finds a part of the record's name, of its ID or of the person's name, ignoring letter case
// (011 FR-003).
func TestQueryLog_Search_NameIDPerson_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	parent := f.logAs(t, priya, parentOne(ActionCreated, ""))
	child := f.logAs(t, daniel, childOne(ActionCreated))
	tests := []struct {
		search string
		want   []Entry
	}{
		{"PARENT one", []Entry{parent}},
		{"c-1", []Entry{child}},
		{"okoro", []Entry{child}},
		{"  one ", []Entry{child, parent}},
		{"nothing like it", []Entry{}},
	}

	for _, tt := range tests {
		t.Run(tt.search, func(t *testing.T) {
			assertPage(t, f.query(t, LogQuery{Search: tt.search, Limit: 50}), tt.want, len(tt.want))
		})
	}
}

// Filters apply together and to the whole log, not to a page of it: Total counts every matching entry,
// UnfilteredTotal the whole log ("Showing N of M entries", 011 FR-003).
func TestQueryLog_CombinedFilters_TotalAndUnfiltered_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	var wanted []Entry
	for i := range 6 {
		a := []actor.Actor{priya, daniel}[i%2]
		e := f.logAs(t, a, parentOne(ActionChanged, ""))
		if a == daniel {
			wanted = append(wanted, e)
		}
		f.logAs(t, daniel, childOne(ActionCreated))
	}
	slices.Reverse(wanted)

	page := f.query(t, LogQuery{ActorIDs: []string{daniel.ID}, RecordTypes: []string{"parent"}, Actions: []string{"changed"}, Limit: 2})

	assertPage(t, page, wanted[:2], 3)
	if page.UnfilteredTotal != 12 {
		t.Errorf("unfilteredTotal = %d, want 12", page.UnfilteredTotal)
	}
}

// The Plan filter finds the entries about the plan and about everything that is or was inside it: records
// inside it now, records deleted from it (in the recycle bin or gone for good), with their entries' paths at
// the time (011 FR-003, D-178).
func TestQueryLog_PlanScope_IsOrWasInside_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	f.put(t, childrenTable, newRecord("C-3", "Child of parent two", "P-2"))
	planCreated := f.logAs(t, priya, parentOne(ActionCreated, ""))
	childCreated := f.logAs(t, priya, childOne(ActionCreated))
	grandchildChanged := f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-3", RecordName: "Grandchild three", Action: ActionChanged})
	f.logAs(t, priya, LogInput{Kind: "parent", RecordID: "P-2", RecordName: "Parent two", Action: ActionCreated})
	f.logAs(t, priya, LogInput{Kind: "child", RecordID: "C-3", RecordName: "Child of parent two", Action: ActionCreated})
	goneChanged := f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-2", RecordName: "Grandchild two", Action: ActionChanged})
	f.moveToBin(t, "grandchild", "G-1") // in the recycle bin
	deleted := f.lastEntry(t)
	f.removeForGood(t, grandchildrenTable, "G-2") // gone for good: not stored and not in the bin

	page := f.query(t, LogQuery{PlanIDs: []string{"P-1"}, Limit: 50})

	assertPage(t, page, []Entry{deleted, goneChanged, grandchildChanged, childCreated, planCreated}, 5)
}

// A record moved into the plan later belongs to it with all its entries, also those from before the move
// and those of the records below it (011 FR-003, D-178).
func TestQueryLog_MovedInRecordMatches_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	f.put(t, childrenTable, newRecord("C-3", "Child of parent two", "P-2"))
	f.put(t, grandchildrenTable, newRecord("G-9", "Grandchild of C-3", "C-3"))
	beforeMove := f.logAs(t, priya, LogInput{Kind: "child", RecordID: "C-3", RecordName: "Child of parent two", Action: ActionCreated})
	below := f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-9", RecordName: "Grandchild of C-3", Action: ActionCreated})
	f.put(t, childrenTable, newRecord("C-3", "Child of parent two", "P-1")) // moved to P-1, then logged
	moved := f.logAs(t, priya, LogInput{Kind: "child", RecordID: "C-3", RecordName: "Child of parent two", Action: ActionMoved})

	page := f.query(t, LogQuery{PlanIDs: []string{"P-1"}, Limit: 50})

	assertPage(t, page, []Entry{moved, below, beforeMove}, 3)
	if other := f.query(t, LogQuery{PlanIDs: []string{"P-2"}, Limit: 50}); !slices.Equal(pageIDs(other), []string{below.ID, beforeMove.ID}) {
		t.Errorf("plan P-2 = %v, want only the entries from while C-3 was inside it: %v", pageIDs(other), []string{below.ID, beforeMove.ID})
	}
}

// A record's history shows the entries about the record only, or with "Include items inside" also those of
// every record that is or was below it: records inside it now, a record deleted from it, and a record moved
// in from elsewhere with its entries from before the move; never its parent's or a sibling's (011 FR-012,
// FR-013).
func TestQueryLog_RecordWithItemsInside_FR012_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.logAs(t, priya, parentOne(ActionChanged, ""))                                     // its parent
	f.logAs(t, priya, LogInput{Kind: "child", RecordID: "C-2", RecordName: "Child two", // a sibling
		Action: ActionChanged})
	movedInCreated := f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-3", RecordName: "Grandchild three",
		Action: ActionCreated}) // under C-2 then
	childChanged := f.logAs(t, priya, childOne(ActionChanged))
	insideChanged := f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-1", RecordName: "Grandchild one",
		Action: ActionChanged})
	f.moveToBin(t, "grandchild", "G-2") // deleted from C-1
	deleted := f.lastEntry(t)
	f.put(t, grandchildrenTable, newRecord("G-3", "Grandchild three", "C-1")) // moved to C-1, then logged
	movedIn := f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-3", RecordName: "Grandchild three",
		Action: ActionMoved})
	childRenamed := f.logAs(t, daniel, childOne(ActionChanged))

	recordOnly := f.query(t, LogQuery{RecordID: "C-1", Limit: 50})
	withInside := f.query(t, LogQuery{RecordID: "C-1", IncludeInside: true, Limit: 50})

	assertPage(t, recordOnly, []Entry{childRenamed, childChanged}, 2)
	assertPage(t, withInside, []Entry{childRenamed, movedIn, deleted, insideChanged, childChanged, movedInCreated}, 6)
	want := &RecordRef{Type: "child", ID: "C-1", Name: "Child one"}
	if !reflect.DeepEqual(withInside.Record, want) || !reflect.DeepEqual(recordOnly.Record, want) {
		t.Errorf("record = %+v and %+v, want %+v", recordOnly.Record, withInside.Record, want)
	}
	if other := f.query(t, LogQuery{RecordID: "C-1", IncludeInside: true, ActorIDs: []string{daniel.ID}, Limit: 50}); other.Total != 1 {
		t.Errorf("with the Person filter: total %d, want 1 (the filters apply together)", other.Total)
	}
}

// Entries come newest first by their sequence number, which also orders entries saved in the same
// millisecond (011 FR-004).
func TestQueryLog_NewestFirstBySeq_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	first := f.logAs(t, priya, parentOne(ActionCreated, ""))
	second := f.logAs(t, priya, parentOne(ActionChanged, ""))
	third := f.logAs(t, priya, parentOne(ActionStatusChanged, "published"))

	assertPage(t, f.query(t, LogQuery{Limit: 50}), []Entry{third, second, first}, 3)
}

// Load more asks for the entries older than the last one it has (beforeSeq): pages follow each other
// without gaps or repeats, and the totals stay the same (011 FR-009).
func TestQueryLog_BeforeSeqPaging_FR009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	var all []Entry
	for range 5 {
		all = append(all, f.logAs(t, priya, parentOne(ActionChanged, "")))
		f.logAs(t, daniel, childOne(ActionCreated)) // filtered out below
	}
	slices.Reverse(all)
	q := LogQuery{ActorIDs: []string{priya.ID}, Limit: 2}

	var got []Entry
	for page := range 3 {
		result := f.query(t, q)
		if result.Total != 5 || result.UnfilteredTotal != 10 {
			t.Errorf("page %d: total %d of %d, want 5 of 10", page+1, result.Total, result.UnfilteredTotal)
		}
		for _, item := range result.Items {
			got = append(got, item.Entry)
		}
		if len(result.Items) > 0 {
			q.BeforeSeq = result.Items[len(result.Items)-1].Seq
		}
	}

	assertEntries(t, got, all)
	q.BeforeSeq = all[len(all)-1].Seq // nothing is older
	if last := f.query(t, q); len(last.Items) != 0 || last.Total != 5 {
		t.Errorf("after the oldest entry = %v (total %d), want nothing (total 5)", pageIDs(last), last.Total)
	}
}

// Each entry says where its record is now (011 FR-008, data model §4): stored, with its current name and
// place even after a rename; in the recycle bin, also inside another record's entry, with that entry; gone
// for good; or no record at all.
func TestQueryLog_RecordNowStates_FR008(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.logAs(t, priya, parentOne(ActionCreated, ""))
	f.put(t, parentsTable, newRecord("P-1", "Parent renamed", ""))
	f.logAs(t, priya, childOne(ActionCreated))
	f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-1", RecordName: "Grandchild one", Action: ActionCreated})
	f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-3", RecordName: "Grandchild three", Action: ActionCreated})
	f.removeForGood(t, grandchildrenTable, "G-3")
	f.appendRaw(t, daniel, Entry{Action: ActionEmptiedRecycleBin, Verb: verbEmptiedRecycleBin,
		RecordType: emptiedBinType, RecordTypeLabel: "recycle bin", RecordName: "Recycle bin", Details: "1 item"})
	f.logAs(t, priya, LogInput{Kind: "attachment", RecordID: "X-1", RecordName: "Unknown kind", Action: ActionCreated, ParentID: "P-1"}, f.withKind(t, "attachment"))
	bin := f.moveToBin(t, "child", "C-1") // holds G-1 too

	page := f.query(t, LogQuery{Limit: 50})

	parentNow := []RecordRef{{Type: "parent", ID: "P-1", Name: "Parent renamed"}}
	want := map[string]RecordNow{ // by the entry's record and action
		"parent P-1 created":            {State: RecordExists, Name: "Parent renamed", Path: []RecordRef{}},
		"child C-1 created":             {State: RecordInRecycleBin, Name: "Child one", Path: []RecordRef{}, BinEntryID: bin.ID},
		"child C-1 deleted":             {State: RecordInRecycleBin, Name: "Child one", Path: []RecordRef{}, BinEntryID: bin.ID},
		"grandchild G-1 created":        {State: RecordInRecycleBin, Name: "Grandchild one", Path: []RecordRef{}, BinEntryID: bin.ID},
		"grandchild G-3 created":        {State: RecordDeletedPermanently, Path: []RecordRef{}},
		"recycleBin  emptiedRecycleBin": {State: RecordNone, Path: []RecordRef{}},
		"attachment X-1 created":        {State: RecordNone, Path: []RecordRef{}},
	}
	got := map[string]RecordNow{}
	for _, item := range page.Items {
		got[fmt.Sprintf("%s %s %s", item.RecordType, item.RecordID, item.Action)] = item.RecordNow
		if item.Items == nil {
			t.Errorf("%s: items = nil, want an empty list", item.ID)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recordNow =\n%s\nwant\n%s", asJSON(got), asJSON(want))
	}

	f.put(t, childrenTable, newRecord("C-2", "Child two", "P-1"))
	f.logAs(t, priya, LogInput{Kind: "child", RecordID: "C-2", RecordName: "Child two", Action: ActionChanged})
	if latest := f.query(t, LogQuery{Limit: 1}).Items[0].RecordNow; !reflect.DeepEqual(latest.Path, parentNow) {
		t.Errorf("path of C-2 now = %+v, want its current place %+v", latest.Path, parentNow)
	}
}

// A query for one record names the record for the Record filter's chip: its current name, or its latest
// name in the log when it is no longer stored, also when only the paths of other entries name it; none
// when the log does not know it (011 FR-013).
func TestQueryLog_RecordRef_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.logAs(t, priya, childOne(ActionCreated))
	f.put(t, childrenTable, newRecord("C-1", "Child renamed", "P-1"))
	f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-3", RecordName: "Grandchild three", Action: ActionCreated})
	f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-3", RecordName: "Grandchild 3", Action: ActionChanged})
	f.removeForGood(t, grandchildrenTable, "G-3")
	tests := []struct {
		recordID string
		want     *RecordRef
	}{
		{"C-1", &RecordRef{Type: "child", ID: "C-1", Name: "Child renamed"}},
		{"G-3", &RecordRef{Type: "grandchild", ID: "G-3", Name: "Grandchild 3"}},
		{"P-1", &RecordRef{Type: "parent", ID: "P-1", Name: "Parent one"}}, // only in paths
		{"Q-9", nil},
	}

	for _, tt := range tests {
		t.Run(tt.recordID, func(t *testing.T) {
			if got := f.query(t, LogQuery{RecordID: tt.recordID, Limit: 50}).Record; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("record = %+v, want %+v", got, tt.want)
			}
		})
	}
	if got := f.query(t, LogQuery{Limit: 50}).Record; got != nil {
		t.Errorf("record without recordId = %+v, want nil", got)
	}
}

// Every problem with a query is reported at once (011 FR-003): an action that is not a filter value, an
// end that is not after the start, a negative beforeSeq and a limit outside 1 to 200.
func TestQueryLog_Validation_AllProblemsAtOnce(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	from, to := start, start

	_, err := f.service.QueryLog(t.Context(), priya, LogQuery{
		Actions: []string{"created", "renamed", "emptiedRecycleBin"}, From: &from, To: &to, BeforeSeq: -1, Limit: 201,
	})

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "action", Message: "Choose an action from the list."},
		errs.FieldError{Field: "to", Message: "The end date must be on or after the start date."},
		errs.FieldError{Field: "beforeSeq", Message: "Reload the page and try again."},
		errs.FieldError{Field: "limit", Message: "Choose a number from 1 to 200."},
	))
	for _, action := range ActionFilters {
		if _, err := f.service.QueryLog(t.Context(), priya, LogQuery{Actions: []string{string(action)}, Limit: 1}); err != nil {
			t.Errorf("action %q: %v, want it accepted", action, err)
		}
	}
}

// The summary cards count over the whole log, for the boundaries the browser sends in the person's time
// zone (011 FR-002, D-179): changes since midnight, since Monday, the different people this week, and
// deletes, permanent deletes and emptied recycle bins this week.
func TestLogSummary_Counts_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	weekStart := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC) // Monday: 12 Mar 2026 is a Thursday
	todayStart := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	f.clock.now = weekStart.Add(-time.Minute) // Sunday 23:59, last week
	f.logAs(t, mei, parentOne(ActionCreated, ""))
	f.logAs(t, mei, LogInput{Kind: "grandchild", RecordID: "G-9", RecordName: "Gone", ParentID: "C-1", Action: ActionDeleted})
	f.clock.now = weekStart // Monday 00:00
	f.logAs(t, priya, parentOne(ActionChanged, ""))
	f.logAs(t, priya, LogInput{Kind: "grandchild", RecordID: "G-9", RecordName: "Gone", ParentID: "C-1", Action: ActionDeletedPermanently})
	f.clock.now = todayStart.Add(-time.Millisecond) // yesterday
	f.logAs(t, daniel, parentOne(ActionChanged, ""))
	f.clock.now = todayStart
	f.logAs(t, daniel, childOne(ActionCreated))
	f.moveToBin(t, "grandchild", "G-3")
	f.appendRaw(t, priya, Entry{Action: ActionEmptiedRecycleBin, Verb: verbEmptiedRecycleBin,
		RecordType: emptiedBinType, RecordTypeLabel: "recycle bin", RecordName: "Recycle bin", Details: "1 item"})

	got := f.summary(t, todayStart, weekStart)

	want := [5]int{3, 6, 2, 3, 8} // today, this week, people this week, deletions this week, total
	if counts := [5]int{got.ChangesToday, got.ChangesThisWeek, got.PeopleActiveThisWeek, got.DeletionsThisWeek, got.Total}; counts != want {
		t.Errorf("today, week, people, deletions, total = %v, want %v", counts, want)
	}
}

// The summary does not follow the filters: it always counts the whole log, every person, kind and action
// (011 FR-002), so the cards stay the same while the list below them is filtered.
func TestLogSummary_IgnoresFilters_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.logAs(t, priya, parentOne(ActionCreated, ""))
	f.logAs(t, daniel, childOne(ActionCreated))
	f.logAs(t, mei, LogInput{Kind: "attachment", RecordID: "X-1", RecordName: "Unknown kind", Action: ActionCreated, ParentID: "P-1"}, f.withKind(t, "attachment"))
	filtered := f.query(t, LogQuery{ActorIDs: []string{priya.ID}, Limit: 50})

	got := f.summary(t, start.Add(-9*time.Hour), start.Add(-81*time.Hour))

	if filtered.Total != 1 || got.ChangesToday != 3 || got.ChangesThisWeek != 3 || got.PeopleActiveThisWeek != 3 || got.Total != 3 {
		t.Errorf("summary = %+v while the filtered list has %d; want every entry counted", got, filtered.Total)
	}
}

// The summary offers the filter choices (011 FR-003): everyone with an entry, the record types in the log
// with their latest label and count (not the emptied recycle bin), and the plans the log names, as a record
// or at the top of a path, with their latest names — each sorted by name.
func TestLogSummary_Facets_FR003(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	plans := f.testKind("plan", "plans", "", "plans")
	campaigns := f.testKind("campaign", "campaigns", "plan", "campaigns")
	f.register(t, plans, campaigns)
	f.put(t, "plans", newRecord("P-1001", "FY2026 Lubricants Europe", ""), newRecord("P-1002", "a small plan", ""))
	f.put(t, "campaigns", newRecord("C-1001", "Helix launch", "P-1001"))
	f.logAs(t, priya, LogInput{Kind: "plan", RecordID: "P-1001", RecordName: "FY2026 Lubricants", Action: ActionCreated})
	f.logAs(t, daniel, LogInput{Kind: "plan", RecordID: "P-1002", RecordName: "a small plan", Action: ActionCreated})
	f.logAs(t, mei, LogInput{Kind: "campaign", RecordID: "C-1001", RecordName: "Helix launch", Action: ActionCreated})
	f.logAs(t, actor.Actor{ID: daniel.ID, Name: "Daniel O."}, LogInput{Kind: "campaign", RecordID: "C-1001", RecordName: "Helix launch", Action: ActionChanged})
	f.appendRaw(t, priya, Entry{Action: ActionEmptiedRecycleBin, Verb: verbEmptiedRecycleBin,
		RecordType: emptiedBinType, RecordTypeLabel: "recycle bin", RecordName: "Recycle bin"})
	f.appendRaw(t, priya, Entry{Action: ActionDeleted, Verb: "deleted", RecordType: "campaign", RecordTypeLabel: "marketing campaign",
		RecordID: "C-1002", RecordName: "Pilot", Path: []RecordRef{{Type: "plan", ID: "P-1003", Name: "Old plan"}}})

	got := f.summary(t, start, start)

	wantPeople := []actor.Actor{{ID: daniel.ID, Name: "Daniel O."}, mei, priya}
	wantTypes := []RecordTypeCount{{Type: "campaign", Label: "marketing campaign", Count: 3}, {Type: "plan", Label: "plan", Count: 2}}
	wantPlans := []RecordRef{
		{Type: "plan", ID: "P-1002", Name: "a small plan"},
		{Type: "plan", ID: "P-1001", Name: "FY2026 Lubricants Europe"},
		{Type: "plan", ID: "P-1003", Name: "Old plan"},
	}
	if !reflect.DeepEqual(got.People, wantPeople) || !reflect.DeepEqual(got.RecordTypes, wantTypes) || !reflect.DeepEqual(got.Plans, wantPlans) {
		t.Errorf("facets =\n%s\n%s\n%s\nwant\n%s\n%s\n%s", asJSON(got.People), asJSON(got.RecordTypes), asJSON(got.Plans),
			asJSON(wantPeople), asJSON(wantTypes), asJSON(wantPlans))
	}
}

// Both boundaries are required, and the week cannot start after today; both problems are reported at once
// (011 FR-002).
func TestLogSummary_Validation(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	reload := "Reload the page and try again."

	_, missing := f.service.LogSummary(t.Context(), priya, time.Time{}, time.Time{})
	_, reversed := f.service.LogSummary(t.Context(), priya, start, start.Add(time.Second))

	assertError(t, missing, errs.Validation(errs.FieldError{Field: "todayStart", Message: reload}, errs.FieldError{Field: "weekStart", Message: reload}))
	assertError(t, reversed, errs.Validation(errs.FieldError{Field: "weekStart", Message: reload}))
}

// With 50,000 entries the first page and each filter change answer in under 2 seconds (011 SC-006).
func TestQueryLog_50000Entries_SC006(t *testing.T) {
	if testing.Short() || race.Enabled {
		t.Skip("a timing test: it runs without -short and without the race detector, in CI's Timing tests step")
	}
	f := newFixture(t)
	f.addTree(t)
	people := []actor.Actor{priya, daniel, mei}
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		for i := range 50_000 {
			path := []RecordRef{{Type: "parent", ID: fmt.Sprintf("P-%d", i%40), Name: "Plan"}, {Type: "child", ID: fmt.Sprintf("C-%d", i%400), Name: "Campaign"}}
			_, err := f.service.appendEntry(ctx, people[i%3], Entry{Action: ActionChanged, Verb: "changed",
				RecordType: "grandchild", RecordTypeLabel: "grandchild", RecordID: fmt.Sprintf("G-%d", i), RecordName: fmt.Sprintf("Activity %d", i), Path: path})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	queries := []LogQuery{
		{Limit: 50},
		{ActorIDs: []string{daniel.ID}, Actions: []string{"changed"}, Limit: 50},
		{PlanIDs: []string{"P-7"}, Search: "activity 12", Limit: 50},
		{RecordID: "C-17", IncludeInside: true, Limit: 20},
	}

	runtime.GC() // the garbage of writing 50,000 entries is not the query's

	for _, q := range queries {
		// The best of three tries: the time the query needs, not the time other programs on a busy machine
		// took from it.
		took := time.Duration(math.MaxInt64)
		var page LogPage
		for range 3 {
			began := time.Now()
			page = f.query(t, q)
			took = min(took, time.Since(began))
		}
		// The race detector slows the code several times over, so the 2 seconds are measured without it.
		tooSlow := took > 2*time.Second
		if tooSlow || len(page.Items) == 0 {
			t.Errorf("query %+v took %v for %d entries, want under 2 seconds", q, took, len(page.Items))
		}
	}
}

// childOne is the log input of a change of the child C-1.
func childOne(action Action) LogInput {
	return LogInput{Kind: "child", RecordID: "C-1", RecordName: "Child one", Action: action}
}

// logAs writes an entry as a at the clock's time and stops the test when that fails. With a kind
// registered only for this entry (withKind), the entry is about a kind the service does not know.
func (f *fixture) logAs(t *testing.T, a actor.Actor, in LogInput, service ...*Service) Entry {
	t.Helper()
	s := f.service
	if len(service) > 0 {
		s = service[0]
	}
	entry, err := s.Log(t.Context(), a, in)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	return entry
}

// withKind returns a second service on the fixture's data that also knows a kind called name under the
// kind "parent": the entries it logs are about a kind the fixture's service does not know.
func (f *fixture) withKind(t *testing.T, name string) *Service {
	t.Helper()
	other := NewService(f.db, NewJSONRepository(), f.service.ids, f.clock.Now)
	for _, k := range append(f.standardKinds(), f.testKind(name, name+"s", "parent", "others")) {
		if err := other.Register(k); err != nil {
			t.Fatalf("Register(%s): %v", k.Name, err)
		}
	}
	return other
}

// appendRaw appends e as a in one Write, as Purge and EmptyBin do, and stops the test when that fails.
func (f *fixture) appendRaw(t *testing.T, a actor.Actor, e Entry) Entry {
	t.Helper()
	var entry Entry
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		entry, err = f.service.appendEntry(ctx, a, e)
		return err
	})
	if err != nil {
		t.Fatalf("appendEntry: %v", err)
	}
	return entry
}

// removeForGood removes a record from its table without the recycle bin, as a permanent delete leaves it:
// not stored and in no entry of the bin.
func (f *fixture) removeForGood(t *testing.T, table, id string) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error { return store.Delete(ctx, table, id) })
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// lastEntry returns the newest entry of the log.
func (f *fixture) lastEntry(t *testing.T) Entry {
	t.Helper()
	entries := f.entries(t)
	return entries[len(entries)-1]
}

// lastEntryAbout returns the newest entry about the record id.
func (f *fixture) lastEntryAbout(t *testing.T, id string) Entry {
	t.Helper()
	entries := f.entries(t)
	for _, e := range slices.Backward(entries) {
		if e.RecordID == id {
			return e
		}
	}
	t.Fatalf("no entry about %s", id)
	return Entry{}
}

// query runs QueryLog as Priya and stops the test when it fails.
func (f *fixture) query(t *testing.T, q LogQuery) LogPage {
	t.Helper()
	page, err := f.service.QueryLog(t.Context(), priya, q)
	if err != nil {
		t.Fatalf("QueryLog(%+v): %v", q, err)
	}
	return page
}

// summary runs LogSummary as Priya and stops the test when it fails.
func (f *fixture) summary(t *testing.T, todayStart, weekStart time.Time) LogSummary {
	t.Helper()
	summary, err := f.service.LogSummary(t.Context(), priya, todayStart, weekStart)
	if err != nil {
		t.Fatalf("LogSummary: %v", err)
	}
	return summary
}

// pageIDs returns the ids of the entries of page, in order.
func pageIDs(page LogPage) []string {
	list := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		list = append(list, item.ID)
	}
	return list
}

// assertPage checks that the page holds exactly the entries want, in order, and that total entries match.
func assertPage(t *testing.T, page LogPage, want []Entry, total int) {
	t.Helper()
	got := make([]Entry, 0, len(page.Items))
	for _, item := range page.Items {
		got = append(got, item.Entry)
	}
	assertEntries(t, got, want)
	if page.Total != total {
		t.Errorf("total = %d, want %d", page.Total, total)
	}
}
