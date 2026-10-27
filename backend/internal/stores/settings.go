package stores

import "mapp/internal/settings"

// SettingsStore is the Settings store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type SettingsStore interface {
	settings.Store
}
