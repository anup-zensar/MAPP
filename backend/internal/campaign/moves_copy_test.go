package campaign

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/settings"
)

// User story 6 — Move or copy a campaign or initiative (moves.go).

// demo is the shape of the plan these tests move and copy in, P-1001: Helix launch (C-1001, 1 Feb – 30 Nov
// 2026, 400,000.00) with Digital & social (I-1001, 1 Mar – 30 Jun 2026, 100,000.00, three activities) and
// Print & outdoor (I-1002, no dates, no budget); Rimula fleet (C-1002, 1 Feb – 31 Dec 2026, 300,000.00)
// with Retail events (I-1003, June, 250,000.00).
type demo struct {
	helix, rimula                CampaignView
	digital, print, retailEvents InitiativeView
}

// newDemo creates the demo shape.
func (f *fixture) newDemo(t *testing.T) demo {
	t.Helper()
	var d demo
	d.helix = f.createCampaign(t, priya, "P-1001", CampaignInput{
		Name: "Helix launch", Description: "Helix Ultra launch.", OwnerID: ptr(priya.ID), StartDate: day("2026-02-01"),
		EndDate: day("2026-11-30"), Budget: input("400000.00"),
		Attributes: settings.AttributeValues{clusterID: json.RawMessage(`["SO-1"]`), regionID: json.RawMessage(`["SO-9"]`)},
	})
	d.rimula = f.createCampaign(t, priya, "P-1001", CampaignInput{
		Name: "Rimula fleet", OwnerID: ptr(daniel.ID), StartDate: day("2026-02-01"), EndDate: day("2026-12-31"), Budget: input("300000.00"),
	})
	d.digital = f.createInitiative(t, priya, d.helix.ID, InitiativeInput{
		Name: "Digital & social", OwnerID: ptr(daniel.ID), StartDate: day("2026-03-01"), EndDate: day("2026-06-30"),
		Budget: input("100000.00"), Attributes: settings.AttributeValues{fit4ID: json.RawMessage(`true`)},
	})
	d.print = f.createInitiative(t, priya, d.helix.ID, InitiativeInput{Name: "Print & outdoor"})
	d.retailEvents = f.createInitiative(t, priya, d.rimula.ID, InitiativeInput{
		Name: "Retail events", StartDate: day("2026-06-01"), EndDate: day("2026-06-30"), Budget: input("250000.00"),
	})
	f.activities.add(
		ActivitySummary{ID: "A-1", InitiativeID: d.digital.ID, Name: "Launch film", TypeID: "AT-1001", PlannedCost: usd("1000.00"),
			Approved: true, LineItems: 2},
		ActivitySummary{ID: "A-2", InitiativeID: d.digital.ID, Name: "Roadshow", TypeID: "AT-1002"},
		ActivitySummary{ID: "A-3", InitiativeID: d.digital.ID, Name: "Untyped"},
	)
	return d
}

// destinations returns the destinations of the campaign or initiative id for purpose, as Priya.
func (f *fixture) destinations(t *testing.T, level, id, purpose string) DestinationList {
	t.Helper()
	var list DestinationList
	var err error
	if level == campaignKind {
		list, err = f.service.CampaignDestinations(t.Context(), priya, id, purpose)
	} else {
		list, err = f.service.InitiativeDestinations(t.Context(), priya, id, purpose)
	}
	if err != nil {
		t.Fatalf("destinations of %s for %s: %v", id, purpose, err)
	}
	return list
}

// summaries writes each destination as "<name>: <reason or available>[ (current)][, suggests <name>]".
func summaries(list DestinationList) []string {
	texts := []string{}
	for _, d := range list.Items {
		text := d.Name + ": available"
		if d.UnavailableReason != nil {
			text = d.Name + ": " + *d.UnavailableReason
		}
		if d.Current {
			text += " (current)"
		}
		if d.SuggestedName != nil {
			text += ", suggests " + *d.SuggestedName
		}
		texts = append(texts, text)
	}
	return texts
}

// A moved initiative keeps its ID, dates, values and everything below it; the roll-ups of both campaigns
// follow, a negative remaining budget included (FR-025, US6 scenario 1).
func TestMoveInitiative_KeepsIDDatesValuesAndChildren_FR025(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.newDemo(t)
	f.advance(time.Hour)

	moved, err := f.service.MoveInitiative(t.Context(), daniel, d.digital.ID, InitiativeMove{CampaignID: d.rimula.ID, Version: ptr(1)})

	if err != nil {
		t.Fatalf("MoveInitiative: %v", err)
	}
	if stored, _ := f.storedInitiative(t, d.digital.ID); !movedWithItsValues(stored, d.rimula.ID) {
		t.Errorf("stored = %+v, want it in Rimula fleet with its values, version 2, changed by Daniel", stored)
	}
	if moved.Campaign.Name != "Rimula fleet" || moved.Counts.Activities != 3 || moved.Counts.LineItems != 2 {
		t.Errorf("view = in %q with %+v, want Rimula fleet with its 3 activities", moved.Campaign.Name, moved.Counts)
	}
	for _, c := range f.list(t, "P-1001").Items {
		if !rolledUpAfterMove(c, d) {
			t.Errorf("%s = %+v %+v", c.Name, c.Counts, c.BudgetAndCost)
		}
	}
}

