package campaigns

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/record"
	"northstar/internal/settings"
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

// An initiative moves only to another campaign of its plan: one that already has an initiative with its
// name, or whose dates do not contain its dates, is listed but not available, with every reason and the fix;
// a campaign without dates fits any dates (FR-023, US6 scenario 2). They come by start date, then name.
func TestInitiativeDestinations_Move_UnavailableWithReasonAndFix_FR023(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	late := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Late launch", StartDate: day("2026-06-01"), EndDate: day("2026-12-31")})
	f.createInitiative(t, priya, late.ID, InitiativeInput{Name: "DIGITAL & social"})
	f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Undated"})
	f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "In another plan"})

	list := f.destinations(t, initiativeKind, d.digital.ID, PurposeMove)

	want := []string{
		"Rimula fleet: available",
		"Late launch: Already has an initiative called 'DIGITAL & social' — rename it first · " +
			"Dates don't fit (01 Jun – 31 Dec 2026) — change the initiative's dates first",
		"Undated: available",
	}
	if got := summaries(list); !reflect.DeepEqual(got, want) || list.Total != 3 {
		t.Errorf("destinations =\n%q\nwant\n%q", got, want)
	}
	if list.Rule != "Initiatives move only between campaigns of the same plan" {
		t.Errorf("rule = %q", list.Rule)
	}
	rimula := list.Items[0]
	if rimula.StartDate.String() != "2026-02-01" || rimula.EndDate.String() != "2026-12-31" || rimula.PlanStatus != nil ||
		rimula.Consequences != nil || amountText(rimula.RemainingAfter) != "-50000.00" || amountText(rimula.OverAllocatedBy) != "50000.00" {
		t.Errorf("Rimula fleet = %+v, want its dates and -50000.00 left (over by 50000.00)", rimula)
	}
}

// A copy of an initiative may go into its own campaign — first, as the default — or another of the plan;
// only dates that do not fit make one unavailable, and each suggests the first free copy name there
// (FR-026).
func TestInitiativeDestinations_Copy_CurrentFirstSuggestedNames_FR026(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	f.createInitiative(t, priya, d.helix.ID, InitiativeInput{Name: "Digital & social (copy)"})
	late := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Late launch", StartDate: day("2026-06-01"), EndDate: day("2026-12-31")})
	f.createInitiative(t, priya, late.ID, InitiativeInput{Name: "Digital & social"})

	list := f.destinations(t, initiativeKind, d.digital.ID, PurposeCopy)

	want := []string{
		"Helix launch: available (current), suggests Digital & social (copy 2)",
		"Rimula fleet: available, suggests Digital & social (copy)",
		"Late launch: Dates don't fit (01 Jun – 31 Dec 2026) — change the initiative's dates first, suggests Digital & social (copy)",
	}
	if got := summaries(list); !reflect.DeepEqual(got, want) {
		t.Errorf("destinations =\n%q\nwant\n%q", got, want)
	}
	if list.Rule != "A copy can go into this campaign or another campaign of the same plan" {
		t.Errorf("rule = %q", list.Rule)
	}
	// Helix launch: 400,000.00 − (100,000.00 + 0 + 0 + 100,000.00 for the copy)
	if helix := list.Items[0]; amountText(helix.RemainingAfter) != "200000.00" || helix.OverAllocatedBy != nil {
		t.Errorf("Helix launch remaining after = %s, want 200000.00", amountText(helix.RemainingAfter))
	}
}

// A campaign moves only to another plan of its plan's year, Draft or Published: other years are not
// offered, and a plan that already has a campaign with its name is listed but not available (FR-023, D-017).
func TestCampaignDestinations_Move_SameYearOnly_FR023(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	f.plans.plans["P-1004"] = PlanFacts{ID: "P-1004", Name: "FY2026 Benelux", Year: 2026, Status: "published", ModelID: "MOD-1001",
		Attributes: settings.AttributeValues{"SA-1001": json.RawMessage(`["SO-1002"]`)}}
	f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "HELIX launch"})

	list := f.destinations(t, campaignKind, d.helix.ID, PurposeMove)

	want := []string{
		"FY2026 Benelux: available",
		"FY2026 Mobility China: Already has a campaign called 'HELIX launch' — rename it first",
	}
	if got := summaries(list); !reflect.DeepEqual(got, want) {
		t.Fatalf("destinations =\n%q\nwant\n%q", got, want)
	}
	if list.Rule != "Campaigns move only between plans of the same year" || *list.Items[0].PlanStatus != "published" ||
		*list.Items[1].PlanStatus != "draft" || list.Items[0].StartDate != nil || list.Items[0].SuggestedName != nil {
		t.Errorf("rule %q, items %+v", list.Rule, list.Items)
	}
}

