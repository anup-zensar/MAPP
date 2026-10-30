# Implementation Plan: Performance Data from a Simulated Feed

**Branch**: `feature/046-performance-data` | **Date**: 2026-10-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/046-performance-data/spec.md`

## Summary

Build the records behind every performance figure: data sources with column mappings, performance facts loaded
by upload or a simulated pull, strict four-key matching to activities, a review queue for what did not match,
and a Performance section on an approved activity. The approach follows two patterns already in the codebase
rather than inventing: feature 015's import/validate/review shape (`internal/bulk`) and feature 030/033's
simulated adapter behind a port (`internal/simulatedgsap`, ADR-0016). The one genuinely new decision — how a
simulated feed produces rows without becoming a live connector — is recorded in [research.md](./research.md)
and ADR-0020.

## Technical Context

**Language/Version**: Go (version pinned in `backend/go.mod`); TypeScript 5.9 strict with React 19

**Primary Dependencies**: Gin (handler layer only), pgx/v5, sqlc, goose; TanStack Query, React Hook Form + Zod,
Radix primitives, `openapi-fetch` with generated types

**Storage**: PostgreSQL 18 behind store interfaces; migrations **00030–00032** (reserved for this feature)

**Testing**: Go `testing` + `httptest` + `go-cmp`; contract tests through `internal/platform/apitest`;
scope-isolation store tests on a real database; every other Go test on fakes. Vitest + Testing Library +
`user-event` + MSW + `vitest-axe`; Playwright + axe in `frontend/e2e/performance-data.spec.ts`

**Target Platform**: local only, `127.0.0.1`; no Docker required by any test or command

**Project Type**: web — Go API in `backend/`, React app in `frontend/`

**Performance Goals**: see *Benchmark* below

**Constraints**: no live connector, scheduler or external credential (constitution non-negotiable 13, D-370);
money as exact decimal minor units with its currency, never a float; every scoped read filtered by the actor

**Scale/Scope**: feature 028's volumes — 5,000 activities and 50 plans in scope — against files up to 50,000
rows; 6 user stories, 45 functional requirements

## Constitution Check

*GATE: passed before Phase 0, re-checked after Phase 1 design.*

| Principle | How this feature satisfies it |
|---|---|
| **I. Spec-First** | Every item below traces to an FR in [spec.md](./spec.md); test names carry the FR they prove |
| **II. Plain Language** | Shell vocabulary only: Activity, not Tactic; *data source*, *performance fact*, *row to review*, *media code*. No Hive9 interface terms |
| **III. Documented Design** | ADR-0020 records the simulated-feed decision; `contracts/api.md` feeds the wave B contract; `data-model.md` covers the eight tables |
| **IV. Two Parts, Server Authoritative** | Matching, validation and every count happen in the service layer; the web app renders what the API returns and recomputes no figure |
| **V. Simplicity** | One package `internal/performance` with the standard `model · store · store_postgres · service · handler` split, each file within 300 non-blank lines; no new framework |
| **VI. Meaningful Tests** | A unit test per rule, a contract test per operation, a scope-isolation test per scoped store read, one test per rendered state, an e2e per P1 story. Fixed clock and seed |
| **VII. Security and Privacy** | `performance.import` and `performance.link` with route rules that fail closed; every read filtered by the actor's scope; a personal-data file refused whole (FR-012) |
| **VIII. Clean Construction** | Comments at most 5% of non-blank lines, explaining *why* only; FR IDs in test names, never in comments |
| **IX. Tasteful Experience** | All validation errors at once; consequences shown before they happen (undo states its counts); every empty view states its reason; at most one yellow primary action per page |
| **X. Shell Palette** | Colours only from existing tokens; status chips reuse the shared components. Red only for danger and errors |
| **B5. Data** | sqlc queries, goose migrations, constraints for every invariant; stores behind interfaces with a fake per store |
| **B6. Contract** | Operations described in `contracts/api.md` and applied to `openapi.yaml` **by the integrator** — this lane never edits that file |

**Result**: no violations. *Complexity Tracking* below is therefore empty.

## Project Structure

### Documentation (this feature)

```text
specs/046-performance-data/
├── spec.md
├── plan.md              # this file
├── research.md          # the four real choices
├── data-model.md        # eight tables across migrations 00030-00032
├── contracts/api.md     # operations for the wave B contract
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks output, not created here
```

### Source code

```text
backend/
├── internal/performance/          # model.go · store.go · store_postgres.go · service.go · handler.go
│   ├── source.go  mapping.go  importer.go  matcher.go  review.go  mediacode.go  externalname.go
│   ├── simulatedfeed/             # the adapter behind a port (ADR-0020)
│   └── performancetest/           # the fake store for other packages
├── internal/app/performance.go    # wiring only
├── internal/platform/database/
│   ├── migrations/00030_performance_sources.sql
│   │                00031_performance_facts.sql
│   │                00032_media_codes.sql
│   └── queries/performance.sql
└── seed/measure.go                # wave B demo facts — see Risk below

