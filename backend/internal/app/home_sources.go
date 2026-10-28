package app

import (
	"context"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/campaign"
	"mapp/internal/history"
	"mapp/internal/home"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/stores"
)

type homeReaders struct {
	people     *people.Service
	plansRepo  stores.PlanStore
	plans      *plan.Service
	campaigns  *campaign.Service
	activities *activity.Service
	approvals  *approval.Service
	history    *history.Service
	rates      func(context.Context) (money.Rates, error)
}

// sources builds the Sources of feature 009's Home over the services of features 003–008 and 011's
// activity log (009 plan "Inputs", D-171, D-196): what Home reads is what those features calculate for their
// own pages, so every figure on Home equals the one there. The names of those packages appear only
// here. Every function joins the read transaction of the Home service.
func (h homeReaders) sources() home.Sources {
	return home.Sources{
		RecentChanges: func(ctx context.Context, a actor.Actor, planIDs []string, limit int) (home.RecentChangesPage, error) {
			return h.recentChanges(ctx, a, planIDs, limit)
		},
		MyTeams: func(ctx context.Context, a actor.Actor) ([]home.Team, error) {
			return h.myTeams(ctx, a)
		},
		Plans: func(ctx context.Context, a actor.Actor) ([]home.Plan, error) {
			overviews, err := h.plans.Overviews(ctx, a)
			if err != nil {
				return nil, err
			}
			list := make([]home.Plan, 0, len(overviews))
			for _, o := range overviews {
				list = append(list, homePlan(o))
			}
			return list, nil
		},
		Containers: func(ctx context.Context, a actor.Actor, ownerID string) ([]home.Container, error) {
			current, err := h.rates(ctx)
			if err != nil {
				return nil, err
			}
			return h.ownedContainers(ctx, a, ownerID, current)
		},
		Activities: func(ctx context.Context, a actor.Actor, ownerID string) ([]home.Activity, error) {
			current, err := h.rates(ctx)
			if err != nil {
				return nil, err
			}
			return h.ownedActivities(ctx, a, ownerID, current)
		},
		WaitingForMe: func(ctx context.Context, a actor.Actor) ([]home.Waiting, error) {
			return h.waitingFor(ctx, a)
		},
		DisplayCurrency: func(ctx context.Context, a actor.Actor) (home.Currency, error) {
			display, err := h.people.DisplayRateOf(ctx, a.ID)
			if err != nil {
				return home.Currency{}, err
			}
			rate, err := money.ParseRate(display.Rate)
			if err != nil || display.Code == "" { // an unusable rate shows US dollars, as on the plan page
				return home.Currency{Code: money.BaseCurrency, Rate: money.One}, nil
			}
			return home.Currency{Code: display.Code, Rate: rate}, nil
		},
	}
}

