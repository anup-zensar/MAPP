// Package stores gathers the store of every business area and the Transactor they share, on PostgreSQL for
// the API and the seed (Postgres), or on the JSON file store, the tests' stand-in until feature 025 (JSON).
package stores

import (
	"context"

	"mapp/internal/access"
	"mapp/internal/activities"
	"mapp/internal/allwork"
	"mapp/internal/approvals"
	"mapp/internal/auth"
	"mapp/internal/budgets"
	"mapp/internal/campaigns"
	"mapp/internal/financials"
	"mapp/internal/history"
	"mapp/internal/lineitems"
	"mapp/internal/people"
	"mapp/internal/plans"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
	"mapp/internal/samples"
	"mapp/internal/settings"
)

type Set struct {
	Tx       store.Transactor
	Counters ids.Counters
	Organisation
	Planning
	Finance
	Samples samples.Repository
	// Ready says why the data cannot be served now; IsEmpty whether nothing has been stored yet.
	Ready   func(ctx context.Context) error
	IsEmpty func(ctx context.Context) (bool, error)
}

type Organisation struct {
	People   people.Repository
	History  history.Repository
	Auth     auth.Repository
	Access   access.Repository
	Settings settings.Repository
}

type Planning struct {
	Plans      plans.Repository
	Campaigns  campaigns.Repository
	Activities activities.Repository
	LineItems  lineitems.Repository
	Approvals  approvals.Repository
	AllWork    allwork.Repository
}

type Finance struct {
	Budgets    budgets.Repository
	Financials financials.Repository
}
