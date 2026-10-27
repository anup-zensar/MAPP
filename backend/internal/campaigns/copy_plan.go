package campaigns

import (
	"context"

	"mapp/internal/platform/actor"
)

// CopyCounts is how many campaigns and initiatives a plan copy made.
type CopyCounts struct {
	Campaigns, Initiatives int
}

// CopyPlanContents copies every campaign of the plan fromPlanID, with its initiatives, into the plan
// toPlanID for feature 004's plan copy (004 FR-027, FR-028; R13): the same names and values, every date
// shiftYears years later (29 February becomes 28 February), new IDs and the actor as creator. It joins the
// caller's Write and writes no log entries: the plan copy writes the one "copied" entry. It returns the
// counts and the old → new ID of every campaign and initiative, for 004's CopyJob.NewIDs; the activities and
// line items below are copied by their own levels of the plan copy, not here (D-195).
func (s *Service) CopyPlanContents(ctx context.Context, a actor.Actor, fromPlanID, toPlanID string, shiftYears int) (CopyCounts, map[string]string, error) { //nolint:revive // argument-limit 5: the call of the plan-copy level in internal/app.
	var counts CopyCounts
	newIDs := map[string]string{}
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		campaigns, err := s.repo.CampaignsOfPlan(ctx, fromPlanID)
		if err != nil {
			return err
		}
		copies := make([]Campaign, 0, len(campaigns))
		for _, c := range campaigns { // by id, so the copies get their IDs in the same order
			copies = append(copies, s.campaignCopy(a, c, copyPlace{parentID: toPlanID, name: c.Name, years: shiftYears}))
		}
		if err = s.addCampaignCopies(ctx, copies); err != nil {
			return err
		}
		pairs := make([]CopyPair, 0, len(campaigns))
		for i, c := range campaigns {
			newIDs[c.ID] = copies[i].ID
			pairs = append(pairs, CopyPair{FromID: c.ID, ToID: copies[i].ID})
		}
		initiatives, err := s.copyInitiativesOf(ctx, a, pairs, shiftYears)
		for _, p := range initiatives {
			newIDs[p.FromID] = p.ToID
		}
		counts = CopyCounts{Campaigns: len(campaigns), Initiatives: len(initiatives)}
		return err
	})
	if err != nil {
		return CopyCounts{}, nil, err
	}
	return counts, newIDs, nil
}
