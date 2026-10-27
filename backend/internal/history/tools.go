package history

import (
	"mapp/internal/platform/clock"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
)

// Tools are what every area's service changes records with, so its constructor stays within four parameters.
type Tools struct {
	Tx  store.Transactor
	IDs *ids.Generator
	Log *Service
	Now clock.Clock
}
