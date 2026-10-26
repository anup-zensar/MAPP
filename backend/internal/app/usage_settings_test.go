package app_test

import (
	"net/http"
	"reflect"
	"testing"

	"mapp/internal/platform/apitest"
)

// Its records list the plans using a model (D-042).
func TestUsedBy_SeesEveryFeature_FR003(t *testing.T) {
	t.Parallel()
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

// Bin records still count, so a restore never meets a missing setting; the plans listed are the live ones.
// A whole plan and one campaign of another plan are in the bin; Undo brings the counts back.
func TestUsedBy_CountsTheRecycleBin_FR003(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all")
	before := summaries(t, priya, settingsPaths)

	planEntry := deletePlan(t, priya, "P-1002")
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

func TestInUseDelete_RefusedForEveryFeature_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all")
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

// The plans and activities a model's new rates recalculate, the line items and activities a currency's.
func TestImpact_CountsTheLiveRecords_FR011_FR030(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	server.GrantRole(t, "priya.raman@shell.com", "administrator", "all")
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

// Lubricants 1.0 does not allow the type of Fleet card launch; Mobility 1.0 allows every type.
func TestPlanChangeEffects_DisallowedActivities_FR019(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	mei := server.SignIn(t, "mei.lin@shell.com")
	server.GrantRole(t, "mei.lin@shell.com", "globalPlanner", "all")
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
