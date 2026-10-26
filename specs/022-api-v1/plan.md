# Implementation Plan: API v1

**Branch**: `feature/022-api-v1` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/022-api-v1/spec.md`

## Summary

Align the API contract with constitution B6, B7 and Principle VII on today's stack. Paths move from `/api/` to
`/api/v1/` everywhere (contract, router patterns, route table, web app, tests). `web.WriteError` wraps the
existing `errs.Error` in `{"error": …}`; the contract's `Error` schema becomes the envelope around a new
`ErrorDetail`. The access middleware and the services' visibility checks answer `errs.NotFound` instead of
`errs.Forbidden` for a record the person cannot see. `/healthz` and `/readyz` replace `/api/health`, whose
`environment` and `demoSignIn` move to `GET /api/v1/environment`.

## Technical Context

**Language/Version**: Go 1.27, TypeScript 5.9

**Primary Dependencies**: unchanged — `net/http`, JSON file store; React, TanStack Query, `openapi-fetch`

**Storage**: JSON file store; one new read-only method `(*store.DB).Ready` for `/readyz`

**Testing**: existing Go handler, contract and access tests updated to the new contract; Vitest mocks and
Playwright helpers updated; new tests only for new rules (out-of-scope 404, `/readyz` 503)

**Target Platform**: local (`127.0.0.1`)

**Project Type**: Web application (Go API + React web app)

**Constraints**: no new dependency, no new tooling; Vite's `/api` proxy already covers `/api/v1`

**Scale/Scope**: 174 contract paths; about 100 Go and 350 web files change their paths mechanically

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | Spec FR-001–FR-009, branch `feature/022-api-v1`, test names carry 022 FR IDs | Pass |
| IV Two parts | The server still decides; the web app only reads the envelope | Pass |
| V Simplicity | Mechanical path change; envelope added in one function; no new layer | Pass |
| VI Meaningful tests | Existing tests follow the contract; new tests only for the 404 rule and `/readyz` | Pass |
| VII Scope isolation | Out-of-scope records answer 404 like missing ones (FR-005); 403 for roles only | Pass |
| VIII Clean construction | New files ≤ 300 lines, comments ≤ 5% | Pass |
| B6 API contract | `/api/v1` paths, one envelope, contract updated with the code | Pass (pagination envelope and `Idempotency-Key` are follow-ups) |
| B7 Observability | `/healthz` without dependencies, `/readyz` checks the store | Pass |
| F4 Errors | `frontend/src/api/client.ts` is still the one place that reads error bodies | Pass |
| Governance → Alignment | Only the contract is aligned; `net/http`, JSON store and folders stay | Pass |

**Post-design re-check**: unchanged — Pass. No Complexity Tracking entries.

## Design

- **Paths**: every `/api/` path in the contract, `Routes(...)` patterns, `access/routes.go`, Go tests, web
  calls, mocks and end-to-end helpers becomes `/api/v1/`. The router's JSON 404 catch-all stays on `/api/`,
  so old paths answer the envelope.
- **Envelope**: `web.WriteError` writes `errorEnvelope{Error: apiErr}`; `WriteJSON`'s encoding fallback does
  the same. `errs.Error` keeps its fields. Contract: `Error` = `{error: ErrorDetail}`; `ErrorDetail` is the
  former `Error`. Web: `isErrorBody` checks `body.error.code`; `ApiError.fromBody` takes the inner object.
- **Scope isolation**: `authz.NotFoundMessage(type)` replaces `NoAccessMessage`; `authz.RequireView`,
  `RequireViewActivity` and the middleware's `checkTarget` answer `errs.NotFound`. History checks visibility
  right after reading a bin entry (`unresolvedEntry`, `putBack`); `historyAccess.CanRestore` answers 404 for
  an entry it cannot see. Financials maps an out-of-scope transaction or record to its not-found message;
  budgets treats an allocation of an invisible plan as missing.
- **Probes**: `app.go` registers `GET /healthz` (204) and `GET /readyz` (`db.Ready()`: 204, else
  `errs.Unavailable()` = 503 `internal_error`); `GET /api/v1/environment` answers `{environment, demoSignIn}`.
  All three are public in the route table. The seed asks `/healthz` for 204; Playwright waits on `/healthz`.

## Project Structure

```text
backend/api/openapi.yaml                  paths, Error/ErrorDetail, probes, Environment
backend/internal/platform/{web,errs,authz,store}/
backend/internal/{access,app,history,financials,budgets}/   scope isolation, probes, route rules
backend/cmd/seed/main.go                  /healthz
backend/internal/**/handler.go, *_test.go /api/v1 paths
frontend/src/api/{client,errors}.ts       envelope; schema.d.ts regenerated
frontend/src/**, frontend/e2e/**          /api/v1 paths, mocked error bodies
frontend/playwright.config.ts             /healthz
```

## Follow-ups (not built)

Pagination envelope for lists; `Idempotency-Key`; `fields` as a map if the product owner wants it;
body-referenced records outside scope (move/copy destinations, GSAP link targets) answered as missing;
aligned wording of not-found messages for a missing parent; Gin, PostgreSQL, OpenTelemetry (023, 024).
