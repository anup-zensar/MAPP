# Implementation Plan: SWBS Codes from a Simulated GSAP

**Branch**: `030-simulated-gsap-swbs` (no branch created: the product owner asked for no commits or
branches) | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/030-simulated-gsap-swbs/spec.md`

## Summary

A line item stores the SWBS code GSAP returns and when it was requested and received; its derived SWBS
status gains **received**, which wins over every other status, so a code survives the activity leaving
approval and locks "Request SWBS code from GSAP" on. A new business area, package `internal/swbs`, owns
SWBS requests: it selects the line items that are ready to send in the actor's `finance.import` scope,
sends them through its adapter interface `swbs.GSAP` — implemented now by `swbs.Simulated`, which numbers
`SIM-<company code|0000>-<nnnn>` per company code from the codes already issued — stores the codes on the
line items through a port implemented in `internal/app` over package `lineitem`, keeps the run as a
record (`SWR-1001`, …) and writes one activity-log entry. `POST` and `GET /api/v1/simulated-gsap/swbs-runs`
expose it. The web app shows a Simulated GSAP panel on the Financials page, an SWBS request card with the
code, dates, source, HX key (copyable) and "Send to simulated GSAP now" on the line item page, and the new
status everywhere the status shows. The seed records one demo run that gives Venue hire its code.

## Technical Context

**Language/Version**: Go (pinned in `backend/go.mod`); TypeScript 5.9 with React 19

**Primary Dependencies**: Gin (handler only), pgx/v5 + sqlc, goose; TanStack Query, React Router, Radix
(through `src/ui`), lucide-react. No new dependency.

**Storage**: PostgreSQL 18 — migration `00017_swbs.sql`: three columns and four constraints on
`line_items`, tables `swbs_runs` and `swbs_run_codes`

**Testing**: Go `testing` + go-cmp on fakes (`fake.DB`), contract tests through `internal/platform/apitest`;
Vitest + Testing Library + user-event + MSW (`reply()`) + vitest-axe; Playwright + axe on the real API and
PostgreSQL (`mapp_e2e_030`, ports 18130/14130)

**Target Platform**: local laptop (API on 127.0.0.1), evergreen browsers

**Project Type**: web application (Go API + React web app)

**Performance Goals**: a run over the demo-sized data answers in well under one second; reads one level of
records per step (no query per row)

**Constraints**: runs are serialised by the database's write lock (`database.Write`), so two runs never
issue the same number; constitution metrics (≤ 4 parameters, complexity ≤ 8, ≤ 300 lines a file, ≤ 5 %
comments a file touched)

**Scale/Scope**: up to a few thousand line items a run; history paged at 20 (at most 100)

## Constitution Check

*GATE: passed before Phase 0 and re-checked after Phase 1.*

| Principle | How this plan meets it |
|---|---|
| I Spec-first | Every behaviour traces to spec FR-001 – FR-021; test names cite them (`030 FR-007`). |
| II Shell business | Shell vocabulary; "Simulated GSAP" on everything the simulation produces (FR-020); GSAP still owns the code: MAPP never lets a person type one (FR-002); the simulation sits behind an adapter (ADR-0016). |
| III Documented design | This plan, research, data model, contracts, quickstart; STRIDE and risk model below; design elements mapped to packages and feature folders. No new ADR: ADR-0016 already decides the adapter. |
| IV Two parts, server truth | Status, readiness, numbering and every count are computed by the API; the web app shows them. `swbs.GSAP` is an adapter interface owned by the package that needs it. |
| V Simplicity | One new package with the standard five files; one new feature folder; no new dependency; the simulation is ~40 lines. |
| VI Meaningful tests | Unit tests for status, numbering, readiness, locking, copy; contract tests for both operations; scope isolation: the run reads activities through `ActivityStore.ListInScope`, whose PostgreSQL scope test exists (`TestPostgresStoreActivityAndLineItemScopeFR009`) — no new scoped store read; component tests per view state with axe; e2e for every P1 story. |
| VII Security | Route rule `need(finance.import)` plus the service check; scope from the actor only (`authz.PermissionScopeOf`); a named line item outside the scope answers 404; body validated (≤ 200 IDs, each ≤ 40 characters); one log entry per run. |
| VIII Clean construction | Every file touched is brought within the comment budget (list under "Shared files"); no TODO. |
| IX Experience | One yellow primary per page kept (the panel's action is secondary); confirmation states consequences; unavailable action says why; locked switch says why; status as word + colour. |
| X Palette | Only existing tokens and status tones. |
| B5 data | Constraints for every invariant: code iff received time, requested ≤ received, request kept on while a code exists, codes unique. Money untouched. |

No violation: Complexity Tracking is empty.

## Project Structure

### Documentation (this feature)

```text
specs/030-simulated-gsap-swbs/
├── spec.md · checklists/requirements.md
├── plan.md · research.md · data-model.md · quickstart.md
├── contracts/swbs-runs.openapi.yaml · contracts/routes.md
└── tasks.md
```

### Source code

```text
backend/internal/platform/database/migrations/00017_swbs.sql      new
backend/internal/platform/database/queries/swbs_runs.sql          new; line_items.sql changed
backend/internal/swbs/                                            new package (business area "SWBS requests")
  model.go · gsap.go · simulated.go · ports.go · store.go · store_postgres.go · service.go · demo.go
  kinds.go · handler.go · texts.go · swbstest/store.go · tests
