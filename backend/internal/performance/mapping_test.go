package performance_test

import (
	"errors"
	"strings"
	"testing"

	"mapp/internal/performance"
	"mapp/internal/platform/errs"
)

func stageMapping(column, stage string) performance.MappingInput {
	return performance.MappingInput{ColumnName: column, Meaning: performance.MeaningStage,
		Stage: stage, ModelID: "MOD-1"}
}

func TestCreateMapping_ReportsEveryFaultAtOnce_FR001(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Eloqua")

	// A stage column with no name, no stage, no model, and a web metric it may not carry: four faults,
	// which must come back together rather than one save at a time.
	_, err := f.service.CreateMapping(t.Context(), alex, src.ID, performance.MappingInput{
		Meaning: performance.MeaningStage, WebMetric: performance.MetricSessions,
	})

	var invalid *errs.Error
	if !errors.As(err, &invalid) {
		t.Fatalf("CreateMapping() error = %v, want an *errs.Error", err)
	}
	fields := map[string]bool{}
	for _, f := range invalid.Fields {
		fields[f.Field] = true
	}
	for _, want := range []string{"columnName", "stage", "webMetric", "modelId"} {
		if !fields[want] {
			t.Errorf("%s is not among the faults reported: %+v", want, invalid.Fields)
		}
	}
}

func TestCreateMapping_WebMetricVocabulary_046C11(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		metric performance.WebMetric
		wantOK bool
	}{
		{name: "046-C11 sessions is a web metric", metric: performance.MetricSessions, wantOK: true},
		{name: "046-C11 visits is a web metric", metric: performance.MetricVisits, wantOK: true},
		{name: "046-C11 engagement touches is a web metric", metric: performance.MetricEngagementTouches, wantOK: true},
		{name: "046-C11 impressions is not", metric: performance.WebMetric("impressions")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			src := f.createSource(t, "Web analytics")

			_, err := f.service.CreateMapping(t.Context(), alex, src.ID, performance.MappingInput{
				ColumnName: "metric", Meaning: performance.MeaningWebMetric, WebMetric: tt.metric,
			})

			if tt.wantOK && err != nil {
				t.Fatalf("CreateMapping(%s) = %v, want nil", tt.metric, err)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("CreateMapping(%s) was accepted, want a refusal", tt.metric)
			}
		})
	}
}

func TestCreateMapping_SecondActiveSupplierRefused_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	first := f.createSource(t, "Eloqua")
	second := f.createSource(t, "Marketo")
	if _, err := f.service.CreateMapping(t.Context(), alex, first.ID, stageMapping("mql", "mql")); err != nil {
		t.Fatalf("the first mapping: %v", err)
	}

	_, err := f.service.CreateMapping(t.Context(), alex, second.ID, stageMapping("mql", "mql"))

	var refusal *errs.Error
	if !errors.As(err, &refusal) {
		t.Fatalf("a second active supplier was accepted, want a refusal; err = %v", err)
	}
	if !strings.Contains(refusal.Message, "Eloqua") {
		t.Errorf("the refusal does not name the source already supplying it: %q", refusal.Message)
	}
}

func TestCreateMapping_AnotherStageOfTheSameModelIsFine_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	first := f.createSource(t, "Eloqua")
	second := f.createSource(t, "Marketo")
	if _, err := f.service.CreateMapping(t.Context(), alex, first.ID, stageMapping("mql", "mql")); err != nil {
		t.Fatalf("the first mapping: %v", err)
	}

	if _, err := f.service.CreateMapping(t.Context(), alex, second.ID, stageMapping("inq", "inq")); err != nil {
		t.Errorf("a different stage of the same model was refused: %v", err)
	}
}

func TestCreateMapping_OnADeactivatedSourceRefused_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	src := f.createSource(t, "Eloqua")
	if err := f.service.SetSourceStatus(t.Context(), alex, src.ID, performance.StatusInactive); err != nil {
		t.Fatalf("SetSourceStatus: %v", err)
	}

	_, err := f.service.CreateMapping(t.Context(), alex, src.ID, stageMapping("mql", "mql"))

	if err == nil {
		t.Fatal("a mapping was added to a deactivated source, want a refusal")
	}
	if !strings.Contains(err.Error(), "Eloqua") {
		t.Errorf("the refusal does not name the source: %v", err)
	}
}