func (h homeReaders) myTeams(ctx context.Context, a actor.Actor) ([]home.Team, error) {
	teamIDs, err := h.people.TeamIDsOf(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	refs, err := h.people.TeamRefs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	teams := make([]home.Team, 0, len(teamIDs))
	for _, id := range teamIDs {
		teams = append(teams, home.Team{ID: id, Name: refs[id].Name, Colour: refs[id].Colour, Active: refs[id].Active})
	}
	return teams, nil
}

func homePlan(o plan.Overview) home.Plan {
	p := home.Plan{
		ID: o.Plan.ID, Name: o.Plan.Name, Year: o.Plan.Year, StrategicAOP: o.Plan.AOP, Status: string(o.Plan.Status),
		OwnerID: o.Plan.OwnerID, Teams: make([]home.Team, 0, len(o.Teams)), Budget: amountOf(o.Figures.Budget),
		PlannedCost: amountOf(o.Figures.PlannedCost), RemainingBudget: amountOf(o.Figures.RemainingBudget),
		CampaignCount: o.Contents.Campaigns, ActivityCount: o.Figures.Activities, ApprovedActivityCount: o.Figures.ActivitiesApproved,
		Missing: o.Completeness.Missing, ItemsBelow: len(o.Completeness.ItemsBelow), UpdatedAt: o.Plan.UpdatedAt,
	}
	for _, t := range o.Teams {
		p.Teams = append(p.Teams, home.Team{ID: t.ID, Name: t.Name, Colour: t.Colour, Active: t.Active})
	}
	return p
}

// ownedContainers works out budgets, roll-ups and completeness with the campaigns' own view builder (D-090).
func (h homeReaders) ownedContainers(ctx context.Context, a actor.Actor, ownerID string, rates money.Rates) ([]home.Container, error) {
	list := []home.Container{}
	if ownerID == "" {
		return list, nil
	}
	all, err := h.plansRepo.ListInScope(ctx, authz.ScopeOf(ctx)) // only plans the person may see
	if err != nil {
		return nil, err
	}
	for _, p := range all {
		tree, err := h.campaigns.ListPlanCampaigns(ctx, a, p.ID)
		if err != nil {
			return nil, err
		}
		planRef := errs.RecordRef{Type: "plan", ID: p.ID, Name: p.Name}
		for _, c := range tree.Items {
			list = append(list, ownedIn(c, planRef, ownerID, rates)...)
		}
	}
	return list, nil
}

func ownedIn(c campaign.CampaignView, planRef errs.RecordRef, ownerID string, rates money.Rates) []home.Container {
	var list []home.Container
	owns := func(id *string) bool { return id != nil && *id == ownerID }
	campaignRef := errs.RecordRef{Type: campaignLevel, ID: c.ID, Name: c.Name}
	if owns(c.OwnerID) {
		list = append(list, home.Container{
			Type: campaignLevel, ID: c.ID, Name: c.Name, Plan: planRef, Path: []errs.RecordRef{planRef},
			Budget: usdOf(c.Budget, rates), PlannedCost: usdOf(c.BudgetAndCost.PlannedCost, rates),
			RemainingBudget: usdOf(c.BudgetAndCost.RemainingForChildren, rates), Missing: c.Completeness.Missing,
			ItemsBelow: c.Completeness.ItemsBelowNeedingAttention, UpdatedAt: c.UpdatedAt,
		})
	}
	for _, i := range c.Initiatives {
		if owns(i.OwnerID) {
			list = append(list, home.Container{
				Type: initiativeLevel, ID: i.ID, Name: i.Name, Plan: planRef, Path: []errs.RecordRef{planRef, campaignRef},
				Budget: usdOf(i.Budget, rates), PlannedCost: usdOf(i.BudgetAndCost.PlannedCost, rates), Missing: i.Completeness.Missing,
				ItemsBelow: i.Completeness.ItemsBelowNeedingAttention, UpdatedAt: i.UpdatedAt,
			})
		}
	}
	return list
}

// ownedActivities shows each Declined activity with the decline its Approval section shows (008).
func (h homeReaders) ownedActivities(ctx context.Context, a actor.Actor, ownerID string, rates money.Rates) ([]home.Activity, error) {
	owned, err := h.activities.OwnedBy(ctx, a, ownerID)
	if err != nil {
		return nil, err
	}
	list := make([]home.Activity, 0, len(owned))
	for _, o := range owned {
		act := home.Activity{
			ID: o.ID, Name: o.Name, Plan: o.Plan, Campaign: o.Campaign, Initiative: o.Initiative,
			Status: string(o.Status), DisplayStatus: string(o.DisplayStatus), StartDate: o.StartDate, EndDate: o.EndDate,
			Ongoing: o.Ongoing, PlannedCost: usdOf(o.PlannedCost, rates), Unassigned: usdOf(o.UnassignedPlannedCost, rates),
			Missing: o.Missing, LineItemsNeedingAttention: o.LineItemsNeedingAttention, UpdatedAt: o.UpdatedAt,
		}
		if o.Status == activity.StatusDeclined {
			appr, err := h.approvals.Approval(ctx, a, o.ID)
			if err != nil {
				return nil, err
			}
			if d := appr.LatestDecline; d != nil {
				act.Decline = &home.Decline{By: d.Actor, At: d.At, Comment: d.Comment}
			}
		}
		list = append(list, act)
	}
	return list, nil
}

// waitingFor lists the activities of a's Waiting for me: Home links its rows to activities (031 research R13).
func (h homeReaders) waitingFor(ctx context.Context, a actor.Actor) ([]home.Waiting, error) {
	rows, err := h.approvals.Inbox(ctx, a, approval.TabWaiting)
	if err != nil {
		return nil, err
	}
	list := make([]home.Waiting, 0, len(rows))
	for _, row := range rows {
		if row.Activity == nil {
			continue
		}
		w := home.Waiting{
			ActivityID: row.Activity.ID, ActivityName: row.Activity.Name, PlannedCost: amountOf(row.PlannedCost),
			Level: row.Round.CurrentLevel, Levels: row.Round.LevelCount, WaitingSince: row.SubmittedAt,
		}
		if row.WaitingSince != nil {
			w.WaitingSince = *row.WaitingSince
		}
		if len(row.Path) > 0 {
			w.Plan = row.Path[0]
		}
		if row.Owner != nil {
			w.Owner = &actor.Actor{ID: row.Owner.ID, Name: row.Owner.Name}
		}
		list = append(list, w)
	}
	return list, nil
}

// amountOf returns the amount of m, nil when there is none. The amounts Home reads are all in US dollars.
func amountOf(m *money.Money) *money.Amount {
	if m == nil {
		return nil
	}
	amount := m.Amount
	return &amount
}

var planningKinds = []string{"plan", "campaign", "initiative", "activity", "lineItem"}

func (h homeReaders) recentChanges(ctx context.Context, a actor.Actor, planIDs []string, limit int) (home.RecentChangesPage, error) {
	page, err := h.history.QueryLog(ctx, a, history.LogQuery{RecordTypes: planningKinds, PlanIDs: planIDs, Limit: limit})
	if err != nil {
		return home.RecentChangesPage{}, err
	}
	items := make([]any, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, item)
	}
	return home.RecentChangesPage{Items: items, Total: page.Total}, nil
}

// usdOf returns m, an amount of a plan currency, in US dollars at the current rates: Home adds
// up records of several plans. nil when there is none; a currency without a rate is read as it is.
func usdOf(m *money.Money, rates money.Rates) *money.Amount {
	if m == nil {
		return nil
	}
	usd, err := rates.Convert(*m, money.BaseCurrency)
	if err != nil {
		return amountOf(m)
	}
	return &usd.Amount
}
