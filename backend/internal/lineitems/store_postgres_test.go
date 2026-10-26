package lineitems_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"mapp/internal/activities"
	"mapp/internal/campaigns"
	"mapp/internal/lineitems"
	"mapp/internal/people"
	"mapp/internal/plans"
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
	pgExact = cmpopts.EquateComparable(money.Amount{}, money.Percent{})
)

// pgStore gives a store whose database holds the activities A-1001 (plan P-1001, team T-1001, owned by U-1002)
// and A-1002 (plan P-1002, team T-1002), with everything they refer to.
func pgStore(t *testing.T) (*database.DB, *lineitems.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	directory, catalogue := people.NewPostgresStore(db), settings.NewPostgresStore(db)
	planStore, containers, activityStore := plans.NewPostgresStore(db), campaigns.NewPostgresStore(db), activities.NewPostgresStore(db)
	puts := []func(ctx context.Context) error{func(ctx context.Context) error {
		return errors.Join(
			catalogue.PutCurrency(ctx, settings.Currency{Code: "USD", Name: "US dollar", Active: true, Meta: pgMeta}),
			catalogue.PutActivityType(ctx, settings.ActivityType{ID: "AT-1001", Name: "Webinar", Meta: pgMeta}),
			catalogue.PutModel(ctx, settings.Model{ID: "MOD-1001", Name: "Leads", ActivityTypeIDs: []string{"AT-1001"},
				InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(1000),
				AverageDealSize: money.USD(money.FromCents(100)), Meta: pgMeta}))
	}}
	for _, n := range []string{"1", "2"} {
		team, plan, campaign, initiative := "T-100"+n, "P-100"+n, "C-100"+n, "I-100"+n
		owner := map[string]string{"1": "U-1002", "2": ""}[n]
		puts = append(puts, func(ctx context.Context) error {
			return errors.Join(
				directory.Put(ctx, people.Person{ID: "U-100" + n, Name: "Person " + n, Email: n + "@example.com", Meta: pgMeta}),
				directory.PutTeam(ctx, people.Team{ID: team, Name: "Team " + n, Colour: "blue", Meta: pgMeta}),
				planStore.Put(ctx, plans.Plan{ID: plan, Name: "Plan " + n, Year: 2026, ModelID: "MOD-1001", TeamIDs: []string{team},
					DefaultSpread: plans.SpreadEvenly, GoalStage: plans.StageINQ, Status: plans.StatusDraft, Meta: pgMeta}),
				containers.PutCampaign(ctx, campaigns.Campaign{ID: campaign, PlanID: plan, Name: "Campaign " + n, Meta: pgMeta}),
				containers.PutInitiative(ctx, campaigns.Initiative{ID: initiative, CampaignID: campaign, Name: "Initiative " + n,
					Meta: pgMeta}),
				activityStore.Put(ctx, activities.Activity{ID: "A-100" + n, InitiativeID: initiative, Name: "Activity " + n,
					ActivityTypeID: "AT-1001", OwnerID: owner, Status: activities.StatusCreated, Meta: pgMeta}))
		})
	}
	pgWrite(t, db, func(ctx context.Context) error {
		var err error
		for _, put := range puts {
			err = errors.Join(err, put(ctx))
		}
		return err
	})
	return db, lineitems.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgLineItem(id, activityID, name string) lineitems.LineItem {
	return lineitems.LineItem{ID: id, ActivityID: activityID, Name: name, Currency: "USD",
		PlannedCost: money.FromCents(120000), Spread: lineitems.Spread{Method: lineitems.SpreadEvenly, Months: []lineitems.MonthAmount{}},
		Attributes: settings.AttributeValues{}, Meta: pgMeta}
}

// pgFixture saves L-1001 in A-1001 with every optional field set and L-1002 in A-1002 with none.
func pgFixture(t *testing.T) (*database.DB, *lineitems.PostgresStore, []lineitems.LineItem) {
	t.Helper()
	db, s := pgStore(t)
	contracted, year := money.FromCents(100000), 2026
	full := pgLineItem("L-1001", "A-1001", "Agency fee")
	full.OwnerID, full.Description, full.ContractedAmount, full.RequestSwbs = "U-1001", "Fee", &contracted, true
	full.FinanceBudgetYear = &year
	full.Attributes = settings.AttributeValues{"SA-1004": json.RawMessage(`true`)}
	full.Spread = lineitems.Spread{Method: lineitems.SpreadCustom, Months: []lineitems.MonthAmount{
		{Month: "2026-04", Amount: money.FromCents(20000)}, {Month: "2026-05", Amount: money.FromCents(100000)}}}
	stored := []lineitems.LineItem{full, pgLineItem("L-1002", "A-1002", "Agency fee")}
	pgWrite(t, db, func(ctx context.Context) error { return errors.Join(s.Put(ctx, stored[0]), s.Put(ctx, stored[1])) })
	return db, s, stored
}

func TestPostgresStoreLineItemsReadBackAsSavedFR008(t *testing.T) {
	t.Parallel()
	db, s, stored := pgFixture(t)
	evenly := stored[0]
	evenly.Spread = lineitems.Spread{Method: lineitems.SpreadEvenly, Months: []lineitems.MonthAmount{}}
	pgWrite(t, db, func(ctx context.Context) error { return s.Put(ctx, evenly) })

	all, allErr := s.List(t.Context())
	ofSecond, byErr := s.ListByActivities(t.Context(), []string{"A-1002"})

	if err := errors.Join(allErr, byErr); err != nil {
		t.Fatalf("List(), ListByActivities() error: %v", err)
	}
	if diff := cmp.Diff([]lineitems.LineItem{evenly, stored[1]}, all, pgExact); diff != "" {
		t.Errorf("List() after replacing the spread mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(stored[1:], ofSecond, pgExact); diff != "" {
		t.Errorf("ListByActivities() mismatch (-want +got):\n%s", diff)
	}
	if one, found, err := s.Get(t.Context(), "L-1002"); err != nil || !found || !cmp.Equal(stored[1], one, pgExact) {
		t.Errorf("Get() = %+v, found %v, error %v; want %+v", one, found, err, stored[1])
	}
}

func TestPostgresStoreLineItemScopeFollowsTheActivityFR009(t *testing.T) {
	t.Parallel()
	_, s, stored := pgFixture(t)
	tests := []struct {
		name  string
		scope authz.Scope
		want  []string
	}{
		{name: "View plans in team T-1002", scope: authz.Scope{PersonID: "U-1001", Holds: true, TeamIDs: []string{"T-1002"}},
			want: []string{"L-1002"}},
		{name: "no role, owner of activity A-1001", scope: authz.Scope{PersonID: "U-1002", TeamIDs: []string{}},
			want: []string{"L-1001"}},
		{name: "no role, owner of the line item only", scope: authz.Scope{PersonID: "U-1001", TeamIDs: []string{}},
			want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			visible, err := s.ListInScope(t.Context(), tt.scope)
			if err != nil {
				t.Fatalf("ListInScope() error: %v", err)
			}
			ids := []string{}
			for _, li := range visible {
				ids = append(ids, li.ID)
			}
			if diff := cmp.Diff(tt.want, ids); diff != "" {
				t.Errorf("ListInScope() mismatch (-want +got):\n%s", diff)
			}
			for _, li := range stored {
				_, found, err := s.GetInScope(t.Context(), tt.scope, li.ID)
				if err != nil || found != cmp.Equal(tt.want, []string{li.ID}) {
					t.Errorf("GetInScope(%s) = found %v, error %v", li.ID, found, err)
				}
			}
		})
	}
}

func TestPostgresStoreRefusesBrokenLineItemsFR004(t *testing.T) {
	t.Parallel()
	unknownCurrency := pgLineItem("L-1003", "A-1001", "Venue")
	unknownCurrency.Currency = "XYZ"
	negativeMonth := pgLineItem("L-1003", "A-1001", "Refund")
	negativeMonth.Spread = lineitems.Spread{Method: lineitems.SpreadCustom, Months: []lineitems.MonthAmount{
		{Month: "2026-04", Amount: money.FromCents(-100)}}}
	tests := []struct {
		name     string
		lineItem lineitems.LineItem
	}{
		{name: "a name taken in the activity, in another letter case", lineItem: pgLineItem("L-1003", "A-1001", "AGENCY FEE")},
		{name: "a currency that does not exist", lineItem: unknownCurrency},
		{name: "a negative amount in a month", lineItem: negativeMonth},
		{name: "an activity that does not exist", lineItem: pgLineItem("L-1003", "A-9999", "Lost")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s, _ := pgFixture(t)

			err := db.Write(t.Context(), func(ctx context.Context) error { return s.Put(ctx, tt.lineItem) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}
