package campaigns

import (
	"encoding/json"
	"reflect"
	"testing"

	"mapp/internal/platform/errs"
	"mapp/internal/settings"
)

// An initiative moves only to another campaign of its plan: one that already has an initiative with its
// name, or whose dates do not contain its dates, is listed but not available, with every reason and the fix;
// a campaign without dates fits any dates (FR-023, US6 scenario 2). They come by start date, then name.
func TestInitiativeDestinations_Move_UnavailableWithReasonAndFix_FR023(t *testing.T) { //nolint:gocyclo // 10: one list, then its rows, rule and figures checked in turn
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
