package app

import (
	"context"
	"reflect"
	"testing"
	"time"

	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/people"
	"northstar/internal/plans"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/store"
	"northstar/internal/settings"
	"northstar/seed"
)

// The campaign and initiative levels add to a plan's facts for feature 004 its campaigns, initiatives and
// campaign budgets, and every campaign and initiative that blocks publishing, from the demo data: FY2026
// Lubricants Europe is complete, FY2026 Mobility China's Retail growth has no cluster or region and its
// Fleet cards no dates (005 data model §9; D-195).
func TestCampaignLevels_AddFacts_For004(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	now := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	if err := seed.Load(t.Context(), db, now); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	svc := campaigns.NewService(db, campaigns.NewJSONRepository(), generator,
		history.NewService(db, history.NewJSONRepository(), generator, now), now, campaigns.Deps{
			Plans:      campaignPlans{repo: plans.NewJSONRepository()},
			People:     people.NewJSONRepository(),
			Settings:   interimSettings{repo: settings.NewJSONRepository()},
			Activities: campaigns.NoActivities{},
		})
	planIDs := []string{"P-1001", "P-1002", "P-1003"}
	facts := map[string]*plans.Facts{}
	for _, id := range planIDs {
		facts[id] = &plans.Facts{}
	}

	err = db.Read(t.Context(), func(ctx context.Context) error { // 004 calls the levels inside its transaction
		for _, level := range campaignLevels(svc) {
			if err := level.AddFacts(ctx, planIDs, facts); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("AddFacts: %v", err)
	}

	type figures struct {
		Campaigns, Initiatives int
		CampaignBudgets        string
	}
	want := map[string]figures{
		"P-1001": {2, 4, "1150000.00"},
		"P-1002": {1, 2, "600000.00"},
		"P-1003": {1, 2, "300000.00"},
	}
	for id, w := range want {
		got := figures{facts[id].Campaigns, facts[id].Initiatives, facts[id].CampaignBudgets.String()}
		if got != w {
			t.Errorf("facts of %s = %+v, want %+v", id, got, w)
		}
	}
	if len(facts["P-1001"].NeedsAttention) != 0 || len(facts["P-1003"].NeedsAttention) != 0 {
		t.Errorf("complete plans need attention: %+v, %+v", facts["P-1001"].NeedsAttention, facts["P-1003"].NeedsAttention)
	}
	wantItems := []plans.AttentionItem{ // the seed's IDs follow data model §9's table order
		{Record: errs.RecordRef{Type: "campaign", ID: "C-1003", Name: "Retail growth"}, BlocksPublish: true},
		{Record: errs.RecordRef{Type: "initiative", ID: "I-1006", Name: "Fleet cards"}, BlocksPublish: true},
	}
	wantMissing := [][]string{{"Cluster", "Region"}, {"Dates"}}
	got := facts["P-1002"].NeedsAttention
	if len(got) != len(wantItems) {
		t.Fatalf("NeedsAttention of P-1002 = %+v, want Retail growth and Fleet cards", got)
	}
	for n, item := range got {
		var labels []string
		for _, m := range item.Missing {
			labels = append(labels, m.Label)
		}
		item.Missing = nil
		if !reflect.DeepEqual(item, wantItems[n]) || !reflect.DeepEqual(labels, wantMissing[n]) {
			t.Errorf("NeedsAttention[%d] = %+v missing %v, want %+v missing %v", n, item, labels, wantItems[n], wantMissing[n])
		}
	}
}
