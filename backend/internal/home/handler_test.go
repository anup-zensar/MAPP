package home_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/activities"
	"northstar/internal/approvals"
	"northstar/internal/home"
	"northstar/internal/plans"
	"northstar/internal/platform/apitest"
	"northstar/internal/platform/money"
)

// The demo people's e-mail addresses (demo-users-and-data.md).
const (
	alexsEmail   = "alex.morgan@northstar.demo"
	priyasEmail  = "priya.raman@northstar.demo"
	danielsEmail = "daniel.okoro@northstar.demo"
	meisEmail    = "mei.lin@northstar.demo"
	carlosEmail  = "carlos.mendes@northstar.demo"
	sarasEmail   = "sara.novak@northstar.demo"
)

// everyone is every seeded person.
var everyone = []string{alexsEmail, priyasEmail, danielsEmail, meisEmail, carlosEmail, sarasEmail}

// now is the time of the test server's clock: 12 Mar 2026 09:00 UTC. The seeded approvals are dated before
// it.
var now = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// listScopes are Home's scopes with the matching scope of the Plans list (feature 004).
var listScopes = map[string]string{"myTeams": "myTeams", "everything": "all"}

// get sends GET path as client, checks that it answers 200 and decodes the answer into a T.
func get[T any](t *testing.T, client *apitest.Client, path string) T {
	t.Helper()
	response := client.Do(t, http.MethodGet, path, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s, want 200", path, response.Status, response.Body)
	}
	var v T
	response.Decode(t, &v)
	return v
}

// usd returns text as US dollars, such as "460000.00".
func usd(t *testing.T, text string) money.Money {
	t.Helper()
	amount, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return money.USD(amount)
}

// orZero is m, or 0.00 USD when there is none.
func orZero(m *money.Money) money.Money {
	if m == nil {
		return money.USD(money.Amount{})
	}
	return *m
}

// sameMoney reports whether two optional amounts are equal.
func sameMoney(x, y *money.Money) bool {
	return (x == nil && y == nil) || (x != nil && y != nil && *x == *y)
}

// User story 1 — See my work at a glance.

// Priya's cards with the seed: her team's one plan, Published; the seven activities she owns, one Declined
// and one Submitted; nothing waits for her; FY2026 Lubricants Europe's planned cost of its 1,200,000.00 USD
// budget. Daniel has Trade magazine ads waiting for him since its resubmission two days ago (FR-005, FR-006,
// US1 scenario 2).
func TestHomeSummary_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)

	got := get[home.SummaryView](t, priya, "/api/home/summary?scope=myTeams")
	everything := get[home.SummaryView](t, priya, "/api/home/summary?scope=everything")
	danielsCards := get[home.SummaryView](t, daniel, "/api/home/summary?scope=myTeams")

	if got.Scope != home.ScopeMyTeams || got.DisplayCurrency != "USD" {
		t.Errorf("scope %q, currency %q", got.Scope, got.DisplayCurrency)
	}
	if got.Plans != (home.PlanCounts{Count: 1, Published: 1}) || everything.Plans != (home.PlanCounts{Count: 3, Published: 1, Draft: 2}) {
		t.Errorf("plans = %+v (everything %+v); want 1 published (3: 1 published, 2 draft)", got.Plans, everything.Plans)
	}
	if got.ActivitiesOwned != (home.ActivityCounts{Count: 7, Declined: 1, Submitted: 1}) {
		t.Errorf("activities owned = %+v, want 7: 1 declined, 1 submitted", got.ActivitiesOwned)
	}
	if got.WaitingForMe != (home.WaitingCounts{}) {
		t.Errorf("waiting for Priya = %+v, want nothing", got.WaitingForMe)
	}
	cost := got.PlannedCost
	if cost.PlanCount != 1 || cost.PlannedCost != usd(t, "460000.00") || cost.PlanBudget == nil ||
		*cost.PlanBudget != usd(t, "1200000.00") || cost.BudgetUsedPercent == nil || *cost.BudgetUsedPercent != 38 ||
		cost.OverBudget || cost.PlansWithoutBudget != 0 {
		t.Errorf("planned cost = %+v, want 460,000.00 of 1,200,000.00 USD, 38 %%", cost)
	}
	w := danielsCards.WaitingForMe
	if w.Count != 1 || w.OldestWaitingSince == nil || !w.OldestWaitingSince.Equal(now.Add(-48*time.Hour)) {
		t.Errorf("waiting for Daniel = %+v, want Trade magazine ads, waiting 2 days", w)
	}
}

