# Implementation Plan: Gin Transport

**Branch**: `feature/024-gin-transport` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/024-gin-transport/spec.md`

## Summary

The API's HTTP layer moves from `net/http` ServeMux to Gin v1.12.0, confined to handler and middleware
files. A fixed middleware chain does the work of today's `web.Middleware` and adds the constitution's
observability:

1. OpenTelemetry tracing and RED metrics (otelgin)
2. request ID
3. request log
4. panic recovery
5. API headers
6. same-host and same-site check
7. body limit
8. sign-in and the access route table, on the routes that need a session

Logs become JSON on standard output carrying `request_id`, `trace_id` and `actor_id`. Telemetry stays in
the process unless the `console` exporter is chosen, which writes to standard error. The server gets
timeouts and a SIGINT/SIGTERM drain. The move is made area by area through a temporary bridge that serves
not-yet-moved `net/http` handlers on the Gin engine. Every step is proved by today's tests passing
unchanged. Decisions and alternatives are in [research.md](research.md).

## Technical Context

**Language/Version**: Go 1.27.3

**Primary Dependencies**: added: `github.com/gin-gonic/gin` v1.12.0;
`go.opentelemetry.io/otel`, `otel/sdk`, `otel/sdk/metric`, `otel/exporters/stdout/stdouttrace`,
`stdoutmetric` v1.47.0; `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin`
v0.72.0 (R1, R2)

**Storage**: unchanged (JSON file store; PostgreSQL is 023)

**Testing**: Go `testing`, `httptest`, `apitest` contract tests, SDK in-memory span exporter and manual metric
reader; Playwright end-to-end suite unchanged

**Target Platform**: local, `127.0.0.1` only

**Project Type**: web application (Go API + React web app); this feature changes only `backend/`

**Constraints**: no API behaviour change beyond 022 except the `X-Request-ID` header; no telemetry off the
machine; every slice leaves every gate green

**Scale/Scope**: 232 operations in today's route table (022 swaps `/api/health` for three probe and
environment operations); 15 handler packages; 429 helper call sites

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | FR-001–FR-014; test names carry `024 FR-…` | Pass |
| III Documented design | Interfaces, threat model and risk model below; ADR task (T001). There is no `data-model.md` or `contracts/` because no record changes and the only contract change is one convention line (R6); the "Verify" section below serves as the quickstart | Pass |
| IV Two parts | No new runtime part: telemetry runs in the process | Pass |
| V Simplicity | B1 names this stack; no new layer; one temporary bridge, deleted in slice D; timeouts and limits are constants | Pass |
| VI Meaningful tests | Today's tests prove nothing changed; new tests cover new behaviour only (R11); determinism through channels, in-memory exporters and a fixed clock, with no sleeps | Pass |
| VII Security | One route table, fail closed, kept (R4); scope still comes only from the actor; no personal data in logs or spans; request and trace IDs are not client-controlled; release age checked (R1, R2) | Pass |
| VIII Clean construction | New files ≤ 300 lines with comments ≤ 5%; one concept per file (`engine.go`, `middleware.go`, `respond.go`, …) | Pass |
| B1 Stack | Gin, `log/slog` JSON, OpenTelemetry | Pass |
| B2 Layers | Gin only in transport files; depguard rule `transport-only` (R14); services unchanged | Pass |
| B3 Lifecycle | `http.Server` timeouts, SIGINT/SIGTERM, bounded shutdown. Closing the database does not apply to the JSON store; 023 closes the pool | Pass |
| B4 Errors | `web.RespondError` is the one mapping; recovery middleware turns panics into a logged 500 | Pass |
| B6 Contract | Body limits per operation; contract unchanged except the `X-Request-ID` convention. GO_STANDARDS' 413 is not adopted because it would change behaviour (follow-up) | Pass |
| B7 Observability | JSON logs with `request_id`/`trace_id`/`actor_id`; inbound HTTP traced; RED metrics per route, exported to standard error with `OTEL_METRICS_EXPORTER=console` (the default `none` keeps them in the process: locally nothing collects them, and standard output is reserved for logs); nothing off the machine; typed config. Database tracing comes with 023; there are no outbound calls | Pass |
| B9 Go testing | `go test -race`; handler tests stay external packages; no `time.Sleep` | Pass |
| Governance → Alignment | Packages move one slice at a time; a package keeps its stack until its slice | Pass |

**Post-design re-check**: unchanged — Pass. No Complexity Tracking entries.

## Design elements and interfaces

| Element (module) | Role | Interface |
|---|---|---|
| `internal/platform/logging` (new) | JSON slog with request attributes | `New(w io.Writer, level slog.Leveler) *slog.Logger` · `WithAttrs(ctx, attrs ...slog.Attr) context.Context`; the handler adds the context's attributes and `trace_id` |
| `internal/platform/telemetry` (new) | OpenTelemetry providers | `Config{Traces, Metrics Exporter}` (`"none"`, `"console"`) · `Start(cfg Config, console io.Writer) (*Telemetry, error)` · `Telemetry{Tracer trace.TracerProvider; Meter metric.MeterProvider}` · `(*Telemetry).Shutdown(ctx) error` |
| `internal/platform/web` (changed) | Engine, middleware, request and response helpers | `NewEngine(Options{Tracer, Meter, BodyLimits map[string]int64}) *gin.Engine` · `RequestID`, `LogRequest`, `Recover`, `SecurityHeaders`, `SameOrigin`, `LimitBody(defaultMax int64, byOperation map[string]int64)` (all `gin.HandlerFunc`) · `Operation(c) string` (`"GET /api/v1/plans/{planId}"`) · `Operations(gin.RoutesInfo) []string` · `BindJSON(c, dst) error` · `Respond(c, status, body)` · `RespondError(c, err)` · bridge until slice D: `Router.Handle(pattern, http.HandlerFunc)`, `Wrap`, `ReadJSON`, `ReadJSONUpTo`, `WriteJSON`, `WriteError` |
| `internal/auth` | Sign-in middleware | `(*Service).RequireSignIn(c *gin.Context)`: 401 without a session; actor and `actor_id` into the request context · `(*Handler).RegisterRoutes(public gin.IRoutes)` |
| `internal/access` | Route table and its enforcement | `RuleFor(operation string) (Rule, bool)` (unchanged) · `(*Service).Authorize(c *gin.Context)` |
| each business area | Gin handlers | `(*Handler).RegisterRoutes(r gin.IRoutes)`, with paths relative to `/api/v1` |
| `internal/app` | Wiring, configuration, server | `New(cfg Config, db *store.DB, now clock.Clock, tel *telemetry.Telemetry) (*App, error)` · `App.Handler http.Handler` (the engine) · `App.Routes []string` (from `engine.Routes()`) · `Serve(ctx, listener net.Listener, handler http.Handler, shutdownTimeout time.Duration) error` · `Config` gains `LogLevel`, `ShutdownTimeout`, `Telemetry` |
| `cmd/api` | Process lifetime | `run`: signals → config → logger → telemetry → store → `app.New` → `listen` → `app.Serve` → telemetry `Shutdown` |

**Request flow** (engine → groups):

```text
otelgin → RequestID → LogRequest → Recover → SecurityHeaders → SameOrigin → LimitBody
  ├─ public group:    /healthz · /readyz · /api/v1/environment · /api/v1/auth/{demo-people,sign-in,sign-out}
  ├─ signed-in group: /api/v1/… → auth.RequireSignIn → access.Authorize → area handler
  └─ NoRoute:         404 not_found envelope
