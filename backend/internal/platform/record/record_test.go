package record_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

var (
	priya   = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel  = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	created = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	changed = time.Date(2026, 3, 12, 14, 32, 0, 0, time.UTC)
)

// Every saved change raises the version and records who changed the record and when; "created" stays
// (FR-029).
func TestTouch_RaisesTheVersionAndRecordsTheChange_FR029(t *testing.T) {
	t.Parallel()
	meta := record.NewMeta(priya, created)

	meta.Touch(daniel, changed)

	want := record.Meta{Version: 2, CreatedAt: created, CreatedBy: priya, UpdatedAt: changed, UpdatedBy: daniel}
	if meta != want {
		t.Errorf("after Touch = %+v, want %+v", meta, want)
	}
}

func TestCheckVersion_SameVersionIsAccepted_FR029(t *testing.T) {
	t.Parallel()
	meta := record.NewMeta(priya, created)

	if err := meta.CheckVersion(1, "sample folder"); err != nil {
		t.Errorf("CheckVersion(1) = %v, want nil", err)
	}
}

// A save based on an out-of-date copy is refused, naming who changed the record and when (FR-029).
func TestCheckVersion_OutOfDateSaveRefusedNamingWhoAndWhen_FR029(t *testing.T) {
	t.Parallel()
	meta := record.NewMeta(priya, created)
	meta.Touch(daniel, changed) // Daniel saved first: the record is now at version 2

	err := meta.CheckVersion(1, "sample folder")

	var apiErr *errs.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("CheckVersion(1) = %v, want an *errs.Error", err)
	}
	if apiErr.Status != http.StatusConflict || apiErr.Code != errs.CodeVersionConflict {
		t.Errorf("got %d %s, want 409 version_conflict", apiErr.Status, apiErr.Code)
	}
	if want := "Not saved — Daniel Okoro changed this sample folder"; apiErr.Message != want {
		t.Errorf("message = %q, want %q", apiErr.Message, want)
	}
	if apiErr.Conflict == nil || *apiErr.Conflict != (errs.Conflict{ChangedBy: "Daniel Okoro", ChangedAt: changed}) {
		t.Errorf("conflict = %+v, want Daniel Okoro at %v", apiErr.Conflict, changed)
	}
}