// User story 5 — Get started as a new user.

// A new person in no team gets the Getting started card with the no-team reason and empty cards; once they
// join a team that has a plan, the card is gone (FR-020, FR-021, US5 scenarios 1 and 2).
func TestHomeSummary_GettingStartedForANewPerson_FR020(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created := alex.Do(t, http.MethodPost, "/api/people", map[string]any{"name": "Nina Berg", "email": "nina.berg@northstar.demo"})
	var nina struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	created.Decode(t, &nina)
	if created.Status != http.StatusCreated && created.Status != http.StatusOK {
		t.Fatalf("POST /api/people = %d %s", created.Status, created.Body)
	}
	newcomer := server.SignIn(t, "nina.berg@northstar.demo")

	alone := get[home.SummaryView](t, newcomer, "/api/home/summary?scope=myTeams")
	joined := alex.Do(t, http.MethodPatch, "/api/people/"+nina.ID, map[string]any{"name": "Nina Berg",
		"email": "nina.berg@northstar.demo", "jobTitle": "", "teamIds": []string{"T-1003"}, "version": nina.Version})
	inTeam := get[home.SummaryView](t, newcomer, "/api/home/summary?scope=myTeams")

	if alone.GettingStarted == nil || *alone.GettingStarted != "notInTeam" {
		t.Errorf("getting started = %v, want notInTeam", alone.GettingStarted)
	}
	empty := home.PlannedCostTotal{PlannedCost: money.USD(money.Amount{})}
	if alone.Plans != (home.PlanCounts{}) || alone.ActivitiesOwned != (home.ActivityCounts{}) ||
		alone.WaitingForMe != (home.WaitingCounts{}) || alone.PlannedCost != empty {
		t.Errorf("cards = %+v, want every card empty", alone)
	}
	if joined.Status != http.StatusOK || inTeam.GettingStarted != nil || inTeam.Plans.Count != 1 {
		t.Errorf("after joining Global Marketing Ops (%d %s): getting started %v, plans %+v; want no card and FY2027 Global Brand",
			joined.Status, joined.Body, inTeam.GettingStarted, inTeam.Plans)
	}
}

// A scope other than myTeams or everything, or none, is refused on the field scope.
func TestHome_InvalidScope_400(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	refused := `{"code":"validation_failed","message":"Some fields need your attention.",` +
		`"fields":[{"field":"scope","message":"Choose My teams or Everything."}]}`

	for _, path := range []string{"/api/home/summary?scope=all", "/api/home/summary", "/api/home/plans?scope=mine", "/api/home/plans"} {
		response := priya.Do(t, http.MethodGet, path, nil, apitest.SkipRequestValidation()) // the contract allows only the two

		if response.Status != http.StatusBadRequest || string(response.Body) != refused {
			t.Errorf("GET %s = %d %s\nwant 400 %s", path, response.Status, response.Body, refused)
		}
	}
}

