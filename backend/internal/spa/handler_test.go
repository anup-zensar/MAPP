package spa_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
	"mapp/internal/spa"
)

const (
	priyasEmail = "priya.raman@shell.com"
	meisEmail   = "mei.lin@shell.com"
	leasEmail   = "lea.santos@shell.com"
	viewPath    = "/api/v1/plans/P-1001/spa"
)

func viewOf(t *testing.T, r *apitest.Response) spa.View {
	t.Helper()
	if r.Status != http.StatusOK {
		t.Fatalf("status %d: %s", r.Status, r.Body)
	}
	var v spa.View
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return v
}

func assertStatus(t *testing.T, r *apitest.Response, status int, code string) {
	t.Helper()
	var envelope struct{ Error struct{ Code string } }
	_ = json.Unmarshal(r.Body, &envelope)
	if r.Status != status || envelope.Error.Code != code {
		t.Errorf("status %d %s, want %d %s", r.Status, r.Body, status, code)
	}
}

func TestGetSpaView_SC003_ThePlansPlannedCostAndCommitted(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya := s.SignIn(t, priyasEmail)
	v := viewOf(t, priya.Do(t, http.MethodGet, viewPath, nil))
	var plan struct {
		Figures struct{ PlannedCost money.Money }
	}
	var card struct {
		Figures struct{ Committed, Paid money.Money }
	}
	priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001", nil).Decode(t, &plan)
	priya.Do(t, http.MethodGet, "/api/v1/financials/records/P-1001", nil).Decode(t, &card)
	if v.Headline.PlannedCost != plan.Figures.PlannedCost || *v.Headline.Committed != card.Figures.Committed ||
		*v.Headline.ActualSpend != card.Figures.Paid || v.Dimension.Label != "Channel / audience" {
		t.Errorf("headline %+v, plan %+v, card %+v", v.Headline, plan.Figures, card.Figures)
	}
}

func TestGetSpaView_FR005_ACampaignsView(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya := s.SignIn(t, priyasEmail)
	v := viewOf(t, priya.Do(t, http.MethodGet, viewPath+"?campaignId=C-1002&dimension=activityType", nil))
	if v.Scope.ID != "C-1002" || v.Scope.Type != spa.LevelCampaign || v.Dimension.Key != "activityType" ||
		len(v.Campaigns) != 2 || v.Activities[0].Campaign.ID != "C-1002" {
		t.Errorf("view scope %+v, dimension %s", v.Scope, v.Dimension.Key)
	}
}

// Grand opening Shanghai's line items exceed its planned cost: the answer has the row, as the contract says.
func TestGetSpaView_LinesAbovePlannedCost_038_FR023_M2(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	p := viewOf(t, s.SignIn(t, "carlos.mendes@shell.com").Do(t, http.MethodGet, "/api/v1/plans/P-1002/spa", nil)).Periods
	if p.LinesAbovePlannedCost == nil || len(p.LinesAbovePlannedCostTrace) != 1 ||
		p.LinesAbovePlannedCostTrace[0].Name != "Grand opening Shanghai" || (p.NotScheduled != nil && p.NotScheduled.Amount.Sign() < 0) {
		t.Errorf("not scheduled %v, above %v, trace %+v", p.NotScheduled, p.LinesAbovePlannedCost, p.LinesAbovePlannedCostTrace)
	}
}

func TestGetSpaView_FR012_ACampaignOfAnotherPlanIs(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya := s.SignIn(t, priyasEmail)
	assertStatus(t, priya.Do(t, http.MethodGet, viewPath+"?campaignId=C-1003", nil), http.StatusNotFound, "not_found")
}

func TestGetSpaView_FR018_AWrongCampaignOrDimensionIs(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya := s.SignIn(t, priyasEmail)
	assertStatus(t, priya.Do(t, http.MethodGet, viewPath+"?dimension=attr:SA-9999", nil), http.StatusBadRequest,
		"validation_failed")
	assertStatus(t, priya.Do(t, http.MethodGet, viewPath+"?campaignId=P-1", nil, apitest.SkipRequestValidation()),
		http.StatusBadRequest, "validation_failed")
}

func TestGetSpaView_FR021_ThePurchaseRequestsOnThePlans(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya := s.SignIn(t, priyasEmail)
	p := viewOf(t, priya.Do(t, http.MethodGet, "/api/v1/plans/P-1004/spa", nil)).Procurement
	if p.Count != 1 || p.Items[0].ID != "PR-1001" || !p.Items[0].Counted || p.Draft != p.Items[0].Amount ||
		p.Draft.Amount.IsZero() {
		t.Errorf("procurement = %+v", p)
	}
}

func TestGetSpaViewAccess_FR012_APlanOutsideThePersonsTeams(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	assertStatus(t, s.SignIn(t, meisEmail).Do(t, http.MethodGet, viewPath, nil), http.StatusNotFound, "not_found")
}

func TestGetSpaViewAccess_FR012_WithoutFinanceviewTheFinancialFiguresAre(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	v := viewOf(t, s.SignIn(t, leasEmail).Do(t, http.MethodGet, viewPath, nil))
	if v.FinanceAccess || v.Headline.Committed != nil || v.FinanceNote == nil || v.Headline.PlannedCost.Amount.IsZero() {
		t.Errorf("view = %+v", v.Headline)
	}
	if v.Activities[0].RaisePurchaseRequest.Available || v.Activities[0].RaisePurchaseRequest.Reason == "" {
		t.Errorf("raise = %+v", v.Activities[0].RaisePurchaseRequest)
	}
}
