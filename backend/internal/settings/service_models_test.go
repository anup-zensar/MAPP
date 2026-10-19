package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
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

// A new model with valid values is listed as Active, with its allowed types by name, and logged (US1-1,
// FR-009).
func TestCreateModel_ListedActive_FR009(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	digital := f.addType(t, "Digital media", true)

	created := f.createModel(t, modelInput("  Lubricants 2.0 ", event.ID, digital.ID, event.ID))

	want := ModelView{
		ID: "MOD-1001", Name: "Lubricants 2.0", InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(500),
		AverageDealSize: money.USD(money.FromCents(1_000_000)),
		ActivityTypes:   []SettingRef{{ID: digital.ID, Name: "Digital media", Active: true}, {ID: event.ID, Name: "Event", Active: true}},
		Active:          true, Usage: Usage{Summary: "Not used yet", Parts: []UsagePart{}, Records: []history.RecordRef{}},
		Meta: record.NewMeta(alex, start),
	}
	if !reflect.DeepEqual(created, want) {
		t.Errorf("created = %+v\nwant      %+v", created, want)
	}
	listed, err := f.service.ListModels(t.Context(), alex)
	if err != nil || len(listed) != 1 || !reflect.DeepEqual(listed[0], want) {
		t.Errorf("ListModels = %+v, %v", listed, err)
	}
	stored, _, _ := f.service.Model(t.Context(), created.ID)
	if !reflect.DeepEqual(stored.ActivityTypeIDs, []string{digital.ID, event.ID}) {
		t.Errorf("stored types = %q, want once each in the order of the type list", stored.ActivityTypeIDs)
	}
	entry := f.lastEntry(t)
	if entry.Action != history.ActionCreated || entry.RecordType != "model" || entry.RecordName != "Lubricants 2.0" || entry.Actor != alex {
		t.Errorf("log entry = %+v", entry)
	}
}

// Every problem of a new model is shown at once and nothing is saved (US1-5, FR-009, SC-003).
func TestCreateModel_AllProblemsAtOnce_FR009(t *testing.T) {
	f := newTestService(t)
	input := modelInput("", "")
	input.InqToMqlRate = "0"

	_, err := f.service.CreateModel(t.Context(), alex, input)

	assertError(t, err, problems(
		"name", "Enter a name.",
		"inqToMqlRate", "Enter a rate above 0 % and at most 100 %, with at most two decimals.",
		"activityTypeIds", "Tick at least one activity type.",
	))
	if models, _ := f.service.Models(t.Context()); len(models) != 0 {
		t.Errorf("models = %+v, want none", models)
	}
}

// Each rate is above 0 % and at most 100 %, with at most two decimals (FR-009).
func TestCreateModel_RateBounds_FR009(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	for _, rate := range []string{"0", "0.00", "100.01", "2.555", "abc", "", "-1", "1,5"} {
		input := modelInput("M "+rate, event.ID)
		input.MqlToDealRate = rate
		_, err := f.service.CreateModel(t.Context(), alex, input)
		assertError(t, err, problems("mqlToDealRate", "Enter a rate above 0 % and at most 100 %, with at most two decimals."))
	}
	for i, rate := range []string{"0.01", "100", " 33.33 "} {
		input := modelInput("Valid "+string(rune('A'+i)), event.ID)
		input.InqToMqlRate = rate
		if _, err := f.service.CreateModel(t.Context(), alex, input); err != nil {
			t.Errorf("rate %q refused: %v", rate, err)
		}
	}
}

