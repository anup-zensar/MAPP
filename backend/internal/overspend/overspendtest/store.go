// Package overspendtest is the in-memory fake of the overspend store: it adds up the activities of the other
// areas' fake tables as the SQL of the PostgreSQL store does.
package overspendtest

import (
	"context"

	"mapp/internal/activity"
	"mapp/internal/budget"
	"mapp/internal/campaign"
	"mapp/internal/overspend"
	"mapp/internal/plan"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/money"
)

const configurationID = "overspend"

type configurationRow struct {
	ID string `json:"id"`
	overspend.Configuration
}

type Store struct {
	plans         *fake.Table[plan.Plan]
	campaigns     *fake.Table[campaign.Campaign]
	initiatives   *fake.Table[campaign.Initiative]
	activities    *fake.Table[activity.Activity]
	budgets       *fake.Table[budget.GlobalBudget]
	allocations   *fake.Table[budget.Allocation]
	configuration *fake.Table[configurationRow]
}

func NewStore(db *fake.DB) Store {
	return Store{
		plans:         fake.TableOf(db, "plans", func(p plan.Plan) string { return p.ID }),
		campaigns:     fake.TableOf(db, "campaigns", func(c campaign.Campaign) string { return c.ID }),
		initiatives:   fake.TableOf(db, "initiatives", func(i campaign.Initiative) string { return i.ID }),
		activities:    fake.TableOf(db, "activities", func(a activity.Activity) string { return a.ID }),
		budgets:       fake.TableOf(db, "global_budgets", func(b budget.GlobalBudget) string { return b.ID }),
		allocations:   fake.TableOf(db, "budget_allocations", func(a budget.Allocation) string { return a.ID }),
		configuration: fake.TableOf(db, "overspend_configuration", func(c configurationRow) string { return c.ID }),
	}
}

// costs is the planned cost under each plan, campaign and initiative, in minor units.
type costs struct {
	byPlan, byCampaign, byInitiative map[string]int64
	currencyOf                       map[string]string
}

func (s Store) costs() (costs, error) {
	c := costs{byPlan: map[string]int64{}, byCampaign: map[string]int64{}, byInitiative: map[string]int64{},
		currencyOf: map[string]string{}}
	plans, err := s.plans.List()
	if err != nil {
		return c, err
	}
	for _, p := range plans {
		c.currencyOf[p.ID] = p.Normalised().Currency
	}
	campaignOf, planOf, err := s.parents()
	if err != nil {
		return c, err
	}
	activities, err := s.activities.List()
	for _, a := range activities {
		if a.PlannedCost == nil {
			continue
		}
		cents := a.PlannedCost.Amount.Cents()
		c.byInitiative[a.InitiativeID] += cents
		c.byCampaign[campaignOf[a.InitiativeID]] += cents
		c.byPlan[planOf[campaignOf[a.InitiativeID]]] += cents
	}
	return c, err
}

func (s Store) parents() (campaignOf, planOf map[string]string, err error) {
	campaignOf, planOf = map[string]string{}, map[string]string{}
	initiatives, err := s.initiatives.List()
	if err != nil {
		return nil, nil, err
	}
	for _, i := range initiatives {
		campaignOf[i.ID] = i.CampaignID
	}
	campaigns, err := s.campaigns.List()
	for _, c := range campaigns {
		planOf[c.ID] = c.PlanID
	}
	return campaignOf, planOf, err
}

func (s Store) PlanningBudgets(context.Context) ([]overspend.Budget, error) {
	c, err := s.costs()
	if err != nil {
		return nil, err
	}
	budgets, err := s.planBudgets(c)
	if err != nil {
		return nil, err
	}
	containers, err := s.containerBudgets(c)
	return append(budgets, containers...), err
}

func (s Store) planBudgets(c costs) ([]overspend.Budget, error) {
	plans, err := s.plans.Where(func(p plan.Plan) bool { return p.Budget != nil })
	budgets := make([]overspend.Budget, 0, len(plans))
	for _, p := range plans {
		currency := c.currencyOf[p.ID]
		budgets = append(budgets, overspend.Budget{Kind: overspend.KindPlan, ID: p.ID, Name: p.Name,
			Mode: p.Overspend.Normalised(), Amount: money.Money{Amount: *p.Budget, Currency: currency},
			Planned: inMinor(c.byPlan[p.ID], currency)})
	}
	return budgets, err
}

func (s Store) containerBudgets(c costs) ([]overspend.Budget, error) {
	campaigns, err := s.campaigns.Where(func(x campaign.Campaign) bool { return x.Budget != nil })
	if err != nil {
		return nil, err
	}
	budgets := []overspend.Budget{}
	for _, x := range campaigns {
		budgets = append(budgets, overspend.Budget{Kind: overspend.KindCampaign, ID: x.ID, Name: x.Name,
			Mode: x.Overspend.Normalised(), Amount: *x.Budget, Planned: inMinor(c.byCampaign[x.ID], x.Budget.Currency)})
	}
	initiatives, err := s.initiatives.Where(func(i campaign.Initiative) bool { return i.Budget != nil })
	for _, i := range initiatives {
		budgets = append(budgets, overspend.Budget{Kind: overspend.KindInitiative, ID: i.ID, Name: i.Name,
			Mode: i.Overspend.Normalised(), Amount: *i.Budget, Planned: inMinor(c.byInitiative[i.ID], i.Budget.Currency)})
	}
	return budgets, err
}

func inMinor(cents int64, currency string) money.Money {
	return money.Money{Amount: money.FromCents(cents), Currency: currency}
}

func (s Store) GlobalBudgets(context.Context) ([]overspend.GlobalBudget, error) {
	c, err := s.costs()
	if err != nil {
		return nil, err
	}
	budgets, err := s.budgets.List()
	if err != nil {
		return nil, err
	}
	allocations, err := s.allocations.List()
	globals := make([]overspend.GlobalBudget, 0, len(budgets))
	for _, b := range budgets {
		g := overspend.GlobalBudget{ID: b.ID, Name: b.Name, Mode: b.Overspend.Normalised(), Amount: b.Amount,
			Plans: []overspend.PlanCost{}}
		for _, a := range allocations {
			if _, planStored := c.currencyOf[a.PlanID]; a.GlobalBudgetID == b.ID && planStored {
				g.Plans = append(g.Plans, overspend.PlanCost{PlanID: a.PlanID,
					Planned: inMinor(c.byPlan[a.PlanID], c.currencyOf[a.PlanID])})
			}
		}
		globals = append(globals, g)
	}
	return globals, err
}

func (s Store) Configuration(context.Context) (overspend.Configuration, error) {
	row, found, err := s.configuration.Get(configurationID)
	if err != nil || !found {
		return overspend.DefaultConfiguration(), err
	}
	return row.Configuration, nil
}

func (s Store) PutConfiguration(ctx context.Context, c overspend.Configuration) error {
	return s.configuration.Put(ctx, configurationRow{ID: configurationID, Configuration: c})
}
