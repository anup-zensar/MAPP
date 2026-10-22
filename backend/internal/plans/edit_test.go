package plans

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// User story 4 — Edit a plan.

// Every save sends the version it started from; without one, or with one below 1, the save is refused
// next to the field "version", together with every other problem, and nothing changes (FR-023).
func TestUpdate_MissingVersion_FR023(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, priya, input("Versioned plan"))
	stored := f.stored(t, created.ID)
	noVersion := updateOf(stored)
	noVersion.Version = nil
	zero := updateOf(stored)
	zero.Version, zero.Name = ptr(0), " "

	_, errMissing := f.service.Update(t.Context(), priya, created.ID, noVersion)
	_, errZero := f.service.Update(t.Context(), priya, created.ID, zero)

	assertError(t, errMissing, problems("version", "Reload the page and try again."))
	assertError(t, errZero, problems("name", "Enter a plan name.", "version", "Reload the page and try again."))
	if after := f.stored(t, created.ID); !reflect.DeepEqual(after, stored) {
		t.Errorf("stored = %+v, want it unchanged", after)
	}
}

// A save from an out-of-date version is refused, naming who changed the plan and when; nothing is
// overwritten or logged (FR-023, spec edge case).
func TestUpdate_VersionConflict_FR023(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, priya, input("Shared plan"))
	stored := f.stored(t, created.ID)
	f.clock.Advance(5*time.Hour + 32*time.Minute)
	byDaniel := updateOf(stored)
	byDaniel.Description = "Daniel's version"
	if _, err := f.service.Update(t.Context(), daniel, created.ID, byDaniel); err != nil {
		t.Fatalf("Daniel's save: %v", err)
	}
	entries := len(f.entries(t))
	byPriya := updateOf(stored) // still version 1
	byPriya.Description = "Priya's version"

	_, err := f.service.Update(t.Context(), priya, created.ID, byPriya)

	assertError(t, err, errs.VersionConflict("plan", "Daniel Okoro", start.Add(5*time.Hour+32*time.Minute)))
	if after := f.stored(t, created.ID); after.Description != "Daniel's version" || after.Version != 2 {
		t.Errorf("stored = %q at version %d, want Daniel's at version 2", after.Description, after.Version)
	}
	if got := len(f.entries(t)); got != entries {
		t.Errorf("%d log entries, want still %d", got, entries)
	}
}

// A save that changes nothing creates no version and no log entry (FR-031, research R5).
func TestUpdate_NoChangeNoLog_FR031(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, priya, input("Unchanged plan"))
	stored := f.stored(t, created.ID)
	same := updateOf(stored)
	same.Name = "  Unchanged plan " // the same name once trimmed

	view, err := f.service.Update(t.Context(), daniel, created.ID, same)

	if err != nil || view.Version != 1 || view.UpdatedBy != priya {
		t.Errorf("view at version %d, changed by %+v, %v; want version 1 by Priya", view.Version, view.UpdatedBy, err)
	}
	if entries := f.entries(t); len(entries) != 1 {
		t.Errorf("%d log entries, want only the created one", len(entries))
	}
}

