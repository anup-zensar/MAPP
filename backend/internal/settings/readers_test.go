package settings

import (
	"context"
	"slices"
	"testing"

	"mapp/internal/platform/money"
)

// Later features read Settings through the readers inside their own transaction: lists come sorted as
// the Settings pages show them, and records in the recycle bin are never returned (FR-007, plan
// "Readers").
func TestReaders_SortedWithoutBinRecords_FR007(t *testing.T) {
	f := newTestService(t)
	webinar := f.addType(t, "webinar", true)
	event := f.addType(t, "Event", false)
	f.addModel(t, "Zeta", event.ID)
	alpha := f.addModel(t, "alpha", event.ID)
	gone := f.addModel(t, "Gone", event.ID)
	f.addCurrency(t, "EUR", "0.92", true)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "CNY", "7.10", false)
	tf := f.addAttributes(t,
		testAttribute{level: LevelActivity, label: "Brands", typ: TypeMultipleChoice, section: "Lubricants attributes", options: []string{"Rimula", "Helix"}},
		testAttribute{level: LevelActivity, label: "Channel", typ: TypeMultipleChoice},
		testAttribute{level: LevelCampaign, label: "Cluster", typ: TypeMultipleChoice},
	)
	europe := f.addItem(t, 2026, "B budget", "Europe", "", true)
	asia := f.addItem(t, 2026, "B budget", "asia", "", true)
	other := f.addItem(t, 2026, "A budget", "Other", "", true)
	f.addItem(t, 2027, "Next budget", "Next", "", true)
	f.write(t, func(ctx context.Context, _ Repository) error {
		_, err := f.history.MoveToBin(ctx, alex, modelKind, gone.ID)
		return err
	})

	f.write(t, func(ctx context.Context, _ Repository) error { // a caller's Write: the readers join it
		models, err := f.service.Models(ctx)
		if err != nil {
			return err
		}
		if names := namesOfModels(models); !slices.Equal(names, []string{"alpha", "Zeta"}) {
			t.Errorf("Models = %q, want alpha and Zeta by name, without the deleted one", names)
		}
		if _, found, _ := f.service.Model(ctx, gone.ID); found {
			t.Error("Model found the model in the recycle bin")
		}
		if m, found, _ := f.service.Model(ctx, alpha.ID); !found || m.Name != "alpha" || !m.Allows(event.ID) || m.Allows(webinar.ID) {
			t.Errorf("Model(%s) = %+v, %v", alpha.ID, m, found)
		}
		types, err := f.service.ActivityTypes(ctx)
		if err != nil || len(types) != 2 || types[0].Name != "Event" || types[1].Name != "webinar" {
			t.Errorf("ActivityTypes = %+v, %v; want Event, webinar", types, err)
		}
		if at, found, _ := f.service.ActivityType(ctx, webinar.ID); !found || at.Name != "webinar" {
			t.Errorf("ActivityType = %+v, %v", at, found)
		}
		attributes, err := f.service.Attributes(ctx, LevelActivity)
		if err != nil || len(attributes) != 2 || attributes[0].Label != "Brands" || attributes[1].Label != "Channel" {
			t.Errorf("Attributes(activity) = %+v, %v; want Brands, Channel", attributes, err)
		}
		options, err := f.service.Options(ctx, tf.attribute(LevelActivity, "Brands").ID)
		if err != nil || len(options) != 2 || options[0].Label != "Rimula" || options[1].Label != "Helix" {
			t.Errorf("Options = %+v, %v; want Rimula, Helix in their order", options, err)
		}
		if o, found, _ := f.service.Option(ctx, tf.option(LevelActivity, "Brands", "Helix")); !found || o.Label != "Helix" {
			t.Errorf("Option = %+v, %v", o, found)
		}
		currencies, err := f.service.Currencies(ctx)
		if err != nil || len(currencies) != 3 || currencies[0].Code != "USD" || currencies[1].Code != "CNY" || currencies[2].Code != "EUR" {
			t.Errorf("Currencies = %+v, %v; want USD, CNY, EUR", currencies, err)
		}
		if c, found, _ := f.service.Currency(ctx, "CNY"); !found || c.Active {
			t.Errorf("Currency(CNY) = %+v, %v; want the inactive CNY", c, found)
		}
		rates, err := f.service.Rates(ctx)
		if err != nil || len(rates) != 3 || rates["CNY"] != money.RateFromMicros(7_100_000) {
			t.Errorf("Rates = %v, %v; want the rate of every currency, inactive ones too", rates, err)
		}
		budgets, err := f.service.FinanceBudgetsOfYear(ctx, 2026)
		if err != nil || len(budgets) != 2 || budgets[0].ID != other.BudgetID || budgets[1].ID != europe.BudgetID {
			t.Errorf("FinanceBudgetsOfYear(2026) = %+v, %v; want A budget, B budget", budgets, err)
		}
		if b, found, _ := f.service.FinanceBudget(ctx, europe.BudgetID); !found || b.Name != "B budget" {
			t.Errorf("FinanceBudget = %+v, %v", b, found)
		}
		items, err := f.service.FinanceBudgetItems(ctx, europe.BudgetID)
		if err != nil || len(items) != 2 || items[0].ID != asia.ID || items[1].ID != europe.ID {
			t.Errorf("FinanceBudgetItems = %+v, %v; want asia, Europe", items, err)
		}
		if i, found, _ := f.service.FinanceBudgetItem(ctx, asia.ID); !found || i.Name != "asia" {
			t.Errorf("FinanceBudgetItem = %+v, %v", i, found)
		}
		return nil
	})
}

// A record may keep an inactive setting it already has, but never choose one newly (FR-004).
func TestCanChoose_KeepsInactiveNeverChoosesNew_FR004(t *testing.T) {
	tests := []struct {
		name           string
		active         bool
		id, previousID string
		want           bool
	}{
		{"active, new", true, "MOD-1001", "", true},
		{"active, other before", true, "MOD-1001", "MOD-1002", true},
		{"inactive, kept", false, "MOD-1001", "MOD-1001", true},
		{"inactive, new", false, "MOD-1001", "", false},
		{"inactive, changed to", false, "MOD-1001", "MOD-1002", false},
	}
	for _, tt := range tests {
		if got := CanChoose(tt.active, tt.id, tt.previousID); got != tt.want {
			t.Errorf("%s: CanChoose = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// An attribute with a condition is shown when the plan's business unit includes any of the condition's
// options; one without a condition is always shown (FR-023, FR-026).
func TestVisible_ConditionAnyBusinessUnit_FR023_FR026(t *testing.T) {
	always := Attribute{Condition: []string{}}
	lubricants := Attribute{Condition: []string{"SO-1002"}}
	either := Attribute{Condition: []string{"SO-1001", "SO-1002"}}
	tests := []struct {
		name         string
		attribute    Attribute
		businessUnit []string
		want         bool
	}{
		{"no condition, no business unit", always, nil, true},
		{"condition met", lubricants, []string{"SO-1001", "SO-1002"}, true},
		{"condition not met", lubricants, []string{"SO-1001"}, false},
		{"no business unit yet", lubricants, []string{}, false},
		{"any of the condition", either, []string{"SO-1001"}, true},
	}
	for _, tt := range tests {
		if got := Visible(tt.attribute, tt.businessUnit); got != tt.want {
			t.Errorf("%s: Visible = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// namesOfModels returns the names of models in their order.
func namesOfModels(models []Model) []string {
	names := []string{}
	for _, m := range models {
		names = append(names, m.Name)
	}
	return names
}
