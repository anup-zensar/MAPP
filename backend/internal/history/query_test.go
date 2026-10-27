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
