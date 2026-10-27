package activity

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/actor"
)

// SplitOptions returns what the Split dialog of the activity id offers (FR-026): its line items with their
// amounts in the plan currency, the suggested name "<name> (split)" free in its initiative, whether the split
// restarts its approval, and the initiatives of the plan — its own first — with Move's date rule as the
// reason an initiative can't take the new activity.
func (s *Service) SplitOptions(ctx context.Context, a actor.Actor, id string) (SplitOptions, error) {
	destinations, destinationsErr := s.Destinations(ctx, a, id) // its own Read
	if destinationsErr != nil {
		return SplitOptions{}, destinationsErr
	}
	var options SplitOptions
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		activity, pc, err := s.load(ctx, id)
		if err != nil {
			return err
		}
		items, err := s.lineItems.Items(ctx, id)
		if err != nil {
			return err
		}
		siblings, err := s.repo.ListByInitiative(ctx, activity.InitiativeID)
		if err != nil {
			return err
		}
		options = SplitOptions{
			ActivityID: activity.ID, Version: activity.Version, Status: activity.Status, PlannedCost: activity.PlannedCost,
			PlanCurrency: pc.plan.Currency, SuggestedName: suffixedName(siblings, activity.Name, splitSuffix),
			NeedsRestart: inApproval(activity), LineItems: make([]SplitLineItem, 0, len(items)),
			Destinations: splitDestinations(destinations),
		}
		for _, li := range items {
			options.LineItems = append(options.LineItems, SplitLineItem(li))
		}
		return nil
	})
	if err != nil {
		return SplitOptions{}, err
	}
	return options, nil
}

// splitDestinations turns Move's destinations into the Split dialog's: the activity's own initiative first
// and available; elsewhere only the date rule applies, because the new activity has a name of its own.
func splitDestinations(all []Destination) []Destination {
	list := make([]Destination, 0, len(all))
	for _, d := range all {
		d.MoveProblem = d.CopyProblem // the dates; nil for its own initiative
		if d.Current {
			list = append([]Destination{d}, list...)
			continue
		}
		list = append(list, d)
	}
	return list
}

// suffixedName returns the first of "<name><suffix>", "<name><suffix without its parenthesis> 2)", … that no
// activity among siblings has, shortened to fit 120 characters.
func suffixedName(siblings []Activity, name, suffix string) string {
	for n := 1; ; n++ {
		end := suffix
		if n > 1 {
			end = fmt.Sprintf("%s %d)", strings.TrimSuffix(suffix, ")"), n)
		}
		base := name
		if room := maxNameLength - utf8.RuneCountInString(end); utf8.RuneCountInString(base) > room {
			base = strings.TrimRight(string([]rune(base)[:room]), " ")
		}
		if _, taken := namedLike(siblings, base+end, ""); !taken {
			return base + end
		}
	}
}
