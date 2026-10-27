// Package stores gathers the store of every business area and the Transactor they share, on PostgreSQL for
// the API and the seed (Postgres), or on the JSON file store, the tests' stand-in until feature 025 (JSON).
package stores

import (
	"context"

	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
)

type Set struct {
	Tx       store.Transactor
	Counters ids.Counters
	Organisation
	Planning
	Finance
	Samples SampleStore
	// Ready says why the data cannot be served now; IsEmpty whether nothing has been stored yet.
	Ready   func(ctx context.Context) error
	IsEmpty func(ctx context.Context) (bool, error)
}

type Organisation struct {
	People   PeopleStore
	History  HistoryStore
	Auth     AuthStore
	Access   AccessStore
	Settings SettingsStore
}

type Planning struct {
	Plans      PlanStore
	Campaigns  CampaignStore
	Activities ActivityStore
	LineItems  LineItemStore
	Approvals  ApprovalStore
	AllWork    AllWorkStore
}

type Finance struct {
	Budgets    BudgetStore
	Financials FinancialsStore
}
