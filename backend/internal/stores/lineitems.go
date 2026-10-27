package stores

import (
	"context"

	"mapp/internal/lineitems"
)

// LineItemStore is the line items store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type LineItemStore interface {
	lineitems.Store
	// List returns every line item (usage counts of feature 002).
	List(ctx context.Context) ([]lineitems.LineItem, error)
	// CountOwnedBy counts the stored line items the person owns.
	CountOwnedBy(ctx context.Context, personID string) (int, error)
}
