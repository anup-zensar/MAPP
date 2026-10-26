package settings_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
)

var (
	savedAt = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	meta    = record.NewMeta(actor.System, savedAt)
)

// Every Settings record is saved in its own table and read back exactly; lists are ordered by id and the
// "…Of" lookups return only the records of their parent (data model §2–§8).
func TestJSONRepository_RoundTrip(t *testing.T) {
	db := openStore(t)
	repo := settings.NewJSONRepository()
	cost := money.USD(money.FromCents(2_500_000))
	model := settings.Model{
		ID: "MOD-1001", Name: "Lubricants 1.0",
		InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(500),
		AverageDealSize: money.USD(money.FromCents(1_000_000)), ActivityTypeIDs: []string{"AT-1001", "AT-1002"},
		Active: true, Meta: meta,
	}
	activityType := settings.ActivityType{ID: "AT-1002", Name: "Event", DefaultPlannedCost: &cost, Active: true, Meta: meta}
	region := settings.Attribute{
		ID: "SA-1004", Label: "Region", Level: settings.LevelCampaign, Type: settings.TypeMultipleChoice,
		Marking: settings.MarkingNeededToPublish, Section: settings.DefaultSection, Position: 2,
		ParentAttributeID: "SA-1003", Condition: []string{}, Active: true, Meta: meta,
	}
	options := []settings.Option{
		{ID: "SO-1010", AttributeID: "SA-1003", Label: "Lubricants – Europe", Position: 1, ParentOptionIDs: []string{}, Active: true, Meta: meta},
		{ID: "SO-1020", AttributeID: "SA-1004", Label: "LUBES – DACH", Position: 1, ParentOptionIDs: []string{"SO-1010"}, Active: true, Meta: meta},
	}
	euro := settings.Currency{
		Code: "EUR", Name: "Euro", Symbol: "€", Rate: money.RateFromMicros(920_000),
		RateChangedAt: savedAt, RateChangedBy: actor.System, Active: true, Meta: meta,
	}
	budget := settings.FinanceBudget{ID: "FB-1001", Year: 2026, Name: "2026 Lubricants Budget", Meta: meta}
	items := []settings.FinanceBudgetItem{
		{ID: "FBI-1001", BudgetID: "FB-1001", Name: "Europe – Digital", Code: "LUB-EU-DIG", Active: true, Meta: meta},
		{ID: "FBI-1004", BudgetID: "FB-1002", Name: "China – Retail", Code: "MOB-CN-RET", Active: true, Meta: meta},
	}

	write(t, db, func(ctx context.Context) error {
		for _, err := range []error{
			repo.PutModel(ctx, model), repo.PutActivityType(ctx, activityType), repo.PutAttribute(ctx, region),
			repo.PutOption(ctx, options[0]), repo.PutOption(ctx, options[1]), repo.PutCurrency(ctx, euro),
			repo.PutFinanceBudget(ctx, budget), repo.PutFinanceBudgetItem(ctx, items[0]), repo.PutFinanceBudgetItem(ctx, items[1]),
		} {
			if err != nil {
				return err
			}
		}
		return nil
	})

	read(t, db, func(ctx context.Context) {
		gotModel, found, err := repo.GetModel(ctx, "MOD-1001")
		check(t, "GetModel", found, err, gotModel, model)
		gotType, found, err := repo.GetActivityType(ctx, "AT-1002")
		check(t, "GetActivityType", found, err, gotType, activityType)
		gotAttribute, found, err := repo.GetAttribute(ctx, "SA-1004")
		check(t, "GetAttribute", found, err, gotAttribute, region)
		gotOption, found, err := repo.GetOption(ctx, "SO-1020")
		check(t, "GetOption", found, err, gotOption, options[1])
		gotCurrency, found, err := repo.GetCurrency(ctx, "EUR")
		check(t, "GetCurrency", found, err, gotCurrency, euro)
		gotBudget, found, err := repo.GetFinanceBudget(ctx, "FB-1001")
		check(t, "GetFinanceBudget", found, err, gotBudget, budget)
		gotItem, found, err := repo.GetFinanceBudgetItem(ctx, "FBI-1001")
		check(t, "GetFinanceBudgetItem", found, err, gotItem, items[0])

		if _, found, err := repo.GetModel(ctx, "MOD-9999"); found || err != nil {
			t.Errorf("GetModel(unknown) = %v, %v; want not found", found, err)
		}
		of, err := repo.OptionsOf(ctx, "SA-1004")
		if err != nil || !reflect.DeepEqual(of, options[1:]) {
			t.Errorf("OptionsOf(SA-1004) = %+v, %v", of, err)
		}
		none, err := repo.OptionsOf(ctx, "SA-9999")
		if err != nil || none == nil || len(none) != 0 {
			t.Errorf("OptionsOf(unknown) = %#v, %v; want an empty list", none, err)
		}
		budgetItems, err := repo.ItemsOf(ctx, "FB-1001")
		if err != nil || !reflect.DeepEqual(budgetItems, items[:1]) {
			t.Errorf("ItemsOf(FB-1001) = %+v, %v", budgetItems, err)
		}
		allOptions, err := repo.ListOptions(ctx)
		if err != nil || len(allOptions) != 2 {
			t.Errorf("ListOptions = %d, %v", len(allOptions), err)
		}
	})

	write(t, db, func(ctx context.Context) error {
		if err := repo.DeleteOption(ctx, "SO-1020"); err != nil {
			return err
		}
		return repo.DeleteCurrency(ctx, "EUR")
	})
	read(t, db, func(ctx context.Context) {
		if _, found, _ := repo.GetOption(ctx, "SO-1020"); found {
			t.Error("the deleted option is still stored")
		}
		if currencies, _ := repo.ListCurrencies(ctx); len(currencies) != 0 {
			t.Errorf("currencies after the delete = %+v", currencies)
		}
	})
}

