package app

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/activities"
	"mapp/internal/approvals"
	"mapp/internal/campaigns"
	"mapp/internal/history"
	"mapp/internal/lineitems"
	"mapp/internal/plans"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// submitCheck is feature 006's ready-to-submit check of one activity, line items included (006 plan
// "Provides": (*activities.Service).SubmitCheck).
type submitCheck func(ctx context.Context, a actor.Actor, activityID string) (activities.SubmitCheck, error)

// approvalsActivities implements approvals.Activities (feature 008) over the stored activities (006),
// their line items (007), the plans, campaigns and initiatives above them (004, 005) and Settings (002).
// Every method runs inside the caller's transaction, which ctx carries.
type approvalsActivities struct {
	activities activities.Repository
	lineItems  lineitems.Repository
	campaigns  campaigns.Repository
	plans      plans.Repository
	settings   settings.Repository
	rates      func(context.Context) (money.Rates, error) // the current rates (016: dated rates), settings.Service.Rates
	check      submitCheck
	now        clock.Clock
}

// The compiler checks here that approvalsActivities implements approvals.Activities.
var _ approvals.Activities = (*approvalsActivities)(nil)

// Activity returns the stored activity id as approvals needs it.
func (p *approvalsActivities) Activity(ctx context.Context, id string) (approvals.ActivityFacts, bool, error) {
	act, found, err := p.activities.Get(ctx, id)
	if err != nil || !found {
		return approvals.ActivityFacts{}, false, err
	}
	facts, err := p.facts(ctx, act)
	return facts, err == nil, err
}

// ListOf returns the stored activities among ids as approvals needs them.
func (p *approvalsActivities) ListOf(ctx context.Context, ids []string) ([]approvals.ActivityFacts, error) {
	stored, err := p.activities.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return p.factsOf(ctx, stored)
}

// WorkflowFields returns the Approval workflow field of every stored activity.
func (p *approvalsActivities) WorkflowFields(ctx context.Context) (map[string]string, error) {
	return p.activities.WorkflowIDs(ctx)
}

// ListInScope returns the stored activities among ids that the scope sees as approvals needs them.
func (p *approvalsActivities) ListInScope(ctx context.Context, scope authz.Scope, ids []string) ([]approvals.ActivityFacts, error) {
	seen, err := p.activities.ListInScopeOf(ctx, scope, ids)
	if err != nil {
		return nil, err
	}
	return p.factsOf(ctx, seen)
}

func (p *approvalsActivities) factsOf(ctx context.Context, all []activities.Activity) ([]approvals.ActivityFacts, error) {
	if err := p.readAround(ctx, all); err != nil {
		return nil, err
	}
	list := make([]approvals.ActivityFacts, 0, len(all))
	for _, act := range all {
		facts, err := p.facts(ctx, act)
		if err != nil {
			return nil, err
		}
		list = append(list, facts)
	}
	return list, nil
}

// SetStatus stores the activity's new status with a new version, last changed by a.
func (p *approvalsActivities) SetStatus(ctx context.Context, a actor.Actor, id, status string) (approvals.ActivityFacts, error) {
	act, found, err := p.activities.Get(ctx, id)
	if err != nil {
		return approvals.ActivityFacts{}, err
	}
	if !found {
		return approvals.ActivityFacts{}, fmt.Errorf("app: activity %s is not stored", id)
	}
	act.Status = activities.Status(status)
	act.Touch(a, p.now())
	if err := p.activities.Put(ctx, act); err != nil {
		return approvals.ActivityFacts{}, err
	}
	return p.facts(ctx, act)
}

// neededToSubmit ends the message of an empty needed-to-submit field: "Owner — needed to submit".
const neededToSubmit = " — needed to submit"

