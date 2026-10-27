package activity_test

import (
	"context"
	"fmt"
	"slices"
	"time"

	"mapp/internal/activity"
	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// fakeLineItems are feature 007's line items as the tests set them.
type fakeLineItems struct {
	facts        map[string]activity.LineItemFacts
	datesChanged []string                            // the activities whose dates changed, in order
	copies       []string                            // the CopyAll calls, "<from> → <to> +<years>", in order
	items        map[string][]activity.LineItemBrief // 016: the line items of each activity, for Split
	moves        []string                            // 016: the MoveToActivity calls, "<ids> → <to>"
}

func (l *fakeLineItems) Facts(_ context.Context, activityIDs []string) (map[string]activity.LineItemFacts, error) {
	facts := map[string]activity.LineItemFacts{}
	for _, id := range activityIDs {
		if f, found := l.facts[id]; found {
			facts[id] = f
		}
	}
	return facts, nil
}

func (l *fakeLineItems) CopyMany(ctx context.Context, by actor.Actor, pairs []activity.CopyPair, years int) (int, error) {
	copied := 0
	for _, p := range pairs {
		n, _ := l.CopyAll(ctx, by, p.FromID, p.ToID, years)
		copied += n
	}
	return copied, nil
}

// CopyAll records the call and answers the activity's line-item count of facts.
func (l *fakeLineItems) CopyAll(_ context.Context, _ actor.Actor, from, to string, years int) (int, error) { //nolint:revive // argument-limit 5: the signature of activity.LineItems
	l.copies = append(l.copies, fmt.Sprintf("%s → %s +%d", from, to, years))
	return l.facts[from].Count, nil
}

func (l *fakeLineItems) DatesChanged(_ context.Context, _ actor.Actor, _, after activity.Activity) error {
	l.datesChanged = append(l.datesChanged, after.ID)
	return nil
}

// Items answers the line items the test set for the activity (016 Split).
func (l *fakeLineItems) Items(_ context.Context, activityID string) ([]activity.LineItemBrief, error) {
	return append([]activity.LineItemBrief{}, l.items[activityID]...), nil
}

// MoveToActivity records the move, "<ids> → <activity>", and moves the line items between the activities'
// facts and items as the real line items would.
func (l *fakeLineItems) MoveToActivity(_ context.Context, _ actor.Actor, ids []string, to string) error {
	l.moves = append(l.moves, fmt.Sprintf("%v → %s", ids, to))
	if l.items == nil {
		return nil
	}
	for from, items := range l.items {
		kept := []activity.LineItemBrief{}
		for _, li := range items {
			if slices.Contains(ids, li.ID) && from != to {
				l.items[to] = append(l.items[to], li)
				l.shift(from, to, li.PlannedCostPlan)
				continue
			}
			kept = append(kept, li)
		}
		l.items[from] = kept
	}
	return nil
}

// shift moves one line item's count and plan-currency amount from one activity's facts to another's.
func (l *fakeLineItems) shift(from, to string, amount money.Money) {
	if l.facts == nil {
		l.facts = map[string]activity.LineItemFacts{}
	}
	f := l.facts[from]
	f.Count--
	f.PlannedCostPlan = f.PlannedCostPlan.Sub(amount.Amount)
	l.facts[from] = f
	t := l.facts[to]
	t.Count++
	t.PlannedCostPlan = t.PlannedCostPlan.Add(amount.Amount)
	t.Currency = amount.Currency
	l.facts[to] = t
}

// fakeHooks record what feature 008 is told.
type fakeHooks struct {
	restarted []string // "<id>: <before status> → <after status>"
	changes   []string // the changes of every restart, "<label>: <old text> → <new text>"
	deleting  []string
	restored  []string
}

func (h *fakeHooks) ApprovalRestarted(_ context.Context, _ actor.Actor, before, after activity.Activity, changes []history.Change) error { //nolint:revive // argument-limit 5: the signature of activity.ApprovalHooks
	h.restarted = append(h.restarted, after.ID+": "+string(before.Status)+" → "+string(after.Status))
	for _, c := range changes {
		h.changes = append(h.changes, c.Label+": "+c.OldText+" → "+c.NewText)
	}
	return nil
}

func (h *fakeHooks) ActivityDeleting(_ context.Context, _ actor.Actor, a activity.Activity) error {
	h.deleting = append(h.deleting, a.ID)
	return nil
}

func (h *fakeHooks) ActivityRestored(_ context.Context, _ history.RestoreContext, id string) ([]string, error) {
	h.restored = append(h.restored, id)
	return nil, nil
}

// testClock is a clock that shows a fixed time until Advance moves it.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }
