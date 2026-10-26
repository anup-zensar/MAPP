package app_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"mapp/internal/people"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/store"
	"mapp/internal/stores"
	"mapp/seed"
)

// The tests below check the wiring of app.New across features (references.go): Settings (002) sees the
// people, plans, campaigns, initiatives, activities and line items that use its records, and people (003)
// sees the plans, campaigns, initiatives, activities, line items and approval workflows that name a person
// or use a team — stored and in the recycle bin, as each feature counts them. Every answer is checked
// against the contract by apitest. The demo data: plan P-1002 "FY2026 Mobility China" (model MOD-1002,
// team T-1002 Mobility China, finance budget FB-1002) is owned by Mei Lin, who also owns its campaign, two
// initiatives, two of its three activities and their line items. The demo history (011) keeps one entry in
// the recycle bin: initiative "Trade shows" with an activity and a USD line item, which "Used by" counts.

// usage is the "Used by" of a Settings record (Usage).
type usage struct {
	InUse   bool
	Summary string
	Records []recordRef
}

// recordRef is a record as Usage.records lists it.
type recordRef struct {
	Type, ID, Name string
}

// usageCount is one count of a person's details (UsageCount).
type usageCount struct {
	Role, RecordType, Label string
	Count                   int
}

// usageAt returns the usage of the Settings record that GET path answers.
func usageAt(t *testing.T, c *apitest.Client, path string) usage {
	t.Helper()
	response := c.Do(t, http.MethodGet, path, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s, want 200", path, response.Status, response.Body)
	}
	var record struct{ Usage usage }
	response.Decode(t, &record)
	return record.Usage
}

// summaries returns the "Used by" summary of each Settings record at its path.
func summaries(t *testing.T, c *apitest.Client, paths []string) map[string]string {
	t.Helper()
	got := map[string]string{}
	for _, path := range paths {
		got[path] = usageAt(t, c, path).Summary
	}
	return got
}

// personUsage returns what the person personID owns and approves, from their details.
func personUsage(t *testing.T, c *apitest.Client, personID string) []usageCount {
	t.Helper()
	var person struct{ Usage []usageCount }
	c.Do(t, http.MethodGet, "/api/v1/people/"+personID, nil).Decode(t, &person)
	return person.Usage
}

// deletePlan deletes the plan id and returns its recycle-bin entry's id.
func deletePlan(t *testing.T, c *apitest.Client, id string) string {
	t.Helper()
	response := c.Do(t, http.MethodDelete, "/api/v1/plans/"+id, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("DELETE plan %s = %d %s, want 200", id, response.Status, response.Body)
	}
	var deleted struct{ RecycleBinEntry struct{ ID string } }
	response.Decode(t, &deleted)
	return deleted.RecycleBinEntry.ID
}

// settingsPaths are the Settings records whose "Used by" the tests below read: one of every kind that
// features 003–007 use.
var settingsPaths = []string{
	"/api/v1/models/MOD-1002",               // plans (004)
	"/api/v1/activity-types/AT-1006",        // activities (006)
	"/api/v1/shell-attributes/SA-1001",      // plan values (004)
	"/api/v1/shell-attributes/SA-1003",      // campaign values (005)
	"/api/v1/shell-attributes/SA-1011",      // initiative values (005)
	"/api/v1/shell-attributes/SA-1012",      // activity values (006)
	"/api/v1/shell-attributes/SA-1033",      // line-item values (007)
	"/api/v1/currencies/USD",                // line items (007) and display currencies (003)
	"/api/v1/currencies/CNY",                // line items (007)
	"/api/v1/finance-budget-items/FBI-1001", // line items (007)
}

// Settings' "Used by" counts the records of every later feature that use a Settings record (002 FR-003,
// D-042), and its records list the plans using a model.
func TestUsedBy_SeesEveryFeature_FR003(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")

	got := summaries(t, priya, settingsPaths)

	want := map[string]string{
		"/api/v1/models/MOD-1002":               "Used by 1 plan",
		"/api/v1/activity-types/AT-1006":        "Used by 1 activity · Allowed in 1 model",
		"/api/v1/shell-attributes/SA-1001":      "Used by 3 plans · Condition of 11 attributes",
		"/api/v1/shell-attributes/SA-1003":      "Used by 4 campaigns · Region depends on it",
		"/api/v1/shell-attributes/SA-1011":      "Used by 11 initiatives (1 in the recycle bin)",
		"/api/v1/shell-attributes/SA-1012":      "Used by 12 activities (1 in the recycle bin)",
		"/api/v1/shell-attributes/SA-1033":      "Used by 9 line items (1 in the recycle bin)",
		"/api/v1/currencies/USD":                "Used by 5 line items (1 in the recycle bin) · Display currency of 6 people · Plan currency of 3 plans",
		"/api/v1/currencies/CNY":                "Used by 1 line item",
		"/api/v1/finance-budget-items/FBI-1001": "Linked to 2 line items",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Used by =\n%v\nwant\n%v", got, want)
	}
	model := usageAt(t, priya, "/api/v1/models/MOD-1002")
	if want := []recordRef{{Type: "plan", ID: "P-1002", Name: "FY2026 Mobility China"}}; !reflect.DeepEqual(model.Records, want) {
		t.Errorf("records of MOD-1002 = %+v, want %+v", model.Records, want)
	}
}