// ReadyCheck runs 006's ready-to-submit check and adds a positive unassigned planned cost as the note of
// the confirmation (FR-011). A line item's problem is keyed lineItems.<id>.<field> (contract SubmitProblem).
// The activity's empty needed-to-submit fields are its completeness, which the preview lists (FR-032).
func (p *approvalsActivities) ReadyCheck(ctx context.Context, a actor.Actor, id string) (approvals.ReadyCheck, error) {
	check, err := p.check(ctx, a, id)
	if err != nil {
		return approvals.ReadyCheck{}, err
	}
	problems := make([]approvals.SubmitProblem, 0, check.ProblemCount)
	missing := []approvals.MissingFieldRef{}
	for _, problem := range check.Activity {
		problems = append(problems, approvals.SubmitProblem{Field: problem.Field, Group: approvals.GroupActivity, Message: problem.Message})
		if problem.Code == activities.ProblemMissingField {
			missing = append(missing, approvals.MissingFieldRef{ID: problem.Field, Label: strings.TrimSuffix(problem.Message, neededToSubmit)})
		}
	}
	for _, problem := range check.LineItems {
		converted := approvals.SubmitProblem{Field: problem.Field, Group: approvals.GroupLineItem, Message: problem.Message}
		if problem.LineItemID != nil {
			converted.LineItemID = *problem.LineItemID
			converted.Field = "lineItems." + *problem.LineItemID + "." + problem.Field
		}
		problems = append(problems, converted)
	}

	result := approvals.ReadyCheck{Problems: problems, Missing: missing}
	act, found, err := p.activities.Get(ctx, id)
	if err != nil || !found {
		return result, err
	}
	unassigned, err := p.unassigned(ctx, act)
	if err != nil {
		return approvals.ReadyCheck{}, err
	}
	if unassigned != nil && unassigned.Amount.Sign() > 0 {
		result.UnassignedPlannedCost = unassigned
	}
	return result, nil
}

// LineItems returns the activity's line items with their planned cost in their own currency and in USD at
// the current rate, and the readable labels of their finance coding (FR-032).
func (p *approvalsActivities) LineItems(ctx context.Context, _ actor.Actor, activityID string) ([]approvals.PreviewLineItem, error) {
	items, err := p.lineItems.ListByActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	rates, err := p.currentRates(ctx)
	if err != nil {
		return nil, err
	}
	labels, err := p.codingLabels(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]approvals.PreviewLineItem, 0, len(items))
	for _, li := range items {
		view := approvals.PreviewLineItem{
			ID: li.ID, Name: li.Name, PlannedCost: money.Money{Amount: li.PlannedCost, Currency: li.Currency},
			SpendType: labels.of(li.Attributes, "Spend type"), GLCode: labels.of(li.Attributes, "GL code"),
		}
		if usd, err := rates.ToUSD(view.PlannedCost); err == nil {
			converted := money.USD(usd)
			view.PlannedCostDisplay = &converted // approvals converts it to the reader's display currency
		}
		if li.FinanceBudgetItemID != "" {
			item, found, err := p.settings.GetFinanceBudgetItem(ctx, li.FinanceBudgetItemID)
			if err != nil {
				return nil, err
			}
			if found {
				view.FinanceBudgetItem = item.Name
			}
		}
		list = append(list, view)
	}
	return list, nil
}