// A plan with another model or business unit lists what would change below the campaign before it moves:
// the destination's model, the activities whose type it does not allow (they keep their type), and the
// attribute groups hidden there — their values kept — or shown (FR-024, R7, R9). A plan with the same model
// and business unit has no consequences.
func TestCampaignDestinations_Consequences_FR024(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	f.plans.plans["P-1004"] = PlanFacts{ID: "P-1004", Name: "FY2026 Benelux", Year: 2026, Status: "draft", ModelID: "MOD-1001",
		Attributes: settings.AttributeValues{"SA-1001": json.RawMessage(`["SO-1002"]`)}}

	list := f.destinations(t, campaignKind, d.helix.ID, PurposeMove)

	if benelux := list.Items[0]; benelux.ID != "P-1004" || benelux.Consequences != nil {
		t.Errorf("same model and business unit: %+v, want no consequences", benelux)
	}
	want := &MoveConsequences{
		ModelName: "Mobility 1.0", ModelChanges: true, BusinessUnitChanges: true,
		TypesNotAllowed: []TypeNotAllowed{{ID: "A-1", Name: "Launch film", TypeName: "Content production"}},
		GroupsHidden:    []string{"Lubricants attributes (activities)", "Lubricants attributes (line items)"},
		GroupsShown:     []string{"Mobility attributes (activities)"},
	}
	if got := list.Items[1].Consequences; !reflect.DeepEqual(got, want) {
		t.Errorf("consequences =\n%+v\nwant\n%+v", got, want)
	}
}

// Every destination says what the record's budget would leave there; a copy that over-allocates its
// destination says by how much, and is still allowed (FR-026, D-015).
func TestDestinations_OverAllocationWarning_FR026(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "Retail growth", Budget: input("500000.00")})

	list := f.destinations(t, campaignKind, d.helix.ID, PurposeCopy)

	got := map[string][2]string{}
	for _, item := range list.Items {
		got[item.ID] = [2]string{amountText(item.RemainingAfter), amountText(item.OverAllocatedBy)}
	}
	want := map[string][2]string{
		"P-1001": {"100000.00", ""},           // 1,200,000.00 − (400,000.00 + 300,000.00 + 400,000.00)
		"P-1002": {"-100000.00", "100000.00"}, // 800,000.00 − (500,000.00 + 400,000.00)
	}
	if !reflect.DeepEqual(got, want) || !list.Items[1].Available {
		t.Errorf("remaining after = %v, want %v", got, want)
	}
}

// With nowhere to go, the list is empty, so the dialog can say so (edge case).
func TestDestinations_NoneAvailable_EmptyList_EdgeCase(t *testing.T) {
	f := newFixture(t)
	only := f.createCampaign(t, priya, "P-1003", CampaignInput{Name: "Brand platform"})
	alone := f.createInitiative(t, priya, only.ID, InitiativeInput{Name: "Brand film"})

	campaignList := f.destinations(t, campaignKind, only.ID, PurposeMove)
	initiativeList := f.destinations(t, initiativeKind, alone.ID, PurposeMove)

	if campaignList.Items == nil || len(campaignList.Items) != 0 || campaignList.Total != 0 ||
		initiativeList.Items == nil || len(initiativeList.Items) != 0 {
		t.Errorf("destinations = %+v and %+v, want empty lists", campaignList, initiativeList)
	}
}

// Destinations are only for a move or a copy, of a record that exists.
func TestDestinations_BadPurpose_Refused(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)

	_, campaignPurpose := f.service.CampaignDestinations(t.Context(), priya, d.helix.ID, "rename")
	_, initiativePurpose := f.service.InitiativeDestinations(t.Context(), priya, d.digital.ID, "")
	_, campaignGone := f.service.CampaignDestinations(t.Context(), priya, "C-9999", PurposeMove)
	_, initiativeGone := f.service.InitiativeDestinations(t.Context(), priya, "I-9999", PurposeCopy)

	assertError(t, campaignPurpose, problems("purpose", "Choose move or copy."))
	assertError(t, initiativePurpose, problems("purpose", "Choose move or copy."))
	assertError(t, campaignGone, errs.NotFound(campaignGoneMessage))
	assertError(t, initiativeGone, errs.NotFound(initiativeGoneMessage))
}

