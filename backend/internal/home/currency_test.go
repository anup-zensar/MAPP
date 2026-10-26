package home_test

import (
	"testing"

	"mapp/internal/home"
	"mapp/internal/platform/apitest"
)

// Home adds up several plans, so every plan's amounts are converted to USD at the current rate before they are
// added, and shown in the reader's display currency (016 FR-014). P-1004 "FY2027 Lubricants UK & Ireland" is
// planned in GBP (0.79 per USD): budget 600,000.00 GBP = 759,493.67 USD; planned 65,000.00 GBP = 82,278.48
// USD; A-1011 "UK launch event" 40,000.00 GBP = 50,632.91 USD.
func TestHomeSources_ConvertsPerPlan_FR014(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	plans := get[home.PlanPage](t, priya, "/api/v1/home/plans?scope=myTeams")
	mine := get[home.MyActivitiesView](t, priya, "/api/v1/home/my-activities")

	var uk *home.PlanRow
	for i := range plans.Items {
		if plans.Items[i].ID == "P-1004" {
			uk = &plans.Items[i]
		}
	}
	if uk == nil || uk.Budget == nil || *uk.Budget != usd(t, "759493.67") || uk.PlannedCost == nil ||
		*uk.PlannedCost != usd(t, "82278.48") || uk.BudgetUsedPercent == nil || *uk.BudgetUsedPercent != 11 {
		t.Errorf("P-1004 row = %+v, want 759,493.67 USD budget, 82,278.48 USD planned, 11 %%", uk)
	}
	var launch *home.MyActivityRow
	for _, g := range mine.Groups {
		for i := range g.Items {
			if g.Items[i].ID == "A-1011" {
				launch = &g.Items[i]
			}
		}
	}
	if launch == nil || launch.PlannedCost == nil || *launch.PlannedCost != usd(t, "50632.91") {
		t.Errorf("UK launch event = %+v, want 50,632.91 USD", launch)
	}
}
