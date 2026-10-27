package settings

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// modelInput is a valid new model: 2 %, 5 %, 10,000 USD, allowing the activity types.
func modelInput(name string, typeIDs ...string) ModelInput {
	return ModelInput{Name: name, InqToMqlRate: "2", MqlToDealRate: "5", AverageDealSize: &money.Input{Amount: "10000", Currency: "USD"},
		ActivityTypeIDs: typeIDs}
}

// createModel creates a model as a and stops the test when that fails.
func (f *fixture) createModel(t *testing.T, input ModelInput) ModelView {
	t.Helper()
	view, err := f.service.CreateModel(t.Context(), alex, input)
	if err != nil {
		t.Fatalf("CreateModel(%q): %v", input.Name, err)
	}
	return view
}

// A change of a model in use logs every old and new value in words, and says that the goals of its plans
// were recalculated (US1-4, FR-006, FR-011).
func TestUpdateModel_LogsEachOldAndNewValue_FR006_FR011(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	f.refs = []Reference{
		planRef("P-1001", "Plan A", m.ID, nil), planRef("P-1002", "Plan B", m.ID, nil),
		activityRef("A-1001", "P-1001", event.ID, nil), activityRef("A-1002", "P-1002", event.ID, nil),
		inRecycleBin(activityRef("A-1003", "P-1002", event.ID, nil)), // in the recycle bin: not counted
	}
	f.clock.now = start.Add(time.Hour)

	updated, err := f.service.UpdateModel(t.Context(), daniel, m.ID, ModelUpdate{
		InqToMqlRate: text("2.5"), AverageDealSize: OptionalMoney{Set: true, Value: &money.Input{Amount: "12000", Currency: "USD"}},
		Version: version(1),
	})
	if err != nil {
		t.Fatalf("UpdateModel: %v", err)
	}

	if updated.Version != 2 || updated.UpdatedBy != daniel || !updated.UpdatedAt.Equal(f.clock.now) || updated.InqToMqlRate.String() != "2.50" {
		t.Errorf("updated = %+v", updated)
	}
	entry := f.lastEntry(t)
	want := []history.Change{
		{Field: "averageDealSize", Label: "Average deal size", Old: json.RawMessage(`{"amount":"10000.00","currency":"USD"}`),
			New: json.RawMessage(`{"amount":"12000.00","currency":"USD"}`), OldText: "10,000.00 USD", NewText: "12,000.00 USD"},
		{Field: "inqToMqlRate", Label: "INQ→MQL rate", Old: json.RawMessage(`"2.00"`), New: json.RawMessage(`"2.50"`), OldText: "2.00 %", NewText: "2.50 %"},
	}
	if entry.Action != history.ActionChanged || entry.Actor != daniel || !reflect.DeepEqual(entry.Changes, want) ||
		entry.Details != "Used by 2 plans (2 activities): calculated goals recalculated." {
		t.Errorf("log entry = %+v", entry)
	}
}

// An inactive activity type the model already allows may stay; it cannot be added again once removed
// (FR-004).
func TestUpdateModel_KeepsInactiveTypeAlreadyAllowed_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	webinar := f.addType(t, "Webinar", true)
	m := f.createModel(t, modelInput("Model", event.ID, webinar.ID))
	f.write(t, func(ctx context.Context, repo Store) error {
		webinar.Active = false
		return repo.PutActivityType(ctx, webinar)
	})

	renamed, err := f.service.UpdateModel(t.Context(), alex, m.ID, ModelUpdate{Name: text("Model 2"), ActivityTypeIDs: &[]string{event.ID, webinar.ID}, Version: version(1)})
	if err != nil || len(renamed.ActivityTypes) != 2 || renamed.ActivityTypes[1].Active {
		t.Errorf("keeping the inactive type: %+v, %v", renamed.ActivityTypes, err)
	}
	removed, err := f.service.UpdateModel(t.Context(), alex, m.ID, ModelUpdate{ActivityTypeIDs: &[]string{event.ID}, Version: version(2)})
	if err != nil || len(removed.ActivityTypes) != 1 {
		t.Fatalf("removing it: %+v, %v", removed, err)
	}
	_, err = f.service.UpdateModel(t.Context(), alex, m.ID, ModelUpdate{ActivityTypeIDs: &[]string{event.ID, webinar.ID}, Version: version(3)})
	assertError(t, err, problems("activityTypeIds", "'Webinar' is inactive — tick active activity types only."))
}

