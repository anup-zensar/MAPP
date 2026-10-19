// Package app builds the NorthStar API: it reads the configuration and wires every package's repository,
// service and routes together. It is the only package that knows all the others.
package app

import (
	"net/http"
	"slices"

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
	historyService := history.NewService(db, history.NewJSONRepository(), generator, now)
	peopleService := people.NewService(db, peopleRepo, generator, historyService, now)
	authService := auth.NewService(db, auth.NewJSONRepository(), peopleRepo, cfg.DemoMode(), cfg.DemoPassword, now)
	signedIn := authService.RequireSignIn // wraps every route that needs a signed-in person

	// The kinds of people, teams and team memberships (feature 003), registered before any other business
	// kind, so that history can log their changes and delete teams with their memberships. A kind that
	// history refuses stops the API from starting.
	for _, kind := range peopleService.Kinds() {
		if err := historyService.Register(kind); err != nil {
			return nil, err
		}
	}

	router := web.NewRouter()
	router.Handle("GET /api/health", health(cfg))
	auth.NewHandler(authService, peopleService).Routes(router, signedIn)
	people.NewHandler(peopleService).Routes(router, signedIn)
	history.NewHandler(historyService).Routes(router, signedIn)

	// Settings (feature 002) exist in every environment. Their kinds are registered with history, parents
	// first, so that their deletes can be undone and restored.
	settingsRepo := settings.NewJSONRepository()
	settingsService := settings.NewService(db, settingsRepo, generator, historyService, now)
	for _, kind := range settingsService.Kinds() {
		if err := historyService.Register(kind); err != nil {
			return nil, err
		}
	}
	// Features 003–007 tell Settings where its records are used ("Used by", in-use deletes, impact
	// statements) by adding their sources here: settingsService.AddReferenceSource(…).
	settings.NewHandler(settingsService).Routes(router, signedIn)

	// Plans (004). Plans reads Settings (002) and people and teams (003) through their services. Its history
	// kind comes before the kinds of the levels below the plan (005–007), which also register themselves
	// with planService.RegisterLevel, top-down.
	plansRepo := plans.NewJSONRepository()
	planService := plans.NewService(db, plansRepo, generator, historyService, settingsService, peopleService, now)
	for _, kind := range planService.Kinds() {
		if err := historyService.Register(kind); err != nil {
			return nil, err
		}
	}
	plans.NewHandler(planService).Routes(router, signedIn)

	// Activities (006) are built before campaigns, which roll their summaries up (campaigns.Activities), and
	// line items (007) next to them: each reads the other (activities.LineItems, lineitems.Activities).
	campaignsRepo := campaigns.NewJSONRepository()
	activitiesRepo := activities.NewJSONRepository()
	activitiesService := activities.NewService(db, activitiesRepo, generator, historyService, now, activities.Deps{
		Plans:     plansRepo,
		Hierarchy: campaignsRepo,
		Settings:  settingsService,
		People:    peopleRepo,
		Workflows: activities.NewSeededWorkflows(approvals.NewJSONRepository()),
	})
	lineItemsService := lineitems.NewService(db, lineitems.NewJSONRepository(), generator, historyService,
		lineItemDeps(settingsService, settingsRepo, activitiesService, activitiesRepo, campaignsRepo, plansRepo, peopleRepo), now)
	activitiesService.UseLineItems(lineItemsForActivities{svc: lineItemsService})

	// Feature 005: campaigns and initiatives, with their history kinds under the plan kind (parents first).
	campaignsService := campaigns.NewService(db, campaignsRepo, generator, historyService, now, campaigns.Deps{
		Plans:      campaignPlans{repo: plansRepo},
		People:     peopleRepo,
		Settings:   settingsService,
		Activities: activitiesForCampaigns{svc: activitiesService},
	})
	// The history kinds, parents first: campaign and initiative (005), activity (006), line item (007).
	for _, kind := range slices.Concat(campaignsService.Kinds(), activitiesService.Kinds(), lineItemsService.Kinds()) {
		if err := historyService.Register(kind); err != nil {
			return nil, err
		}
	}
	// The levels below the plan, top-down: campaign, initiative (005), activity (006), line item (007).
	levels := append(campaignLevels(campaignsService), activityLevel(activitiesService),
		lineItemLevel(lineItemsService, activitiesRepo, campaignsRepo))
	for _, level := range levels {
		if err := planService.RegisterLevel(level); err != nil {
			return nil, err
		}
	}
	campaigns.NewHandler(campaignsService).Routes(router, signedIn)
	activities.NewHandler(activitiesService).Routes(router, signedIn)
	lineitems.NewHandler(lineItemsService).Routes(router, signedIn)

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
