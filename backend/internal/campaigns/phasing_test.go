package campaigns

import (
	"reflect"
	"testing"
	"time"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// quarterly is the custom phasing of the demo campaign "Helix UK & Ireland": 100,000.00 GBP in each of Mar,
// Jun, Sep and Nov 2027 (016 data-model §8).
func quarterly(version int) rollup.PhasingInput {
	return rollup.PhasingInput{Method: "custom", Version: &version, Months: []rollup.PhasingMonthInput{
		{Month: "2027-03", Amount: "100000"}, {Month: "2027-06", Amount: "100000"},
		{Month: "2027-09", Amount: "100000"}, {Month: "2027-11", Amount: "100000"},
	}}
}

// phasedCampaign is the GBP campaign "Helix UK & Ireland" (400,000.00 GBP) with its initiative "UK"
// (250,000.00 GBP) in P-1004.
func phasedCampaign(t *testing.T, f *fixture) (CampaignView, InitiativeView) {
	t.Helper()
	withCurrencyPlans(f)
	c := f.createCampaign(t, priya, "P-1004", CampaignInput{Name: "Helix UK & Ireland", Budget: &money.Input{Amount: "400000", Currency: "GBP"}})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "UK", Budget: &money.Input{Amount: "250000", Currency: "GBP"}})
	return c, i
}

// A campaign's and an initiative's budgets are phased like the plan's: saved with their own operation over
// the plan's year in the plan currency, each change logged month by month (FR-020, FR-023).
func TestSetCampaignPhasing_SavesAndLogs_FR020_FR023(t *testing.T) {
	f := newFixture(t)
	c, i := phasedCampaign(t, f)
	f.advance(time.Hour)

	campaign, err := f.service.SetCampaignBudgetPhasing(t.Context(), daniel, c.ID, quarterly(1))
	initiative, initiativeErr := f.service.SetInitiativeBudgetPhasing(t.Context(), daniel, i.ID,
		rollup.PhasingInput{Method: "evenly", Version: ptr(1)})

	if err != nil || campaign.BudgetPhasing.Method != rollup.MethodCustom || campaign.BudgetPhasing.Monthly[2].Format() != "100,000.00 GBP" ||
		campaign.BudgetPhasing.Total.Format() != "400,000.00 GBP" || campaign.BudgetPhasing.Problem != nil || campaign.Version != 2 {
		t.Fatalf("campaign = %+v, version %d, %v", campaign.BudgetPhasing, campaign.Version, err)
	}
	if initiativeErr != nil || initiative.BudgetPhasing.Monthly[11].Format() != "20,833.37 GBP" || initiative.Version != 2 {
		t.Errorf("initiative = %+v, %v", initiative.BudgetPhasing, initiativeErr)
	}
	entry := f.log(t, "campaign", c.ID)[0]
	var labels []string
	for _, change := range entry.Changes {
		labels = append(labels, change.Label+" "+change.NewText)
	}
	want := []string{"Budget by month Custom", "Budget by month: Mar 2027 100,000.00 GBP", "Budget by month: Jun 2027 100,000.00 GBP",
		"Budget by month: Sep 2027 100,000.00 GBP", "Budget by month: Nov 2027 100,000.00 GBP"}
	if entry.Actor != daniel || !reflect.DeepEqual(labels, want) {
		t.Errorf("log = %q by %v", labels, entry.Actor)
	}
}

// The rules of a phasing save hold for initiatives too: a budget first, months of the plan's year, the
// version the person started from (FR-020).
func TestSetInitiativePhasing_Rules_FR020(t *testing.T) {
	f := newFixture(t)
	c, _ := phasedCampaign(t, f)
	empty := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Ireland"})

	_, noBudget := f.service.SetInitiativeBudgetPhasing(t.Context(), priya, empty.ID, rollup.PhasingInput{Method: "evenly", Version: ptr(1)})
	_, wrongYear := f.service.SetCampaignBudgetPhasing(t.Context(), priya, c.ID, rollup.PhasingInput{Method: "custom", Version: ptr(1),
		Months: []rollup.PhasingMonthInput{{Month: "2026-12", Amount: "1"}}})
	_, stale := f.service.SetCampaignBudgetPhasing(t.Context(), priya, c.ID, rollup.PhasingInput{Method: "evenly", Version: ptr(7)})
	_, gone := f.service.SetInitiativeBudgetPhasing(t.Context(), priya, "I-9999", rollup.PhasingInput{Method: "none", Version: ptr(1)})

	assertError(t, noBudget, problems("method", "Set a budget before phasing it."))
	assertError(t, wrongYear, problems("months", "Months must be in 2027."))
	assertError(t, stale, errs.VersionConflict("campaign", "Priya Raman", start))
	assertError(t, gone, errs.NotFound(initiativeGoneMessage))
}