// A moved initiative keeps its ID, dates, values and everything below it; the roll-ups of both campaigns
// follow, a negative remaining budget included (FR-025, US6 scenario 1).
func TestMoveInitiative_KeepsIDDatesValuesAndChildren_FR025(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	f.advance(time.Hour)

	moved, err := f.service.MoveInitiative(t.Context(), daniel, d.digital.ID, InitiativeMove{CampaignID: d.rimula.ID, Version: ptr(1)})

	if err != nil {
		t.Fatalf("MoveInitiative: %v", err)
	}
	stored, _ := f.storedInitiative(t, d.digital.ID)
	if stored.CampaignID != d.rimula.ID || stored.Name != "Digital & social" || stored.StartDate.String() != "2026-03-01" ||
		amountText(stored.Budget) != "100000.00" || string(stored.Attributes[fit4ID]) != "true" || stored.Version != 2 ||
		stored.UpdatedBy.ID != daniel.ID {
		t.Errorf("stored = %+v, want it in Rimula fleet with its values, version 2, changed by Daniel", stored)
	}
	if moved.Campaign.Name != "Rimula fleet" || moved.Counts.Activities != 3 || moved.Counts.LineItems != 2 {
		t.Errorf("view = in %q with %+v, want Rimula fleet with its 3 activities", moved.Campaign.Name, moved.Counts)
	}
	for _, c := range f.list(t, "P-1001").Items {
		switch c.ID {
		case d.helix.ID:
			if c.Counts.Initiatives != 1 || c.Counts.Activities != 0 || amountText(c.BudgetAndCost.RemainingForChildren) != "400000.00" {
				t.Errorf("Helix launch = %+v %+v", c.Counts, c.BudgetAndCost)
			}
		case d.rimula.ID:
			if c.Counts.Initiatives != 2 || c.Counts.Activities != 3 || amountText(c.BudgetAndCost.OverAllocatedBy) != "50000.00" {
				t.Errorf("Rimula fleet = %+v %+v", c.Counts, c.BudgetAndCost)
			}
		}
	}
}

// A move to a campaign that is gone, of another plan, or the initiative's own is refused with its reason;
// one that already has the name and whose dates do not fit is refused with both reasons at once, and a
// missing version with the others (FR-023).
func TestMoveInitiative_RefusedWithEveryReason_FR023(t *testing.T) {
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

// A campaign copy has every stored field of the original, the chosen name and its destination, with new
// bookkeeping; its initiatives are copied with their names and fields, and their activities and line items
// through feature 006, within the year (FR-027, R13, US6 scenario 5).
func TestCopyCampaign_CopiesEverythingBelow_FR027(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	f.advance(time.Hour)

	copied, err := f.service.CopyCampaign(t.Context(), daniel, d.helix.ID, CampaignCopy{PlanID: "P-1001", Name: " Helix launch (copy) "})

	if err != nil {
		t.Fatalf("CopyCampaign: %v", err)
	}
	original, _ := f.storedCampaign(t, d.helix.ID)
	want := original
	want.ID, want.Name, want.Meta = copied.ID, "Helix launch (copy)", record.NewMeta(daniel, start.Add(time.Hour))
	if got, _ := f.storedCampaign(t, copied.ID); copied.ID == d.helix.ID || !reflect.DeepEqual(got, want) {
		t.Errorf("stored copy =\n%+v\nwant\n%+v", got, want)
	}
	if len(copied.Initiatives) != 2 || copied.Counts.Activities != 3 || copied.Counts.LineItems != 2 {
		t.Fatalf("copy holds %+v, want 2 initiatives, 3 activities and 2 line items", copied.Counts)
	}
	var digitalCopy InitiativeView
	for _, i := range copied.Initiatives {
		if i.Name == "Digital & social" {
			digitalCopy = i
		}
	}
	originalDigital, _ := f.storedInitiative(t, d.digital.ID)
	wantDigital := originalDigital
	wantDigital.ID, wantDigital.CampaignID, wantDigital.Meta = digitalCopy.ID, copied.ID, record.NewMeta(daniel, start.Add(time.Hour))
	if got, _ := f.storedInitiative(t, digitalCopy.ID); digitalCopy.ID == d.digital.ID || !reflect.DeepEqual(got, wantDigital) {
		t.Errorf("initiative copy =\n%+v\nwant\n%+v", got, wantDigital)
	}
	if len(f.activities.copies) != 2 || !strings.HasSuffix(f.activities.copies[0], " +0") {
		t.Errorf("activity copies = %q, want one per initiative, within the year", f.activities.copies)
	}
}

// A copy is made completely or not at all: when copying the activities fails, no campaign, initiative or
// log entry is stored and no ID is used up (FR-027).
func TestCopyCampaign_AllOrNothing_FR027(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	f.activities.copyErr = errors.New("disk full")
	entries := len(f.log(t, "", ""))

	_, err := f.service.CopyCampaign(t.Context(), priya, d.helix.ID, CampaignCopy{PlanID: "P-1001", Name: "Helix launch (copy)"})

	if !errors.Is(err, f.activities.copyErr) {
		t.Fatalf("CopyCampaign = %v, want the activities' error", err)
	}
	if tree := f.list(t, "P-1001"); tree.Totals.Campaigns != 2 || tree.Totals.Initiatives != 3 {
		t.Errorf("plan holds %d campaigns and %d initiatives, want the 2 and 3 of before", tree.Totals.Campaigns, tree.Totals.Initiatives)
	}
	if got := len(f.log(t, "", "")); got != entries {
		t.Errorf("log entries = %d, want %d", got, entries)
	}
	if next := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Next"}); next.ID != "C-1003" {
		t.Errorf("next campaign = %s, want C-1003: the failed copy used up no ID", next.ID)
	}
}

