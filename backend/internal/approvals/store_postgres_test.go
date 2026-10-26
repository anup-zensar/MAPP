package approvals_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"mapp/internal/activities"
	"mapp/internal/approvals"
	"mapp/internal/campaigns"
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
	pgMeta                         = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))
	_         approvals.Repository = (*approvals.PostgresStore)(nil)
	submitted                      = time.Date(2026, time.April, 1, 9, 0, 0, 0, time.UTC)
	approver                       = actor.Actor{ID: "U-1003", Name: "Mei Chen"}
)

// pgPlanned gives a store whose database also holds the activities A-1001 (plan P-1001, team T-1001) and
// A-1002 (plan P-1002, team T-1002).
func pgPlanned(t *testing.T) (*database.DB, *approvals.PostgresStore) {
	t.Helper()
	db, s := pgStore(t)
	directory, catalogue := people.NewPostgresStore(db), settings.NewPostgresStore(db)
	planStore, containers, activityStore := plans.NewPostgresStore(db), campaigns.NewPostgresStore(db), activities.NewPostgresStore(db)
	pgWrite(t, db, func(ctx context.Context) error {
		err := errors.Join(
			directory.Put(ctx, people.Person{ID: "U-1003", Name: "Mei Chen", Email: "mei@example.com", Meta: pgMeta}),
			catalogue.PutCurrency(ctx, settings.Currency{Code: "USD", Name: "US dollar", Active: true, Meta: pgMeta}),
			catalogue.PutActivityType(ctx, settings.ActivityType{ID: "AT-1001", Name: "Webinar", Meta: pgMeta}),
			catalogue.PutModel(ctx, settings.Model{ID: "MOD-1001", Name: "Leads", ActivityTypeIDs: []string{"AT-1001"},
				InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(1000),
				AverageDealSize: money.USD(money.FromCents(100)), Meta: pgMeta}))
		for _, n := range []string{"1", "2"} {
			err = errors.Join(err,
				directory.PutTeam(ctx, people.Team{ID: "T-100" + n, Name: "Team " + n, Colour: "blue", Meta: pgMeta}),
				planStore.Put(ctx, plans.Plan{ID: "P-100" + n, Name: "Plan " + n, Year: 2026, ModelID: "MOD-1001",
					TeamIDs: []string{"T-100" + n}, DefaultSpread: plans.SpreadEvenly, GoalStage: plans.StageINQ,
					Status: plans.StatusDraft, Meta: pgMeta}),
				containers.PutCampaign(ctx, campaigns.Campaign{ID: "C-100" + n, PlanID: "P-100" + n, Name: "Campaign", Meta: pgMeta}),
				containers.PutInitiative(ctx, campaigns.Initiative{ID: "I-100" + n, CampaignID: "C-100" + n, Name: "Initiative",
					Meta: pgMeta}),
				activityStore.Put(ctx, activities.Activity{ID: "A-100" + n, InitiativeID: "I-100" + n, Name: "Activity",
					ActivityTypeID: "AT-1001", Status: activities.StatusSubmitted, Meta: pgMeta}))
		}
		return err
	})
	return db, s
}

func pgRound(id, activityID string, number int, outcome approvals.Outcome) approvals.Round {
	cost := money.USD(money.FromCents(120000))
	return approvals.Round{ID: id, ActivityID: activityID, Number: number, WorkflowID: "WF-1001", WorkflowName: "Standard",
		Levels: []approvals.RoundLevel{{Number: 1, Name: "Finance", Approvers: []actor.Actor{},
			ApproverType: approvals.SourcePeople, Mode: approvals.ModeAny, Resolution: approvals.ResolutionDecide,
			Excluded:  []approvals.Excluded{{Person: actor.Actor{ID: "U-1001", Name: "Priya Raman"}, Reason: approvals.ReasonOwner}},
			StartedAt: &submitted}},
		CurrentLevel: 1, Outcome: outcome, SubmittedAt: submitted, SubmittedBy: actor.Actor{ID: "U-1001", Name: "Priya Raman"},
		Facts: &approvals.RuleFacts{PlannedCost: &cost, BusinessUnits: []approvals.FactOption{}}, Meta: pgMeta}
}

// named names the approver Mei Chen in the round's first level.
func named(r approvals.Round) approvals.Round {
	r.Levels[0].Approvers = []actor.Actor{approver}
	return r
}

func pgAction(id, roundID, activityID string) approvals.Action {
	return approvals.Action{ID: id, RoundID: roundID, RoundNumber: 1, ActivityID: activityID,
		Action: approvals.ActionApproved, Level: 1, LevelName: "Finance", Approvers: []actor.Actor{approver},
		Actor: actor.Actor{ID: "U-1002", Name: "Daniel Okoro"}, OnBehalfOf: []actor.Actor{approver}, At: submitted,
		Changes: []approvals.ChangeText{}, Capacity: approvals.CapacityDelegate, AsDelegateOf: &approver}
}

func TestPostgresStoreRoundScopeIncludesNamedApproversFR009(t *testing.T) {
	t.Parallel()
	db, s := pgPlanned(t)
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutRound(ctx, pgRound("AR-1001", "A-1001", 1, approvals.OutcomeOpen)),
			s.PutRound(ctx, named(pgRound("AR-1002", "A-1002", 1, approvals.OutcomeOpen))),
			s.PutAction(ctx, pgAction("AH-1001", "AR-1001", "A-1001")), s.PutAction(ctx, pgAction("AH-1002", "AR-1002", "A-1002")))
	})
	tests := []struct {
		name         string
		scope        authz.Scope
		wantRounds   []string
		wantActivity string
	}{
		{name: "View plans in team T-1001", scope: authz.Scope{PersonID: "U-1001", Holds: true, TeamIDs: []string{"T-1001"}},
			wantRounds: []string{"AR-1001"}, wantActivity: "A-1001"},
		{name: "no role, approver named in a round of A-1002", scope: authz.Scope{PersonID: "U-1003", TeamIDs: []string{}},
			wantRounds: []string{"AR-1002"}, wantActivity: "A-1002"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rounds, roundsErr := s.RoundsInScope(t.Context(), tt.scope)
			actions, actionsErr := s.ActionsInScope(t.Context(), tt.scope)
			if err := errors.Join(roundsErr, actionsErr); err != nil {
				t.Fatalf("RoundsInScope(), ActionsInScope() error: %v", err)
			}
			if len(rounds) != 1 || rounds[0].ID != tt.wantRounds[0] || len(actions) != 1 || actions[0].ActivityID != tt.wantActivity {
				t.Errorf("RoundsInScope() = %d rounds, ActionsInScope() = %d entries; want only %v of %s",
					len(rounds), len(actions), tt.wantRounds, tt.wantActivity)
			}
		})
	}
}

// pgStore gives a store whose database already holds the approvers U-1001 and U-1002.
func pgStore(t *testing.T) (*database.DB, *approvals.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	directory := people.NewPostgresStore(db)
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(
			directory.Put(ctx, people.Person{ID: "U-1001", Name: "Priya Raman", Email: "priya@example.com", Meta: pgMeta}),
			directory.Put(ctx, people.Person{ID: "U-1002", Name: "Daniel Okoro", Email: "daniel@example.com", Meta: pgMeta}))
	})
	return db, approvals.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}
