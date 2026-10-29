package home_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/approval"
	"mapp/internal/home"
	"mapp/internal/plan"
	"mapp/internal/platform/apitest"
)

// User story 1 — See my work at a glance.

// Priya's cards with the seed: her team's one plan, Published; the seven activities she owns, one Declined
// and one Submitted; nothing waits for her; FY2026 Lubricants Europe's planned cost of its 1,200,000.00 USD
// budget. Daniel has Trade magazine ads waiting for him since its resubmission two days ago (FR-005, FR-006,
// US1 scenario 2).
func TestHomeSummary_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all") // 014: every market's plans
	daniel := server.SignIn(t, danielsEmail)
	server.GrantRole(t, danielsEmail, "globalPlanner", "all") // 014: every market's plans

	got := get[home.SummaryView](t, priya, "/api/v1/home/summary?scope=myTeams")
	everything := get[home.SummaryView](t, priya, "/api/v1/home/summary?scope=everything")
	danielsCards := get[home.SummaryView](t, daniel, "/api/v1/home/summary?scope=myTeams")

	if got.Scope != home.ScopeMyTeams || got.DisplayCurrency != "USD" {
		t.Errorf("scope %q, currency %q", got.Scope, got.DisplayCurrency)
	}
	// Since 016 her team also has P-1004, FY2027 Lubricants UK & Ireland (Draft, planned in GBP), with two of
	// her activities: 600,000.00 GBP = 759,493.67 USD budget, 65,000.00 GBP = 82,278.48 USD planned (FR-014).
	if got.Plans != (home.PlanCounts{Count: 2, Published: 1, Draft: 1}) || everything.Plans != (home.PlanCounts{Count: 4, Published: 1, Draft: 3}) {
		t.Errorf("plans = %+v (everything %+v); want 1 published, 1 draft (4: 1 published, 3 draft)", got.Plans, everything.Plans)
	}
	if o := got.ActivitiesOwned; o.Count != 9 || o.Declined != 1 || o.Submitted != 1 || o.YearFrom == nil ||
		*o.YearFrom != 2026 || o.YearTo == nil || *o.YearTo != 2027 { // 038 FR-032: her plans' years
		t.Errorf("activities owned = %+v, want 9 in 2026–2027: 1 declined, 1 submitted", o)
	}
	if got.WaitingForMe != (home.WaitingCounts{}) {
		t.Errorf("waiting for Priya = %+v, want nothing", got.WaitingForMe)
	}
	cost := got.PlannedCost
	if cost.PlanCount != 2 || cost.PlannedCost != usd(t, "542278.48") || cost.PlanBudget == nil ||
		*cost.PlanBudget != usd(t, "1959493.67") || cost.BudgetUsedPercent == nil || *cost.BudgetUsedPercent != 28 ||
		cost.OverBudget || cost.PlansWithoutBudget != 0 {
		t.Errorf("planned cost = %+v, want 542,278.48 of 1,959,493.67 USD, 28 %%", cost)
	}
	w := danielsCards.WaitingForMe
	if w.Count != 1 || w.OldestWaitingSince == nil || !w.OldestWaitingSince.Equal(now.Add(-48*time.Hour)) {
		t.Errorf("waiting for Daniel = %+v, want Trade magazine ads, waiting 2 days", w)
	}
}

