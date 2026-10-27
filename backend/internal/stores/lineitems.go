package stores

import (
	"context"

	"mapp/internal/lineitem"
)

// LineItemStore is the line items store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type LineItemStore interface {
	lineitem.Store
	// List returns every line item (usage counts of feature 002).
	List(ctx context.Context) ([]lineitem.LineItem, error)
	// CountOwnedBy counts the stored line items the person owns.
	CountOwnedBy(ctx context.Context, personID string) (int, error)
}
