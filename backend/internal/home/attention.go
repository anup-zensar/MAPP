package home

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// The reasons a record needs attention (HomeAttentionReason.kind, FR-011), in the order a record lists them.
const (
	reasonDeclined        = "declined"
	reasonNegativeBalance = "negativeBalance"
	reasonOverAllocated   = "overAllocated"
	reasonOverBudget      = "overBudget"
	reasonMissingFields   = "missingFields"
)

// reasonGroups order the items by their first reason (FR-012): declined, then balance, then budget, then
// missing fields.
var reasonGroups = map[string]int{
	reasonDeclined: 0, reasonNegativeBalance: 1, reasonOverAllocated: 2, reasonOverBudget: 2, reasonMissingFields: 3,
}

// Attention returns Needs your attention (FR-011, FR-012): every plan, campaign, initiative and activity a
// owns, in any plan, that has at least one reason — a Declined activity with its decline; line items above
// an activity's planned cost; a plan or campaign whose children's budgets exceed its budget; planned cost
// above a budget; fields still needed or records below that need attention. A record appears once with all
// its reasons. Items are ordered by their first reason's group, then the most recently changed first, then
// by ID; all of them are returned (D-173). Line items never appear on their own: their problems count in
// their activity (Assumptions).
func (s *Service) Attention(ctx context.Context, a actor.Actor) (HomeAttentionPage, error) {
	var page HomeAttentionPage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		all, err := s.src.Plans(ctx, a)
		if err != nil {
			return err
		}
		containers, err := s.src.Containers(ctx, a, a.ID)
		if err != nil {
			return err
		}
		owned, err := s.src.Activities(ctx, a, a.ID)
		if err != nil {
			return err
		}

		items := []HomeAttentionItem{}
		add := func(item HomeAttentionItem) {
			if len(item.Reasons) > 0 {
				items = append(items, item)
			}
		}
		for _, p := range all {
			if p.OwnerID == a.ID {
				add(planItem(p, c))
			}
		}
		for _, container := range containers {
			add(containerItem(container, c))
		}
		for _, act := range owned {
			add(activityItem(act, c))
		}
		slices.SortFunc(items, func(x, y HomeAttentionItem) int {
			return cmp.Or(cmp.Compare(reasonGroups[x.Reasons[0].Kind], reasonGroups[y.Reasons[0].Kind]),
				y.ChangedAt.Compare(x.ChangedAt), strings.Compare(x.ID, y.ID))
		})
		page = HomeAttentionPage{Items: items, Total: len(items)}
		return nil
	})
	if err != nil {
		return HomeAttentionPage{}, err
	}
	return page, nil
}

// planItem is the plan p with its reasons: over-allocated, over budget, missing fields.
func planItem(p Plan, c converter) HomeAttentionItem {
	ref := errs.RecordRef{Type: "plan", ID: p.ID, Name: p.Name}
	item := newItem(ref, ref, []errs.RecordRef{}, p.UpdatedAt)
	item.Reasons = budgetReasons(p.RemainingBudget, p.PlannedCost, p.Budget, c)
	item.Reasons = appendMissing(item.Reasons, p.Missing, p.ItemsBelow)
	return item
}

// containerItem is the campaign or initiative with its reasons: over-allocated (campaigns), over budget,
// missing fields.
func containerItem(ct Container, c converter) HomeAttentionItem {
	path := ct.Path
	if path == nil {
		path = []errs.RecordRef{}
	}
	item := newItem(errs.RecordRef{Type: ct.Type, ID: ct.ID, Name: ct.Name}, ct.Plan, path, ct.UpdatedAt)
	item.Reasons = budgetReasons(ct.RemainingBudget, ct.PlannedCost, ct.Budget, c)
	item.Reasons = appendMissing(item.Reasons, ct.Missing, ct.ItemsBelow)
	return item
}

// activityItem is the activity with its reasons: declined, line items above its planned cost, missing
// fields or line items needing attention.
func activityItem(act Activity, c converter) HomeAttentionItem {
	item := newItem(errs.RecordRef{Type: "activity", ID: act.ID, Name: act.Name}, act.Plan,
		[]errs.RecordRef{act.Plan, act.Campaign, act.Initiative}, act.UpdatedAt)
	if act.Status == activityDeclined {
		reason := newReason(reasonDeclined)
		if d := act.Decline; d != nil {
			by, at, comment := d.By, d.At, d.Comment
			reason.DeclinedBy, reason.DeclinedAt, reason.Comment = &by, &at, &comment
		}
		item.Reasons = append(item.Reasons, reason)
	}
	if act.Unassigned != nil && act.Unassigned.Sign() < 0 { // domain model §4.3, D-016
		item.Reasons = append(item.Reasons, amountReason(reasonNegativeBalance, act.Unassigned.Neg(), c))
	}
	item.Reasons = appendMissing(item.Reasons, act.Missing, act.LineItemsNeedingAttention)
	return item
}

// newItem returns the item of record, without reasons yet.
func newItem(record, plan errs.RecordRef, path []errs.RecordRef, changedAt time.Time) HomeAttentionItem {
	return HomeAttentionItem{
		RecordType: record.Type, RecordTypeLabel: record.Type, ID: record.ID, Name: record.Name, Plan: plan, Path: path,
		ChangedAt: changedAt, Reasons: []HomeAttentionReason{},
	}
}

// budgetReasons are the budget reasons of a plan, campaign or initiative (domain model §4.1, §4.2, D-015):
// over-allocated when its children's budgets exceed its budget (remaining below 0.00), and over budget when
// its planned cost is above its budget.
func budgetReasons(remaining, planned, budget *money.Amount, c converter) []HomeAttentionReason {
	reasons := []HomeAttentionReason{}
	if remaining != nil && remaining.Sign() < 0 {
		reasons = append(reasons, amountReason(reasonOverAllocated, remaining.Neg(), c))
	}
	if budget != nil && planned != nil && planned.Cmp(*budget) > 0 {
		reasons = append(reasons, amountReason(reasonOverBudget, planned.Sub(*budget), c))
	}
	return reasons
}

// appendMissing adds the missing-fields reason when the record has fields still needed or records below
// that need attention (domain model §2.6).
func appendMissing(reasons []HomeAttentionReason, missing []MissingField, itemsBelow int) []HomeAttentionReason {
	if len(missing) == 0 && itemsBelow == 0 {
		return reasons
	}
	reason := newReason(reasonMissingFields)
	reason.Missing = append(reason.Missing, missing...)
	reason.ItemsBelow = itemsBelow
	return append(reasons, reason)
}

// amountReason is a reason of kind with how much is over, converted with c.
func amountReason(kind string, usd money.Amount, c converter) HomeAttentionReason {
	reason := newReason(kind)
	amount := c.of(usd)
	reason.Amount = &amount
	return reason
}

// newReason is a reason of kind with nothing filled in.
func newReason(kind string) HomeAttentionReason {
	return HomeAttentionReason{Kind: kind, Missing: []MissingField{}}
}
