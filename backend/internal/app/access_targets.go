package app

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/access"
	"mapp/internal/plan"
	"mapp/internal/platform/authz"
	"mapp/internal/stores"
)

// planningTargets finds, for the access middleware, the record an operation names by path with its plan's
// access facts and, for an activity and what is under it, its owner and the people named to approve it.
type planningTargets struct {
	db         stores.Transactor
	plans      stores.PlanStore
	campaigns  stores.CampaignStore
	activities stores.ActivityStore
	lineItems  stores.LineItemStore
	approvals  stores.ApprovalStore
}

// finders also finds recordId, All work's and the financials' id of any of the five planning records.
func (t planningTargets) finders() map[string]access.TargetFinder {
	return map[string]access.TargetFinder{
		"planId":       t.inRead(t.plan),
		"campaignId":   t.inRead(t.campaign),
		"initiativeId": t.inRead(t.initiative),
		"activityId":   t.inRead(t.activity),
		"lineItemId":   t.inRead(t.lineItem),
		"recordId":     t.inRead(t.record),
	}
}

func (t planningTargets) inRead(find access.TargetFinder) access.TargetFinder {
	return func(ctx context.Context, id string) (access.Target, bool, error) {
		var target access.Target
		var found bool
		err := t.db.Read(ctx, func(ctx context.Context) error {
			var err error
			target, found, err = find(ctx, id)
			return err
		})
		return target, found, err
	}
}

func (t planningTargets) plan(ctx context.Context, id string) (access.Target, bool, error) {
	p, found, err := t.plans.Get(ctx, id)
	if err != nil || !found {
		return access.Target{}, false, err
	}
	return access.Target{RecordType: access.RecordPlan, Plan: plan.AccessFacts(p)}, true, nil
}

func (t planningTargets) campaign(ctx context.Context, id string) (access.Target, bool, error) {
	c, found, err := t.campaigns.GetCampaign(ctx, id)
	if err != nil || !found {
		return access.Target{}, false, err
	}
	target, found, err := t.plan(ctx, c.PlanID)
	target.RecordType = access.RecordCampaign
	return target, found, err
}

func (t planningTargets) initiative(ctx context.Context, id string) (access.Target, bool, error) {
	i, found, err := t.campaigns.GetInitiative(ctx, id)
	if err != nil || !found {
		return access.Target{}, false, err
	}
	target, found, err := t.campaign(ctx, i.CampaignID)
	target.RecordType = access.RecordInitiative
	return target, found, err
}

func (t planningTargets) activity(ctx context.Context, id string) (access.Target, bool, error) {
	a, found, err := t.activities.Get(ctx, id)
	if err != nil || !found {
		return access.Target{}, false, err
	}
	target, found, err := t.initiative(ctx, a.InitiativeID)
	if err != nil || !found {
		return access.Target{}, false, err
	}
	approverIDs, err := namedApprovers(ctx, t.approvals, a.ID)
	if err != nil {
		return access.Target{}, false, err
	}
	target.RecordType = access.RecordActivity
	target.Activity = &authz.ActivityFacts{Plan: target.Plan, OwnerID: a.OwnerID, ApproverIDs: approverIDs}
	return target, true, nil
}

// lineItem gives a line item its activity's facts: whoever sees the activity sees its line items.
func (t planningTargets) lineItem(ctx context.Context, id string) (access.Target, bool, error) {
	li, found, err := t.lineItems.Get(ctx, id)
	if err != nil || !found {
		return access.Target{}, false, err
	}
	target, found, err := t.activity(ctx, li.ActivityID)
	target.RecordType = access.RecordLineItem
	return target, found, err
}

// record finds any planning record by the prefix of its readable id.
func (t planningTargets) record(ctx context.Context, id string) (access.Target, bool, error) {
	prefix, _, _ := strings.Cut(id, "-")
	switch prefix {
	case "P":
		return t.plan(ctx, id)
	case "C":
		return t.campaign(ctx, id)
	case "I":
		return t.initiative(ctx, id)
	case "A":
		return t.activity(ctx, id)
	case "L":
		return t.lineItem(ctx, id)
	}
	return access.Target{}, false, nil
}

// namedApprovers returns, without repeats, everyone named at any level of any of the activity's rounds.
func namedApprovers(ctx context.Context, repo stores.ApprovalStore, activityID string) ([]string, error) {
	rounds, err := repo.RoundsOf(ctx, activityID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, r := range rounds {
		for _, level := range r.Levels {
			for _, approver := range level.Approvers {
				if !slices.Contains(ids, approver.ID) {
					ids = append(ids, approver.ID)
				}
			}
		}
	}
	return ids, nil
}
