package purchaserequest_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/approval"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/record"
	"mapp/internal/purchaserequest"
	"mapp/internal/stores"
	"mapp/seed"
)

var seededAt = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// seededDB is the demo data on PostgreSQL, with Mei named to approve PR-1001 (on the UK plan of Lubricants Europe).
func seededDB(t *testing.T) *purchaserequest.PostgresStore {
	t.Helper()
	db := dbtest.NewFilledDB(t, "pr032", func(db *database.DB) {
		if err := seed.Load(context.Background(), stores.Postgres(db), func() time.Time { return seededAt }); err != nil {
			t.Fatalf("seed: %v", err)
		}
	})
	round := approval.Round{ID: "AR-1999", SubjectKind: approval.KindPurchaseRequest, SubjectID: "PR-1001", Number: 1,
		WorkflowID: "WF-1004", WorkflowName: "Purchase request approval (MoA)", CurrentLevel: 1, Outcome: approval.OutcomeOpen,
		Levels: []approval.RoundLevel{{Number: 1, Name: "MoA approver", Approvers: []actor.Actor{{ID: "U-1004", Name: "Mei Lin"}},
			ApproverType: approval.SourceRecordField, Mode: approval.ModeAny, Resolution: approval.ResolutionDecide,
			Excluded: []approval.Excluded{}, StartedAt: &seededAt}},
		SubmittedAt: seededAt, SubmittedBy: actor.Actor{ID: "U-1002", Name: "Priya Raman"},
		Facts: &approval.RuleFacts{BusinessUnits: []approval.FactOption{}}, Meta: record.NewMeta(actor.System, seededAt)}
	if err := db.Write(t.Context(), func(ctx context.Context) error { return approval.NewPostgresStore(db).PutRound(ctx, round) }); err != nil {
		t.Fatalf("round: %v", err)
	}
	return purchaserequest.NewPostgresStore(db)
}

func scopeOf(person string, holds bool, teams ...string) authz.Scope {
	return authz.Scope{PersonID: person, Holds: holds, TeamIDs: append([]string{}, teams...)}
}

func TestPostgresStore_ScopeIsolation_032_FR050(t *testing.T) {
	t.Parallel()
	store := seededDB(t)
	nobody := scopeOf("U-9999", false)
	tests := []struct {
		name  string
		scope purchaserequest.ViewScope
		want  []string
	}{
		{name: "the plan's team with plans.view", scope: purchaserequest.ViewScope{Plans: scopeOf("U-1006", true, "T-1001"), List: nobody},
			want: []string{"PR-1001"}},
		{name: "another team", scope: purchaserequest.ViewScope{Plans: scopeOf("U-1006", true, "T-1002"), List: nobody}, want: []string{}},
		{name: "the owner without a team", scope: purchaserequest.ViewScope{Plans: scopeOf("U-1002", false), List: scopeOf("U-1002", false)},
			want: []string{"PR-1001"}},
		{name: "the MoA approver", scope: purchaserequest.ViewScope{Plans: scopeOf("U-1003", false), List: scopeOf("U-1003", false)},
			want: []string{"PR-1001"}},
		{name: "the Purchase requests page's scope", scope: purchaserequest.ViewScope{Plans: scopeOf("U-1007", false),
			List: scopeOf("U-1007", true, "T-1001")}, want: []string{"PR-1001"}},
		{name: "an approver named in a round", scope: purchaserequest.ViewScope{Plans: scopeOf("U-1004", true, "T-1002"),
			List: scopeOf("U-1004", false)}, want: []string{"PR-1001"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			listed, err := store.ListInScope(t.Context(), tt.scope, []string{"PR-1001", "PR-9999"})
			if err != nil {
				t.Fatalf("ListInScope: %v", err)
			}
			_, found, err := store.GetInScope(t.Context(), tt.scope, "PR-1001")
			if err != nil {
				t.Fatalf("GetInScope: %v", err)
			}
			on, err := store.OnLineItems(t.Context(), tt.scope, []string{"L-1010"})
			if err != nil {
				t.Fatalf("OnLineItems: %v", err)
			}
			if diff := cmp.Diff(tt.want, ids(listed)); diff != "" || found != (len(tt.want) == 1) || len(on) != len(tt.want) {
				t.Errorf("ListInScope (-want +got):\n%s found %v, on line items %d", diff, found, len(on))
			}
		})
	}
}

func TestPostgresStore_PageScopeIsolation_032_FR050(t *testing.T) {
	t.Parallel()
	store := seededDB(t)
	q := purchaserequest.ListQuery{Sort: "id", Page: 1, PageSize: 50}
	tests := []struct {
		name  string
		scope authz.Scope
		want  []string
	}{
		{name: "purchaseRequests.list in the plan's team", scope: scopeOf("U-1007", true, "T-1001"), want: []string{"PR-1001"}},
		{name: "in another team", scope: scopeOf("U-1007", true, "T-1002"), want: []string{}},
		{name: "the owner without the permission", scope: scopeOf("U-1002", false), want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page, err := store.Page(t.Context(), tt.scope, q)
			if err != nil {
				t.Fatalf("Page: %v", err)
			}
			if diff := cmp.Diff(tt.want, ids(page.Items)); diff != "" || page.Total != len(tt.want) {
				t.Errorf("Page (-want +got):\n%s total %d", diff, page.Total)
			}
		})
	}
}

func ids(list []purchaserequest.PurchaseRequest) []string {
	got := []string{}
	for _, pr := range list {
		got = append(got, pr.ID)
	}
	return got
}
