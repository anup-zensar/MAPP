# Architecture Overview

NorthStar is deliberately small: **one React web app and one Go API**. This page explains how the parts
fit together. Exact library versions are fixed in feature 001's `plan.md` and `research.md`.

```
 Browser                                   Developer laptop (now)          Azure (later)
┌──────────────────────────┐   /api/*   ┌──────────────────────────────┐
│ React + TypeScript app   │──────────►│ Go API (one program)          │
│  pages · components ·    │  JSON      │  handler → service → repo     │
│  generated API types     │◄──────────│  demo sign-in · activity log  │
└──────────────────────────┘           └──────────────┬───────────────┘
                                                       │ repository interfaces
                                          ┌────────────▼────────────┐
                                          │ JSON file store (now)    │──► PostgreSQL (later)
                                          │ backend/data/*.json      │
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

- Go, one program (`cmd/api`), standard-library HTTP server.
- One package per business area, each with the same files and layers:

| Layer | Responsibility | Example |
|---|---|---|
| **Handler** | Read the HTTP request, call the service, write JSON. No business rules. | `POST /api/plans` |
| **Service** | All business rules: validation, calculations, status changes, activity log. Receives the actor. | `plans.Service.Create(ctx, actor, input)` |
| **Repository** | Read and write records. An interface plus one implementation per storage. | `plans.Repository` → JSON store now, PostgreSQL later |

- Shared platform code (small and boring): JSON helpers and the error format, ID generation, money
  (exact decimals), clock (so tests can fix "today"), the JSON file store, demo sign-in middleware.
- **Server is authoritative:** totals, remaining budgets, unassigned balance, spreads, currency
  conversion, goals, statuses and completeness are always computed in services and returned to the web
  app.

### API conventions (detailed in feature 001's contract)

- REST + JSON under `/api`; resource names in plural (`/api/plans/{id}/campaigns`).
- One OpenAPI file in the repository is the source of truth; each feature adds its endpoints to it.
- One error format for every failure: a message for the user plus a list of field problems, so the web
  app can show **all** validation errors at once.
- Lists support search, filters and paging; responses include totals where the screen needs them.
- Every write returns the updated record, including recalculated values and status.
- Optimistic concurrency: records carry a version; a stale save is refused with a clear message instead
  of silently overwriting someone else's change.

## 3. Storage

- **Now:** a local JSON file store (one file per record type in `backend/data/`, git-ignored), written
  safely (write to a temporary file, then rename) and guarded by a lock so concurrent requests cannot
  corrupt it.
- **Later:** PostgreSQL. Each repository gets a PostgreSQL implementation; services and the web app do not
  change. To make that easy, records are relational in shape: each level is its own record type with a
  parent id, no deep nesting, ids that never change.
- Seed data: a command loads the demo dataset into an empty data folder (see
  `docs/product/demo-users-and-data.md`).

## 4. Sign-in and the actor

- **Now (decision D-004):** a demo sign-in page. The user picks a seeded demo user (or types the e-mail
  and the shared demo password). The API issues a session token; every request carries it; middleware
  turns it into the **actor** (id, name) that services use for owners, approvals and the activity log.
- There are **no roles or permissions**: every signed-in user can do everything. Services still receive
  the actor, so adding a permission check later is a change in one place.
- **Later:** Microsoft Entra ID single sign-on and role-based access (Shell's target architecture, see
  `docs/reference/README.md`). Demo sign-in is disabled outside local/dev.

## 5. Quality

| Test type | Where | Covers |
|---|---|---|
| Unit tests (Go) | next to each service | every business rule and calculation |
| Contract tests (Go) | API package | every endpoint matches the OpenAPI file |
| Component tests (web) | next to components | shared components and complex forms |
| End-to-end tests (browser) | `frontend/e2e/` | every P1 user story, against the real API with seed data |

## 6. Target architecture (for later, not built now)

Shell's final architecture (`Hive9_Final_Architecture.pptx`, summarised in
`docs/reference/shell-target-architecture.md`): Azure Front Door + WAF → React app and API Management →
Go API on Azure Container Apps → Azure Database for PostgreSQL; a separate Go GSAP worker exchanging CSV
files with GSAP through Blob SFTP and SAP Cloud Integration; Entra ID, Key Vault, Azure Monitor; GitHub
Actions delivery under Shell SEMS. NorthStar's two-part design grows into that without rewrites.
