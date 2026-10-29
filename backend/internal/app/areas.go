package app

import (
	"context"

	"mapp/internal/activity"
	"mapp/internal/allwork"
	"mapp/internal/approval"
	"mapp/internal/budget"
	"mapp/internal/campaign"
	"mapp/internal/financials"
	"mapp/internal/history"
	"mapp/internal/home"
	"mapp/internal/lineitem"
	"mapp/internal/overspend"
	"mapp/internal/plan"
	"mapp/internal/sample"
	"mapp/internal/settings"
)

// areas builds the business areas after people and access, each after those it reads; where people's and
// Settings' records are used is wired last, once every area exists.
func (w *wiring) areas() error {
	steps := []func() error{
		w.settingsArea, w.overspendArea, w.plansArea, w.planningArea, w.budgetsArea, w.samplesArea, w.approvalsArea, w.financialsArea,
		w.swbsArea, w.purchaseRequestsArea,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	w.useOverspend()
	w.views()
	w.spaArea()
	w.dashboards()
	w.usesOfPeople()
	w.usesInSettings()
	return nil
}

func (w *wiring) settingsArea() error {
	w.settings = settings.NewService(history.Tools{Tx: w.set.Tx, IDs: w.generator, Log: w.history, Now: w.now}, w.set.Settings)
	if err := w.register(w.settings.Kinds()); err != nil {
		return err
	}
	if err := w.rates(); err != nil {
		return err
	}
	mount(w.signedIn, settings.NewHandler(w.settings))
	return nil
}

// rates gives a store of before 016 each currency's stored rate as its first dated rate, once; GSAP daily
// rates come from a file import behind the rate feed (D-259).
func (w *wiring) rates() error {
	if err := w.settings.MigrateRates(context.Background()); err != nil {
		return err
	}
	w.settings.UseRateFeed(settings.GSAPRateFile{})
	return nil
}

func (w *wiring) plansArea() error {
	tools := history.Tools{Tx: w.guard, IDs: w.generator, Log: w.history, Now: w.now}
	w.plans = plan.NewService(tools, w.set.Plans, w.settings, w.people)
	if err := w.register(overspend.Described(w.plans.Kinds())); err != nil {
		return err
	}
	mount(w.signedIn, plan.NewHandler(w.plans))
	w.history.UseAccess(historyAccess{plans: w.set.Plans, history: w.history, purchaseRequests: w.set.PurchaseRequests})
	return nil
}

// planningArea builds activities and line items before campaigns, which roll their summaries up.
func (w *wiring) planningArea() error {
	w.buildActivities()
	w.buildCampaigns()
	if err := w.planningKinds(); err != nil {
		return err
	}
	mount(w.signedIn, campaign.NewHandler(w.campaigns), activity.NewHandler(w.activities),
		lineitem.NewHandler(w.lineItems))
	return nil
}

func (w *wiring) buildActivities() {
	s := w.set
	w.activities = activity.NewService(history.Tools{Tx: w.guard, IDs: w.generator, Log: w.history, Now: w.now}, s.Activities, activity.Deps{
		Plans: s.Plans, Hierarchy: s.Campaigns, Settings: w.settings, People: s.People,
		Workflows: activityWorkflows{repo: s.Approvals},
	})
	w.lineItems = lineitem.NewService(history.Tools{Tx: s.Tx, IDs: w.generator, Log: w.history, Now: w.now}, s.LineItems, lineitem.Deps{
		Activities: lineItemActivities{
			activities: s.Activities, campaigns: s.Campaigns, plans: s.Plans, settings: w.settings,
		},
		Currencies: w.settings, FinanceBudgets: lineItemBudgets{repo: s.Settings},
		Attributes: lineItemAttributes{settings: w.settings}, People: lineItemPeople{repo: s.People, service: w.people},
	})
	w.activities.UseLineItems(lineItemsForActivities{svc: w.lineItems})
	w.activities.UseDisplayRates(w.people)
}

func (w *wiring) buildCampaigns() {
	w.campaigns = campaign.NewService(history.Tools{Tx: w.guard, IDs: w.generator, Log: w.history, Now: w.now}, w.set.Campaigns, campaign.Deps{
		Plans: campaignPlans{repo: w.set.Plans}, People: w.set.People, Settings: w.settings,
		Activities: activitiesForCampaigns{svc: w.activities},
	})
	w.campaigns.UseDisplayRates(w.people)
}

// planningKinds registers the kinds and the plan's levels top-down: campaign, initiative, activity, line item.
func (w *wiring) planningKinds() error {
	if err := w.register(overspend.Described(w.campaigns.Kinds()), w.activities.Kinds(), w.lineItems.Kinds()); err != nil {
		return err
	}
	for _, level := range w.planLevels() {
		if err := w.plans.RegisterLevel(level); err != nil {
			return err
		}
	}
	return nil
}

func (w *wiring) planLevels() []plan.Level {
	return append(campaignLevels(w.campaigns), activityLevel(w.activities, w.set.Activities, w.set.Campaigns),
		lineItemLevel(w.lineItems, w.set.Activities, w.set.Campaigns))
}

// budgetsArea comes after the planning kinds: an allocation is a child of its plan, which takes it along.
func (w *wiring) budgetsArea() error {
	tools := history.Tools{Tx: w.guard, IDs: w.generator, Log: w.history, Now: w.now}
	budgetsService := budget.NewService(tools, w.set.Budgets, budget.Deps{
		Plans:     budgetPlans{plans: w.plans, repo: w.set.Plans, rates: w.settings.Rates},
		Directory: w.people, Rates: w.settings, Access: budgetAccess{access: w.access}, Overspend: w.overspend,
	})
	if err := w.register(overspend.Described(budgetsService.Kinds())); err != nil {
		return err
	}
	w.plans.UseAllocations(planAllocations{budgets: budgetsService})
	w.budgets = budgetsService
	mount(w.signedIn, budget.NewHandler(budgetsService))
	return nil
}

func (w *wiring) samplesArea() error {
	if !w.cfg.DemoMode() {
		return nil
	}
	tools := history.Tools{Tx: w.set.Tx, IDs: w.generator, Log: w.history, Now: w.now}
	samplesService := sample.NewService(tools, w.set.Samples)
	if err := w.register(samplesService.Kinds()); err != nil {
		return err
	}
	mount(w.signedIn, sample.NewHandler(samplesService))
	return nil
}

func (w *wiring) approvalsArea() error {
	s := w.set
	tools := history.Tools{Tx: s.Tx, IDs: w.generator, Log: w.history, Now: w.now}
	w.approvals = approval.NewService(tools, s.Approvals, s.People, &approvalsActivities{
		activities: s.Activities, lineItems: s.LineItems, campaigns: s.Campaigns, plans: s.Plans, settings: s.Settings,
		rates: w.settings.Rates, check: w.activities.SubmitCheck, now: w.now,
	})
	if err := w.register(w.approvals.Kinds()); err != nil {
		return err
	}
	w.connectApprovals()
	w.accessTargets()
	mount(w.signedIn, approval.NewHandler(w.approvals))
	return nil
}

func (w *wiring) connectApprovals() {
	w.activities.UseApprovalHooks(activityApprovalHooks{svc: w.approvals})
	w.approvals.UseDisplayRates(w.people)
	w.approvals.UseAccess(approvalsAccess{access: w.access})
	w.approvals.UseRuleChoices(approvalRuleChoices{settings: w.settings})
	w.access.UseApprovalFacts(accessApprovals{repo: w.set.Approvals})
	w.approvals.UseApprovers(w.access)
}

func (w *wiring) accessTargets() {
	s := w.set
	w.access.UseTargets(planningTargets{db: s.Tx, plans: s.Plans, campaigns: s.Campaigns, activities: s.Activities,
		lineItems: s.LineItems, approvals: s.Approvals}.finders())
}

func (w *wiring) financialsArea() error {
	s := w.set
	w.finance = financialPlanning{
		plans: s.Plans, campaigns: w.campaigns, campaignsRepo: s.Campaigns, activities: s.Activities,
		lineItems: s.LineItems, settings: w.settings, people: w.people, history: w.history,
	}
	tools := history.Tools{Tx: s.Tx, IDs: w.generator, Log: w.history, Now: w.now}
	w.financials = financials.NewService(tools, s.Financials, financials.Deps{
		Planning: w.finance, Rates: w.settings, Displays: financialDisplays{people: w.people},
		Access: financialAccess{tx: s.Tx, plans: s.Plans},
	})
	if err := w.register(w.financials.Kinds()); err != nil {
		return err
	}
	mount(w.signedIn, financials.NewHandler(w.financials))
	return nil
}

// views wires Home and All work, read-only views over the other areas; All work's saved views and columns
// are personal settings, not business records, so they have no history kinds (D-121).
func (w *wiring) views() {
	s := w.set
	homeService := home.NewService(s.Tx, homeReaders{
		people: w.people, plansRepo: s.Plans, plans: w.plans, campaigns: w.campaigns, activities: w.activities,
		approvals: w.approvals, history: w.history, rates: w.settings.Rates,
	}.sources())
	w.allWork = allWorkSource{
		plans: w.plans, plansRepo: s.Plans, campaigns: w.campaigns, campaignsRepo: s.Campaigns,
		activities: w.activities, activitiesRepo: s.Activities, lineItems: w.lineItems,
		lineItemsRepo: s.LineItems, settings: w.settings, budgetItems: lineItemBudgets{repo: s.Settings},
		workflows: activityWorkflows{repo: s.Approvals}, people: w.people, peopleRepo: s.People,
		history: w.history, financials: w.financials, financeAccess: financialAccess{tx: s.Tx, plans: s.Plans},
	}
	mount(w.signedIn, home.NewHandler(homeService),
		allwork.NewHandler(allwork.NewService(history.Tools{Tx: s.Tx, IDs: w.generator, Log: w.history, Now: w.now},
			s.AllWork, w.allWork)))
}

func (w *wiring) usesOfPeople() {
	w.people.UsePlans(planUses(w.plans))
	w.people.UseCurrencies(peopleCurrencies(w.settings))
	for _, usage := range personUsages(w.set.Campaigns, w.set.Activities, w.set.LineItems, w.set.Approvals) {
		w.people.AddPersonUsage(usage)
	}
}

// usesInSettings feeds Settings' "Used by", in-use deletes and impact statements, the recycle bin included.
func (w *wiring) usesInSettings() {
	s := w.set
	planning := planningRecords{history: w.history, campaigns: s.Campaigns, activities: s.Activities, lineItems: s.LineItems}
	w.settings.AddReferenceSource(personReferences(w.people))
	w.settings.AddReferenceSource(planReferences(w.plans))
	w.settings.AddReferenceSource(planning.references)
}