// The average deal size is in USD, above 0 and at most 100,000,000.00, with at most two decimals (FR-009,
// D-053).
func TestCreateModel_DealSizeAboveZeroUSD_FR009(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	tests := []struct {
		name    string
		size    *money.Input
		message string
	}{
		{"not filled in", nil, "Enter an average deal size above 0 USD, with at most two decimals."},
		{"empty", &money.Input{Amount: "", Currency: "USD"}, "Enter an average deal size above 0 USD, with at most two decimals."},
		{"zero", &money.Input{Amount: "0", Currency: "USD"}, "Enter an average deal size above 0 USD, with at most two decimals."},
		{"negative", &money.Input{Amount: "-5", Currency: "USD"}, "Enter an average deal size above 0 USD, with at most two decimals."},
		{"three decimals", &money.Input{Amount: "1.234", Currency: "USD"}, "Enter an average deal size above 0 USD, with at most two decimals."},
		{"too large", &money.Input{Amount: "100000000.01", Currency: "USD"}, "Use 100,000,000.00 USD or less."},
		{"far too large", &money.Input{Amount: "1000000000000000000", Currency: "USD"}, "Use 100,000,000.00 USD or less."},
		{"in euros", &money.Input{Amount: "5000", Currency: "EUR"}, "Enter the average deal size in USD."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := modelInput("Model", event.ID)
			input.AverageDealSize = tt.size
			_, err := f.service.CreateModel(t.Context(), alex, input)
			assertError(t, err, problems("averageDealSize", tt.message))
		})
	}
	input := modelInput("Largest", event.ID)
	input.AverageDealSize = &money.Input{Amount: "100000000.00", Currency: "USD"}
	if _, err := f.service.CreateModel(t.Context(), alex, input); err != nil {
		t.Errorf("100,000,000.00 USD refused: %v", err)
	}
}

// A model's name has at most 80 characters and is unique ignoring letter case and surrounding spaces; the
// message names the model it clashes with (FR-005, FR-009).
func TestCreateModel_NameUniqueIgnoringCaseAndSpaces_FR005(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	f.createModel(t, modelInput("Lubricants 1.0", event.ID))

	_, taken := f.service.CreateModel(t.Context(), alex, modelInput("  LUBRICANTS 1.0 ", event.ID))
	_, long := f.service.CreateModel(t.Context(), alex, modelInput(strings.Repeat("é", 81), event.ID))

	assertError(t, taken, problems("name", "A model called 'Lubricants 1.0' already exists."))
	assertError(t, long, problems("name", "Use 80 characters or fewer."))
	f.createModel(t, modelInput(strings.Repeat("é", 80), event.ID))
}

// A new model can allow active activity types only; a type that no longer exists is reported (FR-004).
func TestCreateModel_InactiveTypeRefused_FR004(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	webinar := f.addType(t, "Webinar", false)

	_, err := f.service.CreateModel(t.Context(), alex, modelInput("Model", event.ID, webinar.ID, "AT-9999"))

	assertError(t, err, problems(
		"activityTypeIds", "'Webinar' is inactive — tick active activity types only.",
		"activityTypeIds", "An activity type you ticked no longer exists — reload the page and try again.",
	))
}