// Records in the recycle bin still count in "Used by", marked as such, so a Settings record they use
// cannot be deleted and a restore never meets a missing setting; the plans listed are the live ones only
// (002 FR-003, data model §10). Here a whole plan and, from another plan, one campaign with everything under
// it are in the bin. Undo brings the counts back.
func TestUsedBy_CountsTheRecycleBin_FR003(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all") // 014: Settings, people and every plan
	before := summaries(t, priya, settingsPaths)

	planEntry := deletePlan(t, priya, "P-1002") // with its campaign, initiatives, activities and line items
	deleted := priya.Do(t, http.MethodDelete, "/api/v1/campaigns/C-1002", nil)
	if deleted.Status != http.StatusOK {
		t.Fatalf("DELETE campaign C-1002 = %d %s, want 200", deleted.Status, deleted.Body)
	}
	var campaignEntry struct{ RecycleBinEntry struct{ ID string } }
	deleted.Decode(t, &campaignEntry)

	got := summaries(t, priya, settingsPaths)
	want := map[string]string{
		"/api/v1/models/MOD-1002":               "Used by 1 plan (1 in the recycle bin)",
		"/api/v1/activity-types/AT-1006":        "Used by 1 activity (1 in the recycle bin) · Allowed in 1 model",
		"/api/v1/shell-attributes/SA-1001":      "Used by 3 plans · Condition of 11 attributes", // P-1002 has no business unit
		"/api/v1/shell-attributes/SA-1003":      "Used by 4 campaigns (1 in the recycle bin) · Region depends on it",
		"/api/v1/shell-attributes/SA-1011":      "Used by 11 initiatives (5 in the recycle bin)",
		"/api/v1/shell-attributes/SA-1012":      "Used by 12 activities (6 in the recycle bin)",
		"/api/v1/shell-attributes/SA-1033":      "Used by 9 line items (2 in the recycle bin)",
		"/api/v1/currencies/USD":                "Used by 5 line items (2 in the recycle bin) · Display currency of 6 people · Plan currency of 3 plans (1 in the recycle bin)",
		"/api/v1/currencies/CNY":                "Used by 1 line item (1 in the recycle bin)",
		"/api/v1/finance-budget-items/FBI-1001": "Linked to 2 line items",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Used by with P-1002 and C-1002 in the recycle bin =\n%v\nwant\n%v", got, want)
	}
	if model := usageAt(t, priya, "/api/v1/models/MOD-1002"); len(model.Records) != 0 || !model.InUse {
		t.Errorf("MOD-1002 = %+v, want in use with no live plan listed", model)
	}
	refused := priya.Do(t, http.MethodDelete, "/api/v1/models/MOD-1002", nil)
	if refused.Status != http.StatusConflict {
		t.Errorf("DELETE MOD-1002 while its plan is in the recycle bin = %d %s, want 409 in_use", refused.Status, refused.Body)
	}

	for _, entryID := range []string{campaignEntry.RecycleBinEntry.ID, planEntry} {
		if undone := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+entryID+"/undo", nil); undone.Status != http.StatusOK {
			t.Fatalf("undo %s = %d %s, want 200", entryID, undone.Status, undone.Body)
		}
	}

	if after := summaries(t, priya, settingsPaths); !reflect.DeepEqual(after, before) {
		t.Errorf("Used by after the Undo =\n%v\nwant the counts from before the deletes\n%v", after, before)
	}
}

// A Settings record in use cannot be deleted (002 FR-004): a currency of a line item, a model of a plan.
func TestInUseDelete_RefusedForEveryFeature_FR004(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all") // 014: Settings, people and every plan
	want := map[string]string{
		"/api/v1/currencies/GBP":  `{"error":{"code":"in_use","message":"'GBP' is in use (used by 2 line items · Plan currency of 1 plan) — set it inactive instead.","fields":[]}}`,
		"/api/v1/models/MOD-1002": `{"error":{"code":"in_use","message":"'Mobility 1.0' is in use (used by 1 plan) — set it inactive instead.","fields":[]}}`,
	}

	for path, body := range want {
		response := priya.Do(t, http.MethodDelete, path, nil)

		if response.Status != http.StatusConflict || string(response.Body) != body {
			t.Errorf("DELETE %s = %d %s\nwant 409 %s", path, response.Status, response.Body, body)
		}
	}
}

