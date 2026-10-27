package stores

import "mapp/internal/history"

// HistoryStore is the activity log store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type HistoryStore interface {
	history.Store
}