// A change of a model in use logs every old and new value in words, and says that the goals of its plans
// were recalculated (US1-4, FR-006, FR-011).
func TestUpdateModel_LogsEachOldAndNewValue_FR006_FR011(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	f.refs = []Reference{
		planRef("P-1001", "Plan A", m.ID, false, nil), planRef("P-1002", "Plan B", m.ID, false, nil),
		activityRef("A-1001", "P-1001", event.ID, false, nil), activityRef("A-1002", "P-1002", event.ID, false, nil),
		activityRef("A-1003", "P-1002", event.ID, true, nil), // in the recycle bin: not counted
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
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	webinar := f.addType(t, "Webinar", true)
	m := f.createModel(t, modelInput("Model", event.ID, webinar.ID))
	f.write(t, func(ctx context.Context, repo Repository) error {
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

// Before new rates or deal size are saved, the plans and activities of a model in use are stated (FR-011,
// US1-4); typed goals stay, as the message says.
func TestModelImpact_RatesInUse_FR011(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	f.refs = []Reference{planRef("P-1001", "A", m.ID, false, nil), planRef("P-1002", "B", m.ID, false, nil), planRef("P-1003", "C", m.ID, true, nil)}
	for i := range 31 {
		f.refs = append(f.refs, activityRef(fmt.Sprintf("A-%d", 1001+i), "P-1001", event.ID, false, nil))
	}

	impact, err := f.service.ModelImpact(t.Context(), alex, m.ID, ModelUpdate{MqlToDealRate: text("6"), Version: version(1)})

	want := []ImpactWarning{{Kind: "goalsRecalculated", Count: 2, Message: "Used by 2 plans (31 activities). Their calculated MQL and " +
		"revenue goals will be recalculated; typed goals ('Set later stages myself') are kept."}}
	if err != nil || !reflect.DeepEqual(impact.Warnings, want) {
		t.Errorf("impact = %+v, %v\nwant %+v", impact, err, want)
	}
	if stored, _, _ := f.service.Model(t.Context(), m.ID); stored.MqlToDealRate.String() != "5.00" {
		t.Errorf("the impact check saved the rate: %s", stored.MqlToDealRate)
	}
	_, invalid := f.service.ModelImpact(t.Context(), alex, m.ID, ModelUpdate{MqlToDealRate: text("0")})
	assertError(t, invalid, problems("mqlToDealRate", "Enter a rate above 0 % and at most 100 %, with at most two decimals."))
}

// A model nothing uses has nothing to confirm; a change of its name alone neither (FR-011).
func TestModelImpact_Unused_NoWarning_FR011(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Model", event.ID))
	used := f.createModel(t, modelInput("Used", event.ID))
	f.refs = []Reference{planRef("P-1001", "A", used.ID, false, nil)}

	unused, err1 := f.service.ModelImpact(t.Context(), alex, m.ID, ModelUpdate{InqToMqlRate: text("3"), Version: version(1)})
	renamed, err2 := f.service.ModelImpact(t.Context(), alex, used.ID, ModelUpdate{Name: text("Used 2"), Version: version(1)})

	if err1 != nil || err2 != nil || len(unused.Warnings) != 0 || len(renamed.Warnings) != 0 {
		t.Errorf("impacts = %+v (%v), %+v (%v); want no warnings", unused, err1, renamed, err2)
	}
}

// Removing an allowed activity type states how many activities in the model's plans use it; they keep it
// (US1-6, FR-012, R7).
func TestModelImpact_RemovedTypeInUse_FR012(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	digital := f.addType(t, "Digital media", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID, digital.ID))
	other := f.createModel(t, modelInput("Other", event.ID))
	f.refs = []Reference{planRef("P-1001", "A", m.ID, false, nil), planRef("P-1002", "B", other.ID, false, nil)}
	for _, id := range []string{"A-1001", "A-1002", "A-1003", "A-1004", "A-1005"} {
		f.refs = append(f.refs, activityRef(id, "P-1001", event.ID, false, nil))
	}
	f.refs = append(f.refs,
		activityRef("A-1006", "P-1002", event.ID, false, nil), // another model's plan
		activityRef("A-1007", "P-1001", event.ID, true, nil),  // in the recycle bin
		activityRef("A-1008", "P-1001", digital.ID, false, nil))

	five, err1 := f.service.ModelImpact(t.Context(), alex, m.ID, ModelUpdate{ActivityTypeIDs: &[]string{digital.ID}})
	one, err2 := f.service.ModelImpact(t.Context(), alex, m.ID, ModelUpdate{ActivityTypeIDs: &[]string{event.ID}})

	wantFive := []ImpactWarning{{Kind: "activityTypeRemoved", Count: 5, Message: "5 activities use 'Event'. They keep it; new activities can't choose it."}}
	wantOne := []ImpactWarning{{Kind: "activityTypeRemoved", Count: 1, Message: "1 activity uses 'Digital media'. It keeps it; new activities can't choose it."}}
	if err1 != nil || !reflect.DeepEqual(five.Warnings, wantFive) {
		t.Errorf("removing Event: %+v, %v", five, err1)
	}
	if err2 != nil || !reflect.DeepEqual(one.Warnings, wantOne) {
		t.Errorf("removing Digital media: %+v, %v", one, err2)
	}
}

// A model's usage counts its plans in the recycle bin too, and lists the live ones (FR-003, US1-3).
func TestModelUsage_CountsRecycleBin_FR003(t *testing.T) {
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	f.refs = []Reference{planRef("P-1001", "Lubricants FY2026", m.ID, false, nil), planRef("P-1003", "Old", m.ID, true, nil)}

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
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	inBin := f.createModel(t, modelInput("Mobility 1.0", event.ID))
	f.refs = []Reference{planRef("P-1001", "A", m.ID, false, nil), planRef("P-1002", "B", m.ID, false, nil), planRef("P-1003", "C", inBin.ID, true, nil)}

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
