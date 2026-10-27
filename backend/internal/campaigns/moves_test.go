package campaigns

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// Services this package provides to other features (provide.go): plan roll-ups for feature 004 and
// initiative facts for feature 006.

// PlanRollups counts what each plan holds below it, adds up the campaign budgets, the planned costs and the
// goals with the same view builder as the plan tree, and lists the campaigns and initiatives that block
// publishing, in tree order. A plan without campaigns, or one that is gone, gets zeros (feature 004 FR-024).
func TestPlanRollups_CountsMoneyAndPublishProblems_For004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	complete := settings.AttributeValues{clusterID: json.RawMessage(`["SO-1"]`), regionID: json.RawMessage(`["SO-9"]`)}
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{
		Name: "Helix launch", OwnerID: ptr("U-1002"), StartDate: day("2026-02-01"), EndDate: day("2026-11-30"),
		Budget: input("300000.00"), Attributes: complete,
	})
	rimula := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Rimula fleet", Budget: input("200000.50")})
	digital := f.createInitiative(t, priya, helix.ID, InitiativeInput{
		Name: "Digital & social", OwnerID: ptr("U-1003"), StartDate: day("2026-03-01"), EndDate: day("2026-06-30"),
	})
	printOutdoor := f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Print & outdoor"})
	f.createInitiative(t, priya, rimula.ID, InitiativeInput{
		Name: "Retail events", OwnerID: ptr("U-1004"), StartDate: day("2026-04-01"), EndDate: day("2026-04-30"),
	})
	f.activities.add(
		ActivitySummary{ID: "A-1", InitiativeID: digital.ID, Name: "Launch film", PlannedCost: usd("1000.00"), Approved: true,
			LineItems: 2, Incomplete: true, IncompleteLineItems: []errs.RecordRef{{Type: "lineItem", ID: "L-1", Name: "Agency fee"}}},
		ActivitySummary{ID: "A-2", InitiativeID: printOutdoor.ID, Name: "Billboards", INQ: ptr(int64(5)), Revenue: usd("2500.00")},
	)

	var rollups map[string]PlanRollup
	err := f.db.Read(t.Context(), func(ctx context.Context) error { // joins the caller's transaction
		var err error
		rollups, err = f.service.PlanRollups(ctx, []string{"P-1001", "P-1002", "P-9999"})
		return err
	})
	if err != nil {
		t.Fatalf("PlanRollups: %v", err)
	}

	got := rollups["P-1001"]
	if got.Campaigns != 2 || got.Initiatives != 3 || got.Activities != 2 || got.ApprovedActivities != 1 || got.LineItems != 2 {
		t.Errorf("counts = %d campaigns, %d initiatives, %d activities, %d approved, %d line items; want 2, 3, 2, 1, 2",
			got.Campaigns, got.Initiatives, got.Activities, got.ApprovedActivities, got.LineItems)
	}
	if got.AllocatedToCampaigns.Amount.String() != "500000.50" || got.AllocatedToCampaigns.Currency != "USD" {
		t.Errorf("AllocatedToCampaigns = %+v, want 500000.50 USD", got.AllocatedToCampaigns)
	}
	if amountText(got.PlannedCost) != "1000.00" {
		t.Errorf("PlannedCost = %q, want 1000.00", amountText(got.PlannedCost))
	}
	if got.Goals.INQ == nil || *got.Goals.INQ != 5 || got.Goals.MQL != nil || amountText(got.Goals.Revenue) != "2500.00" {
		t.Errorf("Goals = %+v, want INQ 5, no MQL, revenue 2500.00", got.Goals)
	}
	// Rimula fleet (no owner, dates, cluster or region), Print & outdoor (no owner or dates), the incomplete
	// Launch film and its incomplete line item.
	if got.ItemsNeedingAttention != 4 {
		t.Errorf("ItemsNeedingAttention = %d, want 4", got.ItemsNeedingAttention)
	}
	wantProblems := []RecordProblems{
		{Record: errs.RecordRef{Type: "initiative", ID: printOutdoor.ID, Name: "Print & outdoor"}, Missing: []settings.MissingField{
			{Field: "ownerId", Label: "Owner"}, {Field: "startDate", Label: "Dates"},
		}},
		{Record: errs.RecordRef{Type: "campaign", ID: rimula.ID, Name: "Rimula fleet"}, Missing: []settings.MissingField{
			{Field: "ownerId", Label: "Owner"}, {Field: "startDate", Label: "Dates"},
			{Field: "attributes." + clusterID, Label: "Cluster"}, {Field: "attributes." + regionID, Label: "Region"},
		}},
	}
	if !reflect.DeepEqual(got.PublishProblems, wantProblems) {
		t.Errorf("PublishProblems =\n%+v\nwant\n%+v", got.PublishProblems, wantProblems)
	}

	for _, planID := range []string{"P-1002", "P-9999"} {
		empty := rollups[planID]
		if empty.Campaigns != 0 || empty.Initiatives != 0 || empty.AllocatedToCampaigns.Amount.String() != "0.00" ||
			empty.PlannedCost != nil || empty.ItemsNeedingAttention != 0 || empty.PublishProblems == nil || len(empty.PublishProblems) != 0 {
			t.Errorf("rollup of %s = %+v, want zeros", planID, empty)
		}
	}
}

