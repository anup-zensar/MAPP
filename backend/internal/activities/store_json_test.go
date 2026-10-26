package activities_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"mapp/internal/activities"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
)

// An activity is saved in the table "activities" and read back exactly — dates, USD money, goals, Shell
// attribute values — and the activities of initiatives are found (data model §1).
func TestJSONRepository_RoundTrip(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := activities.NewJSONRepository()
	meta := record.NewMeta(actor.System, time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC))
	start, end := date.MustParse("2026-03-01"), date.MustParse("2026-05-31")
	cost := money.USD(money.FromCents(12_000_000))
	inq := int64(250_000)
	film := activities.Activity{
		ID: "A-1001", InitiativeID: "I-1001", Name: "Launch film", ActivityTypeID: "AT-1004", OwnerID: "U-1002",
		StartDate: &start, EndDate: &end, PlannedCost: &cost, ApprovalWorkflowID: "WF-1001", INQ: &inq,
		Attributes: settings.AttributeValues{"SA-1011": json.RawMessage(`[{"optionId":"SO-1050","share":"50.00"}]`)},
		Status:     activities.StatusCreated, Meta: meta,
	}
	fund := activities.Activity{
		ID: "A-1007", InitiativeID: "I-1003", Name: "Partner co-marketing fund", ActivityTypeID: "AT-1005",
		StartDate: &start, Ongoing: true, Attributes: settings.AttributeValues{}, Status: activities.StatusApproved, Meta: meta,
	}
	promo := activities.Activity{
		ID: "A-1010", InitiativeID: "I-1005", Name: "Lunar new year promo", ActivityTypeID: "AT-1003",
		Attributes: settings.AttributeValues{}, Status: activities.StatusCreated, Meta: meta,
	}

	err = db.Write(t.Context(), func(ctx context.Context) error {
		for _, a := range []activities.Activity{film, fund, promo} {
			if err := repo.Put(ctx, a); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = db.Read(t.Context(), func(ctx context.Context) error {
		if got, found, err := repo.Get(ctx, "A-1001"); err != nil || !found || !reflect.DeepEqual(got, film) {
			t.Errorf("Get = %+v, %v, %v", got, found, err)
		}
		raw, _, _ := store.Get[map[string]json.RawMessage](ctx, "activities", "A-1010")
		if string(raw["startDate"]) != "null" || string(raw["plannedCost"]) != "null" || string(raw["ownerId"]) != `""` {
			t.Errorf("an empty activity is stored as %s / %s / %s", raw["startDate"], raw["plannedCost"], raw["ownerId"])
		}
		if of, err := repo.ListByInitiative(ctx, "I-1003"); err != nil || !reflect.DeepEqual(of, []activities.Activity{fund}) {
			t.Errorf("ListByInitiative = %+v, %v", of, err)
		}
		of, err := repo.ListByInitiatives(ctx, []string{"I-1001", "I-1005"})
		if err != nil || len(of) != 2 || of[0].ID != "A-1001" || of[1].ID != "A-1010" {
			t.Errorf("ListByInitiatives = %+v, %v", of, err)
		}
		if none, err := repo.ListByInitiative(ctx, "I-9999"); err != nil || none == nil || len(none) != 0 {
			t.Errorf("ListByInitiative(unknown) = %#v, %v; want an empty list", none, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	err = db.Write(t.Context(), func(ctx context.Context) error { return repo.Delete(ctx, "A-1010") })
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = db.Read(t.Context(), func(ctx context.Context) error {
		if all, _ := repo.List(ctx); len(all) != 2 {
			t.Errorf("%d activities after the delete, want 2", len(all))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}
