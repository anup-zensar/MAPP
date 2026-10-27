package stores

import "mapp/internal/sample"

// SampleStore is the samples store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type SampleStore interface {
	sample.Store
}