```

## Threat model (STRIDE)

New flows: inbound headers that reach logs and telemetry, the telemetry output, and panic logging. Trust
boundaries are unchanged.

| Threat | Where | Response |
|---|---|---|
| Spoofing / repudiation: a client picks the request or trace ID to forge or confuse log correlation | `X-Request-ID`, `traceparent` | Both are ignored; IDs come from `crypto/rand` and the OpenTelemetry SDK (R6, R8) |
| Spoofing: a forged client address in traces | `X-Forwarded-For` | `SetTrustedProxies(nil)` (R3) |
| Tampering: log injection through IDs or paths | logs | The JSON handler escapes every value; IDs are made by the server |
| Information disclosure: personal data or secrets in logs and spans | logs, spans | Only `actor_id`. No bodies, cookies or query strings: otelgin records `url.path`, not `url.query`. Gin's request dump is off (`gin.New`, release mode, a nil recovery writer). A panic logs its stack, not the request |
| Information disclosure: telemetry leaves the machine | exporters | Only `none` and `console` (standard error) are accepted, validated at startup (FR-012) |
| Information disclosure: internal errors reach the client | `RespondError` | Unchanged: the client gets `internal_error`, details go only to the log |
| Denial of service: slow clients, large bodies | server, middleware | Timeouts (FR-011); body limits per operation (FR-010) |
| Elevation of privilege: a route served without sign-in or a rule | registration | One route table, fail closed; areas only get the signed-in group; the existing coverage tests (R4) |
| Supply chain | Gin, OpenTelemetry | Release age verified (R1, R2); `govulncheck` and software composition analysis gates; `go.sum` reviewed |

Cryptography: `crypto/rand` for request IDs only.

## Risks

| Risk | Response | Accepted by |
|---|---|---|
| ServeMux and Gin match differently (HEAD, trailing slash, 405, encoded paths) | Engine settings (R3); the unknown-path, other-host and header tests stay unchanged; end-to-end suite after slices A, B and D. HEAD now answers 404 | Technical lead |
| 022 not merged, or its paths change again | Every task that touches 022's files waits for it (marked ⏳022 in `tasks.md`) | — |
| 023 lands in parallel and both change `app.go`, `apitest` and `/readyz` | The second to merge rebases; both add a parameter to `app.New` | Technical lead |
| The half-moved state lingers | Slice D belongs to this feature's Definition of Done; the feature is not Done while the bridge exists | Product owner (schedules) |
| Gin brings in a large set of indirect modules | `govulncheck` and SCA gates; accepted because B1 mandates Gin | Technical lead |
| Creating spans for every request costs time | No exporter locally; measure if a performance requirement appears | Technical lead |

## Project Structure

### Documentation (this feature)

```text
specs/024-gin-transport/
├── spec.md
├── plan.md        # this file
├── research.md
└── tasks.md
```

### Source code (files that change)

```text
backend/go.mod, go.sum                         Gin, otelgin, OpenTelemetry (R1, R2)
backend/.golangci.yml                          depguard rule transport-only (R14)
backend/api/openapi.yaml                       conventions: X-Request-ID (description only)
backend/cmd/api/main.go, main_test.go          signals, logger, telemetry, app.Serve, startup record
backend/internal/app/config.go, config_test.go LOG_LEVEL, SHUTDOWN_TIMEOUT, OTEL_*_EXPORTER
backend/internal/app/server.go, server_test.go Serve (new)
backend/internal/app/app.go, app_test.go,      engine, groups, RegisterRoutes, App.Routes, body limits,
  access_routes_test.go, logging_test.go         log attributes end to end (new test file)
