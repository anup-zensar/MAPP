// Package app builds the NorthStar API: it reads the configuration and wires every package's repository,
// service and routes together. It is the only package that knows all the others.
package app

import (
	"net/http"

	"northstar/internal/activities"
	"northstar/internal/approvals"
	"northstar/internal/auth"
	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/lineitems"
	"northstar/internal/people"
	"northstar/internal/plans"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/store"
	"northstar/internal/platform/web"
	"northstar/internal/samples"
	"northstar/internal/settings"
)

// App is the NorthStar API, ready to serve.
type App struct {
	// Handler serves every route of the API, wrapped in web.Middleware.
	Handler http.Handler
	// Routes lists the "METHOD /path" pattern of every operation the API serves. Tests compare it with
	// the operations of the contract, backend/api/openapi.yaml.
	Routes []string
}

// New builds the API on the data in db, with now as the clock of the services that need one (a fixed clock
// in tests). It keeps no package-level state, so tests can build as many APIs as they need. An error means
// the API cannot start.
func New(cfg Config, db *store.DB, now clock.Clock) (*App, error) {
	// One generator gives out every readable ID, from the counters in the store.
	generator := ids.NewGenerator(ids.NewJSONCounters())

	// The services, each on its own repositories, with the store as their Transactor. Sign-in also looks
	// people up, so it shares the people repository.
	peopleRepo := people.NewJSONRepository()
	peopleService := people.NewService(db, peopleRepo)
	authService := auth.NewService(db, auth.NewJSONRepository(), peopleRepo, cfg.DemoMode(), cfg.DemoPassword, now)
	historyService := history.NewService(db, history.NewJSONRepository(), generator, now)
	signedIn := authService.RequireSignIn // wraps every route that needs a signed-in person

	router := web.NewRouter()
	router.Handle("GET /api/health", health(cfg))
	auth.NewHandler(authService, peopleService).Routes(router, signedIn)
	people.NewHandler(peopleService).Routes(router, signedIn)
	history.NewHandler(historyService).Routes(router, signedIn)

	// The developer sample records exist only in demo mode (D-036). Their kinds are registered with
	// history, so that their deletes can be undone and restored; a kind that history refuses stops the API
	// from starting.
	if cfg.DemoMode() {
		samplesService := samples.NewService(db, samples.NewJSONRepository(), generator, historyService, now)
		for _, kind := range samplesService.Kinds() {
			if err := historyService.Register(kind); err != nil {
				return nil, err
			}
		}
		samples.NewHandler(samplesService).Routes(router, signedIn)
	}

	// Activity approvals (008). Approval actions are logged on the activity, so its history kind must be
	// registered. INTEGRATION with 006: delete the stand-in kind and pass check: activitiesService.SubmitCheck.
	activitiesRepo := activities.NewJSONRepository()
	if err := historyService.Register(standInActivityKind(activitiesRepo)); err != nil {
		return nil, err
	}
	approvalsService := approvals.NewService(db, approvals.NewJSONRepository(), peopleRepo, &approvalsActivities{
		activities: activitiesRepo, lineItems: lineitems.NewJSONRepository(), campaigns: campaigns.NewJSONRepository(),
		plans: plans.NewJSONRepository(), settings: settings.NewJSONRepository(), check: storedFieldsCheck(activitiesRepo), now: now,
	}, generator, historyService, now)
	for _, kind := range approvalsService.Kinds() {
		if err := historyService.Register(kind); err != nil {
			return nil, err
		}
	}
	approvals.NewHandler(approvalsService).Routes(router, signedIn)

	// Routes lists the operations only: the catch-all registered next is not an operation of the contract.
	routes := router.Patterns()
	// Every other path or method under /api/ answers JSON 404 (http.ServeMux would answer plain text).
	router.Handle("/api/", web.NotFound)

	return &App{Handler: web.Middleware(router), Routes: routes}, nil
}

// healthAnswer is the answer of GET /api/health.
type healthAnswer struct {
	Status      string `json:"status"`
	Environment string `json:"environment"`
	DemoSignIn  bool   `json:"demoSignIn"`
}

// health answers GET /api/health without a session: the API is running, in which environment, and
// whether demo sign-in is on. The web app's developer pages and the seed command ask it.
func health(cfg Config) http.HandlerFunc {
	answer := healthAnswer{Status: "ok", Environment: cfg.Env, DemoSignIn: cfg.DemoMode()}
	return func(w http.ResponseWriter, r *http.Request) {
		web.WriteJSON(w, http.StatusOK, answer)
	}
}
