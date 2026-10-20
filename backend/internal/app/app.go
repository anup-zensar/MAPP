// Package app builds the NorthStar API: it reads the configuration and wires every package's repository,
// service and routes together. It is the only package that knows all the others.
package app

import (
	"net/http"
	"slices"

	"northstar/internal/activities"
	"northstar/internal/allwork"
	"northstar/internal/approvals"
	"northstar/internal/auth"
	"northstar/internal/campaigns"
	"northstar/internal/financials"
	"northstar/internal/history"
	"northstar/internal/home"
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

	// allWork is the source All work reads (allwork_source.go), kept for the adapter's tests.
	allWork allwork.Source
	// finance is the planning tree financials reads (financials.go), kept for the adapter's tests.
	finance financials.Planning
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
	// Features 003–007 tell Settings where its records are used at the end of New, once they are built.
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
	lineItemsRepo := lineitems.NewJSONRepository()
	approvalsRepo := approvals.NewJSONRepository() // the approval workflows of 008 are the activities' workflows
	activitiesService := activities.NewService(db, activitiesRepo, generator, historyService, now, activities.Deps{
		Plans:     plansRepo,
		Hierarchy: campaignsRepo,
		Settings:  settingsService,
		People:    peopleRepo,
		Workflows: activityWorkflows{repo: approvalsRepo},
	})
	lineItemsService := lineitems.NewService(db, lineItemsRepo, generator, historyService,
		lineItemDeps(settingsService, settingsRepo, activitiesService, activitiesRepo, campaignsRepo, plansRepo, peopleRepo,
			peopleService), now)
	activitiesService.UseLineItems(lineItemsForActivities{svc: lineItemsService})
	activitiesService.UseDisplayRates(peopleService) // amounts of activities and plan-tree rows (D-220)

	// Feature 005: campaigns and initiatives, with their history kinds under the plan kind (parents first).
	campaignsService := campaigns.NewService(db, campaignsRepo, generator, historyService, now, campaigns.Deps{
		Plans:      campaignPlans{repo: plansRepo},
		People:     peopleRepo,
		Settings:   settingsService,
		Activities: activitiesForCampaigns{svc: activitiesService},
	})
	campaignsService.UseDisplayRates(peopleService) // amounts of campaigns, initiatives and the tree (D-220)
	// The history kinds, parents first: campaign and initiative (005), activity (006), line item (007).
	for _, kind := range slices.Concat(campaignsService.Kinds(), activitiesService.Kinds(), lineItemsService.Kinds()) {
		if err := historyService.Register(kind); err != nil {
			return nil, err
		}
	}
	// The levels below the plan, top-down: campaign, initiative (005), activity (006), line item (007).
	levels := append(campaignLevels(campaignsService), activityLevel(activitiesService, activitiesRepo, campaignsRepo),
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

	// Activity approvals (008), built after activities and line items: approval actions are logged on the
	// activity, and Submit runs 006's ready-to-submit check, line items included.
	approvalsService := approvals.NewService(db, approvalsRepo, peopleRepo, &approvalsActivities{
		activities: activitiesRepo, lineItems: lineItemsRepo, campaigns: campaignsRepo,
		plans: plansRepo, settings: settingsRepo, check: activitiesService.SubmitCheck, now: now,
	}, generator, historyService, now)
	for _, kind := range approvalsService.Kinds() {
		if err := historyService.Register(kind); err != nil {
			return nil, err
		}
	}
	// Activities (006, and 007 through it) tell approvals when a re-approval change or a restore from the
	// recycle bin ends an activity's approval (FR-022, FR-023).
	activitiesService.UseApprovalHooks(activityApprovalHooks{svc: approvalsService})
	approvalsService.UseDisplayRates(peopleService) // amounts on the Approvals page (D-213)
	approvals.NewHandler(approvalsService).Routes(router, signedIn)

	// Financial transactions (015): GSAP's POs, invoices, payments and accruals, read-only, from the
	// simulated GSAP extract and the demo data, with figures over the planning tree (financials.go). Its two
	// history kinds are top level, so their order is free.
	finance := financialPlanning{
		plans: plansRepo, campaigns: campaignsService, activities: activitiesService, lineItems: lineItemsRepo,
		settings: settingsService, people: peopleService, history: historyService,
	}
	financialsService := financials.NewService(db, financials.NewJSONRepository(), generator, historyService, financials.Deps{
		Planning: finance, Rates: settingsService, Displays: financialDisplays{people: peopleService},
		Access: financialAccess{},
	}, now)
	for _, kind := range financialsService.Kinds() {
		if err := historyService.Register(kind); err != nil {
			return nil, err
		}
	}
	financials.NewHandler(financialsService).Routes(router, signedIn)

	// Home (009): a read-only dashboard over the services of 003–008 (home_sources.go).
	homeService := home.NewService(db, homeSources(peopleService, plansRepo, planService, campaignsService, activitiesService,
		approvalsService))
	home.NewHandler(homeService).Routes(router, signedIn)

	// All work (010): a read-only view of a year over plans (004) and the records below them (005–007),
	// read through the adapter allWorkSource (allwork_source.go), with its own personal settings — saved
	// views and remembered columns — which are not business records: no history kinds (D-121).
	allWork := allWorkSource{
		plans: planService, plansRepo: plansRepo, campaigns: campaignsService, campaignsRepo: campaignsRepo,
		activities: activitiesService, activitiesRepo: activitiesRepo, lineItems: lineItemsService,
		lineItemsRepo: lineItemsRepo, settings: settingsService, budgetItems: lineItemBudgets{repo: settingsRepo},
		workflows: activityWorkflows{repo: approvalsRepo}, people: peopleService, peopleRepo: peopleRepo,
		history: historyService,
	}
	allWorkService := allwork.NewService(db, allwork.NewJSONRepository(), allWork, generator, now)
	allwork.NewHandler(allWorkService).Routes(router, signedIn)

	// Where the records of people (003) and Settings (002) are used, now that every feature is built
	// (references.go): people's plans (004), owner and approver counts (005–008) and currencies (002);
	// Settings' "Used by", in-use deletes and impact statements over people (003), plans (004) and the
	// records below them (005–007), stored and in the recycle bin.
	peopleService.UsePlans(planUses(planService))
	peopleService.UseCurrencies(peopleCurrencies(settingsService))
	for _, usage := range personUsages(campaignsRepo, activitiesRepo, lineItemsRepo, approvalsRepo) {
		peopleService.AddPersonUsage(usage)
	}
	planning := planningRecords{history: historyService, campaigns: campaignsRepo, activities: activitiesRepo, lineItems: lineItemsRepo}
	settingsService.AddReferenceSource(personReferences(peopleService))
	settingsService.AddReferenceSource(planReferences(planService))
	settingsService.AddReferenceSource(planning.references)

	// Routes lists the operations only: the catch-all registered next is not an operation of the contract.
	routes := router.Patterns()
	// Every other path or method under /api/ answers JSON 404 (http.ServeMux would answer plain text).
	router.Handle("/api/", web.NotFound)

	return &App{Handler: web.Middleware(router), Routes: routes, allWork: allWork, finance: finance}, nil
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
