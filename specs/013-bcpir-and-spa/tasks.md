---
description: "Task list for feature 013 SP&A Plan View and BCPIR Marker"
---

# Tasks: SP&A Plan View and BCPIR Marker

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml), [contracts/routes.md](contracts/routes.md),
[quickstart.md](quickstart.md)

**Tests**: required (constitution VI). Write each test first and see it fail. Go test names carry the FR, e.g.
`TestHeadline/013 FR-015 …`; web tests `describe('013 FR-015: …')`.

## Format: `[ID] [P?] [Story?] Description`

---

## Phase 1: Setup

- [ ] T001 Merge the contract (contracts/openapi.yaml, rules 1–4) into `backend/api/openapi.yaml`; add the tag;
  add `getSpaView` to the served operations check in `backend/internal/app/app_test.go` if it lists them.
- [ ] T002 [P] Regenerate the web app's API types (`pnpm run api:types`) and fix type errors.

## Phase 2: Foundation

- [ ] T003 [P] `backend/internal/financials`: test first `TestFactsByLineItemMonth/013 FR-010 …` (by posting
  month, currencies no longer in Settings left out, not-linked left out), then `FactsByLineItemMonth` sharing the
  loop of `FactsByLineItem`.
- [ ] T004 [P] `backend/internal/purchaserequest`: test first `TestDrawnOnLineItems/013 FR-021 …` (only lines on
  the given line items, USD at today's rates, view scope applied by the store), then `drawn.go`
  (`DrawnOnLineItems`, `Drawn`, `DrawnLine`).
- [ ] T005 Create `backend/internal/spa`: `model.go` (API types of data-model §3), `ports.go` (Planning, Finance,
  Purchases, Displays and the input types), `service.go` (`NewService`, `View`), package doc; `spatest/` fakes of
  the ports for the handler and contract tests.
- [ ] T006 `backend/internal/access/routes_spa.go` with the rule of contracts/routes.md; join it in `routes.go`.

## Phase 3: User Story 1 — SP&A view of a plan or campaign (P1) 🎯 MVP

- [ ] T007 [P] [US1] Test first `headline_test.go` (FR-015, FR-016: scenarios 1–3, percent rounding, overspend),
  then `headline.go` and `percent.go`.
- [ ] T008 [P] [US1] Test first `periods_test.go` (FR-017, FR-010: scenario 4, balance over months, outside the year,
  no dates, negative balance, months add up to quarters), then `periods.go` and `months.go`.
- [ ] T009 [P] [US1] Test first `dimensions_test.go` (FR-018: scenarios 5–6, several values, not filled in, not yet
  itemised, order), then `dimensions.go`.
- [ ] T010 [US1] Test first `activities_test.go` (FR-019: rows, line items, start quarter, channel) and
  `trace_test.go` (FR-007: every trace sums to its figure, property test), then `activities.go` and `trace.go`.
- [ ] T011 [US1] Test first `service_test.go` (FR-005, FR-008, FR-009, FR-012: campaign scope, campaign of another
  plan 404, plan out of scope 404, unknown dimension 400, no finance.view, display currency, data as of, empty
  plan), then the service's assembly.
- [ ] T012 [US1] `handler.go` + `handler_test.go` (external package: query parsing, 400, 404) and the contract test
  of `getSpaView` on the demo data (`backend/internal/app` contract tests or `spa` handler test through `apitest`).
- [ ] T013 [US1] `backend/internal/app/spa*.go`: the adapters (planning over All work's `PlanYear`, attributes and
  activity types; finance over `financials` and 015's access; purchases over 032) and the wiring.
- [ ] T014 [P] [US1] Web: `frontend/src/features/spa/` — `index.ts`, `api.ts` (`useSpaView`), `schemas.ts` (URL
  state, fast-check property), `spa-text.ts`, `SpaPage.tsx` with header, scope picker, headline cards, periods
  card, dimension card, activity table, trace dialog; route in `app/routes.tsx` and `app/pages.tsx`.
- [ ] T015 [US1] Web tests: `SpaPage.test.tsx` (loading, error, not found, empty, success; overspend; scope switch;
  trace dialog; sort; line items; axe).
- [ ] T016 [US1] Entry points: "SP&A view" on the plan page header and on the campaign panel.

## Phase 4: User Story 2 — BCPIR marker and filter (P1)

- [ ] T017 [P] [US2] Backend Plans list: test first `TestList/013 FR-003 …` (Only, Exclude, wrong value 400, marker),
  then `PlanSummary.BCPIR` and the `bcpir` filter.
- [ ] T018 [P] [US2] Backend All work: test first `bcpir_test.go` (filter Only/Exclude, row marker, saved view keeps
  it, export note, Breakdown grouping Yes/No), then `Node.BCPIR`, `Row.BCPIR`, query, tree filter, views, export
  notes, grouping; set `BCPIR` in `app/allwork_source.go`.
- [ ] T019 [P] [US2] Web Plans list: marker beside the name, filter "BCPIR plans" with its chip; plan header marker;
  tests.
- [ ] T020 [P] [US2] Web All work: filter, chip, URL state, saved view, row marker; tests.
- [ ] T021 [US2] Seed: FY2026 Lubricants Europe is a BCPIR plan; adjust the tests that read its flag.

## Phase 5: User Story 3 — Budget and procurement (P2)

- [ ] T022 [US3] Test first `panels_test.go` (FR-020, FR-021: scenarios 1–5, lines in scope only, no finance.view)
  and `raise_test.go` (FR-022), then `panels.go` and the raise availability.
- [ ] T023 [US3] Web: budget panel, procurement panel (open a PR), Raise purchase request menu; tests (scenarios 6–8).

## Phase 6: Polish

- [ ] T024 End-to-end `frontend/e2e/spa.spec.ts`: US1 (plan view, quarter/month, dimension, trace, campaign scope,
  Back) and US2 (Plans list and All work filter, marker) with axe.
- [ ] T025 Docs: glossary (BCPIR, SP&A, Actual spend), field catalogue §1 #11, design system (SP&A page), feature map
  row "Built — …".
- [ ] T026 Gates: `golangci-lint fmt` (no diff), `golangci-lint run --new-from-rev=587eade2`, sqlc generate + vet (no
  diff), `go vet`, `go test -race` with `TEST_DATABASE_URL`, govulncheck, `pnpm run check`, `pnpm run build`,
  full e2e; comment budget and 300-line limit on every touched file.

## Dependencies

T001 → everything; T003–T006 → US1; US1 backend (T007–T013) → T014–T016 can start on MSW after T002; US2
independent of US1; US3 after T011. T024 after US1 and US2.
