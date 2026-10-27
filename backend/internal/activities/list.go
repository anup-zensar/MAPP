package activities

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// ListForPlan returns every activity of every initiative of the plan planID as plan-tree rows, ordered by
// start date, then name ignoring letter case, activities without a start date last (FR-033, D-111).
func (s *Service) ListForPlan(ctx context.Context, a actor.Actor, planID string) ([]Summary, error) {
	var summaries []Summary
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		plan, found, err := s.plans.Get(ctx, planID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(planGoneMessage)
		}
		all, err := s.activitiesOf(ctx, plan)
		if err != nil {
			return err
		}
		facts, err := s.lineItemFacts(ctx, all.activities)
		if err != nil {
			return err
		}
		summaries = make([]Summary, 0, len(all.activities))
		for _, activity := range all.activities {
			d, err := s.storedDerive(ctx, activity, all.contextOf(activity), facts[activity.ID])
			if err != nil {
				return err
			}
			summaries = append(summaries, displaySummary(money.DisplayUSD, Summary{
				ID: activity.ID, Name: activity.Name, InitiativeID: activity.InitiativeID, ActivityType: d.activityType,
				Owner: d.owner, StartDate: activity.StartDate, EndDate: activity.EndDate, Ongoing: activity.Ongoing,
				PlannedCost: activity.PlannedCost, UnassignedPlannedCost: d.unassigned, Status: activity.Status,
				DisplayStatus: d.displayStatus, Completeness: d.completeness, Version: activity.Version,
				SplitFromID: splitFromID(activity),
			}))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(summaries, func(x, y Summary) int { return comparePlanOrder(x.StartDate, x.Name, y.StartDate, y.Name) })
	return s.shownSummaries(ctx, a, summaries)
}

// SortInPlanOrder sorts activities as a plan's list shows them: by start date, none last, then by name; it is
// stable, so activities read in id order keep it otherwise.
func SortInPlanOrder(list []Activity) {
	slices.SortStableFunc(list, func(x, y Activity) int { return comparePlanOrder(x.StartDate, x.Name, y.StartDate, y.Name) })
}

func comparePlanOrder(xStart *date.Date, xName string, yStart *date.Date, yName string) int {
	if c := compareStart(xStart, yStart); c != 0 {
		return c
	}
	return strings.Compare(strings.ToLower(xName), strings.ToLower(yName))
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