// facts returns act as approvals needs it: with its plan, campaign and initiative, type name, display
// status and unassigned planned cost, all derived now (D-041).
func (p *approvalsActivities) facts(ctx context.Context, act activities.Activity) (approvals.ActivityFacts, error) {
	place, err := p.placeOf(ctx, act.InitiativeID)
	if err != nil {
		return approvals.ActivityFacts{}, err
	}
	path, plan := slices.Clone(place.path), place.plan
	units, err := p.unitsOf(ctx, plan)
	if err != nil {
		return approvals.ActivityFacts{}, err
	}
	unassigned, err := p.unassigned(ctx, act)
	if err != nil {
		return approvals.ActivityFacts{}, err
	}
	planned, err := p.inUSD(ctx, act.PlannedCost)
	if err != nil {
		return approvals.ActivityFacts{}, err
	}
	facts := approvals.ActivityFacts{
		ID: act.ID, Name: act.Name, Status: string(act.Status), DisplayStatus: displayStatus(act, date.Today(p.now())),
		Path: path, OwnerID: act.OwnerID, StartDate: act.StartDate, EndDate: act.EndDate, Ongoing: act.Ongoing,
		PlannedCost: planned, UnassignedPlannedCost: unassigned, Description: act.Description,
		WorkflowID: act.ApprovalWorkflowID, Meta: act.Meta,
		// 017: what the approval rules and 014's scopes look at (data-model §3).
		Plan:           approvals.PlanFacts{ID: plan.ID, TeamIDs: slices.Clone(plan.TeamIDs), OwnerID: plan.OwnerID, CreatedByID: plan.CreatedBy.ID},
		ActivityTypeID: act.ActivityTypeID, BusinessUnits: units,
		Access: plans.AccessFacts(plan), // 014: who sees the activity and who may submit, recall or decide
	}
	if act.ActivityTypeID != "" {
		activityType, found, err := p.settings.GetActivityType(ctx, act.ActivityTypeID)
		if err != nil {
			return approvals.ActivityFacts{}, err
		}
		if found {
			facts.TypeName = activityType.Name
		}
	}
	return facts, nil
}

// displayStatus is the status shown (006 FR-021): an approved activity is Complete after its end date (never
// when ongoing), In progress from its start date, otherwise its stored status.
func displayStatus(act activities.Activity, today date.Date) string {
	switch {
	case act.Status != activities.StatusApproved || act.StartDate == nil:
		return string(act.Status)
	case !act.Ongoing && act.EndDate != nil && today.After(*act.EndDate):
		return string(activities.DisplayComplete)
	case !today.Before(*act.StartDate):
		return string(activities.DisplayInProgress)
	default:
		return string(act.Status)
	}
}

// path returns the plan, campaign and initiative of the initiative initiativeID, top level first, and the
// plan itself.
func (p *approvalsActivities) path(ctx context.Context, initiativeID string) ([]errs.RecordRef, plans.Plan, error) {
	initiative, found, err := p.campaigns.GetInitiative(ctx, initiativeID)
	if err != nil {
		return nil, plans.Plan{}, err
	}
	if !found {
		return nil, plans.Plan{}, fmt.Errorf("app: initiative %s is not stored", initiativeID)
	}
	campaign, found, err := p.campaigns.GetCampaign(ctx, initiative.CampaignID)
	if err != nil {
		return nil, plans.Plan{}, err
	}
	if !found {
		return nil, plans.Plan{}, fmt.Errorf("app: campaign %s is not stored", initiative.CampaignID)
	}
	plan, found, err := p.plans.Get(ctx, campaign.PlanID)
	if err != nil {
		return nil, plans.Plan{}, err
	}
	if !found {
		return nil, plans.Plan{}, fmt.Errorf("app: plan %s is not stored", campaign.PlanID)
	}
	return []errs.RecordRef{
		{Type: "plan", ID: plan.ID, Name: plan.Name},
		{Type: "campaign", ID: campaign.ID, Name: campaign.Name},
		{Type: "initiative", ID: initiative.ID, Name: initiative.Name},
	}, plan, nil
}

