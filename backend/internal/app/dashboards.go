package app

import (
	"context"

	"mapp/internal/allwork"
	"mapp/internal/dashboard"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// dashboards wires feature 036 over All work's source and the services that work its figures out (D-342).
func (w *wiring) dashboards() {
	service := dashboard.NewService(w.set.Tx, dashboard.Deps{
		Planning: dashboardPlanning{allWorkSource: w.allWork}, Plans: dashboardPlans{repo: w.set.Plans}, Spending: w.financials, Rounds: w.approvals,
		Purchases: w.prs, GlobalBudgets: w.budgets,
		Directory: dashboardDirectory{people: w.people, settings: w.settings},
	}, w.now)
	mount(w.signedIn, dashboard.NewHandler(service))
}

// dashboardPlanning reads a year without All work's GSAP figures: the board reads them for its line items only.
type dashboardPlanning struct {
	allWorkSource
}

func (p dashboardPlanning) Year(ctx context.Context, year int, _ date.Date, columns []string) (allwork.YearData, error) {
	yearPlans, err := plansOfYear(ctx, p.plansRepo, year)
	if err != nil {
		return allwork.YearData{}, err
	}
	data, err := p.nodesOf(ctx, yearPlans, year, columns)
	if err != nil || len(data.Nodes) == 0 || p.financials == nil {
		return data, err
	}
	data.FinanceAsOf, err = p.financials.DataAsOf(ctx)
	return data, err
}

type dashboardPlans struct {
	repo stores.PlanStore
}

func (p dashboardPlans) YearOf(ctx context.Context, planID string) (int, bool, error) {
	found, ok, err := p.repo.GetInScope(ctx, authz.ScopeOf(ctx), planID)
	return found.Year, ok, err
}

type dashboardDirectory struct {
	people   *people.Service
	settings *settings.Service
}

func (d dashboardDirectory) Currencies(ctx context.Context, personID string) (dashboard.Currencies, error) {
	options, err := d.people.DisplayCurrencies(ctx, actor.Actor{ID: personID})
	if err != nil {
		return dashboard.Currencies{}, err
	}
	rates, err := d.settings.Rates(ctx)
	if err != nil {
		return dashboard.Currencies{}, err
	}
	own, err := d.people.DisplayRateOf(ctx, personID)
	out := dashboard.Currencies{Default: own.Code, Items: make([]dashboard.Currency, 0, len(options))}
	for _, o := range options {
		if rate, found := rates[o.Code]; found {
			out.Items = append(out.Items, dashboard.Currency{Code: o.Code, Name: o.Name, Rate: rate.String()})
		}
	}
	return out, err
}

func (d dashboardDirectory) TeamRefs(ctx context.Context, teamIDs []string) (map[string]people.TeamRef, error) {
	return d.people.TeamRefs(ctx, teamIDs)
}
