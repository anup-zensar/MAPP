package campaigns_test

import (
	"net/http"
	"slices"
	"testing"

	"mapp/internal/platform/apitest"
)

// The demo people who sign in.
const (
	priyasEmail  = "priya.raman@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
	meiLinsEmail = "mei.lin@shell.com"
)

// The answers of the "no longer exists" cases (plan's message table).
const (
	planGoneBody       = `{"error":{"code":"not_found","message":"This plan no longer exists — someone may have deleted it.","fields":[]}}`
	campaignGoneBody   = `{"error":{"code":"not_found","message":"This campaign no longer exists — someone may have deleted it.","fields":[]}}`
	initiativeGoneBody = `{"error":{"code":"not_found","message":"This initiative no longer exists — someone may have deleted it.","fields":[]}}`
)

// amount is money as the API answers it.
type amount struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// initiative is the part of an Initiative answer these tests read.
type initiative struct {
	ID                    string  `json:"id"`
	PlanID                string  `json:"planId"`
	CampaignID            string  `json:"campaignId"`
	Name                  string  `json:"name"`
	StartDate             *string `json:"startDate"`
	EndDate               *string `json:"endDate"`
	Budget                *amount `json:"budget"`
	ParentRemainingBudget *amount `json:"parentRemainingBudget"`
	Timing                *string `json:"timing"`
	Version               int     `json:"version"`
	UpdatedBy             struct {
		Name string `json:"name"`
	} `json:"updatedBy"`
}

// campaign is the part of a Campaign answer these tests read.
type campaign struct {
	ID                    string  `json:"id"`
	PlanID                string  `json:"planId"`
	Name                  string  `json:"name"`
	OwnerID               *string `json:"ownerId"`
	Budget                *amount `json:"budget"`
	OutsidePlanYear       bool    `json:"outsidePlanYear"`
	ParentRemainingBudget *amount `json:"parentRemainingBudget"`
	BudgetAndCost         struct {
		AllocatedToChildren  amount  `json:"allocatedToChildren"`
		RemainingForChildren *amount `json:"remainingForChildren"`
		OverAllocatedBy      *amount `json:"overAllocatedBy"`
	} `json:"budgetAndCost"`
	Completeness struct {
		Missing []struct {
			Field string `json:"field"`
			Label string `json:"label"`
		} `json:"missing"`
		ItemsBelowNeedingAttention int `json:"itemsBelowNeedingAttention"`
	} `json:"completeness"`
	Counts struct {
		Initiatives int `json:"initiatives"`
	} `json:"counts"`
	Initiatives []initiative `json:"initiatives"`
	Version     int          `json:"version"`
	UpdatedBy   struct {
		Name string `json:"name"`
	} `json:"updatedBy"`
}

// planCampaigns is a PlanCampaigns answer.
type planCampaigns struct {
	Plan struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Year int    `json:"year"`
	} `json:"plan"`
	PlanOwner *struct {
		Name string `json:"name"`
	} `json:"planOwner"`
	Items  []campaign `json:"items"`
	Total  int        `json:"total"`
	Totals struct {
		Campaigns            int     `json:"campaigns"`
		Initiatives          int     `json:"initiatives"`
		Activities           int     `json:"activities"`
		AllocatedToCampaigns amount  `json:"allocatedToCampaigns"`
		PlanBudget           *amount `json:"planBudget"`
		RemainingPlanBudget  *amount `json:"remainingPlanBudget"`
		OverAllocatedBy      *amount `json:"overAllocatedBy"`
		PlannedCost          *amount `json:"plannedCost"`
	} `json:"totals"`
}

// assertStatus fails the test unless the answer has status want.
func assertStatus(t *testing.T, r *apitest.Response, want int) {
	t.Helper()
	if r.Status != want {
		t.Fatalf("status = %d %s, want %d", r.Status, r.Body, want)
	}
}

// assertBody fails the test unless the answer is status with exactly body.
func assertBody(t *testing.T, r *apitest.Response, status int, body string) {
	t.Helper()
	if r.Status != status || string(r.Body) != body {
		t.Errorf("answer = %d %s\nwant %d %s", r.Status, r.Body, status, body)
	}
}

// validationBody is the answer of validation_failed with the given field problems (JSON objects).
func validationBody(fields ...string) string {
	body := `{"error":{"code":"validation_failed","message":"Some fields need your attention.","fields":[`
	for i, f := range fields {
		if i > 0 {
			body += ","
		}
		body += f
	}
	return body + "]}}"
}

// usd is a USD amount as the API answers it.
func usd(text string) amount { return amount{Amount: text, Currency: "USD"} }

