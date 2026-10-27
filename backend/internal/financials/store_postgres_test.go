package financials_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/activity"
	"mapp/internal/campaign"
	"mapp/internal/financials"
	"mapp/internal/lineitem"
	"mapp/internal/people"
	"mapp/internal/plan"
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
	pgMeta = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))
)

// pgStore gives a store whose database holds the line items L-1001 (plan P-1001, team T-1001) and L-1002
// (plan P-1002, team T-1002), the import FIM-1001 and four transactions: linked to each line item, not
// linked, and linked to a line item that is not stored.
func pgStore(t *testing.T) (*database.DB, *financials.PostgresStore, []financials.Transaction) {
	t.Helper()
	db := dbtest.NewDB(t)
	s := financials.NewPostgresStore(db)
	directory, catalogue := people.NewPostgresStore(db), settings.NewPostgresStore(db)
	planStore, containers := plan.NewPostgresStore(db), campaign.NewPostgresStore(db)
	activityStore, items := activity.NewPostgresStore(db), lineitem.NewPostgresStore(db)
	linked := pgTransaction("FT-1001", "L-1001")
	linked.MatchedBy, linked.ManualLink = financials.MatchedByManual, &financials.ManualLink{By: actor.System, At: pgMeta.CreatedAt}
	unlinked := pgTransaction("FT-1003", "")
	unlinked.Type, unlinked.AccrualMonth, unlinked.MatchedBy, unlinked.NotLinkedReason = financials.TypeAccrual, "2026-09", "", "No match"
	stored := []financials.Transaction{linked, pgTransaction("FT-1002", "L-1002"), unlinked, pgTransaction("FT-1004", "L-9999")}
	pgWrite(t, db, func(ctx context.Context) error {
		err := errors.Join(
			catalogue.PutCurrency(ctx, settings.Currency{Code: "USD", Name: "US dollar", Active: true, Meta: pgMeta}),
			catalogue.PutActivityType(ctx, settings.ActivityType{ID: "AT-1001", Name: "Webinar", Meta: pgMeta}),
			catalogue.PutModel(ctx, settings.Model{ID: "MOD-1001", Name: "Leads", ActivityTypeIDs: []string{"AT-1001"},
				InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(1000),
				AverageDealSize: money.USD(money.FromCents(100)), Meta: pgMeta}),
			s.PutImport(ctx, financials.Import{ID: "FIM-1001", Source: financials.SourceDemo, FeedName: "Demo data", Rows: 4,
				New: 4, NotLinked: 1, Meta: pgMeta}))
		for _, n := range []string{"1", "2"} {
			err = errors.Join(err,
				directory.PutTeam(ctx, people.Team{ID: "T-100" + n, Name: "Team " + n, Colour: "blue", Meta: pgMeta}),
				planStore.Put(ctx, plan.Plan{ID: "P-100" + n, Name: "Plan " + n, Year: 2026, ModelID: "MOD-1001",
					TeamIDs: []string{"T-100" + n}, DefaultSpread: plan.SpreadEvenly, GoalStage: plan.StageINQ,
					Status: plan.StatusDraft, Meta: pgMeta}),
				containers.PutCampaign(ctx, campaign.Campaign{ID: "C-100" + n, PlanID: "P-100" + n, Name: "Campaign", Meta: pgMeta}),
				containers.PutInitiative(ctx, campaign.Initiative{ID: "I-100" + n, CampaignID: "C-100" + n, Name: "Initiative",
					Meta: pgMeta}),
				activityStore.Put(ctx, activity.Activity{ID: "A-100" + n, InitiativeID: "I-100" + n, Name: "Activity",
					ActivityTypeID: "AT-1001", Status: activity.StatusCreated, Meta: pgMeta}),
				items.Put(ctx, lineitem.LineItem{ID: "L-100" + n, ActivityID: "A-100" + n, Name: "Item", Currency: "USD",
					Spread: lineitem.Spread{Method: lineitem.SpreadEvenly}, Meta: pgMeta}))
		}
		for _, transaction := range stored {
			err = errors.Join(err, s.PutTransaction(ctx, transaction))
		}
		return err
	})
	return db, s, stored
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgTransaction(id, lineItemID string) financials.Transaction {
	return financials.Transaction{ID: id, Fields: financials.Fields{Type: financials.TypeInvoice, DocumentNumber: "51" + id[3:],
		LineNumber: "1", PONumber: "45001", POLineNumber: "10", PostingDate: date.MustParse("2026-03-31"),
		Amount: money.FromCents(-12050), Currency: "EUR", VendorCode: "V-1", VendorName: "Agency", Description: "Credit",
		SwbsCode: "SWBS-1", TrackingKey: "HX1001", SentLineItemID: lineItemID},
		LineItemID: lineItemID, MatchedBy: financials.MatchedByLineItemID, ImportID: "FIM-1001", Meta: pgMeta}
}

func TestPostgresStoreTransactionScopeIsolationFR009(t *testing.T) {
	t.Parallel()
	_, s, _ := pgStore(t)
	inTeam := func(teamID string) authz.Scope {
		return authz.Scope{PersonID: "U-1001", Holds: true, TeamIDs: []string{teamID}}
	}
	none := authz.Scope{PersonID: "U-1001", TeamIDs: []string{}}
	everything := authz.Scope{Everything: true, TeamIDs: []string{}}
	tests := []struct {
		name            string
		plans, finance  authz.Scope
		canImport       bool
		wantTransaction []string
	}{
		{name: "an internal call", plans: everything, finance: everything, canImport: true,
			wantTransaction: []string{"FT-1001", "FT-1002", "FT-1003", "FT-1004"}},
		{name: "View plans and finance.view in team T-1001", plans: inTeam("T-1001"), finance: inTeam("T-1001"),
			wantTransaction: []string{"FT-1001"}},
		{name: "View plans in T-1002 but finance.view in T-1001", plans: inTeam("T-1002"), finance: inTeam("T-1001"),
			wantTransaction: []string{}},
		{name: "allowed to import, no finance.view", plans: none, finance: none, canImport: true,
			wantTransaction: []string{"FT-1003"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			visible, err := s.TransactionsInScope(t.Context(), tt.plans, tt.finance, tt.canImport)
			if err != nil {
				t.Fatalf("TransactionsInScope() error: %v", err)
			}
			ids := []string{}
			for _, transaction := range visible {
				ids = append(ids, transaction.ID)
			}
			if diff := cmp.Diff(tt.wantTransaction, ids); diff != "" {
				t.Errorf("TransactionsInScope() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
