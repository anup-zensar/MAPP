package stores

import (
	"context"

	"mapp/internal/performance"

	"mapp/internal/access"
	"mapp/internal/activity"
	"mapp/internal/allwork"
	"mapp/internal/approval"
	"mapp/internal/auth"
	"mapp/internal/budget"
	"mapp/internal/bulk"
	"mapp/internal/campaign"
	"mapp/internal/financials"
	"mapp/internal/history"
	"mapp/internal/lineitem"
	"mapp/internal/overspend"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/database"
	"mapp/internal/platform/ids"
	"mapp/internal/purchaserequest"
	"mapp/internal/sample"
	"mapp/internal/settings"
	"mapp/internal/swbs"
	"mapp/internal/workbook"
)

// Postgres is the set the API and the seed run on. Nothing stored yet means no number given out yet: every
// record takes its readable ID from the counters.
func Postgres(db *database.DB) Set {
	counters := ids.NewPostgresCounters(db)
	return Set{
		Tx: db, Counters: counters, Samples: sample.NewPostgresStore(db),
		Organisation: postgresOrganisation(db), Planning: postgresPlanning(db), Finance: postgresFinance(db),
		Ready: db.Ping,
		IsEmpty: func(ctx context.Context) (bool, error) {
			used, err := counters.Used(ctx)
			return !used, err
		},
	}
}

func postgresOrganisation(db *database.DB) Organisation {
	return Organisation{
		People: people.NewPostgresStore(db), History: history.NewPostgresStore(db), Auth: auth.NewPostgresStore(db),
		Access: access.NewPostgresStore(db), Settings: settings.NewPostgresStore(db),
	}
}

func postgresPlanning(db *database.DB) Planning {
	return Planning{
		Plans: plan.NewPostgresStore(db), Campaigns: campaign.NewPostgresStore(db),
		Activities: activity.NewPostgresStore(db), LineItems: lineitem.NewPostgresStore(db),
		Overspend: overspend.NewPostgresStore(db),
		Approvals: approval.NewPostgresStore(db), AllWork: allwork.NewPostgresStore(db), Bulk: bulk.NewPostgresStore(db),
		Workbook: workbook.NewPostgresStore(db),
	}
}

func postgresFinance(db *database.DB) Finance {
	return Finance{
		Budgets: budget.NewPostgresStore(db), Financials: financials.NewPostgresStore(db), SwbsRuns: swbs.NewPostgresStore(db),
		Performance:      performance.NewPostgresStore(db),
		PurchaseRequests: purchaserequest.NewPostgresStore(db),
	}
}
