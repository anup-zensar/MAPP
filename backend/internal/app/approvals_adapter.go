package app

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/activities"
	"mapp/internal/approvals"
	"mapp/internal/history"
	"mapp/internal/plans"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// submitCheck is feature 006's ready-to-submit check of one activity, line items included (006 plan
// "Provides": (*activities.Service).SubmitCheck).
type submitCheck func(ctx context.Context, a actor.Actor, activityID string) (activities.SubmitCheck, error)

// approvalsActivities implements approvals.Activities (feature 008) over the stored activities (006),
// their line items (007), the plans, campaigns and initiatives above them (004, 005) and Settings (002).
// Every method runs inside the caller's transaction, which ctx carries.
type approvalsActivities struct {
	activities stores.ActivityStore
	lineItems  stores.LineItemStore
	campaigns  stores.CampaignStore
	plans      stores.PlanStore
	settings   stores.SettingsStore
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
