package seed

import (
	"cmp"
	"context"
	"fmt"

	"northstar/internal/plans"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/rollup"
	"northstar/internal/settings"
)

// demoPlan is one plan of 004 plan "Seed".
type demoPlan struct {
	name, model, team, owner, budget string
	year                             int
	status                           plans.Status
	businessUnits                    []string // Business unit option labels
	smp                              bool     // Strategic Marketing Plan (SMP): yes
	aop                              bool
	goalStage                        plans.GoalStage
	goalAmount                       string
	financeBudget                    string // the default finance budget's name; "" for none
	description                      string
	planType                         plans.PlanType       // 016: "" is Operational
	currency                         string               // 016: the plan currency; "" is USD
	phasing                          rollup.PhasingMethod // 016: the budget's phasing; "" is not phased
}

// demoPlans are P-1001 … P-1003 and, since 016 (data-model §8), P-1004: a 2027 Cross-market plan planned in
// GBP with its budget phased evenly. FY2026 Mobility China has no business unit on purpose: it shows what
// Publish needs.
var demoPlans = []demoPlan{
	{name: "FY2026 Lubricants Europe", year: 2026, model: "Lubricants 1.0", team: "Lubricants Europe", owner: "Priya Raman",
		budget: "1200000.00", status: plans.StatusPublished, businessUnits: []string{"Lubricants"},
		goalStage: plans.StageINQ, goalAmount: "250000", financeBudget: "2026 Lubricants Budget",
		description: "Lubricants marketing in Europe for 2026."},
	{name: "FY2026 Mobility China", year: 2026, model: "Mobility 1.0", team: "Mobility China", owner: "Mei Lin",
		budget: "800000.00", status: plans.StatusDraft, goalStage: plans.StageINQ, goalAmount: "150000",
		financeBudget: "2026 Mobility Budget"},
	{name: "FY2027 Global Brand", year: 2027, model: "Lubricants 1.0", team: "Global Marketing Ops", owner: "Alex Morgan",
		budget: "500000.00", status: plans.StatusDraft, businessUnits: []string{"Mobility", "Lubricants"}, smp: true, aop: true,
		goalStage: plans.StageRevenue, goalAmount: "2000000.00", planType: plans.TypeStrategic},
	{name: ukPlan, year: 2027, model: "Lubricants 1.0", team: "Lubricants Europe", owner: "Priya Raman",
		budget: "600000.00", status: plans.StatusDraft, businessUnits: []string{"Lubricants"}, goalStage: plans.StageINQ,
		description: "Lubricants marketing in the United Kingdom and Ireland for 2027, planned in pounds sterling.",
		planType:    plans.TypeCrossMarket, currency: "GBP", phasing: rollup.MethodEvenly},
}

// ukPlan is P-1004 of 016's demo data, planned in GBP.
const ukPlan = "FY2027 Lubricants UK & Ireland"

// addPlans creates the three demo plans through the plans repository, looking up the model, team, owner,
// finance budget and business unit by name.
func (d *demo) addPlans(ctx context.Context) error {
	repo := plans.NewJSONRepository()
	d.plans, d.planCurrencies = map[string]string{}, map[string]string{}
	for _, p := range demoPlans {
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
		plan := plans.Plan{
			Name: p.name, Year: p.year, ModelID: r.id(d.settings.Models, "model", p.model),
			TeamIDs: []string{r.id(d.teams, "team", p.team)}, OwnerID: r.id(d.people, "person", p.owner),
			AOP: p.aop, Description: p.description, DefaultSpread: plans.SpreadEvenly, GoalStage: p.goalStage,
			GoalAmount: p.goalAmount, Attributes: attributes, Status: p.status, Meta: d.meta(),
			Currency: cmp.Or(p.currency, money.BaseCurrency), PlanType: cmp.Or(p.planType, plans.TypeOperational),
			BudgetPhasing: rollup.Phasing{Method: cmp.Or(p.phasing, rollup.MethodNone), Months: []rollup.PhasingMonth{}},
		}
		if p.financeBudget != "" {
			plan.DefaultFinanceBudgetID = r.id(d.settings.Budgets, "finance budget", fmt.Sprintf("%d/%s", p.year, p.financeBudget))
		}
		if r.err != nil {
			return r.err
		}
		budget, err := money.ParseAmount(p.budget)
		if err != nil {
			return fmt.Errorf("seed: budget of %s: %w", p.name, err)
		}
		plan.Budget = &budget
		if plan.ID, err = d.next(ctx, ids.PrefixPlan); err != nil {
			return err
		}
		if err := repo.Put(ctx, plan); err != nil {
			return err
		}
		d.plans[p.name] = plan.ID
		d.planCurrencies[p.name] = plan.Currency
	}
	return nil
}