// businessUnits returns the plan's business units (017 data-model §3): the options chosen in the plan's
// values of the plan-level Business unit attribute (002, Attribute.Role businessUnit), with their labels
// and active flags now; none without that attribute or value.
func (p *approvalsActivities) businessUnits(ctx context.Context, values settings.AttributeValues) ([]approvals.Choice, error) {
	attributes, err := p.settings.ListAttributes(ctx)
	if err != nil {
		return nil, err
	}
	units := []approvals.Choice{}
	for _, a := range attributes {
		if a.Role != settings.RoleBusinessUnit || a.Level != settings.LevelPlan {
			continue
		}
		var ids []string
		if raw, found := values[a.ID]; found && json.Unmarshal(raw, &ids) != nil {
			var one string
			if json.Unmarshal(raw, &one) == nil && one != "" {
				ids = []string{one}
			}
		}
		for _, id := range ids {
			option, found, err := p.settings.GetOption(ctx, id)
			if err != nil {
				return nil, err
			}
			if found {
				units = append(units, approvals.Choice{ID: option.ID, Label: option.Label, Active: option.Active})
			}
		}
	}
	return units, nil
}

// unassigned is the activity's planned cost (0.00 when empty) minus its line items' planned costs in the plan
// currency at the current rates (domain model §4.3, 016 FR-015), converted to USD once — approvals works in
// USD and shows the reader's display currency (D-213); nil when it has neither a planned cost nor a line item.
func (p *approvalsActivities) unassigned(ctx context.Context, act activities.Activity) (*money.Money, error) {
	items, err := p.lineItems.ListByActivity(ctx, act.ID)
	if err != nil {
		return nil, err
	}
	if act.PlannedCost == nil && len(items) == 0 {
		return nil, nil
	}
	rates, err := p.currentRates(ctx)
	if err != nil {
		return nil, err
	}
	currency, err := p.keptPlanCurrency(ctx, act.InitiativeID)
	if err != nil {
		return nil, err
	}
	var balance money.Amount
	if act.PlannedCost != nil {
		balance = act.PlannedCost.Amount
	}
	for _, li := range items {
		inPlan, err := rates.Convert(money.Money{Amount: li.PlannedCost, Currency: li.Currency}, currency)
		if err != nil {
			return nil, err
		}
		balance = balance.Sub(inPlan.Amount)
	}
	return p.inUSD(ctx, &money.Money{Amount: balance, Currency: currency})
}

// inUSD converts an amount of a plan currency to USD at the current rate (016 FR-014); nil stays nil.
func (p *approvalsActivities) inUSD(ctx context.Context, m *money.Money) (*money.Money, error) {
	if m == nil || m.Currency == money.BaseCurrency {
		return m, nil
	}
	rates, err := p.currentRates(ctx)
	if err != nil {
		return nil, err
	}
	usd, err := rates.Convert(*m, money.BaseCurrency)
	if err != nil {
		return nil, err
	}
	return &usd, nil
}

// planCurrency returns the plan currency of the initiative initiativeID's plan (016 FR-011).
func (p *approvalsActivities) planCurrency(ctx context.Context, initiativeID string) (string, error) {
	initiative, found, err := p.campaigns.GetInitiative(ctx, initiativeID)
	if err != nil || !found {
		return money.BaseCurrency, err
	}
	campaign, found, err := p.campaigns.GetCampaign(ctx, initiative.CampaignID)
	if err != nil || !found {
		return money.BaseCurrency, err
	}
	plan, found, err := p.plans.Get(ctx, campaign.PlanID)
	if err != nil || !found {
		return money.BaseCurrency, err
	}
	return plan.Currency, nil
}

// coding finds the labels of the single-choice values of the line-item attributes, by attribute label.
type coding struct {
	attributeIDs map[string]string // attribute label → id
	options      map[string]string // option id → label
}

// codingLabels reads the line-item attributes and their options (002).
func (p *approvalsActivities) codingLabels(ctx context.Context) (coding, error) {
	labels := coding{attributeIDs: map[string]string{}, options: map[string]string{}}
	attributes, err := p.settings.ListAttributes(ctx)
	if err != nil {
		return coding{}, err
	}
	for _, attribute := range attributes {
		if attribute.Level == settings.LevelLineItem {
			labels.attributeIDs[attribute.Label] = attribute.ID
		}
	}
	options, err := p.settings.ListOptions(ctx)
	if err != nil {
		return coding{}, err
	}
	for _, option := range options {
		labels.options[option.ID] = option.Label
	}
	return labels, nil
}

