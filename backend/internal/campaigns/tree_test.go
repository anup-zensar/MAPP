package campaigns

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/platform/money"
)

// Rows are ordered by start date, then name ignoring case, records without dates last — campaigns and,
// inside each, initiatives (FR-020).
func TestListPlanCampaigns_OrderByStartThenNameUndatedLast_FR020(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for _, in := range []CampaignInput{
		{Name: "Zulu undated"},
		{Name: "beta", StartDate: day("2026-02-01"), EndDate: day("2026-12-31")},
		{Name: "Alpha undated"},
		{Name: "Alpha", StartDate: day("2026-02-01"), EndDate: day("2026-03-31")},
		{Name: "Gamma", StartDate: day("2026-01-15"), EndDate: day("2026-02-28")},
	} {
		f.createCampaign(t, priya, "P-1001", in)
	}
	gamma := f.list(t, "P-1001").Items[0]
	for _, in := range []InitiativeInput{
		{Name: "Undated"},
		{Name: "Second", StartDate: day("2026-02-01"), EndDate: day("2026-02-28")},
		{Name: "First", StartDate: day("2026-01-15"), EndDate: day("2026-02-28")},
	} {
		f.createInitiative(t, priya, gamma.ID, in)
	}

	tree := f.list(t, "P-1001")

	var campaigns, initiatives []string
	for _, c := range tree.Items {
		campaigns = append(campaigns, c.Name)
	}
	for _, i := range tree.Items[0].Initiatives {
		initiatives = append(initiatives, i.Name)
	}
	if want := []string{"Gamma", "Alpha", "beta", "Alpha undated", "Zulu undated"}; !slices.Equal(campaigns, want) {
		t.Errorf("campaigns = %q, want %q", campaigns, want)
	}
	if want := []string{"First", "Second", "Undated"}; !slices.Equal(initiatives, want) {
		t.Errorf("initiatives = %q, want %q", initiatives, want)
	}
}

// The totals row counts campaigns, initiatives and activities, sums the campaign budgets ("Allocated to
// campaigns") and gives the plan's planned cost and remaining budget (FR-022, US4 scenario 5).
func TestListPlanCampaigns_Totals_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	helix := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", Budget: input("700000.00")})
	rimula := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Rimula fleet", Budget: input("450000.00")})
	f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "Elsewhere", Budget: input("1.00")})
	var initiatives []InitiativeView
	for _, c := range []CampaignView{helix, helix, rimula, rimula} {
		initiatives = append(initiatives, f.createInitiative(t, priya, c.ID, InitiativeInput{Name: fmt.Sprintf("Initiative %c", 'A'+len(initiatives))}))
	}
	for n, i := range []int{0, 0, 1, 1, 2, 3, 3} {
		f.activities.add(ActivitySummary{ID: "A-" + string(rune('1'+n)), InitiativeID: initiatives[i].ID, PlannedCost: usd("1000.25")})
	}

	totals := f.list(t, "P-1001").Totals

	want := PlanCampaignTotals{
		Campaigns: 2, Initiatives: 4, Activities: 7, AllocatedToCampaigns: *usd("1150000.00"),
		PlanBudget: usd("1200000.00"), RemainingPlanBudget: usd("50000.00"), PlannedCost: usd("7001.75"),
	}
	if !reflect.DeepEqual(totals, want) {
		t.Errorf("totals = %+v\nwant %+v", totals, want)
	}
	if empty := f.list(t, "P-1003").Totals; empty.AllocatedToCampaigns != money.USD(money.Amount{}) || empty.Campaigns != 0 {
		t.Errorf("empty plan totals = %+v, want 0 campaigns and 0.00 allocated", empty)
	}
}
