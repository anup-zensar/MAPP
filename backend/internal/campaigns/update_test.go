package campaigns

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// US3 — View and edit a campaign or initiative.

// updateOf returns the update that saves campaign c's form unchanged, from version.
func updateOf(c CampaignView, version int) CampaignUpdate {
	var budget *money.Input
	if c.Budget != nil {
		budget = input(c.Budget.Amount.String())
	}
	return CampaignUpdate{
		Name: c.Name, Description: c.Description, OwnerID: c.OwnerID, StartDate: c.StartDate, EndDate: c.EndDate,
		Budget: budget, Attributes: c.Attributes, Version: &version,
	}
}

// initiativeUpdateOf returns the update that saves initiative i's form unchanged, from version.
func initiativeUpdateOf(i InitiativeView, version int) InitiativeUpdate {
	var budget *money.Input
	if i.Budget != nil {
		budget = input(i.Budget.Amount.String())
	}
	return InitiativeUpdate{
		Name: i.Name, Description: i.Description, OwnerID: i.OwnerID, StartDate: i.StartDate, EndDate: i.EndDate,
		Budget: budget, Attributes: i.Attributes, Version: &version,
	}
}

// putStored writes records straight into the store, for states the service would refuse to create.
func (f *fixture) putStored(t *testing.T, campaigns []Campaign, initiatives []Initiative) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		repo := NewJSONStore()
		for _, c := range campaigns {
			if err := repo.PutCampaign(ctx, c); err != nil {
				return err
			}
		}
		for _, i := range initiatives {
			if err := repo.PutInitiative(ctx, i); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
}

// An update without a version, or with one below 1, is refused next to "version", together with every
// other problem; a missing record answers 404 first (FR-018).
func TestUpdateCampaign_MissingVersion_FR018(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})

	noVersion := updateOf(c, 1)
	noVersion.Version = nil
	_, err := f.service.UpdateCampaign(t.Context(), priya, c.ID, noVersion)
	assertError(t, err, problems("version", "Reload the page and try again."))

	zero := updateOf(c, 0)
	zero.Name = ""
	_, err = f.service.UpdateCampaign(t.Context(), priya, c.ID, zero)
	assertError(t, err, problems("name", "Enter a name.", "version", "Reload the page and try again."))

	_, err = f.service.UpdateCampaign(t.Context(), priya, "C-9999", noVersion)
	assertError(t, err, errs.NotFound("This campaign no longer exists — someone may have deleted it."))
	_, err = f.service.UpdateInitiative(t.Context(), priya, "I-9999", InitiativeUpdate{Name: "x"})
	assertError(t, err, errs.NotFound("This initiative no longer exists — someone may have deleted it."))
}

// A save from an out-of-date copy is refused with version_conflict naming who changed the record and when;
// nothing is saved (FR-018, D-030).
func TestUpdateCampaign_VersionConflict_FR018(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	f.advance(time.Hour)
	renamed := updateOf(c, 1)
	renamed.Name = "Helix launch 2026"
	if _, err := f.service.UpdateCampaign(t.Context(), daniel, c.ID, renamed); err != nil {
		t.Fatalf("first save: %v", err)
	}

	stale := updateOf(c, 1)
	stale.Name = ""
	_, err := f.service.UpdateCampaign(t.Context(), priya, c.ID, stale)

	assertError(t, err, errs.VersionConflict("campaign", "Daniel Okoro", start.Add(time.Hour)))
	stored, _ := f.storedCampaign(t, c.ID)
	if stored.Name != "Helix launch 2026" || stored.Version != 2 {
		t.Errorf("stored = %+v", stored)
	}
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social"})
	_, err = f.service.UpdateInitiative(t.Context(), priya, i.ID, initiativeUpdateOf(i, 7))
	assertError(t, err, errs.VersionConflict("initiative", "Priya Raman", start.Add(time.Hour)))
}