// Money is stored as exact decimal strings and attribute values keep their JSON exactly (D-010, D-048).
func TestJSONShapes_MoneyAndValues_D048(t *testing.T) {
	encoded, err := json.Marshal(settings.ActivityType{ID: "AT-1001", Name: "Agency fees – Creative", Active: true, Meta: meta})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["defaultPlannedCost"] != nil || fields["version"] != float64(1) {
		t.Errorf("activity type JSON = %s", encoded)
	}

	values := settings.AttributeValues{
		"SA-1003": json.RawMessage(`["SO-1010","SO-1011"]`),
		"SA-1011": json.RawMessage(`[{"optionId":"SO-1050","share":"60.00"}]`),
		"SA-1005": json.RawMessage(`true`),
	}
	again, err := json.Marshal(values)
	if err != nil || string(again) != `{"SA-1003":["SO-1010","SO-1011"],"SA-1005":true,"SA-1011":[{"optionId":"SO-1050","share":"60.00"}]}` {
		t.Errorf("values JSON = %s, %v", again, err)
	}
	var shares []settings.SplitShare
	if err := json.Unmarshal(values["SA-1011"], &shares); err != nil || shares[0].Share.String() != "60.00" {
		t.Errorf("split shares = %+v, %v", shares, err)
	}
	if !(settings.Attribute{Type: settings.TypeMultipleChoiceSplit}).IsChoice() || settings.TypeYesNo.IsChoice() {
		t.Error("IsChoice is wrong")
	}
	if !(settings.Model{ActivityTypeIDs: []string{"AT-1001"}}).Allows("AT-1001") || (settings.Model{}).Allows("AT-1001") {
		t.Error("Allows is wrong")
	}
}

// An update's money field tells "left out", "null" and an amount apart (D-203).
func TestOptionalMoney_LeftOutNullOrAmount_D203(t *testing.T) {
	var update settings.ActivityTypeUpdate
	if err := json.Unmarshal([]byte(`{"version":3}`), &update); err != nil || update.DefaultPlannedCost.Set {
		t.Errorf("left out: %+v, %v", update.DefaultPlannedCost, err)
	}
	update = settings.ActivityTypeUpdate{}
	if err := json.Unmarshal([]byte(`{"defaultPlannedCost":null,"version":3}`), &update); err != nil ||
		!update.DefaultPlannedCost.Set || update.DefaultPlannedCost.Value != nil {
		t.Errorf("null: %+v, %v", update.DefaultPlannedCost, err)
	}
	update = settings.ActivityTypeUpdate{}
	err := json.Unmarshal([]byte(`{"defaultPlannedCost":{"amount":"25000","currency":"USD"},"version":3}`), &update)
	if err != nil || !update.DefaultPlannedCost.Set || *update.DefaultPlannedCost.Value != (money.Input{Amount: "25000", Currency: "USD"}) {
		t.Errorf("amount: %+v, %v", update.DefaultPlannedCost, err)
	}
}

func openStore(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	return db
}

func write(t *testing.T, db *store.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func read(t *testing.T, db *store.DB, fn func(ctx context.Context)) {
	t.Helper()
	err := db.Read(t.Context(), func(ctx context.Context) error {
		fn(ctx)
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

func check(t *testing.T, what string, found bool, err error, got, want any) {
	t.Helper()
	if err != nil || !found || !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %+v, %v, %v; want %+v", what, got, found, err, want)
	}
}
