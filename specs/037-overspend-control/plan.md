# Implementation Plan: Overspend Control — Warn or Refuse per Budget

**Branch**: worktree branch of feature 037 (merged into `plan-module` by the orchestrator) | **Date**: 2026-10-28 |
**Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/037-overspend-control/spec.md` (amends 004, 005, 014, 032)

**Design artifacts**: [research.md](research.md) (R1–R8) · [data-model.md](data-model.md) ·
[contracts/openapi.yaml](contracts/openapi.yaml) (fragment) · [contracts/routes.md](contracts/routes.md) ·
[quickstart.md](quickstart.md) · [tasks.md](tasks.md).

## Summary

1. **The setting** (FR-001 – FR-007). Plans, campaigns, initiatives and global budgets get a stored
   `overspend` mode (`warn` · `refuse`, migration 00025). Their create and update inputs take it (nil keeps
   it; on create nil takes the organisation default), their views show it with whether the person may change
   it (owner or `globalBudgets.manage`, D-348), and their history kinds log it. An allocation sets the plan's
   mode through `plan.Service.SetBudgetFromAllocation` (still the one write path of an allocated plan's
   budget). Settings › Overspend control is a singleton like Settings › Purchase requests.
2. **Enforcement** (FR-008 – FR-013). New package `internal/overspend`: a guarded transactor wraps the
   transaction of the plans, campaigns, activities, budgets and history services; its outermost `Write`
   snapshots every budget with its planned cost, runs the work, snapshots again and refuses with 409
   `over_budget` (R2–R4). Nothing else in those services changes for the check.
3. **Before saving** (FR-014 – FR-016). The activity form answer gains `overspendLimits`; the web app's
   activity form and budget forms check the typed amounts against them and the figures they already have,
   and show the API's `over_budget` message in their error summaries.
4. **Purchase-request gate** (FR-017 – FR-019, D-349). `LineItemFacts.ActivityOverBy` and one more problem
   in `purchaserequest.gates`.

## Technical Context

**Language/Version**: Go (pinned in `go.mod`); TypeScript 5.9 strict + React 19

**Primary Dependencies**: none new

**Storage**: PostgreSQL; migration `00025_overspend.sql` (four columns, one table); sqlc queries for the
columns, the snapshot reads and the configuration

**Testing**: all in `backend/internal/overspend`: unit tests of the rule, the guard and the access check on
fakes (`rules_test.go`, `guard_test.go`, `access_test.go`), and contract-validated API tests through `apitest`
(fakes) of every write path and setting (`enforcement_test.go`, `paths_test.go`, `budgets_test.go`,
`limits_test.go`, `handler_test.go`); the purchase-request gate in `backend/internal/purchaserequest`
(`overspent_gate_test.go`, `handler_overspent_test.go`). No scope-isolation test: the guard's budget read is
unscoped by design, a recorded deviation (Complexity Tracking, D-355, ADR-0019). Vitest + Testing Library +
MSW for each form state; Playwright `frontend/e2e/overspend-control.spec.ts` for the two P1 stories (US1, US5)

**Target Platform**: developers' machines (127.0.0.1); GitHub Actions

**Project Type**: Web application (Go API + React web app)

**Performance Goals**: each planning write adds four aggregate reads (R3); measured with `apibench`
(`BenchmarkOverspend`, T029: an activity create under a campaign set to Refuse, allowed and refused). On
2026-10-29, demo data, a loaded developer machine: allowed 51 statements, 335 ms; refused 43 statements (rolled
back before the answer), 250 ms

**Constraints**: every touched file within 300 non-blank lines and its 5 % comment budget; functions within
the metrics; shared files touched minimally (openapi.yaml, schema.d.ts regenerated, app wiring, seed,
decision log, feature map, domain model)

**Scale/Scope**: 1 new Go package; 5 changed Go packages (`plan`, `campaign`, `activity`, `budget`,
`purchaserequest`) + `app`, `access`, `stores` (`errs` unchanged: the code is `overspend.CodeOverBudget`); 4 web feature folders (`plans`, `plans/campaigns`,
`global-budgets`, `activities`) + a Settings page

## Constitution Check

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | 037 FR-001 – FR-020, acceptance scenarios per story; amendments listed and noted in 032 | Pass |
| II Plain language | "When planned cost goes over this budget: Warn / Refuse"; no Hive9 words ("hard limit" is not shown) | Pass |
| III Documented design | plan, research R1–R8, data model, contract fragment and route table, quickstart | Pass |
| IV Server source of truth | The rule runs in `overspend.Guard` inside the write transaction; the web app repeats it only for feedback, from figures the API gives | Pass |
| V Simplicity | One guard instead of twenty call sites (R3); one new package following the handler → service → store layout | Pass |
| VI Meaningful tests | Unit tests of each rule and edge case; a contract-validated API test per write path that changes planned cost or a budget; e2e for US1 and US5; no scope-isolation test for the unscoped guard read (Complexity Tracking) | Pass |
| VII Security, scope isolation | New routes in `access/routes_overspend.go`; the mode change needs owner or budget manager (403); a refusal names only budgets the save reaches; the guard's budget read is not filtered by the actor's scope | Pass with 1 recorded deviation (Complexity Tracking, D-355, ADR-0019) |
| VIII Clean construction | Files within budgets; requirement IDs in test names | Pass |
| IX Experience | Consequences shown before saving (activity form, budget forms, allocation dialog, Settings note); refusal shown in place; read-only choice says why | Pass |
| X Palette | Error red only for the refusal; the "Refuses overspending" tag uses the neutral status style | Pass |
| B5 Data | Check constraints on the new columns; singleton key check; `ListOverspendBudgets` and `ListOverspendGlobalBudgets` read plan-owned data without the scope filter | Pass with the same recorded deviation (Complexity Tracking) |
| B6 Contract | Additive: optional inputs, new view properties, new error code and property | Pass |

### Threat model (STRIDE)

| Threat | Where | Response |
|---|---|---|
| Spoofing / elevation: a planner turns a budget back to Warn | plan, campaign, initiative updates | `overspend.RequireChange`: owner or `globalBudgets.manage` in scope, else 403 and nothing saved; logged with the actor |
| Tampering: a write path that skips the check | every planning write | The guard wraps the transactor of every service that changes planned cost or budgets; new paths in those services are covered by construction |
| Repudiation | setting changes | Logged in the same transaction (FR-005) |
| Information disclosure: a refusal naming a budget the person cannot see | global budget overage | A refusal names only budgets the save itself reaches; a planner whose save would overspend a global budget learns its name and figures — accepted (the plan page already names its allocation's global budget and amount, 014 FR-031) |
| Denial of service | snapshot reads per write | Aggregate reads over indexed keys; writes are already serialised |

### Risk model

| Risk | Response | Owner |
|---|---|---|
| Shell expected over-allocation to be blocked too | Q-024; one more rule in the same guard if confirmed | Product owner |
| Shell expected the default to be inherited | Q-025; a third mode value if confirmed | Product owner |
| Exchange-rate changes leave a global budget over | By design (FR-012); next increase refused | Product owner |
| Parallel features (012 bulk, import) add write paths | Covered when they use the guarded services | Orchestrator |

## Design elements and touch points

| Element (module) | Change | FR |
|---|---|---|
| `backend/internal/platform/database/migrations/00025_overspend.sql` | columns, table | FR-001, FR-006 |
| `backend/internal/overspend` (new) | `model.go` (Mode, Control, Budget, Overage, Limit, Configuration), `rules.go` (refusal rule, overage, limits), `guard.go` (guarded transactor), `access.go` (ControlOf, RequireChange), `service.go` (configuration, DefaultMode, Limits), `kinds.go`, `handler.go`, `store.go`, `store_postgres.go`, `overspendtest/` (fake) | FR-001 – FR-016 |
| `backend/internal/overspend/rules.go` | `CodeOverBudget` and `Refusal(...)`: a 409 `errs.Error` with the message (no change to `errs`) | FR-010 |
| `backend/internal/plan` | `Plan.Overspend`, input, view, defaults, check, log label; allocation sets it (`BudgetFromAllocation`) | FR-001 – FR-005, FR-007 |
| `backend/internal/campaign` | `Campaign/Initiative.Overspend`, inputs, views, check, copies keep it | FR-001 – FR-005, FR-007 |
| `backend/internal/budget` | `GlobalBudget.Overspend`, allocation input/rows/options carry the plan's | FR-001, FR-004 |
| `backend/internal/activity` | `FormView.OverspendLimits` through an `Overspend` port | FR-014 |
| `backend/internal/purchaserequest` | `LineItemFacts.ActivityOverBy`, gate and message | FR-019 |
| `backend/internal/app` | guarded transactor for the five services, ports, routes; PR facts from line-item summaries | all |
| `backend/internal/access/routes_overspend.go` | two rules | FR-006 |
| `backend/api/openapi.yaml` | schemas `OverspendMode`, `OverspendControl`, `OverspendLimit`, `OverspendConfiguration(Input)`; properties on plan, campaign, initiative, global budget, allocation, activity form; error code `over_budget` | all |
| `frontend/src/features/overspend` (new) | `OverspendField`, `OverspendNote`, texts, the Settings page, API hooks | FR-002, FR-006, FR-015 |
| `frontend/src/features/plans` and `plans/campaigns` | forms show the field and the before-save checks | FR-002, FR-015 |
| `frontend/src/features/global-budgets` | form and allocation dialog | FR-004, FR-015 |
| `frontend/src/features/activities` | planned-cost check against `overspendLimits` | FR-014 |
| Docs | domain model §4.1, decision log, feature map, 032 FR-021 note | FR-020 |

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| The guard's budget read (`overspend.Budgets`: `ListOverspendBudgets`, `ListOverspendGlobalBudgets`) is not filtered by the actor's scope — constitution VII (scope isolation) and B5 ("every query on plan-owned data MUST filter by the actor's scope"). Recorded as D-355, ADR-0019 (proposed). | The rule is about money, not about what a person may see: a global budget's planned cost is the sum of every plan it allocates to, in every market (FR-008), so a planner whose scope holds one market must still be refused when their save takes the global budget over. The guard wraps the transaction of about twenty write paths (R3) and cannot know which records a save will reach before it runs, so it compares every budget before and after. Limits: the read takes only each budget's kind, ID, name, mode, amount and planned cost; nothing of it is returned except, in a refusal, the budgets the actor's own save took over (FR-010, Threat model); it is never a list, a view or a filter. | **Filter by the actor's scope**: undercounts a global budget's planned cost and misses overages, so Refuse would leak (SC-001). **Read only the ancestors of the records being saved**: needs every write path (create, update, move, copy, copy-many, split, restore, Undo, plan copy, allocations) to declare beforehand the records it will reach, including those a copy or restore has not created yet — the twenty call sites R3 replaces, each a place to forget the check; and a global budget still sums plans outside the scope. |
