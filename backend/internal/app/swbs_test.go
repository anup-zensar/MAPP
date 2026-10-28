package app

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/history"
	"mapp/internal/platform/authz"
	"mapp/internal/stores/storestest"
	"mapp/internal/swbs"
)

func TestSwbsCandidates_030_FR007(t *testing.T) {
	t.Parallel()
	db, _ := seededStore(t)
	set := storestest.New(db)
	items := swbsLineItems{activities: set.Activities, lineItems: set.LineItems, settings: set.Settings}
	everything := authz.Scope{Everything: true, TeamIDs: []string{}}
	tests := []struct {
		name  string
		scope authz.Scope
		ids   []string
		want  []swbs.Candidate
	}{
		{name: "company codes and readiness", scope: everything, ids: []string{"L-1001", "L-1003", "L-1004", "L-1005", "L-1006"},
			want: []swbs.Candidate{
				{Request: swbs.Request{LineItemID: "L-1001", LineItemName: "Agency fee"}},
				{Request: swbs.Request{LineItemID: "L-1003", LineItemName: "Venue hire", CompanyCode: "DE03"}},
				{Request: swbs.Request{LineItemID: "L-1004", LineItemName: "Speaker fees", CompanyCode: "DE03"}},
				{Request: swbs.Request{LineItemID: "L-1005", LineItemName: "Media booking", CompanyCode: "NL02"}},
				{Request: swbs.Request{LineItemID: "L-1006", LineItemName: "Stand build", CompanyCode: "GB01"}, Ready: true},
			}},
		{name: "a scope that sees no activity", scope: authz.Scope{PersonID: "U-9999", TeamIDs: []string{}}, ids: nil,
			want: []swbs.Candidate{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []swbs.Candidate
			err := db.Read(t.Context(), func(ctx context.Context) error {
				var err error
				got, err = items.Candidates(ctx, tt.scope, tt.ids)
				return err
			})
			if err != nil {
				t.Fatalf("Candidates: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Candidates (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHistoryAccess_SwbsRun_030_FR010(t *testing.T) {
	t.Parallel()
	h := historyAccess{}
	e := history.Entry{RecordType: "swbsRun", RecordID: "SWR-1001"}
	importer, auditor, planner := withPermissions(authz.FinanceImport), withPermissions(authz.ActivityLogViewAll),
		withPermissions(authz.PlansView, authz.FinanceView)

	if !h.CanSeeEntry(importer, e) || !h.CanSeeEntry(auditor, e) || h.CanSeeEntry(planner, e) {
		t.Errorf("importer, auditor, planner see an SWBS run = %v, %v, %v; want true, true, false",
			h.CanSeeEntry(importer, e), h.CanSeeEntry(auditor, e), h.CanSeeEntry(planner, e))
	}
}
