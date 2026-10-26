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
	"mapp/internal/platform/database"
	"mapp/internal/platform/ids"
	"mapp/internal/samples"
	"mapp/internal/settings"
)

// Postgres is the set the API and the seed run on. Nothing stored yet means no number given out yet: every
// record takes its readable ID from the counters.
func Postgres(db *database.DB) Set {
	counters := ids.NewPostgresCounters(db)
	return Set{
		Tx: db, Counters: counters, Samples: samples.NewPostgresStore(db),
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
		Plans: plans.NewPostgresStore(db), Campaigns: campaigns.NewPostgresStore(db),
		Activities: activities.NewPostgresStore(db), LineItems: lineitems.NewPostgresStore(db),
		Approvals: approvals.NewPostgresStore(db), AllWork: allwork.NewPostgresStore(db),
	}
}

func postgresFinance(db *database.DB) Finance {
	return Finance{Budgets: budgets.NewPostgresStore(db), Financials: financials.NewPostgresStore(db)}
}
