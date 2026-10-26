# Research: Gin Transport (024)

Facts come from `go list -m` on 2026-10-26 (Go 1.27.3), the source of Gin v1.12.0 and otelgin v0.72.0 in
the module cache, and today's code on `feature/019-alignment-guardrails` (`28993f6`) plus 022's design
(`specs/022-api-v1/`, in progress).

## R1 — Gin version

- **Decision**: `github.com/gin-gonic/gin` **v1.12.0**.
- **Facts**: `go list -m -versions` lists v1.12.0 as the latest release. Published **2026-02-28 10:10 UTC**,
  about eight months ago — well past the 24-hour release age (Principle VII). Requires Go 1.25; MAPP uses
  1.27.3. MIT licence. Maintained: v1.11.0 came on 2025-09-20, v1.12.0 on 2026-02-28.
- **Why not the standard library**: B1 names Gin as the HTTP transport.
- **Cost**: Gin's core package imports `quic-go/http3`, `x/net/http2/h2c`, `go-playground/validator`,
  `goccy/go-yaml`, `go-toml`, `ugorji/go/codec`, `mongo-driver/v2/bson` and `protobuf`, so they are compiled
  in although MAPP uses none of them. We accept this because B1 requires Gin. `govulncheck` (gate 6),
  software composition analysis (gate 17) and review of `go.sum` cover them. Gin asks for `golang.org/x/net`
  v0.51.0; minimal version selection keeps MAPP's v0.60.0 (019 R3).
- **Alternatives**: stay on `net/http` (against B1); chi or Echo (not the approved stack).

## R2 — OpenTelemetry modules

- **Decision**: `go.opentelemetry.io/otel`, `otel/sdk`, `otel/sdk/metric`,
  `otel/exporters/stdout/stdouttrace` and `stdoutmetric` **v1.47.0** (published 2026-10-02 08:10 UTC), and
  `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin` **v0.72.0** (published
  2026-10-02 16:25 UTC). These are the latest releases, all older than 24 hours, Apache-2.0, and need Go 1.26.
  otelgin v0.72.0 itself requires Gin v1.12.0 and otel v1.47.0, so R1 and R2 move together.
- **What otelgin gives**: one server span per request, named `<METHOD> <route>`, with `http.route` and the
  status. It also records the HTTP server metric `http.server.request.duration`, a histogram labelled with
  `http.route` and `http.response.status_code`. Its count gives the request rate, the status gives the error
  rate, and the buckets give the duration: B7's RED metrics come from one instrument. It records `url.path`
  but not `url.query`, so search text never reaches a span.
- **Alternatives**: `otelhttp` around the engine (its span starts before Gin has matched a route, so it has no
  route template); hand-written spans (would re-implement the semantic conventions).

## R3 — Engine settings (`web.NewEngine`)

- `gin.New()`, not `gin.Default()`: the default Logger writes text to standard output, and the default
  Recovery dumps the request, session cookie included, in debug mode.
- `gin.SetMode(gin.ReleaseMode)` at the start of `NewEngine`. In debug mode Gin prints the route list and
  warnings to standard output, which would break the JSON log stream. The mode is Gin's process-wide atomic
  value, always set to the same constant; it is the only Gin global MAPP touches.
- `RedirectTrailingSlash = false` and `RedirectFixedPath = false`: a trailing slash answers 404 today, not a
  301 redirect. `HandleMethodNotAllowed` stays false, so a wrong method answers 404 as today.
- `SetTrustedProxies(nil)`: `X-Forwarded-For` is never trusted, so `c.ClientIP()` and the span's
  `client.address` are the connection's address.
- `NoRoute` answers `errs.NotFound` in the envelope, replacing ServeMux's `/api/` catch-all.
- No Gin binding or validator. Gin's JSON binder accepts unknown fields unless a package-wide switch is set,
  and validation belongs to the services (all problems at once). Bodies go through `web.BindJSON` (R10).
- No `c.JSON`. When encoding fails, Gin records the error and sends the status with an empty body (`Render`
  in `context.go`). `web.Respond` encodes first and answers `internal_error` instead, as `WriteJSON` does
  today.

## R4 — The route table and access rules on Gin

- **Decision**: `access/routes.go` stays the one route table (Principle VII). Its keys keep the contract's
  form, `"GET /api/v1/plans/{planId}"`. `web.Operation(c)` builds the key from `c.Request.Method` and
  `c.FullPath()` (`/api/v1/plans/:planId` → `{planId}`). `App.Routes` is built the same way from
  `engine.Routes()` and replaces `web.Router.Patterns()`.
