package campaigns_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"mapp/internal/campaigns"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// Campaigns and initiatives are saved in their own tables and read back exactly — dates as YYYY-MM-DD,
// budgets as USD money, no dates or owner as null — and found by their parents (data model §1–§3).
func TestJSONRepository_RoundTrip(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := campaigns.NewJSONRepository()
	meta := record.NewMeta(actor.System, time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC))
	priya := "U-1002"
	start, end := date.MustParse("2026-01-01"), date.MustParse("2026-12-31")
	budget := money.USD(money.FromCents(70_000_000))
	helix := campaigns.Campaign{
		ID: "C-1001", PlanID: "P-1001", Name: "Helix launch", OwnerID: &priya, StartDate: &start, EndDate: &end,
		Budget: &budget, Attributes: settings.AttributeValues{"SA-1005": json.RawMessage(`true`)}, BudgetPhasing: rollup.NotPhased, Meta: meta,
	}
	retail := campaigns.Campaign{ID: "C-1003", PlanID: "P-1002", Name: "Retail growth", Attributes: settings.AttributeValues{},
		BudgetPhasing: rollup.NotPhased, Meta: meta}
	digital := campaigns.Initiative{ID: "I-1001", CampaignID: "C-1001", Name: "Digital & social", OwnerID: &priya, Attributes: settings.AttributeValues{},
		BudgetPhasing: rollup.NotPhased, Meta: meta}
	fleet := campaigns.Initiative{ID: "I-1006", CampaignID: "C-1003", Name: "Fleet cards", Attributes: settings.AttributeValues{},
		BudgetPhasing: rollup.NotPhased, Meta: meta}

	err = db.Write(t.Context(), func(ctx context.Context) error {
		for _, err := range []error{
			repo.PutCampaign(ctx, helix), repo.PutCampaign(ctx, retail), repo.PutInitiative(ctx, digital), repo.PutInitiative(ctx, fleet),
		} {
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = db.Read(t.Context(), func(ctx context.Context) error {
		if got, found, err := repo.GetCampaign(ctx, "C-1001"); err != nil || !found || !reflect.DeepEqual(got, helix) {
			t.Errorf("GetCampaign = %+v, %v, %v", got, found, err)
		}
		raw, _, _ := store.Get[map[string]json.RawMessage](ctx, "campaigns", "C-1003")
		if string(raw["startDate"]) != "null" || string(raw["ownerId"]) != "null" || string(raw["budget"]) != "null" {
			t.Errorf("an empty campaign is stored as %s / %s / %s, want nulls", raw["startDate"], raw["ownerId"], raw["budget"])
		}
		raw, _, _ = store.Get[map[string]json.RawMessage](ctx, "campaigns", "C-1001")
		if string(raw["startDate"]) != `"2026-01-01"` || string(raw["budget"]) != `{"amount":"700000.00","currency":"USD"}` {
			t.Errorf("stored dates and budget = %s, %s", raw["startDate"], raw["budget"])
		}
		if ofPlan, err := repo.CampaignsOfPlan(ctx, "P-1002"); err != nil || !reflect.DeepEqual(ofPlan, []campaigns.Campaign{retail}) {
			t.Errorf("CampaignsOfPlan = %+v, %v", ofPlan, err)
		}
		if ofPlans, err := repo.CampaignsOfPlans(ctx, []string{"P-1001", "P-1002"}); err != nil || len(ofPlans) != 2 {
			t.Errorf("CampaignsOfPlans = %d, %v", len(ofPlans), err)
		}
		if got, found, err := repo.GetInitiative(ctx, "I-1001"); err != nil || !found || !reflect.DeepEqual(got, digital) {
			t.Errorf("GetInitiative = %+v, %v, %v", got, found, err)
		}
		if of, err := repo.InitiativesOfCampaign(ctx, "C-1003"); err != nil || !reflect.DeepEqual(of, []campaigns.Initiative{fleet}) {
			t.Errorf("InitiativesOfCampaign = %+v, %v", of, err)
		}
		if none, err := repo.InitiativesOfCampaigns(ctx, nil); err != nil || none == nil || len(none) != 0 {
			t.Errorf("InitiativesOfCampaigns(none) = %#v, %v; want an empty list", none, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	err = db.Write(t.Context(), func(ctx context.Context) error {
		if err := repo.DeleteInitiative(ctx, "I-1006"); err != nil {
			return err
		}
		return repo.DeleteCampaign(ctx, "C-1003")
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = db.Read(t.Context(), func(ctx context.Context) error {
		left, _ := repo.ListCampaigns(ctx)
		initiatives, _ := repo.ListInitiatives(ctx)
		if len(left) != 1 || len(initiatives) != 1 {
			t.Errorf("after the deletes: %d campaigns, %d initiatives; want 1 and 1", len(left), len(initiatives))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// Without feature 006 there are no activities to roll up or copy.
func TestNoActivities_NothingBelow(t *testing.T) {
	var activities campaigns.Activities = campaigns.NoActivities{}
	summaries, err := activities.Summaries(t.Context(), []string{"I-1001"})
	if err != nil || summaries == nil || len(summaries) != 0 {
		t.Errorf("Summaries = %#v, %v", summaries, err)
	}
	counts, err := activities.CopyAll(t.Context(), actor.System, "I-1001", "I-1009", 0)
	if err != nil || counts != (campaigns.ActivityCopyCounts{}) {
		t.Errorf("CopyAll = %+v, %v", counts, err)
	}
}

// A campaign or initiative stored before 016 reads as not phased (016 data-model §4.5, T012).
func TestJSONRepository_PhasingOfOlderRows(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := campaigns.NewJSONRepository()
	err = db.Write(t.Context(), func(ctx context.Context) error {
		if err := store.Put(ctx, "campaigns", "C-1001", map[string]any{"id": "C-1001", "planId": "P-1001", "name": "Old"}); err != nil {
			return err
		}
		return store.Put(ctx, "initiatives", "I-1001", map[string]any{"id": "I-1001", "campaignId": "C-1001", "name": "Old"})
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = db.Read(t.Context(), func(ctx context.Context) error {
		c, _, err := repo.GetCampaign(ctx, "C-1001")
		if err != nil || c.BudgetPhasing.Method != rollup.MethodNone || c.BudgetPhasing.Months == nil {
			t.Errorf("campaign phasing = %+v, %v", c.BudgetPhasing, err)
		}
		list, err := repo.ListInitiatives(ctx)
		if err != nil || len(list) != 1 || list[0].BudgetPhasing.Method != rollup.MethodNone {
			t.Errorf("initiatives = %+v, %v", list, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}
