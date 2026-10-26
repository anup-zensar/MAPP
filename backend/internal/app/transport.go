package app

import (
	"context"
	"os"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/telemetry"
	"mapp/internal/platform/web"
	"mapp/internal/settings"
)

const apiV1 = "/api/v1"

// Close flushes what telemetry still holds; the server must have stopped.
func (a *App) Close(ctx context.Context) error {
	return a.telemetry.Shutdown(ctx)
}

func newEngine(cfg Config) (*gin.Engine, *telemetry.Telemetry, error) {
	tel, err := telemetry.Start(cfg.Telemetry, os.Stderr)
	if err != nil {
		return nil, nil, err
	}
	engine, err := web.NewEngine(web.Options{Tracer: tel.Tracer, Meter: tel.Meter, BodyLimits: map[string]int64{
		"POST /api/v1/exchange-rates/import-check": settings.MaxRateUploadBytes,
		"POST /api/v1/exchange-rates/import":       settings.MaxRateUploadBytes,
	}})
	if err != nil {
		return nil, nil, err
	}
	return engine, tel, nil
}