// A Published plan stays Published when it is edited; the log holds one "changed" entry with every changed
// field's old and new value in words, each Shell attribute on its own; the year never changes (FR-018,
// FR-022, FR-031, US4 scenario 5).
func TestUpdate_PublishedStaysPublishedAndLogsFields_FR022(t *testing.T) {
	f := newFixture(t)
	f.put(t, seedPlan(t, "P-1001", "FY2026 Lubricants Europe", 2026, teamLubricants, priya.ID, "1200000.00", StatusPublished))
	edit := updateOf(f.stored(t, "P-1001"))
	edit.Budget = &money.Input{Amount: "1300000", Currency: "USD"}
	edit.ModelID = ptr(mobilityModel)
	edit.TeamIDs = []string{teamMobility, teamLubricants}
	edit.Attributes = attributesOf(lubricants, mobility)
	f.clock.Advance(time.Hour)

	view, err := f.service.Update(t.Context(), daniel, "P-1001", edit)

	if err != nil || view.Status != StatusPublished || view.Version != 2 || view.Year != 2026 || view.UpdatedBy != daniel {
		t.Fatalf("view: %s, version %d, year %d, by %+v, %v; want Published, version 2, 2026, by Daniel",
			view.Status, view.Version, view.Year, view.UpdatedBy, err)
	}
	entries := f.entries(t)
	if len(entries) != 1 || entries[0].Action != history.ActionChanged || entries[0].Actor != daniel {
		t.Fatalf("log = %+v, want one changed entry by Daniel", entries)
	}
	type text struct{ field, label, old, new string }
	var got []text
	for _, c := range entries[0].Changes {
		got = append(got, text{c.Field, c.Label, c.OldText, c.NewText})
	}
	want := []text{
		{"budget", "Plan budget", "1,200,000.00 USD", "1,300,000.00 USD"},
		{"modelId", "Model", "Lubricants 1.0", "Mobility 1.0"},
		{"teamIds", "Teams", "Lubricants Europe", "Lubricants Europe, Mobility China"},
		{"attributes.SA-BU", "Business unit", "Lubricants", "Lubricants, Mobility"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("changes = %+v\nwant      %+v", got, want)
	}
}

// When the owner is in none of the chosen teams, the save is refused naming the person, until another
// owner is chosen or the field is cleared (FR-021, R1, US4 scenario 4).
func TestUpdate_OwnerNotInTeams_FR021(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, priya, input("Owned plan"))
	edit := updateOf(f.stored(t, created.ID))
	edit.TeamIDs = []string{teamMobility} // Priya's only team removed

	_, err := f.service.Update(t.Context(), priya, created.ID, edit)

	assertError(t, err, problems("ownerId", "Priya Raman isn't in any of the chosen teams — choose another owner or clear the field."))
	edit.OwnerID = nil
	view, err := f.service.Update(t.Context(), priya, created.ID, edit)
	if err != nil || view.Owner != nil || !slices.Equal(f.stored(t, created.ID).TeamIDs, []string{teamMobility}) {
		t.Errorf("owner %+v, %v; want saved without owner", view.Owner, err)
	}
}

// A model, team or owner that became inactive stays on the plan when it is saved again; only choosing
// one anew is refused (spec edge case, 002 FR-027).
func TestUpdate_InactiveValuesKept_EdgeCase(t *testing.T) {
	f := newFixture(t)
	p := seedPlan(t, "P-1001", "Old settings", 2026, teamClosed, "U-1009", "", StatusDraft)
	p.ModelID = oldModel
	f.put(t, p)
	edit := updateOf(f.stored(t, "P-1001"))
	edit.Description = "Still here"

	view, err := f.service.Update(t.Context(), priya, "P-1001", edit)

	if err != nil || view.Model.Active || view.Model.Name != "Old model" || view.Teams[0].Active || view.Owner.Active {
		t.Errorf("model %+v, teams %+v, owner %+v, %v; want the inactive ones kept", view.Model, view.Teams, view.Owner, err)
	}
	if view.Description != "Still here" {
		t.Errorf("description = %q", view.Description)
	}
}

// Values of Shell attributes that a business-unit change hides are kept, so they come back when the
// business unit is added again (FR-020, R9, US4 scenario 3).
func TestUpdate_HiddenAttributeValuesKept_FR020(t *testing.T) {
	f := newFixture(t)
	in := input("Mobility plan")
	in.Attributes = settings.AttributeValues{businessUnit: json.RawMessage(`["SO-MOB"]`), smp: json.RawMessage(`true`)}
	created := f.create(t, priya, in)
	edit := updateOf(f.stored(t, created.ID))
	edit.Attributes[businessUnit] = json.RawMessage(`["SO-LUB"]`) // SMP is hidden now; the form keeps its value

	if _, err := f.service.Update(t.Context(), priya, created.ID, edit); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if stored := f.stored(t, created.ID); string(stored.Attributes[smp]) != "true" {
		t.Errorf("attributes = %s, want the hidden SMP value kept", stored.Attributes)
	}
}

