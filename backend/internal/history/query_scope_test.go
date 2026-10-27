package history

import (
	"reflect"
	"slices"
	"testing"
)

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
