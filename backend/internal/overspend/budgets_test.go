package overspend_test

import (
	"net/http"
	"strings"
	"testing"

	"mapp/internal/budget"
	"mapp/internal/overspend"
	"mapp/internal/plan"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

const lubricantsEurope = "P-1001"

func getPlan(t *testing.T, c *apitest.Client, id string) plan.PlanView {
	t.Helper()
	return decoded[plan.PlanView](t, c.Do(t, http.MethodGet, "/api/v1/plans/"+id, nil), http.StatusOK)
}

// planForm is the edit form of p with its budget and setting.
func planForm(p plan.PlanView, budgetAmount money.Amount, mode string) map[string]any {
	teamIDs := []string{}
	for _, team := range p.Teams {
		teamIDs = append(teamIDs, team.ID)
	}
	form := map[string]any{
		"version": p.Version, "name": p.Name, "modelId": p.Model.ID, "teamIds": teamIDs, "ownerId": nil, "aop": p.AOP,
		"description": p.Description, "budget": map[string]string{"amount": budgetAmount.String(), "currency": p.Currency},
		"defaultFinanceBudgetId": nil, "defaultSpread": p.DefaultSpread, "bcpir": p.BCPIR, "goalStage": p.GoalStage,
		"goalAmount": p.GoalAmount, "attributes": p.Attributes, "overspend": mode,
	}
	if p.Owner != nil {
		form["ownerId"] = p.Owner.ID
	}
	return form
}

func TestAllocatedPlan_SettingChangesOnlyThroughItsAllocation_Contract_037_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya, alex := server.SignIn(t, priyasEmail), server.SignIn(t, alexsEmail)
	europe := getPlan(t, priya, lubricantsEurope)
	planned := europe.PlanFigures.PlannedCost.Amount

	onThePlan := priya.Do(t, http.MethodPatch, "/api/v1/plans/"+lubricantsEurope, planForm(europe, europe.Budget.Amount, "refuse"))
	allocated := alex.Do(t, http.MethodPatch, "/api/v1/budget-allocations/BA-1001", map[string]any{
		"amount": money2(planned.String()), "overspend": "refuse", "version": 1, "planVersion": europe.Version})
	after := getPlan(t, priya, lubricantsEurope)
	over := createActivity(t, priya, digital, newActivity("Over the plan", "0.01"))
	copied := priya.Do(t, http.MethodPost, "/api/v1/campaigns/C-1002/copy", map[string]any{"planId": lubricantsEurope,
		"name": "Rimula again"})
	lowered := alex.Do(t, http.MethodPatch, "/api/v1/budget-allocations/BA-1001", map[string]any{
		"amount": money2("1.00"), "version": 2, "planVersion": after.Version})

	e := decoded[apiError](t, onThePlan, http.StatusConflict)
	if e.Error.Code != "not_allowed" || !strings.HasPrefix(europe.Overspend.CanChange.Reason, "Allocated from '") {
		t.Errorf("037 FR-004 on the plan form: %+v, control %+v", e, europe.Overspend)
	}
	decoded[budget.GlobalBudgetView](t, allocated, http.StatusOK)
	if after.Overspend.Mode != overspend.Refuse || after.Overspend.CanChange.Available {
		t.Errorf("037 FR-004 after the allocation: %+v", after.Overspend)
	}
	for _, r := range []*apitest.Response{over, copied, lowered} {
		refusedNaming(t, r, "plan 'FY2026 Lubricants Europe'")
	}
}

func TestGlobalBudgetRefuse_Contract_037_US3(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya, alex := server.SignIn(t, priyasEmail), server.SignIn(t, alexsEmail)
	gb := decoded[budget.GlobalBudgetView](t, alex.Do(t, http.MethodGet, "/api/v1/global-budgets/GB-1001", nil), http.StatusOK)

	set := decoded[budget.GlobalBudgetView](t, alex.Do(t, http.MethodPatch, "/api/v1/global-budgets/GB-1001", map[string]any{
		"amount": money2(gb.Figures.PlannedCost.Amount.String()), "overspend": "refuse", "version": gb.Version}), http.StatusOK)
	over := createActivity(t, priya, digital, newActivity("Over the global budget", "1.00"))
	cut := alex.Do(t, http.MethodPatch, "/api/v1/global-budgets/GB-1001", map[string]any{
		"amount": money2("1.00"), "version": set.Version})

	if set.Overspend.Mode != overspend.Refuse || !set.Overspend.CanChange.Available {
		t.Errorf("037 FR-001 global budget = %+v", set.Overspend)
	}
	refusedNaming(t, over, "global budget '"+gb.Name+"'")
	refusedNaming(t, cut, "global budget '"+gb.Name+"'")
}

func TestPlanCopyOverARefuseBudget_Contract_037_US2(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	uk := getPlan(t, priya, "P-1004")
	below := uk.PlanFigures.PlannedCost.Amount.Sub(money.FromCents(100))

	warnBelow := decoded[plan.PlanView](t, priya.Do(t, http.MethodPatch, "/api/v1/plans/P-1004", planForm(uk, below, "warn")),
		http.StatusOK)
	refuseOver := decoded[plan.PlanView](t, priya.Do(t, http.MethodPatch, "/api/v1/plans/P-1004",
		planForm(warnBelow, below, "refuse")), http.StatusOK)
	copied := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1004/copy",
		map[string]any{"name": "UK copy", "year": 2027, "includeContents": true})
	empty := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1004/copy",
		map[string]any{"name": "UK shell", "year": 2027, "includeContents": false})

	if refuseOver.Overspend.Mode != overspend.Refuse {
		t.Errorf("037 FR-011 Refuse may be chosen while over: %+v", refuseOver.Overspend)
	}
	refusedNaming(t, copied, "plan 'UK copy'")
	if empty.Status != http.StatusCreated {
		t.Errorf("037 FR-009 a copy without contents has no cost: %d %s", empty.Status, empty.Body)
	}
}
