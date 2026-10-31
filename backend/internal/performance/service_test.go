package performance_test

import (
	"errors"
	"strings"
	"testing"

	"mapp/internal/performance"
	"mapp/internal/platform/errs"
)

func TestCreateSource_ReportsEveryProblemAtOnce_FR001(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.CreateSource(t.Context(), alex, performance.SourceInput{})

	var invalid *errs.Error
	if !errors.As(err, &invalid) {
		t.Fatalf("CreateSource() error = %v, want an *errs.Error", err)
	}
	if len(invalid.Fields) != 2 {
		t.Errorf("CreateSource() reported %d fields, want 2 together: %+v", len(invalid.Fields), invalid.Fields)
	}
}

func TestDeleteSource_RefusedWhileItHoldsFacts_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Marketing automation")
	if err := f.store.PutFact(t.Context(), performance.Fact{
		ID: "PF-1", SourceID: src.ID, ImportID: "PIM-1", SourceReference: "r1", Stage: "inq",
	}); err != nil {
		t.Fatalf("PutFact: %v", err)
	}

	err := f.service.DeleteSource(t.Context(), alex, src.ID)

	var refusal *errs.Error
	if !errors.As(err, &refusal) || refusal.Code != errs.CodeInUse {
		t.Fatalf("DeleteSource() error = %v, want %s", err, errs.CodeInUse)
	}
	if !strings.Contains(refusal.Message, "Deactivate") {
		t.Errorf("DeleteSource() message = %q, want it to offer Deactivate", refusal.Message)
	}
}

func TestDeleteSource_AllowedWhileItHoldsNone_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Marketing automation")

	if err := f.service.DeleteSource(t.Context(), alex, src.ID); err != nil {
		t.Fatalf("DeleteSource() error = %v, want nil", err)
	}
	left, err := f.store.ListSources(t.Context(), "")
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if len(left) != 0 {
		t.Errorf("after DeleteSource() %d sources remain, want 0", len(left))
	}
}

func TestCheckSupplier_NamesTheSourceAlreadySupplying_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Eloqua")
	if err := f.store.PutMapping(t.Context(), performance.Mapping{
		ID: "PSM-1", SourceID: src.ID, ColumnName: "mql", Meaning: performance.MeaningStage,
		Stage: "mql", ModelID: "MOD-1",
	}); err != nil {
		t.Fatalf("PutMapping: %v", err)
	}

	err := f.service.CheckSupplier(t.Context(), "MOD-1", performance.MeaningStage, "mql")

	var refusal *errs.Error
	if !errors.As(err, &refusal) {
		t.Fatalf("CheckSupplier() error = %v, want a refusal", err)
	}
	if !strings.Contains(refusal.Message, "Eloqua") {
		t.Errorf("CheckSupplier() message = %q, want it to name Eloqua", refusal.Message)
	}
}

func TestCheckSupplier_AnInactiveSourceDoesNotConflict_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Eloqua")
	if err := f.store.PutMapping(t.Context(), performance.Mapping{
		ID: "PSM-1", SourceID: src.ID, ColumnName: "mql", Meaning: performance.MeaningStage,
		Stage: "mql", ModelID: "MOD-1",
	}); err != nil {
		t.Fatalf("PutMapping: %v", err)
	}
	if err := f.service.SetSourceStatus(t.Context(), alex, src.ID, performance.StatusInactive); err != nil {
		t.Fatalf("SetSourceStatus: %v", err)
	}

	if err := f.service.CheckSupplier(t.Context(), "MOD-1", performance.MeaningStage, "mql"); err != nil {
		t.Errorf("CheckSupplier() after deactivating error = %v, want nil: FR-004 keeps a deactivated source's "+
			"mappings but it no longer supplies the stage", err)
	}
}

func TestCheckAliasFree_RefusesANameAnotherActivityHolds_FR035(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if err := f.store.PutAlias(t.Context(), performance.Alias{
		ID: "XN-1", ActivityID: "A-1002", Name: "spring-launch",
	}); err != nil {
		t.Fatalf("PutAlias: %v", err)
	}

	err := f.service.CheckAliasFree(t.Context(), "A-1001", "Spring-Launch")

	var refusal *errs.Error
	if !errors.As(err, &refusal) || refusal.Code != errs.CodeNameTaken {
		t.Fatalf("CheckAliasFree() = %v, want %s: names and aliases share one namespace", err, errs.CodeNameTaken)
	}
	if !strings.Contains(refusal.Message, "A-1002") {
		t.Errorf("the refusal does not name who holds it: %q", refusal.Message)
	}
}

func TestCheckAliasFree_AllowsTheActivitysOwnName_FR035(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if err := f.store.PutAlias(t.Context(), performance.Alias{
		ID: "XN-1", ActivityID: "A-1001", Name: "spring-launch",
	}); err != nil {
		t.Fatalf("PutAlias: %v", err)
	}

	if err := f.service.CheckAliasFree(t.Context(), "A-1001", "spring-launch"); err != nil {
		t.Errorf("CheckAliasFree() for the activity's own alias = %v, want nil", err)
	}
}
