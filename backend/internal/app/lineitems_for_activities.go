package app

import (
	"context"

	"mapp/internal/activity"
	"mapp/internal/lineitem"
	"mapp/internal/platform/actor"
	"mapp/internal/settings"
)

// The line-item problem codes activities shows in its Ready to submit panel (007 data model §3.6).
const (
	lineItemMissingField         = "missingField"
	lineItemSplitIncomplete      = "splitIncomplete"
	lineItemInactiveBudgetItem   = "inactiveFinanceBudgetItem"
	lineItemCustomSpreadMismatch = "customSpreadMismatch"
)

// lineItemsForActivities is activity.LineItems over the line-items service.
type lineItemsForActivities struct {
	svc *lineitem.Service
}

// Facts returns each activity's line items in short, with every missing field of each incomplete line item
// as a problem.
func (x lineItemsForActivities) Facts(ctx context.Context, activityIDs []string) (map[string]activity.LineItemFacts, error) {
	summaries, err := x.svc.Summaries(ctx, activityIDs)
	if err != nil {
		return nil, err
	}
	facts := make(map[string]activity.LineItemFacts, len(summaries))
	for id, s := range summaries {
		f := activity.LineItemFacts{Count: s.Count, PlannedCostPlan: s.PlannedCostPlan, Currency: s.Currency,
			AutoSpread: s.AutoSpread, CustomSpread: s.CustomSpread}
		for _, item := range s.Incomplete {
			for _, m := range item.Missing {
				f.Problems = append(f.Problems, lineItemProblem(item, m))
			}
		}
		facts[id] = f
	}
	return facts, nil
}

// lineItemProblem is one missing field of a line item as a problem of its activity.
func lineItemProblem(item lineitem.Incomplete, m settings.MissingField) activity.LineItemProblem {
	p := activity.LineItemProblem{
		LineItemID: item.LineItemID, LineItemName: item.LineItemName, Field: m.Field,
		Code: lineItemMissingField, Message: m.Label + " — needed to submit",
	}
	switch {
	case m.Field == "spread":
		p.Code, p.Message = lineItemCustomSpreadMismatch, m.Label
	case m.Field == "financeBudgetItemId" && m.Label != "Finance budget item":
		p.Code, p.Message = lineItemInactiveBudgetItem, m.Label
	case m.Reason != "":
		p.Code, p.Message = lineItemSplitIncomplete, m.Reason
	}
	return p
}

// CopyAll copies every line item of an activity into another activity: the copies of an activity, and of
// the campaigns and initiatives above it (006 FR-030, 005 FR-027).
func (x lineItemsForActivities) CopyAll(ctx context.Context, a actor.Actor, fromActivityID, toActivityID string, yearShift int) (int, error) { //nolint:revive // argument-limit 5: the signature of activity.LineItems
	return x.svc.CopyLineItems(ctx, a, fromActivityID, toActivityID, yearShift)
}

// CopyMany copies the line items of many activities at once: a campaign or initiative copy.
func (x lineItemsForActivities) CopyMany(ctx context.Context, a actor.Actor, pairs []activity.CopyPair, yearShift int) (int, error) {
	converted := make([]lineitem.CopyPair, 0, len(pairs))
	for _, p := range pairs {
		converted = append(converted, lineitem.CopyPair{FromID: p.FromID, ToID: p.ToID})
	}
	return x.svc.CopyLineItemsOf(ctx, a, converted, yearShift)
}

// DatesChanged needs nothing stored: a line item spread Evenly, First or Last month follows its activity's
// dates when it is read, and a custom spread is kept as typed (007 FR-022, D-141).
func (lineItemsForActivities) DatesChanged(context.Context, actor.Actor, activity.Activity, activity.Activity) error {
	return nil
}

// Items returns the line items of an activity in short, with their planned costs in the plan currency (016
// FR-026, the Split dialog).
func (x lineItemsForActivities) Items(ctx context.Context, activityID string) ([]activity.LineItemBrief, error) {
	briefs, err := x.svc.Briefs(ctx, activityID)
	if err != nil {
		return nil, err
	}
	items := make([]activity.LineItemBrief, 0, len(briefs))
	for _, b := range briefs {
		items = append(items, activity.LineItemBrief{ID: b.ID, Name: b.Name, PlannedCost: b.PlannedCost, PlannedCostPlan: b.PlannedCostPlan})
	}
	return items, nil
}

// MoveToActivity moves line items to another activity: the split of an activity (016 FR-028).
func (x lineItemsForActivities) MoveToActivity(ctx context.Context, a actor.Actor, lineItemIDs []string, toActivityID string) error {
	return x.svc.MoveToActivity(ctx, a, lineItemIDs, toActivityID)
}
