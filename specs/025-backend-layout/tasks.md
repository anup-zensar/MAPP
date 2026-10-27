# Tasks: Backend Layout

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md)

Each task says which helper area did it (plan.md, "Helper areas"). `[x]` done, `[ ]` open.

## Phase 1: Setup

- [x] T001 Branch `feature/025-backend-layout` from `develop`; spec, checklist and plan
- [x] T002 List the Go files over 300 non-blank lines (113) with their planned splits (research.md R1)

## Phase 2: Foundation (lead)

- [x] T003 Each area's `repository.go` becomes `store.go` and its interface `Store`; the JSON stand-in is the
  exported value type `JSONStore` (FR-002)
- [x] T004 `internal/stores/<area>.go` declares the store each area's wiring and seed use (`stores.<X>Store`);
  `stores.Set`, `internal/app`, `seed` and `internal/platform/apitest` use them (FR-002)
- [x] T005 `auth.People`, `approvals.People` and the seeded workflows' reader replace other areas' stores
  inside the areas (FR-002)
- [x] T006 `history.Tools` bundles the transaction, IDs, activity log and clock for every constructor (FR-002)

## Phase 3: User Story 2 — each service names what it needs (P1)

- [x] T007 [US2] Every `NewService` within four parameters, callers updated (all helpers, round 1)
- [x] T008 [US2] Every area's `Store` trimmed to what its service calls; the rest in `stores.<X>Store`
  (round 1; verified for every area, research.md R2)

## Phase 4: User Story 1 — every file within 300 lines (P1)

- [x] T009 [US1] settings (rounds 1 and 2: settings-services, settings-rules, settings-handler, r2-settings)
- [x] T010 [US1] approvals (approvals-rounds, approvals-workflows, r2-approvals)
- [x] T011 [US1] allwork and home (allwork-views, allwork-home, r3-allwork-finance)
- [x] T012 [US1] plans and rollup (plans, r3-activities-plans)
- [x] T013 [US1] campaigns (campaigns, r2-history-campaigns)
- [x] T014 [US1] activities (activities, r3-activities-plans)
- [x] T015 [US1] lineitems (lineitems, r3-allwork-finance)
- [x] T016 [US1] people, access, auth and samples (people, access, r3-people-access)
- [x] T017 [US1] history (history, r2-history-campaigns)
- [x] T018 [US1] financials and budgets (finance, r3-allwork-finance)
- [x] T019 [US1] seed (seed-store, r3-people-access)
- [x] T020 [US1] `internal/platform/store` (r4-store)
- [x] T021 [US1] `internal/app` (r4-app)
- [x] T022 [US1] Old lint findings in moved code fixed, 0 on new lines (every helper)

## Phase 5: User Story 3 — fakes, no JSON store, singular names (P2)

- [ ] T023 [US3] Service tests on hand-written fakes of each area's `Store` (research.md R4)
- [ ] T024 [US3] Handler and contract tests without the JSON stand-in; delete `internal/platform/store`
  and every `store_json*.go` (research.md R4)
- [ ] T025 [US3] Singular package names in one mechanical commit, then the lint findings it surfaces
  (research.md R5)

## Phase 6: Polish

- [x] T026 Gates: format, vet, lint on new lines, sqlc, `go test -race`, `go test -short`, govulncheck
- [ ] T027 End-to-end suite on PostgreSQL and `pnpm run api:check`
- [x] T028 Feature map row 025
