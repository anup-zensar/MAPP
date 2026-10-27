package app

import (
	"context"
	"slices"

	"mapp/internal/allwork"
	"mapp/internal/campaign"
	"mapp/internal/financials"
	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// financialPlanning reads the plan tree in the order its own areas give it, inside the caller's transaction,
// and never writes.
type financialPlanning struct {
	plans         stores.PlanStore
	campaigns     *campaign.Service
	campaignsRepo stores.CampaignStore
	activities    stores.ActivityStore
	lineItems     stores.LineItemStore
	settings      *settings.Service
	people        *people.Service
	history       *history.Service
}

var (
	_ financials.Planning = financialPlanning{}
	_ financials.Displays = financialDisplays{}
	_ financials.Access   = financialAccess{}
	_ financials.Rates    = (*settings.Service)(nil)
)

// LineItemsInBin counts a line item deleted itself or with a record above it, until it is undone or restored.
func (p financialPlanning) LineItemsInBin(ctx context.Context) ([]string, error) {
	snapshots, err := p.history.SnapshotsInBin(ctx, "lineItem")
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(snapshots))
	for _, s := range snapshots {
		ids = append(ids, s.ID)
	}
	return ids, nil
}

func (p financialPlanning) MyTeamPlanIDs(ctx context.Context, personID string) ([]string, error) {
	teams, err := p.people.TeamIDsOf(ctx, personID)
	if err != nil {
		return nil, err
	}
	all, err := p.plans.List(ctx)
	if err != nil {
		return nil, err
	}
	var mine []string
	for _, p := range all {
		if slices.ContainsFunc(p.TeamIDs, func(id string) bool { return slices.Contains(teams, id) }) {
			mine = append(mine, p.ID)
		}
	}
	return mine, nil
}

type financialDisplays struct {
	people *people.Service
}

// Display shows US dollars when the person's display currency has no usable rate.
func (d financialDisplays) Display(ctx context.Context, personID string) (money.Display, error) {
	rate, err := d.people.DisplayRateOf(ctx, personID)
	if err != nil {
		return money.Display{}, err
	}
	return money.DisplayIn(rate.Code, rate.Rate), nil
}

// financialAccess checks the grants the request's context carries; an internal call, without grants, may do
// everything.
type financialAccess struct {
	tx    stores.Transactor
	plans stores.PlanStore
}

func (financialAccess) SeeFinance(ctx context.Context, _ actor.Actor) error {
	return authz.Require(ctx, authz.FinanceView)
}

func (f financialAccess) SeeFinanceIn(ctx context.Context, _ actor.Actor, planID string) error {
	if _, restricted := authz.From(ctx); !restricted || planID == "" {
		return authz.Require(ctx, authz.FinanceView)
	}
	facts, found, err := f.planFacts(ctx, planID)
	if err != nil || !found {
		return err
	}
	if err := authz.RequireView(ctx, facts, "plan"); err != nil {
		return err
	}
	return authz.RequireIn(ctx, authz.FinanceView, facts)
}

func (financialAccess) Import(ctx context.Context, _ actor.Actor) error {
	return authz.Require(ctx, authz.FinanceImport)
}

// Link with planID "" asks about any plan.
func (f financialAccess) Link(ctx context.Context, _ actor.Actor, planID string) error {
	if _, restricted := authz.From(ctx); !restricted || planID == "" {
		return authz.Require(ctx, authz.FinanceLink)
	}
	facts, found, err := f.planFacts(ctx, planID)
	if err != nil || !found {
		return err
	}
	return authz.RequireIn(ctx, authz.FinanceLink, facts)
}

// VisiblePlanIDs gives nil, every plan, to an internal call and to whoever holds finance.view for All markets.
func (f financialAccess) VisiblePlanIDs(ctx context.Context, _ actor.Actor) ([]string, error) {
	g, restricted := authz.From(ctx)
	if !restricted {
		return nil, nil
	}
	if all, _ := g.Teams(authz.FinanceView); all && g.Has(authz.FinanceView) {
		return nil, nil
	}
	visible := []string{}
	if f.plans == nil || !g.Has(authz.FinanceView) {
		return visible, nil
	}
	finance := authz.PermissionScopeOf(ctx, authz.FinanceView)
	err := f.tx.Read(ctx, func(ctx context.Context) error {
		list, err := f.plans.ListInScope(ctx, authz.ScopeOf(ctx))
		if err != nil {
			return err
		}
		for _, p := range list {
			if finance.Covers(plan.AccessFacts(p)) {
				visible = append(visible, p.ID)
			}
		}
		return nil
	})
	return visible, err
}

func (f financialAccess) planFacts(ctx context.Context, planID string) (authz.PlanFacts, bool, error) {
	if f.plans == nil {
		return authz.PlanFacts{}, false, nil
	}
	var facts authz.PlanFacts
	var found bool
	err := f.tx.Read(ctx, func(ctx context.Context) error {
		p, ok, err := f.plans.Get(ctx, planID)
		facts, found = plan.AccessFacts(p), ok
		return err
	})
	return facts, found, err
}

// addFinance leaves a line item without transactions without figures: All work shows 0.00 for it.
func (s allWorkSource) addFinance(ctx context.Context, data *allwork.YearData) error {
	if s.financials == nil {
		return nil
	}
	facts, err := s.financials.FactsByLineItem(ctx)
	if err != nil {
		return err
	}
	for i := range data.Nodes {
		n := &data.Nodes[i]
		f, found := facts[n.ID]
		if n.Level != allwork.LevelLineItem || !found {
			continue
		}
		n.Finance = &allwork.FinanceFacts{
			Committed: f.Committed, Invoiced: f.Invoiced, Paid: f.Paid, Accrued: f.Accrued, POCount: f.POCount,
			InvoiceCount: f.InvoiceCount,
		}
	}
	data.FinanceAsOf, err = s.financials.DataAsOf(ctx)
	return err
}

func (s allWorkSource) seesFinance(ctx context.Context, personID string) bool {
	return s.financeAccess != nil && s.financeAccess.SeeFinance(ctx, actor.Actor{ID: personID}) == nil
}
