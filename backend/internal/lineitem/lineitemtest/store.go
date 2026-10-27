// Package lineitemtest is the in-memory fake of the line items store, for tests that build services or the
// whole API without a database.
package lineitemtest

import (
	"cmp"
	"context"
	"slices"

	"mapp/internal/lineitem"
	"mapp/internal/platform/fake"
	"mapp/internal/settings"
)

type Store struct {
	lineItems *fake.Table[lineitem.LineItem]
}

func NewStore(db *fake.DB) Store {
	return Store{lineItems: fake.TableOf(db, "line_items", func(li lineitem.LineItem) string { return li.ID })}
}

func (s Store) Get(_ context.Context, id string) (lineitem.LineItem, bool, error) {
	return s.lineItems.Get(id)
}

func (s Store) List(context.Context) ([]lineitem.LineItem, error) {
	return s.lineItems.List()
}

func (s Store) ListByActivity(ctx context.Context, activityID string) ([]lineitem.LineItem, error) {
	return s.ListByActivities(ctx, []string{activityID})
}

func (s Store) ListByActivities(_ context.Context, activityIDs []string) ([]lineitem.LineItem, error) {
	return s.lineItems.Where(func(li lineitem.LineItem) bool { return slices.Contains(activityIDs, li.ActivityID) })
}

func (s Store) ListByIDs(_ context.Context, ids []string) ([]lineitem.LineItem, error) {
	return s.lineItems.Where(func(li lineitem.LineItem) bool { return slices.Contains(ids, li.ID) })
}

func (s Store) Put(ctx context.Context, li lineitem.LineItem) error {
	return s.lineItems.Put(ctx, normalised(li))
}

func (s Store) AddMany(ctx context.Context, list []lineitem.LineItem) error {
	stored := make([]lineitem.LineItem, 0, len(list))
	for _, li := range list {
		stored = append(stored, normalised(li))
	}
	return s.lineItems.Put(ctx, stored...)
}

func (s Store) Delete(ctx context.Context, id string) error {
	return s.lineItems.Delete(ctx, id)
}

func (s Store) DeleteMany(ctx context.Context, ids []string) error {
	return s.lineItems.Delete(ctx, ids...)
}

func (s Store) CountOwnedBy(_ context.Context, personID string) (int, error) {
	owned, err := s.lineItems.Where(func(li lineitem.LineItem) bool { return li.OwnerID != "" && li.OwnerID == personID })
	return len(owned), err
}

// normalised is what the PostgreSQL store reads back: the attributes an object, the spread months by month.
func normalised(li lineitem.LineItem) lineitem.LineItem {
	if li.Attributes == nil {
		li.Attributes = settings.AttributeValues{}
	}
	months := append([]lineitem.MonthAmount{}, li.Spread.Months...)
	slices.SortStableFunc(months, func(a, b lineitem.MonthAmount) int { return cmp.Compare(a.Month, b.Month) })
	li.Spread.Months = months
	return li
}
