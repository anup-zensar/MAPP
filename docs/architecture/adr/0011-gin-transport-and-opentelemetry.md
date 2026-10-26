# ADR-0011: Gin transport and OpenTelemetry

**Status:** Accepted · **Date:** 2026-10-26 · **Supersedes:** ADR-0003's "standard-library HTTP server"
(one layered program stays)

## Context
Constitution 3.1.0 makes Gin the HTTP transport, confined to the handler layer (B1, B2), and asks for
OpenTelemetry traces and RED metrics with no exporter sending data off the machine, JSON logs with the
request ID, trace ID and actor's id in every record, `http.Server` timeouts with graceful shutdown, and a
recovery middleware (B3, B4, B7). The API serves 232 operations through `net/http` ServeMux with one
`web.Middleware`. Feature 024 specifies the move (`specs/024-gin-transport/`).

## Decision
- **Gin v1.12.0** handles HTTP. It is imported only by handler and middleware files,
  `internal/platform/web`, `internal/app` and `cmd/api`; the depguard rule `transport-only` enforces this.
  Services keep importing neither Gin nor `net/http`. Handlers use MAPP's own helpers (`web.BindJSON`,
  `web.Respond`, `web.RespondError`), not Gin's binding or `c.JSON`.
- **One route table stays**: `access/routes.go`, keyed in the contract's form
  (`GET /api/v1/plans/{planId}`), fails closed. Business areas register on a signed-in group; only the
  application and sign-in register public routes.
- **Middleware order**:
  1. otelgin
  2. request ID
  3. request log
  4. recovery
  5. API headers
  6. same-host and same-site check
  7. body limit
  8. sign-in, on the signed-in group
  9. access, on the signed-in group

  Recovery sits inside tracing, the request ID and the request log. otelgin restores the outer context when
  it returns and records status only when the chain returns normally, so a recovery outside it would lose
  the panic's trace and request IDs and its 500 in traces and metrics.
- **OpenTelemetry v1.47.0 with otelgin v0.72.0**. The SDK providers are always on, so trace IDs reach the
  logs. They are passed explicitly, never set as globals. The exporters are `none` (default) or `console`
  (standard error, since standard output is the JSON log); any other value stops the API at startup.
  Trace context sent by clients is ignored.
- **Logs**: `log/slog` JSON on standard output through `internal/platform/logging`. The server has
  read-header, read, write and idle timeouts and drains on SIGINT or SIGTERM within `SHUTDOWN_TIMEOUT`.

## Consequences
- One way to add a route in every package. The move happens area by area through a temporary bridge that
  feature 024 deletes when the last area has moved.
- Gin compiles in several indirect modules MAPP does not use (QUIC, YAML, TOML, BSON, protobuf); the
  vulnerability and composition-analysis gates cover them.
- HEAD requests answer 404 (ServeMux served GET handlers for them). A trailing slash and a wrong method
  still answer the JSON 404.
- Follow-ups: OTLP export and trace propagation, by ADR, when hosting is chosen; database tracing with
  PostgreSQL (feature 023).