// For every seeded person and both scopes, Home's figures equal those of the Plans list and the Approvals
// page: the plans by status, the planned cost and plan budget of the plans in scope, and what waits for the
// person (SC-002, FR-017).
func TestHome_EqualsPlanAndApprovalPages_SC002_FR017(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	for _, email := range everyone {
		client := server.SignIn(t, email)
		counts := get[approval.Counts](t, client, "/api/v1/approvals/summary", apitest.SkipValidation())
		rows := map[string]home.PlanPage{}
		for scope, listScope := range listScopes {
			summary := get[home.SummaryView](t, client, "/api/v1/home/summary?scope="+scope)
			list := get[plan.PlanList](t, client, "/api/v1/plans?scope="+listScope, apitest.SkipValidation())

			if summary.Plans != (home.PlanCounts{Count: list.Total, Published: list.Summary.Published, Draft: list.Summary.Draft}) {
				t.Errorf("%s, %s: plans %+v, Plans list %d (%+v)", email, scope, summary.Plans, list.Total, list.Summary)
			}
			if summary.PlannedCost.PlannedCost != orZero(list.Summary.PlannedCost) ||
				!same(summary.PlannedCost.PlanBudget, list.Summary.Budget) {
				t.Errorf("%s, %s: planned cost %v of %v, Plans list %v of %v", email, scope, summary.PlannedCost.PlannedCost,
					summary.PlannedCost.PlanBudget, list.Summary.PlannedCost, list.Summary.Budget)
			}
			if summary.WaitingForMe.Count != counts.WaitingForMe {
				t.Errorf("%s: waiting for me %d, Approvals page %d", email, summary.WaitingForMe.Count, counts.WaitingForMe)
			}

			rows[scope] = get[home.PlanPage](t, client, "/api/v1/home/plans?scope="+scope)
			inOrder := slices.EqualFunc(rows[scope].Items, list.Items, func(r home.PlanRow, p plan.PlanSummary) bool { return r.ID == p.ID })
			if got := rows[scope]; got.Total != list.Total || !inOrder {
				t.Errorf("%s, %s: %d plan rows of %d, Plans list %d of %d, or in another order", email, scope, len(got.Items), got.Total, len(list.Items), list.Total)
			}
		}
		for _, row := range rows["myTeams"].Items { // the same row under Everything, so only Everything's are compared with their pages
			if !slices.ContainsFunc(rows["everything"].Items, func(r home.PlanRow) bool { return reflect.DeepEqual(r, row) }) {
				t.Errorf("%s: My teams row %+v is not a row of Everything", email, row)
			}
		}
		for _, row := range rows["everything"].Items {
			page := get[plan.PlanView](t, client, "/api/v1/plans/"+row.ID, apitest.SkipValidation())
			fig := page.Figures
			if row.Status != string(page.Status) || row.Year != page.Year || row.StrategicAOP != page.AOP ||
				!same(row.Budget, fig.Budget) || !same(row.PlannedCost, fig.PlannedCost) ||
				!same(row.BudgetUsedPercent, fig.BudgetUsedPercent) || row.OverBudget != fig.OverBudget ||
				row.ActivityCount != fig.Activities || row.ApprovedActivityCount != fig.ActivitiesApproved ||
				row.CampaignCount != page.Contents.Campaigns || len(row.Teams) != len(page.Teams) {
				t.Errorf("%s: row %+v\nplan page %+v, contents %+v", email, row, fig, page.Contents)
			}
		}
	}
}

// User story 2 — See how my teams' plans are doing.

