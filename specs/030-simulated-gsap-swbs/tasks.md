---

description: "Task list for feature 030 — SWBS codes from a simulated GSAP"
---

# Tasks: SWBS Codes from a Simulated GSAP

**Input**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/](contracts/), [research.md](research.md). **Prerequisites**: features 007, 008, 014, 015 built.

**Tests**: required (constitution VI). In every task, write the named tests first and see them fail. Test
names carry the requirement ID (`030 FR-007`).

**State (2026-10-27)**: every task done; analyze reported no CRITICAL finding (one MEDIUM — FR-003's move
and restore had no test — fixed in T022); converge found nothing left to build.

**Organization**: Setup and foundation (schema, contract, line-item fields), then the stories in priority
order, then polish. Every file a task touches is brought within the comment budget (≤ 5 % of its
non-blank lines) and the 300-line limit.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [x] T001 Apply the shared base patch, set `.specify/feature.json` to `specs/030-simulated-gsap-swbs`, install the web app's packages offline.
- [x] T002 [P] Add the contract of [contracts/swbs-runs.openapi.yaml](contracts/swbs-runs.openapi.yaml) to `backend/api/openapi.yaml` (tag Simulated GSAP, both operations, `SwbsRun*` schemas, `SwbsStatus.received`, the four new `LineItem` fields); regenerate `frontend/src/api/generated/schema.d.ts` (`pnpm run api:types`).
- [x] T003 [P] Migration `backend/internal/platform/database/migrations/00017_swbs.sql` (line-item columns and constraints, `swbs_runs`, `swbs_run_codes`); queries `queries/swbs_runs.sql` and the new columns in `queries/line_items.sql`; `sqlc generate && sqlc vet`.
- [x] T004 [P] Prefix `SWR` in `backend/internal/platform/ids/ids.go`.

## Phase 2: Foundation — the line item's SWBS fields (blocks every story)

- [x] T005 Tests first in `backend/internal/lineitem/swbs_test.go`: `TestSwbsStatusOf_030_FR001` (table: received wins with the flag on, with the activity declined; not requested; ready for approved; waiting for created, submitted, declined), `TestHXKey_030_FR016`, `TestReceiveSwbsCodes_030_FR002` (code and both times set, version + 1, updated by the person; a line item that already has a code or is gone fails and nothing is saved).
- [x] T006 `backend/internal/lineitem/swbs.go`: `Swbs`, `SwbsReceived`, `SwbsStatusOf` (moved from `completeness.go`), `HXKey`, `SwbsCodeReceived`, `ReceiveSwbsCodes`; embed `Swbs` in `LineItem` and `View` (`model.go`), `View.HXKey` (`view.go`); `store_postgres.go` reads and writes the three columns. Bring `model.go`, `view.go`, `completeness.go` within the comment budget.

## Phase 3: User Story 1 — Send the waiting SWBS requests (P1) 🎯 MVP