- **Groups**: `app` makes two groups. The public group holds `/healthz`, `/readyz`, `/api/v1/environment`
  and auth's demo-people, sign-in and sign-out. The signed-in group, `/api/v1` with `auth.RequireSignIn` and
  `access.Authorize`, holds everything else. A business area's `RegisterRoutes(r gin.IRoutes)` receives only
  the signed-in group; only `app` and `auth` see the public one. A route on the wrong group is still caught
  by the existing tests `TestEveryProtectedOperationRequiresSignIn_FR005` and the public-rules test in
  `app/access_routes_test.go`.
- **Authorize**: it finds the rule with `RuleFor(web.Operation(c))`, and an operation without a rule is
  refused with 403 (fail closed, as today). The target record comes from `c.Params`, the first parameter that
  has a finder; this replaces parsing `r.Pattern`. The lookup lives in `middleware.go`, so `targets.go`
  stays free of the transport. A refusal by a service is still logged after `c.Next()`, from
  `c.Writer.Status()`. 022's 404 for a record the person cannot see is kept.
- **Parameter names**: Gin panics at registration when two routes use different wildcard names at the same
  position. A check of the 232 operations in the route table found none. A static route next to a wildcard
  (`/exchange-rates/current` beside `/exchange-rates/{rateId}`) has been allowed since Gin 1.8. Every
  parameter is a whole path segment.
- **Alternatives**: a rule attached at registration, `r.GET(path, rule, h)`, which would spread the table over
  15 files against "one route table"; keys in Gin's `:param` form, which would stop the table reading like
  the contract.

## R5 — Middleware order

