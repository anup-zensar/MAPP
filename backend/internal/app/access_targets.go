package app

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/access"
	"mapp/internal/plans"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/store"
	"mapp/internal/stores"
)

// planningTargets finds the planning records of features 004–007 that operations name by path, for the
// access middleware (feature 014, plan "Authorization model"): each record with its plan's access facts and,
// for an activity and what is under it, its owner and the people named to approve it in any approval round
// (data model §2.2). It reads the stored records directly, so a check costs a few lookups.
type planningTargets struct {
	db         store.Transactor
	plans      stores.PlanStore
	campaigns  stores.CampaignStore
	activities stores.ActivityStore
	lineItems  stores.LineItemStore
	approvals  stores.ApprovalStore
}

// finders returns the finder of each path parameter: the five planning records, and recordId, which All
// work (010) and the financials of a record (015) use for any of them.
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

// inRead runs a finder inside a read transaction.
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

// plan finds the plan id.
func (t planningTargets) plan(ctx context.Context, id string) (access.Target, bool, error) {
	p, found, err := t.plans.Get(ctx, id)
	if err != nil || !found {
		return access.Target{}, false, err
	}
	return access.Target{RecordType: access.RecordPlan, Plan: plans.AccessFacts(p)}, true, nil
}

// campaign finds the campaign id with its plan.
func (t planningTargets) campaign(ctx context.Context, id string) (access.Target, bool, error) {
	c, found, err := t.campaigns.GetCampaign(ctx, id)
	if err != nil || !found {
		return access.Target{}, false, err
	}
	target, found, err := t.plan(ctx, c.PlanID)
	target.RecordType = access.RecordCampaign
	return target, found, err
}

// initiative finds the initiative id with its plan.
func (t planningTargets) initiative(ctx context.Context, id string) (access.Target, bool, error) {
	i, found, err := t.campaigns.GetInitiative(ctx, id)
	if err != nil || !found {
		return access.Target{}, false, err
	}
	target, found, err := t.campaign(ctx, i.CampaignID)
	target.RecordType = access.RecordInitiative
	return target, found, err
}

// activity finds the activity id with its plan, its owner and its named approvers.
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

// lineItem finds the line item id with its activity's facts: whoever sees the activity sees its line items.
func (t planningTargets) lineItem(ctx context.Context, id string) (access.Target, bool, error) {
	li, found, err := t.lineItems.Get(ctx, id)
	if err != nil || !found {
		return access.Target{}, false, err
	}
	target, found, err := t.activity(ctx, li.ActivityID)
	target.RecordType = access.RecordLineItem
	return target, found, err
}

// record finds any planning record by the prefix of its readable id (All work's record location, 010).
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

// namedApprovers returns the ids of the people named to approve the activity in any level of any of its
// approval rounds (008's copy of the levels at submission), without repeats.
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
