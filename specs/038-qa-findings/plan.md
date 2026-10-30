# Implementation Plan: QA Findings — Fixes from Exploratory QA of develop@85a3d495

**Branch**: `feature/bug-fixes` (from `develop` at `85a3d495`) | **Date**: 2026-10-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/038-qa-findings/spec.md` (amends 003, 009, 011, 013, 014, 015, 032,
036, 037)

**Design artifacts**: [research.md](research.md) (R1–R19, root cause with file and line for every finding) ·
[data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (fragment) ·
[contracts/routes.md](contracts/routes.md) · [quickstart.md](quickstart.md) · [tasks.md](tasks.md).

## Summary

A bug-fix feature: 62 requirements close 53 QA findings and the two security gaps; L-26 needs only a doc
correction. No new operation, two migrations, a handful of additive contract properties, two error codes. The
approach per area (research):

1. **C-1** — the claim of a line under a pending change reserves `max(approved, changed)`: one query, one store
   mapping, one balance function (R1).
2. **H-4/H-5** — financials link/unlink reorder their checks (visible → permission → version → state), hidden =
   missing; `finance.link` requires `finance.import` (migration 00027, role validation, access check); the panel
   follows `canLink`/`canUnlink` (R2, D-359).
3. **H-2** — the workbook reads the file outside the write lock, prepares reference data once per check and calls
   import entry points of the activity and campaign services that reuse their rule functions without rebuilding
   context or overspend limits; problem values capped (R3).
4. **H-1** — dashboards fall back in the API; the page rewrites its address with the filters used (R4).
5. **H-3/L-3** — delete flows await the mutation, drop the deleted record's queries, navigate, then invalidate (R5).
6. **Security** — `auth.Attempts` limiter (R6); one header map for API, Vite preview/dev and nginx with the theme
   script's hash (R7); Referer check (R8); control characters refused while binding, IDs with them 404 (R9).
7. **M-18** — `overspend_configuration.approval_over_planned_cost` (00026); `DecideSubject` consults it; approval
   views carry `overPlannedCost` (R10, D-360).
8. Mediums and lows — local fixes listed in R11–R19.

## Technical Context

**Language/Version**: Go (pinned in `go.mod`); TypeScript 5.9 strict + React 19

**Primary Dependencies**: none new (Go `crypto/sha256` in the build helper is standard; no npm package added)

**Storage**: PostgreSQL 18; migrations `00026_overspend_approval.sql` (lane B1) and `00027_finance_link_roles.sql`
(lane B2); sqlc queries `purchase_requests.sql` (claims), `overspend.sql`, workbook batch reads

**Testing**: Go unit/service tests on fakes, handler and contract tests through `internal/platform/apitest`,
scope-isolation tests on PostgreSQL for each new scoped store read (workbook batch reads, Home owned-activity
years), fuzz test of the binder scan (B9: parsers), benchmark `BenchmarkWorkbookCheck` in `platform/apibench`;
Vitest + Testing Library + MSW (+ `fast-check` for the money input and the clipboard split); fuzz seeds for the changed rate and GSAP file readers; Playwright + axe for the
P1 stories (US1–US6) and US7

**Target Platform**: developers' machines (127.0.0.1); GitHub Actions; the compose images

**Project Type**: Web application (Go API + React web app)

**Performance Goals**: workbook check ≤ 5 s for 1,000 rows and ≤ 20 s for 5,000; ≤ 4 statements per row + 150 per
check (SC-002); sign-in limiter O(1) per attempt; each approve adds one configuration read (memoised per
transaction)

**Constraints**: every touched file within 300 non-blank lines and its 5 % comment budget (VIII: a touched file is
brought within budget); functions within the metrics; shared files touched only by the lane that owns them
(below); `backend/api/openapi.yaml` changed once, in Phase 1

**Scale/Scope**: Go packages changed: `purchaserequest`, `approval`, `overspend`, `financials`, `app`, `access`
(roles), `people`, `platform/{authz,web,record,money,errs,sheet}`, `auth`, `workbook`, `activity`, `campaign`, `plan`,
`lineitem`, `budget`, `settings`, `allwork`, `dashboard`, `spa`, `home`, `history`, `seed`; web features
`purchase-requests`, `approvals`/`activities` (approval section), `overspend`, `financials`, `dashboards`, `home`,
`all-work`, `spa`, `plans`, `line-items`, `global-budgets`, `settings`, `workbook`, `bulk`, `recycle-bin`, `people`,
`dev`, `activity-log`, shared `format`, `session`, `ui`; `frontend/{vite.config.ts,nginx.conf,index.html}`

## Constitution Check

*GATE: passed before research; re-checked after design (below).*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | 038 FR-001 – FR-062, a traceability row for each of the 54 findings, amendments noted in the earlier specs | Pass |
| II Plain language | Messages in the spec; "Warn the approver / Refuse approval"; no Hive9 terms | Pass |
| III Documented design | research R1–R19 with file:line, data model, contract fragment, routes, quickstart; D-359 – D-363 | Pass |
| IV Server source of truth | Reservation, scope, limits, approval rule, dashboard fallback and period figures all in the API; the web app shows the API's figures (the grid, SP&A rows, counts) | Pass |
| V Simplicity | One query and one function for C-1; one header map for three servers; no new package (helpers go in existing `platform/record` and `auth`) | Pass |
| VI Meaningful tests | A failing test first for every finding (tasks: each fix task depends on its test task); test names cite `038 FR-xxx` and the finding ID; contract tests for every changed operation; scope-isolation tests for the new scoped reads; e2e for US1–US7 | Pass |
| VII Security, scope isolation | Hidden = missing for financials (H-5), scope before permission/version/state; input checked at the binder; sign-in limit; CSP (F7); no personal data in logs (limiter logs the key kind only) | Pass |
| VIII Clean construction | Files within budgets; `financials/service.go` and `purchaserequest/balance.go` checked for length before adding (split by responsibility if they pass 300) | Pass |
| IX Experience | Delete → parent + Undo; consequences first (Submit changes, Approve warning); all problems at once; View only note; nothing disabled without reason | Pass |
| X Palette | Warning uses the existing warning tone; contrast fixes use `--ink` per §3.1; no new colours | Pass |
| B5 Data | Check constraint on the new column; forward-only migrations; scope filter in SQL for new reads | Pass |
| B6 Contract | Additive properties; two new error codes; behaviour changes documented in the fragment and in 036's contract | Pass |
| F7 Frontend security | CSP without `unsafe-inline`/`unsafe-eval` for scripts; theme script by hash | Pass (style-src exception only if T072 proves Radix needs it, with an ADR) |

### Threat model (STRIDE) — what changes

| Threat | Where | Response |
|---|---|---|
| Spoofing: password guessing | sign-in | FR-014 limiter per e-mail and per client; same answer for unknown e-mails |
| Tampering: link to an invisible transaction | financials link | visibility first (R2) |
| Tampering: double commitment | purchase-request change | reservation max (R1) |
| Repudiation: approval despite overage | approve | overage recorded in the decision's log entry (FR-019) |
| Information disclosure: enumeration of line items, names in conflicts | financials | hidden = missing; no name before visibility (FR-007) |
| Information disclosure: referrer leakage | web pages | `Referrer-Policy: same-origin`; API `no-referrer` |
| Denial of service: legal 5,000-row upload holding the write lock | workbook | parse outside the lock; constant work per row (FR-008, FR-009) |
| Denial of service: sign-in lockout of others | limiter | per-e-mail limit locks only that e-mail; client limit 20 per proxy-reported client |
| Elevation: clickjacking, injected script | web pages | `frame-ancestors 'none'`, `X-Frame-Options`, CSP |
| Cross-site request without Origin | API | Referer check (FR-016) |

### Risk model

| Risk | Response | Owner |
|---|---|---|
| Radix injects a `<style>` element the CSP blocks | T072 verifies on the built app; hash or style-only exception with ADR | Lane F3 |
| Theme script hash drifts (editor CRLF, comment edits) | hash computed from `index.html` at build; test fails on mismatch; `.gitattributes` LF | Lane F3 |
| Workbook import refactor changes problem messages | existing workbook tests unchanged must stay green; new entry points call the same rule functions | Lane B4 |
| Built-in role change surprises planners who linked | D-359 recorded; release note in quickstart | Product owner |
| In-memory limiter resets on restart | accepted for local phase (assumption) | Product owner |

## Lanes — parallel implementation without collisions

Each lane owns its files; a file listed for one lane is changed by no other. Phase 1 (contract, migrations,
`errs`) is done first by one person; then all lanes run in parallel.

| Lane | Scope (FR) | Owns |
|---|---|---|
| **B1 Purchase requests & approvals** | FR-001 – FR-003, FR-018 – FR-021, FR-025 (backend message), FR-027, FR-056, FR-057 | `backend/internal/purchaserequest/**`, `backend/internal/approval/**`, `backend/internal/overspend/**`, `migrations/00026_*`, `queries/purchase_requests.sql`, `queries/overspend.sql`, `app/` wiring files for these packages (`app/purchase_requests*.go`, `app/approval*.go`, `app/overspend*.go`) |
| **B2 Financials & access** | FR-004 – FR-007, FR-035 | `backend/internal/financials/**`, `app/financials.go`, `backend/internal/access/**` (role rule and its tests; no route rule changes), `platform/authz/**` (built-in grants in `grants.go`), `migrations/00027_*`, `backend/internal/people/**` (team-name call of the B3 helper) |
| **B3 Platform & security** | FR-014 – FR-017, FR-054, FR-055 | `platform/web/**`, `platform/record/**`, `backend/internal/auth/**`, `app/transport.go`, `app/config*.go`, `backend/internal/campaign/model.go` + `validate.go` (dates, names), name checks in `plan/validate.go`, `activity/fields.go`, `lineitem/validate.go`, `budget/validate.go`, name checks of Settings lists (`settings/validate*.go`, not the rate files). B1 and B2 add the helper's call in `purchaserequest` and `people` themselves |
| **B4 Imports & exports** | FR-008 – FR-010, FR-033, FR-034, FR-059 | `backend/internal/workbook/**`, `app/workbook_*.go`, `activity/import.go` (new), `campaign/import.go` (new), `platform/apibench/workbook_test.go`, `settings/ratefeed.go`, `settings/rates*.go`, `allwork/status.go` (the nginx body limit of FR-011 is changed by F3, which owns `nginx.conf`) |
| **B5 Figures & demo data** | FR-012, FR-022 – FR-024 (API), FR-032 (API), FR-049 (bin notes), FR-060, FR-061 | `backend/internal/dashboard/**`, `lineitem/display.go`, `platform/money/display.go`, `allwork/budget_by_month.go`, `backend/internal/spa/**`, `backend/internal/home/**`, `history/bin.go`, `activity/kinds.go`, `backend/seed/**` (except roles) |
| **F1 Purchase requests & approvals UI** | FR-013 (PR), FR-019 – FR-020 (UI), FR-025 – FR-028, FR-050, FR-057 | `frontend/src/features/purchase-requests/**`, `features/approvals/**`, the activity approval section files, `features/overspend/**` |
| **F2 Financials, dashboards, home, All work, SP&A** | FR-004 – FR-005 (UI), FR-012 (UI), FR-023 – FR-024 (UI), FR-032, FR-033 (UI), FR-036 (SP&A), FR-038 (dashboard hint), FR-051, FR-053 | `features/financials/**` (except `financials-text.ts`), `features/dashboards/**` (except `dashboard-text.ts`), `features/home/**` (except `financials-text.ts`), `features/all-work/**`, `features/spa/**` |
| **F3 Shared, session, security headers, dates** | FR-011, FR-015 (web), FR-031, FR-039 (shared constant), FR-044 (compact, ConfirmDialog), FR-046, FR-047, FR-058 | `frontend/src/format/**`, `src/session/**`, `src/api/errors.ts`, `src/ui/**`, `features/sign-in/**`, `features/activity-log/**`, every month-label helper of R18 (`financials-text.ts` ×2, `RecordFinancials.tsx`, `line-item-math.ts`, `phasing-math.ts`, `dashboard-text.ts`), `frontend/vite.config.ts`, `frontend/nginx.conf`, `frontend/index.html`, `frontend/security-headers.ts`, `.gitattributes`, `frontend/e2e/security.spec.ts` |
| **F4 Planning screens & Settings** | FR-013 (plan, line item), FR-029, FR-030, FR-036 (Plans), FR-037, FR-038 (split links, PR counter via `ui` handled by F3), FR-039 – FR-045, FR-048, FR-049 (UI), FR-052 | `features/plans/**` (except `RecordFinancials.tsx`, `phasing-math.ts`), `features/activities/**` (except approval section), `features/line-items/**` (except `line-item-math.ts`), `features/global-budgets/**`, `features/approval-workflows/**`, `features/settings/**`, `features/workbook/**`, `features/bulk/**`, `features/recycle-bin/**`, `features/people/**`, `features/dev/**` |
| **D Docs** | FR-062 | `docs/sdd/*`, `docs/product/*`, amendment notes in `specs/0xx` |

**Dependencies between lanes**: Phase 1 → all. F1 needs B1's view fields only through the regenerated types
(Phase 1), so F1 starts with MSW answers and runs in parallel. F2's Financials gating needs nothing beyond Phase 1.
F3's `formatMonth`/`weekdayDate` (T080) is needed by the other lanes only where they display month labels, which
F3 itself replaces; no other lane edits those helpers. End-to-end tasks (Phase 10) run after the lanes they cover.

## Design elements and touch points

| Element | Change | FR | Proving test (written first) |
|---|---|---|---|
| `purchaserequest` claims and balance | `Claim.Approved`, `RemovedByChange`; `requested` = max | FR-001 – FR-003 | `balance_test.go` `Test..._038_FR001`, `handler_change_test.go` QA sequence |
| `financials` service + `app/financials.go` | check order, hidden = missing, `RequireView` | FR-004 – FR-007 | `service_test.go`, `handler_test.go` parity |
| `access/roles.go`, `authz/grants.go`, 00027 | link needs import | FR-006 | roles service test, `grants_test.go` |
| `workbook` + `activity/import.go`, `campaign/import.go` | prepared context, parse outside lock, value cap | FR-008 – FR-010 | `apibench` statements test, `apply_test.go` unchanged problems |
| `dashboard/input.go` | fallback | FR-012 | `handler_test.go` (changed cases) |
| `auth/attempts.go` | limiter | FR-014 | `attempts_test.go`, `handler_test.go` 429 |
| `platform/web` | headers, Referer, control chars, 413, path params | FR-015 – FR-017, FR-054 | `web_test.go`, fuzz |
| `platform/record` | `CheckSingleLine` | FR-055 | `record` unit test + one per package |
| `approval/rounds_decide.go`, subjects | overage gate, permission first | FR-019 – FR-021, FR-056 | `rounds_decide_test.go`, handler tests |
| `overspend` configuration | second setting, nullable last change | FR-018, FR-057 | `service_test.go`, `handler_test.go` |
| `lineitem/display.go`, `money.Display.Parts` | grid adds up | FR-022 | `display_test.go` (new), `money/display_test.go` |
| `spa/scope.go`, `periods.go` | per-activity unscheduled, excess row | FR-023 | `periods_test.go` |
| `dashboard/months.go`, `model.go`; `allwork/budget_by_month.go` | `negativeBalanceCount` | FR-024 | `panels_test.go`, `allwork` budget-by-month test |
| `home/summary.go` + store read | year window | FR-032 | `summary_test.go`, scope-isolation test of the read |
| `allwork/status.go` | "Partly approved" | FR-033 | `export_test.go` |
| `settings/ratefeed.go`, `rates_import.go`, `rates.go` | lazy quotes, empty, all columns, too large, min date | FR-034 | `ratefeed_test.go`, `rates_import_test.go` |
| `financials/feed_csv.go`, `platform/sheet` | delimiter detection | FR-035 | `feed_csv_test.go` |
| `workbook/build.go`, `apply.go` | money as numbers, parsed compare | FR-059 | workbook export and round-trip tests |
| `seed` | created meta, FY2027 items | FR-060, FR-061 | `seed/history_test.go`, `settings_test.go`, `lineitems_test.go` |
| `history/bin.go`, `activity/kinds.go` | no duplicate notes | FR-049 | `history` bin preview test |
| Web: see tasks per lane | | | component test per state; e2e per P1 story |

## Project Structure

### Documentation (this feature)

```text
specs/038-qa-findings/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── openapi.yaml
│   └── routes.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── api/openapi.yaml                        # Phase 1 only
├── internal/
│   ├── platform/{web,record,money,errs,authz,sheet,apibench}/
│   ├── platform/database/{migrations/00026_*,00027_*,queries/}
│   ├── auth/ purchaserequest/ approval/ overspend/ financials/ people/
│   ├── workbook/ activity/ campaign/ plan/ lineitem/ budget/ settings/
│   ├── allwork/ dashboard/ spa/ home/ history/ app/
│   └── …
└── seed/
frontend/
├── index.html  vite.config.ts  nginx.conf  security-headers.ts
├── src/{format,session,api,ui}/
├── src/features/{purchase-requests,approvals,overspend,financials,dashboards,home,all-work,spa,
│                 plans,activities,line-items,global-budgets,settings,workbook,bulk,recycle-bin,
│                 people,dev,activity-log}/
└── e2e/{qa-findings-*.spec.ts,security.spec.ts}
```

**Structure Decision**: the existing web-application layout; no new package or feature folder. New files:
`auth/attempts.go`, `activity/import.go`, `campaign/import.go`, `lineitem/display_test.go`,
`platform/apibench/workbook_test.go`, `frontend/security-headers.ts` (+ test), e2e specs.

## Complexity Tracking

No constitution violation. Watch items: `financials/service.go` and `approval/rounds_decide.go` may pass 300
non-blank lines with the new checks — split by responsibility (e.g. `financials/link.go`,
`approval/rounds_overage.go`) rather than justify.