// InitiativeFacts gives feature 006 an initiative with its dates, owner, campaign and plan (R3 and the
// defaults of a new activity); InitiativesOfPlan lists every initiative of a plan in tree order, campaign by
// campaign (move targets grouped by campaign).
func TestInitiativeFacts_For006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	rimula := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Rimula fleet", StartDate: day("2026-05-01"), EndDate: day("2026-12-31")})
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", StartDate: day("2026-02-01"), EndDate: day("2026-11-30")})
	f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "Retail growth"})
	events := f.createInitiative(t, priya, rimula.ID, InitiativeInput{Name: "Retail events"})
	printOutdoor := f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Print & outdoor", StartDate: day("2026-06-01"), EndDate: day("2026-08-31")})
	digital := f.createInitiative(t, priya, helix.ID, InitiativeInput{
		Name: "Digital & social", OwnerID: ptr("U-1003"), StartDate: day("2026-03-01"), EndDate: day("2026-06-30"),
	})

	facts, found, err := f.service.InitiativeFacts(t.Context(), digital.ID)

	want := InitiativeFacts{
		ID: digital.ID, Name: "Digital & social", OwnerID: ptr("U-1003"), StartDate: day("2026-03-01"), EndDate: day("2026-06-30"),
		CampaignID: helix.ID, CampaignName: "Helix launch", PlanID: "P-1001",
	}
	if err != nil || !found || !reflect.DeepEqual(facts, want) {
		t.Errorf("InitiativeFacts = %+v, %v, %v; want %+v", facts, found, err, want)
	}
	if _, found, err := f.service.InitiativeFacts(t.Context(), "I-9999"); found || err != nil {
		t.Errorf("InitiativeFacts(I-9999) = %v, %v; want not found", found, err)
	}

	list, err := f.service.InitiativesOfPlan(t.Context(), "P-1001")
	if err != nil {
		t.Fatalf("InitiativesOfPlan: %v", err)
	}
	var ids []string
	for _, i := range list {
		ids = append(ids, i.ID+" "+i.CampaignName)
	}
	wantIDs := []string{digital.ID + " Helix launch", printOutdoor.ID + " Helix launch", events.ID + " Rimula fleet"}
	if !reflect.DeepEqual(ids, wantIDs) {
		t.Errorf("InitiativesOfPlan = %v, want %v", ids, wantIDs)
	}
	if empty, err := f.service.InitiativesOfPlan(t.Context(), "P-1003"); err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("InitiativesOfPlan(P-1003) = %v, %v; want an empty list", empty, err)
	}
}

// CopyPlanContents copies a plan's campaigns and initiatives for feature 004's plan copy: the same names
// and values, every date moved by the year shift (29 February → 28 February), new IDs, the copier as
// creator and no log entries; it returns the old → new IDs for the levels below.
func TestCopyPlanContents_ShiftsDatesKeepsNames_For004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	brand := f.createCampaign(t, priya, "P-1003", CampaignInput{
		Name: "Brand platform", OwnerID: ptr("U-1001"), StartDate: day("2027-01-01"), EndDate: day("2028-02-29"),
		Budget: input("300000.00"), Attributes: settings.AttributeValues{clusterID: json.RawMessage(`["SO-1"]`)},
	})
	film := f.createInitiative(t, priya, brand.ID, InitiativeInput{
		Name: "Brand film", OwnerID: ptr("U-1001"), StartDate: day("2027-01-01"), EndDate: day("2027-06-30"), Budget: input("250000.00"),
	})
	undated := f.createCampaign(t, priya, "P-1003", CampaignInput{Name: "Undated"})
	f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "In another plan"})
	entriesBefore := len(f.log(t, "", ""))

	var counts CopyCounts
	var newIDs map[string]string
	err := f.db.Write(t.Context(), func(ctx context.Context) error { // joins the plan copy's Write
		var err error
		counts, newIDs, err = f.service.CopyPlanContents(ctx, daniel, "P-1003", "P-1002", 1)
		return err
	})

	if err != nil || counts != (CopyCounts{Campaigns: 2, Initiatives: 1}) || len(newIDs) != 3 {
		t.Fatalf("CopyPlanContents = %+v, %v, %v; want 2 campaigns, 1 initiative and their 3 new IDs", counts, newIDs, err)
	}
	brandCopy, _ := f.storedCampaign(t, newIDs[brand.ID])
	original, _ := f.storedCampaign(t, brand.ID)
	want := original
	want.ID, want.PlanID, want.Meta = newIDs[brand.ID], "P-1002", record.NewMeta(daniel, start)
	want.StartDate, want.EndDate = day("2028-01-01"), day("2029-02-28")
	if !reflect.DeepEqual(brandCopy, want) {
		t.Errorf("campaign copy =\n%+v\nwant\n%+v", brandCopy, want)
	}
	filmCopy, _ := f.storedInitiative(t, newIDs[film.ID])
	if filmCopy.Name != "Brand film" || filmCopy.CampaignID != brandCopy.ID || filmCopy.StartDate.String() != "2028-01-01" ||
		filmCopy.EndDate.String() != "2028-06-30" || amountText(filmCopy.Budget) != "250000.00" || filmCopy.CreatedBy.ID != daniel.ID {
		t.Errorf("initiative copy = %+v, want Brand film in %s, 2028-01-01 – 2028-06-30, 250000.00, by Daniel", filmCopy, brandCopy.ID)
	}
	if undatedCopy, _ := f.storedCampaign(t, newIDs[undated.ID]); undatedCopy.Name != "Undated" || undatedCopy.StartDate != nil {
		t.Errorf("undated copy = %+v, want Undated without dates", undatedCopy)
	}
	if got := len(f.log(t, "", "")); got != entriesBefore {
		t.Errorf("log entries = %d, want %d: the plan copy writes the one entry", got, entriesBefore)
	}
}