// A save that changes nothing — also with the name's surrounding spaces, a budget written without decimals
// or attribute values written with spaces — answers the campaign as it is: no new version and no log entry
// (FR-030).
func TestUpdateCampaign_NoChangeNoVersionNoLog_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{
		Name: "Helix launch", OwnerID: ptr("U-1002"), Budget: input("700000.00"),
		Attributes: settings.AttributeValues{clusterID: json.RawMessage(`["SO-1003"]`)},
	})
	same := updateOf(c, 1)
	same.Name = "  Helix launch "
	same.Budget = input("700000")
	same.Attributes = settings.AttributeValues{clusterID: json.RawMessage(`[ "SO-1003" ]`)}
	f.advance(time.Hour)

	view, err := f.service.UpdateCampaign(t.Context(), daniel, c.ID, same)

	if err != nil || view.Version != 1 || view.UpdatedBy != priya {
		t.Errorf("UpdateCampaign = version %d by %v, %v; want version 1 by Priya", view.Version, view.UpdatedBy, err)
	}
	if entries := f.log(t, campaignKind, c.ID); len(entries) != 1 {
		t.Errorf("%d log entries, want only the created one", len(entries))
	}
}

// A saved change gets a new version and one "changed" entry listing every changed field with readable
// texts: the owner's name, money as "400,000.00 USD", dates as "01 Mar 2026", and one change per attribute
// (FR-030, US3 scenario 2).
func TestUpdateCampaign_LogsEachChangedFieldWithTexts_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{
		Name: "Helix launch", OwnerID: ptr("U-1002"), StartDate: day("2026-01-01"), EndDate: day("2026-12-31"),
		Budget: input("300000.00"), Attributes: settings.AttributeValues{clusterID: json.RawMessage(`["SO-1"]`)},
	})
	update := updateOf(c, 1)
	update.OwnerID = ptr("U-1003")
	update.Budget = input("400000.00")
	update.StartDate = day("2026-03-01")
	update.Attributes = settings.AttributeValues{clusterID: json.RawMessage(`["SO-2"]`), regionID: json.RawMessage(`["SO-9"]`)}
	f.advance(time.Hour)

	view, err := f.service.UpdateCampaign(t.Context(), daniel, c.ID, update)

	if err != nil || view.Version != 2 || view.UpdatedBy != daniel || !view.UpdatedAt.Equal(start.Add(time.Hour)) || view.Owner.Name != "Daniel Okoro" {
		t.Fatalf("UpdateCampaign = %+v, %v", view, err)
	}
	entries := f.log(t, campaignKind, c.ID)
	if len(entries) != 2 || entries[0].Action != history.ActionChanged || entries[0].Actor != daniel {
		t.Fatalf("entries = %+v, want created and changed", entries)
	}
	type change struct{ Field, Label, OldText, NewText string }
	var got []change
	for _, ch := range entries[0].Changes {
		got = append(got, change{ch.Field, ch.Label, ch.OldText, ch.NewText})
	}
	want := []change{
		{"budget", "Budget", "300,000.00 USD", "400,000.00 USD"},
		{"ownerId", "Owner", "Priya Raman", "Daniel Okoro"},
		{"startDate", "Start date", "01 Jan 2026", "01 Mar 2026"},
		{"attributes." + clusterID, "Label " + clusterID, "Option SO-1", "Option SO-2"},
		{"attributes." + regionID, "Label " + regionID, "Not filled in", "Option SO-9"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("changes =\n%+v\nwant\n%+v", got, want)
	}
}

// New campaign dates that would leave initiatives outside are refused, naming each initiative with its
// dates; nothing is saved (FR-007, US3 scenario 3).
func TestUpdateCampaign_DatesLeavingInitiativesOutside_Refused_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", StartDate: day("2026-01-01"), EndDate: day("2026-12-31")})
	f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social", StartDate: day("2026-02-01"), EndDate: day("2026-11-30")})
	f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Print & outdoor", StartDate: day("2026-03-01"), EndDate: day("2026-12-31")})
	f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Undated"})

	narrower := updateOf(c, 1)
	narrower.StartDate = day("2026-03-01")
	_, err := f.service.UpdateCampaign(t.Context(), priya, c.ID, narrower)
	assertError(t, err, problems("startDate",
		"Digital & social (01 Feb – 30 Nov 2026) would be outside these dates. Change its dates first, or choose dates that include it."))

	narrowest := updateOf(c, 1)
	narrowest.StartDate, narrowest.EndDate = day("2026-04-01"), day("2026-10-31")
	_, err = f.service.UpdateCampaign(t.Context(), priya, c.ID, narrowest)
	assertError(t, err, problems("startDate",
		"These initiatives would be outside these dates: Digital & social (01 Feb – 30 Nov 2026), Print & outdoor (01 Mar – 31 Dec 2026). "+
			"Change their dates first, or choose dates that include them."))

	if stored, _ := f.storedCampaign(t, c.ID); *stored.StartDate != *day("2026-01-01") || stored.Version != 1 {
		t.Errorf("stored = %+v, want it unchanged", stored)
	}
	cleared := updateOf(c, 1)
	cleared.StartDate, cleared.EndDate = nil, nil
	if _, err := f.service.UpdateCampaign(t.Context(), priya, c.ID, cleared); err != nil {
		t.Errorf("clearing the dates: %v, want it allowed (no dates limit nothing)", err)
	}
}

