package seed

import (
	"cmp"
	"context"
	"fmt"

	"mapp/internal/overspend"
	"mapp/internal/plan"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

type demoPlan struct {
	name, model, team, owner, budget string
	year                             int
	status                           plan.Status
	businessUnits                    []string
	smp                              bool
	aop                              bool
	bcpir                            bool
	goalStage                        plan.GoalStage
	goalAmount                       string
	financeBudget                    string
	description                      string
	planType                         plan.PlanType
	currency                         string
	phasing                          rollup.PhasingMethod
	overspend                        string
}

// demoPlans: FY2026 Mobility China has no business unit on purpose (it shows what Publish needs), and FY2026
// Lubricants Europe is the BCPIR plan of feature 013.
var demoPlans = []demoPlan{
	{name: "FY2026 Lubricants Europe", year: 2026, model: "Lubricants 1.0", team: "Lubricants Europe", owner: "Priya Raman",
		budget: lubricantsEuropeBudget, status: plan.StatusPublished, businessUnits: []string{unitLubricants},
		goalStage: plan.StageINQ, goalAmount: "250000", financeBudget: lubricantsBudget, bcpir: true,
		description: "Lubricants marketing in Europe for 2026."},
	{name: "FY2026 Mobility China", year: 2026, model: "Mobility 1.0", team: "Mobility China", owner: "Mei Lin",
		budget: mobilityChinaBudget, status: plan.StatusDraft, goalStage: plan.StageINQ, goalAmount: "150000",
		financeBudget: mobilityBudget},
	{name: "FY2027 Global Brand", year: 2027, model: "Lubricants 1.0", team: "Global Marketing Ops", owner: "Alex Morgan",
		budget: globalBrandBudget, status: plan.StatusDraft, businessUnits: []string{unitMobility, unitLubricants}, smp: true, aop: true,
		goalStage: plan.StageRevenue, goalAmount: "2000000.00", planType: plan.TypeStrategic},
	{name: ukPlan, year: 2027, model: "Lubricants 1.0", team: "Lubricants Europe", owner: "Priya Raman",
		budget: ukPlanBudget, status: plan.StatusDraft, businessUnits: []string{unitLubricants}, goalStage: plan.StageINQ,
		description: "Lubricants marketing in the United Kingdom and Ireland for 2027, planned in pounds sterling.",
		planType:    plan.TypeCrossMarket, currency: ukPlanCurrency, phasing: rollup.MethodEvenly},
}

const (
	ukPlan         = "FY2027 Lubricants UK & Ireland"
	ukPlanCurrency = "GBP"
)

const (
	lubricantsEuropeBudget = "1200000.00"
	mobilityChinaBudget    = "800000.00"
	globalBrandBudget      = "500000.00"
	ukPlanBudget           = "600000.00"
)

func (d *demo) addPlans(ctx context.Context) error {
	d.plans, d.planCurrencies = map[string]string{}, map[string]string{}
	return d.putPlans(ctx, demoPlans)
}

func (d *demo) putPlans(ctx context.Context, list []demoPlan) error {
	for _, p := range list {
		if err := d.putPlan(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func (d *demo) putPlan(ctx context.Context, p demoPlan) error {
	var r resolver
	values := d.newValues(settings.LevelPlan)
	if len(p.businessUnits) > 0 {
		values.choices("Business unit", p.businessUnits...)
	}
	if p.smp {
		values.yesNo("Strategic Marketing Plan (SMP)", true)
	}
	attributes, err := values.done()
	if err != nil {
		return err
	}
	pl := plan.Plan{
		Name: p.name, Year: p.year, ModelID: r.id(d.settings.Models, "model", p.model),
		TeamIDs: []string{r.id(d.teams, "team", p.team)}, OwnerID: r.id(d.people, "person", p.owner),
		AOP: p.aop, BCPIR: p.bcpir, Overspend: overspend.Mode(p.overspend).Normalised(), Description: p.description, DefaultSpread: plan.SpreadEvenly, GoalStage: p.goalStage,
		GoalAmount: p.goalAmount, Attributes: attributes, Status: p.status, Meta: d.meta(),
		Currency: cmp.Or(p.currency, money.BaseCurrency), PlanType: cmp.Or(p.planType, plan.TypeOperational),
		BudgetPhasing: rollup.Phasing{Method: cmp.Or(p.phasing, rollup.MethodNone), Months: []rollup.PhasingMonth{}},
	}
	if p.financeBudget != "" {
		pl.DefaultFinanceBudgetID = r.id(d.settings.Budgets, "finance budget", fmt.Sprintf("%d/%s", p.year, p.financeBudget))
	}
	if r.err != nil {
		return r.err
	}
	budget, err := money.ParseAmount(p.budget)
	if err != nil {
		return fmt.Errorf("seed: budget of %s: %w", p.name, err)
	}
	pl.Budget = &budget
	if pl.ID, err = d.next(ctx, ids.PrefixPlan); err != nil {
		return err
	}
	if err := d.stores.Plans.Put(ctx, pl); err != nil {
		return err
	}
	d.plans[p.name] = pl.ID
	d.planCurrencies[p.name] = pl.Currency
	return nil
}
