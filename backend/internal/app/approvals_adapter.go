package app

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/history"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/stores"
)

type submitCheck func(ctx context.Context, a actor.Actor, activityID string) (activity.SubmitCheck, error)

// approvalsActivities implements approval.Activities; every method runs inside the caller's transaction.
type approvalsActivities struct {
	activities stores.ActivityStore
	lineItems  stores.LineItemStore
	campaigns  stores.CampaignStore
	plans      stores.PlanStore
	settings   stores.SettingsStore
	rates      func(context.Context) (money.Rates, error)
	check      submitCheck
	now        clock.Clock
}

var _ approval.Activities = (*approvalsActivities)(nil)

func (p *approvalsActivities) Activity(ctx context.Context, id string) (approval.ActivityFacts, bool, error) {
	act, found, err := p.activities.Get(ctx, id)
	if err != nil || !found {
		return approval.ActivityFacts{}, false, err
	}
	facts, err := p.facts(ctx, act)
	return facts, err == nil, err
}

func (p *approvalsActivities) ListOf(ctx context.Context, ids []string) ([]approval.ActivityFacts, error) {
	stored, err := p.activities.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return p.factsOf(ctx, stored)
}

func (p *approvalsActivities) WorkflowFields(ctx context.Context) (map[string]string, error) {
	return p.activities.WorkflowIDs(ctx)
}

func (p *approvalsActivities) ListInScope(ctx context.Context, scope authz.Scope, ids []string) ([]approval.ActivityFacts, error) {
	seen, err := p.activities.ListInScopeOf(ctx, scope, ids)
	if err != nil {
		return nil, err
	}
	return p.factsOf(ctx, seen)
}

func (p *approvalsActivities) factsOf(ctx context.Context, all []activity.Activity) ([]approval.ActivityFacts, error) {
	if err := p.readAround(ctx, all); err != nil {
		return nil, err
	}
	list := make([]approval.ActivityFacts, 0, len(all))
	for _, act := range all {
		facts, err := p.facts(ctx, act)
		if err != nil {
			return nil, err
		}
		list = append(list, facts)
	}
	return list, nil
}

func (p *approvalsActivities) SetStatus(ctx context.Context, a actor.Actor, id, status string) (approval.ActivityFacts, error) {
	act, found, err := p.activities.Get(ctx, id)
	if err != nil {
		return approval.ActivityFacts{}, err
	}
	if !found {
		return approval.ActivityFacts{}, fmt.Errorf("app: activity %s is not stored", id)
	}
	act.Status = activity.Status(status)
	act.Touch(a, p.now())
	if err := p.activities.Put(ctx, act); err != nil {
		return approval.ActivityFacts{}, err
	}
	return p.facts(ctx, act)
}

// neededToSubmit ends the message of an empty needed-to-submit field: "Owner — needed to submit".
const neededToSubmit = " — needed to submit"

// ReadyCheck adds a positive unassigned planned cost as the confirmation's note; a line item's problem is
// keyed lineItems.<id>.<field>.
func (p *approvalsActivities) ReadyCheck(ctx context.Context, a actor.Actor, id string) (approval.ReadyCheck, error) {
	check, err := p.check(ctx, a, id)
	if err != nil {
		return approval.ReadyCheck{}, err
	}
	result := approval.ReadyCheck{Problems: submitProblems(check), Missing: missingToSubmit(check)}
	act, found, err := p.activities.Get(ctx, id)
	if err != nil || !found {
		return result, err
	}
	unassigned, err := p.unassigned(ctx, act)
	if err != nil {
		return approval.ReadyCheck{}, err
	}
	if unassigned != nil && unassigned.Amount.Sign() > 0 {
		result.UnassignedPlannedCost = unassigned
	}
	return result, nil
}

func submitProblems(check activity.SubmitCheck) []approval.SubmitProblem {
	problems := make([]approval.SubmitProblem, 0, check.ProblemCount)
	for _, problem := range check.Activity {
		problems = append(problems, approval.SubmitProblem{Field: problem.Field, Group: approval.GroupActivity, Message: problem.Message})
	}
	for _, problem := range check.LineItems {
		converted := approval.SubmitProblem{Field: problem.Field, Group: approval.GroupLineItem, Message: problem.Message}
		if problem.LineItemID != nil {
			converted.LineItemID = *problem.LineItemID
			converted.Field = "lineItems." + *problem.LineItemID + "." + problem.Field
		}
		problems = append(problems, converted)
	}
	return problems
}

func missingToSubmit(check activity.SubmitCheck) []approval.MissingFieldRef {
	fields := []approval.MissingFieldRef{}
	for _, problem := range check.Activity {
		if problem.Code == activity.ProblemMissingField {
			fields = append(fields, approval.MissingFieldRef{ID: problem.Field, Label: strings.TrimSuffix(problem.Message, neededToSubmit)})
		}
	}
	return fields
}

