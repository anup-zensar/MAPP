package stores

import (
	"context"

	"mapp/internal/access"
	"mapp/internal/activity"
	"mapp/internal/allwork"
	"mapp/internal/approval"
	"mapp/internal/auth"
	"mapp/internal/budget"
	"mapp/internal/campaign"
	"mapp/internal/financials"
	"mapp/internal/history"
	"mapp/internal/lineitem"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
	"mapp/internal/sample"
	"mapp/internal/settings"
)

// JSON is the set of the tests' stand-in store; feature 025 replaces it with fakes and deletes it.
func JSON(db *store.DB) Set {
	return Set{
		Tx: db, Counters: ids.NewJSONCounters(), Samples: sample.NewJSONStore(),
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
		Plans: plan.NewJSONStore(), Campaigns: campaign.NewJSONStore(),
		Activities: activity.NewJSONStore(), LineItems: lineitem.NewJSONStore(),
		Approvals: approval.NewJSONStore(), AllWork: allwork.NewJSONStore(),
	}
}

func jsonFinance() Finance {
	return Finance{Budgets: budget.NewJSONStore(), Financials: financials.NewJSONStore()}
}
