package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
)

// The people of these tests.
var (
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	mei    = actor.Actor{ID: "U-1004", Name: "Mei Lin"}
)

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// A kind that history could not work with stops the API from starting (research R8): every mistake is
// refused with an error, and a refused kind is not registered.
func TestRegister_RefusesBadKinds(t *testing.T) {
	f := newBareFixture(t)
	kinds := f.standardKinds()
	parent, child := kinds[0], kinds[1]
	if err := f.service.Register(parent); err != nil {
		t.Fatalf("Register(parent): %v", err)
	}
	mistakes := []struct {
		name   string
		change func(k *Kind) // applied to a copy of the child kind
	}{
		{"no name", func(k *Kind) { k.Name = "" }},
		{"a name that is already registered", func(k *Kind) { k.Name, k.Parent = "parent", "" }},
		{"itself as its parent", func(k *Kind) { k.Parent = k.Name }},
		{"a parent that is not registered yet", func(k *Kind) { k.Parent = "grandparent" }},
		{"no Snapshot", func(k *Kind) { k.Snapshot = nil }},
		{"no Remove", func(k *Kind) { k.Remove = nil }},
		{"no Put", func(k *Kind) { k.Put = nil }},
		{"a parent but no ChildIDs", func(k *Kind) { k.ChildIDs = nil }},
	}
	for _, mistake := range mistakes {
		t.Run(mistake.name, func(t *testing.T) {
			bad := child
			mistake.change(&bad)

			if err := f.service.Register(bad); err == nil {
				t.Errorf("Register accepted a kind with %s", mistake.name)
			}
		})
	}

	if _, found := f.service.findKind("child"); found {
		t.Error("a refused kind was registered")
	}
	if err := f.service.Register(child); err != nil {
		t.Errorf("Register(child) = %v, want nil", err)
	}
}

