// Package activitytest is the in-memory fake of the activities store, for tests that build services or the
// whole API without a database.
package activitytest

import (
	"context"
	"slices"

	"mapp/internal/activity"
	"mapp/internal/approval/approvaltest"
	"mapp/internal/campaign/campaigntest"
	"mapp/internal/plan/plantest"
	"mapp/internal/platform/fake"
	"mapp/internal/settings"
)

type Store struct {
	activities *fake.Table[activity.Activity]
	plans      plantest.Store
	campaigns  campaigntest.Store
	approvals  approvaltest.Store
}

func NewStore(db *fake.DB) Store {
	return Store{
		activities: fake.TableOf(db, "activities", func(a activity.Activity) string { return a.ID }),
		plans:      plantest.NewStore(db),
		campaigns:  campaigntest.NewStore(db),
		approvals:  approvaltest.NewStore(db),
	}
}

func (s Store) Get(_ context.Context, id string) (activity.Activity, bool, error) {
	return s.activities.Get(id)
}

func (s Store) List(context.Context) ([]activity.Activity, error) {
	return s.activities.List()
}

func (s Store) ListByIDs(_ context.Context, ids []string) ([]activity.Activity, error) {
	return s.activities.Where(func(a activity.Activity) bool { return slices.Contains(ids, a.ID) })
}

func (s Store) ListByInitiative(ctx context.Context, initiativeID string) ([]activity.Activity, error) {
	return s.ListByInitiatives(ctx, []string{initiativeID})
}

func (s Store) ListByInitiatives(_ context.Context, initiativeIDs []string) ([]activity.Activity, error) {
	return s.activities.Where(func(a activity.Activity) bool { return slices.Contains(initiativeIDs, a.InitiativeID) })
}

func (s Store) ListSplitFrom(_ context.Context, activityID string) ([]activity.Activity, error) {
	return s.activities.Where(func(a activity.Activity) bool { return a.SplitFromID == activityID })
}

// Put stores no attributes as an empty object, as the PostgreSQL column reads back.
func (s Store) Put(ctx context.Context, a activity.Activity) error {
	return s.activities.Put(ctx, saved(a))
}

func (s Store) AddMany(ctx context.Context, list []activity.Activity) error {
	stored := make([]activity.Activity, len(list))
	for i, a := range list {
		stored[i] = saved(a)
	}
	return s.activities.Put(ctx, stored...)
}

func (s Store) Delete(ctx context.Context, id string) error {
	return s.activities.Delete(ctx, id)
}

func (s Store) DeleteMany(ctx context.Context, ids []string) error {
	return s.activities.Delete(ctx, ids...)
}

func (s Store) WorkflowIDs(context.Context) (map[string]string, error) {
	all, err := s.activities.List()
	fields := make(map[string]string, len(all))
	for _, a := range all {
		fields[a.ID] = a.ApprovalWorkflowID
	}
	return fields, err
}

func (s Store) CountOwnedBy(_ context.Context, personID string) (int, error) {
	owned, err := s.activities.Where(func(a activity.Activity) bool { return personID != "" && a.OwnerID == personID })
	return len(owned), err
}

func saved(a activity.Activity) activity.Activity {
	if a.Attributes == nil {
		a.Attributes = settings.AttributeValues{}
	}
	return a
}
