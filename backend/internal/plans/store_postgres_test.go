package plans_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"mapp/internal/people"
	"mapp/internal/plans"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

var (
	pgAt    = time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC)
	pgExact = cmpopts.EquateComparable(money.Amount{}, money.Percent{})
)

// pgStore gives a store whose database holds what plans refer to: US dollars, the model MOD-1001, the
// people U-1001 to U-1003 and the teams T-1001 and T-1002.
func pgStore(t *testing.T) (*database.DB, *plans.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	directory, catalogue := people.NewPostgresStore(db), settings.NewPostgresStore(db)
	meta := record.NewMeta(actor.System, pgAt)
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(
			catalogue.PutCurrency(ctx, settings.Currency{Code: "USD", Name: "US dollar", Active: true, Meta: meta}),
			catalogue.PutActivityType(ctx, settings.ActivityType{ID: "AT-1001", Name: "Webinar", Meta: meta}),
			catalogue.PutModel(ctx, settings.Model{ID: "MOD-1001", Name: "Lead generation", ActivityTypeIDs: []string{"AT-1001"},
				InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(1000),
				AverageDealSize: money.USD(money.FromCents(100)), Meta: meta}),
			directory.Put(ctx, people.Person{ID: "U-1001", Name: "Priya Raman", Email: "priya@example.com", Meta: meta}),
			directory.Put(ctx, people.Person{ID: "U-1002", Name: "Daniel Okoro", Email: "daniel@example.com", Meta: meta}),
			directory.Put(ctx, people.Person{ID: "U-1003", Name: "Mei Chen", Email: "mei@example.com", Meta: meta}),
			directory.PutTeam(ctx, people.Team{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue", Meta: meta}),
			directory.PutTeam(ctx, people.Team{ID: "T-1002", Name: "Mobility China", Colour: "teal", Meta: meta}))
	})
	return db, plans.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgPlan(id, createdBy, ownerID string, teamIDs ...string) plans.Plan {
	return plans.Plan{ID: id, Name: "Plan " + id, Year: 2026, ModelID: "MOD-1001", TeamIDs: append([]string{}, teamIDs...),
		OwnerID: ownerID, DefaultSpread: plans.SpreadEvenly, GoalStage: plans.StageINQ,
		Attributes: settings.AttributeValues{}, Status: plans.StatusDraft, Currency: "USD", PlanType: plans.TypeOperational,
		BudgetPhasing: rollup.NotPhased, Meta: record.NewMeta(actor.Actor{ID: createdBy, Name: createdBy}, pgAt)}
}

func TestPostgresStorePlansReadBackAsSavedFR008(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	budget := money.FromCents(12000000)
	europe := pgPlan("P-1001", "U-1001", "U-1002", "T-1002", "T-1001")
	europe.Budget, europe.Attributes = &budget, settings.AttributeValues{"SA-1001": json.RawMessage(`"SO-1001"`)}
	europe.BudgetPhasing = rollup.Phasing{Method: rollup.MethodCustom, Months: []rollup.PhasingMonth{
		{Month: "2026-01", Amount: money.FromCents(4000000)}, {Month: "2026-02", Amount: money.FromCents(8000000)}}}
	nextYear := pgPlan("P-1002", "U-1001", "", "T-1001")
	nextYear.Year = 2027
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.Put(ctx, nextYear), s.Put(ctx, europe), s.Put(ctx, pgPlan("P-1003", "U-1001", "")))
	})
	pgWrite(t, db, func(ctx context.Context) error { return s.Delete(ctx, "P-1003") })

	all, allErr := s.List(t.Context())
	of2027, yearErr := s.PlansOfYear(t.Context(), 2027)

	if err := errors.Join(allErr, yearErr); err != nil {
		t.Fatalf("List(), PlansOfYear() error: %v", err)
	}
	if diff := cmp.Diff([]plans.Plan{europe, nextYear}, all, pgExact); diff != "" {
		t.Errorf("List() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]plans.Plan{nextYear}, of2027, pgExact); diff != "" {
		t.Errorf("PlansOfYear() mismatch (-want +got):\n%s", diff)
	}
	europe.TeamIDs, europe.BudgetPhasing = []string{"T-1001"}, rollup.NotPhased
	pgWrite(t, db, func(ctx context.Context) error { return s.Put(ctx, europe) })
	if got, found, err := s.Get(t.Context(), "P-1001"); err != nil || !found || !cmp.Equal(europe, got, pgExact) {
		t.Errorf("Get() after replacing teams and phasing = %+v, found %v, error %v; want %+v", got, found, err, europe)
	}
}

