package performance_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"mapp/internal/activity"
	"mapp/internal/campaign"
	"mapp/internal/people"
	"mapp/internal/performance"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

var pgMeta = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))

// team1Only holds plans.view in team T-1001 alone, so it reaches A-1001 and nothing of A-1002.
var team1Only = authz.Scope{PersonID: "U-2001", Holds: true, TeamIDs: []string{"T-1001"}}

// pgPerformance gives a store whose database holds A-1001 (team T-1001) and A-1002 (team T-1002), each with
// a fact, a media code, an integration id and an external name of its own.
func pgPerformance(t *testing.T) *performance.PostgresStore {
	t.Helper()
	store, _ := pgPerformanceDB(t)
	return store
}

func pgPerformanceDB(t *testing.T) (*performance.PostgresStore, *database.DB) {
	t.Helper()
	db := dbtest.NewDB(t)
	store := performance.NewPostgresStore(db)
	directory, catalogue := people.NewPostgresStore(db), settings.NewPostgresStore(db)
	plans, containers := plan.NewPostgresStore(db), campaign.NewPostgresStore(db)
	activities := activity.NewPostgresStore(db)

	pgWrite(t, db, func(ctx context.Context) error {
		err := errors.Join(
			directory.Put(ctx, people.Person{ID: "U-2001", Name: "Sam Patel", Email: "sam@example.com", Meta: pgMeta}),
			catalogue.PutCurrency(ctx, settings.Currency{Code: "USD", Name: "US dollar", Active: true, Meta: pgMeta}),
			catalogue.PutActivityType(ctx, settings.ActivityType{ID: "AT-1001", Name: "Webinar", Meta: pgMeta}),
			catalogue.PutModel(ctx, settings.Model{ID: "MOD-1001", Name: "Leads", ActivityTypeIDs: []string{"AT-1001"},
				InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(1000),
				AverageDealSize: money.USD(money.FromCents(100)), Meta: pgMeta}))
		for _, n := range []string{"1", "2"} {
			err = errors.Join(err,
				directory.PutTeam(ctx, people.Team{ID: "T-100" + n, Name: "Team " + n, Colour: "blue", Meta: pgMeta}),
				plans.Put(ctx, plan.Plan{ID: "P-100" + n, Name: "Plan " + n, Year: 2026, ModelID: "MOD-1001",
					TeamIDs: []string{"T-100" + n}, DefaultSpread: plan.SpreadEvenly, GoalStage: plan.StageINQ,
					Status: plan.StatusDraft, Meta: pgMeta}),
				containers.PutCampaign(ctx, campaign.Campaign{ID: "C-100" + n, PlanID: "P-100" + n, Name: "Campaign", Meta: pgMeta}),
				containers.PutInitiative(ctx, campaign.Initiative{ID: "I-100" + n, CampaignID: "C-100" + n, Name: "Initiative", Meta: pgMeta}),
				activities.Put(ctx, activity.Activity{ID: "A-100" + n, InitiativeID: "I-100" + n, Name: "Activity " + n,
					ExternalName: "activity-" + n, ActivityTypeID: "AT-1001", Status: activity.StatusApproved, Meta: pgMeta}))
		}
		return err
	})

	pgWrite(t, db, func(ctx context.Context) error {
		err := store.PutSource(ctx, performance.Source{ID: "PS-1", Name: "Eloqua", OwnerID: "U-2001",
			Category: performance.CategoryMarketingAutomation, Status: performance.StatusActive,
			DeliveryRoute: performance.RouteUpload, Cadence: performance.CadenceMonthly, Meta: pgMeta})
		err = errors.Join(err, store.PutImport(ctx, performance.Import{ID: "PIM-1", SourceID: "PS-1",
			FileName: "feed.csv", Origin: performance.OriginUpload, Status: performance.ImportCompleted,
			Counts: performance.Counts{Read: 2, Matched: 2}, Meta: pgMeta}))
		for _, n := range []string{"1", "2"} {
			err = errors.Join(err,
				store.PutFact(ctx, performance.Fact{ID: "PF-" + n, SourceID: "PS-1",
					ImportID: "PIM-1", SourceReference: "r" + n, ActivityID: "A-100" + n,
					MatchedOn: performance.MatchedActivityID, FactDate: pgMeta.CreatedAt, Stage: "inq",
					Count: countOf(5), Meta: pgMeta}),
				store.PutMediaCode(ctx, performance.MediaCode{ID: "MC-" + n, Code: int64(9000000 + atoiOf(n)),
					ActivityID: "A-100" + n, Attributes: map[string]string{}, Status: performance.StatusActive,
					Meta: pgMeta}),
				store.PutIntegrationID(ctx, performance.IntegrationID{ID: "IID-" + n, ActivityID: "A-100" + n,
					SourceID: "PS-1", Value: "ext-" + n, Meta: pgMeta}))
		}
		return err
	})
	return store, db
}

