---

description: "Tasks for 028 Query Efficiency"
---

# Tasks: Query Efficiency

**Input**: Design documents from `specs/028-query-efficiency/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md)

**Tests** (constitution 4.x): no test of SQL or of query counts; unit tests only for new logic (memo,
query count, ID ranges, batched walks and copies); the existing service, handler and contract tests and the
end-to-end suite run unchanged; benchmarks on PostgreSQL measure every optimisation (B9, research R1).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: the user story the task belongs to (US1 lists, US2 changes, US3 request log)

---

## Phase 1: Setup

- [X] T001 Branch `feature/028-query-efficiency` from `origin/develop`; spec, checklist, plan, research, tasks; feature-map row 028

---

## Phase 2: Measuring (blocks every fix)

- [X] T002 [US3] `internal/platform/querycount` and the pool's tracer (`database/tracer.go`, statements and COPY); `queries` on the request log record in `web.LogRequest` (FR-009)
- [X] T003 [US1] `dbtest.NewFilledDB`: a filled template per migration set, copied per benchmark (FR-008, R3)
- [X] T004 [US1] `internal/platform/apibench`: the volume (R3), the server and client, `BenchmarkLists` (demo and volume), `BenchmarkRecords` (small and large), `BenchmarkChanges`, `BenchmarkApprovals`, `BenchmarkSettings`, `BenchmarkSettingsImports`, `BenchmarkGSAPImport`; a request is stopped after a minute (FR-008)
- [X] T005 [US1] Baseline: every benchmark on the code before this feature (research R4)

---

## Phase 3: User Story 1 — lists and overviews (P1)

- [X] T006 [US1] Transaction memo in `Write` as well as `Read`, by area; Settings, people, teams, preferences, roles, assignments, access rules and approval workflows read once per transaction, forgotten on change (FR-005, FR-006)
- [X] T007 [US1] Planning stores keep rows by id and by parent (`memo.Records`); list queries take many ids (FR-002)
- [X] T008 [US1] Plan roll-ups: every plan's tree from one read per level (`campaigns.planTrees`); the plan levels ask once (FR-002)
- [X] T009 [US1] Derived values (an activity's completeness, an activity's line-item summary) kept per transaction in the `Derived` area that any change forgets (FR-002, FR-006)
- [X] T010 [P] [US1] Financials: the planning nodes from one read per level; a card reads only its plan, Home only its plans; global budgets read plans once (FR-002)
- [X] T011 [P] [US1] Approvals board, summary, monitoring, Home waiting: only activities with a round, in scope (`activities.ListInScopeOf`, scope-isolation test); facts built from one read per level; workflow usage from the workflow fields only; notifications at once (FR-002, FR-003, FR-007)
- [X] T012 [P] [US1] Activity log, an activity's history, recycle bin: where each entry's record is now, read kind by kind; plans for the access checks once (FR-002)
- [X] T013 [P] [US1] Line items read their spread months in one grouped join (FR-002)

---

## Phase 4: User Story 2 — changes to many records (P1)

- [X] T014 [US2] `ids.Generator.NextIDs`; `AddCampaigns`, `AddInitiatives`, `AddMany` (COPY) and `DeleteMany` on the planning stores and the JSON stand-in (FR-004)
- [X] T015 [P] [US2] Copies of plans, campaigns, initiatives and activities write each level at once, IDs in the order they had (FR-004)
- [X] T016 [P] [US2] Delete with everything below, Undo, Restore and Empty recycle bin walk and write the tree one level at a time; names, notes and entries unchanged (FR-004)
- [X] T017 [P] [US2] Rate and option imports save their rows at once; a GSAP import takes its new transactions' IDs at once and adds them with COPY; a restore reads the open approval rounds once (FR-004)

---

## Phase 5: Single-record pages (FR-003)

- [X] T018 [P] A person's page counts what they own in SQL; Budget by month reads its plan alone; an activity reads the activities split off it only (FR-003)

---

## Phase 6: Polish

- [X] T019 After: every benchmark on the final code; research R5–R7 and the pull request carry the tables
- [X] T020 Gates: gofmt, golangci-lint on new lines, go vet, `go test -race ./...` with `TEST_DATABASE_URL`, `go test -short ./...`, sqlc generate (no diff) and vet, govulncheck, the end-to-end suite on PostgreSQL
