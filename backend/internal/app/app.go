// Package app builds the MAPP API: it reads the configuration and wires every package's repository,
// service and routes together. It is the only package that knows all the others.
package app

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"

	"mapp/internal/access"
	"mapp/internal/activities"
	"mapp/internal/allwork"
	"mapp/internal/approvals"
	"mapp/internal/auth"
	"mapp/internal/campaigns"
	"mapp/internal/financials"
	"mapp/internal/history"
	"mapp/internal/lineitems"
	"mapp/internal/people"
	"mapp/internal/plans"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/telemetry"
	"mapp/internal/platform/web"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// App is the MAPP API; tests compare Routes ("METHOD /path") with the contract and read allWork and finance.
type App struct {
	Handler   http.Handler
	Routes    []string
	allWork   allwork.Source
	finance   financials.Planning
	telemetry *telemetry.Telemetry
}

type wiring struct {
	cfg        Config
	set        stores.Set
	now        clock.Clock
	generator  *ids.Generator
	signedIn   gin.IRoutes
	history    *history.Service
	people     *people.Service
	access     *access.Service
	settings   *settings.Service
	plans      *plans.Service
	campaigns  *campaigns.Service
	activities *activities.Service
	lineItems  *lineitems.Service
	approvals  *approvals.Service
	financials *financials.Service
	allWork    allWorkSource
	finance    financialPlanning
}

type routeRegistrar interface {
	RegisterRoutes(r gin.IRoutes)
}

// New keeps no package-level state, so tests build as many APIs as they need, each on its own clock.
func New(cfg Config, set stores.Set, now clock.Clock) (*App, error) {
	w := &wiring{cfg: cfg, set: set, now: now, generator: ids.NewGenerator(set.Counters)}
	engine, tel, err := w.engine()
	if err != nil {
		return nil, err
	}
	if err = w.areas(); err != nil {
		return nil, err
	}
	return &App{Handler: engine, Routes: web.Operations(engine.Routes()), allWork: w.allWork, finance: w.finance,
		telemetry: tel}, nil
}

// engine puts the sign-in check, then the access middleware (access/routes.go, fail closed) on signed-in routes.
func (w *wiring) engine() (*gin.Engine, *telemetry.Telemetry, error) {
	authService := w.organisation()
	if err := w.register(w.people.Kinds(), w.access.Kinds()); err != nil {
		return nil, nil, err
	}
	engine, tel, err := newEngine(w.cfg)
	if err != nil {
		return nil, nil, err
	}
	w.signedIn = engine.Group(apiV1, authService.RequireSignIn, w.access.Authorize)
	w.organisationRoutes(engine, authService)
	return engine, tel, nil
}

func (w *wiring) organisation() *auth.Service {
	s := w.set
	w.history = history.NewService(s.Tx, s.History, w.generator, w.now)
	w.people = people.NewService(history.Tools{Tx: s.Tx, IDs: w.generator, Log: w.history, Now: w.now}, s.People)
	authService := auth.NewService(s.Tx, s.Auth, s.People, w.cfg.DemoMode(), w.cfg.DemoPassword, w.now)
	w.access = access.NewService(s.Tx, s.Access, w.generator, w.history, accessDirectory{people: w.people}, w.now)
	w.people.UseAccess(peopleAccessHooks(w.access))
	return authService
}

func (w *wiring) organisationRoutes(engine *gin.Engine, authService *auth.Service) {
	probeRoutes(engine, w.cfg, w.set.Ready)
	mount(engine.Group(apiV1), auth.NewHandler(authService, w.people))
	mount(w.signedIn, people.NewHandler(w.people), history.NewHandler(w.history), access.NewHandler(w.access))
}

// register gives history the kinds in order, parents first; a kind it refuses stops the API from starting.
func (w *wiring) register(kinds ...[]history.Kind) error {
	for _, kind := range slices.Concat(kinds...) {
		if err := w.history.Register(kind); err != nil {
			return err
		}
	}
	return nil
}

func mount(r gin.IRoutes, handlers ...routeRegistrar) {
	for _, h := range handlers {
		h.RegisterRoutes(r)
	}
}
