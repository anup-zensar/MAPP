package app

import (
	"context"
	"strings"

	"mapp/internal/allwork"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
)

// Locate returns the stored record recordID with its plan (and, for a line item, its activity), and false
// when it, or a record above it, is not stored.
func (s allWorkSource) Locate(ctx context.Context, recordID string) (allwork.Location, bool, error) {
	prefix, _, _ := strings.Cut(recordID, "-")
	switch prefix {
	case ids.PrefixPlan:
		return s.locatePlan(ctx, recordID)
	case ids.PrefixCampaign:
		return s.locateCampaign(ctx, recordID)
	case ids.PrefixInitiative:
		return s.locateInitiative(ctx, recordID)
	case ids.PrefixActivity:
		return s.locateActivity(ctx, recordID)
	case ids.PrefixLineItem:
		return s.locateLineItem(ctx, recordID)
	}
	return allwork.Location{}, false, nil
}

func (s allWorkSource) locatePlan(ctx context.Context, planID string) (allwork.Location, bool, error) {
	p, found, err := s.plansRepo.Get(ctx, planID)
	if err != nil || !found {
		return allwork.Location{}, false, err
	}
	return allwork.Location{Level: allwork.LevelPlan, Name: p.Name, PlanID: p.ID, Year: p.Year}, true, nil
}

func (s allWorkSource) locateCampaign(ctx context.Context, campaignID string) (allwork.Location, bool, error) {
	c, found, err := s.campaignsRepo.GetCampaign(ctx, campaignID)
	if err != nil || !found {
		return allwork.Location{}, false, err
	}
	return s.withPlan(ctx, allwork.Location{Level: allwork.LevelCampaign, Name: c.Name}, c.PlanID)
}

func (s allWorkSource) locateInitiative(ctx context.Context, initiativeID string) (allwork.Location, bool, error) {
	i, found, err := s.campaignsRepo.GetInitiative(ctx, initiativeID)
	if err != nil || !found {
		return allwork.Location{}, false, err
	}
	return s.underCampaign(ctx, allwork.Location{Level: allwork.LevelInitiative, Name: i.Name}, i.CampaignID)
}

func (s allWorkSource) locateActivity(ctx context.Context, activityID string) (allwork.Location, bool, error) {
	a, found, err := s.activitiesRepo.Get(ctx, activityID)
	if err != nil || !found {
		return allwork.Location{}, false, err
	}
	return s.underInitiative(ctx, allwork.Location{Level: allwork.LevelActivity, Name: a.Name}, a.InitiativeID)
}

func (s allWorkSource) locateLineItem(ctx context.Context, lineItemID string) (allwork.Location, bool, error) {
	li, found, err := s.lineItemsRepo.Get(ctx, lineItemID)
	if err != nil || !found {
		return allwork.Location{}, false, err
	}
	a, found, err := s.activitiesRepo.Get(ctx, li.ActivityID)
	if err != nil || !found {
		return allwork.Location{}, false, err
	}
	location := allwork.Location{Level: allwork.LevelLineItem, Name: li.Name, ActivityID: a.ID}
	return s.underInitiative(ctx, location, a.InitiativeID)
}

// underInitiative completes location with the plan of the initiative initiativeID.
func (s allWorkSource) underInitiative(ctx context.Context, location allwork.Location, initiativeID string) (allwork.Location, bool, error) {
	i, found, err := s.campaignsRepo.GetInitiative(ctx, initiativeID)
	if err != nil || !found {
		return allwork.Location{}, false, err
	}
	return s.underCampaign(ctx, location, i.CampaignID)
}

// underCampaign completes location with the plan of the campaign campaignID.
func (s allWorkSource) underCampaign(ctx context.Context, location allwork.Location, campaignID string) (allwork.Location, bool, error) {
	c, found, err := s.campaignsRepo.GetCampaign(ctx, campaignID)
	if err != nil || !found {
		return allwork.Location{}, false, err
	}
	return s.withPlan(ctx, location, c.PlanID)
}

// withPlan completes location with the plan planID, when it is stored.
func (s allWorkSource) withPlan(ctx context.Context, location allwork.Location, planID string) (allwork.Location, bool, error) {
	if _, found, err := s.plansRepo.Get(ctx, planID); err != nil || !found {
		return allwork.Location{}, false, err
	}
	location.PlanID = planID
	return location, true, nil
}

// InRecycleBin reports whether a recycle-bin entry not yet undone or restored holds the record recordID:
// deleted itself or with a record above it.
func (s allWorkSource) InRecycleBin(ctx context.Context, recordID string) (bool, error) {
	entries, err := s.history.ListBin(ctx, actor.System) // entries undone or restored are left out
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		for _, snapshot := range entry.Snapshots {
			if snapshot.ID == recordID {
				return true, nil
			}
		}
	}
	return false, nil
}
