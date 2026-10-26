package campaigns_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"mapp/internal/campaigns"
	"mapp/internal/people"
	"mapp/internal/plans"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

var (
	pgMeta  = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))
	pgExact = cmpopts.EquateComparable(money.Amount{}, money.Percent{}, date.Date{})
)

// pgStore gives a store whose database holds the plans P-1001 (team T-1001) and P-1002 (team T-1002, owned
// by U-1002), with what they refer to.
func pgStore(t *testing.T) (*database.DB, *campaigns.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	directory, catalogue, planStore := people.NewPostgresStore(db), settings.NewPostgresStore(db), plans.NewPostgresStore(db)
	plan := func(id, teamID, ownerID string) plans.Plan {
		return plans.Plan{ID: id, Name: "Plan " + id, Year: 2026, ModelID: "MOD-1001", TeamIDs: []string{teamID},
			OwnerID: ownerID, DefaultSpread: plans.SpreadEvenly, GoalStage: plans.StageINQ, Status: plans.StatusDraft,
			Currency: "USD", Meta: pgMeta}
	}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(
			catalogue.PutCurrency(ctx, settings.Currency{Code: "USD", Name: "US dollar", Active: true, Meta: pgMeta}),
			catalogue.PutActivityType(ctx, settings.ActivityType{ID: "AT-1001", Name: "Webinar", Meta: pgMeta}),
			catalogue.PutModel(ctx, settings.Model{ID: "MOD-1001", Name: "Lead generation", ActivityTypeIDs: []string{"AT-1001"},
				InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(1000),
				AverageDealSize: money.USD(money.FromCents(100)), Meta: pgMeta}),
			directory.Put(ctx, people.Person{ID: "U-1001", Name: "Priya Raman", Email: "priya@example.com", Meta: pgMeta}),
			directory.Put(ctx, people.Person{ID: "U-1002", Name: "Daniel Okoro", Email: "daniel@example.com", Meta: pgMeta}),
			directory.PutTeam(ctx, people.Team{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue", Meta: pgMeta}),
			directory.PutTeam(ctx, people.Team{ID: "T-1002", Name: "Mobility China", Colour: "teal", Meta: pgMeta}),
			planStore.Put(ctx, plan("P-1001", "T-1001", "")), planStore.Put(ctx, plan("P-1002", "T-1002", "U-1002")))
	})
	return db, campaigns.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgCampaign(id, planID, name string) campaigns.Campaign {
	return campaigns.Campaign{ID: id, PlanID: planID, Name: name, Attributes: settings.AttributeValues{},
		BudgetPhasing: rollup.NotPhased, Meta: pgMeta}
}

func pgInitiative(id, campaignID string) campaigns.Initiative {
	return campaigns.Initiative{ID: id, CampaignID: campaignID, Name: "Initiative " + id,
		Attributes: settings.AttributeValues{}, BudgetPhasing: rollup.NotPhased, Meta: pgMeta}
}

// pgFixture saves C-1001 and C-1003 in P-1001 and C-1002 in P-1002, each with one initiative.
func pgFixture(t *testing.T) (*database.DB, *campaigns.PostgresStore, []campaigns.Campaign, []campaigns.Initiative) {
	t.Helper()
	db, s := pgStore(t)
	owner, start, end := "U-1001", date.MustParse("2026-02-01"), date.MustParse("2026-06-30")
	budget := money.USD(money.FromCents(5000000))
	first := pgCampaign("C-1001", "P-1001", "Brand")
	first.OwnerID, first.StartDate, first.EndDate, first.Budget = &owner, &start, &end, &budget
	first.Attributes = settings.AttributeValues{"SA-1002": json.RawMessage(`["SO-1003","SO-1004"]`)}
	second := pgCampaign("C-1002", "P-1002", "Brand")
	second.Budget = &budget
	second.BudgetPhasing = rollup.Phasing{Method: rollup.MethodCustom, Months: []rollup.PhasingMonth{
		{Month: "2026-03", Amount: money.FromCents(5000000)}}}
	stored := []campaigns.Campaign{first, second, pgCampaign("C-1003", "P-1001", "Retail")}
	initiative := pgInitiative("I-1001", "C-1001")
	initiative.StartDate, initiative.EndDate, initiative.Budget = &start, &start, &budget
	initiatives := []campaigns.Initiative{initiative, pgInitiative("I-1002", "C-1002")}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutCampaign(ctx, stored[0]), s.PutCampaign(ctx, stored[1]), s.PutCampaign(ctx, stored[2]),
			s.PutInitiative(ctx, initiatives[0]), s.PutInitiative(ctx, initiatives[1]))
	})
	return db, s, stored, initiatives
}