// For every seeded person and both scopes, Home's figures equal those of the Plans list and the Approvals
// page: the plans by status, the planned cost and plan budget of the plans in scope, and what waits for the
// person (SC-002, FR-017).
func TestHome_EqualsPlanAndApprovalPages_SC002_FR017(t *testing.T) {
	server := apitest.New(t)
	for _, email := range everyone {
		client := server.SignIn(t, email)
		counts := get[approvals.Counts](t, client, "/api/approvals/summary")
		for scope, listScope := range listScopes {
			summary := get[home.SummaryView](t, client, "/api/home/summary?scope="+scope)
			list := get[plans.PlanList](t, client, "/api/plans?scope="+listScope)

			if summary.Plans != (home.PlanCounts{Count: list.Total, Published: list.Summary.Published, Draft: list.Summary.Draft}) {
				t.Errorf("%s, %s: plans %+v, Plans list %d (%+v)", email, scope, summary.Plans, list.Total, list.Summary)
			}
			if summary.PlannedCost.PlannedCost != orZero(list.Summary.PlannedCost) ||
				!sameMoney(summary.PlannedCost.PlanBudget, list.Summary.Budget) {
				t.Errorf("%s, %s: planned cost %v of %v, Plans list %v of %v", email, scope, summary.PlannedCost.PlannedCost,
					summary.PlannedCost.PlanBudget, list.Summary.PlannedCost, list.Summary.Budget)
			}
			if summary.WaitingForMe.Count != counts.WaitingForMe {
				t.Errorf("%s: waiting for me %d, Approvals page %d", email, summary.WaitingForMe.Count, counts.WaitingForMe)
			}

			rows := get[home.PlanPage](t, client, "/api/home/plans?scope="+scope)
			if rows.Total != list.Total || len(rows.Items) != len(list.Items) {
				t.Errorf("%s, %s: %d plan rows of %d, Plans list %d of %d", email, scope, len(rows.Items), rows.Total, len(list.Items), list.Total)
			}
			for i, row := range rows.Items {
				page := get[plans.PlanView](t, client, "/api/plans/"+row.ID)
				fig := page.Figures
				if row.ID != list.Items[i].ID || row.Status != string(page.Status) || row.Year != page.Year || row.StrategicAOP != page.AOP ||
					!sameMoney(row.Budget, fig.Budget) || !sameMoney(row.PlannedCost, fig.PlannedCost) ||
					!samePercent(row.BudgetUsedPercent, fig.BudgetUsedPercent) || row.OverBudget != fig.OverBudget ||
					row.ActivityCount != fig.Activities || row.ApprovedActivityCount != fig.ActivitiesApproved ||
					row.CampaignCount != page.Contents.Campaigns || len(row.Teams) != len(page.Teams) {
					t.Errorf("%s, %s: row %+v\nplan page %+v, contents %+v", email, scope, row, fig, page.Contents)
				}
			}
		}
	}
}

// samePercent reports whether two optional percentages are equal.
func samePercent(x, y *int) bool {
	return (x == nil && y == nil) || (x != nil && y != nil && *x == *y)
}

// User story 2 — See how my teams' plans are doing.

