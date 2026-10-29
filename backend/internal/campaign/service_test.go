package campaign

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/overspend"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

func TestCreateCampaign_OnlyNameRequired_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "  Helix launch  "})

	stored, _ := f.storedCampaign(t, view.ID)
	want := Campaign{ID: "C-1001", PlanID: "P-1001", Name: "Helix launch", Attributes: settings.AttributeValues{},
		BudgetPhasing: rollup.NotPhased, Overspend: overspend.Warn, Meta: record.NewMeta(priya, start)}
	if !reflect.DeepEqual(stored, want) {
		t.Errorf("stored = %+v\nwant %+v", stored, want)
	}
	if view.Owner != nil || view.StartDate != nil || view.Budget != nil || view.Timing != nil {
		t.Errorf("view = %+v, want no owner, dates, budget or timing", view)
	}
	if len(view.Completeness.Missing) != 4 {
		t.Errorf("missing = %+v, want owner, dates, cluster and region", view.Completeness.Missing)
	}
}

func TestCreateCampaign_EveryProblemAtOnce_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})

	_, err := f.service.CreateCampaign(t.Context(), priya, "P-1001", CampaignInput{
		Name: "helix LAUNCH", Description: strings.Repeat("d", 2001),
		StartDate: day("2026-12-31"), EndDate: day("2026-03-01"), Budget: input("-1.00"),
	})

	assertError(t, err, problems(
		"name", "'FY2026 Lubricants Europe' already has a campaign called 'Helix launch'.",
		"description", "Use 2,000 characters or fewer.",
		"endDate", "End date must be on or after the start date (31 Dec 2026).",
		"budget", "Budget can't be negative.",
	))

	tests := []struct {
		name string
		in   CampaignInput
		want *errs.Error
	}{
		{"no name", CampaignInput{Name: "   "}, problems("name", "Enter a name.")},
		{"121 characters", CampaignInput{Name: strings.Repeat("é", 121)}, problems("name", "Use 120 characters or fewer.")},
		{"start only", CampaignInput{Name: "A", StartDate: day("2026-01-01")},
			problems("endDate", "Enter an end date too, or clear the start date.")},
		{"end only", CampaignInput{Name: "A", EndDate: day("2026-01-01")},
			problems("startDate", "Enter a start date too, or clear the end date.")},
		{"three decimals", CampaignInput{Name: "A", Budget: input("10.005")},
			problems("budget", "Enter an amount in USD with at most 2 decimals, such as 12500 or 12500.50.")},
		{"not a number", CampaignInput{Name: "A", Budget: input("lots")},
			problems("budget", "Enter an amount in USD with at most 2 decimals, such as 12500 or 12500.50.")},
		{"not USD", CampaignInput{Name: "A", Budget: &moneyInputEUR},
			problems("budget", "Enter the amount in USD, the plan's currency.")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.CreateCampaign(t.Context(), priya, "P-1001", tt.in)

			assertError(t, err, tt.want)
		})
	}
	if tree := f.list(t, "P-1001"); tree.Total != 1 {
		t.Errorf("%d campaigns, want only the first one", tree.Total)
	}
}

var moneyInputEUR = moneyInput("100.00", "EUR")

func TestCreateCampaign_NameUniqueInPlanIgnoringCaseAndSpaces_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})

	_, err := f.service.CreateCampaign(t.Context(), daniel, "P-1001", CampaignInput{Name: "  HELIX launch "})
	assertError(t, err, problems("name", "'FY2026 Lubricants Europe' already has a campaign called 'Helix launch'."))

	other := f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "Helix launch"})
	if other.PlanID != "P-1002" {
		t.Errorf("campaign in another plan = %+v", other)
	}
}

func TestCreateCampaign_ReadableIDAndCreatedEntry_FR004_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	first := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	second := f.createCampaign(t, daniel, "P-1001", CampaignInput{Name: "Rimula fleet"})

	if first.ID != "C-1001" || second.ID != "C-1002" {
		t.Errorf("IDs = %s, %s; want C-1001, C-1002", first.ID, second.ID)
	}
	entries := f.log(t, campaignKind, "C-1002")
	if len(entries) != 1 {
		t.Fatalf("%d log entries, want 1: %+v", len(entries), entries)
	}
	e := entries[0]
	wantPath := []history.RecordRef{{Type: planKind, ID: "P-1001", Name: "FY2026 Lubricants Europe"}}
	if e.Action != history.ActionCreated || e.Actor != daniel || e.RecordName != "Rimula fleet" || !reflect.DeepEqual(e.Path, wantPath) {
		t.Errorf("entry = %+v", e)
	}
}

