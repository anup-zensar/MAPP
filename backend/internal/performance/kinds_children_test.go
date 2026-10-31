package performance_test

import (
	"testing"

	"mapp/internal/history"
	"mapp/internal/performance"
)

// One child kind per table whose foreign key points at activities: 00031's performance_facts, and 00032's
// media_codes, activity_external_name_aliases and activity_integration_ids.
var activityChildren = []string{"performanceFact", "mediaCode", "externalNameAlias", "integrationId"}

// The recycle bin deletes an activity's children before the activity itself, so every performance record
// that references activities must be a child kind, or the activity's DELETE fails on its foreign key.
func TestKinds_EveryActivityReferenceIsAChildKind_FR021(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	kinds := f.service.Kinds()

	for _, want := range activityChildren {
		if kindNamed(t, kinds, want).Parent != "activity" {
			t.Errorf("%s is not a child of activity: binning an activity that has one fails on its "+
				"foreign key", want)
		}
	}
}

func TestKinds_EveryChildCarriesWhatTheBinCalls_FR021(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	kinds := f.service.Kinds()

	for _, name := range activityChildren {
		k := kindNamed(t, kinds, name)
		if k.ChildIDs == nil || k.Snapshot == nil || k.Remove == nil || k.Put == nil {
			t.Errorf("child kind %s is missing one of ChildIDs, Snapshot, Remove or Put", name)
		}
	}
}

func TestKinds_ChildIsCollectedUnderItsActivity_FR021(t *testing.T) {
	t.Parallel()
	_, kind := binFixture(t)

	ids, err := kind.ChildIDs(t.Context(), "A-1001")
	if err != nil {
		t.Fatalf("ChildIDs() error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "MC-1" {
		t.Fatalf("ChildIDs(A-1001) = %v, want [MC-1]", ids)
	}
	snapshot, found, err := kind.Snapshot(t.Context(), "MC-1")
	if err != nil || !found {
		t.Fatalf("Snapshot(MC-1) found=%v err=%v", found, err)
	}
	if snapshot.ParentID != "A-1001" {
		t.Errorf("snapshot.ParentID = %q, want A-1001: the bin restores it under its activity", snapshot.ParentID)
	}
}

func TestKinds_ChildIsRemovedThenRestored_FR021(t *testing.T) {
	t.Parallel()
	f, kind := binFixture(t)
	snapshot, _, err := kind.Snapshot(t.Context(), "MC-1")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if err = kind.Remove(t.Context(), "MC-1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, stillThere, _ := f.store.GetMediaCode(t.Context(), "MC-1"); stillThere {
		t.Fatal("Remove left the media code behind, so the activity's DELETE would still fail")
	}

	if err = kind.Put(t.Context(), snapshot); err != nil {
		t.Fatalf("Put: %v", err)
	}
	back, restored, _ := f.store.GetMediaCode(t.Context(), "MC-1")
	if !restored || back.Code != 9000001 {
		t.Errorf("after restore the media code is %+v, restored=%v; want code 9000001", back, restored)
	}
}

func binFixture(t *testing.T) (*fixture, history.Kind) {
	t.Helper()
	f := newFixture(t)
	err := f.store.PutMediaCode(t.Context(), performance.MediaCode{ID: "MC-1", Code: 9000001,
		ActivityID: "A-1001", Attributes: map[string]string{}, Status: performance.StatusActive})
	if err != nil {
		t.Fatalf("PutMediaCode: %v", err)
	}
	return f, kindNamed(t, f.service.Kinds(), "mediaCode")
}

// A restore from the recycle bin returns the activity to Created (activity/kinds.go R14), so its facts must
// come back held — otherwise unapproved data counts in every figure (FR-020, FR-021).
func TestKinds_RestoredFactComesBackHeld_FR021(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if err := f.store.PutFact(t.Context(), performance.Fact{ID: "PF-1", SourceID: "PS-1", ImportID: "PIM-1",
		SourceReference: "r1", ActivityID: "A-1001", Stage: "inq", Held: false}); err != nil {
		t.Fatalf("PutFact: %v", err)
	}
	kind := kindNamed(t, f.service.Kinds(), "performanceFact")
	snapshot, _, err := kind.Snapshot(t.Context(), "PF-1")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if _, err = kind.PrepareRestore(t.Context(), history.RestoreContext{}, &snapshot); err != nil {
		t.Fatalf("PrepareRestore: %v", err)
	}
	if err = kind.Put(t.Context(), snapshot); err != nil {
		t.Fatalf("Put: %v", err)
	}

	back, found, _ := f.store.GetFact(t.Context(), "PF-1")
	if !found {
		t.Fatal("the fact did not come back")
	}
	if !back.Held {
		t.Error("a restored fact came back unheld: it would count under an activity returned to Created")
	}
}

func TestKinds_UndoKeepsTheFactAsItWas_FR021(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if err := f.store.PutFact(t.Context(), performance.Fact{ID: "PF-1", SourceID: "PS-1", ImportID: "PIM-1",
		SourceReference: "r1", ActivityID: "A-1001", Stage: "inq", Held: false}); err != nil {
		t.Fatalf("PutFact: %v", err)
	}
	kind := kindNamed(t, f.service.Kinds(), "performanceFact")
	snapshot, _, _ := kind.Snapshot(t.Context(), "PF-1")

	if _, err := kind.PrepareRestore(t.Context(), history.RestoreContext{Undo: true}, &snapshot); err != nil {
		t.Fatalf("PrepareRestore: %v", err)
	}
	if err := kind.Put(t.Context(), snapshot); err != nil {
		t.Fatalf("Put: %v", err)
	}

	back, _, _ := f.store.GetFact(t.Context(), "PF-1")
	if back.Held {
		t.Error("an Undo re-held the fact: an Undo keeps the activity's status, so the fact keeps its own")
	}
}
