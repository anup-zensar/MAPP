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
	"mapp/internal/platform/store"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// The wiring of feature 015, financial transactions (D-196, D-251): the ports of internal/financials over
// plans (004), campaigns and initiatives (005), activities (006), line items (007), the recycle bin (001),
// Settings' rates (002), people's display currencies and teams (003), and access (014).

// financialPlanning is financials.Planning over the planning features. It reads the plan tree as the
// owning features order it — 005's campaigns and initiatives, 006's activities with their planned costs —
// and 007's line items with their USD planned cost at the current rates. It reads inside the caller's
// transaction and never writes.
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

// The compiler checks here that the adapters implement the ports of financials.
var (
	_ financials.Planning = financialPlanning{}
	_ financials.Displays = financialDisplays{}
	_ financials.Access   = financialAccess{}
	_ financials.Rates    = (*settings.Service)(nil)
)

// LineItemsInBin returns the ids of the line items in the recycle bin: deleted themselves or with a record
// above them, in entries not yet undone or restored (001's SnapshotsInBin).
func (p financialPlanning) LineItemsInBin(ctx context.Context) ([]string, error) {
	snapshots, err := p.history.SnapshotsInBin(ctx, "lineItem") // 007's kind
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(snapshots))
	for _, s := range snapshots {
		ids = append(ids, s.ID)
	}
	return ids, nil
}

// MyTeamPlanIDs returns the plans with at least one of the person's teams (003, as Home's My teams).
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

// financialDisplays is financials.Displays over people's display currencies (003) and Settings' rates.
type financialDisplays struct {
	people *people.Service
}

// Display returns the person's display currency with its current rate; an unusable rate shows US dollars.
func (d financialDisplays) Display(ctx context.Context, personID string) (money.Display, error) {
	rate, err := d.people.DisplayRateOf(ctx, personID)
	if err != nil {
		return money.Display{}, err
	}
	return money.DisplayIn(rate.Code, rate.Rate), nil
}

// financialAccess is financials.Access (FR-028, D-252, D-301) over feature 014's grants, which the request's
// context carries: finance.view (scoped), finance.import and finance.link (scoped to the line item's plan).
// An internal call (no grants) may do everything.
type financialAccess struct {
	tx    store.Transactor
	plans stores.PlanStore
}

// SeeFinance needs finance.view in some scope.
func (financialAccess) SeeFinance(ctx context.Context, _ actor.Actor) error {
	return authz.Require(ctx, authz.FinanceView)
}

// SeeFinanceIn needs the plan to be visible and finance.view in its scope.
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

// Import needs finance.import.
func (financialAccess) Import(ctx context.Context, _ actor.Actor) error {
	return authz.Require(ctx, authz.FinanceImport)
}

// Link needs finance.link in the plan's scope; "" asks about any plan.
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

// VisiblePlanIDs gives nil (every plan) to an internal call and to a person who holds finance.view for All
// markets; otherwise the plans they see and hold finance.view in.
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

// planFacts returns the access facts of the plan planID, and false when there is none.
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

// addFinance gives All work the GSAP figures of the year's line items, as feature 015 works them out for
// its cards (FR-023, SC-002), and when the GSAP data was loaded, for the export's note. A line item without
// transactions keeps no figures: All work shows 0.00 for it.
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

// seesFinance reports whether the person may see financial data in at least one plan (015 FR-028): All
// work then offers the Financials columns and filter.
func (s allWorkSource) seesFinance(ctx context.Context, personID string) bool {
	return s.financeAccess != nil && s.financeAccess.SeeFinance(ctx, actor.Actor{ID: personID}) == nil
}