// facts derives what approvals needs of act now: its path, type name, display status and costs.
func (p *approvalsActivities) facts(ctx context.Context, act activity.Activity) (approval.ActivityFacts, error) {
	place, err := p.placeOf(ctx, act.InitiativeID)
	if err != nil {
		return approval.ActivityFacts{}, err
	}
	path, pl := slices.Clone(place.path), place.plan
	units, err := p.unitsOf(ctx, pl)
	if err != nil {
		return approval.ActivityFacts{}, err
	}
	inPlan, err := p.unassignedInPlan(ctx, act)
	if err != nil {
		return approval.ActivityFacts{}, err
	}
	unassigned, err := p.inUSD(ctx, inPlan)
	if err != nil {
		return approval.ActivityFacts{}, err
	}
	planned, err := p.inUSD(ctx, act.PlannedCost)
	if err != nil {
		return approval.ActivityFacts{}, err
	}
	facts := approval.ActivityFacts{
		ID: act.ID, Name: act.Name, Status: string(act.Status), DisplayStatus: displayStatus(act, date.Today(p.now())),
		Path: path, OwnerID: act.OwnerID, StartDate: act.StartDate, EndDate: act.EndDate, Ongoing: act.Ongoing,
		PlannedCost: planned, UnassignedPlannedCost: unassigned, OverPlannedCost: overPlannedCost(inPlan),
		Description: act.Description,
		WorkflowID:  act.ApprovalWorkflowID, Meta: act.Meta,
		Plan:           approval.PlanFacts{ID: pl.ID, TeamIDs: slices.Clone(pl.TeamIDs), OwnerID: pl.OwnerID, CreatedByID: pl.CreatedBy.ID},
		ActivityTypeID: act.ActivityTypeID, BusinessUnits: units,
		Access: plan.AccessFacts(pl),
	}
	if act.ActivityTypeID != "" {
		activityType, found, err := p.settings.GetActivityType(ctx, act.ActivityTypeID)
		if err != nil {
			return approval.ActivityFacts{}, err
		}
		if found {
			facts.TypeName = activityType.Name
		}
	}
	return facts, nil
}

// displayStatus is the status shown (006 FR-021): an approved activity is Complete after its end date (never
// when ongoing), In progress from its start date, otherwise its stored status.
func displayStatus(act activity.Activity, today date.Date) string {
	switch {
	case act.Status != activity.StatusApproved || act.StartDate == nil:
		return string(act.Status)
	case !act.Ongoing && act.EndDate != nil && today.After(*act.EndDate):
		return string(activity.DisplayComplete)
	case !today.Before(*act.StartDate):
		return string(activity.DisplayInProgress)
	default:
		return string(act.Status)
	}
}

func (p *approvalsActivities) path(ctx context.Context, initiativeID string) ([]errs.RecordRef, plan.Plan, error) {
	initiative, found, err := p.campaigns.GetInitiative(ctx, initiativeID)
	if err != nil {
		return nil, plan.Plan{}, err
	}
	if !found {
		return nil, plan.Plan{}, fmt.Errorf("app: initiative %s is not stored", initiativeID)
	}
	campaign, found, err := p.campaigns.GetCampaign(ctx, initiative.CampaignID)
	if err != nil {
		return nil, plan.Plan{}, err
	}
	if !found {
		return nil, plan.Plan{}, fmt.Errorf("app: campaign %s is not stored", initiative.CampaignID)
	}
	pl, found, err := p.plans.Get(ctx, campaign.PlanID)
	if err != nil {
		return nil, plan.Plan{}, err
	}
	if !found {
		return nil, plan.Plan{}, fmt.Errorf("app: plan %s is not stored", campaign.PlanID)
	}
	return []errs.RecordRef{
		{Type: planKind, ID: pl.ID, Name: pl.Name},
		{Type: campaignLevel, ID: campaign.ID, Name: campaign.Name},
		{Type: initiativeLevel, ID: initiative.ID, Name: initiative.Name},
	}, pl, nil
}

// activityApprovalHooks: a re-approval change ends the activity's approval and a restore ends the round open
// at the delete, inside the activities Write that caused it.
type activityApprovalHooks struct {
	svc *approval.Service
}

var _ activity.ApprovalHooks = activityApprovalHooks{}

func (h activityApprovalHooks) ApprovalRestarted(ctx context.Context, a actor.Actor, before, after activity.Activity, changes []history.Change) error { //nolint:revive // argument-limit 5: the signature of activity.ApprovalHooks
	return h.svc.ApprovalRestarted(ctx, a, after.ID, string(before.Status), changes)
}

// ActivityDeleting changes nothing: the open round stays stored so that Undo brings the activity back exactly,
// still Submitted.
func (activityApprovalHooks) ActivityDeleting(context.Context, actor.Actor, activity.Activity) error {
	return nil
}

// ActivityRestored ends the round open at the delete and returns the note of the restore's log entry.
func (h activityApprovalHooks) ActivityRestored(ctx context.Context, rc history.RestoreContext, activityID string) ([]string, error) {
	return h.svc.ActivityRestored(ctx, rc, activityID)
}
