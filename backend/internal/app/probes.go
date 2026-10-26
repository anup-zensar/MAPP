package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

const readinessTimeout = 2 * time.Second

type environmentAnswer struct {
	Environment string `json:"environment"`
	DemoSignIn  bool   `json:"demoSignIn"`
}

// probeRoutes registers the liveness and readiness probes, which stay outside /api/v1, and the environment
// answer the web app's developer pages read.
func probeRoutes(r gin.IRoutes, cfg Config, ready func(ctx context.Context) error) {
	r.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
		defer cancel()
		if err := ready(ctx); err != nil {
			slog.ErrorContext(c.Request.Context(), "not ready", "error", err)
			web.RespondError(c, errs.Unavailable())
			return
		}
		c.Status(http.StatusNoContent)
	})
	environment := environmentAnswer{Environment: cfg.Env, DemoSignIn: cfg.DemoMode()}
	r.GET(apiV1+"/environment", func(c *gin.Context) {
		web.Respond(c, http.StatusOK, environment)
	})
}
