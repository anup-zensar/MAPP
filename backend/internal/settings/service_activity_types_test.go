package settings

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// typeInput is a new activity type with a default planned cost in USD ("" for none), allowed in the
// models.
func typeInput(name, cost string, modelIDs ...string) ActivityTypeInput {
	input := ActivityTypeInput{Name: name, ModelIDs: modelIDs}
	if cost != "" {
		input.DefaultPlannedCost = &money.Input{Amount: cost, Currency: "USD"}
	}
	return input
}

// createType creates an activity type as Alex and stops the test when that fails.
func (f *fixture) createType(t *testing.T, input ActivityTypeInput) ActivityTypeView {
	t.Helper()
	view, err := f.service.CreateActivityType(t.Context(), alex, input)
	if err != nil {
		t.Fatalf("CreateActivityType(%q): %v", input.Name, err)
	}
	return view
}

// usd returns a pointer to the USD amount of cents, as an activity type's default planned cost.
func usd(cents int64) *money.Money {
	m := money.USD(money.FromCents(cents))
	return &m
}

// A new activity type is listed as Active, and each model ticked on the type's form allows it from then on;
// the model's change is logged on the model (US5-1, FR-013).
func TestCreateActivityType_AllowedInModel_FR013(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	lubricants := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	mobility := f.createModel(t, modelInput("Mobility 1.0", event.ID))
	f.clock.now = start.Add(time.Hour)

	created, err := f.service.CreateActivityType(t.Context(), daniel, typeInput(" Webinar ", "5000", lubricants.ID, lubricants.ID))
	if err != nil {
		t.Fatalf("CreateActivityType: %v", err)
	}

	want := ActivityTypeView{
		ID: "AT-1002", Name: "Webinar", DefaultPlannedCost: usd(500_000),
		Models: []SettingRef{{ID: lubricants.ID, Name: "Lubricants 1.0", Active: true}}, Active: true,
		Usage: Usage{InUse: true, Summary: "Allowed in 1 model", Parts: []UsagePart{{Kind: "model", Text: "Allowed in 1 model", Count: 1}},
			Records: []history.RecordRef{}},
		Meta: record.NewMeta(daniel, f.clock.now),
	}
	if !reflect.DeepEqual(created, want) {
		t.Errorf("created = %+v\nwant      %+v", created, want)
	}
	listed, err := f.service.ListActivityTypes(t.Context(), alex)
	if err != nil || len(listed) != 2 || listed[1].Name != "Webinar" || !listed[1].Active {
		t.Errorf("ListActivityTypes = %+v, %v", listed, err)
	}
	allowing, _, _ := f.service.Model(t.Context(), lubricants.ID)
	if !reflect.DeepEqual(allowing.ActivityTypeIDs, []string{event.ID, created.ID}) || allowing.Version != 2 || allowing.UpdatedBy != daniel {
		t.Errorf("Lubricants 1.0 = %+v, want it to allow Webinar after Event", allowing)
	}
	if other, _, _ := f.service.Model(t.Context(), mobility.ID); other.Version != 1 || len(other.ActivityTypeIDs) != 1 {
		t.Errorf("Mobility 1.0 changed: %+v", other)
	}
	entries := f.logEntries(t)
	createdEntry, modelEntry := entries[len(entries)-2], entries[len(entries)-1]
	if createdEntry.Action != history.ActionCreated || createdEntry.RecordType != "activityType" || createdEntry.RecordName != "Webinar" {
		t.Errorf("created entry = %+v", createdEntry)
	}
	wantChange := history.Change{Field: "activityTypeIds", Label: "Allowed activity types", Old: json.RawMessage(`["AT-1001"]`),
		New: json.RawMessage(`["AT-1001","AT-1002"]`), OldText: "Event", NewText: "Event, Webinar"}
	if modelEntry.Action != history.ActionChanged || modelEntry.RecordType != "model" || modelEntry.RecordName != "Lubricants 1.0" ||
		modelEntry.Actor != daniel || !reflect.DeepEqual(modelEntry.Changes, []history.Change{wantChange}) {
		t.Errorf("model entry = %+v", modelEntry)
	}

	plain := f.createType(t, typeInput("Content production", ""))
	if plain.DefaultPlannedCost != nil || len(plain.Models) != 0 || plain.Usage.InUse {
		t.Errorf("a type without default cost or models = %+v", plain)
	}
}