backend/internal/lineitem/   swbs.go (new: Swbs, SwbsStatusOf, HXKey, ReceiveSwbsCodes) · model.go ·
  view.go · update.go · duplicate.go · completeness.go · store_postgres.go
backend/internal/app/swbs.go (new: the port over line items and activities, the wiring) · areas.go ·
  history_access.go
backend/internal/access/routes_gsap.go (new) · routes.go
backend/internal/stores/swbs.go (new) · stores.go · postgres.go · storestest/storestest.go
backend/internal/platform/ids/ids.go · backend/internal/allwork/status.go
backend/internal/financials/feed_demo.go (demo codes) · seed/swbs.go (new) · seed/financials.go
backend/api/openapi.yaml
frontend/src/features/simulated-gsap/   new feature folder: index.ts · api.ts · SimulatedGsapPanel.tsx ·
  SwbsRunList.tsx · SendNowButton.tsx · simulated-gsap-text.ts · *.module.css · tests
frontend/src/features/line-items/   SwbsCard.tsx · HxKey.tsx · swbs-text.ts (new) · SwbsChip.tsx ·
  LineItemPage.tsx · LineItemForm.tsx · LineItemFormCards.tsx · FinancialsSection.tsx ·
  gsap-financials.ts · line-item-text.ts
frontend/src/features/financials/FinancialsPage.tsx
frontend/src/api/generated/schema.d.ts (regenerated) · src/testing/fixtures/*
frontend/e2e/simulated-gsap-swbs.spec.ts
docs/product/field-catalogue.md §5 · domain-model.md §2.4, §4.6 · glossary.md
```

**Structure Decision**: the web-application layout of constitution B2 and F2. SWBS requests are their own
business area (`internal/swbs`, `features/simulated-gsap`) because runs are records of their own with a
lifecycle unlike a line item's; the line item keeps what belongs to it (its code, dates, status, HX key)
in package `lineitem`.

## Backend design

### Package `lineitem` (owns the line item's SWBS fields and status)

- `Swbs` (embedded in `LineItem` and `View`): `SwbsCode string`, `SwbsRequestedAt, SwbsReceivedAt
  *time.Time`; JSON `swbsCode`, `swbsRequestedAt`, `swbsReceivedAt`. `View` adds `HXKey string` (`hxKey`).
- `SwbsStatusOf(li, f) SwbsStatus`: received (code set) → notRequested (flag off) → readyToSend (activity
  stored status `approved`) → waitingForApproval. `SwbsReceived = "received"`.
- `(LineItem) HXKey() string`: `"HX" + strings.TrimPrefix(li.ID, "L-")` (L-1003 → HX1003), the key
  `financials/match.go` already matches.
- `ReceiveSwbsCodes(ctx, repo Store, receipt SwbsReceipt) error` (`SwbsReceipt{By, At, Codes}`): inside the
  caller's Write; for each line item sets code and both times to `At`, `Touch(By, At)`, `Put`. A line item
  that is gone or already has a code is an error (the run then saves nothing).
- `Update`: the stored `Swbs` is carried to the saved line item; switching the flag off while a code exists
  is a field problem on `requestSwbs` (FR-004), reported with every other problem.
- `copyOf` (duplicate, activity copy, plan copy) clears `Swbs` (FR-005).

### Package `swbs` (owns SWBS requests and runs)

| File | Contents |
|---|---|
| `model.go` | `Run` (ID, Source, Codes, `record.Meta`), `Code` (LineItemID, LineItemName, CompanyCode, SwbsCode), `Source` (`simulatedGsap`, `demoData`), `RunInput`, `RunView`, `RunPage` |
| `gsap.go` | `Request{LineItemID, LineItemName, CompanyCode}`, `Answer{LineItemID, SwbsCode}`, `GSAP interface { RequestCodes(ctx, []Request) ([]Answer, error) }` |
| `simulated.go` | `Simulated{issued Issued}`, `Issued interface { CountCodes(ctx, prefix) (int, error) }` (codes already given that start `SIM-<cc>-`); numbers per company code, `0000` for none; the company code kept to ten letters and digits |
| `ports.go` | `Candidate{Request; Ready bool}`, `LineItems interface { Candidates(ctx, authz.Scope, ids []string) ([]Candidate, error); Receiver }`, `Receiver interface { Receive(ctx, actor.Actor, []Answer, time.Time) error }` |
| `store.go` | `Transactor`; `Store { AddRun; GetRun; DeleteRun; ListRuns(ctx, limit) ([]Run, int, error); CountCodes }` |
| `store_postgres.go` | sqlc-backed `PostgresStore` |
| `service.go` | `Service.Run(ctx, a, RunInput) (RunView, error)`, `Service.Runs(ctx, a, limit) (RunPage, error)` |
| `demo.go` | `LoadDemo(ctx, repo, generator, DemoLoad) (Run, error)` — the seed's run through the same `send` |
| `kinds.go` | history kind `swbsRun` ("SWBS run", top level; Snapshot, Remove, Put) |
| `handler.go` | `POST` and `GET /simulated-gsap/swbs-runs` |
| `texts.go` | messages and log texts |

**Run** (one `Write`): require `finance.import` → validate input → `items.Candidates(scope, ids)` with
`scope = authz.PermissionScopeOf(ctx, authz.FinanceImport)` → named IDs not found → 404 `not_found`; named
but not ready → 409 `not_allowed` naming them → ready requests in ID order → `gsap.RequestCodes` → one
answer per request (otherwise an internal error) → `items.Receive(answers, now)` → run `SWR-n`
(`ids.PrefixSwbsRun`) with `record.NewMeta(a, now)` → `repo.AddRun` → `history.Log` (kind `swbsRun`,
action created, verb "ran", name "Simulated GSAP", details listing the codes or "Nothing was ready to
send"). Any error rolls the whole Write back.

**Runs**: require `finance.import`; Read: `ListRuns(limit)` newest first with the total; `Candidates(scope,
nil)` counted for `readyToSend`.

### Port implementation (`internal/app/swbs.go`)

`swbsLineItems{activities stores.ActivityStore, lineItems stores.LineItemStore, settings
stores.SettingsStore}`: `Candidates` reads the activities in scope (`ListInScope`, the SQL scope filter),
their line items (`ListByActivities`, one query), keeps the named IDs, works out each one's status with
`lineitem.SwbsStatusOf` and its company code (the activity-level attribute labelled "Company code", the
chosen option's label; one read of attributes and options). `Receive` calls `lineitem.ReceiveSwbsCodes`.
`swbsArea()` builds the service with `swbs.NewSimulated(store)`, registers the kind and mounts the
handler.

### Access (`access/routes_gsap.go`, Table C)

| Operation | Rule |
|---|---|
| `GET /api/v1/simulated-gsap/swbs-runs` | `need(finance.import)` |
| `POST /api/v1/simulated-gsap/swbs-runs` | `need(finance.import)` |

Activity log (`app/history_access.go`): kind `swbsRun` joins the GSAP imports (D-304): visible with
`finance.import` or `activityLog.viewAll`; restoring (never offered) would need `finance.import`.

### All work

`allwork/status.go`: `received` → "SWBS code received", tone done, lifecycle order 3.

### Demo data (`seed/swbs.go`, called first in `seed/financials.go`)

`swbs.LoadDemo` sends Venue hire (L-1003, company code DE03 from its activity) through the simulated GSAP
as MAPP at the seed time, source `demoData`: run `SWR-1001`, code `SIM-DE03-0001`; no log entry (D-188).
`financials/feed_demo.go`: the demo transactions of Venue hire carry `SIM-DE03-0001`, those of Stand build
`SIM-GB01-0001` (the code the first run gives it).

## Frontend design

- `features/simulated-gsap` (new design element "Simulated GSAP"): `api.ts` — `useSwbsRuns()` (key
  `['simulated-gsap', 'swbs-runs']`), `useSendSwbsRequests()` (POST; on settled invalidates
  `['simulated-gsap']`, `['line-items']`, `['all-work']`, `['financials']`, `['activity-log']`, `['home']`);
  `SimulatedGsapPanel` — the Financials page card: ready count, secondary action with `Send` icon,
  reason when nothing is ready, `ConfirmDialog`, result, `SwbsRunList` (5 latest, "Showing 5 of n");
  `SendNowButton` — the line item page's action with its confirmation and refusal message.
  `index.ts` exports `SimulatedGsapPanel` and `SendNowButton`.
- `features/line-items`: `SwbsCard` (status chip + explanation, details list: SWBS code, Source, Requested,
  Received, `HxKey`; `SendNowButton` when ready and the person holds `finance.import`); `HxKey` (read-only
  key, Copy button, toast "HX key copied", a failure toast that says to select and copy it);
  `swbs-text.ts` holds every SWBS text (moved out of `line-item-text.ts`); `SwbsChip` gains received;
  the form locks the switch with the reason and shows the code; the Financials section drops the SWBS
  line (FR-019).
- `FinancialsPage` renders `SimulatedGsapPanel` after the transactions for people with `finance.import`.
- States: the panel renders loading (skeleton), error (ErrorState + retry), forbidden (not shown — the
  page shows it only with `finance.import`; a 403 still renders the no-access sentence), empty history
  ("No runs yet"), success; the card renders each status.
- **JavaScript budget**: no new dependency; the feature folder builds into its own chunk of 4.7 kB (1.9 kB
  gzipped), loaded with the Financials and line-item pages; the initial route is unchanged (the repository
  records no initial-route budget yet).

## Threat model (STRIDE) — new operations and the adapter

| Threat | Where | Response |
|---|---|---|
| Spoofing | Both operations | Session cookie of demo sign-in; the actor comes from the session only. |
| Tampering | `lineItemIds` | Validated (≤ 200, each 1–40 characters); IDs are only looked up among the line items in scope; codes are never accepted from a client. |
| Repudiation | Runs | Each run is a stored record with who and when, plus one activity-log entry; line items record who touched them last. |
| Information disclosure | Run history lists line-item names and codes of every plan | Needs `finance.import`, which applies to All markets by definition (014 Table A); a named line item outside the scope answers 404 like a missing one. If `finance.import` ever becomes scoped, the history must filter by scope — recorded as risk R3. |
| Denial of service | A run touches every ready line item | Writes are serialised; a run is bounded by the line items in scope; the list is paged (≤ 100); request bodies are size-limited by the middleware. |
| Elevation of privilege | Running without the permission | Route rule `need(finance.import)` (fail closed) plus the service's `authz.Require`; contract tests prove 403. |

No cryptography, browser storage or third-party origin is added. The clipboard write is a user gesture
and holds a non-secret key.

## Risk model

| # | Risk | Response | Owner |
|---|---|---|---|
| R1 | A simulated code is taken for a real one | Every code starts `SIM-`; screens and log say "Simulated GSAP" | Product owner (accepted, ADR-0016) |
| R2 | Live GSAP behaves differently (24–48 h, rejections, formats) | Adapter interface; statuses and exceptions for live added with the integration | Product owner (accepted) |
| R3 | `finance.import` becomes a scoped permission | History would then need a scope filter; noted in the threat model | Technical lead |
| R4 | The demo transactions carry `SIM-GB01-0001` before Stand build receives it | True only if Stand build is the first GB01 line item sent, which the demo's first run does | Product owner (accepted, spec Clarifications) |
| R5 | Comment-budget clean-ups in shared files conflict with other features at integration | Edits kept local; the orchestrator merges | Orchestrator |

## Business rules and the tests that prove them

| Rule | Test |
|---|---|
| Status order received > notRequested > readyToSend > waitingForApproval (FR-001) | `lineitem` `TestSwbsStatusOf_030_FR001` (table) |
| HX key from the ID (FR-016) | `lineitem` `TestHXKey_030_FR016` |
| Receive stores code and times, touches; refuses a second code, saving nothing (FR-002, FR-003) | `lineitem` `TestReceiveSwbsCodes_StoresCodeAndTimes_030_FR002`, `TestReceiveSwbsCodes_RefusesASecondCode_030_FR003` |
| Update keeps the code; switching off refused with the message (FR-003, FR-004) | `lineitem` `TestUpdate_KeepsSwbsCode_030_FR004`, `TestUpdate_RequestOffWithoutCode_030_FR004` |
| Code kept when the activity leaves approval, on move and on delete + undo (FR-003) | `lineitem` `TestSwbsStatus_ActivityLeavesApproval_030_FR003`, `TestMoveAndRestore_KeepSwbsCode_030_FR003` |
| Duplicate and copies drop the code (FR-005) | `lineitem` `TestDuplicate_DropsSwbsCode_030_FR005`, `TestCopyLineItems_DropsSwbsCode_030_FR005` |
| Numbering per company code, 0000, continues across runs, ID order (FR-007) | `swbs` `TestSimulated_030_FR007` (table), `TestRun_NumberingGoesOnAcrossRuns_030_FR007` |
| Run sends ready line items in scope, stores codes, keeps the run (FR-006, FR-009) | `swbs` `TestRun_SendsReady_030_FR006`, `TestRun_ReceiveFails_030_FR006`; contract `TestSwbsRun_Contract_030_FR006` |
| One log entry per run (FR-010) | `swbs` `TestRun_LogsOnce_030_FR010` |
| Named IDs: unknown → 404, not ready → 409, nothing saved (FR-008) | `swbs` `TestRun_Named_030_FR008` (table); contract `TestSwbsRun_Named_Contract_030_FR008`, `TestSwbsRun_OneNamed_Contract_030_FR015` |
| Empty run kept and logged (FR-009, FR-010) | `swbs` `TestRun_NothingReady_030_FR009` |
| Permission refused (FR-012) | `swbs` `TestRun_Forbidden_030_FR012`, contract `TestSwbsRuns_Forbidden_030_FR012` |
| Input validation | `swbs` `TestRun_Validation_030_FR008`, `TestRuns_Limit_030_FR011` |
| History newest first, ready count (FR-011) | `swbs` `TestRuns_030_FR011`, contract `TestSwbsRuns_Contract_030_FR011` |
| Log visibility (FR-010) | `app` `TestHistoryAccess_SwbsRun_030_FR010` |
| Received chip in All work (FR-017) | `allwork` `TestSwbsChip_030_FR017` |
| Demo run (FR-021) | `swbs` `TestLoadDemo_030_FR021`; contract `TestSwbsRuns_Contract_030_FR011`, `TestSwbsCode_LineItemSeed_030_FR021` |
| Company code read from the activity; nothing outside the scope (FR-007) | `app` `TestSwbsCandidates_030_FR007` |

## Shared files this feature changes

`backend/api/openapi.yaml` (two operations, four schemas, `SwbsStatus` + `LineItem` fields, tag) ·
`backend/internal/access/routes.go` (one area) · `backend/internal/app/areas.go`,
`app/history_access.go` · `backend/internal/platform/ids/ids.go` (`SWR`) · `backend/internal/stores/*` ·
`backend/seed/financials.go`, new `seed/swbs.go` · `backend/internal/financials/feed_demo.go` and four of its
tests (demo codes: `import_test.go`, `transactions_test.go`, `record_test.go`, `handler_test.go`) ·
`backend/internal/allwork/status.go` · `backend/internal/settings/fieldhelp_catalogue.go` (the SWBS code's
default help text) · `frontend/src/api/generated/schema.d.ts` · `frontend/src/features/financials/FinancialsPage.tsx`
and `routes.test.tsx` · `frontend/src/testing/fixtures/{line-items,financial-transactions,financials-page}.ts(x)` ·
`docs/product/{field-catalogue,domain-model,glossary}.md` (owned sections). Each touched file is brought
within the comment budget.

## Complexity Tracking

None.
