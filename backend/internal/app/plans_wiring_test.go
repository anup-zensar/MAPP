package app

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/plans"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
	"mapp/internal/stores"
	"mapp/seed"
)

// Plans reads Settings and people and teams through 002's and 003's services, which app.New passes to
// plans.NewService as its plans.Settings and plans.Directory. Here on the demo data: Lubricants 1.0 with
// its rates and deal size, Priya's team, the Business unit options and US dollars as her display currency
// (004 T030).
func TestPlansDependencies_ReadTheDemoData(t *testing.T) {
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
	var s plans.Settings = settings.NewService(history.Tools{Tx: db, IDs: generator, Log: historyService, Now: now}, settings.NewJSONStore())
	var d plans.Directory = people.NewService(history.Tools{Tx: db, IDs: generator, Log: historyService, Now: now}, people.NewJSONStore())

	err = db.Read(t.Context(), func(ctx context.Context) error {
		model, found, err := s.Model(ctx, "MOD-1001")
		if err != nil || !found || model.Name != "Lubricants 1.0" || model.InqToMqlRate.String() != "2.00" ||
			model.MqlToDealRate.String() != "5.00" || model.AverageDealSize.Format() != "10,000.00 USD" || !model.Active {
			t.Errorf("Model(MOD-1001) = %+v, %v, %v; want Lubricants 1.0, 2.00 %%, 5.00 %%, 10,000.00 USD, active", model, found, err)
		}
		teams, err := d.TeamIDsOf(ctx, "U-1002")
		if err != nil || !reflect.DeepEqual(teams, []string{"T-1001"}) {
			t.Errorf("TeamIDsOf(Priya) = %v, %v; want Lubricants Europe", teams, err)
		}
		unit, err := s.BusinessUnit(ctx, settings.AttributeValues{"SA-1001": json.RawMessage(`["SO-1002"]`)})
		if err != nil || !reflect.DeepEqual(unit, []string{"SO-1002"}) {
			t.Errorf("BusinessUnit = %v, %v; want Lubricants", unit, err)
		}
		option, found, err := s.Option(ctx, "SO-1002")
		if err != nil || !found || option.Label != "Lubricants" {
			t.Errorf("Option(SO-1002) = %+v, %v, %v; want Lubricants", option, found, err)
		}
		missing, err := s.MissingAttributes(ctx, settings.LevelPlan, settings.AttributeValues{}, nil)
		if err != nil || len(missing) != 1 || missing[0].Label != "Business unit" {
			t.Errorf("MissingAttributes(no values) = %+v, %v; want Business unit", missing, err)
		}
		rate, err := d.DisplayRateOf(ctx, "U-1002")
		if err != nil || rate != (people.DisplayRate{Code: "USD", Rate: "1"}) {
			t.Errorf("DisplayRateOf(Priya) = %+v, %v; want USD at 1", rate, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}