// of returns the label of the option chosen in values for the attribute labelled attributeLabel, "" when
// none is chosen.
func (c coding) of(values settings.AttributeValues, attributeLabel string) string {
	var optionID string
	if raw, found := values[c.attributeIDs[attributeLabel]]; !found || json.Unmarshal(raw, &optionID) != nil {
		return ""
	}
	return c.options[optionID]
}

// activityWorkflows implements activities.Workflows (feature 006) over the stored approval workflows, so
// the activity form offers the active ones and labels the inactive ones (FR-007; 008 plan, Reconciliation).
// Its reads join the caller's transaction.
type activityWorkflows struct {
	repo approvals.Repository
}

// The compiler checks here that activityWorkflows implements activities.Workflows.
var _ activities.Workflows = activityWorkflows{}

// GetWorkflow returns the workflow id, active or not, and false when there is none.
func (w activityWorkflows) GetWorkflow(ctx context.Context, id string) (activities.WorkflowFacts, bool, error) {
	workflow, found, err := w.repo.GetWorkflow(ctx, id)
	if err != nil || !found {
		return activities.WorkflowFacts{}, false, err
	}
	return workflowFacts(workflow), true, nil
}

// ListWorkflows returns every workflow, active and inactive, sorted by name ignoring letter case.
func (w activityWorkflows) ListWorkflows(ctx context.Context) ([]activities.WorkflowFacts, error) {
	workflows, err := w.repo.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]activities.WorkflowFacts, 0, len(workflows))
	for _, workflow := range workflows {
		list = append(list, workflowFacts(workflow))
	}
	slices.SortStableFunc(list, func(x, y activities.WorkflowFacts) int {
		return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
	})
	return list, nil
}

// workflowFacts returns a stored workflow as activities needs it.
func workflowFacts(w approvals.Workflow) activities.WorkflowFacts {
	levels := make([]activities.WorkflowLevelFacts, 0, len(w.Levels))
	for _, l := range w.Levels {
		levels = append(levels, activities.WorkflowLevelFacts{Name: l.Name, ApproverIDs: slices.Clone(l.ApproverIDs)})
	}
	return activities.WorkflowFacts{ID: w.ID, Name: w.Name, Active: w.Active, Levels: levels}
}

// activityApprovalHooks implements activities.ApprovalHooks (feature 006) over the approvals service (008
// plan, Reconciliation): a re-approval change ends the activity's approval (FR-022) and a restore from the
// recycle bin ends the round that was open at the delete (FR-023). Every call runs inside the activities
// Write that caused it.
type activityApprovalHooks struct {
	svc *approvals.Service
}

// The compiler checks here that activityApprovalHooks implements activities.ApprovalHooks.
var _ activities.ApprovalHooks = activityApprovalHooks{}

// ApprovalRestarted ends the approval of after, which a re-approval change sent back from before's status
// to Created.
func (h activityApprovalHooks) ApprovalRestarted(ctx context.Context, a actor.Actor, before, after activities.Activity, changes []history.Change) error {
	return h.svc.ApprovalRestarted(ctx, a, after.ID, string(before.Status), changes)
}

// ActivityDeleting changes nothing: the open round of a deleted activity stays stored as it is, so that an
// Undo brings the activity back exactly, still Submitted (008-P3, D-031). The activity simply leaves every
// approvals list while it is not stored.
func (activityApprovalHooks) ActivityDeleting(context.Context, actor.Actor, activities.Activity) error {
	return nil
}

// ActivityRestored ends the round that was open at the delete, now that a restore brings the activity back
// Created, and returns the note for the "restored" log entry.
func (h activityApprovalHooks) ActivityRestored(ctx context.Context, rc history.RestoreContext, activityID string) ([]string, error) {
	return h.svc.ActivityRestored(ctx, rc, activityID)
}
