package plans

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// fakeSettings is Settings with two active models, one inactive, two finance budgets and a few Shell
// attributes. CheckAttributes keeps every value and adds problems, when the test sets them.
type fakeSettings struct {
	models            map[string]settings.Model
	budgets           map[string]settings.FinanceBudget
	attributes        []settings.Attribute
	options           map[string]settings.Option
	attributeProblems []errs.FieldError
}

func newFakeSettings() *fakeSettings {
	model := func(id, name, r1, r2, deal string, active bool) settings.Model {
		p1, _ := money.ParsePercent(r1)
		p2, _ := money.ParsePercent(r2)
		size, _ := money.ParseAmount(deal)
		return settings.Model{ID: id, Name: name, InqToMqlRate: p1, MqlToDealRate: p2, AverageDealSize: money.USD(size),
			ActivityTypeIDs: []string{"AT-1", "AT-2"}, Active: active}
	}
	return &fakeSettings{
		models: map[string]settings.Model{
			lubricantsModel: model(lubricantsModel, "Lubricants 1.0", "2.00", "5.00", "10000.00", true),
			mobilityModel:   model(mobilityModel, "Mobility 1.0", "3.00", "4.00", "5000.00", true),
			oldModel:        model(oldModel, "Old model", "1.00", "1.00", "1000.00", false),
		},
		budgets: map[string]settings.FinanceBudget{
			budget2026: {ID: budget2026, Year: 2026, Name: "2026 Lubricants Budget"},
			budget2027: {ID: budget2027, Year: 2027, Name: "2027 Lubricants Budget"},
		},
		attributes: []settings.Attribute{
			{ID: businessUnit, Label: "Business unit", Level: settings.LevelPlan, Type: settings.TypeMultipleChoice,
				Marking: settings.MarkingNeededToPublish, Position: 1, Role: settings.RoleBusinessUnit, Active: true},
			{ID: smp, Label: "Strategic Marketing Plan (SMP)", Level: settings.LevelPlan, Type: settings.TypeYesNo,
				Marking: settings.MarkingOptional, Position: 2, Condition: []string{mobility}, Active: true},
			{ID: brands, Label: "Brands", Level: settings.LevelActivity, Type: settings.TypeMultipleChoice,
				Marking: settings.MarkingNeededToSubmit, Position: 1, Condition: []string{lubricants}, Active: true},
			{ID: sector, Label: "Sector", Level: settings.LevelActivity, Type: settings.TypeMultipleChoice,
				Marking: settings.MarkingNeededToSubmit, Position: 2, Condition: []string{lubricants}, Active: true},
			{ID: functionMob, Label: "Function (Mobility)", Level: settings.LevelActivity, Type: settings.TypeSingleChoice,
				Marking: settings.MarkingOptional, Position: 3, Condition: []string{mobility}, Active: true},
		},
		options: map[string]settings.Option{
			mobility:   {ID: mobility, AttributeID: businessUnit, Label: "Mobility", Position: 1, Active: true},
			lubricants: {ID: lubricants, AttributeID: businessUnit, Label: "Lubricants", Position: 2, Active: true},
			marine:     {ID: marine, AttributeID: businessUnit, Label: "Marine", Position: 3, Active: false},
		},
	}
}

// fakeCurrencies are the currencies of the tests (016): USD, EUR, GBP active, INR inactive, at the rates of
// the 016 examples.
var fakeCurrencies = map[string]settings.Currency{
	"USD": {Code: "USD", Rate: money.One, Active: true},
	"EUR": {Code: "EUR", Rate: money.RateFromMicros(920_000), Active: true},
	"GBP": {Code: "GBP", Rate: money.RateFromMicros(790_000), Active: true},
	"INR": {Code: "INR", Rate: money.RateFromMicros(83_500_000), Active: false},
}

func (f *fakeSettings) Currency(_ context.Context, code string) (settings.Currency, bool, error) {
	c, found := fakeCurrencies[code]
	return c, found, nil
}

func (f *fakeSettings) Rates(context.Context) (money.Rates, error) {
	rates := money.Rates{}
	for code, c := range fakeCurrencies {
		rates[code] = c.Rate
	}
	return rates, nil
}

func (f *fakeSettings) Model(_ context.Context, id string) (settings.Model, bool, error) {
	m, found := f.models[id]
	return m, found, nil
}

func (f *fakeSettings) Models(context.Context) ([]settings.Model, error) {
	models := slices.Collect(maps.Values(f.models))
	slices.SortFunc(models, func(x, y settings.Model) int { return strings.Compare(x.Name, y.Name) })
	return models, nil
}

func (f *fakeSettings) FinanceBudget(_ context.Context, id string) (settings.FinanceBudget, bool, error) {
	b, found := f.budgets[id]
	return b, found, nil
}

func (f *fakeSettings) Attributes(_ context.Context, level settings.Level) ([]settings.Attribute, error) {
	ofLevel := []settings.Attribute{}
	for _, a := range f.attributes {
		if a.Level == level {
			ofLevel = append(ofLevel, a)
		}
	}
	return ofLevel, nil
}

func (f *fakeSettings) Option(_ context.Context, id string) (settings.Option, bool, error) {
	o, found := f.options[id]
	return o, found, nil
}

func (f *fakeSettings) CheckAttributes(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error) {
	missing, _ := f.MissingAttributes(ctx, level, in.Values, nil)
	return settings.AttributeResult{
		Problems: slices.Clone(f.attributeProblems), Cleared: []settings.ClearedValue{}, Missing: missing,
		Values: maps.Clone(in.Values),
	}, nil
}

// MissingAttributes reports the Business unit when it is empty.
func (f *fakeSettings) MissingAttributes(ctx context.Context, _ settings.Level, values settings.AttributeValues, _ []string) ([]settings.MissingField, error) {
	if unit, _ := f.BusinessUnit(ctx, values); len(unit) > 0 {
		return []settings.MissingField{}, nil
	}
	return []settings.MissingField{{Field: "attributes." + businessUnit, Label: "Business unit"}}, nil
}

func (f *fakeSettings) BusinessUnit(_ context.Context, values settings.AttributeValues) ([]string, error) {
	unit := []string{}
	_ = json.Unmarshal(values[businessUnit], &unit)
	return unit, nil
}

// DescribeAttribute names the attribute and the labels of its options.
func (f *fakeSettings) DescribeAttribute(_ context.Context, field string, value any) (string, string, error) {
	id := strings.TrimPrefix(field, "attributes.")
	i := slices.IndexFunc(f.attributes, func(a settings.Attribute) bool { return a.ID == id })
	if i < 0 {
		return "", "", nil
	}
	var labels []string
	list, _ := value.([]any)
	for _, item := range list {
		labels = append(labels, f.options[item.(string)].Label)
	}
	return f.attributes[i].Label, strings.Join(labels, ", "), nil
}