// Sara Novak (two teams) sees both FY2026 plans once each, with her teams marked and the figures of their
// plan pages; Everything adds FY2027 Global Brand, newest year first (FR-008 – FR-010, US2 scenarios 1, 4).
func TestHomePlans_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	sara := server.SignIn(t, sarasEmail)
	server.GrantRole(t, sarasEmail, "viewer", "all") // 014: every market's plans

	mine := get[home.PlanPage](t, sara, "/api/v1/home/plans?scope=myTeams")
	all := get[home.PlanPage](t, sara, "/api/v1/home/plans?scope=everything")

	names := func(page home.PlanPage) []string {
		list := []string{}
		for _, row := range page.Items {
			list = append(list, row.Name)
		}
		return list
	}
	if got := names(mine); !slices.Equal(got, []string{"FY2027 Lubricants UK & Ireland", "FY2026 Lubricants Europe", "FY2026 Mobility China"}) ||
		mine.Total != 3 {
		t.Errorf("My teams = %q of %d, want the GBP plan of 016 and both FY2026 plans", got, mine.Total)
	}
	if got := names(all); !slices.Equal(got, []string{"FY2027 Global Brand", "FY2027 Lubricants UK & Ireland", "FY2026 Lubricants Europe",
		"FY2026 Mobility China"}) || all.Total != 4 {
		t.Errorf("Everything = %q of %d", got, all.Total)
	}
	lubricants := mine.Items[1]
	if lubricants.Status != "published" || len(lubricants.Teams) != 1 || lubricants.Teams[0] != (home.TeamView{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue", Active: true, Mine: true}) ||
		lubricants.CampaignCount != 2 || lubricants.ActivityCount != 7 || lubricants.ApprovedActivityCount != 4 ||
		*lubricants.PlannedCost != usd(t, "460000.00") || *lubricants.BudgetUsedPercent != 38 || lubricants.OverBudget {
		t.Errorf("FY2026 Lubricants Europe = %+v", lubricants)
	}
	if uk := mine.Items[0]; uk.Budget == nil || *uk.Budget != usd(t, "759493.67") || uk.PlannedCost == nil ||
		*uk.PlannedCost != usd(t, "82278.48") || uk.BudgetUsedPercent == nil || *uk.BudgetUsedPercent != 11 {
		t.Errorf("FY2027 Lubricants UK & Ireland = %+v, want its GBP converted: 759,493.67 USD budget, 82,278.48 USD planned, 11 %%", uk)
	}
	if global := all.Items[0]; len(global.Teams) != 1 || global.Teams[0].Mine || !global.StrategicAOP {
		t.Errorf("FY2027 Global Brand = %+v, want an AOP plan of a team that is not Sara's", global)
	}
}

// User story 3 — Know what needs me.

// Priya's Declined activity comes first, with who declined it, when and why; her Launch film lists the
// fields it still needs. Alex sees his FY2027 campaign whose initiatives' budgets exceed its own (FR-011,
// FR-012, US3 scenarios 1 and 3).
func TestHomeAttention_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	alex := server.SignIn(t, alexsEmail)

	page := get[home.HomeAttentionPage](t, priya, "/api/v1/home/attention")
	alexs := get[home.HomeAttentionPage](t, alex, "/api/v1/home/attention")

	if len(page.Items) == 0 || page.Total != len(page.Items) {
		t.Fatalf("Priya's attention = %d items of %d", len(page.Items), page.Total)
	}
	first := page.Items[0]
	declined := first.Reasons[0]
	if first.ID != "A-1005" || first.Name != "Fleet roadshow" || first.RecordType != "activity" || declined.Kind != "declined" ||
		declined.DeclinedBy == nil || declined.DeclinedBy.Name != "Daniel Okoro" || declined.DeclinedAt == nil ||
		!declined.DeclinedAt.Equal(now.Add(-5*24*time.Hour)) || declined.Comment == nil ||
		*declined.Comment != "The planned cost is above this quarter's budget — please reduce it or move part of it to Q3." {
		t.Errorf("first item = %+v, reason %+v; want Fleet roadshow declined by Daniel Okoro 5 days ago", first, declined)
	}
	if len(first.Path) != 3 || first.Plan.Name != "FY2026 Lubricants Europe" || first.Path[2].Name != "Retail events" {
		t.Errorf("Fleet roadshow's place = %+v › %+v", first.Plan, first.Path)
	}
	var launchFilm *home.HomeAttentionItem
	for i := range page.Items {
		if page.Items[i].ID == "A-1001" {
			launchFilm = &page.Items[i]
		}
	}
	if launchFilm == nil {
		t.Fatalf("Launch film is not listed: %+v", page.Items)
	}
	var labels []string
	for _, m := range launchFilm.Reasons[len(launchFilm.Reasons)-1].Missing {
		labels = append(labels, m.Label)
	}
	if !slices.Equal(labels, []string{"Company code", "Cost centre", "PIR"}) {
		t.Errorf("Launch film needs %q, want Company code, Cost centre, PIR", labels)
	}
	var brandPlatform *home.HomeAttentionItem
	for i := range alexs.Items {
		if alexs.Items[i].Name == "Brand platform" {
			brandPlatform = &alexs.Items[i]
		}
	}
	if brandPlatform == nil || brandPlatform.RecordType != "campaign" || brandPlatform.Reasons[0].Kind != "overAllocated" ||
		brandPlatform.Reasons[0].Amount == nil || *brandPlatform.Reasons[0].Amount != usd(t, "50000.00") {
		t.Errorf("Alex's Brand platform = %+v, want over-allocated by 50,000.00 USD", brandPlatform)
	}
}

