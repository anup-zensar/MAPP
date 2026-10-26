package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"mapp/internal/platform/web"
)

const clientTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"

// Every request gets one server span named by its route and a duration measured with route and status; a
// panic counts as a 500 in both; trace context sent by the client is not accepted.
func TestEngine_TracesAndMeasuresEveryRoute_024_FR008_FR009(t *testing.T) {
	spans := tracetest.NewInMemoryExporter()
	reader := sdkmetric.NewManualReader()
	engine, err := web.NewEngine(web.Options{
		Tracer: sdktrace.NewTracerProvider(sdktrace.WithSyncer(spans)),
		Meter:  sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)),
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	engine.GET("/api/v1/things/:thingId", ok)
	engine.GET("/api/v1/broken", func(*gin.Context) { panic("boom") })
	sent := newRequest(http.MethodGet, "/api/v1/things/T-1", "")
	sent.Header.Set("traceparent", "00-"+clientTraceID+"-00f067aa0ba902b7-01")
	for _, r := range []*http.Request{sent, newRequest(http.MethodGet, "/api/v1/broken", "")} {
		engine.ServeHTTP(httptest.NewRecorder(), r)
	}

	assertSpans(t, spans.GetSpans())
	durations := requestDurations(t, reader)
	for _, want := range []struct {
		route  string
		status int
	}{{"/api/v1/things/:thingId", http.StatusOK}, {"/api/v1/broken", http.StatusInternalServerError}} {
		if durations[want.route] != want.status {
			t.Errorf("duration of %s measured with status %d, want %d", want.route, durations[want.route], want.status)
		}
	}
}

func assertSpans(t *testing.T, ended tracetest.SpanStubs) {
	t.Helper()
	if len(ended) != 2 || ended[0].Name != "GET /api/v1/things/:thingId" || ended[1].Name != "GET /api/v1/broken" {
		t.Fatalf("spans = %v, want one per request named by its route", ended.Snapshots())
	}
	if ended[0].SpanContext.TraceID().String() == clientTraceID || ended[0].Parent.IsValid() {
		t.Error("the span continued the trace the client sent")
	}
	if ended[1].Status.Code != codes.Error {
		t.Errorf("the panicking request's span status = %v, want Error", ended[1].Status)
	}
}

// requestDurations returns the status of each route's http.server.request.duration measurement.
func requestDurations(t *testing.T, reader *sdkmetric.ManualReader) map[string]int {
	t.Helper()
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	statuses := map[string]int{}
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			histogram, isHistogram := measured.Data.(metricdata.Histogram[float64])
			if measured.Name != "http.server.request.duration" || !isHistogram {
				continue
			}
			for _, point := range histogram.DataPoints {
				route, _ := point.Attributes.Value(attribute.Key("http.route"))
				status, _ := point.Attributes.Value(attribute.Key("http.response.status_code"))
				statuses[route.AsString()] = int(status.AsInt64())
			}
		}
	}
	return statuses
}