// TestPostgresStorePlanScopeIsCanViewFR009 proves that the scoped reads keep exactly the plans authz.CanView
// lets each person see: by team, as owner, as creator of a plan without teams, or everything.
func TestPostgresStorePlanScopeIsCanViewFR009(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	stored := []plans.Plan{pgPlan("P-1001", "U-1003", "U-1003", "T-1001"), pgPlan("P-1002", "U-1001", "U-1002", "T-1002"),
		pgPlan("P-1003", "U-1001", ""), pgPlan("P-1004", "U-1003", "")}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.Put(ctx, stored[0]), s.Put(ctx, stored[1]), s.Put(ctx, stored[2]), s.Put(ctx, stored[3]))
	})
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{name: "an internal call", ctx: t.Context()},
		{name: "View plans in All markets", ctx: viewing(t, "U-1001", authz.ScopeAll)},
		{name: "View plans in one team, creator of a plan without teams", ctx: viewing(t, "U-1001", authz.ScopeTeams, "T-1001")},
		{name: "no role, owner of one plan", ctx: viewing(t, "U-1002", "")},
		{name: "View plans in one team, owner and creator elsewhere", ctx: viewing(t, "U-1003", authz.ScopeTeams, "T-1002")},
		{name: "no role, creator of a plan without teams", ctx: viewing(t, "U-1003", "")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			visible, err := s.ListInScope(tt.ctx, authz.ScopeOf(tt.ctx))
			if err != nil {
				t.Fatalf("ListInScope() error: %v", err)
			}
			if diff := cmp.Diff(seenByCanView(tt.ctx, stored), planIDs(visible)); diff != "" {
				t.Errorf("ListInScope() mismatch with authz.CanView (-want +got):\n%s", diff)
			}
			for _, p := range stored {
				_, found, err := s.GetInScope(tt.ctx, authz.ScopeOf(tt.ctx), p.ID)
				if err != nil || found != slices.Contains(planIDs(visible), p.ID) {
					t.Errorf("GetInScope(%s) = found %v, error %v; want found only when listed", p.ID, found, err)
				}
			}
		})
	}
}

// viewing returns a request context of personID holding View plans in scope (none when scope is empty).
func viewing(t *testing.T, personID string, scope authz.ScopeType, teamIDs ...string) context.Context {
	t.Helper()
	var roles []authz.GrantedRole
	if scope != "" {
		roles = []authz.GrantedRole{{ID: "RL-1001", Name: "Viewer", Scope: scope, TeamIDs: teamIDs}}
	}
	grants := authz.NewGrants(personID, roles, nil, map[string][]authz.Permission{"RL-1001": {authz.PlansView}}, nil, authz.Rules{})
	return authz.With(t.Context(), grants)
}

func seenByCanView(ctx context.Context, stored []plans.Plan) []string {
	ids := []string{}
	for _, p := range stored {
		if authz.CanView(ctx, authz.PlanFacts{ID: p.ID, TeamIDs: p.TeamIDs, OwnerID: p.OwnerID, CreatedByID: p.CreatedBy.ID}) {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

func planIDs(list []plans.Plan) []string {
	ids := []string{}
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	return ids
}

func TestPostgresStoreRefusesBrokenPlansFR004(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		plan plans.Plan
	}{
		{name: "a name that differs only in letter case", plan: func() plans.Plan {
			p := pgPlan("P-1002", "U-1001", "", "T-1001")
			p.Name = "PLAN P-1001"
			return p
		}()},
		{name: "a team that does not exist", plan: pgPlan("P-1002", "U-1001", "", "T-9999")},
		{name: "an owner who does not exist", plan: pgPlan("P-1002", "U-1001", "U-9999")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s := pgStore(t)
			pgWrite(t, db, func(ctx context.Context) error { return s.Put(ctx, pgPlan("P-1001", "U-1001", "", "T-1001")) })

			err := db.Write(t.Context(), func(ctx context.Context) error { return s.Put(ctx, tt.plan) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}
