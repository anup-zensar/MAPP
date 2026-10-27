package home

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
)

// MyActivities returns My activities (FR-014): the activities a owns, in every plan, grouped by the status
// their page shows — an ongoing activity is never Complete (domain model §2.3) — in the order Declined,
// Created, Submitted, Approved, In progress, Complete, empty groups left out. Within a group they are
// ordered by start date (none last), then by name ignoring letter case, then by ID; every row is returned
// (D-173), with its planned cost in a's display currency.
func (s *Service) MyActivities(ctx context.Context, a actor.Actor) (MyActivitiesView, error) {
	var view MyActivitiesView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		owned, err := s.src.Activities(ctx, a, a.ID)
		if err != nil {
			return err
		}
		slices.SortFunc(owned, func(x, y Activity) int {
			return cmp.Or(compareStart(x.StartDate, y.StartDate), compareNames(x.Name, y.Name), strings.Compare(x.ID, y.ID))
		})
		byStatus := map[string][]MyActivityRow{}
		for _, act := range owned {
			missing := act.Missing
			if missing == nil {
				missing = []MissingField{}
			}
			byStatus[act.DisplayStatus] = append(byStatus[act.DisplayStatus], MyActivityRow{
				ID: act.ID, Name: act.Name, Plan: act.Plan, Campaign: act.Campaign, Initiative: act.Initiative,
				Status: act.DisplayStatus, StartDate: act.StartDate, EndDate: act.EndDate, Ongoing: act.Ongoing,
				PlannedCost: c.optional(act.PlannedCost), Missing: missing, ItemsBelow: act.LineItemsNeedingAttention,
			})
		}
		view = MyActivitiesView{Groups: []MyActivityGroup{}, Total: len(owned)}
		for _, status := range groupOrder {
			if rows := byStatus[status]; len(rows) > 0 {
				view.Groups = append(view.Groups, MyActivityGroup{Status: status, Count: len(rows), Items: rows})
			}
		}
		return nil
	})
	if err != nil {
		return MyActivitiesView{}, err
	}
	return view, nil
}

// compareStart orders start dates, earliest first and no date last.
func compareStart(x, y *date.Date) int {
	switch {
	case x == nil && y == nil:
		return 0
	case x == nil:
		return 1
	case y == nil:
		return -1
	}
	return x.Compare(*y)
}
