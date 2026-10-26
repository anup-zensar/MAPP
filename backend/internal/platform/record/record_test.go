package record_test

import (
	"encoding/json"
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

// A new record starts at version 1, and "updated" equals "created" until the first change (data model).
func TestNewMeta_StartsAtVersion1(t *testing.T) {
	got := record.NewMeta(priya, created)

	want := record.Meta{Version: 1, CreatedAt: created, CreatedBy: priya, UpdatedAt: created, UpdatedBy: priya}
	if got != want {
		t.Errorf("NewMeta = %+v, want %+v", got, want)
	}
}

// Every saved change raises the version and records who changed the record and when; "created" stays
// (FR-029).
func TestTouch_RaisesTheVersionAndRecordsTheChange_FR029(t *testing.T) {
	meta := record.NewMeta(priya, created)

	meta.Touch(daniel, changed)

	want := record.Meta{Version: 2, CreatedAt: created, CreatedBy: priya, UpdatedAt: changed, UpdatedBy: daniel}
	if meta != want {
		t.Errorf("after Touch = %+v, want %+v", meta, want)
	}
}

func TestCheckVersion_SameVersionIsAccepted_FR029(t *testing.T) {
	meta := record.NewMeta(priya, created)

	if err := meta.CheckVersion(1, "sample folder"); err != nil {
		t.Errorf("CheckVersion(1) = %v, want nil", err)
	}
}

// A save based on an out-of-date copy is refused, naming who changed the record and when (FR-029).
func TestCheckVersion_OutOfDateSaveRefusedNamingWhoAndWhen_FR029(t *testing.T) {
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

// Meta is embedded in every business record and its fields appear at the top level of the record's JSON
// (data model "Bookkeeping").
func TestMeta_FieldsAreFlattenedIntoTheRecord(t *testing.T) {
	type folder struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		record.Meta
	}

	body, err := json.Marshal(folder{ID: "SF-1001", Name: "Ideas", Meta: record.NewMeta(priya, created)})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	want := `{"id":"SF-1001","name":"Ideas","version":1,` +
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},` +
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1002","name":"Priya Raman"}}`
	if string(body) != want {
		t.Errorf("JSON =\n%s\nwant\n%s", body, want)
	}
}
