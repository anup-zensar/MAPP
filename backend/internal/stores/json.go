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

// JSON is the set of the tests' stand-in store; feature 025 replaces it with fakes and deletes it.
func JSON(db *store.DB) Set {
	return Set{
		Tx: db, Counters: ids.NewJSONCounters(), Samples: samples.NewJSONStore(),
		Organisation: jsonOrganisation(), Planning: jsonPlanning(), Finance: jsonFinance(),
		Ready:   func(context.Context) error { return db.Ready() },
		IsEmpty: func(context.Context) (bool, error) { return db.IsEmpty(), nil },
	}
}

func jsonOrganisation() Organisation {
	return Organisation{
		People: people.NewJSONStore(), History: history.NewJSONStore(), Auth: auth.NewJSONStore(),
		Access: access.NewJSONStore(), Settings: settings.NewJSONStore(),
	}
}

func jsonPlanning() Planning {
	return Planning{
		Plans: plans.NewJSONStore(), Campaigns: campaigns.NewJSONStore(),
		Activities: activities.NewJSONStore(), LineItems: lineitems.NewJSONStore(),
		Approvals: approvals.NewJSONStore(), AllWork: allwork.NewJSONStore(),
	}
}

func jsonFinance() Finance {
	return Finance{Budgets: budgets.NewJSONStore(), Financials: financials.NewJSONStore()}
}
