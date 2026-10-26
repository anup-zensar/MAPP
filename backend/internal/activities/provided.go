package activities

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"strings"

	"mapp/internal/campaigns"
	"mapp/internal/plans"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// Summaries returns one roll-up summary per activity under the initiatives initiativeIDs (FR-005, D-105):
// what campaigns, initiatives and plans add up when they are read — planned cost, approval, dates, goals as
// shown and completeness. It is the Summaries of feature 005's campaigns.Activities and runs inside the
// caller's transaction. Roll-ups are never stored, so a create, edit or delete shows at once.
func (s *Service) Summaries(ctx context.Context, initiativeIDs []string) ([]campaigns.ActivitySummary, error) {
	var summaries []campaigns.ActivitySummary
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		list, err := s.repo.ListByInitiatives(ctx, initiativeIDs)
		if err != nil {
			return err
		}
		facts, err := s.lineItemFacts(ctx, list)
		if err != nil {
			return err
		}
		contexts := map[string]planContext{} // by initiative
		summaries = make([]campaigns.ActivitySummary, 0, len(list))
		for _, a := range list {
			pc, known := contexts[a.InitiativeID]
			if !known {
				if pc, known, err = s.contextOf(ctx, a.InitiativeID); err != nil {
					return err
				}
				if !known {
					continue // the initiative is gone: so is the activity, for every roll-up
				}
				contexts[a.InitiativeID] = pc
			}
			li := facts[a.ID]
			d, err := s.storedDerive(ctx, a, pc, li)
			if err != nil {
				return err
			}
			goals := goalsOf(a, pc.model)
			summaries = append(summaries, campaigns.ActivitySummary{
				ID: a.ID, InitiativeID: a.InitiativeID, Name: a.Name, TypeID: a.ActivityTypeID,
				PlannedCost: a.PlannedCost, Approved: a.Status == StatusApproved,
				StartDate: a.StartDate, EndDate: a.EndDate, Ongoing: a.Ongoing,
				INQ: goals.INQ, MQL: goals.MQL, Revenue: goals.Revenue,
				Incomplete: d.completeness.MissingFields > 0, LineItems: li.Count,
				IncompleteLineItems: incompleteLineItems(li),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return summaries, nil
}

// incompleteLineItems lists the line items with a problem once each, in the order of their problems.
func incompleteLineItems(li LineItemFacts) []errs.RecordRef {
	refs := []errs.RecordRef{}
	for _, p := range li.Problems {
		if !slices.ContainsFunc(refs, func(r errs.RecordRef) bool { return r.ID == p.LineItemID }) {
			refs = append(refs, errs.RecordRef{Type: "lineItem", ID: p.LineItemID, Name: p.LineItemName})
		}
	}
	return refs
}

// AddPlanFacts adds the activities' figures of every plan in planIDs to facts[planID] (FR-005, D-105): how
// many activities and how many of them approved (Approved, In progress or Complete), the sum of their
// planned costs (left nil when none has one), the sums of their goals as shown, and every activity with
// fields still needed, by name. It is the AddFacts of the activity level of feature 004's registry (D-072),
// which internal/app registers, and runs inside the caller's transaction. Plans not found, or without an
// entry in facts, are skipped.
func (s *Service) AddPlanFacts(ctx context.Context, planIDs []string, facts map[string]*plans.Facts) error {
	return s.tx.Read(ctx, func(ctx context.Context) error {
		var all []planActivities
		var list []Activity
		for _, planID := range planIDs {
			plan, found, err := s.plans.Get(ctx, planID)
			if err != nil {
				return err
			}
			if !found || facts[planID] == nil {
				continue
			}
			inPlan, err := s.activitiesOf(ctx, plan)
			if err != nil {
				return err
			}
			all, list = append(all, inPlan), append(list, inPlan.activities...)
		}
		lineItems, err := s.lineItemFacts(ctx, list) // once for every plan
		if err != nil {
			return err
		}
		for _, inPlan := range all {
			if err := s.addFacts(ctx, inPlan, lineItems, facts[inPlan.base.plan.ID]); err != nil {
				return err
			}
		}
		return nil
	})
}

// addFacts adds the figures of the activities of one plan to f.
func (s *Service) addFacts(ctx context.Context, inPlan planActivities, lineItems map[string]LineItemFacts, f *plans.Facts) error {
	var attention []plans.AttentionItem
	for _, a := range inPlan.activities {
		f.Activities++
		if a.Status == StatusApproved {
			f.ApprovedActivities++
		}
		if a.PlannedCost != nil {
			sum := a.PlannedCost.Amount
			if f.PlannedCost != nil {
				sum = f.PlannedCost.Add(sum)
			}
			f.PlannedCost = &sum
			f.AmountsSet++ // an amount in the plan currency, which locks it (016 FR-012)
		}
		goals := goalsOf(a, inPlan.base.model)
		if goals.INQ != nil {
			f.ActivityGoals.INQ += *goals.INQ
		}
		if goals.MQL != nil {
			f.ActivityGoals.MQL += *goals.MQL
		}
		if goals.Revenue != nil {
			f.ActivityGoals.Revenue = money.USD(f.ActivityGoals.Revenue.Amount.Add(goals.Revenue.Amount))
		}
		d, err := s.storedDerive(ctx, a, inPlan.contextOf(a), lineItems[a.ID])
		if err != nil {
			return err
		}
		if len(d.missing) > 0 {
			attention = append(attention, plans.AttentionItem{
				Record: errs.RecordRef{Type: kindName, ID: a.ID, Name: a.Name}, Missing: d.missing,
			})
		}
	}
	slices.SortStableFunc(attention, func(x, y plans.AttentionItem) int {
		return cmp.Or(strings.Compare(strings.ToLower(x.Record.Name), strings.ToLower(y.Record.Name)),
			strings.Compare(x.Record.ID, y.Record.ID))
	})
	f.NeedsAttention = append(f.NeedsAttention, attention...)
	return nil
}

// SubmitCheck returns the ready-to-submit check of the activity id, line items included (FR-023, D-204):
// what feature 008 checks before it accepts a Submit for approval. It joins the caller's transaction and
// answers not found for an activity that is not stored.
func (s *Service) SubmitCheck(ctx context.Context, a actor.Actor, id string) (SubmitCheck, error) {
	view, err := s.Get(ctx, a, id)
	if err != nil {
		return SubmitCheck{}, err
	}
	return view.Readiness, nil
}

// CopyCounts is how many activities a copy made, and how many line items came with them.
type CopyCounts struct {
	Activities, LineItems int
}

// CopyAll copies every activity of the initiative fromInitiativeID, with its line items, into the
// initiative toInitiativeID (R13, D-195): the same names and values, every date yearShift years later (29
// February becomes 28 February; an ongoing activity keeps no end date), new IDs, the status Created and
// the actor as creator — no approvals, comments, history or GSAP values. It joins the caller's Write and
// writes no log entries: the copy of the campaign or initiative above writes the one "copied" entry. It is
// the CopyAll of feature 005's campaigns.Activities.
func (s *Service) CopyAll(ctx context.Context, a actor.Actor, fromInitiativeID, toInitiativeID string, yearShift int) (CopyCounts, error) {
	return s.CopyMany(ctx, a, []CopyPair{{FromID: fromInitiativeID, ToID: toInitiativeID}}, yearShift)
}

// CopyMany is CopyAll for many initiatives at once, each level written in one statement.
func (s *Service) CopyMany(ctx context.Context, a actor.Actor, pairs []CopyPair, yearShift int) (CopyCounts, error) {
	var counts CopyCounts
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		fromIDs, newInitiativeIDs := make([]string, 0, len(pairs)), make(map[string]string, len(pairs))
		for _, p := range pairs {
			fromIDs = append(fromIDs, p.FromID)
			newInitiativeIDs[p.FromID] = p.ToID
		}
		list, err := s.repo.ListByInitiatives(ctx, fromIDs)
		if err != nil {
			return err
		}
		list = inInitiativeOrder(pairs, list)
		newIDs, err := s.addCopies(ctx, copyBy{actor: a, years: yearShift}, list, newInitiativeIDs)
		if err != nil {
			return err
		}
		counts.Activities = len(newIDs)
		counts.LineItems, err = s.lineItems.CopyMany(ctx, a, lineItemCopyOrder(pairs, list, newIDs), yearShift)
		return err
	})
	if err != nil {
		return CopyCounts{}, err
	}
	return counts, nil
}

func inInitiativeOrder(pairs []CopyPair, list []Activity) []Activity {
	byInitiative := map[string][]Activity{}
	for _, activity := range list {
		byInitiative[activity.InitiativeID] = append(byInitiative[activity.InitiativeID], activity)
	}
	ordered := make([]Activity, 0, len(list))
	for _, p := range pairs {
		ordered = append(ordered, byInitiative[p.FromID]...)
	}
	return ordered
}

// lineItemCopyOrder orders the copied activities as one CopyAll per initiative did — by the text of their ids
// within an initiative — so their line items keep getting the same IDs.
func lineItemCopyOrder(pairs []CopyPair, list []Activity, newIDs map[string]string) []CopyPair {
	byInitiative := map[string][]string{}
	for _, activity := range list {
		byInitiative[activity.InitiativeID] = append(byInitiative[activity.InitiativeID], activity.ID)
	}
	order := make([]CopyPair, 0, len(list))
	for _, p := range pairs {
		for _, id := range slices.Sorted(slices.Values(byInitiative[p.FromID])) {
			order = append(order, CopyPair{FromID: id, ToID: newIDs[id]})
		}
	}
	return order
}

// CopyForInitiatives copies the activities of every initiative of newInitiativeIDs (the id of a copied
// initiative → the id of its copy; other ids match no activity) into that copy, as CopyAll does but without
// their line items, and returns the old → new ID of every activity it copied. It is the Copy of the
// activity level of feature 004's plan copy (D-195): the line-item level copies the line items next.
func (s *Service) CopyForInitiatives(ctx context.Context, a actor.Actor, newInitiativeIDs map[string]string, yearShift int) (map[string]string, error) {
	var newIDs map[string]string
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		var err error
		list, err := s.repo.ListByInitiatives(ctx, slices.Sorted(maps.Keys(newInitiativeIDs)))
		if err != nil {
			return err
		}
		newIDs, err = s.addCopies(ctx, copyBy{actor: a, years: yearShift}, list, newInitiativeIDs)
		return err
	})
	if err != nil {
		return nil, err
	}
	return newIDs, nil
}

