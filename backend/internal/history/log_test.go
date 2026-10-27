package history

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/errs"
)

// An entry keeps everything feature 011 shows, fixed at the time of the change (FR-022, data model §5):
// who and when, the record's type, ID and name, its path top level first, and every changed field with a
// readable label and readable values, from the kind's Describe or else the defaults.
func TestLog_StoresPathAndTexts_FR022(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	kinds := f.standardKinds()
	kinds[2].Describe = describeOwner // grandchildren name the person in their field "owner"
	f.register(t, kinds...)
	f.addTree(t)
	before := map[string]any{"owner": "U-1002", "plannedCost": json.Number("5"), "ongoing": false,
		"tags": []string{"Digital", "Print"}, "notes": ""}
	after := map[string]any{"owner": "U-1003", "plannedCost": json.Number("7.50"), "ongoing": true,
		"tags": []string{}, "notes": "Call the agency", "startDate": "2026-03-12"}
	changes, err := Diff(before, after)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	got, err := f.service.Log(t.Context(), priya, LogInput{
		Kind: "grandchild", RecordID: "G-1", RecordName: "Grandchild one",
		Action: ActionChanged, Changes: changes, Comment: "The agency asked for it",
	})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	want := Entry{
		ID: "LOG-1001", Seq: 1001, At: start, Actor: priya, Action: ActionChanged, Verb: "changed",
		RecordType: "grandchild", RecordTypeLabel: "grandchild", RecordID: "G-1", RecordName: "Grandchild one",
		Path: []RecordRef{{Type: "parent", ID: "P-1", Name: "Parent one"}, {Type: "child", ID: "C-1", Name: "Child one"}},
		Changes: []Change{
			{Field: "notes", Label: "Notes", Old: raw(`""`), New: raw(`"Call the agency"`), OldText: "Not filled in", NewText: "Call the agency"},
			{Field: "ongoing", Label: "Ongoing", Old: raw(`false`), New: raw(`true`), OldText: "No", NewText: "Yes"},
			{Field: "owner", Label: "Owner", Old: raw(`"U-1002"`), New: raw(`"U-1003"`), OldText: "Priya Raman", NewText: "Daniel Okoro"},
			{Field: "plannedCost", Label: "Planned cost", Old: raw(`5`), New: raw(`7.50`), OldText: "5", NewText: "7.50"},
			{Field: "startDate", Label: "Start date", Old: raw(`null`), New: raw(`"2026-03-12"`), OldText: "Not filled in", NewText: "2026-03-12"},
			{Field: "tags", Label: "Tags", Old: raw(`["Digital","Print"]`), New: raw(`[]`), OldText: "Digital, Print", NewText: "Not filled in"},
		},
		Comment: "The agency asked for it",
	}
	assertEntries(t, []Entry{got}, []Entry{want})
	assertEntries(t, f.entries(t), []Entry{want}) // stored exactly as returned
}

// For a record that is no longer stored, the caller names its parent and the path starts there. Without
// it, history cannot know where the record was: a mistake in the calling code (FR-022).
func TestLog_RecordThatIsGone_PathStartsAtItsParent_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)

	got, err := f.service.Log(t.Context(), priya, LogInput{
		Kind: "grandchild", RecordID: "G-9", RecordName: "Gone", ParentID: "C-2", Action: ActionDeleted,
	})

	wantPath := []RecordRef{{Type: "parent", ID: "P-1", Name: "Parent one"}, {Type: "child", ID: "C-2", Name: "Child two"}}
	if err != nil || !reflect.DeepEqual(got.Path, wantPath) {
		t.Errorf("Log = path %+v, %v; want %+v", got.Path, err, wantPath)
	}
	if got.Changes == nil || len(got.Changes) != 0 {
		t.Errorf("changes = %#v, want an empty list", got.Changes)
	}

	_, err = f.service.Log(t.Context(), priya, LogInput{
		Kind: "grandchild", RecordID: "G-9", RecordName: "Gone", Action: ActionDeleted,
	})
	if err == nil {
		t.Error("Log without ParentID for a record that is gone = nil, want an error")
	}
}

