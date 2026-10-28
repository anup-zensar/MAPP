package purchaserequesttest

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"strings"

	"mapp/internal/purchaserequest"
)

func (s Store) AddRouting(ctx context.Context, r purchaserequest.Routing) error {
	return s.routings.Put(ctx, r)
}

// Routings lists the purchase request's routings newest first, as the SQL does.
func (s Store) Routings(_ context.Context, purchaseRequestID string) ([]purchaserequest.Routing, error) {
	list, err := s.routings.Where(func(r purchaserequest.Routing) bool { return r.PurchaseRequestID == purchaseRequestID })
	slices.SortFunc(list, func(a, b purchaserequest.Routing) int {
		return cmp.Or(b.RoutedAt.Compare(a.RoutedAt), cmp.Compare(len(b.ID), len(a.ID)), strings.Compare(b.ID, a.ID))
	})
	return list, err
}

func (s Store) Configuration(context.Context) (purchaserequest.Configuration, error) {
	row, found, err := s.configuration.Get(configurationID)
	if err != nil || !found {
		return purchaserequest.DefaultConfiguration(), err
	}
	return row.Configuration, nil
}

func (s Store) PutConfiguration(ctx context.Context, c purchaserequest.Configuration) error {
	return s.configuration.Put(ctx, configurationRow{ID: configurationID, Configuration: c})
}

func jsonStrings(raw json.RawMessage, into *[]string) error {
	return json.Unmarshal(raw, into)
}