func TestCreateCampaign_StoresWhatItReceives_DecisionD(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	values := settings.AttributeValues{clusterID: json.RawMessage(`["SO-1003"]`), regionID: json.RawMessage(`["SO-1010"]`)}

	view := f.createCampaign(t, priya, "P-1001", CampaignInput{
		Name: "Helix launch", Description: " Premium ", OwnerID: ptr("U-1003"),
		StartDate: day("2026-01-01"), EndDate: day("2026-12-31"), Budget: input("700000"), Attributes: values,
	})

	stored, _ := f.storedCampaign(t, view.ID)
	if *stored.OwnerID != "U-1003" || stored.Description != "Premium" || *stored.StartDate != *day("2026-01-01") ||
		*stored.EndDate != *day("2026-12-31") || amountText(stored.Budget) != "700000.00" || !reflect.DeepEqual(stored.Attributes, values) {
		t.Errorf("stored = %+v", stored)
	}
	if view.Owner == nil || view.Owner.Name != "Daniel Okoro" || len(view.Completeness.Missing) != 0 {
		t.Errorf("view owner %+v, missing %+v; want Daniel and nothing missing", view.Owner, view.Completeness.Missing)
	}
	noOwner := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Without owner", OwnerID: ptr("")})
	if noOwner.OwnerID != nil {
		t.Errorf("owner \"\" stored as %v, want none", *noOwner.OwnerID)
	}
}

func TestCreateCampaign_NewOwnerMustBeActive_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for _, id := range []string{sam.ID, "U-9999"} {
		_, err := f.service.CreateCampaign(t.Context(), priya, "P-1001", CampaignInput{Name: "Helix launch", OwnerID: &id})

		assertError(t, err, problems("ownerId", "Choose an active person."))
	}
}

func TestCreateCampaign_AttributeProblemsReportedWithOthers_FR009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	regionProblem := errs.FieldError{Field: "attributes." + regionID, Message: "'RETAIL – CHINA' doesn't belong to the chosen cluster — choose another region."}
	f.settings.problems[settings.LevelCampaign] = []errs.FieldError{regionProblem}

	_, err := f.service.CreateCampaign(t.Context(), priya, "P-1001", CampaignInput{Name: "", Budget: input("-5")})

	assertError(t, err, problems(
		"name", "Enter a name.",
		"budget", "Budget can't be negative.",
		regionProblem.Field, regionProblem.Message,
	))
}

func TestCreateCampaign_OutsidePlanYearWarnsOnly_FR008(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	outside := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Long", StartDate: day("2026-01-01"), EndDate: day("2027-03-31")})
	before := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Early", StartDate: day("2025-12-31"), EndDate: day("2026-01-31")})
	inside := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Inside", StartDate: day("2026-01-01"), EndDate: day("2026-12-31")})
	undated := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Undated"})

	if !outside.OutsidePlanYear || !before.OutsidePlanYear || inside.OutsidePlanYear || undated.OutsidePlanYear {
		t.Errorf("outsidePlanYear = %v, %v, %v, %v; want true, true, false, false",
			outside.OutsidePlanYear, before.OutsidePlanYear, inside.OutsidePlanYear, undated.OutsidePlanYear)
	}
}

func TestCreateCampaign_OverAllocationAllowed_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.createCampaign(t, meiLin, "P-1002", CampaignInput{Name: "Retail growth", Budget: input("600000.00")})

	view := f.createCampaign(t, meiLin, "P-1002", CampaignInput{Name: "Fleet growth", Budget: input("300000.00")})

	if amountText(view.ParentRemainingBudget) != "-100000.00" {
		t.Errorf("remaining plan budget = %s, want -100000.00", amountText(view.ParentRemainingBudget))
	}
	totals := f.list(t, "P-1002").Totals
	if amountText(totals.RemainingPlanBudget) != "-100000.00" || amountText(totals.OverAllocatedBy) != "100000.00" {
		t.Errorf("totals = %+v, want remaining -100000.00, over by 100000.00", totals)
	}
}

func TestCreateCampaign_PlanGone_NotFound(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.CreateCampaign(t.Context(), priya, "P-9999", CampaignInput{Name: "Helix launch"})

	assertError(t, err, errs.NotFound("This plan no longer exists — someone may have deleted it."))
	_, err = f.service.ListPlanCampaigns(t.Context(), priya, "P-9999")
	assertError(t, err, errs.NotFound("This plan no longer exists — someone may have deleted it."))
}