// An entry keeps everything feature 011 shows, fixed at the time of the change (FR-022, data model §5):
// who and when, the record's type, ID and name, its path top level first, and every changed field with a
// readable label and readable values, from the kind's Describe or else the defaults.
func TestLog_StoresPathAndTexts_FR022(t *testing.T) {
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
func TestLog_EveryAction_FR022(t *testing.T) {
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
		if _, err := f.service.Log(ctx, priya, parentOne(ActionChanged, "")); err != nil {
			return err
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

// A delete moves the record and everything under it into one recycle-bin entry with who deleted it and
// when, removes them (children first) and writes one log entry (FR-024). The entry counts what was under
// the record by kind, in the order the kinds were registered, and lists only kinds it holds.
func TestMoveToBin_WithChildren_FR024(t *testing.T) {
	f := newBareFixture(t)
	attachments := f.testKind("attachment", "attachments", "parent", "attachments") // none under P-1
	f.register(t, append(f.standardKinds(), attachments)...)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	f.put(t, childrenTable, newRecord("C-3", "Child of parent two", "P-2"))
	f.clock.Advance(time.Hour)

	got, err := f.service.MoveToBin(t.Context(), priya, "parent", "P-1")
	if err != nil {
		t.Fatalf("MoveToBin: %v", err)
	}

	want := BinEntryView{
		ID: "BIN-1001", RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one",
		Path: []RecordRef{}, Parent: nil, DeletedAt: start.Add(time.Hour), DeletedBy: priya,
		Contents: []ContentCount{
			{RecordType: "child", Label: "children", Count: 2},
			{RecordType: "grandchild", Label: "grandchildren", Count: 3},
		},
	}
	if view := got.View(); !reflect.DeepEqual(view, want) {
		t.Errorf("MoveToBin =\n%s\nwant\n%s", asJSON(view), asJSON(want))
	}
	entry := f.binEntry(t, "BIN-1001")
	if !reflect.DeepEqual(entry, got) {
		t.Errorf("stored entry =\n%s\nwant it exactly as returned:\n%s", asJSON(entry), asJSON(got))
	}

	// The entry keeps every record as it was stored: the deleted one first, parents before children.
	wantSnapshots := []Snapshot{
		snapshotOf("parent", newRecord("P-1", "Parent one", "")),
		snapshotOf("child", newRecord("C-1", "Child one", "P-1")),
		snapshotOf("grandchild", newRecord("G-1", "Grandchild one", "C-1")),
		snapshotOf("grandchild", newRecord("G-2", "Grandchild two", "C-1")),
		snapshotOf("child", newRecord("C-2", "Child two", "P-1")),
		snapshotOf("grandchild", newRecord("G-3", "Grandchild three", "C-2")),
	}
	if !reflect.DeepEqual(entry.Snapshots, wantSnapshots) {
		t.Errorf("snapshots =\n%s\nwant\n%s", asJSON(entry.Snapshots), asJSON(wantSnapshots))
	}
	if entry.Resolution != nil {
		t.Errorf("resolution = %+v, want none while the entry is in the bin", entry.Resolution)
	}

	// The records are removed, children first; the other parent keeps its child.
	if want := []string{"G-3", "C-2", "G-2", "G-1", "C-1", "P-1"}; !slices.Equal(f.removed, want) {
		t.Errorf("removed %v, want %v", f.removed, want)
	}
	wantLeft := map[string][]testRecord{
		parentsTable:       {newRecord("P-2", "Parent two", "")},
		childrenTable:      {newRecord("C-3", "Child of parent two", "P-2")},
		grandchildrenTable: {},
	}
	if left := f.allRecords(t); !reflect.DeepEqual(left, wantLeft) {
		t.Errorf("stored records = %+v, want %+v", left, wantLeft)
	}

	// One log entry for the whole delete.
	assertEntries(t, f.entries(t), []Entry{{
		ID: "LOG-1001", Seq: 1001, At: start.Add(time.Hour), Actor: priya, Action: ActionDeleted, Verb: "deleted",
		RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one",
		Path: []RecordRef{}, Changes: []Change{}, Details: "with 2 children and 3 grandchildren",
	}})
}

// One record of a kind under the deleted one is counted in the singular: "with 1 grandchild". A record with
// a parent keeps where it was: its path and its parent (FR-024).
func TestMoveToBin_OneNoteSaysSingular_FR024(t *testing.T) {
	f := newFixture(t)
	f.addTree(t)

	got := f.moveToBin(t, "child", "C-2") // C-2 holds one grandchild, G-3

	parentOneRef := RecordRef{Type: "parent", ID: "P-1", Name: "Parent one"}
	want := BinEntryView{
		ID: "BIN-1001", RecordType: "child", RecordTypeLabel: "child", RecordID: "C-2", RecordName: "Child two",
		Path: []RecordRef{parentOneRef}, Parent: &parentOneRef, DeletedAt: start, DeletedBy: priya,
		Contents: []ContentCount{{RecordType: "grandchild", Label: "grandchild", Count: 1}},
	}
	if view := got.View(); !reflect.DeepEqual(view, want) {
		t.Errorf("MoveToBin =\n%s\nwant\n%s", asJSON(view), asJSON(want))
	}
	entries := f.entries(t)
	if len(entries) != 1 || entries[0].Details != "with 1 grandchild" || !reflect.DeepEqual(entries[0].Path, []RecordRef{parentOneRef}) {
		t.Errorf("log =\n%s\nwant one entry with the details \"with 1 grandchild\" and the path [P-1]", asJSON(entries))
	}
}

// A delete, its recycle-bin entry and its log entry are saved together with the caller's change, or not at
// all: when the caller's Write fails, every record is still there and nothing was binned or logged (FR-024).
func TestMoveToBin_JoinsTheCallersWrite_FR024(t *testing.T) {
	f := newFixture(t)
	f.addTree(t)
	before := f.allRecords(t)
	failed := errors.New("the caller's change failed")

	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		if _, err := f.service.MoveToBin(ctx, priya, "parent", "P-1"); err != nil {
			return err
		}
		return failed
	})

	if !errors.Is(err, failed) {
		t.Errorf("Write = %v, want the caller's error", err)
	}
	if after := f.allRecords(t); !reflect.DeepEqual(after, before) {
		t.Errorf("records = %+v, want them all still there: %+v", after, before)
	}
	if bin := f.listBin(t); len(bin) != 0 {
		t.Errorf("recycle bin = %s, want it empty", asJSON(bin))
	}
	if entries := f.entries(t); len(entries) != 0 {
		t.Errorf("log = %s, want it empty", asJSON(entries))
	}
}

// Undo puts back exactly what was deleted: the same IDs, data and versions (FR-026, domain model R14). The
// entry leaves the bin with who undid it and when, keeps no snapshots, and one "restored" entry is logged.
func TestUndo_RestoresExactly_FR026(t *testing.T) {
	f := newFixture(t)
	f.addTree(t)
	changed := newRecord("C-1", "Child one", "P-1")
	changed.Touch(daniel, start) // version 2: Undo must bring the version back too
	f.put(t, childrenTable, changed)
	before := f.allRecords(t)
	deleted := f.moveToBin(t, "parent", "P-1")
	f.clock.Advance(time.Minute)

	got, err := f.service.Undo(t.Context(), daniel, deleted.ID)

	want := RestoreResult{RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one"}
	if err != nil || got != want {
		t.Fatalf("Undo = %+v, %v; want %+v", got, err, want)
	}
	if after := f.allRecords(t); !reflect.DeepEqual(after, before) {
		t.Errorf("records after Undo =\n%+v\nwant exactly\n%+v", after, before)
	}
	entry := f.binEntry(t, deleted.ID)
	wantResolution := Resolution{How: "undone", By: daniel, At: start.Add(time.Minute)}
	if entry.Resolution == nil || *entry.Resolution != wantResolution {
		t.Errorf("resolution = %+v, want %+v", entry.Resolution, wantResolution)
	}
	if entry.Snapshots == nil || len(entry.Snapshots) != 0 {
		t.Errorf("snapshots = %d, want an empty list once the entry is resolved", len(entry.Snapshots))
	}
	if bin := f.listBin(t); len(bin) != 0 {
		t.Errorf("recycle bin = %s, want it empty", asJSON(bin))
	}
	assertEntries(t, f.entries(t)[1:], []Entry{{
		ID: "LOG-1002", Seq: 1002, At: start.Add(time.Minute), Actor: daniel, Action: ActionRestored, Verb: "restored",
		RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one",
		Path: []RecordRef{}, Changes: []Change{}, Details: "Undo, with 2 children and 3 grandchildren",
	}})
}

// The server accepts Undo for 2 minutes after the delete (FR-026, D-037): at exactly 2 minutes it works; a
// millisecond later it is refused with undo_expired and nothing changes. Restore has no time limit.
func TestUndo_ExpiresAfterTwoMinutes_FR026(t *testing.T) {
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	onTime := f.moveToBin(t, "parent", "P-1")
	late := f.moveToBin(t, "parent", "P-2")
	f.clock.Advance(2 * time.Minute)

	if _, err := f.service.Undo(t.Context(), priya, onTime.ID); err != nil {
		t.Errorf("Undo exactly 2 minutes after the delete: %v, want it to work", err)
	}
	f.clock.Advance(time.Millisecond)
	entriesBefore := len(f.entries(t))

	_, err := f.service.Undo(t.Context(), priya, late.ID)

	assertError(t, err, errs.UndoExpired())
	if _, found := f.stored(t, parentsTable, "P-2"); found {
		t.Error("P-2 came back although Undo was refused")
	}
	if bin := f.listBin(t); len(bin) != 1 || bin[0].ID != late.ID {
		t.Errorf("recycle bin = %s, want only %s", asJSON(bin), late.ID)
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}
	if _, err := f.service.Restore(t.Context(), priya, late.ID); err != nil {
		t.Errorf("Restore after the Undo time: %v, want it to work", err)
	}
}

// A record whose parent is no longer stored cannot come back on its own: Restore and Undo are refused,
// naming the parent to restore first, and nothing changes. Once the parent is back, the record follows
// (FR-025).
func TestRestore_ParentMissing_FR025(t *testing.T) {
	f := newFixture(t)
	f.addTree(t)
	child := f.moveToBin(t, "child", "C-1")
	parent := f.moveToBin(t, "parent", "P-1")
	entriesBefore := len(f.entries(t))
	attempts := []struct {
		name string
		call func(ctx context.Context, a actor.Actor, entryID string) (RestoreResult, error)
	}{
		{"Restore", f.service.Restore},
		{"Undo", f.service.Undo},
	}

	for _, attempt := range attempts {
		t.Run(attempt.name, func(t *testing.T) {
			_, err := attempt.call(t.Context(), priya, child.ID)

			assertError(t, err, errs.RestoreParentFirst(RecordRef{Type: "parent", ID: "P-1", Name: "Parent one"}))
		})
	}
	if _, found := f.stored(t, childrenTable, "C-1"); found {
		t.Error("C-1 came back without its parent")
	}
	if entry := f.binEntry(t, child.ID); entry.Resolution != nil || len(entry.Snapshots) != 3 {
		t.Errorf("entry = %s, want it unchanged in the bin with its 3 records", asJSON(entry))
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}

	f.restore(t, priya, parent.ID)
	f.restore(t, priya, child.ID)
	if _, found := f.stored(t, childrenTable, "C-1"); !found {
		t.Error("C-1 did not come back after its parent")
	}
}

// An entry that has already left the bin cannot be undone or restored again: the refusal says how, by whom
// and when, so the person understands what happened (FR-025, 011 FR-032). Nothing changes.
func TestUndo_AlreadyResolved_FR025(t *testing.T) {
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	undone := f.moveToBin(t, "parent", "P-1")
	restored := f.moveToBin(t, "parent", "P-2")
	f.clock.Advance(time.Minute)
	f.undo(t, daniel, undone.ID)
	f.restore(t, mei, restored.ID)
	entriesBefore := len(f.entries(t))
	tests := []struct {
		name    string
		entryID string
		want    *errs.Error
	}{
		{"undone", undone.ID, errs.AlreadyResolved("Parent one", Resolution{How: "undone", By: daniel, At: start.Add(time.Minute)})},
		{"restored", restored.ID, errs.AlreadyResolved("Parent two", Resolution{How: "restored", By: mei, At: start.Add(time.Minute)})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, undoErr := f.service.Undo(t.Context(), priya, tt.entryID)
			_, restoreErr := f.service.Restore(t.Context(), priya, tt.entryID)

			assertError(t, undoErr, tt.want)
			assertError(t, restoreErr, tt.want)
		})
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}
}

// An entry can come back only where all its kinds are registered: in an environment without the developer
// sample records, their entries answer not_found. So does an id the bin never had (FR-025).
func TestUndo_KindNotRegistered_FR025(t *testing.T) {
	f := newFixture(t)
	f.addTree(t)
	deleted := f.moveToBin(t, "parent", "P-1")
	noKinds := NewService(f.db, NewJSONRepository(), ids.NewGenerator(ids.NewJSONCounters()), f.clock.Now)
	parentsOnly := NewService(f.db, NewJSONRepository(), ids.NewGenerator(ids.NewJSONCounters()), f.clock.Now)
	if err := parentsOnly.Register(f.standardKinds()[0]); err != nil {
		t.Fatalf("Register: %v", err)
	}
	services := []struct {
		name    string
		service *Service
	}{
		{"no kinds", noKinds},
		{"the parent kind but not its children's", parentsOnly},
	}

	for _, s := range services {
		t.Run(s.name, func(t *testing.T) {
			_, undoErr := s.service.Undo(t.Context(), priya, deleted.ID)
			_, restoreErr := s.service.Restore(t.Context(), priya, deleted.ID)

			assertError(t, undoErr, errs.NotFound("This item can't be restored here."))
			assertError(t, restoreErr, errs.NotFound("This item can't be restored here."))
		})
	}

	_, undoErr := f.service.Undo(t.Context(), priya, "BIN-9999")
	_, restoreErr := f.service.Restore(t.Context(), priya, "BIN-9999")
	assertError(t, undoErr, errs.NotFound("This item is not in the recycle bin."))
	assertError(t, restoreErr, errs.NotFound("This item is not in the recycle bin."))
	if bin := f.listBin(t); len(bin) != 1 {
		t.Errorf("recycle bin = %s, want the entry still in it", asJSON(bin))
	}
}

// Restore applies each kind's restore rules through PrepareRestore, which learns who restores, when and
// whether it is an Undo. The changes it makes are saved and listed in the "restored" log entry, and its
// notes end the entry's details and are the answer's details (FR-025, research R8).
func TestRestore_AppliesPrepareRestore_FR025(t *testing.T) {
	f := newBareFixture(t)
	kinds := f.standardKinds()
	var seen []RestoreContext
	kinds[0].PrepareRestore = func(_ context.Context, rc RestoreContext, s *Snapshot) ([]string, error) {
		seen = append(seen, rc)
		if rc.Undo {
			return nil, nil // an Undo puts the record back exactly
		}
		var r testRecord
		if err := json.Unmarshal(s.Data, &r); err != nil {
			return nil, err
		}
		r.Name += " (restored)"
		r.Touch(rc.Actor, rc.Now)
		data, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		s.Name, s.Data = r.Name, data
		return []string{"Renamed to '" + r.Name + "' because the name was taken."}, nil
	}
	f.register(t, kinds...)
	f.addTree(t)
	deleted := f.moveToBin(t, "parent", "P-1")
	f.clock.Advance(time.Hour)
	restoredAt := start.Add(time.Hour)

	got, err := f.service.Restore(t.Context(), daniel, deleted.ID)

	note := "Renamed to 'Parent one (restored)' because the name was taken."
	want := RestoreResult{RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one (restored)", Details: note}
	if err != nil || got != want {
		t.Fatalf("Restore = %+v, %v; want %+v", got, err, want)
	}
	if want := []RestoreContext{{Actor: daniel, Now: restoredAt, Undo: false}}; !slices.Equal(seen, want) {
		t.Errorf("PrepareRestore saw %+v, want %+v", seen, want)
	}
	wantParent := newRecord("P-1", "Parent one (restored)", "")
	wantParent.Touch(daniel, restoredAt)
	if stored, _ := f.stored(t, parentsTable, "P-1"); stored != wantParent {
		t.Errorf("stored parent = %+v, want %+v", stored, wantParent)
	}
	if _, found := f.stored(t, grandchildrenTable, "G-3"); !found {
		t.Error("the grandchild G-3 did not come back with its parent")
	}
	assertEntries(t, f.entries(t)[1:], []Entry{{
		ID: "LOG-1002", Seq: 1002, At: restoredAt, Actor: daniel, Action: ActionRestored, Verb: "restored",
		RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one (restored)",
		Path: []RecordRef{},
		Changes: []Change{{Field: "name", Label: "Name", Old: raw(`"Parent one"`), New: raw(`"Parent one (restored)"`),
			OldText: "Parent one", NewText: "Parent one (restored)"}},
		Details: "Restored from the recycle bin, with 2 children and 3 grandchildren. " + note,
	}})

	undone := f.moveToBin(t, "parent", "P-1")
	f.undo(t, daniel, undone.ID)
	if last := seen[len(seen)-1]; !last.Undo {
		t.Errorf("PrepareRestore during Undo saw %+v, want Undo true", last)
	}
}

// When a kind's PrepareRestore refuses, nothing comes back, even the records already put back in the same
// restore, and the entry stays in the bin (spec edge case: an Undo of a name that is now taken).
func TestUndo_RefusedByPrepareRestoreChangesNothing_FR026(t *testing.T) {
	f := newBareFixture(t)
	kinds := f.standardKinds()
	refusal := errs.NameTaken("Can't undo — there is now another grandchild called 'Grandchild one'. Restore it from the recycle bin instead.")
	kinds[2].PrepareRestore = func(_ context.Context, rc RestoreContext, _ *Snapshot) ([]string, error) {
		if rc.Undo {
			return nil, refusal // grandchildren come back after their parent and child
		}
		return nil, nil
	}
	f.register(t, kinds...)
	f.addTree(t)
	deleted := f.moveToBin(t, "parent", "P-1")
	entriesBefore := len(f.entries(t))

	_, err := f.service.Undo(t.Context(), priya, deleted.ID)

	assertError(t, err, refusal)
	for table, records := range f.allRecords(t) {
		if len(records) != 0 {
			t.Errorf("%s = %+v, want nothing back", table, records)
		}
	}
	if bin := f.listBin(t); len(bin) != 1 {
		t.Errorf("recycle bin = %s, want the entry still in it", asJSON(bin))
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}
}

// The log answers the newest matching entries first, at most as many as asked for, with the number of all
// matching entries; it filters by record type and record (011 FR-009, FR-011). The limit is 1 to 200.
func TestListLog_FilterLimitTotal(t *testing.T) {
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

// The recycle bin lists what is still in it, newest first; entries that were undone or restored are left
// out (data model §6).
func TestListBin_UnresolvedNewestFirst(t *testing.T) {
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""), newRecord("P-3", "Parent three", ""))
	if bin := f.listBin(t); bin == nil || len(bin) != 0 {
		t.Errorf("empty bin: ListBin = %#v, want an empty list", bin)
	}
	first := f.moveToBin(t, "parent", "P-1")
	f.clock.Advance(time.Minute)
	undone := f.moveToBin(t, "parent", "P-2")
	f.clock.Advance(time.Minute)
	last := f.moveToBin(t, "parent", "P-3")
	f.undo(t, priya, undone.ID)

	got := f.listBin(t)

	if want := []BinEntry{last, first}; !reflect.DeepEqual(got, want) {
		t.Errorf("ListBin =\n%s\nwant\n%s", asJSON(got), asJSON(want))
	}
}

// SnapshotsInBin gives the records of one kind that are in the recycle bin — deleted records and the
// records under them — oldest entry first, leaving out entries that were undone or restored, so that
// feature 002 can count them as still in use (FR-003, data model §10). It joins the caller's transaction.
func TestSnapshotsInBin_KindAndContents_FR003(t *testing.T) {
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	f.put(t, childrenTable, newRecord("C-3", "Child of parent two", "P-2"))
	f.moveToBin(t, "child", "C-2") // with G-3
	f.clock.Advance(time.Minute)
	f.moveToBin(t, "parent", "P-2") // with C-3
	f.clock.Advance(time.Minute)
	undone := f.moveToBin(t, "child", "C-1") // with G-1 and G-2, then undone
	f.undo(t, priya, undone.ID)

	inBin := func(kind string) []Snapshot {
		t.Helper()
		snapshots, err := f.service.SnapshotsInBin(t.Context(), kind)
		if err != nil {
			t.Fatalf("SnapshotsInBin(%s): %v", kind, err)
		}
		return snapshots
	}

	want := map[string][]Snapshot{
		"parent":     {snapshotOf("parent", newRecord("P-2", "Parent two", ""))},
		"child":      {snapshotOf("child", newRecord("C-2", "Child two", "P-1")), snapshotOf("child", newRecord("C-3", "Child of parent two", "P-2"))},
		"grandchild": {snapshotOf("grandchild", newRecord("G-3", "Grandchild three", "C-2"))},
		"unknown":    {},
	}
	for kind, wantSnapshots := range want {
		if got := inBin(kind); !reflect.DeepEqual(got, wantSnapshots) {
			t.Errorf("SnapshotsInBin(%s) =\n%s\nwant\n%s", kind, asJSON(got), asJSON(wantSnapshots))
		}
	}

	// Inside a Write, a delete made earlier in the same Write is already in the bin.
	var joined []Snapshot
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		if _, err := f.service.MoveToBin(ctx, priya, "parent", "P-1"); err != nil {
			return err
		}
		var err error
		joined, err = f.service.SnapshotsInBin(ctx, "parent")
		return err
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	ids := []string{}
	for _, s := range joined {
		ids = append(ids, s.ID)
	}
	if !slices.Equal(ids, []string{"P-2", "P-1"}) {
		t.Errorf("SnapshotsInBin(parent) inside the Write = %v, want [P-2 P-1]", ids)
	}
}

// Tables of the test kinds.
const (
	parentsTable       = "parents"
	childrenTable      = "children"
	grandchildrenTable = "grandchildren"
)

// testRecord is the record of every test kind: a parent, a child under a parent, or a grandchild under a
// child.
type testRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
	record.Meta
}