// The copy's name is checked like a new campaign's name in the destination plan, and the destination must
// be a plan of the same year; all problems come at once. The same name is fine in another plan (FR-026).
func TestCopyCampaign_NameCheckedInDestination_FR026(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "HELIX launch"})
	copyTo := func(planID, name string) (CampaignView, error) {
		return f.service.CopyCampaign(t.Context(), priya, d.helix.ID, CampaignCopy{PlanID: planID, Name: name})
	}

	_, taken := copyTo("P-1002", "helix launch")
	_, both := copyTo("P-1003", " ")
	_, gone := copyTo("P-9999", "Helix launch")
	_, own := copyTo("P-1001", "Rimula FLEET")
	elsewhere, err := copyTo("P-1001", "Helix launch 2")
	_, missing := f.service.CopyCampaign(t.Context(), priya, "C-9999", CampaignCopy{PlanID: "P-1001", Name: "Copy"})

	assertError(t, taken, problems("name", "'FY2026 Mobility China' already has a campaign called 'HELIX launch'."))
	assertError(t, both, problems("name", "Enter a name.", "planId", "Campaigns can be copied only into plans of the same year (2026)."))
	assertError(t, gone, problems("planId", "That plan no longer exists — choose another one."))
	assertError(t, own, problems("name", "'FY2026 Lubricants Europe' already has a campaign called 'Rimula fleet'."))
	assertError(t, missing, errs.NotFound(campaignGoneMessage))
	if err != nil || elsewhere.Name != "Helix launch 2" || elsewhere.PlanID != "P-1001" {
		t.Errorf("CopyCampaign = %q in %s, %v; want Helix launch 2 in P-1001", elsewhere.Name, elsewhere.PlanID, err)
	}
}

// An initiative is copied only into a campaign of its plan whose dates contain its dates; the name is
// checked in that campaign (FR-026, R3).
func TestCopyInitiative_DatesMustFit_FR026(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	late := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Late launch", StartDate: day("2026-06-01"), EndDate: day("2026-12-31")})
	other := f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "Retail growth"})
	copyTo := func(campaignID, name string) (InitiativeView, error) {
		return f.service.CopyInitiative(t.Context(), priya, d.digital.ID, InitiativeCopy{CampaignID: campaignID, Name: name})
	}

	_, dates := copyTo(late.ID, "Digital & social")
	_, otherPlan := copyTo(other.ID, "Digital & social")
	_, gone := copyTo("C-9999", "Digital & social")
	_, taken := copyTo(d.helix.ID, "digital & SOCIAL")
	copied, err := copyTo(d.rimula.ID, "Digital & social")
	_, missing := f.service.CopyInitiative(t.Context(), priya, "I-9999", InitiativeCopy{CampaignID: d.helix.ID, Name: "Copy"})

	assertError(t, dates, problems("campaignId", "Dates don't fit (01 Jun – 31 Dec 2026) — change the initiative's dates first."))
	assertError(t, otherPlan, problems("campaignId", "Initiatives can be copied only into campaigns of the same plan."))
	assertError(t, gone, problems("campaignId", "That campaign no longer exists — choose another one."))
	assertError(t, taken, problems("name", "'Helix launch' already has an initiative called 'Digital & social'."))
	assertError(t, missing, errs.NotFound(initiativeGoneMessage))
	if err != nil || copied.CampaignID != d.rimula.ID || copied.ID == d.digital.ID || copied.Counts.Activities != 3 ||
		copied.StartDate.String() != "2026-03-01" {
		t.Errorf("CopyInitiative = %+v, %v; want a new initiative in Rimula fleet with the 3 activities", copied, err)
	}
}