// The date rules run only when the dates being saved change: a campaign that already has an initiative
// outside its dates can still have other fields edited, and so can that initiative (D-095).
func TestUpdateCampaign_UnchangedDatesNeverRefused_DecisionF(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	meta := record.NewMeta(priya, start)
	c := Campaign{ID: "C-1001", PlanID: "P-1001", Name: "Helix launch", StartDate: day("2026-03-01"), EndDate: day("2026-12-31"),
		Attributes: settings.AttributeValues{}, Meta: meta}
	i := Initiative{ID: "I-1001", CampaignID: c.ID, Name: "Digital & social", StartDate: day("2026-02-01"), EndDate: day("2026-11-30"),
		Attributes: settings.AttributeValues{}, Meta: meta}
	f.putStored(t, []Campaign{c}, []Initiative{i})
	view, _ := f.service.GetCampaign(t.Context(), priya, c.ID)

	renamed := updateOf(view, 1)
	renamed.Name = "Helix launch 2026"
	if _, err := f.service.UpdateCampaign(t.Context(), priya, c.ID, renamed); err != nil {
		t.Errorf("campaign rename: %v, want it saved", err)
	}
	initiative, _ := f.service.GetInitiative(t.Context(), priya, i.ID)
	budget := initiativeUpdateOf(initiative, 1)
	budget.Budget = input("1000.00")
	if _, err := f.service.UpdateInitiative(t.Context(), priya, i.ID, budget); err != nil {
		t.Errorf("initiative budget: %v, want it saved", err)
	}
}

// Editing is allowed whatever the plan's status: P-1001 is published (FR-018).
func TestUpdateCampaign_AllowedInPublishedPlan_FR018(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	update := updateOf(c, 1)
	update.Description = "Premium launch"

	view, err := f.service.UpdateCampaign(t.Context(), priya, c.ID, update)

	if err != nil || view.Description != "Premium launch" || view.Version != 2 {
		t.Errorf("UpdateCampaign = %+v, %v", view, err)
	}
}

// An owner who has become inactive stays when the form is saved with them; only a newly chosen owner must
// be active (spec edge case, FR-002).
func TestUpdateCampaign_InactiveOwnerKept_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := Campaign{ID: "C-1001", PlanID: "P-1001", Name: "Helix launch", OwnerID: ptr(sam.ID), Attributes: settings.AttributeValues{},
		Meta: record.NewMeta(priya, start)}
	f.putStored(t, []Campaign{c}, nil)
	view, _ := f.service.GetCampaign(t.Context(), priya, c.ID)
	if view.Owner == nil || view.Owner.Active {
		t.Fatalf("owner = %+v, want Sam flagged inactive", view.Owner)
	}

	update := updateOf(view, 1)
	update.Description = "Still the same owner"
	if _, err := f.service.UpdateCampaign(t.Context(), priya, c.ID, update); err != nil {
		t.Errorf("keeping the inactive owner: %v, want it saved", err)
	}
	other := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Rimula fleet"})
	toSam := updateOf(other, 1)
	toSam.OwnerID = ptr(sam.ID)
	_, err := f.service.UpdateCampaign(t.Context(), priya, other.ID, toSam)
	assertError(t, err, problems("ownerId", "Choose an active person."))
}

