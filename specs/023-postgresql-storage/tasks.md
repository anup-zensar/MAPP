---

description: "Tasks for 023 PostgreSQL Storage"
---

# Tasks: PostgreSQL Storage

**Input**: Design documents from `specs/023-postgresql-storage/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md)

**Tests**: every store gets tests on PostgreSQL for each promise of its interface, for the constraints of
its tables (including that the deletes its service allows are not refused), and an isolation test for
every scoped read (FR-008, FR-009). No test of generated code.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: the user story the task belongs to (US1, US2, US3)
- **Waits for 022 / 024**: cannot start until that feature is merged into `plan-module`

---

## Phase 1: Setup

- [X] T001 Branch `feature/023-postgresql-storage` from `feature/019-alignment-guardrails`; spec, research, data model, plan, tasks
- [X] T002 Dependencies in `backend/go.mod` and `backend/go.sum`: pgx v5.11.0, goose v3.29.0, go-cmp v0.7.0; every new or raised version older than 24 hours (R1)
- [X] T003 `backend/sqlc.yaml`; sqlc v1.31.1 run with `go run` (R7)

---

## Phase 2: Foundational (blocks every area)

- [X] T004 [US2] `backend/internal/platform/database/database.go`: `Open` (pool, ping, UTC timestamps), `Close`, `Write` (one at a time behind a transaction-scoped advisory lock), `Read` (read-only snapshot), `Queries` (FR-001, FR-005–FR-007, R2, R3)
- [X] T005 [US1] `backend/internal/platform/database/migrate.go`: embedded migrations, `Migrate` with the PostgreSQL session locker (FR-002, R6)
- [X] T006 [US3] `backend/internal/platform/database/dbtest/dbtest.go`: `NewDB` from a template per migration set, `NewEmptyDB`; skip under `-short`, fail without `TEST_DATABASE_URL` (FR-010, R5)
- [X] T007 [US2] `backend/internal/platform/database/database_test.go`: a failed `Write` keeps nothing (nested `Write` included), a second `Write` waits for the first and sees its change, `Write` inside `Read` refused, timestamps in UTC, concurrent `Migrate`

---

## Phase 3: Pilot — samples (R8)

- [X] T008 [US1] `migrations/00001_sample_records.sql`, `queries/samples.sql`, generated `sqlcdb/`
- [X] T009 [US1] `backend/internal/samples/store_postgres.go`: `PostgresStore` implements `samples.Repository` (FR-008)
- [X] T010 [US2] `backend/internal/samples/store_postgres_test.go`: read back as saved, not found, replace, shorter ids first, deletes, notes of a folder, parents checked at commit (a folder may go before its notes in one `Write`), orphan and duplicate refused, racing renames → one `version_conflict` (FR-004, FR-006, FR-008)

---

## Phase 4: The remaining stores (in migration order, data model §2)

Each task: the migration, `queries/<area>.sql`, `store_postgres.go` implementing the area's existing
interface, and its store tests (read back as saved, the JSON store's order, the constraints, deletes the
service allows). No service, handler or wiring changes. Plan-owned stores also get `…InScope` reads with an
isolation test (FR-009); after 022 the store interfaces stayed unscoped (022 kept today's stores), so the
scoped reads are additions the services adopt in T025a.

- [X] T011 [US1] ids counters: `00002_ids_counters`, `ids.PostgresCounters`; concurrent Writes never give a number twice
- [X] T012 [US2] history: `00003_history` (`activity_log` read in `seq` order, `recycle_bin`); write-once trigger added in `00016_write_once` and tested with raw statements through `dbtest.Exec`
- [X] T013 [P] [US1] people and teams: `00004_people` (people, preferences with `pinned_plan_ids text[]`, teams, memberships)
- [X] T014 [US1] auth sessions: `00005_sessions`
- [X] T015 [P] [US1] settings: `00006_settings`; `money.Rate`/`money.Percent` as `bigint` fixed point (R3); legacy currency rate columns kept so `MigrateRates` still finds them. The `store.Memo` measurement moved to T025a (it needs the services on PostgreSQL)
- [X] T016 [US1] access: `00007_access` (roles with `permissions text[]`, assignments with ordered teams, rules)
- [X] T017 [US1] approval workflows: `00008_approval_workflows`; a level's role is a soft reference (data model §4)
- [X] T018 [US2] plans: `00010_plans`, `plan_in_scope()`, `authz.ScopeOf`; scoped reads equal `authz.CanView` (FR-009)
- [X] T019 [US2] campaigns and initiatives: `00011_campaigns`; scoped through the plan
- [X] T020 [US2] activities and line items: `00012_activities`, `activity_in_scope()`; an activity's workflow is a soft reference
- [X] T021 [US2] approvals: `00013_approvals` (rounds, history, notifications, delegations with a `btree_gist` exclusion); `activity_in_scope()` gains named approvers; write-once trigger in `00016_write_once` (also on `rate_imports`)
- [X] T022 [P] [US2] global budgets and allocations: `00014_global_budgets`; an allocation's global budget is a soft reference
- [X] T023 [P] [US2] financials: `00015_financials`, `plan_in_permission_scope()`, `authz.PermissionScopeOf`; finance.view and import visibility in SQL
- [X] T024 [P] [US1] All work: `00009_all_work` (person-scoped `ListViews`)

---

## Phase 5: Switch to PostgreSQL (US1)

- [ ] T025 [US1] Move the 34 test files and `internal/platform/apitest` that open the JSON store onto `dbtest.NewDB` (they become store-backed tests, skipped under `-short`) — depends on T011–T024
- [ ] T025a [US2] Services read person-facing lists through the stores' `…InScope` methods with `authz.ScopeOf` / `PermissionScopeOf` instead of filtering loaded records in Go; measure the pages that use `store.Memo` and add a per-transaction memo to `database` only if they miss today's times (R11) — depends on T025
- [ ] T026 [US1] `internal/app`: `DATABASE_URL` in the configuration (replaces `DATA_DIR`), pool opened once and closed on shutdown, start refused when migrations are pending (`HasPending`), every PostgreSQL store and `database.DB` wired as the `Transactor`; `/readyz` pings the pool — **waits for 024** (022 is merged)
- [ ] T027 [US1] `cmd/seed`: open, `Migrate`, load the demo data; `-reset` empties the tables with `TRUNCATE` (the write-once triggers refuse `DELETE`); seed idempotent (FR-002, spec US1-1) — depends on T026
- [ ] T028 [US1] End-to-end tests start the API on a PostgreSQL database seeded per run (`frontend/e2e` setup, `DATABASE_URL`); the `e2e` CI job gains the PostgreSQL service — depends on T027
- [X] T029 [US1] CI (`.github/workflows/ci.yml`): PostgreSQL 18 service for `backend` (trust authentication, bound to `127.0.0.1`, health check), `TEST_DATABASE_URL` for the Test step, a "Generated queries" step (`sqlc generate` leaves no change, `sqlc vet`) (FR-012, quality gates 3–4); the dbtest template build applies every migration to an empty database (gate 4)
- [ ] T030 [P] Documentation: ADR superseding ADR-0004; `backend/README.md` and root `README.md` (PostgreSQL set-up, `DATABASE_URL`, `TEST_DATABASE_URL`, `CREATEDB` and the `btree_gist` extension for tests); `GO_STANDARDS.md` §6 corrected to the one-at-a-time `Write` and the UTC codec (R2, R3); `CLAUDE.md` "Current code" storage lines; feature map row 023

---

## Phase 6: Remove the JSON store

- [ ] T031 Delete `internal/platform/store`, every `store_json*.go` and its tests, `NewJSON…` constructors and the data-folder handling of `cmd/api` and `cmd/seed` — depends on T025–T028
- [ ] T032 Run the full gates: `gofmt`, `go vet`, golangci-lint on changed lines, `sqlc generate` (no diff), `go test -race ./...` with PostgreSQL, `go test -short ./...` without, `govulncheck`, the frontend suites and `npm run test:e2e` on PostgreSQL (SC-001–SC-005)

---

## Dependencies & Execution Order

- Phases 1–4 are done on this branch; the application still runs on the JSON store.
- T025 needs every store; T025a follows it. T026 waits for 024 (it rewrites `internal/app`); T027 → T028.
- T030 any time before the pull request; T031 after the switch; T032 last.
