package approval_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/platform/authz"
)

// The Approvals board reads only the activities with a round, in the scope: a person sees an activity of
// their teams' plans or one whose round names them, and only among the ids asked for (028 FR-007).
func TestPostgresStoreActivitiesInScopeOfIDs_028_FR007(t *testing.T) {
	t.Parallel()
	db, s := pgPlanned(t)
	pgWrite(t, db, func(ctx context.Context) error {
		return s.PutRound(ctx, named(pgRound("AR-1002", "A-1002", 1, approval.OutcomeOpen)))
	})
	store := activity.NewPostgresStore(db)
	both := []string{"A-1001", "A-1002"}
	tests := []struct {
		name  string
		scope authz.Scope
		ids   []string
		want  []string
	}{
		{name: "View plans in team T-1001", scope: authz.Scope{PersonID: "U-1001", Holds: true, TeamIDs: []string{"T-1001"}},
			ids: both, want: []string{"A-1001"}},
		{name: "no role, approver named in a round of A-1002", scope: authz.Scope{PersonID: "U-1003", TeamIDs: []string{}},
			ids: both, want: []string{"A-1002"}},
		{name: "everything, among the ids asked for", scope: authz.Scope{Everything: true, TeamIDs: []string{}},
			ids: []string{"A-1002", "A-1404"}, want: []string{"A-1002"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seen, err := store.ListInScopeOf(t.Context(), tt.scope, tt.ids)
			if err != nil {
				t.Fatalf("ListInScopeOf() error: %v", err)
			}
			got := []string{}
			for _, a := range seen {
				got = append(got, a.ID)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ListInScopeOf(%v) mismatch (-want +got):\n%s", tt.ids, diff)
			}
		})
	}
}

// A dashboard reads the rounds and history of the records in its scope only: those of another market's activity
// (A-1002, plan P-1002 of team T-1002) are never read (D-356).
func TestPostgresStoreRoundsOfSubjectsScopeIsolation_036_FR002(t *testing.T) {
	t.Parallel()
	db, s := pgPlanned(t)
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutRound(ctx, pgRound("AR-1001", "A-1001", 1, approval.OutcomeOpen)),
			s.PutRound(ctx, pgRound("AR-1002", "A-1002", 1, approval.OutcomeOpen)),
			s.PutAction(ctx, pgAction("AH-1001", "AR-1001", "A-1001")), s.PutAction(ctx, pgAction("AH-1002", "AR-1002", "A-1002")))
	})
	tests := []struct {
		name     string
		subjects []string
		want     [][]string
	}{
		{name: "team T-1001's activity", subjects: []string{"A-1001"}, want: [][]string{{"AR-1001"}, {"AH-1001"}}},
		{name: "both markets' activities", subjects: []string{"A-1001", "A-1002"},
			want: [][]string{{"AR-1001", "AR-1002"}, {"AH-1001", "AH-1002"}}},
		{name: "none", subjects: []string{}, want: [][]string{{}, {}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rounds, roundsErr := s.RoundsOfSubjects(t.Context(), tt.subjects)
			actions, actionsErr := s.ActionsOfSubjects(t.Context(), tt.subjects)
			if err := errors.Join(roundsErr, actionsErr); err != nil {
				t.Fatalf("RoundsOfSubjects(), ActionsOfSubjects() error: %v", err)
			}
			got := [][]string{{}, {}}
			for _, r := range rounds {
				got[0] = append(got[0], r.ID)
			}
			for _, a := range actions {
				got[1] = append(got[1], a.ID)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("rounds and entries of %v mismatch (-want +got):\n%s", tt.subjects, diff)
			}
		})
	}
}
