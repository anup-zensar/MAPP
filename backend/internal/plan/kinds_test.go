package plan

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// User story 6 — Delete a plan, with Undo.

// fakeChildren is a kind of record below the plan, such as 005's campaigns, kept in memory: enough for
// history to move it to the recycle bin with its plan and bring it back.
type fakeChildren struct {
	records map[string]history.Snapshot // id → the record, its parent's id in ParentID
}

// register registers the fake kind "campaign" under "plan" and stores the records ids under planID.
func (c *fakeChildren) register(t *testing.T, f *fixture, planID string, ids ...string) {
	t.Helper()
	c.records = map[string]history.Snapshot{}
	for _, id := range ids {
		c.records[id] = history.Snapshot{ID: id, Name: "Campaign " + id, ParentID: planID, Data: json.RawMessage(`{"id":"` + id + `"}`)}
	}
	err := f.history.Register(history.Kind{
		Name: "campaign", Singular: "campaign", Plural: "campaigns", Parent: kindName,
		Snapshot: func(_ context.Context, id string) (history.Snapshot, bool, error) {
			s, found := c.records[id]
			return s, found, nil
		},
		ChildIDs: func(_ context.Context, parentID string) ([]string, error) {
			var childIDs []string
			for id, s := range c.records {
				if s.ParentID == parentID {
					childIDs = append(childIDs, id)
				}
			}
			slices.Sort(childIDs)
			return childIDs, nil
		},
		Remove: func(_ context.Context, id string) error { delete(c.records, id); return nil },
		Put:    func(_ context.Context, s history.Snapshot) error { c.records[s.ID] = s; return nil },
	})
	if err != nil {
		t.Fatalf("Register(campaign): %v", err)
	}
}

// deletePlan deletes the plan id as a and stops the test when that fails.
func (f *fixture) deletePlan(t *testing.T, id string) history.BinEntry {
	t.Helper()
	entry, err := f.service.Delete(t.Context(), priya, id)
	if err != nil {
		t.Fatalf("Delete(%s): %v", id, err)
	}
	return entry
}

// Deleting a plan moves it, with everything below it, to the recycle bin as one entry, and logs one
// "deleted" entry naming what it held; the plan is gone (FR-029, US6 scenario 3).
func TestDelete_MovesToBin_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, input("FY2026 Lubricants Test"))
	var campaigns fakeChildren
	campaigns.register(t, f, plan.ID, "C-1001", "C-1002")

	entry, err := f.service.Delete(t.Context(), daniel, plan.ID)

	if err != nil || entry.RecordType != "plan" || entry.RecordTypeLabel != "plan" || entry.RecordID != plan.ID ||
		entry.RecordName != "FY2026 Lubricants Test" || entry.DeletedBy != daniel || len(entry.Snapshots) != 3 {
		t.Fatalf("Delete = %+v, %v; want the plan and its 2 campaigns in one entry, by Daniel", entry, err)
	}
	if want := []history.ContentCount{{RecordType: "campaign", Label: "campaigns", Count: 2}}; !reflect.DeepEqual(entry.Contents, want) {
		t.Errorf("contents = %+v, want %+v", entry.Contents, want)
	}
	if _, err := f.service.Get(t.Context(), priya, plan.ID); err == nil || len(campaigns.records) != 0 {
		t.Errorf("after the delete: Get error %v, %d campaigns left; want not found and none", err, len(campaigns.records))
	}
	entries := f.entries(t)
	if len(entries) != 2 || entries[0].Action != history.ActionDeleted || entries[0].Actor != daniel ||
		entries[0].Details != "with 2 campaigns" {
		t.Errorf("log = %+v, want one deleted entry by Daniel \"with 2 campaigns\"", entries)
	}
	bin, err := f.history.ListBin(t.Context(), priya)
	if err != nil || len(bin) != 1 {
		t.Errorf("recycle bin = %d entries, %v; want 1", len(bin), err)
	}
}

// A plan that does not exist cannot be deleted (FR-029).
func TestDelete_NotFound_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.Delete(t.Context(), priya, "P-9999")

	assertError(t, err, errs.NotFound(planGoneMessage))
}

// Undo puts the plan back exactly as it was: the same ID, version and status, Published included, and
// what was below it (FR-030, US6 scenario 4).
func TestUndo_RestoresExactly_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.put(t, seedPlan(t, "P-1001", "FY2026 Lubricants Europe", 2026, teamLubricants, priya.ID, "1200000.00", StatusPublished))
	before := f.stored(t, "P-1001")
	var campaigns fakeChildren
	campaigns.register(t, f, "P-1001", "C-1001")
	entry := f.deletePlan(t, "P-1001")
	f.clock.Advance(10 * time.Second)

	result, err := f.history.Undo(t.Context(), priya, entry.ID)

	if err != nil || result.RecordID != "P-1001" || result.RecordName != "FY2026 Lubricants Europe" || result.Details != "" {
		t.Fatalf("Undo = %+v, %v; want P-1001 back without notes", result, err)
	}
	if after, want := asJSON(t, f.stored(t, "P-1001")), asJSON(t, before); after != want {
		t.Errorf("after Undo: %s\nwant         %s", after, want)
	}
	if _, found := campaigns.records["C-1001"]; !found {
		t.Error("the campaign did not come back with its plan")
	}
}