// A copy writes one "copied" entry, on the new record only, naming the original and what came with it;
// the records copied below it get none (D-097, FR-030).
func TestCopy_OneCopiedEntryWithDetails_DecisionH(t *testing.T) {
	f := newFixture(t)
	d := f.newDemo(t)
	initiativeEntries := len(f.log(t, initiativeKind, ""))

	campaignCopy, err := f.service.CopyCampaign(t.Context(), daniel, d.helix.ID, CampaignCopy{PlanID: "P-1001", Name: "Helix launch (copy)"})
	if err != nil {
		t.Fatalf("CopyCampaign: %v", err)
	}
	printCopy, err := f.service.CopyInitiative(t.Context(), daniel, d.print.ID, InitiativeCopy{CampaignID: d.helix.ID, Name: "Print & outdoor (copy)"})
	if err != nil {
		t.Fatalf("CopyInitiative: %v", err)
	}

	entries := f.log(t, campaignKind, campaignCopy.ID)
	if len(entries) != 1 || entries[0].Action != history.ActionCopied || entries[0].Actor != daniel ||
		entries[0].Details != "Copied from 'Helix launch' ("+d.helix.ID+"), with 2 initiatives and 3 activities" {
		t.Errorf("entries of the copy = %+v, want one copied entry with the counts", entries)
	}
	printEntries := f.log(t, initiativeKind, printCopy.ID)
	if len(printEntries) != 1 || printEntries[0].Details != "Copied from 'Print & outdoor' ("+d.print.ID+")" {
		t.Errorf("entries of the initiative copy = %+v, want one, without contents", printEntries)
	}
	if got := len(f.log(t, initiativeKind, "")); got != initiativeEntries+1 {
		t.Errorf("initiative entries = %d, want %d: the initiatives copied with the campaign write none", got, initiativeEntries+1)
	}
}

// A copy suggests "<name> (copy)", then "(copy 2)", "(copy 3)", … — the first name taken does not report
// (R13).
func TestSuggestCopyName_CopyThenNumbers_R13(t *testing.T) {
	taken := []string{"helix launch (COPY)", "Helix launch (copy 2)"}
	isTaken := func(name string) bool {
		return slices.ContainsFunc(taken, func(t string) bool { return strings.EqualFold(t, name) })
	}

	if got := suggestCopyName("Helix launch", func(string) bool { return false }); got != "Helix launch (copy)" {
		t.Errorf("first copy = %q", got)
	}
	if got := suggestCopyName("Helix launch", isTaken); got != "Helix launch (copy 3)" {
		t.Errorf("third copy = %q, want Helix launch (copy 3)", got)
	}
}

// A copy name stays within 120 characters: the original part is shortened, without trailing spaces, so the
// suffix fits (edge case).
func TestSuggestCopyName_ShortensLongNames_EdgeCase(t *testing.T) {
	long := strings.Repeat("ü", 112) + " " + strings.Repeat("x", 7) // 120 characters
	if got, want := suggestCopyName(long, func(string) bool { return false }), strings.Repeat("ü", 112)+" (copy)"; got != want {
		t.Errorf("copy name = %q, want %q", got, want)
	}
	second := suggestCopyName(long, func(n string) bool { return strings.HasSuffix(n, " (copy)") })
	if utf8.RuneCountInString(second) != 120 || !strings.HasSuffix(second, " (copy 2)") {
		t.Errorf("second copy name = %q (%d characters), want 120 ending in (copy 2)", second, utf8.RuneCountInString(second))
	}
}
