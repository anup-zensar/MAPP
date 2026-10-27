package budget_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"mapp/internal/budget"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

var (
	pgMeta  = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))
	pgExact = cmpopts.EquateComparable(money.Amount{})
)

// pgStore gives a store whose database holds the plans P-1001 (team T-1001) and P-1002 (team T-1002), the
// owner U-1001 and the global budgets GB-1001 and GB-1002, with two allocations each to one plan.
func pgStore(t *testing.T) (*database.DB, *budget.PostgresStore, []budget.Allocation) {
	t.Helper()
	db := dbtest.NewDB(t)
	s := budget.NewPostgresStore(db)
	directory, catalogue, planStore := people.NewPostgresStore(db), settings.NewPostgresStore(db), plan.NewPostgresStore(db)
	allocations := []budget.Allocation{pgAllocation("BA-1001", "GB-1001", "P-1001"), pgAllocation("BA-1002", "GB-1001", "P-1002")}
	pgWrite(t, db, func(ctx context.Context) error {
		err := errors.Join(
			catalogue.PutCurrency(ctx, settings.Currency{Code: "USD", Name: "US dollar", Active: true, Meta: pgMeta}),
			catalogue.PutActivityType(ctx, settings.ActivityType{ID: "AT-1001", Name: "Webinar", Meta: pgMeta}),
			catalogue.PutModel(ctx, settings.Model{ID: "MOD-1001", Name: "Leads", ActivityTypeIDs: []string{"AT-1001"},
				InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(1000),
				AverageDealSize: money.USD(money.FromCents(100)), Meta: pgMeta}),
			directory.Put(ctx, people.Person{ID: "U-1001", Name: "Priya Raman", Email: "priya@example.com", Meta: pgMeta}),
			s.PutBudget(ctx, pgBudget("GB-1001", "Brand 2026")), s.PutBudget(ctx, pgBudget("GB-1002", "Retail 2026")),
			s.PutAllocation(ctx, allocations[0]), s.PutAllocation(ctx, allocations[1]))
		for _, n := range []string{"1", "2"} {
			err = errors.Join(err,
				directory.PutTeam(ctx, people.Team{ID: "T-100" + n, Name: "Team " + n, Colour: "blue", Meta: pgMeta}),
				planStore.Put(ctx, plan.Plan{ID: "P-100" + n, Name: "Plan " + n, Year: 2026, ModelID: "MOD-1001",
					TeamIDs: []string{"T-100" + n}, DefaultSpread: plan.SpreadEvenly, GoalStage: plan.StageINQ,
					Status: plan.StatusDraft, Meta: pgMeta}))
		}
		return err
	})
	return db, s, allocations
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgBudget(id, name string) budget.GlobalBudget {
	return budget.GlobalBudget{ID: id, Name: name, Year: 2026, Amount: money.FromCents(100000000), OwnerID: "U-1001",
		Description: "Global " + name, Meta: pgMeta}
}

func pgAllocation(id, budgetID, planID string) budget.Allocation {
	return budget.Allocation{ID: id, GlobalBudgetID: budgetID, PlanID: planID,
		Amount: money.USD(money.FromCents(2500000)), Meta: pgMeta}
}

func TestPostgresStoreAllocationScopeIsolationFR009(t *testing.T) {
	t.Parallel()
	_, s, allocations := pgStore(t)
	tests := []struct {
		name  string
		scope authz.Scope
		want  []budget.Allocation
	}{
		{name: "View plans in team T-1002", scope: authz.Scope{PersonID: "U-1001", Holds: true, TeamIDs: []string{"T-1002"}},
			want: allocations[1:]},
		{name: "no role", scope: authz.Scope{PersonID: "U-1001", TeamIDs: []string{}}, want: []budget.Allocation{}},
		{name: "everything", scope: authz.Scope{Everything: true, TeamIDs: []string{}}, want: allocations},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := s.AllocationsInScope(t.Context(), tt.scope, "GB-1001")
			if err != nil {
				t.Fatalf("AllocationsInScope() error: %v", err)
			}
			if diff := cmp.Diff(tt.want, got, pgExact); diff != "" {
				t.Errorf("AllocationsInScope() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
