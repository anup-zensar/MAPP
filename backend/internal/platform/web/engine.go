package web

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"mapp/internal/platform/errs"
)

const serviceName = "mapp-api"

type Options struct {
	Tracer         trace.TracerProvider
	Meter          metric.MeterProvider
	BodyLimits     map[string]int64
	TrustedProxies []string
}

// NewEngine puts recovery inside tracing, request ID and log, so a panic keeps their attributes and is a 500.
func NewEngine(o Options) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode) // debug mode prints to standard output, which carries only JSON logs
	engine := gin.New()
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false
	// Only these proxies may report the client's address, which the sign-in attempt limit counts by.
	if err := engine.SetTrustedProxies(append([]string{"127.0.0.1", "::1"}, o.TrustedProxies...)); err != nil {
		return nil, fmt.Errorf("trusted proxies: %w", err)
	}
	engine.Use(
		otelgin.Middleware(serviceName, otelgin.WithTracerProvider(o.Tracer), otelgin.WithMeterProvider(o.Meter),
			otelgin.WithPropagators(propagation.NewCompositeTextMapPropagator())),
		RequestID(),
		LogRequest(),
		Recover(),
		SecurityHeaders(),
		SameOrigin(),
		RejectControlInPath(),
		LimitBody(maxBodyBytes, o.BodyLimits),
	)
	engine.NoRoute(func(c *gin.Context) { RespondError(c, errs.NotFound(notFoundMessage)) })
	return engine, nil
}