func movedWithItsValues(stored Initiative, campaignID string) bool {
	return stored.CampaignID == campaignID && stored.Name == "Digital & social" && stored.StartDate.String() == "2026-03-01" &&
		amountText(stored.Budget) == "100000.00" && string(stored.Attributes[fit4ID]) == "true" && stored.Version == 2 &&
		stored.UpdatedBy.ID == daniel.ID
}

// rolledUpAfterMove passes every campaign other than Helix launch and Rimula fleet.
func rolledUpAfterMove(c CampaignView, d demo) bool {
	switch c.ID {
	case d.helix.ID:
		return c.Counts.Initiatives == 1 && c.Counts.Activities == 0 && amountText(c.BudgetAndCost.RemainingForChildren) == "400000.00"
	case d.rimula.ID:
		return c.Counts.Initiatives == 2 && c.Counts.Activities == 3 && amountText(c.BudgetAndCost.OverAllocatedBy) == "50000.00"
	}
	return true
}

// A move to a campaign that is gone, of another plan, or the initiative's own is refused with its reason;
// one that already has the name and whose dates do not fit is refused with both reasons at once, and a
// missing version with the others (FR-023).
func TestMoveInitiative_RefusedWithEveryReason_FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.newDemo(t)
	late := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Late launch", StartDate: day("2026-06-01"), EndDate: day("2026-12-31")})
	f.createInitiative(t, priya, late.ID, InitiativeInput{Name: "DIGITAL & social"})
	other := f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "Retail growth"})
	move := func(campaignID string, version *int) error {
		_, err := f.service.MoveInitiative(t.Context(), priya, d.digital.ID, InitiativeMove{CampaignID: campaignID, Version: version})
		return err
	}

	assertError(t, move("C-9999", ptr(1)), problems("campaignId", "That campaign no longer exists — choose another one."))
	assertError(t, move(other.ID, ptr(1)), problems("campaignId", "Initiatives move only between campaigns of the same plan."))
	assertError(t, move(d.helix.ID, ptr(1)), problems("campaignId", "'Digital & social' is already in 'Helix launch'."))
	assertError(t, move(late.ID, nil), problems(
		"campaignId", "'Late launch' already has an initiative called 'DIGITAL & social' — rename it first.",
		"campaignId", "Dates don't fit (01 Jun – 31 Dec 2026) — change the initiative's dates first.",
		"version", "Reload the page and try again."))
	if stored, _ := f.storedInitiative(t, d.digital.ID); stored.CampaignID != d.helix.ID || stored.Version != 1 {
		t.Errorf("stored = %+v, want it unchanged in Helix launch", stored)
	}
	_, err := f.service.MoveInitiative(t.Context(), priya, "I-9999", InitiativeMove{CampaignID: d.rimula.ID, Version: ptr(1)})
	assertError(t, err, errs.NotFound(initiativeGoneMessage))
}

// A campaign moves only to another plan of the same year that has no campaign with its name; it takes
// everything below it along (FR-023, FR-025, D-017).
func TestMoveCampaign_SameYearOnly_FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.newDemo(t)
	f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "helix launch"})
	f.plans.plans["P-1004"] = PlanFacts{ID: "P-1004", Name: "FY2026 Benelux", Year: 2026, Status: "draft", ModelID: "MOD-1001"}
	move := func(planID string) (CampaignView, error) {
		return f.service.MoveCampaign(t.Context(), priya, d.helix.ID, CampaignMove{PlanID: planID, Version: ptr(1)})
	}

	_, gone := move("P-9999")
	_, otherYear := move("P-1003")
	_, own := move("P-1001")
	_, taken := move("P-1002")
	moved, err := move("P-1004")

	assertError(t, gone, problems("planId", "That plan no longer exists — choose another one."))
	assertError(t, otherYear, problems("planId", "Campaigns move only between plans of the same year (2026)."))
	assertError(t, own, problems("planId", "'Helix launch' is already in 'FY2026 Lubricants Europe'."))
	assertError(t, taken, problems("planId", "'FY2026 Mobility China' already has a campaign called 'helix launch' — rename it first."))
	if err != nil || moved.PlanID != "P-1004" || moved.Plan.Name != "FY2026 Benelux" || moved.Counts.Initiatives != 2 ||
		moved.Counts.Activities != 3 || moved.Version != 2 {
		t.Fatalf("MoveCampaign = %+v, %v; want it in P-1004 with its 2 initiatives and 3 activities, version 2", moved, err)
	}
	if digital, _ := f.storedInitiative(t, d.digital.ID); digital.CampaignID != d.helix.ID || digital.Version != 1 {
		t.Errorf("initiative below = %+v, want it unchanged under the campaign", digital)
	}
}