// The plan tree of FY2026 Lubricants Europe: its 2 campaigns with their 4 initiatives in tree order, the
// plan owner as the default owner, and the totals row — allocated 1,150,000.00 of 1,200,000.00, and the 7
// activities of 006's demo data with 460,000.00 planned (FR-019 – FR-022).
func TestListPlanCampaigns_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001/campaigns", nil)

	assertStatus(t, response, http.StatusOK)
	var tree planCampaigns
	response.Decode(t, &tree)
	var names []string
	for _, c := range tree.Items {
		names = append(names, c.Name)
		for _, i := range c.Initiatives {
			names = append(names, "  "+i.Name)
		}
	}
	want := []string{"Helix launch", "  Digital & social", "  Print & outdoor", "Rimula fleet", "  Events", "  Retail events"}
	if !slices.Equal(names, want) {
		t.Errorf("tree = %q, want %q", names, want)
	}
	if tree.PlanOwner == nil || tree.PlanOwner.Name != "Priya Raman" || tree.Total != 2 {
		t.Errorf("plan owner %+v, total %d", tree.PlanOwner, tree.Total)
	}
	totals := tree.Totals
	if totals.Campaigns != 2 || totals.Initiatives != 4 || totals.Activities != 7 ||
		totals.AllocatedToCampaigns != usd("1150000.00") || *totals.PlanBudget != usd("1200000.00") ||
		*totals.RemainingPlanBudget != usd("50000.00") || totals.OverAllocatedBy != nil ||
		totals.PlannedCost == nil || *totals.PlannedCost != usd("460000.00") {
		t.Errorf("totals = %+v", totals)
	}
	helix := tree.Items[0]
	if helix.BudgetAndCost.RemainingForChildren == nil || *helix.BudgetAndCost.RemainingForChildren != usd("150000.00") {
		t.Errorf("Helix launch remaining for initiatives = %+v, want 150000.00", helix.BudgetAndCost.RemainingForChildren)
	}
}

// An unknown plan answers 404 with the plan's message.
func TestListPlanCampaigns_UnknownPlan404(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	assertBody(t, priya.Do(t, http.MethodGet, "/api/v1/plans/P-9999/campaigns", nil), http.StatusNotFound, planGoneBody)
	assertBody(t, priya.Do(t, http.MethodPost, "/api/v1/plans/P-9999/campaigns", map[string]any{"name": "Fleet growth"}),
		http.StatusNotFound, planGoneBody)
}

// Adding a campaign answers 201 with the new campaign: its readable ID, the plan's remaining budget going
// negative without blocking (D-015), and the R4 flag (FR-004, FR-008, FR-010).
func TestCreateCampaign_Contract201(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	meiLin := server.SignIn(t, meiLinsEmail)

	response := meiLin.Do(t, http.MethodPost, "/api/v1/plans/P-1002/campaigns", map[string]any{
		"name": "Fleet growth", "description": "", "ownerId": "U-1004", "startDate": "2026-01-01", "endDate": "2027-03-31",
		"budget": usd("300000.00"), "attributes": map[string]any{},
	})

	assertStatus(t, response, http.StatusCreated)
	var created campaign
	response.Decode(t, &created)
	if created.ID != "C-1006" || created.PlanID != "P-1002" || !created.OutsidePlanYear || // C-1005 is the GBP plan's (016)
		*created.ParentRemainingBudget != usd("-100000.00") || created.Version != 1 {
		t.Errorf("created = %+v", created)
	}
	missing := created.Completeness.Missing
	if len(missing) != 2 || missing[0].Label != "Cluster" || missing[1].Label != "Region" {
		t.Errorf("missing = %+v, want Cluster and Region", missing)
	}
	minimal := meiLin.Do(t, http.MethodPost, "/api/v1/plans/P-1002/campaigns", map[string]any{"name": "Only a name"})
	assertStatus(t, minimal, http.StatusCreated)
}

// Adding an initiative under Helix launch with only a name answers 201; it appears under its campaign
// (FR-001, FR-004, US2 scenario 1).
func TestCreateInitiative_Contract201(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/v1/campaigns/C-1001/initiatives", map[string]any{"name": "Events & sponsoring"})

	assertStatus(t, response, http.StatusCreated)
	var created initiative
	response.Decode(t, &created)
	// I-1011 is the demo history's deleted initiative "Trade shows", in the recycle bin (011 T019); I-1009 and
	// I-1010 are 016's GBP plan.
	if created.ID != "I-1012" || created.CampaignID != "C-1001" || created.PlanID != "P-1001" || created.Timing != nil ||
		*created.ParentRemainingBudget != usd("150000.00") {
		t.Errorf("created = %+v", created)
	}
	full := priya.Do(t, http.MethodPost, "/api/v1/campaigns/C-1001/initiatives", map[string]any{
		"name": "Search", "description": "Paid search", "ownerId": "U-1002", "startDate": "2026-02-01", "endDate": "2026-11-30",
		"budget": usd("200000.00"), "attributes": map[string]any{},
	})
	assertStatus(t, full, http.StatusCreated)
	var helix campaign
	priya.Do(t, http.MethodGet, "/api/v1/campaigns/C-1001", nil).Decode(t, &helix)
	if helix.Counts.Initiatives != 4 || *helix.BudgetAndCost.OverAllocatedBy != usd("50000.00") {
		t.Errorf("Helix launch = %+v, want 4 initiatives, over-allocated by 50000.00", helix)
	}
}

// Dates outside the campaign's are refused with the allowed range (FR-006, R3); a campaign that is gone
// answers 404.
func TestCreateInitiative_OutsideCampaign400(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	daniel := server.SignIn(t, danielsEmail)

	response := daniel.Do(t, http.MethodPost, "/api/v1/campaigns/C-1002/initiatives",
		map[string]any{"name": "Trade fairs", "startDate": "2026-01-15", "endDate": "2026-06-30"})

	assertBody(t, response, http.StatusBadRequest, validationBody(
		`{"field":"startDate","message":"Dates must be inside the campaign's dates (01 Feb 2026 – 31 Dec 2026)."}`))
	assertBody(t, daniel.Do(t, http.MethodPost, "/api/v1/campaigns/C-9999/initiatives", map[string]any{"name": "Lost"}),
		http.StatusNotFound, campaignGoneBody)
}
