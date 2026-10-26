package activities

import (
	"context"
	"maps"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// CopyCounts is how many activities a copy made, and how many line items came with them.
type CopyCounts struct {
	Activities, LineItems int
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
