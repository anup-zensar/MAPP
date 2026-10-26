package activities_test

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
	"mapp/internal/people"
	"mapp/internal/plans"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

var (
	pgMeta  = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))
	pgExact = cmpopts.EquateComparable(money.Amount{}, money.Percent{}, date.Date{})
)

// pgStore gives a store whose database holds the initiatives I-1001 (plan P-1001, team T-1001) and I-1002
// (plan P-1002, team T-1002), with everything they refer to.
func pgStore(t *testing.T) (*database.DB, *activities.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	directory, catalogue := people.NewPostgresStore(db), settings.NewPostgresStore(db)
	planStore, containers := plans.NewPostgresStore(db), campaigns.NewPostgresStore(db)
	puts := []func(ctx context.Context) error{
		func(ctx context.Context) error {
			return catalogue.PutCurrency(ctx, settings.Currency{Code: "USD", Name: "US dollar", Active: true, Meta: pgMeta})
		},
		func(ctx context.Context) error {
			return catalogue.PutActivityType(ctx, settings.ActivityType{ID: "AT-1001", Name: "Webinar", Meta: pgMeta})
		},
		func(ctx context.Context) error {
			return catalogue.PutModel(ctx, settings.Model{ID: "MOD-1001", Name: "Leads", ActivityTypeIDs: []string{"AT-1001"},
				InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(1000),
				AverageDealSize: money.USD(money.FromCents(100)), Meta: pgMeta})
		},
	}
	for _, n := range []string{"1", "2"} {
		team, plan, campaign, initiative := "T-100"+n, "P-100"+n, "C-100"+n, "I-100"+n
		puts = append(puts, func(ctx context.Context) error {
			return errors.Join(
				directory.Put(ctx, people.Person{ID: "U-100" + n, Name: "Person " + n, Email: n + "@example.com", Meta: pgMeta}),
				directory.PutTeam(ctx, people.Team{ID: team, Name: "Team " + n, Colour: "blue", Meta: pgMeta}),
				planStore.Put(ctx, plans.Plan{ID: plan, Name: "Plan " + n, Year: 2026, ModelID: "MOD-1001", TeamIDs: []string{team},
					DefaultSpread: plans.SpreadEvenly, GoalStage: plans.StageINQ, Status: plans.StatusDraft, Meta: pgMeta}),
				containers.PutCampaign(ctx, campaigns.Campaign{ID: campaign, PlanID: plan, Name: "Campaign " + n, Meta: pgMeta}),
				containers.PutInitiative(ctx, campaigns.Initiative{ID: initiative, CampaignID: campaign, Name: "Initiative " + n,
					Meta: pgMeta}))
		})
	}
	pgWrite(t, db, func(ctx context.Context) error {
		var err error
		for _, put := range puts {
			err = errors.Join(err, put(ctx))
		}
		return err
	})
	return db, activities.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgActivity(id, initiativeID, name string) activities.Activity {
	return activities.Activity{ID: id, InitiativeID: initiativeID, Name: name, ActivityTypeID: "AT-1001",
		Attributes: settings.AttributeValues{}, Status: activities.StatusCreated, Meta: pgMeta}
}

// pgFixture saves A-1001 in plan P-1001, owned by U-1002, with every optional field set, and A-1002 in plan
// P-1002 with none.
func pgFixture(t *testing.T) (*database.DB, *activities.PostgresStore, []activities.Activity) {
	t.Helper()
	db, s := pgStore(t)
	start, cost, revenue := date.MustParse("2026-04-01"), money.USD(money.FromCents(250000)), money.USD(money.FromCents(9000000))
	inq, mql := int64(400), int64(80)
	full := pgActivity("A-1001", "I-1001", "Launch film")
	full.ExternalName, full.OwnerID, full.StartDate, full.Ongoing, full.PlannedCost = "Film", "U-1002", &start, true, &cost
	full.ApprovalWorkflowID, full.Description, full.SplitFromID = "WF-1001", "A film", "A-1000"
	full.Attributes = settings.AttributeValues{"SA-1003": json.RawMessage(`[{"optionId":"SO-1001","share":"60.00"}]`)}
	full.INQ, full.SetLaterStagesMyself, full.MQL, full.Revenue = &inq, true, &mql, &revenue
	full.Status = activities.StatusSubmitted
	stored := []activities.Activity{full, pgActivity("A-1002", "I-1002", "Launch film")}
	pgWrite(t, db, func(ctx context.Context) error { return errors.Join(s.Put(ctx, stored[0]), s.Put(ctx, stored[1])) })
	return db, s, stored
}

func TestPostgresStoreActivitiesReadBackAsSavedFR008(t *testing.T) {
	t.Parallel()
	_, s, stored := pgFixture(t)

	all, allErr := s.List(t.Context())
	ofSecond, byErr := s.ListByInitiatives(t.Context(), []string{"I-1002"})

	if err := errors.Join(allErr, byErr); err != nil {
		t.Fatalf("List(), ListByInitiatives() error: %v", err)
	}
	if diff := cmp.Diff(stored, all, pgExact); diff != "" {
		t.Errorf("List() mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(stored[1:], ofSecond, pgExact); diff != "" {
		t.Errorf("ListByInitiatives() mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreActivityScopeIsolationFR009(t *testing.T) {
	t.Parallel()
	_, s, _ := pgFixture(t)
	tests := []struct {
		name  string
		scope authz.Scope
		want  []string
	}{
		{name: "View plans in team T-1002", scope: authz.Scope{PersonID: "U-1001", Holds: true, TeamIDs: []string{"T-1002"}},
			want: []string{"A-1002"}},
		{name: "no role, owner of A-1001 only", scope: authz.Scope{PersonID: "U-1002", TeamIDs: []string{}},
			want: []string{"A-1001"}},
		{name: "everything", scope: authz.Scope{Everything: true, TeamIDs: []string{}}, want: []string{"A-1001", "A-1002"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			visible, err := s.ListInScope(t.Context(), tt.scope)
			if err != nil {
				t.Fatalf("ListInScope() error: %v", err)
			}
			if diff := cmp.Diff(tt.want, activityIDs(visible)); diff != "" {
				t.Errorf("ListInScope() mismatch (-want +got):\n%s", diff)
			}
			for _, id := range []string{"A-1001", "A-1002"} {
				_, found, err := s.GetInScope(t.Context(), tt.scope, id)
				if err != nil || found != (len(tt.want) == 2 || tt.want[0] == id) {
					t.Errorf("GetInScope(%s) = found %v, error %v", id, found, err)
				}
			}
		})
	}
}

func TestPostgresStoreRefusesBrokenActivitiesFR004(t *testing.T) {
	t.Parallel()
	mql := int64(10)
	laterStagesWithoutTheChoice := pgActivity("A-1003", "I-1001", "Later stages")
	laterStagesWithoutTheChoice.MQL = &mql
	unknownType := pgActivity("A-1003", "I-1001", "Unknown type")
	unknownType.ActivityTypeID = "AT-9999"
	tests := []struct {
		name     string
		activity activities.Activity
	}{
		{name: "a name taken in the initiative, in another letter case", activity: pgActivity("A-1003", "I-1001", "LAUNCH FILM")},
		{name: "an MQL without Set later stages myself", activity: laterStagesWithoutTheChoice},
		{name: "an activity type that does not exist", activity: unknownType},
		{name: "an initiative that does not exist", activity: pgActivity("A-1003", "I-9999", "Lost")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s, _ := pgFixture(t)

			err := db.Write(t.Context(), func(ctx context.Context) error { return s.Put(ctx, tt.activity) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}

func activityIDs(list []activities.Activity) []string {
	ids := []string{}
	for _, a := range list {
		ids = append(ids, a.ID)
	}
	return ids
}
