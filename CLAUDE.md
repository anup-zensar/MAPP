# CLAUDE.md — guidance for AI agents working on MAPP

Read this with `.specify/memory/constitution.md` (the rules, version 2.1) before doing anything. Where
this file and the constitution disagree, the constitution wins and this file is corrected. Working in
`backend/`? `backend/CLAUDE.md` loads the Go standards (`backend/docs/GO_STANDARDS.md`).

## Project

MAPP is Shell's marketing planning application replacing the Hive9 Plan module. This phase covers
only the **Plan module**: demo sign-in, application shell, home dashboard, settings, people and teams,
plans, campaigns, initiatives, activities, line items, activity approvals, planning views, activity log
and recycle bin. Spec-driven development with GitHub Spec Kit. The constitution applies Shell's
Software Design & Construction Policy and SEMS, which are mandatory.

## Non-negotiables

1. **Spec first.** Only build behaviour described in `specs/NNN-short-name/spec.md`. Use the Spec Kit
   skills: `/speckit-specify`, `/speckit-clarify`, `/speckit-plan`, `/speckit-checklist`,
   `/speckit-tasks`, `/speckit-analyze`, `/speckit-implement`, `/speckit-converge`.
2. **Keep it simple and readable** (constitution V): the approved stack only (B1, F1), the same folder
   pattern everywhere, functions within the quality metrics (at most 4 parameters, complexity 8,
   nesting 4, fan-out 7, 40 statements, 5 terms per expression). A beginner must be able to follow it.
3. **Comments are rare** (constitution VIII): at most 5% of a file's non-blank lines, rounded up; only
   *why*, a constraint or non-obvious behaviour; no doc comment that repeats a name. Requirement IDs go
   in test names and pull requests, not in comments. Tool directives don't count.
4. **Two parts only:** React + TypeScript in `frontend/`, Go in `backend/`. Go packages per business
   area, each handler → service → store; Gin only in handlers; all business rules in services. The
   server is the source of truth: the web app never recomputes a figure the API provides.
5. **Storage:** PostgreSQL in every environment — pgx pool, sqlc queries, goose migrations — behind
   store interfaces. Locally it runs natively or through the optional `compose.yaml`; no test or
   command may require Docker. Keep the schema relational, with constraints for every invariant.
6. **Contract first:** endpoints are described in `backend/api/openapi.yaml` (OpenAPI 3.1, `/api/v1`,
   one error envelope); the web app's API types are generated from it.
7. **Demo sign-in, role- and team-based access** (D-004, D-223, ADR-0008): sign-in stays a demo until
   single sign-on; every action needs its permission (014 Table A) in the plan's scope, enforced by the
   API through the route table (fail closed) and scoped service checks. MAPP is for Shell only, so
   scopes play the part of tenants: the scope comes only from the signed-in actor, every query on
   plan-owned data filters by it, a record outside it answers 404 like a missing one, and 403
   `forbidden` refuses only an action on a record the actor can see. Give every new operation its route
   rule. Pass the actor to every service call and write the activity log.
8. **Meaningful tests only** (constitution VI). Required: unit tests for every rule and calculation, a
   test for every acceptance scenario, an end-to-end test for every P1 story, a contract test for every
   endpoint, a scope-isolation test for every scoped store read, a test for each state a data view
   renders, and a failing test first for every bug. Never write tests that restate the code, test the
   framework or generated code, only check that a mock was called, repeat another test, snapshot the DOM,
   or exist only for coverage. New code reaches 80% decision coverage (SEMS). Deterministic: fixed clock
   and seed.
9. **Money** = exact decimal + currency (base USD), never a float; stored as integer minor units.
   Dates are calendar dates; timestamps in UTC.
10. **Shell vocabulary** on screen: Plan → Campaign → Initiative → Activity → Line item. Never Program or
    Tactic; never Hive9 interface terms (see `docs/product/design-system.md` §8).
11. **Tasteful, intuitive experience** (constitution IX): modern and calm, clear visual hierarchy (the
    title is the most prominent text, at most one yellow primary action per page or dialog), and no
    surprises — a control does exactly what it says; Enter submits, Escape closes without saving, Back
    keeps filters; consequences are shown before they happen. The standard CRUD pattern, status and
    completeness chips on every record, all validation errors at once, delete to the recycle bin with
    Undo, nothing greyed out without a reason, WCAG 2.2 AA. Jost headings, Arial body, no Shell pecten.
12. **Shell colour palette, non-negotiable** (constitution X): every colour comes from a token with
    Shell's exact values — Primary Yellow `#FFC600` (hover `#FFCB14`, press `#EBB600`), Secondary Red
    `#DD1D21`, and the light and dark theme tokens listed there. Yellow means "act here" or "you are
    here"; red only for danger and errors; respect the contrast limits listed with the palette.
    `docs/product/design-system.md` and `tokens.css` still hold older values (for example `#FFC800`)
    until they are aligned; the constitution's values win.
13. **Out of scope:** the Finance budgets module, purchase requests, accrual calculation, Measure, live
    GSAP integration, e-mail, real sign-in (SSO), AI. Do not build them. GSAP rates and transactions come
    only from simulated feeds behind adapters (ADR-0009).
14. **Local only:** servers listen on `127.0.0.1`; no cloud services at run time; Docker only optionally,
    for PostgreSQL; no secrets in the repository.
15. **Alignment** (constitution, Governance): code written before constitution 2.0.0 is aligned by
    dedicated features later. Until a package or screen is aligned, a change to it keeps its current
    stack and structure — don't migrate it as a side effect of another change. New packages, screens and
    files follow the constitution in full.
