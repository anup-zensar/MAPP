package seed_test

import (
	"net/http"
	"testing"

	"mapp/internal/home"
	"mapp/internal/platform/apitest"
)

// The demo data holds the facts Home's story is told with (009 plan "Demo data"), read through the Home
// endpoints: each person's teams and their plans; FY2026 Lubricants Europe Published with a 1,200,000.00 USD
// budget; Priya's activity declined by Daniel Okoro with a comment; Alex Morgan's FY2027 campaign whose
// initiatives' budgets exceed its own; an activity waiting for Daniel Okoro at level 1 of 1.
func TestSeed_HomeStory_009(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	homeOf := func(email string) (home.SummaryView, home.HomeAttentionPage, home.WaitingPage) {
		t.Helper()
		client := server.SignIn(t, email)
		summary, attention, waiting := home.SummaryView{}, home.HomeAttentionPage{}, home.WaitingPage{}
		for path, v := range map[string]any{"/api/v1/home/summary?scope=myTeams": &summary, "/api/v1/home/attention": &attention,
			"/api/v1/home/waiting": &waiting} {
			response := client.Do(t, http.MethodGet, path, nil)
			if response.Status != http.StatusOK {
				t.Fatalf("%s: GET %s = %d %s", email, path, response.Status, response.Body)
			}
			response.Decode(t, v)
		}
		return summary, attention, waiting
	}

	// Lubricants Europe also has 016's GBP plan FY2027 Lubricants UK & Ireland.
	teamPlans := map[string]int{
		"priya.raman@shell.com": 2, "daniel.okoro@shell.com": 2, "sara.novak@shell.com": 3,
		"mei.lin@shell.com": 1, "carlos.mendes@shell.com": 2, "alex.morgan@shell.com": 1,
	}
	summaries, attention, waiting := map[string]home.SummaryView{}, map[string]home.HomeAttentionPage{}, map[string]home.WaitingPage{}
	for email, want := range teamPlans {
		summaries[email], attention[email], waiting[email] = homeOf(email)
		if summary := summaries[email]; summary.Plans.Count != want || summary.GettingStarted != nil {
			t.Errorf("%s: %d team plans (getting started %v), want %d", email, summary.Plans.Count, summary.GettingStarted, want)
		}
	}

	priya, priyasAttention := summaries["priya.raman@shell.com"], attention["priya.raman@shell.com"]
	// 1,200,000.00 USD and 600,000.00 GBP = 759,493.67 USD at 0.79, each plan converted to USD before adding (016 FR-014).
	if cost := priya.PlannedCost; priya.Plans.Published != 1 || cost.PlanBudget == nil || cost.PlanBudget.Format() != "1,959,493.67 USD" {
		t.Errorf("Priya's plans: %+v, budget %v; want FY2026 Lubricants Europe Published, 1,959,493.67 USD with the GBP plan", priya.Plans, cost.PlanBudget)
	}
	if len(priyasAttention.Items) == 0 || priyasAttention.Items[0].Reasons[0].Kind != "declined" ||
		priyasAttention.Items[0].Reasons[0].DeclinedBy.Name != "Daniel Okoro" || *priyasAttention.Items[0].Reasons[0].Comment == "" {
		t.Errorf("Priya's first item = %+v, want an activity declined by Daniel Okoro with a comment", priyasAttention.Items)
	}

	alexsAttention := attention["alex.morgan@shell.com"]
	overAllocated := false
	for _, item := range alexsAttention.Items {
		if item.RecordType == "campaign" && item.Plan.Name == "FY2027 Global Brand" && item.Reasons[0].Kind == "overAllocated" {
			overAllocated = true
		}
	}
	if !overAllocated {
		t.Errorf("Alex's items = %+v, want his over-allocated FY2027 Global Brand campaign", alexsAttention.Items)
	}

	danielsWaiting := waiting["daniel.okoro@shell.com"]
	if danielsWaiting.Total < 1 || danielsWaiting.Items[0].Level != 1 || danielsWaiting.Items[0].Levels != 1 {
		t.Errorf("waiting for Daniel = %+v, want an activity at level 1 of 1", danielsWaiting)
	}
}