// While the plan was in the bin, another plan took its name: Undo is refused and says to restore it from
// the recycle bin instead (spec edge case, plan "Restore notes").
func TestUndo_NameTaken_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, input("FY2026 Lubricants Test"))
	entry := f.deletePlan(t, plan.ID)
	f.create(t, daniel, input("fy2026 lubricants test"))

	_, err := f.history.Undo(t.Context(), priya, entry.ID)

	assertError(t, err, errs.NameTaken("Can't undo — there is now another plan called 'fy2026 lubricants test'. "+
		"Restore it from the recycle bin instead."))
	if _, err := f.service.Get(t.Context(), priya, plan.ID); err == nil {
		t.Error("the plan came back although the Undo was refused")
	}
}

// A Restore from the recycle bin renames a plan whose name is taken and returns a Published plan as Draft,
// saying both in its notes, with a new version (R14, plan "Restore notes").
func TestRestore_RenamesAndReturnsDraft_R14(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.put(t, seedPlan(t, "P-1050", "FY2026 Lubricants Europe", 2026, teamLubricants, priya.ID, "", StatusPublished))
	entry := f.deletePlan(t, "P-1050")
	f.create(t, priya, input("FY2026 Lubricants Europe"))
	f.clock.Advance(time.Hour)

	result, err := f.history.Restore(t.Context(), daniel, entry.ID)

	wantNotes := "Renamed to 'FY2026 Lubricants Europe (restored)' because the name was taken. " +
		"Returned as Draft — publish it again when it's ready."
	if err != nil || result.RecordName != "FY2026 Lubricants Europe (restored)" || result.Details != wantNotes {
		t.Fatalf("Restore = %+v, %v; want renamed with the notes %q", result, err, wantNotes)
	}
	after := f.stored(t, "P-1050")
	if after.Name != "FY2026 Lubricants Europe (restored)" || after.Status != StatusDraft || after.Version != 2 ||
		after.UpdatedBy != daniel {
		t.Errorf("restored plan: %q, %s, version %d, by %+v; want renamed, Draft, version 2, by Daniel",
			after.Name, after.Status, after.Version, after.UpdatedBy)
	}
}

// A deleted plan's name is free again (spec edge case, FR-010).
func TestCreate_DeletedNameReusable_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, input("FY2026 Lubricants Test"))
	f.deletePlan(t, plan.ID)

	_, err := f.service.Create(t.Context(), priya, input("FY2026 Lubricants Test"))

	if err != nil {
		t.Errorf("Create with a deleted plan's name: %v, want no error", err)
	}
}

// The activity log shows a plan's fields by their form labels and its values as people read them: names
// for ids, money with its currency, and the words of the choices (FR-031, plan "Activity log").
func TestDescribe_ReadableLog_FR031(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	cases := []struct {
		field       string
		value       any
		label, text string
	}{
		{"name", "FY2026 Lubricants Europe", "Plan name", ""},
		{"modelId", lubricantsModel, "Model", "Lubricants 1.0"},
		{"teamIds", []any{teamLubricants, teamMobility}, "Teams", "Lubricants Europe, Mobility China"},
		{"ownerId", priya.ID, "Owner", "Priya Raman"},
		{"aop", true, "Strategic / AOP plan", ""},
		{"budget", "1200000.00", "Plan budget", "1,200,000.00 USD"},
		{"defaultFinanceBudgetId", budget2026, "Default finance budget", "2026 Lubricants Budget"},
		{"defaultSpread", "firstMonth", "Default line-item spread", "All in first month"},
		{"bcpir", false, "BCPIR plan", ""},
		{"goalStage", "revenue", "Goal stage", "Revenue"},
		{"goalAmount", "250000", "Goal amount", "250,000"},
		{"goalAmount", "2500000.00", "Goal amount", "2,500,000.00 USD"},
		{"status", "published", "Status", "Published"},
		{"modelId", "MOD-9999", "Model", ""}, // no longer resolves: history shows the id
	}
	for _, c := range cases {
		label, text, err := f.service.describe(t.Context(), c.field, c.value)

		if err != nil || label != c.label || text != c.text {
			t.Errorf("describe(%s, %v) = %q, %q, %v; want %q, %q", c.field, c.value, label, text, err, c.label, c.text)
		}
	}
}
