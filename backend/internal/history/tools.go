package history

import (
	"mapp/internal/platform/clock"
	"mapp/internal/platform/ids"
)

// Tools are what every area's service changes records with, so its constructor stays within four parameters.
type Tools struct {
	Tx  Transactor
	IDs *ids.Generator
	Log *Service
	Now clock.Clock
}