// Every kind of change can be logged (FR-022). The verb is the action's own; a status change names its
// own verb, such as "published" (011 FR-006). Mistakes in the calling code are refused and write nothing.
func TestLog_EveryAction_FR022(t *testing.T) { //nolint:gocyclo,tparallel // 10: each action, then each mistake; the count after the subtests needs them done
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	actions := []struct {
		action Action
		verb   string // the verb sent
		want   string // the verb stored
	}{
		{ActionCreated, "", "created"},
		{ActionChanged, "", "changed"},
		{ActionStatusChanged, "published", "published"},
		{ActionMoved, "", "moved"},
		{ActionCopied, "", "copied"},
		{ActionDeleted, "", "deleted"},
		{ActionRestored, "", "restored"},
	}
	for _, tt := range actions {
		t.Run(string(tt.action), func(t *testing.T) {
			got, err := f.service.Log(t.Context(), priya, parentOne(tt.action, tt.verb))

			if err != nil || got.Action != tt.action || got.Verb != tt.want {
				t.Errorf("Log = action %q, verb %q, %v; want %q, %q", got.Action, got.Verb, err, tt.action, tt.want)
			}
			if got.Path == nil || len(got.Path) != 0 {
				t.Errorf("path of a top-level record = %#v, want an empty list", got.Path)
			}
		})
	}

	mistakes := map[string]LogInput{
		"a status change without a verb": parentOne(ActionStatusChanged, ""),
		"an unknown action":              parentOne("renamed", ""),
		"no action":                      parentOne("", ""),
		"a kind that is not registered":  {Kind: "plan", RecordID: "P-1", RecordName: "Parent one", Action: ActionCreated},
	}
	for name, in := range mistakes {
		if _, err := f.service.Log(t.Context(), priya, in); err == nil {
			t.Errorf("Log with %s = nil, want an error", name)
		}
	}
	if got := len(f.entries(t)); got != len(actions) {
		t.Errorf("%d entries, want %d: the refused ones must write nothing", got, len(actions))
	}
}

// Two changes of the same record in the same millisecond are both kept, in the order they were made: their
// sequence numbers order them, because their times are equal (spec edge case).
func TestLog_SameMillisecondKeepsOrder_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)

	first := f.log(t, parentOne(ActionChanged, ""))
	second := f.log(t, parentOne(ActionStatusChanged, "published"))

	if !first.At.Equal(second.At) || second.Seq <= first.Seq {
		t.Fatalf("first at %v (seq %d), second at %v (seq %d); want the same time and a higher seq",
			first.At, first.Seq, second.At, second.Seq)
	}
	entries, total, err := f.service.ListLog(t.Context(), priya, LogFilter{RecordID: "P-1", Limit: 50})
	if err != nil || total != 2 || !slices.Equal(entryIDs(entries), []string{second.ID, first.ID}) {
		t.Errorf("ListLog = %v (total %d), %v; want %s then %s", entryIDs(entries), total, err, second.ID, first.ID)
	}
}

// Log joins the caller's Write: entries written in one transaction are saved together with the change,
// each with its own id, and a Write that fails saves none of them (FR-022, research R3).
func TestLog_TwoEntriesInOneWrite_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)

	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		if _, err := f.service.Log(ctx, priya, parentOne(ActionChanged, "")); err != nil {
			return err
		}
		_, err := f.service.Log(ctx, priya, parentOne(ActionStatusChanged, "published"))
		return err
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := entryIDs(f.entries(t)); !slices.Equal(got, []string{"LOG-1001", "LOG-1002"}) {
		t.Errorf("entries = %v, want LOG-1001 and LOG-1002", got)
	}

	failed := errors.New("the change failed")
	err = f.db.Write(t.Context(), func(ctx context.Context) error {
		if _, logErr := f.service.Log(ctx, priya, parentOne(ActionChanged, "")); logErr != nil {
			return logErr
		}
		return failed
	})

	if !errors.Is(err, failed) {
		t.Errorf("Write = %v, want the change's error", err)
	}
	if got := len(f.entries(t)); got != 2 {
		t.Errorf("%d entries after a failed Write, want still 2", got)
	}
}