- [x] T007 [US1] Tests first in `backend/internal/swbs/simulated_test.go`: `TestSimulated_030_FR007` (table: first GB01 is 0001; continues after codes already issued; two company codes in one run numbered separately; empty company code → `SIM-0000-nnnn`).
- [x] T008 [US1] Tests first in `backend/internal/swbs/service_test.go` (fakes: `store_fake_test.go`, a fake `LineItems`): `TestRun_SendsReady_030_FR006` (only ready line items, in ID order, codes received, run kept with source simulatedGsap, one log entry with the codes), `TestRun_NothingReady_030_FR009` (empty run kept, "Nothing was ready to send"), `TestRun_Named_030_FR008` (table: unknown → not_found; not ready → not_allowed naming it, nothing saved; ready → only it), `TestRun_Validation_030_FR008` (empty list, 201 IDs, long ID), `TestRun_Forbidden_030_FR012`, `TestRun_ReceiveFails_030_FR006` (nothing saved), `TestRuns_030_FR011` (newest first, total, ready count, limit validation).
- [x] T009 [US1] Package `backend/internal/swbs`: `model.go`, `gsap.go`, `simulated.go`, `ports.go`, `store.go`, `store_postgres.go`, `service.go`, `texts.go`, `kinds.go` (kind `swbsRun`), `handler.go`, `demo.go` (`LoadDemo`), fake `swbstest/store.go` and `store_fake_test.go`.
- [x] T010 [US1] Store wiring: `backend/internal/stores/swbs.go` (`SwbsStore`), `stores.go`, `postgres.go`, `storestest/storestest.go`.
- [x] T011 [US1] Tests first in `backend/internal/app/swbs_test.go`: `TestSwbsCandidates_030_FR007` (company code from the activity's "Company code" attribute; none → ""; only activities in scope; named IDs kept; ready worked out by `lineitem.SwbsStatusOf`).
- [x] T012 [US1] `backend/internal/app/swbs.go`: `swbsLineItems` (Candidates, Receive) and `swbsArea()`; add the step in `app/areas.go`.
- [x] T013 [US1] Access: `backend/internal/access/routes_gsap.go` (`need(finance.import)` for both operations), joined in `routes.go`; log visibility of kind `swbsRun` in `backend/internal/app/history_access.go` with `TestHistoryAccess_SwbsRun_030_FR010` in `app/history_access_test.go`. Bring `routes.go` and `history_access.go` within the comment budget.
- [x] T014 [US1] Contract tests in `backend/internal/swbs/handler_test.go` on the demo data: `TestSwbsRuns_Contract_030_FR011` (GET: SWR-1001, readyToSend 1; limit 0 → 400), `TestSwbsRun_Contract_030_FR006` (POST `{}` → 201, Stand build SIM-GB01-0001, GET line item shows received; second POST → empty run), `TestSwbsRun_Named_Contract_030_FR008` (404, 409, 400), `TestSwbsRuns_Forbidden_030_FR012` (Priya → 403 on both).
- [x] T015 [US1] Demo data: `backend/seed/swbs.go` (`addSwbsRun` through `swbs.LoadDemo`: Venue hire, DE03, as MAPP at the seed time, no log entry), called first in `seed/financials.go`; `backend/internal/financials/feed_demo.go` codes `SIM-DE03-0001` (Venue hire) and `SIM-GB01-0001` (Stand build); update `financials/{import,transactions,record,handler}_test.go`; the demo run is proved by `TestLoadDemo_030_FR021` and the contract tests `TestSwbsRuns_Contract_030_FR011`, `TestSwbsCode_LineItemSeed_030_FR021` (a separate seed test would repeat them).
- [x] T016 [P] [US1] Web tests first in `frontend/src/features/simulated-gsap/SimulatedGsapPanel.test.tsx` (`030 FR-013`): loading, error with retry, forbidden sentence, no runs yet, history success, nothing ready → action unavailable with the reason, send → confirmation text → POST `{}` → result list and refreshed count, refused → message; axe on each state.
- [x] T017 [US1] `frontend/src/features/simulated-gsap/`: `index.ts`, `api.ts` (`useSwbsRuns`, `useSendSwbsRequests` with invalidation), `SimulatedGsapPanel.tsx`, `SwbsRunList.tsx`, `simulated-gsap-text.ts`, CSS module; render the panel in `frontend/src/features/financials/FinancialsPage.tsx` for people with `finance.import` (bring it within the comment budget); MSW fixtures in `frontend/src/testing/fixtures/swbs-runs.ts`.

## Phase 4: User Story 2 — See where a line item's request stands (P1)

- [x] T018 [P] [US2] `TestSwbsChip_030_FR017` in `backend/internal/allwork/status_test.go`; `received` chip ("SWBS code received", tone done, order 3) in `allwork/status.go` (comment budget).
- [x] T019 [P] [US2] Web tests first in `frontend/src/features/line-items/SwbsCard.test.tsx` (`030 FR-014`, `FR-001`): each of the four statuses with its words and explanation; received shows code, Requested, Received and "Simulated GSAP"; not received shows "Not received yet"; axe.
- [x] T020 [US2] `frontend/src/features/line-items/swbs-text.ts` (SWBS texts moved out of `line-item-text.ts`, which is brought within the comment budget), `SwbsChip.tsx` (received), `SwbsCard.tsx`, rendered in `LineItemPage.tsx` (comment budget); fixtures in `src/testing/fixtures/line-items.ts` gain the new fields.
- [x] T021 [US2] Financials section without the transactions' SWBS line (FR-019): `FinancialsSection.tsx`, `gsap-financials.ts`, their CSS and tests (comment budget).

## Phase 5: User Story 3 — A received code stays (P1)

- [x] T022 [US3] Tests first: `TestUpdate_KeepsSwbsCode_030_FR004` (an edit keeps code and times; switching the flag off is a `requestSwbs` problem with the message, reported with the other problems) in `backend/internal/lineitem/swbs_update_test.go`; `TestDuplicate_DropsSwbsCode_030_FR005` and `TestCopyLineItems_DropsSwbsCode_030_FR005`; `TestSwbsStatus_ActivityLeavesApproval_030_FR003` (restart keeps code and status received); `TestMoveAndRestore_KeepSwbsCode_030_FR003` (a move and a delete + undo keep code and times).
- [x] T023 [US3] `backend/internal/lineitem/update.go` (carry `Swbs`, the lock), `duplicate.go` (`copyOf` clears `Swbs`) — both within the comment budget.
- [x] T024 [US3] Web: the form locks "Request SWBS code from GSAP" with the reason and shows the received code (`LineItemFormCards.tsx`, `LineItemForm.tsx` passes the stored code; comment budget); test `030 FR-004, FR-018` in `frontend/src/features/line-items/LineItemEditPage.swbs.test.tsx`; the SWBS code's default field help (`settings/fieldhelp_catalogue.go`) follows.

## Phase 6: User Story 4 — Send one line item now (P2)

- [x] T025 [US4] Web tests first in `frontend/src/features/simulated-gsap/SendNowButton.test.tsx` (`030 FR-015`): confirmation text, POST with the line item's ID, success refreshes, 409 shows the refusal; axe. `SwbsCard` offers it only when ready and the person holds `finance.import` (`SwbsCard.test.tsx`).
- [x] T026 [US4] `frontend/src/features/simulated-gsap/SendNowButton.tsx`, exported from `index.ts`, used by `SwbsCard`.

## Phase 7: User Story 5 — HX key (P2)

- [x] T027 [US5] Web tests first in `frontend/src/features/line-items/HxKey.test.tsx` (`030 FR-016`): label, read-only value, Copy writes HX1006 and toasts "HX key copied", a failed copy says to select it; axe.
- [x] T028 [US5] `frontend/src/features/line-items/HxKey.tsx` used by `SwbsCard`.

## Phase 8: End to end and polish

- [x] T029 E2E (run with the whole suite: 44 of 44 journeys pass on `mapp_e2e_030`) `frontend/e2e/simulated-gsap-swbs.spec.ts` (US1–US3, US5): Alex sends Stand build alone from its page (SIM-GB01-0001); Daniel approves Trade magazine ads; Alex's Financials panel shows 1 ready, sends, reads SIM-NL02-0001; Media booking shows received, its HX key and the locked switch; Venue hire shows SIM-DE03-0001; Priya sees no panel; axe on each page.
- [x] T030 [P] Docs: `docs/product/field-catalogue.md` §5 (#13, #16, HX key), `domain-model.md` §2.4 and §4.6, `glossary.md` (SWBS code, HX key, Simulated GSAP).
- [x] T031 Gates: `gofmt`, `golangci-lint run --new ./...`, `sqlc generate && sqlc vet`, `go vet ./...`, `go test -race ./...` with `TEST_DATABASE_URL`, `govulncheck ./...`; `pnpm run api:check`, `typecheck`, `lint`, `format:check`, `test`, `build`; comment budget and file length of every touched file.
- [x] T032 Run the app on 18130/14130 with `mapp_f030`, exercise both operations with curl and record what was seen (quickstart).
- [x] T033 [FR-022] `features/simulated-gsap/GsapProgress.tsx` and `pace.ts`: the send closes its dialog, shows
  the progress animation for at least 1.5 s, then the code (SendNowButton and SimulatedGsapPanel; tests with a
  short pace, reduced motion in CSS)