// newRecord returns a record that Priya created when the test started.
func newRecord(id, name, parentID string) testRecord {
	return testRecord{ID: id, Name: name, ParentID: parentID, Meta: record.NewMeta(priya, start)}
}

// fixture is one test's history service on a temporary data folder, with a clock that moves only when the
// test says so.
type fixture struct {
	service *Service
	db      *store.DB
	dir     string
	clock   *testClock
	removed []string // the ids the test kinds removed, in order
}

// newBareFixture returns a fixture whose service knows no kinds yet.
func newBareFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{dir: t.TempDir(), clock: &testClock{now: start}}
	f.open(t)
	return f
}

// newFixture returns a fixture whose service knows the kinds parent, child and grandchild.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := newBareFixture(t)
	f.register(t, f.standardKinds()...)
	return f
}

// open opens the data folder and builds a new service on it, as the API does when it starts.
func (f *fixture) open(t *testing.T) {
	t.Helper()
	db, err := store.Open(f.dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	f.db = db
	f.service = NewService(db, NewJSONRepository(), ids.NewGenerator(ids.NewJSONCounters()), f.clock.Now)
}

// reopen starts again from the data folder with the standard kinds, as after a restart of the API.
func (f *fixture) reopen(t *testing.T) {
	t.Helper()
	f.open(t)
	f.register(t, f.standardKinds()...)
}

// standardKinds returns the kinds parent, child and grandchild, parents first, for a test to adjust and
// register.
func (f *fixture) standardKinds() []Kind {
	return []Kind{
		f.testKind("parent", "parents", "", parentsTable),
		f.testKind("child", "children", "parent", childrenTable),
		f.testKind("grandchild", "grandchildren", "child", grandchildrenTable),
	}
}

// testKind returns a kind whose records are testRecords kept in table; its singular is its name. Remove
// notes every id in f.removed, so tests can check the order.
func (f *fixture) testKind(name, plural, parent, table string) Kind {
	k := Kind{
		Name: name, Singular: name, Plural: plural, Parent: parent,
		Snapshot: func(ctx context.Context, id string) (Snapshot, bool, error) {
			r, found, err := store.Get[testRecord](ctx, table, id)
			if err != nil || !found {
				return Snapshot{}, false, err
			}
			data, err := json.Marshal(r)
			return Snapshot{ID: r.ID, Name: r.Name, ParentID: r.ParentID, Data: data}, true, err
		},
		Remove: func(ctx context.Context, id string) error {
			f.removed = append(f.removed, id)
			return store.Delete(ctx, table, id)
		},
		Put: func(ctx context.Context, s Snapshot) error {
			var r testRecord
			if err := json.Unmarshal(s.Data, &r); err != nil {
				return err
			}
			return store.Put(ctx, table, r.ID, r)
		},
	}
	if parent != "" {
		k.ChildIDs = func(ctx context.Context, parentID string) ([]string, error) {
			records, err := store.List[testRecord](ctx, table)
			if err != nil {
				return nil, err
			}
			var childIDs []string
			for _, r := range slices.Backward(records) { // backwards, so the test sees history sort them
				if r.ParentID == parentID {
					childIDs = append(childIDs, r.ID)
				}
			}
			return childIDs, nil
		}
	}
	return k
}

// describeOwner is a Describe hook that names the person in the field "owner", as a kind holding people's
// ids would. It leaves every other field to the defaults.
func describeOwner(_ context.Context, field string, value any) (string, string, error) {
	if field != "owner" {
		return "", "", nil
	}
	names := map[string]string{"U-1002": "Priya Raman", "U-1003": "Daniel Okoro"}
	id, _ := value.(string)
	return "Owner", names[id], nil
}

// register registers kinds in order and stops the test when one is refused.
func (f *fixture) register(t *testing.T, kinds ...Kind) {
	t.Helper()
	for _, k := range kinds {
		if err := f.service.Register(k); err != nil {
			t.Fatalf("Register(%s): %v", k.Name, err)
		}
	}
}

// addTree saves the records most tests use: the parent P-1 "Parent one" with the children C-1 and C-2;
// the grandchildren G-1 and G-2 under C-1, and G-3 under C-2.
func (f *fixture) addTree(t *testing.T) {
	t.Helper()
	f.put(t, parentsTable, newRecord("P-1", "Parent one", ""))
	f.put(t, childrenTable, newRecord("C-1", "Child one", "P-1"), newRecord("C-2", "Child two", "P-1"))
	f.put(t, grandchildrenTable, newRecord("G-1", "Grandchild one", "C-1"),
		newRecord("G-2", "Grandchild two", "C-1"), newRecord("G-3", "Grandchild three", "C-2"))
}

// put saves records in table, in one Write.
func (f *fixture) put(t *testing.T, table string, records ...testRecord) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		for _, r := range records {
			if err := store.Put(ctx, table, r.ID, r); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

// stored returns the record id of table, and false when it is not stored.
func (f *fixture) stored(t *testing.T, table, id string) (testRecord, bool) {
	t.Helper()
	var r testRecord
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		r, found, err = store.Get[testRecord](ctx, table, id)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return r, found
}

// allRecords returns every stored record of the test kinds, by table.
func (f *fixture) allRecords(t *testing.T) map[string][]testRecord {
	t.Helper()
	all := map[string][]testRecord{}
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		for _, table := range []string{parentsTable, childrenTable, grandchildrenTable} {
			records, err := store.List[testRecord](ctx, table)
			if err != nil {
				return err
			}
			all[table] = records
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return all
}

// entries returns the whole activity log, oldest first.
func (f *fixture) entries(t *testing.T) []Entry {
	t.Helper()
	var entries []Entry
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entries, err = NewJSONRepository().Entries(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return entries
}

// binEntry returns the stored recycle-bin entry id, with its snapshots and resolution.
func (f *fixture) binEntry(t *testing.T, id string) BinEntry {
	t.Helper()
	var entry BinEntry
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entry, found, err = NewJSONRepository().GetBinEntry(ctx, id)
		return err
	})
	if err != nil || !found {
		t.Fatalf("GetBinEntry(%s) = found %v, %v", id, found, err)
	}
	return entry
}

// log writes an entry as Priya and stops the test when that fails.
func (f *fixture) log(t *testing.T, in LogInput) Entry {
	t.Helper()
	entry, err := f.service.Log(t.Context(), priya, in)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	return entry
}

// moveToBin deletes the record id of kind as Priya and stops the test when that fails.
func (f *fixture) moveToBin(t *testing.T, kind, id string) BinEntry {
	t.Helper()
	entry, err := f.service.MoveToBin(t.Context(), priya, kind, id)
	if err != nil {
		t.Fatalf("MoveToBin(%s %s): %v", kind, id, err)
	}
	return entry
}

// undo undoes the delete of entryID as a and stops the test when that fails.
func (f *fixture) undo(t *testing.T, a actor.Actor, entryID string) {
	t.Helper()
	if _, err := f.service.Undo(t.Context(), a, entryID); err != nil {
		t.Fatalf("Undo(%s): %v", entryID, err)
	}
}

// restore restores entryID as a and stops the test when that fails.
func (f *fixture) restore(t *testing.T, a actor.Actor, entryID string) {
	t.Helper()
	if _, err := f.service.Restore(t.Context(), a, entryID); err != nil {
		t.Fatalf("Restore(%s): %v", entryID, err)
	}
}

// listBin returns what is in the recycle bin and stops the test when that fails.
func (f *fixture) listBin(t *testing.T) []BinEntry {
	t.Helper()
	bin, err := f.service.ListBin(t.Context(), priya)
	if err != nil {
		t.Fatalf("ListBin: %v", err)
	}
	return bin
}

// parentOne is the log input of a change of the parent P-1.
func parentOne(action Action, verb string) LogInput {
	return LogInput{Kind: "parent", RecordID: "P-1", RecordName: "Parent one", Action: action, Verb: verb}
}

// snapshotOf is the snapshot history keeps of r, a record of kind.
func snapshotOf(kind string, r testRecord) Snapshot {
	data, _ := json.Marshal(r) // cannot fail: a testRecord holds only strings, numbers and times
	return Snapshot{Kind: kind, ID: r.ID, Name: r.Name, ParentID: r.ParentID, Data: data}
}

// raw is a stored JSON value, written as in the data file.
func raw(value string) json.RawMessage {
	return json.RawMessage(value)
}

func entryIDs(entries []Entry) []string {
	list := make([]string, 0, len(entries))
	for _, e := range entries {
		list = append(list, e.ID)
	}
	return list
}

// assertEntries checks that got are exactly want, and shows both as JSON when they differ.
func assertEntries(t *testing.T, got, want []Entry) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("entries =\n%s\nwant\n%s", asJSON(got), asJSON(want))
	}
}

// assertError checks that err is exactly want: the same status, code, message and details.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %v, want %s", err, asJSON(want))
	}
}

// asJSON writes v as indented JSON, for failure messages.
func asJSON(v any) string {
	text, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(text)
}

// testClock is a clock that shows a fixed time until Advance moves it.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }
