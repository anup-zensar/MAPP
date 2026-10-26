---

description: "Tasks for 023 PostgreSQL Storage"
---

# Tasks: PostgreSQL Storage

**Input**: Design documents from `specs/023-postgresql-storage/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md)

**Tests** (constitution 4.0.0): one scope-isolation test per scoped area on PostgreSQL, covering each of
its `…InScope` reads (FR-009, FR-010), and the end-to-end tests on PostgreSQL. No database test of queries,
constraints, migrations or the write lock; no test of generated code. The store tests written by T007,
T010 and T011–T024 before that rule were deleted except the scope-isolation ones.

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
- [X] T006 [US3] `backend/internal/platform/database/dbtest/dbtest.go`: `NewDB` from a template per migration set; skip under `-short`, fail without `TEST_DATABASE_URL` (FR-010, R5)
- [X] T007 [US2] ~~`database_test.go`~~ (deleted under constitution 4.0.0: no database test of the write lock or migrations)

---

## Phase 3: Pilot — samples (R8)

- [X] T008 [US1] `migrations/00001_sample_records.sql`, `queries/samples.sql`, generated `sqlcdb/`
- [X] T009 [US1] `backend/internal/samples/store_postgres.go`: `PostgresStore` implements `samples.Repository` (FR-008)
- [X] T010 [US2] ~~samples store tests~~ (deleted under constitution 4.0.0: samples hold no scoped records)

---

## Phase 4: The remaining stores (in migration order, data model §2)

Each task: the migration, `queries/<area>.sql` and `store_postgres.go` implementing the area's existing
interface. No service, handler or wiring changes. Plan-owned stores also get `…InScope` reads, with one
scope-isolation test per area (FR-009); after 022 the store interfaces stayed unscoped (022 kept today's stores), so the
scoped reads are additions the services adopt in T025a.

- [X] T011 [US1] ids counters: `00002_ids_counters`, `ids.PostgresCounters`; concurrent Writes never give a number twice
- [X] T012 [US2] history: `00003_history` (`activity_log` read in `seq` order, `recycle_bin`); write-once trigger added in `00016_write_once` (no test, constitution 4.0.0)
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

- [X] T025 ~~Move the tests that open the JSON store onto `dbtest.NewDB`~~ — dropped: constitution 4.0.0 keeps service, fixture and handler tests off the database; they stay on the JSON store until 025 replaces it with fakes
- [X] T025a [US2] Person-facing lists read through `…InScope` with `authz.ScopeOf` instead of filtering loaded records in Go: plans (`plans.Repository.ListInScope`: the Plans list, Home, All work, financials' visible plans) and activities (`activities.Repository.ListInScope`: the Approvals board). `authz.Scope.Covers`/`Sees`/`SeesActivity` are the SQL scope functions in Go; the `Grants` checks and the JSON stand-in use them. Budgets' allocation rows and the financials list keep their access ports until 025, because their service tests fake visibility there (`AllocationsInScope` and `TransactionsInScope` are ready). `store.Memo` loads on every call outside a JSON transaction; the measurement is in T026 (R11)
- [ ] T026 [US1] `internal/app`: `DATABASE_URL` in the configuration (replaces `DATA_DIR`), pool opened once and closed on shutdown, start refused when migrations are pending (`HasPending`), every PostgreSQL store and `database.DB` wired as the `Transactor`; `/readyz` pings the pool — after 024 (merged)
- [ ] T027 [US1] `cmd/seed`: open, `Migrate`, load the demo data; `-reset` empties the tables with `TRUNCATE` (the write-once triggers refuse `DELETE`); seed idempotent (FR-002, spec US1-1) — depends on T026
- [ ] T028 [US1] End-to-end tests start the API on a PostgreSQL database seeded per run (`frontend/e2e` setup, `DATABASE_URL`); the `e2e` CI job gains the PostgreSQL service — depends on T027
- [X] T029 [US1] CI (`.github/workflows/ci.yml`): PostgreSQL 18 service for `backend` (trust authentication, bound to `127.0.0.1`, health check), `TEST_DATABASE_URL` for the Test step, a "Generated queries" step (`sqlc generate` leaves no change, `sqlc vet`) (FR-012, quality gates 3–4); the dbtest template build applies every migration to an empty database (gate 4)
- [ ] T030 [P] Documentation: ADR superseding ADR-0004; `backend/README.md` and root `README.md` (PostgreSQL set-up, `DATABASE_URL`, `TEST_DATABASE_URL`, `CREATEDB` and the `btree_gist` extension for tests); `GO_STANDARDS.md` §6 corrected to the one-at-a-time `Write` and the UTC codec (R2, R3); the command that drops leftover `mapp_template_*` and `mapp_test_*` databases in the backend README's test section; `CLAUDE.md` "Current code" storage lines; feature map row 023

---

## Phase 6: JSON store out of the application

- [ ] T031 Remove the JSON store from the wiring and the seed: the data-folder setting and `-reset`'s file handling in `cmd/api`, `cmd/seed` and `internal/app`. `internal/platform/store` and the `store_json*.go` files stay as the test stand-in, with a one-line note that 025 deletes them — depends on T026–T028
- [ ] T032 Run the full gates: `gofmt`, `go vet`, golangci-lint on changed lines, `sqlc generate` (no diff), `go test -race ./...` with PostgreSQL, `go test -short ./...` without, `govulncheck`, the frontend suites and `npm run test:e2e` on PostgreSQL (SC-001–SC-005)

---

## Dependencies & Execution Order

- Phases 1–4 are done on this branch; the application still runs on the JSON store.
- T025a needs every store. T026 follows 024 (it rewrote `internal/app`); T027 → T028.
- T030 any time before the pull request; T031 after the switch; T032 last.
