package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"mapp/internal/history"
	"mapp/internal/plan"
	"mapp/internal/platform/money"
)

// addCreated spreads the "created" entries evenly from 14 to 10 days before, in tree order, so parents come first.
func (h *demoHistory) addCreated(ctx context.Context) error {
	first, last := h.at.Add(-createdFrom), h.at.Add(-createdUntil)
	step := time.Duration(0)
	if n := len(h.records); n > 1 {
		step = last.Sub(first) / time.Duration(n-1)
	}
	for i, r := range h.records {
		at, by := wholeMinuteFrom(first.Add(time.Duration(i)*step)), h.ownerOrAlex(r.ownerID)
		h.add(at, by, history.ActionCreated, string(history.ActionCreated), r.kind, r.id, r.name, r.path, nil, "", "")
		if err := r.created(ctx, at, by); err != nil {
			return err
		}
	}
	return nil
}

// wholeMinuteFrom keeps the first entry within the 14 days whatever the seed time's seconds are.
func wholeMinuteFrom(t time.Time) time.Time {
	if minute := t.Truncate(time.Minute); minute.Before(t) {
		return minute.Add(time.Minute)
	}
	return t
}

// addPublished keeps one "published" entry for every Published plan, by its owner, 7 days before (step 2).
func (h *demoHistory) addPublished(context.Context) error {
	for _, r := range h.records {
		if r.plan == nil || r.plan.Status != plan.StatusPublished {
			continue
		}
		change, err := statusChange(plan.StatusDraft, plan.StatusPublished, planStatusWords())
		if err != nil {
			return err
		}
		h.add(h.at.Add(-publishedAgo), h.ownerOrAlex(r.ownerID), history.ActionStatusChanged, "published",
			r.kind, r.id, r.name, r.path, []history.Change{change}, "", "")
	}
	return nil
}

func statusChange[S ~string](from, to S, words map[S]string) (history.Change, error) {
	oldValue, err := json.Marshal(from)
	if err != nil {
		return history.Change{}, err
	}
	newValue, err := json.Marshal(to)
	if err != nil {
		return history.Change{}, err
	}
	return history.Change{Field: "status", Label: "Status", Old: oldValue, New: newValue, OldText: words[from], NewText: words[to]}, nil
}

// addCostChange: Priya's first FY2026 Lubricants Europe activity went up by 5,000.00 USD, 5 days before (step 4).
func (h *demoHistory) addCostChange(context.Context) error {
	var r resolver
	priyaID := r.id(h.demo.people, "person", "Priya Raman")
	planID := r.id(h.plans, "plan", lubricantsPlan)
	if r.err != nil {
		return r.err
	}
	i := slices.IndexFunc(h.records, func(rec treeRecord) bool {
		return rec.activity != nil && rec.ownerID == priyaID && rec.activity.PlannedCost != nil && rec.path[0].ID == planID
	})
	if i < 0 {
		return fmt.Errorf("seed: Priya Raman has no activity with a planned cost in %s", lubricantsPlan)
	}
	activity := h.records[i]
	now := *activity.activity.PlannedCost
	by, err := money.ParseAmount(costChangedBy)
	if err != nil {
		return err
	}
	before := money.Money{Amount: now.Amount.Sub(by), Currency: now.Currency}
	oldValue, err := json.Marshal(before)
	if err != nil {
		return err
	}
	newValue, err := json.Marshal(now)
	if err != nil {
		return err
	}
	change := history.Change{Field: "plannedCost", Label: "Planned cost", Old: oldValue, New: newValue,
		OldText: before.Format(), NewText: now.Format()}
	h.add(h.at.Add(-costChangedAgo), h.people[priyaID], history.ActionChanged, string(history.ActionChanged),
		activity.kind, activity.id, activity.name, activity.path, []history.Change{change}, "", "")
	return nil
}