// Daniel's Waiting for my approval lists Trade magazine ads at Level 1 of 1, waiting since its
// resubmission; Alex's lists Fleet card launch at level 2 of 2; nothing waits for Priya (FR-013, US3
// scenarios 5 and 6).
func TestHomeWaiting_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	daniel := server.SignIn(t, danielsEmail)
	alex := server.SignIn(t, alexsEmail)
	priya := server.SignIn(t, priyasEmail)

	danielsPage := get[home.WaitingPage](t, daniel, "/api/v1/home/waiting")
	alexsPage := get[home.WaitingPage](t, alex, "/api/v1/home/waiting")
	priyasPage := get[home.WaitingPage](t, priya, "/api/v1/home/waiting")

	if danielsPage.Total != 1 || len(danielsPage.Items) != 1 {
		t.Fatalf("Daniel's page = %+v, want 1", danielsPage)
	}
	w := danielsPage.Items[0]
	if w.ID != "A-1004" || w.Name != "Trade magazine ads" || w.Plan.Name != "FY2026 Lubricants Europe" || w.Owner == nil ||
		w.Owner.Name != "Priya Raman" || w.PlannedCost == nil || *w.PlannedCost != usd(t, "30000.00") || w.Level != 1 ||
		w.Levels != 1 || !w.WaitingSince.Equal(now.Add(-48*time.Hour)) {
		t.Errorf("Daniel's waiting item = %+v", w)
	}
	if alexsPage.Total != 1 || alexsPage.Items[0].ID != "A-1009" || alexsPage.Items[0].Level != 2 || alexsPage.Items[0].Levels != 2 {
		t.Errorf("Alex's page = %+v, want Fleet card launch at level 2 of 2", alexsPage)
	}
	if priyasPage.Total != 0 || len(priyasPage.Items) != 0 {
		t.Errorf("Priya's page = %+v, want nothing", priyasPage)
	}
}

// User story 4 — Follow my activities.

// Priya's seeded activities each appear under the status their page shows, in the order Declined,
// Created, Submitted, Approved, In progress, Complete (FR-014, US4 scenario 1).
func TestHomeMyActivities_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	view := get[home.MyActivitiesView](t, priya, "/api/v1/home/my-activities")

	var groups []string
	for _, g := range view.Groups {
		var ids []string
		for _, row := range g.Items {
			ids = append(ids, row.ID)
		}
		groups = append(groups, g.Status+": "+strings.Join(ids, " "))
	}
	want := []string{"declined: A-1005", "created: A-1001 A-1011 A-1012", "submitted: A-1004", "approved: A-1006",
		"inProgress: A-1002 A-1007", "complete: A-1003"} // A-1011, A-1012: the GBP plan of 016
	if !slices.Equal(groups, want) || view.Total != 9 {
		t.Errorf("groups = %q (total %d)\nwant %q", groups, view.Total, want)
	}
	launchFilm, ukLaunch := view.Groups[1].Items[0], view.Groups[1].Items[1]
	if launchFilm.Plan.Name != "FY2026 Lubricants Europe" || launchFilm.Campaign.Name != "Helix launch" ||
		launchFilm.StartDate == nil || launchFilm.StartDate.String() != "2026-03-01" || launchFilm.EndDate.String() != "2026-05-31" ||
		launchFilm.PlannedCost == nil || *launchFilm.PlannedCost != usd(t, "120000.00") || len(launchFilm.Missing) != 3 ||
		ukLaunch.PlannedCost == nil || *ukLaunch.PlannedCost != usd(t, "50632.91") {
		t.Errorf("Launch film = %+v\nUK launch event = %+v, want its 40,000.00 GBP as 50,632.91 USD", launchFilm, ukLaunch)
	}
	if partner := view.Groups[4].Items[1]; !partner.Ongoing || partner.EndDate != nil {
		t.Errorf("Partner co-marketing fund = %+v, want ongoing without an end date", partner)
	}
}
