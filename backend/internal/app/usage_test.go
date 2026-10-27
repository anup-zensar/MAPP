package app_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"mapp/internal/people"
	"mapp/internal/people/peopletest"
	"mapp/internal/platform/apitest"
)

// A team used by a plan cannot be deleted, also while the plan is in the bin; one no plan uses can be.
func TestTeamUsedByPlans_InUseDelete_FR014(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all")
	type team struct {
		Plans []struct {
			ID, Name     string
			InRecycleBin bool
		}
	}
	inUse := `{"error":{"code":"in_use","message":"Mobility China is used by 1 plan, so it can't be deleted. Set it inactive instead.","fields":[]}}`

	var before team
	priya.Do(t, http.MethodGet, "/api/v1/teams/T-1002", nil).Decode(t, &before)
	refused := priya.Do(t, http.MethodDelete, "/api/v1/teams/T-1002", nil)
	deletePlan(t, priya, "P-1002")
	var after team
	priya.Do(t, http.MethodGet, "/api/v1/teams/T-1002", nil).Decode(t, &after)
	stillRefused := priya.Do(t, http.MethodDelete, "/api/v1/teams/T-1002", nil)

	if len(before.Plans) != 1 || before.Plans[0].ID != "P-1002" || before.Plans[0].Name != "FY2026 Mobility China" || before.Plans[0].InRecycleBin {
		t.Errorf("plans of T-1002 = %+v, want FY2026 Mobility China", before.Plans)
	}
	if len(after.Plans) != 1 || after.Plans[0].ID != "P-1002" || !after.Plans[0].InRecycleBin {
		t.Errorf("plans of T-1002 after deleting P-1002 = %+v, want it in the recycle bin", after.Plans)
	}
	for name, response := range map[string]*apitest.Response{"stored plan": refused, "plan in the recycle bin": stillRefused} {
		if response.Status != http.StatusConflict || string(response.Body) != inUse {
			t.Errorf("%s: DELETE T-1002 = %d %s\nwant 409 %s", name, response.Status, response.Body, inUse)
		}
	}

	var created struct{ ID string }
	priya.Do(t, http.MethodPost, "/api/v1/teams", map[string]any{"name": "Pop-up events", "description": "", "colour": "", "memberIds": []string{}}).Decode(t, &created)
	if deleted := priya.Do(t, http.MethodDelete, "/api/v1/teams/"+created.ID, nil); deleted.Status != http.StatusOK {
		t.Errorf("DELETE an unused team = %d %s, want 200", deleted.Status, deleted.Body)
	}
}

// Records in the recycle bin are left out.
func TestPersonUsage_EveryFeature_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all")

	mei := personUsage(t, priya, "U-1004")
	carlos := personUsage(t, priya, "U-1005")
	deletePlan(t, priya, "P-1002")
	meiAfter := personUsage(t, priya, "U-1004")

	wantMei := []usageCount{
		{Role: "owner", RecordType: "plan", Label: "plan", Count: 1},
		{Role: "owner", RecordType: "campaign", Label: "campaign", Count: 1},
		{Role: "owner", RecordType: "initiative", Label: "initiatives", Count: 2},
		{Role: "owner", RecordType: "activity", Label: "activities", Count: 2},
		{Role: "owner", RecordType: "lineItem", Label: "line items", Count: 2},
	}
	if !reflect.DeepEqual(mei, wantMei) {
		t.Errorf("Mei Lin's usage =\n%+v\nwant\n%+v", mei, wantMei)
	}
	wantCarlos := []usageCount{ // 017 FR-016: and line manager of Mei Lin
		{Role: "approver", RecordType: "approvalWorkflow", Label: "approval workflow", Count: 1},
		{Role: "manager", RecordType: "person", Label: "person", Count: 1},
	}
	if !reflect.DeepEqual(carlos, wantCarlos) {
		t.Errorf("Carlos Mendes' usage = %+v, want %+v", carlos, wantCarlos)
	}
	if len(meiAfter) != 0 {
		t.Errorf("Mei Lin's usage with her plan in the recycle bin = %+v, want nothing", meiAfter)
	}
}

