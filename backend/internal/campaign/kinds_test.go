package campaign

import (
	"context"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// US5 — Delete a campaign or initiative and undo it.

// lubricants is the plan P-1001 as a path step of the log and the recycle bin.
var lubricants = errs.RecordRef{Type: planKind, ID: "P-1001", Name: "FY2026 Lubricants Europe"}

// Deleting a campaign moves it with its initiatives and the activities below them to the recycle bin in one
// entry, which counts what was below; one "deleted" log entry says so, and the plan tree no longer shows
// any of them (FR-028, US5 scenario 1).
func TestDeleteCampaign_EverythingInOneBinEntry_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", Budget: input("300000.00")})
	digital := f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Digital & social"})
	printOutdoor := f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Print & outdoor"})
	f.activities.add(
		ActivitySummary{ID: "A-1", InitiativeID: digital.ID, Name: "Launch film"},
		ActivitySummary{ID: "A-2", InitiativeID: digital.ID, Name: "Social always-on"},
	)
	f.advance(time.Hour)

	entry, err := f.service.DeleteCampaign(t.Context(), daniel, helix.ID)

	if err != nil {
		t.Fatalf("DeleteCampaign: %v", err)
	}
	wantContents := []history.ContentCount{
		{RecordType: initiativeKind, Label: "initiatives", Count: 2},
		{RecordType: "activity", Label: "activities", Count: 2},
	}
	if entry.RecordType != campaignKind || entry.RecordID != helix.ID || entry.RecordName != "Helix launch" ||
		!reflect.DeepEqual(entry.Path, []errs.RecordRef{lubricants}) || entry.Parent == nil || *entry.Parent != lubricants ||
		entry.DeletedBy != daniel || !entry.DeletedAt.Equal(start.Add(time.Hour)) || !reflect.DeepEqual(entry.Contents, wantContents) {
		t.Errorf("entry = %+v, want Helix launch in FY2026 Lubricants Europe with 2 initiatives and 2 activities", entry)
	}
	if _, found := f.storedCampaign(t, helix.ID); found {
		t.Error("the campaign is still stored")
	}
	for _, id := range []string{digital.ID, printOutdoor.ID} {
		if _, found := f.storedInitiative(t, id); found {
			t.Errorf("initiative %s is still stored", id)
		}
	}
	if left, _ := f.activities.Summaries(t.Context(), []string{digital.ID}); len(left) != 0 {
		t.Errorf("activities left = %+v", left)
	}
	if tree := f.list(t, "P-1001"); tree.Total != 0 || tree.Totals.AllocatedToCampaigns.Amount.String() != "0.00" {
		t.Errorf("plan tree after the delete = %+v", tree)
	}

	entries := f.log(t, campaignKind, helix.ID)
	if len(entries) != 2 || entries[0].Action != history.ActionDeleted || entries[0].Actor != daniel ||
		entries[0].Details != "with 2 initiatives and 2 activities" || !reflect.DeepEqual(entries[0].Path, []errs.RecordRef{lubricants}) {
		t.Errorf("campaign log = %+v, want one deleted entry with its contents", entries)
	}
	if initiativeLog := f.log(t, initiativeKind, digital.ID); len(initiativeLog) != 1 || initiativeLog[0].Action != history.ActionCreated {
		t.Errorf("initiative log = %+v, want only its created entry (one entry per change)", initiativeLog)
	}
}

// Deleting an initiative takes its activities; its campaign and the other initiatives stay (FR-028).
func TestDeleteInitiative_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	digital := f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Digital & social", Budget: input("100000.00")})
	printOutdoor := f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Print & outdoor"})
	f.activities.add(ActivitySummary{ID: "A-1", InitiativeID: digital.ID, Name: "Launch film"})

	entry, err := f.service.DeleteInitiative(t.Context(), priya, digital.ID)

	if err != nil {
		t.Fatalf("DeleteInitiative: %v", err)
	}
	helixStep := errs.RecordRef{Type: campaignKind, ID: helix.ID, Name: "Helix launch"}
	if entry.RecordType != initiativeKind || entry.RecordID != digital.ID ||
		!reflect.DeepEqual(entry.Path, []errs.RecordRef{lubricants, helixStep}) || entry.Parent == nil || *entry.Parent != helixStep ||
		!reflect.DeepEqual(entry.Contents, []history.ContentCount{{RecordType: "activity", Label: "activity", Count: 1}}) {
		t.Errorf("entry = %+v, want Digital & social under Helix launch with 1 activity", entry)
	}
	if _, found := f.storedInitiative(t, digital.ID); found {
		t.Error("the initiative is still stored")
	}
	if _, found := f.storedInitiative(t, printOutdoor.ID); !found {
		t.Error("the other initiative was deleted too")
	}
	view, err := f.service.GetCampaign(t.Context(), priya, helix.ID)
	if err != nil || view.Counts.Initiatives != 1 || view.BudgetAndCost.AllocatedToChildren.Amount.String() != "0.00" {
		t.Errorf("campaign after the delete = %+v, %v", view, err)
	}
	if entries := f.log(t, initiativeKind, digital.ID); len(entries) != 2 || entries[0].Action != history.ActionDeleted ||
		entries[0].Details != "with 1 activity" {
		t.Errorf("log = %+v, want one deleted entry", entries)
	}
}

