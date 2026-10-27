package plan_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/people"
	"mapp/internal/plan"
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
	pgAt = time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC)
)

// pgStore gives a store whose database holds what plans refer to: US dollars, the model MOD-1001, the
// people U-1001 to U-1003 and the teams T-1001 and T-1002.
func pgStore(t *testing.T) (*database.DB, *plan.PostgresStore) {
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
	return db, plan.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgPlan(id, createdBy, ownerID string, teamIDs ...string) plan.Plan {
	return plan.Plan{ID: id, Name: "Plan " + id, Year: 2026, ModelID: "MOD-1001", TeamIDs: append([]string{}, teamIDs...),
		OwnerID: ownerID, DefaultSpread: plan.SpreadEvenly, GoalStage: plan.StageINQ,
		Attributes: settings.AttributeValues{}, Status: plan.StatusDraft, Currency: "USD", PlanType: plan.TypeOperational,
		BudgetPhasing: rollup.NotPhased, Meta: record.NewMeta(actor.Actor{ID: createdBy, Name: createdBy}, pgAt)}
}

// TestPostgresStorePlanScopeIsCanViewFR009 proves that the scoped reads keep exactly the plans authz.CanView
// lets each person see: by team, as owner, as creator of a plan without teams, or everything.
func TestPostgresStorePlanScopeIsCanViewFR009(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	stored := []plan.Plan{pgPlan("P-1001", "U-1003", "U-1003", "T-1001"), pgPlan("P-1002", "U-1001", "U-1002", "T-1002"),
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

func seenByCanView(ctx context.Context, stored []plan.Plan) []string {
	ids := []string{}
	for _, p := range stored {
		if authz.CanView(ctx, authz.PlanFacts{ID: p.ID, TeamIDs: p.TeamIDs, OwnerID: p.OwnerID, CreatedByID: p.CreatedBy.ID}) {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

func planIDs(list []plan.Plan) []string {
	ids := []string{}
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	return ids
}
