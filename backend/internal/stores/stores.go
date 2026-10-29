// Package stores gathers the store of every business area and the transaction they share: on PostgreSQL for
// the API and the seed (Postgres), in memory for tests (storestest).
package stores

import (
	"context"

	"mapp/internal/platform/ids"
)

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

type Set struct {
	Tx       Transactor
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
	Overspend  OverspendStore
}

type Finance struct {
	Budgets    BudgetStore
	Financials FinancialsStore
	SwbsRuns   SwbsStore

	PurchaseRequests PurchaseRequestStore
}