func TestGetCampaign_WithInitiatives_FR016(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social"})

	got, err := f.service.GetCampaign(t.Context(), daniel, c.ID)

	if err != nil || len(got.Initiatives) != 1 || got.Initiatives[0].Name != "Digital & social" || got.Counts.Initiatives != 1 {
		t.Errorf("GetCampaign = %+v, %v", got, err)
	}
	_, err = f.service.GetCampaign(t.Context(), daniel, "C-9999")
	assertError(t, err, errs.NotFound("This campaign no longer exists — someone may have deleted it."))
}

func TestCreateInitiative_DatesInsideCampaign_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", StartDate: day("2026-03-01"), EndDate: day("2026-12-31")})
	outside := problems("startDate", "Dates must be inside the campaign's dates (01 Mar 2026 – 31 Dec 2026).")
	tests := []struct {
		name       string
		start, end string
		want       *errs.Error
	}{
		{"starts before", "2026-02-01", "2026-06-30", outside},
		{"ends after", "2026-03-01", "2027-01-01", outside},
		{"exactly the campaign's dates", "2026-03-01", "2026-12-31", nil},
		{"inside", "2026-04-01", "2026-05-31", nil},
	}
	for n, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := "Initiative " + string(rune('A'+n))
			_, err := f.service.CreateInitiative(t.Context(), priya, helix.ID, InitiativeInput{Name: name, StartDate: day(tt.start), EndDate: day(tt.end)})

			if tt.want == nil {
				if err != nil {
					t.Errorf("CreateInitiative: %v, want it saved", err)
				}
				return
			}
			assertError(t, err, tt.want)
		})
	}
	undated := f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Undated"})
	if undated.StartDate != nil || undated.Campaign.StartDate == nil {
		t.Errorf("undated initiative = %+v; an initiative without dates is allowed", undated)
	}
}

func TestCreateInitiative_CampaignWithoutDates_NotChecked_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})

	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social", StartDate: day("2025-01-01"), EndDate: day("2028-12-31")})

	if i.StartDate == nil || i.Campaign.StartDate != nil {
		t.Errorf("initiative = %+v", i)
	}
}

func TestCreateInitiative_NameUniqueInCampaign_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	rimula := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Rimula fleet"})
	f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Digital & social"})

	_, err := f.service.CreateInitiative(t.Context(), priya, helix.ID, InitiativeInput{
		Name: " DIGITAL & SOCIAL ", OwnerID: ptr(sam.ID), StartDate: day("2026-05-01"), Budget: input("1.234"),
	})

	assertError(t, err, problems(
		"name", "'Helix launch' already has an initiative called 'Digital & social'.",
		"endDate", "Enter an end date too, or clear the start date.",
		"budget", "Enter an amount in USD with at most 2 decimals, such as 12500 or 12500.50.",
		"ownerId", "Choose an active person.",
	))
	f.createInitiative(t, priya, rimula.ID, InitiativeInput{Name: "Digital & social"})
}

func TestCreateInitiative_IDRemainingAndCreatedEntry_FR004_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", Budget: input("400000.00")})
	f.createInitiative(t, priya, helix.ID, InitiativeInput{Name: "Print & outdoor", Budget: input("300000.00")})

	digital := f.createInitiative(t, daniel, helix.ID, InitiativeInput{Name: "Digital & social", Budget: input("150000.00")})

	if digital.ID != "I-1002" || digital.PlanID != "P-1001" || amountText(digital.ParentRemainingBudget) != "-50000.00" {
		t.Errorf("initiative = %+v", digital)
	}
	campaign, _ := f.service.GetCampaign(t.Context(), priya, helix.ID)
	if len(campaign.Initiatives) != 2 || amountText(campaign.BudgetAndCost.OverAllocatedBy) != "50000.00" {
		t.Errorf("campaign = %+v, want 2 initiatives and over-allocated by 50000.00", campaign)
	}
	entries := f.log(t, initiativeKind, digital.ID)
	wantPath := []history.RecordRef{{Type: planKind, ID: "P-1001", Name: "FY2026 Lubricants Europe"}, {Type: campaignKind, ID: helix.ID, Name: "Helix launch"}}
	if len(entries) != 1 || entries[0].Action != history.ActionCreated || entries[0].Actor != daniel || !reflect.DeepEqual(entries[0].Path, wantPath) {
		t.Errorf("entries = %+v", entries)
	}
	_, err := f.service.CreateInitiative(t.Context(), priya, "C-9999", InitiativeInput{Name: "Lost"})
	assertError(t, err, errs.NotFound("This campaign no longer exists — someone may have deleted it."))
	_, err = f.service.GetInitiative(t.Context(), priya, "I-9999")
	assertError(t, err, errs.NotFound("This initiative no longer exists — someone may have deleted it."))
}
