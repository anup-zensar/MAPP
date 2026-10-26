// Package telemetry starts OpenTelemetry tracing and metrics. Only the Console exporter writes anything, to
// the writer Start is given: no setting sends telemetry off this computer (constitution B7).
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type Exporter string

const (
	None    Exporter = "none"
	Console Exporter = "console"
)

const serviceName = "mapp-api"

type Config struct {
	Traces  Exporter
	Metrics Exporter
}

type Telemetry struct {
	Tracer *sdktrace.TracerProvider
	Meter  *sdkmetric.MeterProvider
}

// Start always builds the SDK's providers, so every request gets a trace ID for the logs even when nothing
// is exported.
func Start(cfg Config, console io.Writer) (*Telemetry, error) {
	service := resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(serviceName))
	traceOptions := []sdktrace.TracerProviderOption{sdktrace.WithResource(service)}
	if cfg.Traces == Console {
		exporter, err := stdouttrace.New(stdouttrace.WithWriter(console))
		if err != nil {
			return nil, fmt.Errorf("start the console trace exporter: %w", err)
		}
		traceOptions = append(traceOptions, sdktrace.WithBatcher(exporter))
	}
	metricOptions := []sdkmetric.Option{sdkmetric.WithResource(service)}
	if cfg.Metrics == Console {
		exporter, err := stdoutmetric.New(stdoutmetric.WithWriter(console))
		if err != nil {
			return nil, fmt.Errorf("start the console metric exporter: %w", err)
		}
		metricOptions = append(metricOptions, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)))
	}
	return &Telemetry{
		Tracer: sdktrace.NewTracerProvider(traceOptions...),
		Meter:  sdkmetric.NewMeterProvider(metricOptions...),
	}, nil
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(t.Tracer.Shutdown(ctx), t.Meter.Shutdown(ctx))
}
