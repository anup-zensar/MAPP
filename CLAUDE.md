# CLAUDE.md — guidance for AI agents working on NorthStar

Read this with `.specify/memory/constitution.md` (the rules) before doing anything.

## Project

NorthStar is Shell's marketing planning application replacing the Hive9 Plan module. This phase covers
only the **Plan module**: demo sign-in, application shell, home dashboard, settings, people and teams,
plans, campaigns, initiatives, activities, line items, activity approvals, planning views, activity log
and recycle bin. Spec-driven development with GitHub Spec Kit.

## Non-negotiables

1. **Spec first.** Only build behaviour described in `specs/NNN-short-name/spec.md`. Use the Spec Kit
   skills: `/speckit-specify`, `/speckit-clarify`, `/speckit-plan`, `/speckit-checklist`,
   `/speckit-tasks`, `/speckit-analyze`, `/speckit-implement`, `/speckit-converge`.
2. **Keep it simple and readable** (constitution VII): standard library first, few well-known libraries,
   the same folder pattern everywhere, comments that explain why. A beginner must be able to follow it.
3. **Two parts only:** React + TypeScript in `frontend/`, Go in `backend/`. Go packages per business area,
   each handler → service → repository. All business rules in services.
4. **Storage via repository interfaces**; the JSON file store now, PostgreSQL later. Keep records
   relational in shape.
5. **Contract first:** endpoints are described in `backend/api/openapi.yaml`; the web app's API types
   are generated from it.
6. **Demo sign-in, role- and team-based access** (D-004, D-223, ADR-0008): sign-in stays a demo until
   single sign-on; every action needs its permission (014 Table A) in the plan's scope, enforced by the
   API through the route table (fail closed, 403 `forbidden`) and scoped service checks. Give every new
   operation its route rule. Still pass the actor to every service call and write the activity log.
7. **Tests:** unit tests for every rule and calculation, a test for every acceptance scenario, an
   end-to-end test for every P1 story, contract tests for every endpoint. Deterministic (fixed clock and
   seed).
8. **Money** = exact decimal + currency (base USD). Dates are calendar dates; timestamps in UTC.
9. **Shell vocabulary** on screen: Plan → Campaign → Initiative → Activity → Line item. Never Program or
   Tactic; never Hive9 interface terms (see `docs/product/design-system.md` §8).
10. **Shell enterprise design, beginner-friendly:** follow `docs/product/design-system.md` (version 2;
    D-310–D-314) — light grey page with white cards, at most one Shell-yellow primary button per page or
    dialog, red only for danger and errors, Jost headings and Arial body, no Shell pecten, WCAG 2.2 AA; the
    standard CRUD pattern, status and completeness chips on every record, all validation errors at once,
    delete to the recycle bin with Undo, nothing greyed out without a reason.
11. **Out of scope:** the Finance budgets module, purchase requests, accrual calculation, Measure, live
    GSAP integration, e-mail, real sign-in (SSO), AI. Do not build them. GSAP rates and transactions come
    only from simulated feeds behind adapters (ADR-0009).
12. **Local only:** servers listen on `127.0.0.1`; no cloud services; no secrets in the repository.

## Key documents

| Need | File |
|---|---|
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

- Branch per feature: `NNN-short-name`, the same as its folder under `specs/`.
- Conventional Commits with the feature number and requirement IDs, e.g. `feat(004): copy plan (FR-012)`.
- Test names mention the requirement ID they prove.
- Shell environment: Windows; Spec Kit scripts are PowerShell (`.specify/scripts/powershell/`).

## Active technologies

Fixed by feature 001 (`specs/001-app-shell-and-sign-in/research.md` explains each choice).

- **API (`backend/`)**: Go 1.27, standard library only at run time (`net/http` ServeMux patterns,
  `encoding/json`, `log/slog`). Module name `northstar`. Packages: `internal/app` (wiring, config),
  `internal/platform/{actor,clock,errs,web,store,ids,record,apitest}`, then one package per business area
  (`auth`, `people`, `history`, `samples`, …), each `model.go · repository.go · store_json.go ·
  service.go · handler.go` plus tests.
- **Storage**: JSON file store (`internal/platform/store`) with all-or-nothing transactions carried in
  the `context.Context`; services depend on `store.Transactor` and repository interfaces only.
- **Contract**: `backend/api/openapi.yaml` (OpenAPI 3.1). Contract tests use
  `github.com/pb33f/libopenapi-validator` through `internal/platform/apitest`; handler tests are external
  test packages (`package <name>_test`).
- **Web app (`frontend/`)**: React 19 + TypeScript 5.9 (strict), Vite 8, React Router 8 (data router
  without loaders or actions: `createBrowserRouter` + `RouterProvider`), TanStack Query 5, Radix primitives (`radix-ui`), lucide-react icons, CSS Modules +
  `src/styles/tokens.css`, `openapi-fetch` with types generated by `openapi-typescript` into
  `src/api/schema.d.ts` (never edit by hand). Import alias `@/` = `frontend/src`.
- **Tests**: Go `testing` + `httptest`; Vitest 5 + Testing Library + jsdom; Playwright 1.64 + axe
  (`frontend/e2e/`).
- **Commands**: backend `go run ./cmd/seed`, `go run ./cmd/api`, `go vet ./...`, `go test ./...`;
  frontend `npm run dev`, `npm run check`, `npm run test:e2e`, `npm run api:types` after changing the
  contract.
- **Every new record type**: embed `record.Meta` (version + created/updated), check the version on
  update (`version_conflict`), write one activity-log entry per change with `history.Log` inside the same
  `Write`, delete through `history.MoveToBin`, register a `history.Kind`.