| # | Middleware | On | Why it sits here |
|---|---|---|---|
| 1 | `otelgin.Middleware("mapp-api", …)` | engine | Outermost, so the span and the RED metrics see every request's final status, refusals and recovered panics included |
| 2 | `web.RequestID()` | engine | Before anything logs or answers |
| 3 | `web.LogRequest()` | engine | Writes after the rest has run: final status, and `actor_id` once sign-in has set it |
| 4 | `web.Recover()` | engine | A panic below becomes a normal 500 for 1–3 |
| 5 | `web.SecurityHeaders()` | engine | `Cache-Control: no-store` and `nosniff` are set before anything answers, so refusals and 500s carry them (today's rule) |
| 6 | `web.SameOrigin()` | engine | Refuses other hosts (DNS rebinding) and cross-site requests with 400, as today's `web.Middleware` does |
| 7 | `web.LimitBody(…)` | engine | Before any handler reads the body (R9) |
| 8 | `auth.RequireSignIn` | signed-in group | 401 without a session; puts the actor and `actor_id` in the context |
| 9 | `access.Authorize` | signed-in group | Route rule (fail closed), grants, path target |

**Why recovery is fourth and not first**, as in GO_STANDARDS §9: otelgin restores the request's original
context when it returns (`defer c.Request = c.Request.WithContext(savedCtx)`), and it records status and
metrics only when `c.Next()` returns normally. A recovery placed outside otelgin would log the panic without
`trace_id` or `request_id`, leave the span without a status, and miss the request in the metrics. Placed
inside 1–3, the panic is an ordinary 500 to all three. A panic inside otelgin, `RequestID` or `LogRequest`
themselves falls to `net/http`'s per-connection recovery and is logged through `http.Server.ErrorLog` (R13).
That is three small pieces of code; the risk is accepted. Gin's recovery is used as
`gin.CustomRecoveryWithWriter(nil, …)`: a nil writer turns off Gin's own request dump, and Gin's
broken-pipe handling is kept.

## R6 — Request ID

- **Decision**: `rand.Text()` from `crypto/rand` (26 base-32 characters) gives the `X-Request-ID` response header
  and the `request_id` log attribute. An incoming `X-Request-ID` is ignored, so a client cannot inject text
  into the logs (GO_STANDARDS §8).
- **Contract**: one line in the conventions of `openapi.yaml` ("Every response has the header X-Request-ID"),
  like `Cache-Control` today. `apitest` does not turn on the validator's strict mode, so an undeclared
  response header is not a failure, and the contract tests do not change.
- The request ID is not the trace ID: `trace_id` comes from the span (R8).

## R7 — Logging

- **Decision**: a new package `internal/platform/logging`. `New(w, level)` wraps `slog.NewJSONHandler` in a
  handler that adds the attributes stored in the context (`WithAttrs(ctx, attrs…)`) and `trace_id` from the
  span context. `main` calls `slog.SetDefault(logging.New(os.Stdout, cfg.LogLevel))`. The standard library's
  default logger is the one process-wide logger, so code keeps calling `slog.XxxContext`; sloglint already
  requires the `…Context` forms.
- **Attributes**: `request_id` comes from `RequestID`, `actor_id` from `RequireSignIn` and `trace_id` from the
  handler. Keys are snake_case. The call sites that log without a context today (`web.go`,
  `access/middleware.go`) switch to the `…Context` forms. Access's `person` attribute gives way to `actor_id`
  from the context; `operation` stays.
- **Startup and failure**: a `MAPP API listening` record with `url`, `environment` and `demo_sign_in` replaces
  the printed line, so standard output carries only JSON. A configuration error is logged by `main` with a
  JSON logger at the info level, because the configured level does not exist yet. `http.Server.ErrorLog`
  is `slog.NewLogLogger(handler, slog.LevelError)`.
- **Known gap**: the JSON store logs two records without a context (`store.go`). 023 replaces the store; its
  errors also reach `RespondError` with the request's context.
- **Alternatives**: a logger passed into every constructor (much churn for one process-wide logger); Gin's
  Logger (text).

## R8 — Telemetry providers and exporters

- **Decision**: a new package `internal/platform/telemetry`. `Start(cfg, console io.Writer)` returns a
  `*Telemetry` with the SDK's `Tracer *sdktrace.TracerProvider` and `Meter *sdkmetric.MeterProvider`, and
  `Shutdown(ctx)` flushes.
  - The providers are always the SDK's, so trace IDs exist and reach the logs even with no exporter.
    `none` (the default) attaches no exporter: spans are dropped when they end and metrics stay in memory.
    `console` attaches `stdouttrace` and `stdoutmetric` writing to standard error, because standard output
    is the log stream. Any other value is refused at startup (R13), so no setting sends telemetry off the
    machine (B7).
  - The providers are passed explicitly to otelgin (`WithTracerProvider`, `WithMeterProvider`) and are never
    set as OpenTelemetry globals (B2: no package-level mutable state). 023's `otelpgx` takes the same
    providers.
  - No propagator: `otelgin.WithPropagators(propagation.NewCompositeTextMapPropagator())`. A client's
    `traceparent` cannot choose the `trace_id` written in the logs; propagation comes back with hosting.
  - Resource `service.name=mapp-api`. The SDK's default sampler (parent-based, always on) is kept; with no
    exporter the cost is creating spans, accepted locally.
- **Tests** use the SDK's in-memory exporter (`sdk/trace/tracetest`) and `sdkmetric.NewManualReader`; both
  are in modules already listed in R2.
- **Alternatives**: OTLP to a local collector (needs a collector and a guard against non-local endpoints;
  deferred to the hosting ADR); no-op providers (no `trace_id` in the logs, which fails B7).

## R9 — Body limits

- **Decision**: `web.LimitBody(defaultMax, byOperation)` wraps `c.Request.Body` once in `http.MaxBytesReader`:
  1 MB, or the operation's own limit from `byOperation`. `app` builds that map from settings' exported
  constant: 3 MB for `import-check` and `import`. A second wrapper cannot raise an outer limit, so per-route
  limits live in this one middleware. The limit is a constant, not a `MAX_BODY_BYTES` setting, because no
  requirement asks to tune it.
- `web.BindJSON` keeps today's strict decoding (`application/json`, unknown fields refused, nothing after the
  value) and today's answer (400 `bad_request`, details only in the log). `ReadJSONUpTo` disappears when
  settings moves.

## R10 — Handler helpers

- **Decision**: the final helpers in `web` are `BindJSON(c, dst) error`, `Respond(c, status, body)` and
  `RespondError(c, err)`. The names differ from the `net/http` helpers (`ReadJSON`, `WriteJSON`,
  `WriteError`), so both sets can live side by side during the move without renaming the 429 call sites
  twice.
- `RespondError` is the one place where errors become HTTP answers (B4): an `*errs.Error` keeps its status,
  and anything else is logged once with the request's context and answered `internal_error`.
  GO_STANDARDS' samples change to these names.

```go
func (h *Handler) get(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	plan, err := h.service.Get(c.Request.Context(), a, c.Param("planId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, plan)
}
```

## R11 — Contract tests keep working

- `apitest` builds the API with `app.New` and serves `App.Handler` through `httptest`. A Gin engine is an
  `http.Handler`, so request and response validation against `openapi.yaml` does not change. The one change:
  `app.New` takes the telemetry, and `apitest` passes `telemetry.Start` with `none`.