// Every problem of an activity type is shown at once: the name (unique ignoring case and spaces, FR-005),
// the default planned cost (0 or more USD) and the models ticked (FR-013, SC-003). Nothing is saved.
func TestActivityType_Validation_FR013_FR005(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)

	_, first := f.service.CreateActivityType(t.Context(), alex, ActivityTypeInput{Name: " ",
		DefaultPlannedCost: &money.Input{Amount: "-1", Currency: "USD"}, ModelIDs: []string{"MOD-9999"}})
	_, second := f.service.CreateActivityType(t.Context(), alex, ActivityTypeInput{Name: "  EVENT ",
		DefaultPlannedCost: &money.Input{Amount: "100", Currency: "EUR"}})
	_, third := f.service.CreateActivityType(t.Context(), alex, typeInput(strings.Repeat("é", 121), "12.345"))
	zero, err := f.service.CreateActivityType(t.Context(), alex, typeInput(strings.Repeat("é", 120), "0"))
	_, renamed := f.service.UpdateActivityType(t.Context(), alex, zero.ID, ActivityTypeUpdate{Name: text("event"), Version: version(1)})

	assertError(t, first, problems("name", "Enter a name.",
		"defaultPlannedCost", "Enter an amount of 0 or more, with at most two decimals.",
		"modelIds", "A model you ticked no longer exists — reload the page and try again."))
	assertError(t, second, problems("name", "An activity type called 'Event' already exists.",
		"defaultPlannedCost", "Enter the default planned cost in USD."))
	assertError(t, third, problems("name", "Use 120 characters or fewer.",
		"defaultPlannedCost", "Enter an amount of 0 or more, with at most two decimals."))
	if err != nil || !reflect.DeepEqual(zero.DefaultPlannedCost, usd(0)) {
		t.Errorf("a default cost of 0 = %+v, %v", zero, err)
	}
	assertError(t, renamed, problems("name", "An activity type called 'Event' already exists."))
	if types, _ := f.service.ActivityTypes(t.Context()); len(types) != 2 || types[0].ID != event.ID {
		t.Errorf("activity types = %+v, want Event and the one valid type", types)
	}
}

