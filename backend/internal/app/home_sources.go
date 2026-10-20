package app

import (
	"context"

	"northstar/internal/activities"
	"northstar/internal/approvals"
	"northstar/internal/campaigns"
	"northstar/internal/home"
	"northstar/internal/people"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// homeSources builds the Sources of feature 009's Home over the services of features 003–008 (009 plan
// "Inputs", D-171, D-196): what Home reads is what those features calculate for their own pages, so every
// figure on Home equals the one there (FR-017). The names of those packages appear only here. Every
// function joins the read transaction of the Home service.
func homeSources(peopleService *people.Service, plansRepo plans.Repository, planService *plans.Service,
	campaignsService *campaigns.Service, activitiesService *activities.Service, approvalsService *approvals.Service) home.Sources {
	return home.Sources{
		MyTeams: func(ctx context.Context, a actor.Actor) ([]home.Team, error) {
			return myTeams(ctx, peopleService, a)
		},
		Plans: func(ctx context.Context, a actor.Actor) ([]home.Plan, error) {
			overviews, err := planService.Overviews(ctx, a)
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
			return ownedContainers(ctx, plansRepo, campaignsService, a, ownerID)
		},
		Activities: func(ctx context.Context, a actor.Actor, ownerID string) ([]home.Activity, error) {
			return ownedActivities(ctx, activitiesService, approvalsService, a, ownerID)
		},
		WaitingForMe: func(ctx context.Context, a actor.Actor) ([]home.Waiting, error) {
			return waitingFor(ctx, approvalsService, a)
		},
		DisplayCurrency: func(ctx context.Context, a actor.Actor) (home.Currency, error) {
			display, err := peopleService.DisplayRateOf(ctx, a.ID)
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

// myTeams returns the teams a belongs to, active or not (003).
func myTeams(ctx context.Context, peopleService *people.Service, a actor.Actor) ([]home.Team, error) {
	teamIDs, err := peopleService.TeamIDsOf(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	refs, err := peopleService.TeamRefs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	teams := make([]home.Team, 0, len(teamIDs))
	for _, id := range teamIDs {
		teams = append(teams, home.Team{ID: id, Name: refs[id].Name, Colour: refs[id].Colour, Active: refs[id].Active})
	}
	return teams, nil
}

// homePlan returns a plan of feature 004, its figures in US dollars, as Home reads it.
func homePlan(o plans.Overview) home.Plan {
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

// ownedContainers returns the campaigns and initiatives ownerID owns, in every plan the person may see, as
// the plan tree shows them (005): budgets, roll-ups and completeness worked out by the campaigns' own view builder (D-090).
func ownedContainers(ctx context.Context, plansRepo plans.Repository, campaignsService *campaigns.Service, a actor.Actor,
	ownerID string) ([]home.Container, error) {
	list := []home.Container{}
	if ownerID == "" {
		return list, nil
	}
	all, err := plansRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	owns := func(id *string) bool { return id != nil && *id == ownerID }
	for _, p := range all {
		if !authz.CanView(ctx, plans.AccessFacts(p)) { // only plans the person may see (014 FR-007)
			continue
		}
		tree, err := campaignsService.ListPlanCampaigns(ctx, a, p.ID)
		if err != nil {
			return nil, err
		}
		planRef := errs.RecordRef{Type: "plan", ID: p.ID, Name: p.Name}
		for _, c := range tree.Items {
			campaignRef := errs.RecordRef{Type: "campaign", ID: c.ID, Name: c.Name}
			if owns(c.OwnerID) {
				list = append(list, home.Container{
					Type: "campaign", ID: c.ID, Name: c.Name, Plan: planRef, Path: []errs.RecordRef{planRef},
					Budget: amountOf(c.Budget), PlannedCost: amountOf(c.BudgetAndCost.PlannedCost),
					RemainingBudget: amountOf(c.BudgetAndCost.RemainingForChildren), Missing: c.Completeness.Missing,
					ItemsBelow: c.Completeness.ItemsBelowNeedingAttention, UpdatedAt: c.UpdatedAt,
				})
			}
			for _, i := range c.Initiatives {
				if owns(i.OwnerID) {
					list = append(list, home.Container{
						Type: "initiative", ID: i.ID, Name: i.Name, Plan: planRef, Path: []errs.RecordRef{planRef, campaignRef},
						Budget: amountOf(i.Budget), PlannedCost: amountOf(i.BudgetAndCost.PlannedCost), Missing: i.Completeness.Missing,
						ItemsBelow: i.Completeness.ItemsBelowNeedingAttention, UpdatedAt: i.UpdatedAt,
					})
				}
			}
		}
	}
	return list, nil
}

// ownedActivities returns the activities ownerID owns, in every plan, as their pages show them (006, 007),
// each Declined one with the decline its Approval section shows (008).
func ownedActivities(ctx context.Context, activitiesService *activities.Service, approvalsService *approvals.Service,
	a actor.Actor, ownerID string) ([]home.Activity, error) {
	owned, err := activitiesService.OwnedBy(ctx, a, ownerID)
	if err != nil {
		return nil, err
	}
	list := make([]home.Activity, 0, len(owned))
	for _, o := range owned {
		act := home.Activity{
			ID: o.ID, Name: o.Name, Plan: o.Plan, Campaign: o.Campaign, Initiative: o.Initiative,
			Status: string(o.Status), DisplayStatus: string(o.DisplayStatus), StartDate: o.StartDate, EndDate: o.EndDate,
			Ongoing: o.Ongoing, PlannedCost: amountOf(o.PlannedCost), Unassigned: amountOf(o.UnassignedPlannedCost),
			Missing: o.Missing, LineItemsNeedingAttention: o.LineItemsNeedingAttention, UpdatedAt: o.UpdatedAt,
		}
		if o.Status == activities.StatusDeclined {
			approval, err := approvalsService.Approval(ctx, a, o.ID)
			if err != nil {
				return nil, err
			}
			if d := approval.LatestDecline; d != nil {
				act.Decline = &home.Decline{By: d.Actor, At: d.At, Comment: d.Comment}
			}
		}
		list = append(list, act)
	}
	return list, nil
}

// waitingFor returns the Waiting for me tab of the Approvals page for a, oldest submission first (008).
func waitingFor(ctx context.Context, approvalsService *approvals.Service, a actor.Actor) ([]home.Waiting, error) {
	rows, err := approvalsService.Inbox(ctx, a, approvals.TabWaiting)
	if err != nil {
		return nil, err
	}
	list := make([]home.Waiting, 0, len(rows))
	for _, row := range rows {
		w := home.Waiting{
			ActivityID: row.Activity.ID, ActivityName: row.Activity.Name, PlannedCost: amountOf(row.PlannedCost),
			Level: row.Round.CurrentLevel, Levels: row.Round.LevelCount, WaitingSince: row.SubmittedAt,
		}
		if row.WaitingSince != nil { // set for every open round
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