// A save from an out-of-date copy is refused, naming who changed the model and when; a missing version is
// reported with the other problems; a model that is gone answers not found (FR-007).
func TestUpdateModel_VersionConflict_FR007(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Model", event.ID))
	f.clock.now = start.Add(92 * time.Minute)
	if _, err := f.service.UpdateModel(t.Context(), daniel, m.ID, ModelUpdate{Name: text("Daniel's model"), Version: version(1)}); err != nil {
		t.Fatalf("Daniel's update: %v", err)
	}

	_, stale := f.service.UpdateModel(t.Context(), alex, m.ID, ModelUpdate{Name: text("Alex's model"), Version: version(1)})
	_, noVersion := f.service.UpdateModel(t.Context(), alex, m.ID, ModelUpdate{Name: text("")})
	_, gone := f.service.UpdateModel(t.Context(), alex, "MOD-9999", ModelUpdate{Name: text("X"), Version: version(1)})

	assertError(t, stale, errs.VersionConflict("model", "Daniel Okoro", f.clock.now))
	assertError(t, noVersion, problems("version", "Reload the page and try again.", "name", "Enter a name."))
	assertError(t, gone, errs.NotFound("This model no longer exists — someone may have deleted it."))
}

// A save that changes nothing writes no new version and no log entry (FR-006).
func TestUpdateModel_NoChangeNoLog_FR006(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Model", event.ID))
	entries := len(f.logEntries(t))

	same, err := f.service.UpdateModel(t.Context(), alex, m.ID, ModelUpdate{
		Name: text(" Model "), InqToMqlRate: text("2.00"), ActivityTypeIDs: &[]string{event.ID}, Version: version(1),
	})

	if err != nil || same.Version != 1 || len(f.logEntries(t)) != entries {
		t.Errorf("unchanged save: version %d, %v, %d log entries (was %d)", same.Version, err, len(f.logEntries(t)), entries)
	}
}

// A model's usage counts its plans in the recycle bin too, and lists the live ones (FR-003, US1-3).
func TestModelUsage_CountsRecycleBin_FR003(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	f.refs = []Reference{planRef("P-1001", "Lubricants FY2026", m.ID, nil), inRecycleBin(planRef("P-1003", "Old", m.ID, nil))}

	view, err := f.service.GetModel(t.Context(), alex, m.ID)

	want := Usage{InUse: true, Summary: "Used by 2 plans (1 in the recycle bin)",
		Parts:   []UsagePart{{Kind: "plan", Text: "Used by 2 plans (1 in the recycle bin)", Count: 2, InRecycleBin: 1}},
		Records: []history.RecordRef{{Type: "plan", ID: "P-1001", Name: "Lubricants FY2026"}}}
	if err != nil || !reflect.DeepEqual(view.Usage, want) {
		t.Errorf("usage = %+v, %v\nwant %+v", view.Usage, err, want)
	}
	_, gone := f.service.GetModel(t.Context(), alex, "MOD-9999")
	assertError(t, gone, errs.NotFound("This model no longer exists — someone may have deleted it."))
}

// A model in use cannot be deleted — also when only a plan in the recycle bin uses it: the answer says
// where it is used and to set it inactive instead (US1-7, FR-004, SC-004).
func TestDeleteModel_InUseRefused_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	inBin := f.createModel(t, modelInput("Mobility 1.0", event.ID))
	f.refs = []Reference{planRef("P-1001", "A", m.ID, nil), planRef("P-1002", "B", m.ID, nil), inRecycleBin(planRef("P-1003", "C", inBin.ID, nil))}

	_, used := f.service.DeleteModel(t.Context(), alex, m.ID)
	_, usedInBin := f.service.DeleteModel(t.Context(), alex, inBin.ID)

	assertError(t, used, errs.InUse("'Lubricants 1.0' is in use (used by 2 plans) — set it inactive instead."))
	assertError(t, usedInBin, errs.InUse("'Mobility 1.0' is in use (used by 1 plan (1 in the recycle bin)) — set it inactive instead."))
	if models, _ := f.service.Models(t.Context()); len(models) != 2 {
		t.Errorf("models = %d, want both kept", len(models))
	}
}