// Before a model change is saved, the activities whose type the new model does not allow are listed; the
// same model lists nothing, and an unknown one is refused (FR-019, R7, US4 scenario 2).
func TestChangeEffects_DisallowedActivities_FR019(t *testing.T) {
	f := newFixture(t)
	created := f.create(t, priya, input("Model change"))
	activities := []errs.RecordRef{{Type: "activity", ID: "A-1003", Name: "Launch film"}, {Type: "activity", ID: "A-1005", Name: "Fleet day"}}
	level := f.addLevel(t, "activity", nil)
	level.disallowed = activities
	lubricantsUnit := attributesOf(lubricants)

	changed, err := f.service.ChangeEffects(t.Context(), priya, created.ID, ChangeEffectsRequest{ModelID: ptr(mobilityModel), Attributes: lubricantsUnit})
	same, _ := f.service.ChangeEffects(t.Context(), priya, created.ID, ChangeEffectsRequest{ModelID: ptr(lubricantsModel), Attributes: lubricantsUnit})
	_, errUnknown := f.service.ChangeEffects(t.Context(), priya, created.ID, ChangeEffectsRequest{ModelID: ptr("MOD-9"), Attributes: lubricantsUnit})
	_, errGone := f.service.ChangeEffects(t.Context(), priya, "P-9999", ChangeEffectsRequest{})

	want := ChangeEffects{ModelChanged: true, DisallowedActivities: activities, HiddenGroups: []AttributeGroupChange{}, ShownGroups: []AttributeGroupChange{}}
	if err != nil || !reflect.DeepEqual(changed, want) {
		t.Errorf("new model: %+v, %v\nwant %+v", changed, err, want)
	}
	if same.ModelChanged || len(same.DisallowedActivities) != 0 {
		t.Errorf("same model: %+v, want no change", same)
	}
	assertError(t, errUnknown, problems("modelId", "Choose a model from the list."))
	assertError(t, errGone, errs.NotFound("This plan no longer exists — someone may have deleted it."))
}

// Before a business-unit change is saved, the attribute groups it hides or shows are listed by unit, with
// the levels they are on and their labels (FR-020, R9, US4 scenario 3).
func TestChangeEffects_AttributeGroups_FR020(t *testing.T) {
	f := newFixture(t)
	in := input("Both units")
	in.Attributes = attributesOf(mobility, lubricants)
	created := f.create(t, priya, in)

	withoutLubricants, err := f.service.ChangeEffects(t.Context(), priya, created.ID,
		ChangeEffectsRequest{ModelID: ptr(lubricantsModel), Attributes: attributesOf(mobility)})
	withoutMobility, _ := f.service.ChangeEffects(t.Context(), priya, created.ID,
		ChangeEffectsRequest{ModelID: ptr(lubricantsModel), Attributes: attributesOf(lubricants)})

	lubricantsGroup := AttributeGroupChange{Unit: "Lubricants", Levels: []string{"activity"}, Attributes: []string{"Brands", "Sector"}}
	mobilityGroup := AttributeGroupChange{Unit: "Mobility", Levels: []string{"plan", "activity"},
		Attributes: []string{"Strategic Marketing Plan (SMP)", "Function (Mobility)"}}
	if err != nil || !reflect.DeepEqual(withoutLubricants.HiddenGroups, []AttributeGroupChange{lubricantsGroup}) ||
		len(withoutLubricants.ShownGroups) != 0 {
		t.Errorf("without Lubricants: %+v, %v", withoutLubricants, err)
	}
	if !reflect.DeepEqual(withoutMobility.HiddenGroups, []AttributeGroupChange{mobilityGroup}) {
		t.Errorf("without Mobility: %+v", withoutMobility)
	}

	f.put(t, func() Plan { p := f.stored(t, created.ID); p.Attributes = attributesOf(mobility); return p }())
	adding, _ := f.service.ChangeEffects(t.Context(), priya, created.ID,
		ChangeEffectsRequest{ModelID: ptr(lubricantsModel), Attributes: attributesOf(mobility, lubricants)})
	if !reflect.DeepEqual(adding.ShownGroups, []AttributeGroupChange{lubricantsGroup}) || len(adding.HiddenGroups) != 0 {
		t.Errorf("adding Lubricants: %+v, want its group shown", adding)
	}
}