// Impact statements count the live records of the later features (002 FR-011, FR-030): the plans and
// activities a model's new rates recalculate, the line items and activities a currency's new rate
// recalculates.
func TestImpact_CountsTheLiveRecords_FR011_FR030(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all") // 014: Settings, people and every plan
	type warning struct {
		Kind, Message string
		Count         int
	}
	impact := func(path string, body map[string]any) []warning {
		t.Helper()
		var answer struct{ Warnings []warning }
		response := priya.Do(t, http.MethodPost, path, body)
		if response.Status != http.StatusOK {
			t.Fatalf("POST %s = %d %s, want 200", path, response.Status, response.Body)
		}
		response.Decode(t, &answer)
		return answer.Warnings
	}

	model := impact("/api/v1/models/MOD-1001/impact", map[string]any{"inqToMqlRate": "3.00", "version": 1})
	currency := impact("/api/v1/currencies/EUR/impact", map[string]any{"rate": "0.80", "version": 1})

	wantModel := []warning{{Kind: "goalsRecalculated", Count: 3, // with P-1004 of 016
		Message: "Used by 3 plans (9 activities). Their calculated MQL and revenue goals will be recalculated; typed goals ('Set later stages myself') are kept."}}
	if !reflect.DeepEqual(model, wantModel) {
		t.Errorf("model impact = %+v\nwant %+v", model, wantModel)
	}
	wantCurrency := []warning{{Kind: "amountsRecalculated", Count: 3, // with L-1010 of 016's GBP plan: 23,000.00 EUR
		Message: "3 line items in 3 activities use EUR. Their EUR amounts stay the same; their USD amounts will be recalculated (85,000.00 → 97,750.00 USD). Unassigned planned cost of these activities will change. No activity goes back for approval."}}
	if !reflect.DeepEqual(currency, wantCurrency) {
		t.Errorf("currency impact = %+v\nwant %+v", currency, wantCurrency)
	}
}

// A team used by a plan cannot be deleted, also while the plan is in the recycle bin, and its details list
// the plans with their recycle-bin state (003 FR-014, through 004's plans). A team no plan uses can be.
func TestTeamUsedByPlans_InUseDelete_FR014(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all") // 014: Settings, people and every plan
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

// A person's details count what they own and approve in every feature — plans, campaigns, initiatives,
// activities, line items and approval workflows — leaving out records in the recycle bin (003 FR-004).
func TestPersonUsage_EveryFeature_FR004(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all") // 014: Settings, people and every plan

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

// Changing a person's teams or a team's members states which plan owners would no longer belong to their
// plans' teams (003 FR-013), with the warning kind ownerNeedsAttention of the contract.
func TestImpact_OwnerNeedsAttention_FR013(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all") // 014: Settings, people and every plan
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

// Display currencies are wired both ways (002 FR-003, D-061): Settings counts the people who show
// amounts in a currency, and people take the currencies and their rates from Settings, so a person whose
// display currency is EUR sees the plans' amounts in euros. My profile, where a person chooses it, comes
// later (003 US4), so the test stores the preference in the data folder before the API starts.
func TestDisplayCurrencies_SettingsAndPeople_FR003_D061(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	now := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	if err = seed.Load(t.Context(), stores.JSON(db), now); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}
	err = db.Write(t.Context(), func(ctx context.Context) error {
		return people.NewJSONRepository().PutPreferences(ctx, people.Preferences{PersonID: "U-1004", Theme: "system", DisplayCurrency: "EUR"})
	})
	if err != nil {
		t.Fatalf("saving Mei Lin's display currency: %v", err)
	}
	server := apitest.New(t, apitest.WithDataDir(dir))
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "globalPlanner", "all") // 014: every market's plans
	mei := server.SignIn(t, "mei.lin@shell.com")
	server.GrantRole(t, "mei.lin@shell.com", "globalPlanner", "all") // 014: every market's plans

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

// Before a plan's model changes, the plan lists the activities whose type the new model does not allow,
// from activities (006) through the plans' level registry (004 FR-019): Lubricants 1.0 does not allow
// MOBILITY – Long-term contract, the type of Fleet card launch; Mobility 1.0 allows every type.
func TestPlanChangeEffects_DisallowedActivities_FR019(t *testing.T) {
	server := apitest.New(t)
	mei := server.SignIn(t, "mei.lin@shell.com")
	server.GrantRole(t, "mei.lin@shell.com", "globalPlanner", "all") // 014: every market's plans
	type effects struct {
		ModelChanged         bool
		DisallowedActivities []recordRef
	}
	changeEffects := func(planID, modelID string) effects {
		t.Helper()
		response := mei.Do(t, http.MethodPost, "/api/v1/plans/"+planID+"/change-effects",
			map[string]any{"modelId": modelID, "attributes": map[string]any{}})
		if response.Status != http.StatusOK {
			t.Fatalf("change effects of %s = %d %s, want 200", planID, response.Status, response.Body)
		}
		var e effects
		response.Decode(t, &e)
		return e
	}

	toLubricants := changeEffects("P-1002", "MOD-1001")
	toMobility := changeEffects("P-1001", "MOD-1002")

	want := effects{ModelChanged: true, DisallowedActivities: []recordRef{{Type: "activity", ID: "A-1009", Name: "Fleet card launch"}}}
	if !reflect.DeepEqual(toLubricants, want) {
		t.Errorf("P-1002 to Lubricants 1.0 = %+v, want %+v", toLubricants, want)
	}
	if !toMobility.ModelChanged || len(toMobility.DisallowedActivities) != 0 {
		t.Errorf("P-1001 to Mobility 1.0 = %+v, want a model change with no activity disallowed", toMobility)
	}
}