// Undo right after a delete puts the campaign, its initiatives and their activities back exactly: same IDs,
// versions, values and plan tree (FR-029, US5 scenario 2).
func TestUndoDelete_RestoresExactly_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{
		Name: "Helix launch", OwnerID: ptr("U-1002"), StartDate: day("2026-02-01"), EndDate: day("2026-11-30"), Budget: input("300000.00"),
	})
	digital := f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Digital & social", Budget: input("100000.00")})
	f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Print & outdoor"})
	f.activities.add(ActivitySummary{ID: "A-1", InitiativeID: digital.ID, Name: "Launch film", PlannedCost: usd("2500.00"), Approved: true})
	storedBefore, _ := f.storedCampaign(t, helix.ID)
	digitalBefore, _ := f.storedInitiative(t, digital.ID)
	treeBefore := f.list(t, "P-1001")
	entry, err := f.service.DeleteCampaign(t.Context(), priya, helix.ID)
	if err != nil {
		t.Fatalf("DeleteCampaign: %v", err)
	}
	f.advance(time.Minute)

	result, err := f.history.Undo(t.Context(), priya, entry.ID)

	if err != nil || result.RecordID != helix.ID || result.RecordName != "Helix launch" || result.Details != "" {
		t.Fatalf("Undo = %+v, %v", result, err)
	}
	if storedAfter, _ := f.storedCampaign(t, helix.ID); !reflect.DeepEqual(storedAfter, storedBefore) {
		t.Errorf("campaign after Undo =\n%+v\nwant\n%+v", storedAfter, storedBefore)
	}
	if digitalAfter, _ := f.storedInitiative(t, digital.ID); !reflect.DeepEqual(digitalAfter, digitalBefore) {
		t.Errorf("initiative after Undo =\n%+v\nwant\n%+v", digitalAfter, digitalBefore)
	}
	if treeAfter := f.list(t, "P-1001"); !reflect.DeepEqual(treeAfter, treeBefore) {
		t.Errorf("plan tree after Undo =\n%+v\nwant\n%+v", treeAfter, treeBefore)
	}
	if entries := f.log(t, campaignKind, helix.ID); len(entries) != 3 || entries[0].Action != history.ActionRestored {
		t.Errorf("log = %+v, want created, deleted, restored", entries)
	}
}

// An Undo must put the record back exactly, so when another campaign of the plan (or initiative of the
// campaign) has taken the name meanwhile it is refused, and the record stays in the recycle bin (FR-029).
func TestUndo_NameTaken_Refused_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	digital := f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Digital & social"})
	initiativeEntry, err := f.service.DeleteInitiative(t.Context(), priya, digital.ID)
	if err != nil {
		t.Fatalf("DeleteInitiative: %v", err)
	}
	f.createInitiative(t, daniel, helix.ID, InitiativeInput{Name: "DIGITAL & SOCIAL"})
	campaignEntry, err := f.service.DeleteCampaign(t.Context(), priya, helix.ID)
	if err != nil {
		t.Fatalf("DeleteCampaign: %v", err)
	}
	f.createCampaign(t, daniel, "P-1001", CampaignInput{Name: "helix launch"})

	_, err = f.history.Undo(t.Context(), priya, campaignEntry.ID)

	assertError(t, err, errs.NameTaken("Can't undo — there is now another campaign called 'helix launch'. Restore it from the recycle bin instead."))
	if _, found := f.storedCampaign(t, helix.ID); found {
		t.Error("the campaign came back")
	}

	// The initiative's campaign is in the bin too now; bring it back first under another name.
	if _, err := f.history.Restore(t.Context(), priya, campaignEntry.ID); err != nil {
		t.Fatalf("Restore campaign: %v", err)
	}
	_, err = f.history.Undo(t.Context(), priya, initiativeEntry.ID)
	assertError(t, err, errs.NameTaken("Can't undo — there is now another initiative called 'DIGITAL & SOCIAL'. Restore it from the recycle bin instead."))
}

