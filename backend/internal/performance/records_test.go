package performance_test

import (
	"errors"
	"testing"

	"mapp/internal/history"
	"mapp/internal/performance"
	"mapp/internal/platform/errs"
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

	source := kindNamed(t, f.service.Kinds(), "performanceSource")

	if source.Parent != "" {
		t.Errorf("the data source's parent is %q, want none: it is a top-level record", source.Parent)
	}
	if source.Snapshot == nil || source.Remove == nil || source.Put == nil {
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

func TestUpdateSource_StaleVersionRefused_FR045(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Marketing automation")
	edit := performance.SourceInput{Name: "Renamed", OwnerID: alex.ID,
		Category: performance.CategoryCRM, DeliveryRoute: performance.RouteUpload,
		Cadence: performance.CadenceMonthly, Version: src.Version}

	if _, err := f.service.UpdateSource(t.Context(), alex, src.ID, edit); err != nil {
		t.Fatalf("UpdateSource with the current version: %v", err)
	}

	_, err := f.service.UpdateSource(t.Context(), alex, src.ID, edit)

	var conflict *errs.Error
	if !errors.As(err, &conflict) || conflict.Code != errs.CodeVersionConflict {
		t.Fatalf("a second save against the old version answered %v, want %s", err, errs.CodeVersionConflict)
	}
	if conflict.Conflict == nil || conflict.Conflict.ChangedBy != alex.Name {
		t.Errorf("the refusal does not name who changed it: %+v", conflict.Conflict)
	}
}

func TestUpdateSource_KeepsTheValuesAndRaisesTheVersion_FR045(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Marketing automation")

	saved, err := f.service.UpdateSource(t.Context(), alex, src.ID, performance.SourceInput{
		Name: "Renamed", OwnerID: alex.ID, Category: performance.CategoryCRM,
		DeliveryRoute: performance.RouteUpload, Cadence: performance.CadenceWeekly, Version: src.Version,
	})
	if err != nil {
		t.Fatalf("UpdateSource: %v", err)
	}

	if saved.Name != "Renamed" || saved.Cadence != performance.CadenceWeekly {
		t.Errorf("saved = %s/%s, want Renamed/weekly", saved.Name, saved.Cadence)
	}
	if saved.Version != src.Version+1 {
		t.Errorf("version is %d after one change, want %d", saved.Version, src.Version+1)
	}
}
