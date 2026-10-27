package lineitem

import (
	"cmp"
	"context"
	"slices"

	"mapp/internal/platform/fake"
	"mapp/internal/settings"
)

// fakeStore is lineitemtest.Store for this package's own tests, which cannot import lineitemtest.
type fakeStore struct {
	lineItems *fake.Table[LineItem]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{lineItems: fake.TableOf(db, "line_items", func(li LineItem) string { return li.ID })}
}

func (s fakeStore) Get(_ context.Context, id string) (LineItem, bool, error) {
	return s.lineItems.Get(id)
}

func (s fakeStore) List(context.Context) ([]LineItem, error) {
	return s.lineItems.List()
}

func (s fakeStore) ListByActivity(ctx context.Context, activityID string) ([]LineItem, error) {
	return s.ListByActivities(ctx, []string{activityID})
}

func (s fakeStore) ListByActivities(_ context.Context, activityIDs []string) ([]LineItem, error) {
	return s.lineItems.Where(func(li LineItem) bool { return slices.Contains(activityIDs, li.ActivityID) })
}

func (s fakeStore) ListByIDs(_ context.Context, ids []string) ([]LineItem, error) {
	return s.lineItems.Where(func(li LineItem) bool { return slices.Contains(ids, li.ID) })
}

func (s fakeStore) Put(ctx context.Context, li LineItem) error {
	return s.lineItems.Put(ctx, normalisedLineItem(li))
}

func (s fakeStore) AddMany(ctx context.Context, list []LineItem) error {
	stored := make([]LineItem, 0, len(list))
	for _, li := range list {
		stored = append(stored, normalisedLineItem(li))
	}
	return s.lineItems.Put(ctx, stored...)
}

func (s fakeStore) Delete(ctx context.Context, id string) error {
	return s.lineItems.Delete(ctx, id)
}

func (s fakeStore) DeleteMany(ctx context.Context, ids []string) error {
	return s.lineItems.Delete(ctx, ids...)
}

func normalisedLineItem(li LineItem) LineItem {
	if li.Attributes == nil {
		li.Attributes = settings.AttributeValues{}
	}
	months := append([]MonthAmount{}, li.Spread.Months...)
	slices.SortStableFunc(months, func(a, b MonthAmount) int { return cmp.Compare(a.Month, b.Month) })
	li.Spread.Months = months
	return li
}
