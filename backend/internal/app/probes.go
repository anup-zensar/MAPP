package app

import (
	"log/slog"
	"net/http"

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
func probeRoutes(router *web.Router, cfg Config, db *store.DB) {
	router.Handle("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	router.Handle("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ready(); err != nil {
			slog.ErrorContext(r.Context(), "not ready", "error", err)
			web.WriteError(w, r, errs.Unavailable())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	environment := environmentAnswer{Environment: cfg.Env, DemoSignIn: cfg.DemoMode()}
	router.Handle("GET /api/v1/environment", func(w http.ResponseWriter, _ *http.Request) {
		web.WriteJSON(w, http.StatusOK, environment)
	})
}
