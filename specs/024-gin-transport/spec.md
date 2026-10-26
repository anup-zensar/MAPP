# Feature Specification: Gin Transport

**Feature Branch**: `feature/024-gin-transport`

**Created**: 2026-10-26

**Status**: Draft

**Input**: User description: "Gin transport (alignment roadmap, feature map 024). Move the API's HTTP layer from
`net/http` ServeMux to Gin, confined to the handler layer, with the middleware constitution B3, B4 and B7 ask
for: panic recovery, OpenTelemetry traces and RED metrics with no exporter sending data off the machine,
request IDs, body limits, sign-in and the access route table; JSON logs carrying request_id, trace_id and
the actor's id; `http.Server` timeouts and graceful shutdown; typed configuration. No API behaviour change
beyond 022's contract."

**References**: Constitution 3.1.0 — B1, B2, B3, B4, B6, B7, Principle VII (one route table, fail closed),
Governance → Alignment · `backend/docs/GO_STANDARDS.md` §4, §5, §7, §9, §10 · Feature 014 (route table
`access/routes.go`) · Feature 022 (API v1 contract) · ADR-0003

## What changes for the team

- **Developers**: a handler is a Gin handler (`func(c *gin.Context)`), registered in its area's
  `RegisterRoutes` on the group the application hands it; its access rule stays in the one route table.
  The same steps in every package.
- **Whoever runs MAPP**: standard output carries only JSON log records; every record of a request names its
  request ID, trace ID and, once signed in, the person's ID; every answer has an `X-Request-ID` header;
  traces and request metrics exist in the process and can be printed to standard error on demand.
- **The server**: read, write and idle timeouts; Ctrl+C or SIGTERM lets requests in progress finish.
- **API clients and the web app**: nothing changes, except the new `X-Request-ID` header.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A developer adds a route the same way everywhere (Priority: P1)

As a developer (or an AI agent), when I add an operation to any business area I write a Gin handler,
register it in that area's `RegisterRoutes`, give it an access rule in `access/routes.go` and describe it in
`openapi.yaml` — the same steps in every package — and the API refuses the operation until its rule exists.

**Why this priority**: B1 and B2 make Gin the transport and keep it out of services; Principle VII requires
one route table that fails closed; one pattern everywhere is what keeps the code easy to follow (V).

**Independent Test**: add a throwaway operation to one area following the steps: it is served, needs a
session, and appears in the served-routes list the coverage tests compare with the contract; take its rule
out and it is refused with 403 `forbidden`.

**Acceptance Scenarios**:

1. **Given** the migrated API, **When** the contract, handler, access and end-to-end tests run, **Then** every
   one passes with the statuses, bodies and headers of 022's contract (plus `X-Request-ID`).
2. **Given** an operation registered without a rule in the route table, **When** a signed-in person calls it,
   **Then** the answer is 403 `forbidden` and the handler does not run.
3. **Given** an operation registered by a business area, **When** it is called without a session, **Then** the
   answer is 401 `sign_in_required`.
4. **Given** a service, store or model file that imports Gin (or a service that imports `net/http`), **When**
   the linter runs, **Then** it reports the import.
5. **Given** the move in progress, some areas on Gin and others not yet, **When** any operation is called,
   **Then** it answers as before and every test passes.
6. **Given** the last area has moved, **When** the code is built, **Then** the temporary bridge and the
   `net/http` handler helpers no longer exist.

---

### User Story 2 - Operators get request-scoped JSON logs and traces (Priority: P2)

As whoever runs or supports MAPP, when someone reports a failure with its request ID I find every log record
of that request, its trace and who made it, and no log record holds personal data.

**Why this priority**: B7; it is what makes a failure reported from the web app traceable.

**Independent Test**: sign in as Sara (Viewer), try to delete a plan (refused), call an operation that
succeeds, and read standard output.

**Acceptance Scenarios**:

1. **Given** any request, **When** it is answered, **Then** the answer has an `X-Request-ID` header made by the
   server, and an `X-Request-ID` the client sent is not used.
2. **Given** Sara's refused request, **When** the log is read, **Then** every record of that request is one JSON
   object with `request_id` (equal to the header), `trace_id` and `actor_id` `U-1006`, and none holds a name,
   an e-mail address, a cookie or a request body.
3. **Given** any request, **When** it is answered, **Then** exactly one request record gives its method, path,
   route, status and `duration_ms`.