- **The safety net, unchanged throughout**: `TestRoutesMatchContract`, `TestEveryOperationHasAnAccessRule_SC001`,
  `TestEveryProtectedOperationRequiresSignIn_FR005`, `TestUnknownPathOrMethod_AnswersJSONNotFound`,
  `TestOtherHost_IsRefused`, `TestAnswers_CarryTheAPIHeaders`, the access matrix, every handler test and the
  end-to-end suite.
- **Rewritten**: `web_test.go` (the middleware, now on the engine) and `access/middleware_test.go`
  (`Authorize` on a small test engine).
- **New tests, for new behaviour only**: the logging handler, the request ID, telemetry (span and duration),
  log attributes end to end, body limits per operation, the operation key, configuration, `Serve`. The
  bridge gets no test of its own: every area's tests run through it, and it is deleted at the end.

## R12 — Moving area by area

- **Slice A — the engine**. ServeMux gives way to the Gin engine with middleware 1–7. `web.Router` becomes the
  bridge: it keeps `Handle("GET /api/v1/plans/{planId}", http.HandlerFunc)` but registers on Gin. It converts
  the path, and before calling the handler it sets `c.Request.Pattern` (the ServeMux pattern) and calls
  `c.Request.SetPathValue` for every Gin parameter. `r.PathValue`, `r.Pattern`, the `signedIn` wrappers and
  the access middleware therefore keep working untouched.
- **Slice B — sign-in and access**. They become Gin middleware on the signed-in group, and auth's handlers
  move to Gin. From then on the bridge is bound to the signed-in group, and the `Wrap` it hands to areas not
  yet moved does nothing, because the group does the work.
- **Slices C1–C7 — the areas**, one to three per slice: C1 samples, home · C2 history, lineitems, budgets ·
  C3 plans, campaigns · C4 activities, approvals · C5 people, access · C6 settings · C7 financials, allwork.
  Each moves its handlers to Gin and `RegisterRoutes(r gin.IRoutes)` and changes its line in `app.go`; its
  tests do not change.
- **Slice D — clean-up**. It deletes `web.Router`, `web.Wrap`, `ReadJSON`, `ReadJSONUpTo`, `WriteJSON`,
  `WriteError` and `NotFound`.
- **Every slice** is one commit, or one pull request, that passes `go vet`, `golangci-lint` (new lines) and
  `go test -race ./...`. Slices A, B and D also run the end-to-end suite. A half-moved tree is consistent:
  each package keeps its stack until its slice (Governance → Alignment).
- **Alternatives**: one big change (a diff of thousands of lines that is hard to review); ServeMux in front with
  Gin mounted per area (two routers, two 404s and two middleware chains at once).

## R13 — Server lifecycle and configuration

- `internal/app/server.go`: `Serve(ctx, listener, handler, shutdownTimeout)` builds the `http.Server` with
  these timeouts:
  - `ReadHeaderTimeout` 5 s.
  - `ReadTimeout` 15 s, enough for the 3 MB upload.
  - `WriteTimeout` 30 s; the slowest operation, the 50,000-entry log query, takes about 2 s.
  - `IdleTimeout` 60 s.

  `ErrorLog` goes to slog. One owned goroutine serves, reporting through a buffered channel. When `ctx` ends,
  `Shutdown` runs under `context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)` (GO_STANDARDS §4).
  gosec's G112 check keeps `ReadHeaderTimeout` from being dropped.
- `cmd/api`: `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`. On exit the server drains first, then
  `telemetry.Shutdown` runs within the same deadline, then the store closes (nothing to close for the JSON
  store; 023 closes the pool). `listen()` and its port-in-use message stay.
- **Configuration**: the four variables of FR-012 are read and validated with today's `errors.Join` pattern;
  every problem is reported in one message. None of them is secret, so the message may echo the value, as
  it does for `APP_ENV`. The timeouts are constants (GO_STANDARDS §4), not settings: nothing asks to tune
  them (V).

## R14 — Keeping Gin in the transport

- **Decision**: add a depguard rule `transport-only` to `.golangci.yml`. It denies `github.com/gin-gonic/gin`
  in every file except `**/*handler*.go`, `**/middleware*.go`, `internal/platform/web/**`,
  `internal/app/**`, `cmd/api/**` and tests. The existing `service-layer` rule keeps Gin and `net/http` out
  of `service.go`. The rule can be added before Gin arrives; check depguard's `!` file-glob syntax when it
  is added.