// Activity-log entries are only ever added: deletes, Undo, Restore and a restart of the API leave every
// earlier entry exactly as it was, in its place (FR-023).
func TestLog_EntriesNeverChangeOrDisappear_FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	renamed, err := Diff(map[string]string{"name": "Parent"}, map[string]string{"name": "Parent one"})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	f.log(t, parentOne(ActionCreated, ""))
	f.log(t, LogInput{Kind: "parent", RecordID: "P-1", RecordName: "Parent one", Action: ActionChanged, Changes: renamed})
	before := f.entries(t)

	undone := f.moveToBin(t, "parent", "P-1")
	f.undo(t, priya, undone.ID)
	restored := f.moveToBin(t, "parent", "P-1")
	f.clock.Advance(time.Hour)
	f.restore(t, priya, restored.ID)
	f.reopen(t)

	after := f.entries(t)
	if len(after) != len(before)+4 {
		t.Fatalf("%d entries, want the %d earlier ones and 4 new ones", len(after), len(before))
	}
	assertEntries(t, after[:len(before)], before)
	var actions []Action
	for _, e := range after[len(before):] {
		actions = append(actions, e.Action)
	}
	if want := []Action{ActionDeleted, ActionRestored, ActionDeleted, ActionRestored}; !slices.Equal(actions, want) {
		t.Errorf("new entries = %v, want %v", actions, want)
	}
}

// The log answers the newest matching entries first, at most as many as asked for, with the number of all
// matching entries; it filters by record type and record (011 FR-009, FR-011). The limit is 1 to 200.
func TestListLog_FilterLimitTotal(t *testing.T) { //nolint:gocyclo // 11: each filter, then each limit below, above and at the bounds
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	none, total, err := f.service.ListLog(t.Context(), priya, LogFilter{Limit: 50})
	if err != nil || none == nil || len(none) != 0 || total != 0 {
		t.Errorf("empty log: ListLog = %v (total %d), %v; want an empty list", none, total, err)
	}
	p1Created := f.log(t, parentOne(ActionCreated, ""))
	c1Created := f.log(t, LogInput{Kind: "child", RecordID: "C-1", RecordName: "Child one", Action: ActionCreated})
	p1Changed := f.log(t, parentOne(ActionChanged, ""))
	c2Created := f.log(t, LogInput{Kind: "child", RecordID: "C-2", RecordName: "Child two", Action: ActionCreated})
	p1Published := f.log(t, parentOne(ActionStatusChanged, "published"))
	tests := []struct {
		name   string
		filter LogFilter
		want   []Entry
		total  int
	}{
		{"everything", LogFilter{Limit: 50}, []Entry{p1Published, c2Created, p1Changed, c1Created, p1Created}, 5},
		{"the newest two", LogFilter{Limit: 2}, []Entry{p1Published, c2Created}, 5},
		{"one record type", LogFilter{RecordType: "child", Limit: 50}, []Entry{c2Created, c1Created}, 2},
		{"one record, limited", LogFilter{RecordID: "P-1", Limit: 2}, []Entry{p1Published, p1Changed}, 3},
		{"record type and record", LogFilter{RecordType: "child", RecordID: "C-1", Limit: 50}, []Entry{c1Created}, 1},
		{"nothing matches", LogFilter{RecordID: "C-9", Limit: 50}, []Entry{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, total, err := f.service.ListLog(t.Context(), priya, tt.filter)

			if err != nil || total != tt.total {
				t.Errorf("ListLog = total %d, %v; want total %d", total, err, tt.total)
			}
			assertEntries(t, got, tt.want)
		})
	}

	for _, limit := range []int{0, -1, 201} {
		_, _, err := f.service.ListLog(t.Context(), priya, LogFilter{Limit: limit})

		assertError(t, err, errs.Validation(errs.FieldError{Field: "limit", Message: "Choose a number from 1 to 200."}))
	}
	for _, limit := range []int{1, 200} {
		if _, _, err := f.service.ListLog(t.Context(), priya, LogFilter{Limit: limit}); err != nil {
			t.Errorf("ListLog with limit %d: %v, want it accepted", limit, err)
		}
	}
}