frontend/
├── src/features/performance/      # Settings › Performance data, review queue, the activity Performance panel
└── e2e/performance-data.spec.ts
```

**Structure Decision**: one business-area package, as constitution B2 requires and every other area follows.
The simulated feed is a sub-package behind a port so the service never knows whether rows came from a file or
the adapter — the same shape `simulatedgsap` uses. Media-code configuration lives in this feature's own
Settings › Performance data area, not on the activity type record (046-C12): activity types belong to feature
002's `settings`, and the build guide §4.1 grants this feature exactly one file outside its own folders — the
Performance slot in `ActivityPage.tsx`.

## Rules, and the tests that prove them

| # | Rule | FR | Test |
|---|---|---|---|
| R1 | Keys resolve in strict order: activity ID → media code → integration ID → external name | FR-018 | unit, table-driven over all four keys and their precedence |
| R2 | Keys that resolve to different activities never guess — unmatched, both candidates named | FR-019 | unit; e2e scenario US2#3 |
| R3 | A fact on an activity not counting as approved is held, counts nowhere, releases on approval | FR-020, FR-028 | unit for the status set; integration for the release |
| R3a | **Implementation note**: *In progress* and *Complete* are **derived from the activity's dates**, not stored — `activities.status` holds only `created`, `submitted`, `declined`, `approved` (domain model §2.3). "Counts as approved" is therefore the single test `status = 'approved'`, which covers all three displayed statuses | FR-028 | the same unit test as R3 |
| R4 | The five outcomes add up to rows read | FR-022 | property test with `fast-check`-style generated files |
| R5 | Loading is idempotent on (source, source reference); a repeat replaces and audits | FR-013 | unit + contract: same file twice changes nothing |
| R6 | A personal-data file is refused whole, before any row is stored | FR-012 | unit: no row persisted after refusal |
| R7 | Revenue without a rate is stored exactly, marked, excluded with its count | FR-014 | unit on money; view test for the excluded-count text |
| R8 | A fact counts in the fiscal year of its own date | FR-015 | unit across a multi-year activity |
| R9 | One active source per stage per model | FR-003 | unit; contract test for the refusal naming the source |
| R10 | Delete refused while a source holds facts; Deactivate offered | FR-002 | unit + e2e US1#4 |
| R11 | Undo restores replaced versions and leaves later facts alone | FR-017 | integration over two uploads then an undo |
| R12 | Matching uses the activity's current external name **and** its aliases; a rename keeps the old name so rows in flight still match. This feature never writes `activities.external_name`, which feature 006 owns (046-C14) | FR-033, FR-034 | unit on alias resolution; integration on a rename |
| R13 | A copied activity carries no external name, alias, integration ID, media code or fact | FR-038 | integration on copy |
| R14 | Media codes unique and never reused, from a configurable start | FR-042 | unit on the generator |
| R15 | Every empty performance view states its reason | FR-032 | one view test per state (six states) |

## Queries, and their scope filter

Every query below filters by the actor's scope through the plan that owns the activity. A record outside scope
answers as missing, never as a refusal (constitution non-negotiable 7, FR-044).

| Query | Scope filter | Notes |
|---|---|---|
| `ListSources`, `GetSource` | organisation-wide (configuration, not plan-owned) | freshness computed from the latest import |
| `ListFactsForActivity` | activity → plan → actor scope | the Performance panel; one query, not one per stage |
| `MatchRows` | candidate activities restricted to the actor's scope | **set-based**: one query per key type per batch, never one per row (R16 below) |
| `ListReviewRows` | row → source → the facts' plans in scope | filterable by key, source, reason, upload, date, status |
| `SummariseImport` | import → actor scope | the five counts, computed in SQL |
| `ListMediaCodes` | activity → plan → actor scope | |

**R16 (efficiency, constitution B9 and feature 028)**: matching a file of N rows MUST use a constant number of
queries, independent of N. A per-row lookup is the failure mode this rule exists to prevent.

## Permissions

| Operation | Permission | Route rule |
|---|---|---|
| Upload a file, run a simulated pull, undo an import | `performance.import` | new rule, fails closed |
| Set external name, alias, integration ID; generate or edit a media code | `performance.link` | new rule, fails closed |
| Create, edit, deactivate or delete a data source or mapping | `performance.import` | configuration is administrator work |
| View the Performance panel and an activity's facts | existing activity read permission | no new permission; scope still applies |
| Correct, dismiss or reopen a review row | `performance.import` | the row belongs to an import |

Both permissions are reserved by D-367. Every new operation gets its rule in the access route table; the rules
themselves are added by the integrator where they live in the shared `routes_measure.go`, reported in this
feature's pull request.

## Benchmark

Recorded here per issue #39 and constitution B9. Volumes are feature 028's.

| Measurement | Target |
|---|---|
| Match a 50,000-row file against 5,000 in-scope activities | under 10 s, and a **constant** query count (R16) |
| Open the Performance panel of an approved activity with 2,000 facts | under 1 s, matching 028's page target |
| Summarise an import (the five counts) | one query, under 200 ms |
| Re-check open review rows after a new import from the same source | constant query count, independent of queue size |

## Contract

The operations this feature adds are listed in [contracts/api.md](./contracts/api.md). They belong to the
**wave B contract**, which this lane writes together with 047 and hands to the integrator **before 048–050
start** (build order step 7). This lane never edits `backend/api/openapi.yaml`.

## Risks reported, not worked around

1. **`backend/seed/measure.go` does not exist.** Lane D creates it and 039 wires its call into `seed.go`.
   Issue #50 asks this lane to seed wave B facts early so lanes A and C can build on them, which is not
   presently possible. Raised in the pull request.
2. **ADR-0019 is already taken.** Issue #39 reserves ADR-0019 for the simulated performance feed, but
   `0019-overspend-guard-unscoped-budget-read.md` was written by feature 037 on 2026-10-29, the day before the
   measure kick-off. This feature therefore writes **ADR-0020**, per the ADR README's "add a new ADR (next
   number)". The integrator should correct the number in issue #39 and the build guide.

## Complexity Tracking

No constitution violations: the table is intentionally empty.