16. **File length** (constitution V): at most 300 non-blank lines per source file, tests included; 301–400
    needs a written justification; 401+ is never merged. Split by responsibility.

## Key documents

| Need | File |
|---|---|
| The rules | `.specify/memory/constitution.md` |
| Go standards, with examples | `backend/docs/GO_STANDARDS.md` |
| Go lint rules | `backend/.golangci.yml` |
| Features, order, scope boundaries | `docs/sdd/feature-map.md` |
| Decisions and open questions | `docs/sdd/decision-log.md` |
| Process, branches, Definition of Done | `docs/sdd/sdd-process.md` |
| Records, statuses, money rules | `docs/product/domain-model.md` |
| Fields per level | `docs/product/field-catalogue.md` |
| Screens and wording | `docs/product/design-system.md` |
| Terms | `docs/product/glossary.md` |
| Demo users and seed story | `docs/product/demo-users-and-data.md` |
| Architecture, folders, ADRs | `docs/architecture/` |
| Source material (read-only) | `docs/reference/` (precedence in its README) |

## Conventions

- Branches: `main` (production), `release` (stabilisation), `develop` (integration). `plan-module` is
  the Plan module's branch and merges into `develop` when the module is ready. Feature branches are
  `feature/NNN-short-name`, matching `specs/NNN-short-name/`, and start from the module branch.
- Never push directly to `main`, `release` or `develop`. Every change reaches them through a reviewed
  pull request from a working branch (`feature/NNN-short-name`, or `docs/…` / `fix/…` for work without a
  spec).
- Conventional Commits with the feature number and requirement IDs, e.g. `feat(004): copy plan (FR-012)`.
- Test names mention the requirement ID they prove.
- Shell environment: Windows; Spec Kit scripts are PowerShell (`.specify/scripts/powershell/`).

## Active technologies

### Target (constitution 2.x) — new code, and each package or screen once aligned

- **API (`backend/`)**: Go (pinned in `go.mod`), Gin in the handler layer only, PostgreSQL 18 through
  `pgx/v5`, sqlc, goose, `log/slog` JSON, OpenTelemetry, `/healthz` and `/readyz`. Module name `mapp`.
  Layout in `backend/docs/GO_STANDARDS.md` §2: `cmd/api`, `cmd/seed`, `internal/app`,
  `internal/platform/{actor,authz,clock,errs,web,database,logging,telemetry,…}`, then one package per
  business area with `model.go · store.go · store_postgres.go · service.go · handler.go` plus tests.
  Lint: `backend/.golangci.yml`. Configuration: `DATABASE_URL`, and `TEST_DATABASE_URL` for store tests.
- **Web app (`frontend/`)**: React 19 + TypeScript 5.9 (strict, F5 options), Vite, React Router (data
  router without loaders or actions), TanStack Query, React Hook Form + Zod, Radix primitives,
  lucide-react, CSS Modules with the design tokens in `src/ui/`, `openapi-fetch` with types generated
  by `openapi-typescript` into `src/api/generated/` (never edit by hand). Folders as in constitution F2.
  Import alias `@/` = `frontend/src`.
- **Tests**: Go `testing` + `httptest` + `go-cmp`, contract tests with `libopenapi-validator` through
  `internal/platform/apitest`, store tests on a fresh PostgreSQL database per test; Vitest + Testing
  Library + `user-event` + MSW + `vitest-axe` + `fast-check`; Playwright + axe against the real API and
  PostgreSQL.
- **Commands**: `docker compose up -d postgres` (only if you run PostgreSQL in Docker); backend
  `golangci-lint fmt ./...`, `golangci-lint run ./...`, `sqlc generate && sqlc vet`, `go test -race ./...`
  (`-short` skips store tests), `govulncheck ./...`, `go run ./cmd/seed`, `go run ./cmd/api`; frontend
  `npm run dev`, `npm run typecheck`, `npm run lint`, `npm run format:check`, `npm run test:coverage`,
  `npm run test:e2e`, `npm run api:types` after changing the contract.

### Current code (until aligned)

Fixed by feature 001 (`specs/001-app-shell-and-sign-in/research.md` explains each choice).

- **API**: standard library at run time (`net/http` ServeMux patterns, `encoding/json`, `log/slog`).
  Packages: `internal/app` (wiring, config), `internal/platform/{actor,clock,errs,web,store,ids,record,apitest}`,
  then one package per business area (`auth`, `people`, `history`, `samples`, …), each `model.go ·
  repository.go · store_json.go · service.go · handler.go` plus tests.
- **Storage**: JSON file store (`internal/platform/store`) with all-or-nothing transactions carried in
  the `context.Context`; services depend on `store.Transactor` and repository interfaces only.
- **Contract**: paths under `/api/`; contract tests use `libopenapi-validator` through
  `internal/platform/apitest`; handler tests are external test packages (`package <name>_test`).
- **Web app**: as in the target, but without React Hook Form, Zod, MSW or `fast-check`; folders
  `components/`, `lib/`, `styles/` (`src/styles/tokens.css`) and `test/`; generated types in
  `src/api/schema.d.ts`.
- **Commands**: backend `go run ./cmd/seed`, `go run ./cmd/api`, `go vet ./...`, `go test ./...`;
  frontend `npm run dev`, `npm run check`, `npm run test:e2e`, `npm run api:types`.

### Every new record type (both)

Embed `record.Meta` (version + created/updated), check the version on update (`version_conflict`), write
one activity-log entry per change with `history.Log` inside the same `Write` transaction, delete through
`history.MoveToBin`, register a `history.Kind`.
