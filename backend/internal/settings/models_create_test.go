package settings

import (
	"reflect"
	"strings"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// A new model with valid values is listed as Active, with its allowed types by name, and logged (US1-1,
// FR-009).
func TestCreateModel_ListedActive_FR009(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	digital := f.addType(t, "Digital media", true)

	created := f.createModel(t, modelInput("  Lubricants 2.0 ", event.ID, digital.ID, event.ID))

	want := ModelView{
		ID: "MOD-1001", Name: "Lubricants 2.0", InqToMqlRate: money.PercentFromHundredths(200), MqlToDealRate: money.PercentFromHundredths(500),
		AverageDealSize: money.USD(money.FromCents(1_000_000)),
		ActivityTypes:   []SettingRef{{ID: digital.ID, Name: "Digital media", Active: true}, {ID: event.ID, Name: "Event", Active: true}},
		Active:          true, Usage: Usage{Summary: "Not used yet", Parts: []UsagePart{}, Records: []history.RecordRef{}},
		Meta: record.NewMeta(alex, start),
	}
	if !reflect.DeepEqual(created, want) {
		t.Errorf("created = %+v\nwant      %+v", created, want)
	}
	listed, err := f.service.ListModels(t.Context(), alex)
	if err != nil || !reflect.DeepEqual(listed, []ModelView{want}) {
		t.Errorf("ListModels = %+v, %v", listed, err)
	}
	stored, _, _ := f.service.Model(t.Context(), created.ID)
	if !reflect.DeepEqual(stored.ActivityTypeIDs, []string{digital.ID, event.ID}) {
		t.Errorf("stored types = %q, want once each in the order of the type list", stored.ActivityTypeIDs)
	}
	entry := f.lastEntry(t)
	if entry.Action != history.ActionCreated || entry.RecordType != "model" || entry.RecordName != "Lubricants 2.0" || entry.Actor != alex {
		t.Errorf("log entry = %+v", entry)
	}
}

// Every problem of a new model is shown at once and nothing is saved (US1-5, FR-009, SC-003).
func TestCreateModel_AllProblemsAtOnce_FR009(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	input := modelInput("", "")
	input.InqToMqlRate = "0"

	_, err := f.service.CreateModel(t.Context(), alex, input)

	assertError(t, err, problems(
		"name", "Enter a name.",
		"inqToMqlRate", "Enter a rate above 0 % and at most 100 %, with at most two decimals.",
		"activityTypeIds", "Tick at least one activity type.",
	))
	if models, _ := f.service.Models(t.Context()); len(models) != 0 {
		t.Errorf("models = %+v, want none", models)
	}
}

// Each rate is above 0 % and at most 100 %, with at most two decimals (FR-009).
func TestCreateModel_RateBounds_FR009(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	for _, rate := range []string{"0", "0.00", "100.01", "2.555", "abc", "", "-1", "1,5"} {
		input := modelInput("M "+rate, event.ID)
		input.MqlToDealRate = rate
		_, err := f.service.CreateModel(t.Context(), alex, input)
		assertError(t, err, problems("mqlToDealRate", "Enter a rate above 0 % and at most 100 %, with at most two decimals."))
	}
	for i, rate := range []string{"0.01", "100", " 33.33 "} {
		input := modelInput("Valid "+string(rune('A'+i)), event.ID)
		input.InqToMqlRate = rate
		if _, err := f.service.CreateModel(t.Context(), alex, input); err != nil {
			t.Errorf("rate %q refused: %v", rate, err)
		}
	}
}

// The average deal size is in USD, above 0 and at most 100,000,000.00, with at most two decimals (FR-009,
// D-053).
func TestCreateModel_DealSizeAboveZeroUSD_FR009(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	tests := []struct {
		name    string
		size    *money.Input
		message string
	}{
		{"not filled in", nil, "Enter an average deal size above 0 USD, with at most two decimals."},
		{"empty", &money.Input{Amount: "", Currency: "USD"}, "Enter an average deal size above 0 USD, with at most two decimals."},
		{"zero", &money.Input{Amount: "0", Currency: "USD"}, "Enter an average deal size above 0 USD, with at most two decimals."},
		{"negative", &money.Input{Amount: "-5", Currency: "USD"}, "Enter an average deal size above 0 USD, with at most two decimals."},
		{"three decimals", &money.Input{Amount: "1.234", Currency: "USD"}, "Enter an average deal size above 0 USD, with at most two decimals."},
		{"too large", &money.Input{Amount: "100000000.01", Currency: "USD"}, "Use 100,000,000.00 USD or less."},
		{"far too large", &money.Input{Amount: "1000000000000000000", Currency: "USD"}, "Use 100,000,000.00 USD or less."},
		{"in euros", &money.Input{Amount: "5000", Currency: "EUR"}, "Enter the average deal size in USD."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input := modelInput("Model", event.ID)
			input.AverageDealSize = tt.size
			_, err := f.service.CreateModel(t.Context(), alex, input)
			assertError(t, err, problems("averageDealSize", tt.message))
		})
	}
	input := modelInput("Largest", event.ID)
	input.AverageDealSize = &money.Input{Amount: "100000000.00", Currency: "USD"}
	if _, err := f.service.CreateModel(t.Context(), alex, input); err != nil {
		t.Errorf("100,000,000.00 USD refused: %v", err)
	}
}

// A model's name has at most 80 characters and is unique ignoring letter case and surrounding spaces; the
// message names the model it clashes with (FR-005, FR-009).
func TestCreateModel_NameUniqueIgnoringCaseAndSpaces_FR005(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	f.createModel(t, modelInput("Lubricants 1.0", event.ID))

	_, taken := f.service.CreateModel(t.Context(), alex, modelInput("  LUBRICANTS 1.0 ", event.ID))
	_, long := f.service.CreateModel(t.Context(), alex, modelInput(strings.Repeat("é", 81), event.ID))

	assertError(t, taken, problems("name", "A model called 'Lubricants 1.0' already exists."))
	assertError(t, long, problems("name", "Use 80 characters or fewer."))
	f.createModel(t, modelInput(strings.Repeat("é", 80), event.ID))
}

// A new model can allow active activity types only; a type that no longer exists is reported (FR-004).
func TestCreateModel_InactiveTypeRefused_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	webinar := f.addType(t, "Webinar", false)

	_, err := f.service.CreateModel(t.Context(), alex, modelInput("Model", event.ID, webinar.ID, "AT-9999"))

	assertError(t, err, problems(
		"activityTypeIds", "'Webinar' is inactive — tick active activity types only.",
		"activityTypeIds", "An activity type you ticked no longer exists — reload the page and try again.",
	))
}
