package campaign

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
)

// The history functions of campaigns and initiatives for many records at once (history.Kind).

func (s *Service) allCampaignIDs(ctx context.Context, planIDs []string) ([]string, error) {
	return idsOf(s.repo.CampaignsOfPlans(ctx, planIDs))
}

func (s *Service) allInitiativeIDs(ctx context.Context, campaignIDs []string) ([]string, error) {
	return idsOf(s.repo.InitiativesOfCampaigns(ctx, campaignIDs))
}

func (s *Service) campaignSnapshots(ctx context.Context, ids []string) ([]history.Snapshot, error) {
	list, err := s.repo.CampaignsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return snapshotsOf(list, func(c Campaign) history.Snapshot {
		return history.Snapshot{ID: c.ID, Name: c.Name, ParentID: c.PlanID}
	})
}

func (s *Service) initiativeSnapshots(ctx context.Context, ids []string) ([]history.Snapshot, error) {
	list, err := s.repo.InitiativesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return snapshotsOf(list, func(i Initiative) history.Snapshot {
		return history.Snapshot{ID: i.ID, Name: i.Name, ParentID: i.CampaignID}
	})
}

func (s *Service) putCampaigns(ctx context.Context, snapshots []history.Snapshot) error {
	list, err := recordsOf[Campaign](snapshots)
	if err != nil {
		return err
	}
	return s.repo.AddCampaigns(ctx, list)
}

func (s *Service) putInitiatives(ctx context.Context, snapshots []history.Snapshot) error {
	list, err := recordsOf[Initiative](snapshots)
	if err != nil {
		return err
	}
	return s.repo.AddInitiatives(ctx, list)
}

func idsOf[T interface{ id() string }](list []T, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(list))
	for _, record := range list {
		ids = append(ids, record.id())
	}
	return ids, nil
}

// snapshotsOf turns records into snapshots: head gives the ID, name and parent, the record itself the Data.
func snapshotsOf[T any](list []T, head func(T) history.Snapshot) ([]history.Snapshot, error) {
	snapshots := make([]history.Snapshot, 0, len(list))
	for _, record := range list {
		data, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}
		snapshot := head(record)
		snapshot.Data = data
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func recordsOf[T any](snapshots []history.Snapshot) ([]T, error) {
	list := make([]T, 0, len(snapshots))
	for _, snapshot := range snapshots {
		var record T
		if err := json.Unmarshal(snapshot.Data, &record); err != nil {
			return nil, err
		}
		list = append(list, record)
	}
	return list, nil
}

func (c Campaign) id() string   { return c.ID }
func (i Initiative) id() string { return i.ID }