type copyBy struct {
	actor actor.Actor
	years int
}

// addCopies stores the copies of list with the next IDs in its order, in one write, and returns the new IDs.
func (s *Service) addCopies(ctx context.Context, by copyBy, list []Activity, newInitiativeIDs map[string]string) (map[string]string, error) {
	fresh, err := s.ids.NextIDs(ctx, ids.PrefixActivity, len(list))
	if err != nil {
		return nil, err
	}
	copies, newIDs := make([]Activity, 0, len(list)), make(map[string]string, len(list))
	for i, activity := range list {
		dup := copyOf(activity, by.years)
		dup.InitiativeID, dup.ID = newInitiativeIDs[activity.InitiativeID], fresh[i]
		dup.Status, dup.Meta = StatusCreated, record.NewMeta(by.actor, s.now())
		copies, newIDs[activity.ID] = append(copies, dup), dup.ID
	}
	if err := s.repo.AddMany(ctx, copies); err != nil {
		return nil, err
	}
	return newIDs, nil
}

// copyOf returns a copy of the activity a that shares no pointer or map with it, its dates moved years on.
func copyOf(a Activity, years int) Activity {
	dup := a
	dup.SplitFromID = "" // copies never carry the split link (016 FR-031)
	dup.StartDate, dup.EndDate = shiftDate(a.StartDate, years), shiftDate(a.EndDate, years)
	dup.PlannedCost, dup.Revenue = clonePtr(a.PlannedCost), clonePtr(a.Revenue)
	dup.INQ, dup.MQL = clonePtr(a.INQ), clonePtr(a.MQL)
	dup.Attributes = maps.Clone(a.Attributes)
	if dup.Attributes == nil {
		dup.Attributes = settings.AttributeValues{}
	}
	return dup
}

// shiftDate returns the date d years later (29 February → 28 February), or nil for no date.
func shiftDate(d *date.Date, years int) *date.Date {
	if d == nil {
		return nil
	}
	shifted := d.AddYears(years)
	return &shifted
}

// clonePtr returns a pointer to a copy of what p points at, or nil.
func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