// Plan owners who would no longer belong to their plans' teams are named (warning ownerNeedsAttention).
func TestImpact_OwnerNeedsAttention_FR013(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all")
	type warning struct {
		Kind, Message string
		Count         int
	}
	ownerWarning := func(uses string) []warning {
		return []warning{{Kind: "ownerNeedsAttention", Count: 1, Message: "Mei Lin owns 1 plan that " + uses +
			": FY2026 Mobility China. Mei will no longer belong to its teams, so its owner will need attention."}}
	}

	var person, team struct{ Warnings []warning }
	priya.Do(t, http.MethodPost, "/api/v1/people/U-1004/impact", map[string]any{
		"name": "Mei Lin", "email": "mei.lin@shell.com", "jobTitle": "Marketer", "teamIds": []string{}, "version": 1,
	}).Decode(t, &person)
	priya.Do(t, http.MethodPost, "/api/v1/teams/T-1002/impact", map[string]any{
		"name": "Mobility China", "description": "Mobility and retail marketing in China.", "colour": "violet", "memberIds": []string{"U-1005", "U-1006"}, "version": 1,
	}).Decode(t, &team)

	if want := ownerWarning("uses Mobility China"); !reflect.DeepEqual(person.Warnings, want) {
		t.Errorf("person impact = %+v\nwant %+v", person.Warnings, want)
	}
	if want := ownerWarning("uses this team"); !reflect.DeepEqual(team.Warnings, want) {
		t.Errorf("team impact = %+v\nwant %+v", team.Warnings, want)
	}
}

// Settings counts the people who show amounts in a currency, and a person showing EUR sees the plans'
// amounts in euros. The preference is stored before the API starts (D-061).
func TestDisplayCurrencies_SettingsAndPeople_FR003_D061(t *testing.T) {
	t.Parallel()
	db := apitest.DemoDB(t, false)
	err := db.Write(t.Context(), func(ctx context.Context) error {
		return peopletest.NewStore(db).PutPreferences(ctx, people.Preferences{PersonID: "U-1004", Theme: "system", DisplayCurrency: "EUR"})
	})
	if err != nil {
		t.Fatalf("saving Mei Lin's display currency: %v", err)
	}
	server := apitest.New(t, apitest.WithDB(db))
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "globalPlanner", "all")
	mei := server.SignIn(t, "mei.lin@shell.com")
	server.GrantRole(t, "mei.lin@shell.com", "globalPlanner", "all")

	got := summaries(t, priya, []string{"/api/v1/currencies/USD", "/api/v1/currencies/EUR"})
	budgetCurrency := func(c *apitest.Client) string {
		t.Helper()
		var plan struct { // the stored budget is USD; the figures are in the person's display currency
			Figures struct {
				Budget *struct{ Amount, Currency string }
			}
		}
		c.Do(t, http.MethodGet, "/api/v1/plans/P-1002", nil).Decode(t, &plan)
		if plan.Figures.Budget == nil {
			t.Fatal("plan P-1002 has no budget")
		}
		return plan.Figures.Budget.Currency
	}

	want := map[string]string{
		"/api/v1/currencies/USD": "Used by 5 line items (1 in the recycle bin) · Display currency of 5 people · Plan currency of 3 plans",
		"/api/v1/currencies/EUR": "Used by 3 line items · Display currency of 1 person",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Used by =\n%v\nwant\n%v", got, want)
	}
	if mine, priyas := budgetCurrency(mei), budgetCurrency(priya); mine != "EUR" || priyas != "USD" {
		t.Errorf("budget figure of P-1002 in %s for Mei Lin and %s for Priya Raman, want EUR and USD", mine, priyas)
	}
}
