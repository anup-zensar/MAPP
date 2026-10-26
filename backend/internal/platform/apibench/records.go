package apibench

import (
	"context"
	"slices"
	"strings"
	"testing"

	"mapp/internal/activities"
	"mapp/internal/campaigns"
	"mapp/internal/platform/date"
	"mapp/internal/stores"
)

type Tree struct {
	Plan, Campaign, Initiative, Activity, LineItem string
	// an activity and an initiative that can move: each with another parent whose dates hold it
	ActivityMove, ActivityTarget     string
	InitiativeMove, InitiativeTarget string
}

// Records are the trees of a small and a large plan of the same database, so that a benchmark shows whether
// queries grow with the records: the demo plan of 7 activities and the plan of 500 (the template's 100 to copy).
type Records struct {
	Small, Large, Template Tree
}

func (s *Server) Records(tb testing.TB) Records {
	tb.Helper()
	v := &volume{tb: tb, ctx: tb.Context(), set: s.set}
	return Records{
		Small:    v.tree(demoLubricantsPlan),
		Large:    v.tree(v.planNamed(BigPlan)),
		Template: v.tree(v.planNamed(TemplatePlan)),
	}
}

func (v *volume) tree(planID string) Tree {
	t := Tree{Plan: planID}
	initiatives := v.initiativesOf(planID)
	list := v.activitiesOf(planID)
	t.Campaign, t.Initiative = initiatives[0].CampaignID, initiatives[0].ID
	for _, a := range list {
		items, err := v.set.LineItems.ListByActivity(v.ctx, a.ID)
		v.must(err)
		if len(items) > 0 && t.LineItem == "" {
			t.Activity, t.LineItem = a.ID, items[0].ID
		}
		if t.ActivityMove == "" {
			t.ActivityMove, t.ActivityTarget = a.ID, initiativeHolding(a, initiatives)
			if t.ActivityTarget == "" {
				t.ActivityMove = ""
			}
		}
	}
	t.InitiativeMove, t.InitiativeTarget = v.initiativeAndOtherCampaign(v.ctx, initiatives, v.campaignsOf(planID))
	return t
}

func initiativeHolding(a activities.Activity, initiatives []campaigns.Initiative) string {
	for _, i := range initiatives {
		if i.ID != a.InitiativeID && holds(i.StartDate, i.EndDate, a.StartDate, a.EndDate) {
			return i.ID
		}
	}
	return ""
}

func (v *volume) initiativeAndOtherCampaign(_ context.Context, initiatives []campaigns.Initiative, list []campaigns.Campaign) (string, string) {
	for _, i := range initiatives {
		for _, c := range list {
			if c.ID != i.CampaignID && holds(c.StartDate, c.EndDate, i.StartDate, i.EndDate) && !named(initiatives, c.ID, i.Name) {
				return i.ID, c.ID
			}
		}
	}
	return "", ""
}

func named(initiatives []campaigns.Initiative, campaignID, name string) bool {
	return slices.ContainsFunc(initiatives, func(i campaigns.Initiative) bool {
		return i.CampaignID == campaignID && strings.EqualFold(i.Name, name)
	})
}

// holds reports whether the dates from–to lie within start–end; a missing date limits nothing.
func holds(start, end, from, to *date.Date) bool {
	startOK := start == nil || from == nil || !from.Before(*start)
	endOK := end == nil || to == nil || !to.After(*end)
	return startOK && endOK
}

func (s *Server) Set() stores.Set {
	return s.set
}