4. **Given** a handler that panics, **When** the request is answered, **Then** the client gets 500
   `internal_error` in the error envelope, the panic is logged once with its stack, `request_id` and
   `trace_id`, and the request's trace and metrics record status 500.
5. **Given** any served operation, **When** it is called, **Then** one server span named
   `<METHOD> <route>` is recorded and the request's duration is measured with its route and status.
6. **Given** `OTEL_TRACES_EXPORTER=console` and `OTEL_METRICS_EXPORTER=console`, **When** requests are served,
   **Then** spans and metrics are written to standard error; with the default `none` they are written
   nowhere; a `traceparent` header sent by a client does not set the trace.

---

### User Story 3 - The server shuts down gracefully (Priority: P3)

As whoever runs MAPP, when I stop the API (Ctrl+C, or SIGTERM from a script or, later, a container platform),
requests in progress finish, telemetry is flushed and the process exits cleanly; a slow client cannot hold a
connection open forever.

**Why this priority**: B3; today only Ctrl+C is handled and only the header read has a timeout.

**Independent Test**: start the API, start a slow request, send SIGTERM; start the API with invalid settings.

**Acceptance Scenarios**:

1. **Given** a request in progress, **When** SIGINT or SIGTERM arrives, **Then** no new request is accepted, the
   request completes, and the process exits 0 within `SHUTDOWN_TIMEOUT` (default 10 s).
2. **Given** a request still running when `SHUTDOWN_TIMEOUT` passes, **Then** the process exits with an error
   that says the shutdown deadline passed.
3. **Given** `LOG_LEVEL=loud`, `SHUTDOWN_TIMEOUT=forever` and `OTEL_TRACES_EXPORTER=otlp`, **When** the API
   starts, **Then** it stops with one message naming all three variables and their allowed values.

### Edge Cases

- A path with a trailing slash, or a known path with another method, answers 404 `not_found` — no redirect,
  no 405 — as on 022.
- A HEAD request answers 404: ServeMux served GET handlers for HEAD; the contract has no HEAD operation and
  the web app sends none.
- An unknown path outside `/api/v1` answers the JSON 404 envelope (022 answered plain text there).
- A panic after the answer has started adds nothing to the answer and is logged once.
- A body over its limit answers 400 `bad_request` with today's message (GO_STANDARDS' 413 would change
  behaviour; follow-up).
- Two routes with different parameter names at the same position stop the API at start; the 232 operations
  of today's route table have none.
- `X-Forwarded-For` is not trusted: the client address recorded in traces is the connection's.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every operation MUST answer exactly as on 022's contract — paths, statuses, bodies, the
  `Cache-Control: no-store` and `nosniff` headers, the JSON 404 for unknown paths and methods, the 400
  refusal of other hosts and sites — and every answer MUST also carry `X-Request-ID`, documented in the
  contract.
- **FR-002** (B1, B2): HTTP MUST be handled by Gin, imported only by the transport: handler and middleware
  files, `internal/platform/web`, `internal/app`, `cmd/api` and tests. Services, stores and models MUST NOT
  import Gin, and services MUST NOT import `net/http`; a lint rule MUST enforce both.
- **FR-003** (VII): Every area MUST register its routes in `RegisterRoutes` on the group the application gives
  it; every operation MUST keep its access rule in the one route table, keyed as in the contract
  (`GET /api/v1/plans/{planId}`); an operation without a rule MUST be refused with 403 `forbidden`.
- **FR-004** (B4, B7, VII): Every request MUST pass these steps in this order: tracing and metrics; request ID;
  request log; panic recovery; API headers; same-host and same-site check; body limit; then, on routes that
  need a session, sign-in and the access rule.
- **FR-005** (B7): Every answer MUST carry `X-Request-ID`, a random ID made by the server; an ID sent by the
  client MUST be ignored.
- **FR-006** (B7, VII): Logs MUST be JSON, one object per line on standard output, written with `log/slog`.
  Every record written while serving a request MUST carry `request_id`, `trace_id` and, once signed in,
  `actor_id`. Tokens, passwords, cookies, request bodies, names and e-mail addresses MUST NOT be logged.
- **FR-007** (B7): Each request MUST produce exactly one request record with method, path, route, status and
  `duration_ms`.