// An unused model moves to the recycle bin and Undo brings it back exactly (US1-7, FR-004).
func TestDeleteModel_UnusedToBin_Undo_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	stored, _, _ := f.service.Model(t.Context(), m.ID)

	entry, err := f.service.DeleteModel(t.Context(), alex, m.ID)
	if err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	if _, found, _ := f.service.Model(t.Context(), m.ID); found || entry.RecordType != "model" || entry.RecordName != "Lubricants 1.0" {
		t.Errorf("after the delete: entry %+v, still stored %v", entry, found)
	}
	if _, err := f.history.Undo(t.Context(), alex, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if back, found, _ := f.service.Model(t.Context(), m.ID); !found || !reflect.DeepEqual(back, stored) {
		t.Errorf("after Undo = %+v, want %+v", back, stored)
	}
	_, gone := f.service.DeleteModel(t.Context(), alex, "MOD-9999")
	assertError(t, gone, errs.NotFound("This model no longer exists — someone may have deleted it."))
}

// Set inactive and Reactivate are logged with their own verbs and need the current version; the same
// status again changes nothing (FR-004, FR-006).
func TestSetModelActive_LoggedAndReversible_FR004_FR006(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))

	inactive, err := f.service.SetModelActive(t.Context(), alex, m.ID, StatusUpdate{Active: yes(false), Version: version(1)})
	if err != nil || inactive.Active || inactive.Version != 2 {
		t.Fatalf("set inactive: %+v, %v", inactive, err)
	}
	deactivated := f.lastEntry(t)
	again, err := f.service.SetModelActive(t.Context(), alex, m.ID, StatusUpdate{Active: yes(false), Version: version(2)})
	if err != nil || again.Version != 2 || len(f.logEntries(t)) != 2 {
		t.Errorf("the same status again: %+v, %v", again, err)
	}
	active, err := f.service.SetModelActive(t.Context(), alex, m.ID, StatusUpdate{Active: yes(true), Version: version(2)})
	if err != nil || !active.Active || active.Version != 3 {
		t.Fatalf("reactivate: %+v, %v", active, err)
	}
	reactivated := f.lastEntry(t)

	wantChange := history.Change{Field: "active", Label: "Status", Old: json.RawMessage(`true`), New: json.RawMessage(`false`), OldText: "Active", NewText: "Inactive"}
	if deactivated.Action != history.ActionStatusChanged || deactivated.Verb != "deactivated" || !reflect.DeepEqual(deactivated.Changes, []history.Change{wantChange}) {
		t.Errorf("deactivated entry = %+v", deactivated)
	}
	if reactivated.Verb != "reactivated" || reactivated.Changes[0].NewText != "Active" {
		t.Errorf("reactivated entry = %+v", reactivated)
	}
	_, stale := f.service.SetModelActive(t.Context(), alex, m.ID, StatusUpdate{Active: yes(false), Version: version(1)})
	assertError(t, stale, errs.VersionConflict("model", "Alex Morgan", start))
	_, missing := f.service.SetModelActive(t.Context(), alex, m.ID, StatusUpdate{})
	assertError(t, missing, problems("active", "Reload the page and try again.", "version", "Reload the page and try again."))
}

// When another model has taken the name meanwhile, Undo is refused and Restore renames the model (FR-004,
// data model §11).
func TestModelRestore_NameTaken_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	entry, err := f.service.DeleteModel(t.Context(), alex, m.ID)
	if err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	f.createModel(t, modelInput("lubricants 1.0", event.ID))

	_, undo := f.history.Undo(t.Context(), alex, entry.ID)
	restored, err := f.history.Restore(t.Context(), daniel, entry.ID)

	assertError(t, undo, errs.NameTaken("Can't undo — there is now another model called 'lubricants 1.0'. Restore it from the recycle bin instead."))
	if err != nil || restored.RecordName != "Lubricants 1.0 (restored)" || restored.Details != "Renamed to 'Lubricants 1.0 (restored)' because the name was taken." {
		t.Errorf("Restore = %+v, %v", restored, err)
	}
	if back, _, _ := f.service.Model(t.Context(), m.ID); back.Name != "Lubricants 1.0 (restored)" || back.Version != 2 || back.UpdatedBy != daniel {
		t.Errorf("restored model = %+v", back)
	}
}
