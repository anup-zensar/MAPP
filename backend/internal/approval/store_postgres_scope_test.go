package approval_test

import (
	"context"
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
