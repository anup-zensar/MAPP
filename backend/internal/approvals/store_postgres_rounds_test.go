package approvals_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/activities"
	"mapp/internal/approvals"
	"mapp/internal/campaigns"
	"mapp/internal/people"
	"mapp/internal/plans"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

var (
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

func pgRound(id, activityID string, number int, outcome approvals.Outcome, approvers ...actor.Actor) approvals.Round {
	cost := money.USD(money.FromCents(120000))
	return approvals.Round{ID: id, ActivityID: activityID, Number: number, WorkflowID: "WF-1001", WorkflowName: "Standard",
		Levels: []approvals.RoundLevel{{Number: 1, Name: "Finance", Approvers: append([]actor.Actor{}, approvers...),
			ApproverType: approvals.SourcePeople, Mode: approvals.ModeAny, Resolution: approvals.ResolutionDecide,
			Excluded:  []approvals.Excluded{{Person: actor.Actor{ID: "U-1001", Name: "Priya Raman"}, Reason: approvals.ReasonOwner}},
			StartedAt: &submitted}},
		CurrentLevel: 1, Outcome: outcome, SubmittedAt: submitted, SubmittedBy: actor.Actor{ID: "U-1001", Name: "Priya Raman"},
		Facts: &approvals.RuleFacts{PlannedCost: &cost, BusinessUnits: []approvals.FactOption{}}, Meta: pgMeta}
}

func pgAction(id, roundID, activityID string) approvals.Action {
	return approvals.Action{ID: id, RoundID: roundID, RoundNumber: 1, ActivityID: activityID,
		Action: approvals.ActionApproved, Level: 1, LevelName: "Finance", Approvers: []actor.Actor{approver},
		Actor: actor.Actor{ID: "U-1002", Name: "Daniel Okoro"}, OnBehalfOf: []actor.Actor{approver}, At: submitted,
		Changes: []approvals.ChangeText{}, Capacity: approvals.CapacityDelegate, AsDelegateOf: &approver}
}

func TestPostgresStoreRoundsAndActionsFR008(t *testing.T) {
	t.Parallel()
	db, s := pgPlanned(t)
	ended := submitted.Add(time.Hour)
	first := pgRound("AR-1001", "A-1001", 1, approvals.OutcomeApproved, approver)
	first.EndedAt = &ended
	first.Levels = append(first.Levels, approvals.RoundLevel{Number: 2, Name: "Before 017"})
	second := pgRound("AR-1000", "A-1001", 2, approvals.OutcomeOpen)
	actions := []approvals.Action{pgAction("AH-1001", "AR-1001", "A-1001"), pgAction("AH-1002", "AR-1003", "A-1002")}
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutRound(ctx, second), s.PutRound(ctx, first),
			s.PutRound(ctx, pgRound("AR-1003", "A-1002", 1, approvals.OutcomeOpen)),
			s.PutAction(ctx, actions[1]), s.PutAction(ctx, actions[0]))
	})

	ofFirst, roundsErr := s.RoundsOf(t.Context(), "A-1001")
	gotActions, actionsErr := s.ActionsOf(t.Context(), "A-1001")

	if err := errors.Join(roundsErr, actionsErr); err != nil {
		t.Fatalf("RoundsOf(), ActionsOf() error: %v", err)
	}
	first.Levels[1] = approvals.RoundLevel{Number: 2, Name: "Before 017", ApproverType: approvals.SourcePeople,
		Mode: approvals.ModeAny, Resolution: approvals.ResolutionDecide, Approvers: []actor.Actor{},
		Excluded: []approvals.Excluded{}}
	if diff := cmp.Diff([]approvals.Round{first, second}, ofFirst, pgExact); diff != "" {
		t.Errorf("RoundsOf() by round number mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(actions[:1], gotActions, pgExact); diff != "" {
		t.Errorf("ActionsOf() mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreRoundScopeIncludesNamedApproversFR009(t *testing.T) {
	t.Parallel()
	db, s := pgPlanned(t)
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutRound(ctx, pgRound("AR-1001", "A-1001", 1, approvals.OutcomeOpen)),
			s.PutRound(ctx, pgRound("AR-1002", "A-1002", 1, approvals.OutcomeOpen, approver)),
			s.PutAction(ctx, pgAction("AH-1001", "AR-1001", "A-1001")), s.PutAction(ctx, pgAction("AH-1002", "AR-1002", "A-1002")))
	})
	tests := []struct {
		name         string
		scope        authz.Scope
		wantRounds   []string
		wantActivity string
	}{
		{name: "View plans in team T-1001", scope: authz.Scope{PersonID: "U-1001", ViewsPlans: true, TeamIDs: []string{"T-1001"}},
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

func TestPostgresStoreNotificationsAndDelegationsFR008(t *testing.T) {
	t.Parallel()
	db, s := pgPlanned(t)
	later, read := submitted.Add(time.Hour), submitted.Add(2*time.Hour)
	notification := func(id, recipientID string, at time.Time) approvals.Notification {
		return approvals.Notification{ID: id, RecipientID: recipientID, Kind: approvals.NotifyApprovalNeeded, Title: "Approve",
			Text: "Activity is waiting", ActivityID: "A-1001", ActivityName: "Activity", Actor: approver, At: at}
	}
	readOne := notification("NT-1001", "U-1001", submitted)
	readOne.Read, readOne.ReadAt = true, &read
	cancelled := approvals.Delegation{ID: "DLG-1001", PersonID: "U-1001", DelegateID: "U-1002",
		StartDate: date.MustParse("2026-05-01"), EndDate: date.MustParse("2026-05-10"), CancelledAt: &read,
		CancelledBy: &approver, Meta: pgMeta}
	overlapping := cancelled
	overlapping.ID, overlapping.CancelledAt, overlapping.CancelledBy = "DLG-1002", nil, nil
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutNotification(ctx, readOne), s.PutNotification(ctx, notification("NT-1002", "U-1001", later)),
			s.PutNotification(ctx, notification("NT-1003", "U-1002", later)), s.PutNotification(ctx, notification("NT-1004", "U-1001", later)),
			s.PutDelegation(ctx, cancelled), s.PutDelegation(ctx, overlapping))
	})

	mine, notificationsErr := s.NotificationsOf(t.Context(), "U-1001")
	delegations, delegationsErr := s.ListDelegations(t.Context())

	if err := errors.Join(notificationsErr, delegationsErr); err != nil {
		t.Fatalf("NotificationsOf(), ListDelegations() error: %v", err)
	}
	want := []approvals.Notification{notification("NT-1004", "U-1001", later), notification("NT-1002", "U-1001", later), readOne}
	if diff := cmp.Diff(want, mine, pgExact); diff != "" {
		t.Errorf("NotificationsOf() newest first, only the recipient's mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]approvals.Delegation{cancelled, overlapping}, delegations, pgExact); diff != "" {
		t.Errorf("ListDelegations() mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresStoreRefusesBrokenApprovalsFR004(t *testing.T) {
	t.Parallel()
	delegation := func(id, start, end string) approvals.Delegation {
		return approvals.Delegation{ID: id, PersonID: "U-1001", DelegateID: "U-1002", StartDate: date.MustParse(start),
			EndDate: date.MustParse(end), Meta: pgMeta}
	}
	tests := []struct {
		name  string
		write func(ctx context.Context, s *approvals.PostgresStore) error
	}{
		{name: "a second open round of one activity", write: func(ctx context.Context, s *approvals.PostgresStore) error {
			return s.PutRound(ctx, pgRound("AR-1002", "A-1001", 2, approvals.OutcomeOpen))
		}},
		{name: "a second round with the same number", write: func(ctx context.Context, s *approvals.PostgresStore) error {
			return s.PutRound(ctx, pgRound("AR-1002", "A-1001", 1, approvals.OutcomeDeclined))
		}},
		{name: "a delegation overlapping another of the same person", write: func(ctx context.Context, s *approvals.PostgresStore) error {
			return s.PutDelegation(ctx, delegation("DLG-1002", "2026-05-05", "2026-05-20"))
		}},
		{name: "a delegation longer than 90 days", write: func(ctx context.Context, s *approvals.PostgresStore) error {
			return s.PutDelegation(ctx, delegation("DLG-1002", "2026-06-01", "2026-09-30"))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s := pgPlanned(t)
			pgWrite(t, db, func(ctx context.Context) error {
				return errors.Join(s.PutRound(ctx, pgRound("AR-1001", "A-1001", 1, approvals.OutcomeOpen)),
					s.PutDelegation(ctx, delegation("DLG-1001", "2026-05-01", "2026-05-10")))
			})

			err := db.Write(t.Context(), func(ctx context.Context) error { return tt.write(ctx, s) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}
