package app

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"northstar/internal/activities"
	"northstar/internal/approvals"
	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/lineitems"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
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

// List returns every stored activity as approvals needs it.
func (p *approvalsActivities) List(ctx context.Context) ([]approvals.ActivityFacts, error) {
	all, err := p.activities.List(ctx)
	if err != nil {
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
	rates, err := p.rates(ctx)
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
			view.PlannedCostUSD = &converted
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
	path, err := p.path(ctx, act.InitiativeID)
	if err != nil {
		return approvals.ActivityFacts{}, err
	}
	unassigned, err := p.unassigned(ctx, act)
	if err != nil {
		return approvals.ActivityFacts{}, err
	}
	facts := approvals.ActivityFacts{
		ID: act.ID, Name: act.Name, Status: string(act.Status), DisplayStatus: displayStatus(act, date.Today(p.now())),
		Path: path, OwnerID: act.OwnerID, StartDate: act.StartDate, EndDate: act.EndDate, Ongoing: act.Ongoing,
		PlannedCost: act.PlannedCost, UnassignedPlannedCost: unassigned, Description: act.Description,
		WorkflowID: act.ApprovalWorkflowID, Meta: act.Meta,
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

// path returns the plan, campaign and initiative of the initiative initiativeID, top level first.
func (p *approvalsActivities) path(ctx context.Context, initiativeID string) ([]errs.RecordRef, error) {
	initiative, found, err := p.campaigns.GetInitiative(ctx, initiativeID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("app: initiative %s is not stored", initiativeID)
	}
	campaign, found, err := p.campaigns.GetCampaign(ctx, initiative.CampaignID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("app: campaign %s is not stored", initiative.CampaignID)
	}
	plan, found, err := p.plans.Get(ctx, campaign.PlanID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("app: plan %s is not stored", campaign.PlanID)
	}
	return []errs.RecordRef{
		{Type: "plan", ID: plan.ID, Name: plan.Name},
		{Type: "campaign", ID: campaign.ID, Name: campaign.Name},
		{Type: "initiative", ID: initiative.ID, Name: initiative.Name},
	}, nil
}

// unassigned is the activity's planned cost (0.00 when empty) minus its line items' planned costs in USD
// at the current rates (domain model §4.3); nil when it has neither a planned cost nor a line item.
func (p *approvalsActivities) unassigned(ctx context.Context, act activities.Activity) (*money.Money, error) {
	items, err := p.lineItems.ListByActivity(ctx, act.ID)
	if err != nil {
		return nil, err
	}
	if act.PlannedCost == nil && len(items) == 0 {
		return nil, nil
	}
	rates, err := p.rates(ctx)
	if err != nil {
		return nil, err
	}
	var balance money.Amount
	if act.PlannedCost != nil {
		balance = act.PlannedCost.Amount
	}
	for _, li := range items {
		usd, err := rates.ToUSD(money.Money{Amount: li.PlannedCost, Currency: li.Currency})
		if err != nil {
			return nil, err
		}
		balance = balance.Sub(usd)
	}
	result := money.USD(balance)
	return &result, nil
}

// rates returns the current rate of every currency (002).
func (p *approvalsActivities) rates(ctx context.Context) (money.Rates, error) {
	currencies, err := p.settings.ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	rates := money.Rates{}
	for _, c := range currencies {
		rates[c.Code] = c.Rate
	}
	return rates, nil
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

// storedFieldsCheck stands in for 006's ready-to-submit check until feature 006 is merged: it lists the
// empty needed-to-submit fields stored on the activity itself (Owner, Start date, End date unless ongoing,
// Planned cost, Approval workflow). Shell attributes, % splits and line items are 006's and 007's rules.
// INTEGRATION: replace it with activitiesService.SubmitCheck in New.
func storedFieldsCheck(repo activities.Repository) submitCheck {
	return func(ctx context.Context, _ actor.Actor, id string) (activities.SubmitCheck, error) {
		act, _, err := repo.Get(ctx, id)
		if err != nil {
			return activities.SubmitCheck{}, err
		}
		missing := []struct {
			empty        bool
			field, label string
		}{
			{act.OwnerID == "", "ownerId", "Owner"},
			{act.StartDate == nil, "startDate", "Start date"},
			{act.EndDate == nil && !act.Ongoing, "endDate", "End date"},
			{act.PlannedCost == nil, "plannedCost", "Planned cost"},
			{act.ApprovalWorkflowID == "", "approvalWorkflowId", "Approval workflow"},
		}
		check := activities.SubmitCheck{
			Activity: []activities.SubmitProblem{}, LineItems: []activities.SubmitProblem{}, Advice: []activities.SubmitProblem{},
		}
		for _, m := range missing {
			if m.empty {
				check.Activity = append(check.Activity, activities.SubmitProblem{
					Code: activities.ProblemMissingField, Field: m.field, Message: m.label + neededToSubmit,
				})
			}
		}
		check.ProblemCount = len(check.Activity)
		check.Ready = check.ProblemCount == 0
		return check, nil
	}
}

// standInActivityKind is the history kind "activity" until feature 006's own is registered: approval
// actions are logged on the activity (008 FR-026). Without a parent or Describe, its entries have no path
// and default texts. INTEGRATION: delete it and its registration in New once 006 registers its kinds.
func standInActivityKind(repo activities.Repository) history.Kind {
	return history.Kind{
		Name: "activity", Singular: "activity", Plural: "activities",
		Snapshot: func(ctx context.Context, id string) (history.Snapshot, bool, error) {
			act, found, err := repo.Get(ctx, id)
			if err != nil || !found {
				return history.Snapshot{}, false, err
			}
			data, err := json.Marshal(act)
			if err != nil {
				return history.Snapshot{}, false, err
			}
			return history.Snapshot{ID: act.ID, Name: act.Name, Data: data}, true, nil
		},
		Remove: repo.Delete,
		Put: func(ctx context.Context, s history.Snapshot) error {
			var act activities.Activity
			if err := json.Unmarshal(s.Data, &act); err != nil {
				return err
			}
			return repo.Put(ctx, act)
		},
	}
}

// activityWorkflows implements activities.Workflows (feature 006) over the stored approval workflows, so
// the activity form offers the active ones and labels the inactive ones (FR-007; 008 plan, Reconciliation).
// Its reads join the caller's transaction. INTEGRATION with 006: pass activityWorkflows{repo:
// approvals.NewJSONRepository()} in activities.Deps.Workflows instead of NewSeededWorkflows().
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
