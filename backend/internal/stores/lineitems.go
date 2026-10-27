package stores

import "mapp/internal/lineitems"

// LineItemStore is the line items store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type LineItemStore interface {
	lineitems.Store
}