// A campaign whose dates leave its plan's year keeps the R4 warning in the plan it moves to; its dates do
// not change (FR-023, R4).
func TestMoveCampaign_KeepsOutsideYearWarning_FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	winter := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Winter wave", StartDate: day("2026-12-01"), EndDate: day("2027-03-31")})

	moved, err := f.service.MoveCampaign(t.Context(), priya, winter.ID, CampaignMove{PlanID: "P-1002", Version: ptr(1)})

	if err != nil || !moved.OutsidePlanYear || moved.EndDate.String() != "2027-03-31" {
		t.Errorf("MoveCampaign = outside %v, ends %v, %v; want the warning and the same dates", moved.OutsidePlanYear, moved.EndDate, err)
	}
}

// A move writes one "moved" entry with the change of parent, both named, and the record's new place as its
// path (FR-030, data model §8).
func TestMove_LogsParentChange_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.newDemo(t)

	if _, err := f.service.MoveInitiative(t.Context(), daniel, d.digital.ID, InitiativeMove{CampaignID: d.rimula.ID, Version: ptr(1)}); err != nil {
		t.Fatalf("MoveInitiative: %v", err)
	}
	if _, err := f.service.MoveCampaign(t.Context(), daniel, d.rimula.ID, CampaignMove{PlanID: "P-1002", Version: ptr(1)}); err != nil {
		t.Fatalf("MoveCampaign: %v", err)
	}

	type move struct{ action, actor, label, oldText, newText, oldValue, path string }
	described := func(e history.Entry) move {
		c := e.Changes[0]
		var path []string
		for _, p := range e.Path {
			path = append(path, p.Name)
		}
		return move{string(e.Action), e.Actor.Name, c.Label, c.OldText, c.NewText, string(c.Old), strings.Join(path, " › ")}
	}
	initiativeEntries, campaignEntries := f.log(t, initiativeKind, d.digital.ID), f.log(t, campaignKind, d.rimula.ID)
	if got, want := described(initiativeEntries[0]), (move{"moved", "Daniel Okoro", "Campaign", "Helix launch", "Rimula fleet",
		`"` + d.helix.ID + `"`, "FY2026 Lubricants Europe › Rimula fleet"}); got != want || len(initiativeEntries[0].Changes) != 1 {
		t.Errorf("initiative entry = %+v\nwant %+v", got, want)
	}
	if got, want := described(campaignEntries[0]), (move{"moved", "Daniel Okoro", "Plan", "FY2026 Lubricants Europe",
		"FY2026 Mobility China", `"P-1001"`, "FY2026 Mobility China"}); got != want {
		t.Errorf("campaign entry = %+v\nwant %+v", got, want)
	}
}

// A move from an out-of-date version is refused with version_conflict, naming who changed the record and
// when; nothing moves (FR-025, D-030).
func TestMove_VersionConflict_FR025(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.newDemo(t)
	f.advance(time.Hour)
	edit := updateOf(d.helix, 1)
	edit.Description = "Edited."
	if _, err := f.service.UpdateCampaign(t.Context(), daniel, d.helix.ID, edit); err != nil {
		t.Fatalf("UpdateCampaign: %v", err)
	}

	_, campaignErr := f.service.MoveCampaign(t.Context(), priya, d.helix.ID, CampaignMove{PlanID: "P-1002", Version: ptr(1)})
	_, initiativeErr := f.service.MoveInitiative(t.Context(), priya, d.digital.ID, InitiativeMove{CampaignID: d.rimula.ID, Version: ptr(3)})

	assertError(t, campaignErr, errs.VersionConflict("campaign", "Daniel Okoro", start.Add(time.Hour)))
	assertError(t, initiativeErr, errs.VersionConflict("initiative", "Priya Raman", start))
	if stored, _ := f.storedCampaign(t, d.helix.ID); stored.PlanID != "P-1001" {
		t.Errorf("campaign moved to %s, want it still in P-1001", stored.PlanID)
	}
}
