package lineitem_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/activity"
	"mapp/internal/campaign"
	"mapp/internal/lineitem"
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
	pgMeta = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))
)

// pgStore gives a store whose database holds the activities A-1001 (plan P-1001, team T-1001, owned by U-1002)
// and A-1002 (plan P-1002, team T-1002), with everything they refer to.
func pgStore(t *testing.T) (*database.DB, *lineitem.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	directory, catalogue := people.NewPostgresStore(db), settings.NewPostgresStore(db)
	planStore, containers, activityStore := plan.NewPostgresStore(db), campaign.NewPostgresStore(db), activity.NewPostgresStore(db)
	puts := []func(ctx context.Context) error{func(ctx context.Context) error {
		return errors.Join(
			catalogue.PutCurrency(ctx, settings.Currency{Code: "USD", Name: "US dollar", Active: true, Meta: pgMeta}),
			catalogue.PutActivityType(ctx, settings.ActivityType{ID: "AT-1001", Name: "Webinar", Meta: pgMeta}),
			catalogue.PutModel(ctx, settings.Model{ID: "MOD-1001", Name: "Leads", ActivityTypeIDs: []string{"AT-1001"},
				InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(1000),
				AverageDealSize: money.USD(money.FromCents(100)), Meta: pgMeta}))
	}}
	for _, n := range []string{"1", "2"} {
		team, p, c, initiative := "T-100"+n, "P-100"+n, "C-100"+n, "I-100"+n
		owner := map[string]string{"1": "U-1002", "2": ""}[n]
		puts = append(puts, func(ctx context.Context) error {
			return errors.Join(
				directory.Put(ctx, people.Person{ID: "U-100" + n, Name: "Person " + n, Email: n + "@example.com", Meta: pgMeta}),
				directory.PutTeam(ctx, people.Team{ID: team, Name: "Team " + n, Colour: "blue", Meta: pgMeta}),
				planStore.Put(ctx, plan.Plan{ID: p, Name: "Plan " + n, Year: 2026, ModelID: "MOD-1001", TeamIDs: []string{team},
					DefaultSpread: plan.SpreadEvenly, GoalStage: plan.StageINQ, Status: plan.StatusDraft, Meta: pgMeta}),
				containers.PutCampaign(ctx, campaign.Campaign{ID: c, PlanID: p, Name: "Campaign " + n, Meta: pgMeta}),
				containers.PutInitiative(ctx, campaign.Initiative{ID: initiative, CampaignID: c, Name: "Initiative " + n,
					Meta: pgMeta}),
				activityStore.Put(ctx, activity.Activity{ID: "A-100" + n, InitiativeID: initiative, Name: "Activity " + n,
					ActivityTypeID: "AT-1001", OwnerID: owner, Status: activity.StatusCreated, Meta: pgMeta}))
		})
	}
	pgWrite(t, db, func(ctx context.Context) error {
		var err error
		for _, put := range puts {
			err = errors.Join(err, put(ctx))
		}
		return err
	})
	return db, lineitem.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgLineItem(id, activityID, name string) lineitem.LineItem {
	return lineitem.LineItem{ID: id, ActivityID: activityID, Name: name, Currency: "USD",
		PlannedCost: money.FromCents(120000), Spread: lineitem.Spread{Method: lineitem.SpreadEvenly, Months: []lineitem.MonthAmount{}},
		Attributes: settings.AttributeValues{}, Meta: pgMeta}
}

// pgFixture saves L-1001 in A-1001 with every optional field set and L-1002 in A-1002 with none.
func pgFixture(t *testing.T) (*database.DB, *lineitem.PostgresStore, []lineitem.LineItem) {
	t.Helper()
	db, s := pgStore(t)
	contracted, year := money.FromCents(100000), 2026
	full := pgLineItem("L-1001", "A-1001", "Agency fee")
	full.OwnerID, full.Description, full.ContractedAmount, full.RequestSwbs = "U-1001", "Fee", &contracted, true
	full.FinanceBudgetYear = &year
	full.Attributes = settings.AttributeValues{"SA-1004": json.RawMessage(`true`)}
	full.Spread = lineitem.Spread{Method: lineitem.SpreadCustom, Months: []lineitem.MonthAmount{
		{Month: "2026-04", Amount: money.FromCents(20000)}, {Month: "2026-05", Amount: money.FromCents(100000)}}}
	stored := []lineitem.LineItem{full, pgLineItem("L-1002", "A-1002", "Agency fee")}
	pgWrite(t, db, func(ctx context.Context) error { return errors.Join(s.Put(ctx, stored[0]), s.Put(ctx, stored[1])) })
	return db, s, stored
}

// TestPostgresStoreActivityAndLineItemScopeFR009 proves every scoped read of activities and line items: an
// activity is seen through its plan's scope or by its owner, and its line items follow it.
func TestPostgresStoreActivityAndLineItemScopeFR009(t *testing.T) {
	t.Parallel()
	db, s, _ := pgFixture(t)
	activityStore := activity.NewPostgresStore(db)
	tests := []struct {
		name  string
		scope authz.Scope
		want  []string
	}{
		{name: "View plans in team T-1002", scope: authz.Scope{PersonID: "U-1001", Holds: true, TeamIDs: []string{"T-1002"}},
			want: []string{"A-1002", "L-1002"}},
		{name: "no role, owner of activity A-1001", scope: authz.Scope{PersonID: "U-1002", TeamIDs: []string{}},
			want: []string{"A-1001", "L-1001"}},
		{name: "no role, owner of the line item only", scope: authz.Scope{PersonID: "U-1001", TeamIDs: []string{}},
			want: []string{}},
		{name: "everything", scope: authz.Scope{Everything: true, TeamIDs: []string{}},
			want: []string{"A-1001", "A-1002", "L-1001", "L-1002"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			listedActivities, activitiesErr := activityStore.ListInScope(t.Context(), tt.scope)
			listedItems, itemsErr := s.ListInScope(t.Context(), tt.scope)
			if err := errors.Join(activitiesErr, itemsErr); err != nil {
				t.Fatalf("ListInScope() error: %v", err)
			}
			if diff := cmp.Diff(tt.want, listedIDs(listedActivities, listedItems)); diff != "" {
				t.Errorf("ListInScope() mismatch (-want +got):\n%s", diff)
			}
			for _, n := range []string{"1001", "1002"} {
				_, activityFound, activityErr := activityStore.GetInScope(t.Context(), tt.scope, "A-"+n)
				_, itemFound, itemErr := s.GetInScope(t.Context(), tt.scope, "L-"+n)
				wantFound := slices.Contains(tt.want, "A-"+n)
				if err := errors.Join(activityErr, itemErr); err != nil || activityFound != wantFound || itemFound != wantFound {
					t.Errorf("GetInScope(A-%s, L-%s) = found %v and %v, error %v; want %v", n, n, activityFound, itemFound, err, wantFound)
				}
			}
		})
	}
}

func listedIDs(listedActivities []activity.Activity, listedItems []lineitem.LineItem) []string {
	ids := []string{}
	for _, a := range listedActivities {
		ids = append(ids, a.ID)
	}
	for _, li := range listedItems {
		ids = append(ids, li.ID)
	}
	return ids
}
