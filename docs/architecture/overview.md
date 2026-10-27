# Architecture Overview

MAPP is deliberately small: **one React web app and one Go API**. This page explains how the parts
fit together. Exact library versions are fixed in feature 001's `plan.md` and `research.md`.

```
 Browser                                   Developer laptop (now)          Azure (later)
┌──────────────────────────┐ /api/v1/*  ┌──────────────────────────────┐
│ React + TypeScript app   │──────────►│ Go API (one program)          │
│  pages · components ·    │  JSON      │  handler → service → store    │
│  generated API types     │◄──────────│  demo sign-in · activity log  │
└──────────────────────────┘           └──────────────┬───────────────┘
                                                       │ store interfaces
                                          ┌────────────▼────────────┐
                                          │ PostgreSQL 18 (local)    │──► Azure Database for
                                          │ pgx · sqlc · goose       │    PostgreSQL (later)
                                          └──────────────────────────┘
```

## 1. Web app (`frontend/`)

- React + TypeScript (strict), built with Vite; a single-page app with client-side routing.
- Talks to the API only through a small API client whose types are **generated from the OpenAPI file**.
- Holds no business rules beyond quick form checks; it shows what the API calculates (roll-ups,
  balances, statuses, completeness).
- Follows `docs/product/design-system.md`; shared components live in one folder and every feature reuses
  them.
- In development the Vite server proxies `/api` to the Go API, so both run on `127.0.0.1` with no CORS.

## 2. API (`backend/`)

- Go, one program (`cmd/api`), Gin in the handler layer only (feature 024, ADR-0011).
- One package per business area, each with the same files and layers:

| Layer | Responsibility | Example |
|---|---|---|
| **Handler** | Read the HTTP request, call the service, write JSON. No business rules. | `POST /api/v1/plans` |
| **Service** | All business rules: validation, calculations, status changes, activity log. Receives the actor. | `plan.Service.Create(ctx, actor, input)` |
| **Store** | Read and write records. The service declares the `Store` it needs (`store.go`); the PostgreSQL store satisfies it. | `plan.Store` → `plan.PostgresStore` |

- Shared platform code (small and boring, `internal/platform/`): the database pool, migrations and
  generated queries; the Gin engine and its middleware; the error format; the actor and authorisation;
  readable IDs; record bookkeeping (version, created/updated); clock (so tests can fix "today"); money
  (exact decimals); logging and telemetry; and the in-memory fakes the tests run on. The full list is in
  `backend/docs/GO_STANDARDS.md` §2.
- The activity log and recycle bin core (`internal/history`) is created by feature 001 and used by every
  feature; feature 011 adds its screens.
- **Server is authoritative:** totals, remaining budgets, unassigned balance, spreads, currency
  conversion, goals, statuses and completeness are always computed in services and returned to the web
  app.

### API conventions (detailed in feature 001's contract)

- REST + JSON under `/api/v1`; resource names in plural (`/api/v1/plans/{id}/campaigns`). The probes
  `/healthz` (liveness) and `/readyz` (readiness, checks the store) sit outside it (feature 022).
- One OpenAPI file in the repository is the source of truth; each feature adds its endpoints to it.
- One error format for every failure, the envelope `{"error": {"code", "message", "fields"}}`: a message
  for the user plus a list of field problems, so the web app can show **all** validation errors at once.
- A record outside the person's access answers 404 `not_found`, exactly like a missing record; 403
  `forbidden` only refuses an action the person's roles do not allow on a record they can see.
- Lists support search, filters and paging; responses include totals where the screen needs them.
- Every write returns the updated record, including recalculated values and status.
- Optimistic concurrency: records carry a version; a stale save is refused with a clear message instead
  of silently overwriting someone else's change.

## 3. Storage

- **PostgreSQL 18** in every environment (ADR-0013): a native install or the optional `compose.yaml` on
  a developer's machine, a service in CI, Azure Database for PostgreSQL later. Each area's
  `store_postgres.go` implements its store interface with sqlc-generated queries over the schema of
  the goose migrations; constraints guard every invariant the database can check, and the reads of
  plan-owned records the person sees filter by their scope in SQL.
- Work runs in transactions (each service's `Transactor`): all changes of one request — for example a delete, its
  recycle-bin entry and its activity-log entry — are saved together or not at all. Writes run one at a
  time; reads see one consistent snapshot and never wait.
- Seed data: `go run ./cmd/seed` creates and migrates the database and loads the demo dataset into it
  (see `docs/product/demo-users-and-data.md`).
- Service, handler and contract tests run on in-memory fakes of the stores; feature 029 deleted the JSON
  file store of feature 001 (ADR-0004).

## 4. Sign-in and the actor

- **Now (decision D-004):** a demo sign-in page. The user picks a seeded demo user (or types the e-mail
  and the shared demo password). The API issues a session in an HttpOnly cookie (only a hash of the
  token is stored; sessions end after 12 hours without activity); middleware turns it into the **actor**
  (id, name) that services use for owners, approvals and the activity log.
- **Roles and access (feature 014, ADR-0008):** five roles with team scopes. Every action needs its
  permission in the plan's scope, enforced by the API through a fail-closed route table
  (`internal/access`) and scoped service checks. The scope comes only from the signed-in actor, every
  query on plan-owned data filters by it, and a record outside it answers 404 like a missing one.
  Nobody approves an activity they own or submitted (segregation of duties).
- **Later:** Microsoft Entra ID single sign-on replaces demo sign-in (Shell's target architecture, see
  `docs/reference/README.md`). Demo sign-in is disabled outside local/dev. Roles and access are **not**
  waiting on it; only the sign-in page and the middleware that produces the actor change.

## 5. Quality

| Test type | Where | Covers |
|---|---|---|
| Unit tests (Go) | next to each service | every business rule and calculation |
| Contract tests (Go) | each area and `internal/app`, through `internal/platform/apitest` | every endpoint matches the OpenAPI file |
| Scope-isolation tests (Go) | next to each store | every scoped store read, on a real PostgreSQL database |
| Component tests (web) | next to components | shared components and complex forms |
| Property-based tests (web) | next to components | invariants over generated input (`fast-check`) |
| End-to-end tests (browser) | `frontend/e2e/` | every P1 user story, against the real API with seed data |
| Accessibility | component tests and `frontend/e2e/` | WCAG 2.2 AA through axe |

Every test but the end-to-end and scope-isolation ones runs on in-memory fakes. New code reaches 80%
decision coverage (SEMS, constitution VI).

## 6. Target architecture (for later, not built now)

Shell's final architecture (`Application_and_Azure_Deployment_Architecture_Enterprise_Updated_v3.pdf`,
summarised in `docs/reference/shell-target-architecture.md`): Azure Front Door + WAF → React app and API
Management → Go API on Azure Container Apps → Azure Database for PostgreSQL; a separate Go GSAP worker
exchanging CSV files with GSAP through Blob SFTP and SAP Cloud Integration; Entra ID + SAM, Key Vault,
Dynatrace; GitHub Actions delivery under Shell TCM and SEMS. MAPP's two-part design grows into that
without rewrites.
