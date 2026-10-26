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
interface, and its store tests. No service, handler or wiring changes.

- [ ] T011 [US1] ids counters: `00002`, `internal/platform/ids/store_postgres.go` (`Increment` in one statement); test that concurrent increments never give a number twice
- [ ] T012 [US2] history: `activity_log` with its append-only trigger and `recycle_bin` (`00002`); tests for order by `seq`, bin entries resolved and not, refused `UPDATE`/`DELETE` of the log
- [ ] T013 [P] [US1] people and teams: `00003` people, teams, memberships, preferences, pinned plans (depends on T011)
- [ ] T014 [US1] auth sessions: rest of `00003` (depends on T013)
- [ ] T015 [P] [US1] settings: `00004`, `money.Rate`/`money.Percent` ↔ `numeric`; measure the pages that use `store.Memo` on PostgreSQL and add a per-transaction memo to `database` only if they miss today's times (R11)
- [ ] T016 [US1] access: `00005` roles, permissions, assignments with teams, rules (depends on T013, T015)
- [ ] T017 [US1] approval workflows: `00008` workflows, levels, approvers (depends on T016)
- [ ] T018 [US2] plans: `00006`; store methods take the actor's scope; isolation test per read — **waits for 022**
- [ ] T019 [US2] campaigns and initiatives: `00007`; scoped through the plan — **waits for 022** (depends on T018)
- [ ] T020 [US2] activities and line items: `00009`; scoped through the chain — **waits for 022** (depends on T017, T019)
- [ ] T021 [US2] approvals: `00010` rounds, actions (write-once trigger), notifications, delegations (`btree_gist` exclusion) — **waits for 022** (depends on T020)
- [ ] T022 [P] [US2] global budgets and allocations: `00011` — **waits for 022** (depends on T018)
- [ ] T023 [P] [US2] financials: `00012` imports and transactions; today's visibility rules in SQL — **waits for 022** (depends on T020)
- [ ] T024 [P] [US1] All work: `00013` preferences and saved views (person-scoped in SQL) (depends on T013)

---

## Phase 5: Switch to PostgreSQL (US1)

- [ ] T025 [US1] Move the 34 test files and `internal/platform/apitest` that open the JSON store onto `dbtest.NewDB` (they become store-backed tests, skipped under `-short`) — depends on T011–T024
- [ ] T026 [US1] `internal/app`: `DATABASE_URL` in the configuration (replaces `DATA_DIR`), pool opened once and closed on shutdown, start refused when migrations are pending (`HasPending`), every PostgreSQL store and `database.DB` wired as the `Transactor`; `/readyz` pings the pool — **waits for 022 and 024**
- [ ] T027 [US1] `cmd/seed`: open, `Migrate`, load the demo data; `-reset` empties the tables; seed idempotent (FR-002, spec US1-1) — depends on T026
- [ ] T028 [US1] End-to-end tests start the API on a PostgreSQL database seeded per run (`frontend/e2e` setup, `DATABASE_URL`) — depends on T027
- [ ] T029 [US1] CI (`.github/workflows/ci.yml`): PostgreSQL 18 service for `backend` and `e2e`, `TEST_DATABASE_URL`, a `sqlc generate` no-diff step and `sqlc vet` (FR-012, quality gates 3–4) — must land before the pull request is opened, because store tests fail without `TEST_DATABASE_URL`
- [ ] T030 [P] Documentation: ADR superseding ADR-0004; `backend/README.md` and root `README.md` (PostgreSQL set-up, `DATABASE_URL`, `TEST_DATABASE_URL`, `CREATEDB` for tests); `GO_STANDARDS.md` §6 corrected to the one-at-a-time `Write` and the UTC codec (R2, R3); `CLAUDE.md` "Current code" storage lines; feature map row 023

---

## Phase 6: Remove the JSON store

- [ ] T031 Delete `internal/platform/store`, every `store_json*.go` and its tests, `NewJSON…` constructors and the data-folder handling of `cmd/api` and `cmd/seed` — depends on T025–T028
- [ ] T032 Run the full gates: `gofmt`, `go vet`, golangci-lint on changed lines, `sqlc generate` (no diff), `go test -race ./...` with PostgreSQL, `go test -short ./...` without, `govulncheck`, the frontend suites and `npm run test:e2e` on PostgreSQL (SC-001–SC-005)

---

## Dependencies & Execution Order

- Phases 1–3 are done on this branch; the application still runs on the JSON store.
- T011 and T012 come first (every area writes ids and log entries). T013–T017 and T024 need nothing from
  022/024 and can start now; T015 and T024 run beside T013's successors.
- T018–T023 wait for 022 (how the actor's scope reaches services and stores, and the 404 rule).
- T025 needs every store; T026 waits for 022 and 024 (both rewrite `internal/app`); T027 → T028.
- T029 must be merged with or before the first pull request of this branch; T030 any time before the
  pull request; T031 after the switch; T032 last.
