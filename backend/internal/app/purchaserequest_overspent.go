package app

import (
	"context"

	"mapp/internal/activity"
	"mapp/internal/platform/money"
)

// overspentActivities returns by how much each activity's line items exceed its planned cost, in the plan currency
// (domain model §4.3), which gates purchase requests (037 FR-019).
func (x prLineItems) overspentActivities(ctx context.Context, acts []activity.Activity) (map[string]*money.Money, error) {
	ids := make([]string, 0, len(acts))
	for _, a := range acts {
		ids = append(ids, a.ID)
	}
	summaries, err := x.lineItems.Summaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	overBy := map[string]*money.Money{}
	for _, a := range acts {
		s, found := summaries[a.ID]
		if !found || a.PlannedCost == nil || s.PlannedCostPlan.Cmp(a.PlannedCost.Amount) <= 0 {
			continue
		}
		over := money.Money{Amount: s.PlannedCostPlan.Sub(a.PlannedCost.Amount), Currency: a.PlannedCost.Currency}
		overBy[a.ID] = &over
	}
	return overBy, nil
}
