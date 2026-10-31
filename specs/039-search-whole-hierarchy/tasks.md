# Tasks: Search the whole hierarchy

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [contracts/api.md](contracts/api.md)

## Phase 1: Setup and documents

- [x] T001 Decision D-366 in `docs/sdd/decision-log.md`; feature map row 039 in `docs/sdd/feature-map.md`;
  amendment notes on 004 FR-003, FR-013, FR-014 in `specs/004-plans/spec.md`

## Phase 2: Foundational (blocks both stories)

- [x] T002 Contract: `GET /api/v1/plans/{planId}/search`, `PlanSearch`, `PlanSearchMatch`, `PlanSummary.foundIn`
  in `backend/api/openapi.yaml`; `pnpm run api:types`
- [x] T003 `SearchMatch`, the `Searcher` port and `UseSearcher` in `backend/internal/plan/search.go`
- [x] T004 The `Searcher` adapter over the four batched level reads in `backend/internal/app/plan_search.go`,
  wired in `internal/app/areas.go`; proven on the demo data by T007's contract test

## Phase 3: User Story 1 — the plan tree's search finds line items (P1) 🎯 MVP

- [x] T005 [US1] Test first: `Service.Search` returns the plan's matches top-down, refuses an empty or too long
  `q`, answers not found outside the scope (`backend/internal/plan/search_test.go`)
- [x] T006 [US1] `Service.Search` in `backend/internal/plan/search.go`; handler `GET /plans/:planId/search` in
  `backend/internal/plan/handler.go`; route rule in `backend/internal/access/routes_planning.go`
- [x] T007 [US1] Contract test of the endpoint (200, 400, 404) in `backend/internal/plan/handler_search_test.go`
- [x] T008 [US1] `usePlanSearch` in `frontend/src/features/plans/api.ts`; `usePlanSearchRows` (300 ms after the
  last key, line item rows by activity, the failure note) in `frontend/src/features/plans/tree/usePlanSearchRows.tsx`
- [x] T009 [US1] The tree shows the rows under their activities, opens their path and the note
  (`frontend/src/features/plans/tree/PlanTree.tsx`, `frontend/src/features/plans/PlanPage.tsx`)
- [x] T010 [US1] Component tests for scenarios 1–5 and the failure note in
  `frontend/src/features/plans/tree/PlanTree.search.test.tsx`
- [x] T011 [US1] End-to-end test of US1 against the real API in `frontend/e2e/plan-search.spec.ts`

## Phase 4: User Story 2 — the Plans list's search finds plans by their contents (P2)

- [x] T012 [US2] Test first: the list lists a plan through its contents with `foundIn`, not a plan whose own name
  matches, keeps the other filters (`backend/internal/plan/search_test.go`)
- [x] T013 [US2] `itemsOf` lists plans through their contents in one `Searcher` call
  (`backend/internal/plan/list.go`, `search.go`)
- [x] T014 [US2] Contract test of `foundIn` on `GET /api/v1/plans?q=` in `backend/internal/plan/handler_search_test.go`
- [x] T015 [US2] "Found in …" on the card and the table (`frontend/src/features/plans/FoundIn.tsx`,
  `PlanCard.tsx`, `PlanTable.tsx`) with component tests in `FoundIn.test.tsx`

## Phase 5: Polish

- [x] T016 Checks: `golangci-lint fmt`, `golangci-lint run`, `go test -race ./...`, `pnpm run check`, the e2e test;
  comment budget of every touched file