func TestPostgresStoreCampaignsAndInitiativesFR008(t *testing.T) {
	t.Parallel()
	_, s, stored, initiatives := pgFixture(t)

	all, allErr := s.ListCampaigns(t.Context())
	ofFirstPlan, planErr := s.CampaignsOfPlans(t.Context(), []string{"P-1001"})
	ofSecond, initiativesErr := s.InitiativesOfCampaigns(t.Context(), []string{"C-1002"})
	one, found, getErr := s.GetInitiative(t.Context(), "I-1001")

	if err := errors.Join(allErr, planErr, initiativesErr, getErr); err != nil || !found {
		t.Fatalf("reads = found %v, error %v", found, err)
	}
	if diff := cmp.Diff(stored, all, pgExact); diff != "" {
		t.Errorf("ListCampaigns() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]campaigns.Campaign{stored[0], stored[2]}, ofFirstPlan, pgExact); diff != "" {
		t.Errorf("CampaignsOfPlans() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(initiatives[1:], ofSecond, pgExact); diff != "" {
		t.Errorf("InitiativesOfCampaigns() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(initiatives[0], one, pgExact); diff != "" {
		t.Errorf("GetInitiative() mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreCampaignScopeIsolationFR009(t *testing.T) {
	t.Parallel()
	_, s, _, _ := pgFixture(t)
	tests := []struct {
		name            string
		scope           authz.Scope
		wantCampaigns   []string
		wantInitiatives []string
	}{
		{name: "View plans in team T-1001", scope: authz.Scope{PersonID: "U-1003", ViewsPlans: true, TeamIDs: []string{"T-1001"}},
			wantCampaigns: []string{"C-1001", "C-1003"}, wantInitiatives: []string{"I-1001"}},
		{name: "no role, owner of P-1002", scope: authz.Scope{PersonID: "U-1002", TeamIDs: []string{}},
			wantCampaigns: []string{"C-1002"}, wantInitiatives: []string{"I-1002"}},
		{name: "no role, owner of nothing", scope: authz.Scope{PersonID: "U-1001", TeamIDs: []string{}},
			wantCampaigns: []string{}, wantInitiatives: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			visible, campaignsErr := s.CampaignsInScope(t.Context(), tt.scope)
			visibleInitiatives, initiativesErr := s.InitiativesInScope(t.Context(), tt.scope)
			_, foundOutside, getErr := s.GetCampaignInScope(t.Context(), tt.scope, "C-9999")
			if err := errors.Join(campaignsErr, initiativesErr, getErr); err != nil || foundOutside {
				t.Fatalf("scoped reads error = %v, found a missing campaign %v", err, foundOutside)
			}
			if diff := cmp.Diff(tt.wantCampaigns, campaignIDs(visible)); diff != "" {
				t.Errorf("CampaignsInScope() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantInitiatives, initiativeIDs(visibleInitiatives)); diff != "" {
				t.Errorf("InitiativesInScope() mismatch (-want +got):\n%s", diff)
			}
			for _, id := range []string{"C-1001", "C-1002"} {
				if _, found, err := s.GetCampaignInScope(t.Context(), tt.scope, id); err != nil || found != contains(tt.wantCampaigns, id) {
					t.Errorf("GetCampaignInScope(%s) = found %v, error %v", id, found, err)
				}
			}
			if _, found, err := s.GetInitiativeInScope(t.Context(), tt.scope, "I-1002"); err != nil || found != contains(tt.wantInitiatives, "I-1002") {
				t.Errorf("GetInitiativeInScope(I-1002) = found %v, error %v", found, err)
			}
		})
	}
}

func TestPostgresStoreRefusesBrokenCampaignsFR004(t *testing.T) {
	t.Parallel()
	start, end := date.MustParse("2026-06-30"), date.MustParse("2026-02-01")
	backwards := pgCampaign("C-1004", "P-1001", "Backwards")
	backwards.StartDate, backwards.EndDate = &start, &end
	tests := []struct {
		name  string
		write func(ctx context.Context, s *campaigns.PostgresStore) error
	}{
		{name: "a campaign name taken in the plan, in another letter case", write: func(ctx context.Context, s *campaigns.PostgresStore) error {
			return s.PutCampaign(ctx, pgCampaign("C-1004", "P-1001", "BRAND"))
		}},
		{name: "a campaign ending before it starts", write: func(ctx context.Context, s *campaigns.PostgresStore) error {
			return s.PutCampaign(ctx, backwards)
		}},
		{name: "a campaign of a plan that does not exist", write: func(ctx context.Context, s *campaigns.PostgresStore) error {
			return s.PutCampaign(ctx, pgCampaign("C-1004", "P-9999", "Lost"))
		}},
		{name: "an initiative of a campaign that does not exist", write: func(ctx context.Context, s *campaigns.PostgresStore) error {
			return s.PutInitiative(ctx, pgInitiative("I-1003", "C-9999"))
		}},
		{name: "a campaign deleted while it holds an initiative", write: func(ctx context.Context, s *campaigns.PostgresStore) error {
			return s.DeleteCampaign(ctx, "C-1001")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s, _, _ := pgFixture(t)

			err := db.Write(t.Context(), func(ctx context.Context) error { return tt.write(ctx, s) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}

func campaignIDs(list []campaigns.Campaign) []string {
	ids := []string{}
	for _, c := range list {
		ids = append(ids, c.ID)
	}
	return ids
}

func initiativeIDs(list []campaigns.Initiative) []string {
	ids := []string{}
	for _, i := range list {
		ids = append(ids, i.ID)
	}
	return ids
}

func contains(ids []string, id string) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}