// Removing a campaign's budget removes its phasing in the same save (FR-022).
func TestCampaignBudgetRemoved_ResetsPhasing_FR022(t *testing.T) {
	f := newFixture(t)
	c, _ := phasedCampaign(t, f)
	if _, err := f.service.SetCampaignBudgetPhasing(t.Context(), priya, c.ID, quarterly(1)); err != nil {
		t.Fatalf("SetCampaignBudgetPhasing: %v", err)
	}

	view, err := f.service.UpdateCampaign(t.Context(), priya, c.ID, CampaignUpdate{Name: c.Name, Version: ptr(2)})

	if err != nil || view.Budget != nil || view.BudgetPhasing.Method != rollup.MethodNone {
		t.Fatalf("after removing the budget: %v %+v, %v", view.Budget, view.BudgetPhasing, err)
	}
	changes := f.log(t, "campaign", c.ID)[0].Changes
	if changes[0].Field != "budget" || changes[1].Label != "Budget by month" || changes[1].NewText != "Not phased" {
		t.Errorf("changes = %+v", changes)
	}
}

// A Custom phasing that does not add up is a needed-to-publish problem of the campaign or initiative, and so
// one of the plan's publish problems (FR-022).
func TestCampaignCompleteness_PhasingProblem_FR022(t *testing.T) {
	f := newFixture(t)
	c, i := phasedCampaign(t, f)
	if _, err := f.service.SetCampaignBudgetPhasing(t.Context(), priya, c.ID, quarterly(1)); err != nil {
		t.Fatalf("SetCampaignBudgetPhasing: %v", err)
	}
	if _, err := f.service.SetInitiativeBudgetPhasing(t.Context(), priya, i.ID, rollup.PhasingInput{Method: "custom", Version: ptr(1),
		Months: []rollup.PhasingMonthInput{{Month: "2027-03", Amount: "200000"}}}); err != nil {
		t.Fatalf("SetInitiativeBudgetPhasing: %v", err)
	}

	tree := f.list(t, "P-1004")
	rollups, err := f.service.PlanRollups(t.Context(), []string{"P-1004"})

	problem := settings.MissingField{Field: "budgetPhasing", Label: "Budget by month", Reason: "Adds up to 200,000.00 GBP, not the budget 250,000.00 GBP"}
	initiative := tree.Items[0].Initiatives[0]
	if len(initiative.Completeness.Missing) == 0 || initiative.Completeness.Missing[len(initiative.Completeness.Missing)-1] != problem {
		t.Errorf("initiative completeness = %+v", initiative.Completeness.Missing)
	}
	for _, m := range tree.Items[0].Completeness.Missing {
		if m.Field == "budgetPhasing" {
			t.Errorf("the campaign's phasing adds up, but it is missing %+v", m)
		}
	}
	found := false
	for _, p := range rollups["P-1004"].PublishProblems {
		for _, m := range p.Missing {
			found = found || (p.Record.ID == i.ID && m == problem)
		}
	}
	if err != nil || !found {
		t.Errorf("publish problems = %+v, %v", rollups["P-1004"].PublishProblems, err)
	}
	if r := rollups["P-1004"]; r.CampaignBudgetsSet != 1 || r.InitiativeBudgetsSet != 1 {
		t.Errorf("budgets set = %d campaign, %d initiative", r.CampaignBudgetsSet, r.InitiativeBudgetsSet)
	}
}

// Moves and copies keep the phasing; a copy into the next year (plan copy) moves the months one year (FR-025).
func TestMoveCopy_KeepPhasing_FR025(t *testing.T) {
	f := newFixture(t)
	c, i := phasedCampaign(t, f)
	if _, err := f.service.SetCampaignBudgetPhasing(t.Context(), priya, c.ID, quarterly(1)); err != nil {
		t.Fatalf("SetCampaignBudgetPhasing: %v", err)
	}
	if _, err := f.service.SetInitiativeBudgetPhasing(t.Context(), priya, i.ID, rollup.PhasingInput{Method: "evenly", Version: ptr(1)}); err != nil {
		t.Fatalf("SetInitiativeBudgetPhasing: %v", err)
	}
	f.plans.plans["P-1006"] = PlanFacts{ID: "P-1006", Name: "FY2028 Lubricants UK & Ireland", Year: 2028, Status: "draft",
		Currency: "GBP", ModelID: "MOD-1001", Attributes: settings.AttributeValues{}}

	copied, err := f.service.CopyCampaign(t.Context(), priya, c.ID, CampaignCopy{PlanID: "P-1004", Name: "Helix (copy)"})
	_, newIDs, planErr := f.service.CopyPlanContents(t.Context(), priya, "P-1004", "P-1006", 1)

	if err != nil || copied.BudgetPhasing.Months[0].Month != "2027-03" || copied.Initiatives[0].BudgetPhasing.Method != rollup.MethodEvenly {
		t.Errorf("copy = %+v, %v", copied.BudgetPhasing, err)
	}
	next, _ := f.storedCampaign(t, newIDs[c.ID])
	if planErr != nil || next.BudgetPhasing.Months[3].Month != "2028-11" {
		t.Errorf("next-year copy = %+v, %v", next.BudgetPhasing, planErr)
	}
}