// New initiative dates that would leave activities outside are refused, naming each; an ongoing activity
// has only its start checked, an undated one is ignored (FR-007).
func TestUpdateInitiative_DatesLeavingActivitiesOutside_Refused_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social", StartDate: day("2026-01-01"), EndDate: day("2026-12-31")})
	f.activities.add(
		ActivitySummary{ID: "A-1001", InitiativeID: i.ID, Name: "Launch film", StartDate: day("2026-03-01"), EndDate: day("2026-06-30")},
		ActivitySummary{ID: "A-1002", InitiativeID: i.ID, Name: "Always-on search", StartDate: day("2026-02-15"), Ongoing: true},
		ActivitySummary{ID: "A-1003", InitiativeID: i.ID, Name: "Undated idea"},
	)

	from := initiativeUpdateOf(i, 1)
	from.StartDate = day("2026-03-01")
	_, err := f.service.UpdateInitiative(t.Context(), priya, i.ID, from)
	assertError(t, err, problems("startDate",
		"Always-on search (ongoing from 15 Feb 2026) would be outside these dates. Change its dates first, or choose dates that include it."))

	narrow := initiativeUpdateOf(i, 1)
	narrow.StartDate, narrow.EndDate = day("2026-04-01"), day("2026-05-31")
	_, err = f.service.UpdateInitiative(t.Context(), priya, i.ID, narrow)
	assertError(t, err, problems("startDate",
		"These activities would be outside these dates: Launch film (01 Mar – 30 Jun 2026), Always-on search (ongoing from 15 Feb 2026). "+
			"Change their dates first, or choose dates that include them."))

	fits := initiativeUpdateOf(i, 1)
	fits.StartDate, fits.EndDate = day("2026-02-01"), day("2026-06-30")
	if _, err := f.service.UpdateInitiative(t.Context(), priya, i.ID, fits); err != nil {
		t.Errorf("dates that hold every activity: %v, want them saved", err)
	}
}

// New initiative dates outside the campaign's are refused (FR-006, R3).
func TestUpdateInitiative_OutsideCampaign_Refused_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", StartDate: day("2026-03-01"), EndDate: day("2026-12-31")})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social", StartDate: day("2026-03-01"), EndDate: day("2026-11-30")})
	update := initiativeUpdateOf(i, 1)
	update.StartDate = day("2026-02-01")

	_, err := f.service.UpdateInitiative(t.Context(), priya, i.ID, update)

	assertError(t, err, problems("startDate", "Dates must be inside the campaign's dates (01 Mar 2026 – 31 Dec 2026)."))
}

// A saved initiative change gets a new version and one "changed" entry; its name stays unique in the
// campaign (FR-005, FR-030).
func TestUpdateInitiative_SavedAndLogged_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Print & outdoor"})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social"})

	taken := initiativeUpdateOf(i, 1)
	taken.Name = "print & OUTDOOR"
	_, err := f.service.UpdateInitiative(t.Context(), priya, i.ID, taken)
	assertError(t, err, problems("name", "'Helix launch' already has an initiative called 'Print & outdoor'."))

	update := initiativeUpdateOf(i, 1)
	update.Budget = input("125000.50")
	view, err := f.service.UpdateInitiative(t.Context(), daniel, i.ID, update)
	if err != nil || view.Version != 2 || amountText(view.Budget) != "125000.50" {
		t.Fatalf("UpdateInitiative = %+v, %v", view, err)
	}
	entries := f.log(t, initiativeKind, i.ID)
	if len(entries) != 2 || len(entries[0].Changes) != 1 || entries[0].Changes[0].NewText != "125,000.50 USD" {
		t.Errorf("entries = %+v", entries)
	}
}

// Nobody's changes are restricted by role: Daniel and Mei Lin change records Priya created (FR-031, D-004).
func TestNoRoleChecks_AnyoneChangesAnyRecord_FR031(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social"})

	campaignUpdate := updateOf(c, 1)
	campaignUpdate.Name = "Helix launch 2026"
	if _, err := f.service.UpdateCampaign(t.Context(), daniel, c.ID, campaignUpdate); err != nil {
		t.Errorf("Daniel's change: %v", err)
	}
	initiativeUpdate := initiativeUpdateOf(i, 1)
	initiativeUpdate.Description = "Social first"
	if _, err := f.service.UpdateInitiative(t.Context(), meiLin, i.ID, initiativeUpdate); err != nil {
		t.Errorf("Mei Lin's change: %v", err)
	}
	if _, err := f.service.CreateInitiative(t.Context(), alex, c.ID, InitiativeInput{Name: "Events"}); err != nil {
		t.Errorf("Alex's initiative: %v", err)
	}
}
