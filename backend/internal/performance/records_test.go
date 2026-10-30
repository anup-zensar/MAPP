package performance_test

import (
	"testing"

	"mapp/internal/history"
	"mapp/internal/performance"
)

func TestCreateSource_WritesOneActivityLogEntry_FR045(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	src := f.createSource(t, "Marketing automation")

	entries := f.log(t)
	if len(entries) != 1 {
		t.Fatalf("after CreateSource the log holds %d entries, want exactly 1", len(entries))
	}
	if entries[0].RecordID != src.ID || entries[0].Action != history.ActionCreated {
		t.Errorf("log entry = %s/%s, want %s/created", entries[0].RecordID, entries[0].Action, src.ID)
	}
}

func TestSetSourceStatus_LogsDeactivationAndReactivation_FR004_FR045(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Marketing automation")

	if err := f.service.SetSourceStatus(t.Context(), alex, src.ID, performance.StatusInactive); err != nil {
		t.Fatalf("SetSourceStatus(inactive): %v", err)
	}
	if err := f.service.SetSourceStatus(t.Context(), alex, src.ID, performance.StatusActive); err != nil {
		t.Fatalf("SetSourceStatus(active): %v", err)
	}

	entries := f.log(t)
	if len(entries) != 3 {
		t.Fatalf("the log holds %d entries, want 3: created, deactivated, reactivated", len(entries))
	}
	verbs := []string{entries[0].Verb, entries[1].Verb}
	if verbs[0] != "reactivated" || verbs[1] != "deactivated" {
		t.Errorf("verbs newest first = %v, want [reactivated deactivated]", verbs)
	}
}

func TestSetSourceStatus_RaisesTheVersionEachTime_FR045(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Marketing automation")
	if src.Version != 1 {
		t.Fatalf("a new source has version %d, want 1", src.Version)
	}

	if err := f.service.SetSourceStatus(t.Context(), alex, src.ID, performance.StatusInactive); err != nil {
		t.Fatalf("SetSourceStatus: %v", err)
	}

	found, err := f.store.ListSources(t.Context(), src.ID)
	if err != nil || len(found) != 1 {
		t.Fatalf("ListSources() = %v, %v", found, err)
	}
	if found[0].Version != 2 {
		t.Errorf("after one change the version is %d, want 2", found[0].Version)
	}
}

func TestKinds_RegistersTheDataSource_FR045(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	kinds := f.service.Kinds()

	if len(kinds) != 1 || kinds[0].Name != "performanceSource" {
		t.Fatalf("Kinds() = %+v, want one kind named performanceSource", kinds)
	}
	if kinds[0].Snapshot == nil || kinds[0].Remove == nil || kinds[0].Put == nil {
		t.Error("the kind must carry Snapshot, Remove and Put so the recycle bin can hold a data source")
	}
}

func TestDeleteSource_LogsAndGoesToTheRecycleBin_FR045(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Marketing automation")

	if err := f.service.DeleteSource(t.Context(), alex, src.ID); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}

	entries := f.log(t)
	if len(entries) != 2 {
		t.Fatalf("after create and delete the log holds %d entries, want 2", len(entries))
	}
	if entries[0].Action != history.ActionDeleted || entries[0].RecordID != src.ID {
		t.Errorf("newest entry = %s/%s, want %s/deleted", entries[0].RecordID, entries[0].Action, src.ID)
	}
}