backend/internal/platform/logging/             new
backend/internal/platform/telemetry/           new
backend/internal/platform/web/                 engine.go, middleware.go, respond.go (new); router.go and
                                                 web.go are the bridge until slice D; web_test.go rewritten
backend/internal/platform/apitest/apitest.go   app.New's telemetry argument
backend/internal/auth/middleware.go, handler.go
backend/internal/access/middleware.go, targets.go, middleware_test.go
backend/internal/{samples,home,history,lineitems,budgets,plans,campaigns,activities,approvals,
  people,access,settings,financials,allwork}/*handler*.go   Gin handlers, RegisterRoutes
backend/internal/settings/ratefeed.go          exported MaxRateUploadBytes
backend/internal/financials/handler_test.go    its own router becomes a Gin engine
backend/docs/GO_STANDARDS.md, backend/README.md, README.md, CLAUDE.md
docs/architecture/adr/0011-gin-transport-and-opentelemetry.md (next free number), adr/README.md,
  0003 (status), docs/architecture/overview.md, docs/sdd/feature-map.md
```

**Structure decision**: no package is added or renamed except the two platform packages B2 and GO_STANDARDS
§2 already list (`logging`, `telemetry`). Package renames are feature 025.

## Verify

```sh
cd backend
go run ./cmd/api                                   # standard output: JSON only, first "MAPP API listening"
curl -si http://127.0.0.1:8080/healthz             # 204 with X-Request-ID
curl -si http://127.0.0.1:8080/api/v1/plans/       # 404 not_found envelope, no redirect
OTEL_TRACES_EXPORTER=console OTEL_METRICS_EXPORTER=console go run ./cmd/api 2>telemetry.json
kill -TERM <pid>                                   # exits 0 once in-flight requests finish
go test -race ./... && golangci-lint run ./...
```
