package lineitems

import (
	"context"

	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// Provided to other features (plan, "Provided to other features"): what activities (006), plans (004),
// campaigns (005) and approvals (008) read about line items, through adapters in internal/app.

// Summary is one activity's line items in short (data model §3.9).
type Summary struct {
	Count          int
	PlannedCostUSD money.Amount // sum of the line items' USD planned costs at today's rates
	AutoSpread     int          // spread Evenly, First or Last month: follows the activity's dates
	CustomSpread   int          // custom spread: kept as typed
	Incomplete     []Incomplete // the line items whose missing list is not empty, in creation order
}

// Incomplete is a line item that still stops its activity from being submitted, with what it misses (data
// model §3.6).
type Incomplete struct {
	LineItemID, LineItemName string
	Missing                  []settings.MissingField
}

// Summaries returns the summary of each activity in activityIDs that has line items; an activity without
// line items, or one that is not stored, has no entry. It runs inside the caller's transaction (it joins
// it), so a change shows at once: nothing here is stored.
func (s *Service) Summaries(ctx context.Context, activityIDs []string) (map[string]Summary, error) {
	summaries := map[string]Summary{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		items, err := s.repo.ListByActivities(ctx, activityIDs)
		if err != nil {
			return err
		}
		byActivity := map[string][]LineItem{}
		for _, li := range items {
			byActivity[li.ActivityID] = append(byActivity[li.ActivityID], li)
		}
		for activityID, list := range byActivity {
			f, found, err := s.deps.Activities.Facts(ctx, activityID)
			if err != nil {
				return err
			}
			if !found {
				continue
			}
			views, err := s.views(ctx, f, list)
			if err != nil {
				return err
			}
			summary := Summary{Count: len(views), Incomplete: []Incomplete{}}
			for _, v := range views {
				summary.PlannedCostUSD = summary.PlannedCostUSD.Add(v.PlannedCostUSD.Amount)
				if v.Spread.Method == SpreadCustom {
					summary.CustomSpread++
				} else {
					summary.AutoSpread++
				}
				if len(v.Missing) > 0 {
					summary.Incomplete = append(summary.Incomplete, Incomplete{LineItemID: v.ID, LineItemName: v.Name, Missing: v.Missing})
				}
			}
			summaries[activityID] = summary
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return summaries, nil
}
