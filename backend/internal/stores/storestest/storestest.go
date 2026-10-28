// Package storestest builds the API's store set in memory from every area's fake, for tests that run the
// whole API or the seed without a database.
package storestest

import (
	"context"

	"mapp/internal/access/accesstest"
	"mapp/internal/activity/activitytest"
	"mapp/internal/allwork/allworktest"
	"mapp/internal/approval/approvaltest"
	"mapp/internal/auth/authtest"
	"mapp/internal/budget/budgettest"
	"mapp/internal/bulk/bulktest"
	"mapp/internal/campaign/campaigntest"
	"mapp/internal/financials/financialstest"
	"mapp/internal/history/historytest"
	"mapp/internal/lineitem/lineitemtest"
	"mapp/internal/people/peopletest"
	"mapp/internal/plan/plantest"
	"mapp/internal/platform/fake"
	"mapp/internal/purchaserequest/purchaserequesttest"
	"mapp/internal/sample/sampletest"
	"mapp/internal/settings/settingstest"
	"mapp/internal/stores"
	"mapp/internal/swbs/swbstest"
)

func New(db *fake.DB) stores.Set {
	counters := fake.NewCounters(db)
	return stores.Set{
		Tx: db, Counters: counters, Samples: sampletest.NewStore(db),
		Organisation: organisation(db), Planning: planning(db), Finance: finance(db),
		Ready:   func(context.Context) error { return nil },
		IsEmpty: func(context.Context) (bool, error) { return !counters.Used(), nil },
	}
}

func organisation(db *fake.DB) stores.Organisation {
	return stores.Organisation{
		People: peopletest.NewStore(db), History: historytest.NewStore(db), Auth: authtest.NewStore(db),
		Access: accesstest.NewStore(db), Settings: settingstest.NewStore(db),
	}
}

func planning(db *fake.DB) stores.Planning {
	return stores.Planning{
		Plans: plantest.NewStore(db), Campaigns: campaigntest.NewStore(db), Activities: activitytest.NewStore(db),
		LineItems: lineitemtest.NewStore(db), Approvals: approvaltest.NewStore(db), AllWork: allworktest.NewStore(db),
		Bulk: bulktest.NewStore(db),
	}
}

func finance(db *fake.DB) stores.Finance {
	return stores.Finance{
		Budgets: budgettest.NewStore(db), Financials: financialstest.NewStore(db), SwbsRuns: swbstest.NewStore(db),
		PurchaseRequests: purchaserequesttest.NewStore(db),
	}
}