// Sara Novak (two teams) sees both FY2026 plans once each, with her teams marked and the figures of their
// plan pages; Everything adds FY2027 Global Brand, newest year first (FR-008 – FR-010, US2 scenarios 1, 4).
func TestHomePlans_Contract(t *testing.T) {
	server := apitest.New(t)
	sara := server.SignIn(t, sarasEmail)

	mine := get[home.PlanPage](t, sara, "/api/home/plans?scope=myTeams")
	all := get[home.PlanPage](t, sara, "/api/home/plans?scope=everything")

	names := func(page home.PlanPage) []string {
		list := []string{}
		for _, row := range page.Items {
			list = append(list, row.Name)
		}
		return list
	}
	if got := names(mine); !slices.Equal(got, []string{"FY2026 Lubricants Europe", "FY2026 Mobility China"}) || mine.Total != 2 {
		t.Errorf("My teams = %q of %d, want both FY2026 plans", got, mine.Total)
	}
	if got := names(all); !slices.Equal(got, []string{"FY2027 Global Brand", "FY2026 Lubricants Europe", "FY2026 Mobility China"}) || all.Total != 3 {
		t.Errorf("Everything = %q of %d", got, all.Total)
	}
	lubricants := mine.Items[0]
	if lubricants.Status != "published" || len(lubricants.Teams) != 1 || lubricants.Teams[0] != (home.TeamView{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue", Active: true, Mine: true}) ||
		lubricants.CampaignCount != 2 || lubricants.ActivityCount != 7 || lubricants.ApprovedActivityCount != 4 ||
		*lubricants.PlannedCost != usd(t, "460000.00") || *lubricants.BudgetUsedPercent != 38 || lubricants.OverBudget {
		t.Errorf("FY2026 Lubricants Europe = %+v", lubricants)
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
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	alex := server.SignIn(t, alexsEmail)

	page := get[home.HomeAttentionPage](t, priya, "/api/home/attention")
	alexs := get[home.HomeAttentionPage](t, alex, "/api/home/attention")

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
	server := apitest.New(t)
	daniel := server.SignIn(t, danielsEmail)
	alex := server.SignIn(t, alexsEmail)
	priya := server.SignIn(t, priyasEmail)

	danielsPage := get[home.WaitingPage](t, daniel, "/api/home/waiting")
	alexsPage := get[home.WaitingPage](t, alex, "/api/home/waiting")
	priyasPage := get[home.WaitingPage](t, priya, "/api/home/waiting")

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

// For every seeded person, Waiting for my approval counts the Waiting for me tab of the Approvals page, and
// each activity that needs attention shows what its page shows: its fields still needed, its line items
// needing attention and how far its line items exceed its planned cost (SC-002, FR-017).
func TestHome_PersonalCardsEqualTheirPages_SC002_FR017(t *testing.T) {
	server := apitest.New(t)
	for _, email := range everyone {
		client := server.SignIn(t, email)
		waiting := get[home.WaitingPage](t, client, "/api/home/waiting")
		tab := get[approvals.RowPage](t, client, "/api/approvals?tab=waiting")
		if waiting.Total != tab.Total {
			t.Errorf("%s: %d waiting, Approvals tab %d", email, waiting.Total, tab.Total)
		}
		for i, item := range waiting.Items {
			if item.ID != tab.Items[i].Activity.ID {
				t.Errorf("%s: waiting row %d is %s, Approvals tab %s", email, i, item.ID, tab.Items[i].Activity.ID)
			}
		}

		attention := get[home.HomeAttentionPage](t, client, "/api/home/attention")
		for _, item := range attention.Items {
			if item.RecordType != "activity" {
				continue
			}
			page := get[activities.View](t, client, "/api/activities/"+item.ID)
			for _, reason := range item.Reasons {
				switch reason.Kind {
				case "missingFields":
					if len(reason.Missing) != page.Completeness.MissingFields || reason.ItemsBelow != page.Completeness.LineItemsNeedingAttention {
						t.Errorf("%s: %s needs %d fields and %d line items, its page %+v", email, item.ID, len(reason.Missing),
							reason.ItemsBelow, page.Completeness)
					}
				case "negativeBalance":
					if page.UnassignedPlannedCost == nil || reason.Amount.Amount != page.UnassignedPlannedCost.Amount.Neg() {
						t.Errorf("%s: %s exceeds by %v, its page %v", email, item.ID, reason.Amount, page.UnassignedPlannedCost)
					}
				}
			}
		}
	}
}

// User story 4 — Follow my activities.

// Priya's seeded activities each appear under the status their page shows, in the order Declined,
// Created, Submitted, Approved, In progress, Complete (FR-014, US4 scenario 1).
func TestHomeMyActivities_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	view := get[home.MyActivitiesView](t, priya, "/api/home/my-activities")

	var groups []string
	for _, g := range view.Groups {
		var ids []string
		for _, row := range g.Items {
			ids = append(ids, row.ID)
		}
		groups = append(groups, g.Status+": "+strings.Join(ids, " "))
	}
	want := []string{"declined: A-1005", "created: A-1001", "submitted: A-1004", "approved: A-1006",
		"inProgress: A-1002 A-1007", "complete: A-1003"}
	if !slices.Equal(groups, want) || view.Total != 7 {
		t.Errorf("groups = %q (total %d)\nwant %q", groups, view.Total, want)
	}
	launchFilm := view.Groups[1].Items[0]
	if launchFilm.Plan.Name != "FY2026 Lubricants Europe" || launchFilm.Campaign.Name != "Helix launch" ||
		launchFilm.StartDate == nil || launchFilm.StartDate.String() != "2026-03-01" || launchFilm.EndDate.String() != "2026-05-31" ||
		launchFilm.PlannedCost == nil || *launchFilm.PlannedCost != usd(t, "120000.00") || len(launchFilm.Missing) != 3 {
		t.Errorf("Launch film = %+v", launchFilm)
	}
	if partner := view.Groups[4].Items[1]; !partner.Ongoing || partner.EndDate != nil {
		t.Errorf("Partner co-marketing fund = %+v, want ongoing without an end date", partner)
	}
}

// For every seeded person, each of their activities shows on Home the status, planned cost and completeness
// of its own page (SC-002, FR-017).
func TestHome_MyActivitiesEqualTheirPages_SC002_FR017(t *testing.T) {
	server := apitest.New(t)
	for _, email := range everyone {
		client := server.SignIn(t, email)
		view := get[home.MyActivitiesView](t, client, "/api/home/my-activities")
		summary := get[home.SummaryView](t, client, "/api/home/summary?scope=myTeams")
		if view.Total != summary.ActivitiesOwned.Count {
			t.Errorf("%s: My activities %d, card %d", email, view.Total, summary.ActivitiesOwned.Count)
		}
		for _, g := range view.Groups {
			for _, row := range g.Items {
				page := get[activities.View](t, client, "/api/activities/"+row.ID)
				if row.Status != string(page.DisplayStatus) || !sameMoney(row.PlannedCost, page.PlannedCost) ||
					len(row.Missing) != page.Completeness.MissingFields || row.ItemsBelow != page.Completeness.LineItemsNeedingAttention {
					t.Errorf("%s: %+v\nits page: %s, %v, %+v", email, row, page.DisplayStatus, page.PlannedCost, page.Completeness)
				}
			}
		}
	}
}

// A deleted record never counts: once Priya deletes Launch film, it leaves her cards, My activities, Needs
// your attention and her plan's row at once (FR-019).
func TestHome_DeletedRecordsNeverCount_FR019(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	deleted := priya.Do(t, http.MethodDelete, "/api/activities/A-1001", nil)
	if deleted.Status != http.StatusOK {
		t.Fatalf("DELETE /api/activities/A-1001 = %d %s", deleted.Status, deleted.Body)
	}
	summary := get[home.SummaryView](t, priya, "/api/home/summary?scope=myTeams")
	view := get[home.MyActivitiesView](t, priya, "/api/home/my-activities")
	attention := get[home.HomeAttentionPage](t, priya, "/api/home/attention")
	rows := get[home.PlanPage](t, priya, "/api/home/plans?scope=myTeams")

	if summary.ActivitiesOwned.Count != 6 || view.Total != 6 || summary.PlannedCost.PlannedCost != usd(t, "340000.00") {
		t.Errorf("after the delete: %d owned, %d in My activities, planned %v; want 6, 6 and 340,000.00 USD",
			summary.ActivitiesOwned.Count, view.Total, summary.PlannedCost.PlannedCost)
	}
	for _, item := range attention.Items {
		if item.ID == "A-1001" {
			t.Errorf("Needs your attention still lists the deleted Launch film: %+v", item)
		}
	}
	if len(rows.Items) != 1 || rows.Items[0].ActivityCount != 6 || rows.Items[0].PlannedCost == nil ||
		*rows.Items[0].PlannedCost != usd(t, "340000.00") {
		t.Errorf("plan row = %+v, want 6 activities and 340,000.00 USD", rows.Items)
	}
}
