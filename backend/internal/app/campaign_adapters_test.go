package app

import (
	"context"
	"reflect"
	"testing"
	"time"

	"mapp/internal/campaign"
	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
	"mapp/internal/stores"
	"mapp/seed"
)

// Lubricants Europe is complete; Mobility China's Retail growth has no cluster or region and Fleet cards no dates.
func TestCampaignLevels_AddFacts_For004(t *testing.T) {
	t.Parallel()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	now := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	if err = seed.Load(t.Context(), stores.JSON(db), now); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	historyService := history.NewService(db, history.NewJSONStore(), generator, now)
	svc := campaign.NewService(history.Tools{Tx: db, IDs: generator, Log: historyService, Now: now}, campaign.NewJSONStore(), campaign.Deps{
		Plans:      campaignPlans{repo: plan.NewJSONStore()},
		People:     people.NewJSONStore(),
		Settings:   settings.NewService(history.Tools{Tx: db, IDs: generator, Log: historyService, Now: now}, settings.NewJSONStore()),
		Activities: campaign.NoActivities{},
	})
	planIDs := []string{"P-1001", "P-1002", "P-1003"}
	facts := map[string]*plan.Facts{}
	for _, id := range planIDs {
		facts[id] = &plan.Facts{}
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
	wantItems := []plan.AttentionItem{ // the seed's IDs follow data model §9's table order
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
