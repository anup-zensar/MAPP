package app

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/store"
	"mapp/internal/platform/web"
)

type environmentAnswer struct {
	Environment string `json:"environment"`
	DemoSignIn  bool   `json:"demoSignIn"`
}

// probeRoutes registers the liveness and readiness probes, which stay outside /api/v1, and the environment
// answer the web app's developer pages read.
func probeRoutes(r gin.IRoutes, cfg Config, db *store.DB) {
	r.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.GET("/readyz", func(c *gin.Context) {
		if err := db.Ready(); err != nil {
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
