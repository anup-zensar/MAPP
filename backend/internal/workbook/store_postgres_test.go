package workbook_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
	"mapp/internal/workbook"
)

// The imports of a plan are read only by people who see the plan, as a missing one otherwise.
func TestPostgresStore_ImportsInScope_012FR035(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	store := workbook.NewPostgresStore(db)
	at := time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	meta := record.NewMeta(actor.System, at)
	priya := actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	err := db.Write(t.Context(), func(ctx context.Context) error {
		directory, catalogue, plans := people.NewPostgresStore(db), settings.NewPostgresStore(db), plan.NewPostgresStore(db)
		err := errors.Join(
			catalogue.PutCurrency(ctx, settings.Currency{Code: "USD", Name: "US dollar", Active: true, Meta: meta}),
			catalogue.PutModel(ctx, settings.Model{ID: "MOD-1001", Name: "Leads", InqToMqlRate: money.PercentFromHundredths(200),
				MqlToDealRate: money.PercentFromHundredths(1000), AverageDealSize: money.USD(money.FromCents(100)), Meta: meta}))
		for _, n := range []string{"1", "2"} {
			err = errors.Join(err,
				directory.PutTeam(ctx, people.Team{ID: "T-100" + n, Name: "Team " + n, Colour: "blue", Meta: meta}),
				plans.Put(ctx, plan.Plan{ID: "P-100" + n, Name: "Plan " + n, Year: 2026, ModelID: "MOD-1001",
					TeamIDs: []string{"T-100" + n}, DefaultSpread: plan.SpreadEvenly, GoalStage: plan.StageINQ,
					Status: plan.StatusDraft, Meta: meta}),
				store.AddImport(ctx, workbook.Import{ID: "IMP-100" + n, PlanID: "P-100" + n, FileName: "plan.xlsx", ImportedAt: at,
					ImportedBy: priya, RowsRead: 2, RowsLeftOut: 1,
					Records:  []workbook.ImportRecord{{Level: workbook.LevelCampaign, RecordID: "C-100" + n, Name: "Spring", ParentID: "P-100" + n, Action: "created"}},
					Problems: []workbook.Problem{{Sheet: "Campaigns", Row: 3, Column: "Owner", Value: "x", Reason: "No active person has this e-mail"}}}))
		}
		return err
	})
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	team1 := authz.Scope{PersonID: "U-1002", Holds: true, TeamIDs: []string{"T-1001"}}

	listed, err := store.ListImports(t.Context(), team1, "P-1001", 10)
	got, found, getErr := store.GetImport(t.Context(), team1, "IMP-1001")
	_, foundOther, otherErr := store.GetImport(t.Context(), team1, "IMP-1002")
	listedOther, listErr := store.ListImports(t.Context(), team1, "P-1002", 10)

	if readErr := errors.Join(err, getErr, otherErr, listErr); readErr != nil {
		t.Fatal(readErr)
	}
	seen := []int{len(listed), len(got.Problems), len(got.Records), len(listedOther)}
	if diff := cmp.Diff([]int{1, 1, 1, 0}, seen); diff != "" || !found || foundOther {
		t.Errorf("listed, problems, records, listed elsewhere (-want +got):\n%s found %v, found elsewhere %v", diff, found, foundOther)
	}
}
