# Implementation Plan: Query Efficiency

**Branch**: `feature/028-query-efficiency` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/028-query-efficiency/spec.md`

## Summary

Measure first, on a database holding the volume of the specs' scale targets ([research](research.md) R3),
then remove every read that grows with the records, in the order the instructions set: (a) read related
records in one query by IDs, (b) keep what a transaction read for the rest of it, (c) a purpose-built
query only where (a) and (b) are not enough. The transaction memo of feature 023 (`internal/platform/memo`)
becomes the backbone: every store keeps the rows it read — reference tables whole, planning records by id
and by parent — until it changes them, in `Read` and `Write` alike, and nothing outlives the transaction
(R5 items 1–3). Services that loop over many plans or records read each level once for all of them (R5
item 4). Writes that copy, move, delete, restore or import many records read and write each level together
(R5 item 5). Each request's
statement count goes into the request log (R2). Behaviour, contract and tests stay as they are.

## Technical Context

**Language/Version**: Go 1.27 (`toolchain go1.27.3`); PostgreSQL 18

**Primary Dependencies**: pgx v5.11.0 (its `QueryTracer` and `CopyFromTracer`), sqlc v1.31.1 (run with
`go run`); no new module

**Storage**: PostgreSQL; no migration (the schema is unchanged, so the benchmark template and the e2e
database need no rebuild for it)

**Testing**: existing service, handler and contract tests on the JSON stand-in, unchanged; unit tests for
the new memo and query-count logic; the e2e suite on PostgreSQL; benchmarks in
`internal/platform/apibench` on PostgreSQL, skipped under `-short` (R1)

**Target Platform**: developers' machines; GitHub Actions

**Project Type**: Web application (Go API + React web app); this feature changes the API only

**Performance Goals**: the scale targets of 003, 004, 009, 010, 011, 012, 015 and 016 (spec SC-001 –
SC-005); queries per request independent of the records (SC-006)

**Constraints**: no change of behaviour or contract (FR-001); no package renames or broad refactors
(feature 025 comes later); new files ≤ 300 non-blank lines and ≤ 5% comment lines; functions within the
quality metrics

**Scale/Scope**: about 40 endpoints and 15 operations measured; stores of 8 areas; the services of
plans, campaigns, activities, line items, approvals, history, All work, Home

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | Spec FR-001–FR-009 with acceptance scenarios; branch `feature/028-query-efficiency` | Pass |
| III Documented design | Plan, research (R1–R9) | Pass |
| IV Server source of truth | No figure moves to the web app; the contract is unchanged | Pass |
| V Simplicity; no premature optimisation | Each change answers a measured count or time (research R7) and is backed by a benchmark (B9) | Pass |
| VI Meaningful tests | No test of SQL or of query counts; benchmarks assert nothing (R1); unit tests only for new logic (memo, query count) | Pass |
| VII Security, scope isolation | Scope stays in SQL: the `…InScope` reads are unchanged; one new scoped read (`activities.ListInScopeOf`, for the Approvals board) has its scope-isolation test; the other batched reads replace unscoped internal reads | Pass |
| VIII Clean construction | Comment budget and file length checked for every new or changed file | Pass |
| B1 Stack | pgx tracer interfaces, sqlc `:copyfrom`; nothing new | Pass |
| B3 Concurrency | The memo table is guarded by a mutex; one per transaction | Pass |
| B5 Data | Every query in `.sql` via sqlc; transactions owned by services; list reads no longer issue a query per row | Pass |
| B7 Observability | `queries` on the request log record | Pass |
| B9 Testing | A benchmark for every optimisation (R1) | Pass |
| Governance → Alignment | No package renamed, no store interface reshaped; methods added only | Pass |

**Post-design re-check (after Phase 1)**: unchanged — Pass.

### Threat model (STRIDE)

| Threat | Where | Response |
|---|---|---|
| Information disclosure: a kept record served to another person | transaction memo | the memo lives in one transaction's context, which one request owns; never shared (FR-005) |
| Information disclosure: a batched read returns records outside the scope | batched reads | they replace unscoped internal reads; every person-facing list still goes through an `…InScope` read, and the new `ListInScopeOf` filters in SQL with a scope-isolation test |
| Tampering: a save sees stale data | memo in `Write` | each store forgets its area before any change (FR-006); the services' version checks are unchanged |
| Denial of service: unbounded memory | memo | bounded by what one transaction reads, which it read before anyway |

### Risk model

| Risk | Response | Owner |
|---|---|---|
| A store that keeps rows misses a write path, so a save reads a stale row | Every write method of a store that keeps rows goes through its `change`, which forgets the area first; the e2e suite runs every write on PostgreSQL | Team |
| A caller changes a record a store handed out, corrupting the kept copy | Stores keep rows and decode a fresh value per call; kept slices are cloned on the way out | Team |
| The JSON stand-in and PostgreSQL disagree on a new batch method | Both implement the same contract (order by id, missing ids left out); service tests run on the stand-in, e2e on PostgreSQL | Team |

## Project Structure

### Documentation (this feature)

```text
specs/028-query-efficiency/
├── spec.md
├── checklists/requirements.md
├── research.md      # R1–R9, before/after tables
├── plan.md          # this file
└── tasks.md
```

### Source Code

```text
backend/internal/platform/
├── querycount/            # the statements of one request
├── memo/                  # Get, Many, Keep, Forget; Table (whole tables); Records (by id and parent)
├── database/tracer.go     # counts every statement; Write keeps a memo too
├── database/dbtest/filled.go   # a filled template copied per benchmark
├── web/middleware.go      # queries on the request log record
└── apibench/              # the volume, the benchmark server and client, the benchmarks
backend/internal/<area>/store_postgres*.go   # keep rows; forget on change; batch reads and writes
backend/internal/<area>/…                    # services read each level once for many records
```

**Structure Decision**: no new layer; the memo is the existing platform package, extended.

## Complexity Tracking

| Deviation | Why needed | Simpler alternative rejected because |
|---|---|---|
| None | — | — |