// A Restore from the recycle bin brings a record back even when its name is taken: it is renamed to
// "<name> (restored)", or "(restored 2)", … when that is taken too, gets a new version and the note says so
// (FR-029, feature 011 FR-022).
func TestRestore_NameTaken_GetsRestoredSuffix_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	digital := f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Digital & social"})
	entry, err := f.service.DeleteCampaign(t.Context(), priya, helix.ID)
	if err != nil {
		t.Fatalf("DeleteCampaign: %v", err)
	}
	f.createCampaign(t, daniel, "P-1001", CampaignInput{Name: "Helix launch"})
	f.createCampaign(t, daniel, "P-1001", CampaignInput{Name: "Helix launch (restored)"})
	f.advance(time.Hour)

	result, err := f.history.Restore(t.Context(), meiLin, entry.ID)

	note := "Renamed to 'Helix launch (restored 2)' because the name was taken."
	if err != nil || result.RecordName != "Helix launch (restored 2)" || result.Details != note {
		t.Fatalf("Restore = %+v, %v", result, err)
	}
	c, _ := f.storedCampaign(t, helix.ID)
	if c.Name != "Helix launch (restored 2)" || c.Version != 2 || c.UpdatedBy != meiLin || !c.UpdatedAt.Equal(start.Add(time.Hour)) {
		t.Errorf("restored campaign = %+v", c)
	}
	if _, found := f.storedInitiative(t, digital.ID); !found {
		t.Error("the initiative did not come back with its campaign")
	}

	initiativeEntry, err := f.service.DeleteInitiative(t.Context(), priya, digital.ID)
	if err != nil {
		t.Fatalf("DeleteInitiative: %v", err)
	}
	f.createInitiative(t, daniel, helix.ID, InitiativeInput{Name: "Digital & social"})
	result, err = f.history.Restore(t.Context(), meiLin, initiativeEntry.ID)
	if err != nil || result.RecordName != "Digital & social (restored)" ||
		result.Details != "Renamed to 'Digital & social (restored)' because the name was taken." {
		t.Errorf("Restore initiative = %+v, %v", result, err)
	}
}

// The log shows what changed in words people read: labels, the owner's name, dates as "01 Mar 2026",
// budgets as "400,000.00 USD", the parent's name for a move and the Shell attribute's label and options
// (FR-030, data model §7).
func TestDescribe_ReadableValues_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	kinds := f.service.Kinds()
	campaignDescribe, initiativeDescribe := kinds[0].Describe, kinds[1].Describe
	cases := []struct {
		describe func(context.Context, string, any) (string, string, error)
		field    string
		value    any
		label    string
		text     string
	}{
		{campaignDescribe, "name", "Helix launch", "Name", ""},
		{campaignDescribe, "description", "Premium launch", "Description", ""},
		{campaignDescribe, "ownerId", "U-1002", "Owner", "Priya Raman"},
		{campaignDescribe, "ownerId", nil, "Owner", ""},
		{campaignDescribe, "startDate", "2026-03-01", "Start date", "01 Mar 2026"},
		{initiativeDescribe, "endDate", "2026-12-31", "End date", "31 Dec 2026"},
		{campaignDescribe, "budget", map[string]any{"amount": "400000.00", "currency": "USD"}, "Budget", "400,000.00 USD"},
		{campaignDescribe, "parent", "P-1001", "Plan", "FY2026 Lubricants Europe"},
		{initiativeDescribe, "parent", helix.ID, "Campaign", "Helix launch"},
		{campaignDescribe, "attributes." + clusterID, []any{"SO-1", "SO-2"}, "Label " + clusterID, "Option SO-1, Option SO-2"},
	}
	err := f.db.Read(t.Context(), func(ctx context.Context) error { // history describes inside its Write
		for _, c := range cases {
			label, text, err := c.describe(ctx, c.field, c.value)
			if err != nil || label != c.label || text != c.text {
				t.Errorf("Describe(%s, %v) = %q, %q, %v; want %q, %q", c.field, c.value, label, text, err, c.label, c.text)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Deleting a campaign or an initiative that no longer exists answers its "no longer exists" message.
func TestDelete_Gone_NotFound(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.DeleteCampaign(t.Context(), priya, "C-9999")
	assertError(t, err, errs.NotFound(campaignGoneMessage))

	_, err = f.service.DeleteInitiative(t.Context(), priya, "I-9999")
	assertError(t, err, errs.NotFound(initiativeGoneMessage))
}
