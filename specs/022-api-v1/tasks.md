---

description: "Tasks for 022 API v1"
---

# Tasks: API v1

**Input**: Design documents from `specs/022-api-v1/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md)

**Tests**: existing tests are updated to the new contract; new tests only where a new rule has none
(out-of-scope 404, `/readyz`).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: the user story the task belongs to (US1–US3)

---

## Phase 1: User Story 1 — One versioned contract with one error shape (P1) 🎯 MVP

- [X] T001 [US1] `backend/api/openapi.yaml`: every path under `/api/v1`; `Error` becomes `{error: ErrorDetail}` (FR-001–FR-003)
- [X] T002 [US1] Every `Routes(...)` pattern in `backend/internal/*/handler*.go` and the route table `backend/internal/access/routes.go` under `/api/v1`; the JSON 404 catch-all stays on `/api/` (FR-001)
- [X] T003 [US1] `backend/internal/platform/web/web.go`: `WriteError` and the `WriteJSON` fallback write the envelope; `errs` package doc (FR-003)
- [X] T004 [US1] Backend tests: `/api/v1` paths and enveloped error bodies (`web_test.go`, `app_test.go`, handler and access tests)
- [X] T005 [US1] `frontend/src/api/client.ts` and `errors.ts` read the envelope; regenerate `src/api/schema.d.ts` with `npm run api:types` (FR-004)
- [X] T006 [P] [US1] Web app calls, `mockApi` keys and test error bodies under `frontend/src/` and `frontend/e2e/` use `/api/v1` and the envelope (FR-002)

---

## Phase 2: User Story 2 — Another market's records cannot be detected (P1)

- [X] T007 [US2] `backend/internal/platform/authz`: `NotFoundMessage` replaces `NoAccessMessage`; `RequireView`, `RequireViewActivity` answer 404 (FR-005)
- [X] T008 [US2] `backend/internal/access/middleware.go`, `targets.go`: a path record the person cannot see answers 404 (FR-005, FR-006)
- [X] T009 [P] [US2] `backend/internal/history/bin.go`, `service.go`, `backend/internal/app/history_access.go`: a bin entry the person cannot see answers 404 "This item is not in the recycle bin." (FR-005)
- [X] T010 [P] [US2] `backend/internal/financials/service.go`: an out-of-scope transaction or record card answers its not-found message (FR-005)
- [X] T011 [P] [US2] `backend/internal/budgets/allocations.go`: an allocation of a plan the person cannot see answers 404 (FR-005)
- [X] T012 [US2] Update the tests that asserted 403 for out-of-scope records; add `TestOutOfScopeRecord_AnswersLikeAMissingOne_022_FR005` comparing out-of-scope and unused IDs (SC-003)

---

## Phase 3: User Story 3 — Probes (P2)

- [X] T013 [US3] `backend/internal/platform/store/store.go`: `Ready()`; `errs.Unavailable()` (FR-008)
- [X] T014 [US3] `backend/internal/app/app.go`: `GET /healthz`, `GET /readyz`, `GET /api/v1/environment`; remove `/api/health`; public rules in `access/routes.go`; contract paths and `Environment` schema (FR-007–FR-009)
- [X] T015 [US3] `backend/cmd/seed/main.go` asks `/healthz` for 204; `frontend/playwright.config.ts` waits on `/healthz`; `frontend/src/app/DevOnly.tsx` asks `/api/v1/environment` (FR-009)
- [X] T016 [US3] Tests: probes answer 204; `TestReadyz_DataFolderGone_Answers503_022_FR008`

---

## Phase 4: Polish

- [X] T017 [P] `docs/architecture/overview.md` (`/api/v1`), `docs/sdd/feature-map.md` 022 row
- [X] T018 Run every gate locally: `gofmt -s -l .`, `go vet ./...`, `go test -race ./...`; `npm ci`, `npm run check`, `npm run build`, `npm run test:e2e` (SC-004)
- [ ] T019 Commit, push, open the pull request into `develop` (stacked on 019)

---

## Dependencies & Execution Order

- T001–T003 first; T004–T006 follow them. Phase 2 and Phase 3 depend on Phase 1's paths only and touch
  different files except `access/routes.go` and `app.go`. T018 follows everything; T019 follows T018.