func countOf(n int64) *int64 { return &n }

func atoiOf(n string) int {
	if n == "2" {
		return 2
	}
	return 1
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func TestPostgresStore_FactsOutsideTheScopeAreInvisible_FR044(t *testing.T) {
	t.Parallel()
	store := pgPerformance(t)

	mine, err := store.ListFactsForActivity(t.Context(), team1Only, "A-1001", false)
	if err != nil {
		t.Fatalf("ListFactsForActivity(A-1001) error: %v", err)
	}
	if len(mine) != 1 {
		t.Errorf("in my own team I see %d facts, want 1", len(mine))
	}

	theirs, err := store.ListFactsForActivity(t.Context(), team1Only, "A-1002", false)
	if err != nil {
		t.Fatalf("ListFactsForActivity(A-1002) error: %v", err)
	}
	if len(theirs) != 0 {
		t.Errorf("outside my scope I see %d facts, want 0: a record outside the scope must look missing", len(theirs))
	}
}

func TestPostgresStore_MatchingNeverReachesOutsideTheScope_FR044(t *testing.T) {
	t.Parallel()
	store := pgPerformance(t)

	t.Run("FR-044 by activity id", func(t *testing.T) {
		t.Parallel()
		found, err := store.MatchByActivityID(t.Context(), team1Only, []string{"A-1001", "A-1002"})
		if err != nil {
			t.Fatalf("MatchByActivityID() error: %v", err)
		}
		if _, reached := found["A-1002"]; reached {
			t.Error("a key outside the scope resolved: matching must not reach it")
		}
		if _, reached := found["A-1001"]; !reached {
			t.Error("a key inside the scope did not resolve")
		}
	})

	t.Run("FR-044 by external name", func(t *testing.T) {
		t.Parallel()
		found, err := store.MatchByExternalName(t.Context(), team1Only, []string{"activity-1", "activity-2"})
		if err != nil {
			t.Fatalf("MatchByExternalName() error: %v", err)
		}
		if _, reached := found["activity-2"]; reached {
			t.Error("an external name outside the scope resolved")
		}
		if _, reached := found["activity-1"]; !reached {
			t.Error("an external name inside the scope did not resolve")
		}
	})
}

func TestPostgresStore_MediaCodesOutsideTheScopeAreInvisible_FR044(t *testing.T) {
	t.Parallel()
	store := pgPerformance(t)

	mine, err := store.ListMediaCodes(t.Context(), team1Only, "A-1001")
	if err != nil {
		t.Fatalf("ListMediaCodes(A-1001) error: %v", err)
	}
	if len(mine) != 1 {
		t.Fatalf("in my own team I see %d media codes, want 1: without a row here the test proves nothing", len(mine))
	}

	theirs, err := store.ListMediaCodes(t.Context(), team1Only, "A-1002")
	if err != nil {
		t.Fatalf("ListMediaCodes(A-1002) error: %v", err)
	}
	if len(theirs) != 0 {
		t.Errorf("outside my scope I see %d media codes, want 0", len(theirs))
	}
}

func TestPostgresStore_MatchByMediaCodeStaysInScope_FR044(t *testing.T) {
	t.Parallel()
	store := pgPerformance(t)

	found, err := store.MatchByMediaCode(t.Context(), team1Only, []int64{9000001, 9000002})
	if err != nil {
		t.Fatalf("MatchByMediaCode() error: %v", err)
	}
	if _, reached := found["9000001"]; !reached {
		t.Error("a media code inside the scope did not resolve")
	}
	if _, reached := found["9000002"]; reached {
		t.Error("a media code outside the scope resolved: matching must not reach it")
	}
}

func TestPostgresStore_MatchByIntegrationIDStaysInScope_FR044(t *testing.T) {
	t.Parallel()
	store := pgPerformance(t)

	found, err := store.MatchByIntegrationID(t.Context(), team1Only, "PS-1", []string{"ext-1", "ext-2"})
	if err != nil {
		t.Fatalf("MatchByIntegrationID() error: %v", err)
	}
	if _, reached := found["ext-1"]; !reached {
		t.Error("an integration id inside the scope did not resolve")
	}
	if _, reached := found["ext-2"]; reached {
		t.Error("an integration id outside the scope resolved: matching must not reach it")
	}
}
