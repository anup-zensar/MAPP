package app

import (
	"context"
	"slices"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/plan"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/memo"
	"mapp/internal/platform/money"
)

type place struct {
	path []errs.RecordRef
	plan plan.Plan
}

// readAround reads the line items of the activities and the initiatives, campaigns and plans above them, one
// query each, so that each activity's facts find them in the transaction's memo.
func (p *approvalsActivities) readAround(ctx context.Context, list []activity.Activity) error {
	activityIDs, initiativeIDs := make([]string, 0, len(list)), make([]string, 0, len(list))
	for _, act := range list {
		activityIDs, initiativeIDs = append(activityIDs, act.ID), append(initiativeIDs, act.InitiativeID)
	}
	if _, err := p.lineItems.ListByActivities(ctx, activityIDs); err != nil {
		return err
	}
	initiatives, err := p.campaigns.InitiativesByIDs(ctx, initiativeIDs)
	if err != nil {
		return err
	}
	campaignIDs := make([]string, 0, len(initiatives))
	for _, i := range initiatives {
		campaignIDs = append(campaignIDs, i.CampaignID)
	}
	campaigns, err := p.campaigns.CampaignsByIDs(ctx, campaignIDs)
	if err != nil {
		return err
	}
	planIDs := make([]string, 0, len(campaigns))
	for _, c := range campaigns {
		planIDs = append(planIDs, c.PlanID)
	}
	_, err = p.plans.ListByIDs(ctx, planIDs)
	return err
}

// The values below are worked out once per transaction until data changes (the memo's Derived area): the
// Approvals board and a workflow's activities read them for every activity.

func (p *approvalsActivities) placeOf(ctx context.Context, initiativeID string) (place, error) {
	return memo.Get(ctx, memo.Derived+":approvalsPlace:"+initiativeID, func(ctx context.Context) (place, error) {
		path, p, err := p.path(ctx, initiativeID)
		return place{path: path, plan: p}, err
	})
}

func (p *approvalsActivities) unitsOf(ctx context.Context, pl plan.Plan) ([]approval.Choice, error) {
	units, err := memo.Get(ctx, memo.Derived+":approvalsUnits:"+pl.ID, func(ctx context.Context) ([]approval.Choice, error) {
		return p.businessUnits(ctx, pl.Attributes)
	})
	return slices.Clone(units), err
}

func (p *approvalsActivities) keptPlanCurrency(ctx context.Context, initiativeID string) (string, error) {
	return memo.Get(ctx, memo.Derived+":approvalsCurrency:"+initiativeID, func(ctx context.Context) (string, error) {
		return p.planCurrency(ctx, initiativeID)
	})
}

func (p *approvalsActivities) currentRates(ctx context.Context) (money.Rates, error) {
	return memo.Get(ctx, memo.Derived+":rates", p.rates)
}