// The default planned cost can change and be removed (sent as null); left out, it stays. Each change is
// logged with the old and new amount; a save from an out-of-date copy is refused (FR-013, FR-006, FR-007).
func TestUpdateActivityType_DefaultCostChangedAndRemoved_FR013(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	webinar := f.createType(t, typeInput("Webinar", "5000"))
	f.clock.now = start.Add(time.Hour)

	changed, err := f.service.UpdateActivityType(t.Context(), daniel, webinar.ID, ActivityTypeUpdate{
		DefaultPlannedCost: OptionalMoney{Set: true, Value: &money.Input{Amount: "7500.5", Currency: "USD"}}, Version: version(1),
	})
	if err != nil {
		t.Fatalf("changing the cost: %v", err)
	}
	changedEntry := f.lastEntry(t)
	removed, err := f.service.UpdateActivityType(t.Context(), daniel, webinar.ID, ActivityTypeUpdate{
		DefaultPlannedCost: OptionalMoney{Set: true}, Version: version(2),
	})
	if err != nil {
		t.Fatalf("removing the cost: %v", err)
	}
	removedEntry := f.lastEntry(t)
	entries := len(f.logEntries(t))
	same, err := f.service.UpdateActivityType(t.Context(), alex, webinar.ID, ActivityTypeUpdate{Name: text(" Webinar "), Version: version(3)})

	if !reflect.DeepEqual(changed.DefaultPlannedCost, usd(750_050)) || changed.Version != 2 || changed.UpdatedBy != daniel {
		t.Errorf("changed = %+v", changed)
	}
	wantChange := history.Change{Field: "defaultPlannedCost", Label: "Default planned cost",
		Old: json.RawMessage(`{"amount":"5000.00","currency":"USD"}`), New: json.RawMessage(`{"amount":"7500.50","currency":"USD"}`),
		OldText: "5,000.00 USD", NewText: "7,500.50 USD"}
	if changedEntry.Action != history.ActionChanged || changedEntry.RecordType != "activityType" || !reflect.DeepEqual(changedEntry.Changes, []history.Change{wantChange}) {
		t.Errorf("changed entry = %+v", changedEntry)
	}
	if removed.DefaultPlannedCost != nil || removed.Version != 3 || removedEntry.Changes[0].NewText != "Not filled in" {
		t.Errorf("removed = %+v, entry %+v", removed, removedEntry)
	}
	if err != nil || same.Version != 3 || same.DefaultPlannedCost != nil || len(f.logEntries(t)) != entries {
		t.Errorf("an unchanged save = %+v, %v", same, err)
	}
	_, stale := f.service.UpdateActivityType(t.Context(), alex, webinar.ID, ActivityTypeUpdate{Name: text("Webinars"), Version: version(1)})
	_, noVersion := f.service.UpdateActivityType(t.Context(), alex, webinar.ID, ActivityTypeUpdate{Name: text("")})
	_, gone := f.service.UpdateActivityType(t.Context(), alex, "AT-9999", ActivityTypeUpdate{Name: text("X"), Version: version(1)})
	assertError(t, stale, errs.VersionConflict("activity type", "Daniel Okoro", f.clock.now))
	assertError(t, noVersion, problems("version", "Reload the page and try again.", "name", "Enter a name."))
	assertError(t, gone, errs.NotFound("This activity type no longer exists — someone may have deleted it."))
}

// The models allowing a type are edited from the type: modelIds is the complete new list, so models added
// gain the type and models left out lose it, each with one "changed" entry on the model. The type itself
// does not change (data model §3, §12; FR-013).
func TestUpdateActivityType_ModelsEditedFromType_FR013(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	lubricants := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	mobility := f.createModel(t, modelInput("Mobility 1.0", event.ID))
	webinar := f.createType(t, typeInput("Webinar", "", lubricants.ID))
	entries := len(f.logEntries(t))

	updated, err := f.service.UpdateActivityType(t.Context(), daniel, webinar.ID, ActivityTypeUpdate{
		ModelIDs: &[]string{mobility.ID}, Version: version(1),
	})
	if err != nil {
		t.Fatalf("UpdateActivityType: %v", err)
	}

	if !reflect.DeepEqual(updated.Models, []SettingRef{{ID: mobility.ID, Name: "Mobility 1.0", Active: true}}) || updated.Version != 1 {
		t.Errorf("updated = %+v, want only Mobility 1.0 and the type's own version unchanged", updated)
	}
	lost, _, _ := f.service.Model(t.Context(), lubricants.ID)
	gained, _, _ := f.service.Model(t.Context(), mobility.ID)
	if !reflect.DeepEqual(lost.ActivityTypeIDs, []string{event.ID}) || lost.Version != 3 {
		t.Errorf("Lubricants 1.0 = %+v", lost)
	}
	if !reflect.DeepEqual(gained.ActivityTypeIDs, []string{event.ID, webinar.ID}) || gained.Version != 2 || gained.UpdatedBy != daniel {
		t.Errorf("Mobility 1.0 = %+v", gained)
	}
	log := f.logEntries(t)[entries:]
	if len(log) != 2 || log[0].RecordName != "Lubricants 1.0" || log[0].Changes[0].NewText != "Event" ||
		log[1].RecordName != "Mobility 1.0" || log[1].Changes[0].NewText != "Event, Webinar" || log[1].RecordType != "model" {
		t.Errorf("log = %+v, want one changed entry per model, by name", log)
	}

	_, gone := f.service.UpdateActivityType(t.Context(), alex, webinar.ID, ActivityTypeUpdate{
		ModelIDs: &[]string{mobility.ID, "MOD-9999"}, Version: version(1),
	})
	assertError(t, gone, problems("modelIds", "A model you ticked no longer exists — reload the page and try again."))
}

