---

description: "Tasks for 024 Gin transport"
---

# Tasks: Gin Transport

**Input**: Design documents from `specs/024-gin-transport/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md)

**Tests**: today's tests are the proof that nothing changed and stay as they are; new tests only for new
behaviour (research R11). Test names cite `024 FR-…`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: the user story the task belongs to (US1–US3)
- **⏳022**: touches files that 022 changes, or builds on code that does; start only after 022 has merged
  into this branch's base

**Every slice ends at a checkpoint**: `gofmt -s -l .`, `goimports -local mapp -l .`, `go vet ./...`,
`golangci-lint run --new-from-merge-base=origin/develop ./...` and `go test -race ./...` from `backend/`, all
clean. Slices A, B and D also run `npm run test:e2e` from `frontend/`. A slice is one commit, or one pull
request.

---

## Phase 1: Setup — can start before 022

- [x] T001 [P] Write `docs/architecture/adr/0011-gin-transport-and-opentelemetry.md` (use the next free
  number if 023 has taken 0011): Gin confined to the transport, the middleware order and its reason,
  OpenTelemetry with exporters `none` and `console` only, no propagation. Mark ADR-0003 "Superseded in part"
  and update `docs/architecture/adr/README.md` (FR-014)
- [x] T002 [P] Add the depguard rule `transport-only` to `backend/.golangci.yml`: deny
  `github.com/gin-gonic/gin` except in `**/*handler*.go`, `**/middleware*.go`, `internal/platform/web/**`,
  `internal/app/**`, `cmd/api/**` and tests (FR-002, R14)

---

## Phase 2: Foundational, part 1 — can start before 022 (none of 022's files)

- [x] T003 Add `LogLevel`, `ShutdownTimeout` and `Telemetry{Traces, Metrics}` to the typed config in
  `backend/internal/app/config.go`, reading `LOG_LEVEL`, `SHUTDOWN_TIMEOUT`, `OTEL_TRACES_EXPORTER` and
  `OTEL_METRICS_EXPORTER`. Use the defaults and allowed values of FR-012; every problem goes into one
  message (FR-012)
- [x] T004 In `backend/internal/app/config_test.go`, extend the defaults and every-variable tests, and write
  the table-driven `TestLoadConfig_RefusesInvalidValues_024_FR012` (`otlp` refused),
  `TestLoadConfig_LimitsAreAccepted_024_FR012` and `TestLoadConfig_ReportsEveryProblem_024_FR012`
- [x] T005 [P] Create `backend/internal/platform/logging/logging.go` with `New` and `WithAttrs` (R7). In
  `logging_test.go`, write `TestHandler_AddsRequestAttributesAndTraceID_024_FR006` and
  `TestWithAttrs_SiblingsKeepTheirOwnAttributes_024_FR006`
- [x] T006 [P] Create `backend/internal/platform/telemetry/telemetry.go` with `Start`, `Shutdown`, exporters
  `none`/`console` and resource `service.name=mapp-api` (R8). First re-check the publish dates of the
  OpenTelemetry v1.47.0 modules, then `go get` them. In `telemetry_test.go`, write
  `TestStart_OnlyConsoleWritesAndTraceIDsAlwaysExist_024_FR008`

**Checkpoint**: gates green; nothing calls the new packages yet except their tests.

---

## Phase 3: User Story 3 — The server shuts down gracefully (Priority: P3) — can start before 022

**Goal**: timeouts, a SIGINT/SIGTERM drain, JSON logs on standard output.

**Independent Test**: start the API, start a slow request, send SIGTERM; start it with invalid settings.

- [x] T007 [US3] Create `backend/internal/app/server.go` with `Serve(ctx, listener, handler,
  shutdownTimeout)`. It sets the timeouts of FR-011, routes `ErrorLog` to slog, serves in one owned goroutine
  and shuts down under `context.WithoutCancel` plus the deadline (R13) (FR-011)
- [x] T008 [US3] In `backend/internal/app/server_test.go`, write
  `TestServe_DrainsTheRequestInProgress_024_FR011` (the handler blocks on a channel; cancel; release; the
  answer is 204 and `Serve` returns nil) and `TestServe_DeadlinePassed_ReturnsError_024_FR011`. Use no sleeps
- [x] T009 [US3] In `backend/cmd/api/main.go`, use `signal.NotifyContext(…, os.Interrupt, syscall.SIGTERM)`,
  `slog.SetDefault(logging.New(os.Stdout, cfg.LogLevel))` and `app.Serve`. The startup line becomes the
  `MAPP API listening` record (`url`, `environment`, `demo_sign_in`), and a configuration error is logged as
  JSON. `TestStartupLine` goes with the printed line (the record only repeats configuration values that
  `config_test.go` covers); `listen` and its tests stay (FR-006, FR-011)

**Checkpoint**: User Story 3 works on today's `net/http` router; gates and the end-to-end suite green.

---

## Phase 4: Foundational, part 2 — the Gin platform ⏳022

**⚠️ CRITICAL**: no area moves before slices A and B are done.

### Slice A — the engine and the bridge

- [x] T010 ⏳022 Re-check the publish dates, then `go get github.com/gin-gonic/gin@v1.12.0` and
  `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin@v0.72.0`. Record both dates
  in the pull request (VII release age)
- [x] T011 ⏳022 Create `backend/internal/platform/web/engine.go`: `NewEngine` with the settings of R3,
  middleware 1–7 of R5 and a `NoRoute` answering the 404 envelope (FR-001, FR-004)
- [x] T012 ⏳022 Create `backend/internal/platform/web/middleware.go` with `RequestID`, `LogRequest`,
  `Recover` (via `gin.CustomRecoveryWithWriter(nil, …)`), `SecurityHeaders`, `SameOrigin` and `LimitBody`.
  Move today's host and cross-origin logic out of `web.Middleware`, which is deleted (FR-004, FR-005,
  FR-007, FR-009, FR-010)
- [x] T013 ⏳022 Create `backend/internal/platform/web/respond.go` with `BindJSON`, `Respond`, `RespondError`,
  `Operation` and `Operations` (R4, R10). The legacy `ReadJSON`, `WriteJSON` and `WriteError` in `web.go`
  log with `slog.…Context(r.Context(), …)` (FR-006)
- [x] T014 ⏳022 Turn `backend/internal/platform/web/router.go` into the bridge. `Router` wraps a
  `gin.IRoutes`; `Handle` converts the ServeMux pattern and, before calling the handler, sets
  `c.Request.Pattern` and calls `c.Request.SetPathValue` for every parameter (R12) (FR-013)
- [x] T015 ⏳022 In `backend/internal/platform/web/web_test.go`, rewrite the middleware, decoding and error
  tests against the engine (`TestEngine_*`, `TestBindJSON_*`, `TestRespondError_*`). Add
  `TestEngine_RequestIDMadeByTheServer_024_FR005`, `TestEngine_OneRequestRecordPerRequest_024_FR007`,
  `TestEngine_PanicAnswersInternalError_024_FR009` and `TestLimitBody_AnOperationMayRaiseTheLimit_024_FR010`.
  Remove the `TestRouter_*` tests; `Operation` is proven by every access test that finds its rule
- [x] T016 ⏳022 Rewire `backend/internal/app/app.go` through the new `internal/app/transport.go`: `New` keeps
  its signature (so feature 023's wiring changes stay small), starts the telemetry from `cfg.Telemetry`
  (console writer `os.Stderr`) and builds the engine with the body limits (export
  `settings.MaxRateUploadBytes` in `internal/settings/ratefeed.go`); the bridge, the probes and
  `/api/v1/environment` sit on the engine root. `NoRoute` replaces the `/api/` catch-all, and
  `App.Routes = web.Operations(engine.Routes())` (FR-001, FR-003)
- [x] T017 ⏳022 `App.Close(ctx)` flushes the telemetry; `backend/cmd/api/main.go` calls it after `Serve`,
  within `SHUTDOWN_TIMEOUT`. `apitest` needs no change (FR-008, FR-011)

**Checkpoint A**: every route is served by Gin, every area still uses `net/http` handlers; gates and the
end-to-end suite green with no test expectation changed.

### Slice B — sign-in and access as Gin middleware

- [ ] T018 ⏳022 In `backend/internal/auth/middleware.go`, make `RequireSignIn(c *gin.Context)` put the actor
  and `logging.WithAttrs(actor_id)` into `c.Request`. In `handler.go`, move to Gin handlers with
  `RegisterRoutes(public gin.IRoutes)` (FR-003, FR-006)
- [ ] T019 ⏳022 In `backend/internal/access/middleware.go`, make `Authorize(c *gin.Context)` look up its rule
  with `web.Operation(c)` and refuse an operation without one with 403. It finds the target from `c.Params`
  (the lookup moves out of `targets.go`), logs with `…Context` (`actor_id` replaces `person`) and keeps
  022's 404 for records the person cannot see (FR-003, FR-006)
- [ ] T020 ⏳022 Rewrite `backend/internal/access/middleware_test.go` to serve through a small Gin engine. The
  log test asserts `operation` and `actor_id` and no name or e-mail (FR-003, FR-006)
- [ ] T021 ⏳022 In `backend/internal/app/app.go`, create the groups: public (probes, environment, auth) and
  signed-in (`/api/v1` with `RequireSignIn` and `Authorize`). The bridge moves onto the signed-in group, and
  the `Wrap` it hands to areas not yet moved does nothing. Delete the old `signedIn` closure (FR-003,
  FR-004)

**Checkpoint B**: gates and the end-to-end suite green.

---

## Phase 5: User Story 2 — Request-scoped JSON logs and traces (Priority: P2) ⏳022

**Goal**: prove FR-005–FR-009 end to end.

**Independent Test**: spec User Story 2.

- [ ] T022 [P] [US2] Create `backend/internal/platform/web/telemetry_test.go` with an engine on an in-memory
  span exporter and a manual metric reader. Write `TestEngine_SpanAndDurationPerRoute_024_FR008` (span
  `GET /x/:id` with `http.route` and status; a duration point with route and status) and
  `TestEngine_PanicCountedAsError_024_FR009`
- [ ] T023 [US2] Create `backend/internal/app/logging_test.go` (through `apitest`, not parallel, swapping the
  default slog logger). In `TestRequestLogs_CarryRequestTraceAndActor_024_FR006`, Sara's refused delete and
  a successful read must produce records that are all JSON, with `request_id` equal to the header and with
  `trace_id`, and `actor_id` `U-1006` after sign-in; none may contain "Sara", "Novak", "@" or the session
  cookie. Add `TestClientTraceparent_Ignored_024_FR008`
- [ ] T024 [US2] ⏳022 Add "Every response has the header X-Request-ID" to the conventions in
  `backend/api/openapi.yaml`. `npm run api:types` shows no diff (FR-005)

**Checkpoint**: User Story 2 testable on its own.

---

## Phase 6: User Story 1 — A developer adds a route the same way everywhere (Priority: P1) 🎯 ⏳022

**Goal**: every area on Gin, then the bridge deleted.

**Independent Test**: spec User Story 1.

**The recipe for every area** (research R10):

- handlers become `func(c *gin.Context)`;
- `r.PathValue("x")` becomes `c.Param("x")`;
- `web.ReadJSON(w, r, &in)` becomes `web.BindJSON(c, &in)`;
- `web.WriteJSON(w, s, v)` becomes `web.Respond(c, s, v)`;
- `web.WriteError(w, r, err)` becomes `web.RespondError(c, err)`;
- `Routes(r, signedIn)` becomes `RegisterRoutes(r gin.IRoutes)`, with paths relative to `/api/v1`, for
  example `r.GET("/plans/:planId", h.get)`.

Then change the area's line in `app.go` to `RegisterRoutes(signedIn)`. Its tests do not change.

- [ ] T025 [P] [US1] ⏳022 Slice C1: `backend/internal/samples/handler.go`
- [ ] T026 [P] [US1] ⏳022 Slice C1: `backend/internal/home/handler.go`
- [ ] T027 [US1] ⏳022 Slice C1: wire samples and home in `backend/internal/app/app.go` — **checkpoint C1**
- [ ] T028 [P] [US1] ⏳022 Slice C2: `backend/internal/history/handler.go`
- [ ] T029 [P] [US1] ⏳022 Slice C2: `backend/internal/lineitems/handler.go`
- [ ] T030 [P] [US1] ⏳022 Slice C2: `backend/internal/budgets/handler.go`
- [ ] T031 [US1] ⏳022 Slice C2: wiring in `app.go` — **checkpoint C2**
- [ ] T032 [P] [US1] ⏳022 Slice C3: `backend/internal/plans/handler.go`
- [ ] T033 [P] [US1] ⏳022 Slice C3: `backend/internal/campaigns/handler.go`
- [ ] T034 [US1] ⏳022 Slice C3: wiring in `app.go` — **checkpoint C3**
- [ ] T035 [P] [US1] ⏳022 Slice C4: `backend/internal/activities/handler.go`
- [ ] T036 [P] [US1] ⏳022 Slice C4: `backend/internal/approvals/handler.go`
- [ ] T037 [US1] ⏳022 Slice C4: wiring in `app.go` — **checkpoint C4**
- [ ] T038 [P] [US1] ⏳022 Slice C5: `backend/internal/people/handler.go`, `people_handler.go`,
  `teams_handler.go`
- [ ] T039 [P] [US1] ⏳022 Slice C5: `backend/internal/access/handler.go`
- [ ] T040 [US1] ⏳022 Slice C5: wiring in `app.go` — **checkpoint C5**
- [ ] T041 [US1] ⏳022 Slice C6: `backend/internal/settings/handler.go` and `handler_rates.go`. The local
  helpers `respond`, `readBody` and `actorOf` move to Gin; `readRateUpload` uses `BindJSON`, because
  `LimitBody` now sets the 3 MB limit (FR-010)
- [ ] T042 [US1] ⏳022 Slice C6: wiring in `app.go` — **checkpoint C6**
- [ ] T043 [P] [US1] ⏳022 Slice C7: `backend/internal/financials/handler.go`, and in `handler_test.go` its own
  router becomes a Gin engine
- [ ] T044 [P] [US1] ⏳022 Slice C7: `backend/internal/allwork/handler.go`
- [ ] T045 [US1] ⏳022 Slice C7: wiring in `app.go` — **checkpoint C7**
- [ ] T046 [US1] ⏳022 Slice D: delete the bridge — `web/router.go`, `web.Wrap`, `ReadJSON`, `ReadJSONUpTo`,
  `WriteJSON`, `WriteError` and `NotFound` in `web.go` — and the bridge's use in `app.go`. The build proves
  that no handler takes `http.ResponseWriter` any more (FR-013, SC-005)
- [ ] T047 [US1] ⏳022 Slice D: `golangci-lint run ./...` reports no `transport-only` or `service-layer`
  finding — **checkpoint D** with the end-to-end suite (FR-002, SC-001)

---

## Phase 7: Polish

- [ ] T048 [P] In `backend/docs/GO_STANDARDS.md`, bring §4, §5, §7, §9 and §10 into line with the code:
  helper names, the middleware order and its reason, telemetry exporters, the new variables. Drop the
  "not yet compiled" caveat for these sections (FR-014)
- [ ] T049 [P] In `CLAUDE.md`, change the API line of "Active technologies" (Gin, slog JSON, OpenTelemetry).
  In `backend/README.md` and `README.md`, update the startup record, SIGTERM and the new variables. Update
  `docs/architecture/overview.md` (FR-014)
- [ ] T050 [P] ⏳022 In `docs/sdd/feature-map.md`, link the 024 row to its folder and set its status
- [ ] T051 Run every gate: the four checkpoint commands, `govulncheck ./...` (new modules), `npm run check`
  and `npm run test:e2e`. The reviewer checks the comment budget and file length of new and changed files
- [ ] T052 Open the pull request with the requirement IDs, the publish dates of Gin, otelgin and the
  OpenTelemetry modules, and any metric exceptions

---

## Dependencies & Execution Order

- **Before 022 merges**: T001–T009 (Setup, Foundational part 1, User Story 3). None of them touches a file
  022 changes.
- **After 022 merges**: T010 onwards. The tasks that touch 022's files:

| Tasks | Files 022 also changes |
|---|---|
| T011–T015 | `internal/platform/web/web.go`, `web_test.go` |
| T016, T021, T027…T045 (wiring), T046 | `internal/app/app.go`, `app_test.go`, `access_routes_test.go` |
| T018 | `internal/auth/handler.go` |
| T019–T020 | `internal/access/middleware.go`, `targets.go`, `middleware_test.go` |
| T024 | `api/openapi.yaml` |
| T025–T044 | every `internal/*/*handler*.go`, `financials/handler_test.go` |
| T050 | `docs/sdd/feature-map.md` |

  T010, T017, T022 and T023 touch no 022 file but build on slice A or B, so they come after 022 too.
- **Order**: slice A (T010–T017), then slice B (T018–T021), then User Story 2 (T022 once A is done, T023
  once B is done), then slices C1–C7 in any order, one at a time because they share `app.go`, then
  slice D (T046–T047), then Polish.
- **Inside a slice**: the `[P]` handler tasks touch different packages; the wiring task closes the slice.

## FR coverage

FR-001 T011, T016, every checkpoint · FR-002 T002, T047 · FR-003 T013, T015, T016, T018–T021,
T025–T045 · FR-004 T011, T012, T021 · FR-005 T012, T015, T024 · FR-006 T005, T009, T013, T018–T020, T023 ·
FR-007 T012, T015 · FR-008 T006, T011, T017, T022, T023 · FR-009 T012, T015, T022 · FR-010 T012, T015,
T016, T041 · FR-011 T007–T009, T017 · FR-012 T003, T004 · FR-013 T014, T025–T046 · FR-014 T001, T048, T049