- **FR-008** (B7): Every request MUST be traced with OpenTelemetry (one server span per request, named by
  method and route), and request rate, error rate and duration MUST be recorded per route as metrics.
  Exporters MUST be `none` (default) or `console` (standard error), chosen with `OTEL_TRACES_EXPORTER` and
  `OTEL_METRICS_EXPORTER`; no setting may send telemetry off the machine. Trace context sent by a client
  MUST NOT be accepted.
- **FR-009** (B4): A panic in a handler or middleware MUST be answered 500 `internal_error` in the error
  envelope (unless the answer has started), logged once with its stack and the request's attributes, and
  recorded as an error in the request's span and metrics.
- **FR-010** (B6): A request body MUST be limited before the handler reads it: 1 MB, and 3 MB for the two
  GSAP rate uploads (`POST /api/v1/exchange-rates/import-check` and `POST /api/v1/exchange-rates/import`);
  a larger body answers as today.
- **FR-011** (B3): The server MUST listen on `127.0.0.1` only, with read-header (5 s), read (15 s), write
  (30 s) and idle (60 s) timeouts. On SIGINT or SIGTERM it MUST stop accepting requests, let the requests in
  progress finish within `SHUTDOWN_TIMEOUT`, flush telemetry and exit 0; a missed deadline MUST end the
  process with an error.
- **FR-012** (B7): `LOG_LEVEL` (debug, info, warn, error; default info), `SHUTDOWN_TIMEOUT` (1 s to 60 s;
  default 10 s), `OTEL_TRACES_EXPORTER` and `OTEL_METRICS_EXPORTER` (none, console; default none) MUST be
  parsed into the typed configuration and validated at startup together with today's variables, every
  problem reported in one message; an invalid value MUST stop the process.
- **FR-013** (Governance → Alignment): The move MUST happen area by area. Until an area moves, its routes MUST
  work unchanged through a temporary bridge, and every step MUST leave all tests green; the bridge and the
  `net/http` handler helpers MUST be removed when the last area has moved.
- **FR-014** (III, B1): An ADR MUST record Gin and OpenTelemetry, superseding ADR-0003's "standard-library
  HTTP server"; `GO_STANDARDS.md`'s transport, logging and lifecycle samples MUST match the code; `CLAUDE.md`
  and the READMEs MUST describe the new stack.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After every migration step, all Go tests (contract tests included) and the end-to-end suite pass
  with no change to an expected status, body or API header.
- **SC-002**: Of the log records the User Story 2 requests produce, 100% are valid JSON with `request_id` and
  `trace_id`, 100% of those after sign-in carry `actor_id`, and 0 contain a name, an e-mail address or a
  cookie.
- **SC-003**: 100% of served operations produce a span and a duration measurement labelled with their route.
- **SC-004**: With a request in flight, SIGTERM ends the process with exit code 0 after the request completes
  and within `SHUTDOWN_TIMEOUT`.
- **SC-005**: After the last step, 0 non-test files outside the transport import Gin and `web.Router` no longer
  exists.

## Decisions

- **No behaviour change**: the move is proved by today's tests passing unchanged; `X-Request-ID` is the only
  addition, documented in the contract's conventions like `Cache-Control`.
- **Recovery sits inside tracing, the request ID and the request log** (research R5), not outermost as
  GO_STANDARDS' sample shows: only there does a panic keep its `trace_id` and `request_id` in the log and
  count as a 500 in traces and metrics. The guide is corrected.
- **Exporters `none` and `console` only** (research R8): standard output is the log stream, so the console
  exporter writes to standard error; export to a collector comes with hosting, by ADR.
- **The route table stays in the contract's form** (research R4), so it still reads line by line against
  `openapi.yaml`.

## Assumptions

- Feature 022 is merged first (`/api/v1`, error envelope, 404 for records outside scope, `/healthz`,
  `/readyz`, `/api/v1/environment`); tasks that touch its files wait for it (see `tasks.md`).
- Feature 023 (PostgreSQL) is independent: tracing database calls (`otelpgx`) and closing the pool belong to
  it and use the providers this feature creates; whichever lands second rebases the wiring and `/readyz`.
- The API makes no outbound calls today (GSAP feeds are simulated by file import), so nothing outbound is
  traced.
- Showing the request ID in the web app (F4) is web-app work (feature 026); this feature only sends it.
- Out of scope: OTLP export and trace propagation (hosting ADR), 413 for oversized bodies, a pagination
  envelope, package renames (025).