// A model must keep at least one activity type, so the type a model allows alone cannot be removed from it
// — from the type's form either (FR-009, data model §3). Every such model is named.
func TestUpdateActivityType_LastTypeOfModelRefused_FR009(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	digital := f.addType(t, "Digital media", true)
	lubricants := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	f.createModel(t, modelInput("Mobility 1.0", event.ID))
	both := f.createModel(t, modelInput("Both", event.ID, digital.ID))

	_, refused := f.service.UpdateActivityType(t.Context(), alex, event.ID, ActivityTypeUpdate{ModelIDs: &[]string{}, Version: version(1)})
	kept, err := f.service.UpdateActivityType(t.Context(), alex, event.ID, ActivityTypeUpdate{
		ModelIDs: &[]string{lubricants.ID, f.modelID(t, "Mobility 1.0")}, Version: version(1),
	})

	assertError(t, refused, problems(
		"modelIds", "'Lubricants 1.0' needs at least one activity type, so 'Event' can't be removed from it.",
		"modelIds", "'Mobility 1.0' needs at least one activity type, so 'Event' can't be removed from it."))
	if err != nil || len(kept.Models) != 2 {
		t.Errorf("removing Event from the model that has another type = %+v, %v", kept, err)
	}
	if stored, _, _ := f.service.Model(t.Context(), both.ID); !reflect.DeepEqual(stored.ActivityTypeIDs, []string{digital.ID}) {
		t.Errorf("Both = %q, want only Digital media", stored.ActivityTypeIDs)
	}
}

// modelID returns the id of the stored model called name.
func (f *fixture) modelID(t *testing.T, name string) string {
	t.Helper()
	models, err := f.service.Models(t.Context())
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	for _, m := range models {
		if m.Name == name {
			return m.ID
		}
	}
	t.Fatalf("no model called %q", name)
	return ""
}

// An unused type can be set inactive and reactivated, each logged; deleted, it moves to the recycle bin and
// Undo brings it back exactly (US5-5, FR-004, FR-006).
func TestDeleteActivityType_Unused_Undo_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	webinar := f.createType(t, typeInput("Webinar", "5000"))

	inactive, err := f.service.SetActivityTypeActive(t.Context(), alex, webinar.ID, StatusUpdate{Active: yes(false), Version: version(1)})
	if err != nil || inactive.Active || inactive.Version != 2 || f.lastEntry(t).Verb != "deactivated" {
		t.Fatalf("set inactive = %+v, %v", inactive, err)
	}
	_, stale := f.service.SetActivityTypeActive(t.Context(), alex, webinar.ID, StatusUpdate{Active: yes(true), Version: version(1)})
	assertError(t, stale, errs.VersionConflict("activity type", "Alex Morgan", start))
	active, err := f.service.SetActivityTypeActive(t.Context(), alex, webinar.ID, StatusUpdate{Active: yes(true), Version: version(2)})
	if err != nil || !active.Active || active.Version != 3 || f.lastEntry(t).Verb != "reactivated" {
		t.Fatalf("reactivate = %+v, %v", active, err)
	}
	stored, _, _ := f.service.ActivityType(t.Context(), webinar.ID)

	entry, err := f.service.DeleteActivityType(t.Context(), alex, webinar.ID)
	if err != nil || entry.RecordType != "activityType" || entry.RecordName != "Webinar" {
		t.Fatalf("DeleteActivityType = %+v, %v", entry, err)
	}
	if _, found, _ := f.service.ActivityType(t.Context(), webinar.ID); found {
		t.Errorf("Webinar is still stored")
	}
	if _, err := f.history.Undo(t.Context(), alex, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if back, found, _ := f.service.ActivityType(t.Context(), webinar.ID); !found || !reflect.DeepEqual(back, stored) {
		t.Errorf("after Undo = %+v, want %+v", back, stored)
	}
}
