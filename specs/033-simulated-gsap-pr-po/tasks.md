---

description: "Task list for feature 033 GSAP PR Number and Purchase Order from a Simulated GSAP"
---

# Tasks: GSAP PR Number and Purchase Order from a Simulated GSAP

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml), [contracts/routes.md](contracts/routes.md),
[quickstart.md](quickstart.md)

**Prerequisites**: 032 merged (its `GSAP` port, `RoutedPR`/`GSAPOrder`, GSAP columns, waiting-for-GSAP rule,
`routes_purchase_requests.go`); 030 merged (SWBS facts; maybe `internal/simulatedgsap`).

**Tests**: required (constitution VI); each test first, seen failing. Names carry the FR:
`TestOrders_Numbers/033 FR-002 …`, `describe("033 FR-001: …")`.

**Tracks**: **[BE]** backend (API port 18133, databases `mapp_f033`, `mapp_e2e_033`) and **[FE]** frontend
(MSW until the backend lands; e2e with the [BE] track merged, ports 18133/14133). No Git branches or commits.

## Format: `[ID] [P?] [Story?] [Track] Description`

---

## Phase 1: Setup

- [x] T001 [BE] Merge `specs/033-simulated-gsap-pr-po/contracts/openapi.yaml` into `backend/api/openapi.yaml` (merge rules 1–4: the operation, `SimulatedGsapOrderRequest`, `PurchaseRequestAccess.gsapOrder`, `PurchaseRequestLine.gsapPending`, `FinancialImport.source` + `simulatedGsap`); add the operation to `notYetServed`; add `"POST /api/v1/simulated-gsap/purchase-requests/{purchaseRequestId}/purchase-order": need(authz.FinanceImport)` to `backend/internal/access/routes_purchase_requests.go`. **Both tracks start after this task.**
- [X] T002 [P] [FE] `pnpm run api:types`; fix the exhaustive switch over the financial import source in `frontend/src/features/financials` (label "Simulated GSAP") and the MSW fixtures in `frontend/src/testing/purchase-requests.ts` for the two new fields.
- [x] T003 [P] [BE] Add `PrefixSimulatedGSAPPR = "GPR"` and `PrefixSimulatedPO = "GPO"` to `backend/internal/platform/ids/ids.go` and `Prefixes`; extend `ids_test.go`.
- [x] T004 [P] [BE] Write `backend/internal/platform/database/migrations/00021_simulated_gsap_orders.sql`: re-create the `financial_imports.source` check as `source IN ('file', 'demo', 'simulatedGsap')`; run `sqlc generate && sqlc vet`.

## Phase 2: Foundation

- [x] T005 [BE] Test first `backend/internal/financials/simulated_test.go` `TestRecordSimulatedGSAP` (033 FR-005, FR-007, FR-011: new purchase-order rows linked by tracking key HX…; the same key with a new amount updates, never adds; one import with source simulatedGsap, feed "Simulated GSAP purchase order" and the PR ID as file name; a `financialImport` log entry; a refused row (currency not set up) fails the whole delivery and books nothing; refused without finance.import); then `SourceSimulatedGSAP` in `model.go`, `feed_rows.go` (`rowsFeed`) and `simulated.go` (`SimulatedDelivery`, `RecordSimulatedGSAP`); the contract's source label in `views.go` if one is mapped there.

---

## Phase 3: User Story 1 — The simulated GSAP creates the GSAP PR and the PO (P1) 🎯 MVP

**Goal / Independent Test**: spec US1.

### Tests first

- [x] T006 [P] [US1] [BE] `backend/internal/simulatedgsap/purchase_order_test.go` with fake `Numbers` and `Ledger`: `TestOrders_Numbers` (033 FR-002, FR-003, SC-004: SIM-0010001001 and SIM-4500001001 from counters 1001; lines 10 and 20 in line order; stored numbers kept), `TestOrders_Answer` (033 FR-003: status Released, the clock's date, value = amount in the line's currency), `TestOrders_Booking` (033 FR-005: one booked line per sent line with HX key, SWBS code, vendor code and name, short text, PR ID; the requesting actor).
- [x] T007 [P] [US1] [BE] `backend/internal/purchaserequest/gsap_order_test.go` with a fake `GSAP`: `TestCreateGSAPOrder_Gate` (033 FR-001, FR-006, FR-010: Created, Submitted, In review, Declined, Closed, Approved not routed, change pending, already ordered, switched off, not connected — each with its message; version conflict 409; without finance.import 403; out of scope 404), `TestCreateGSAPOrder_StoresAnswer` (033 FR-002–FR-004: header GSAP fields and source set; lines' PO fields; `gsapPending` false afterwards; the PR leaves waiting for GSAP), `TestCreateGSAPOrder_GSAPErrorChangesNothing` (033 FR-009), `TestCreateGSAPOrder_LogEntry` (033 FR-008), and in `access_test.go` `TestAccess_GSAPOrder` (033 FR-001: hidden without finance.import; reasons otherwise).
- [x] T008 [P] [US1] [BE] `backend/internal/app/simulated_gsap_orders_test.go` `TestSimulatedGSAPOrder_EndToEnd` through `apitest` (033 FR-005, FR-012, SC-001, SC-002): create, submit, approve and route a PR on a new line item; run as the requisitioner; the PR's GSAP fields; the line item's `GET /api/v1/financials/records/{id}` Committed equals the line amount; `GET /api/v1/financials/transactions` has one PO line linked by tracking key; a second run is refused and adds nothing.
- [x] T009 [P] [US1] [BE] `backend/internal/simulatedgsap/purchase_order_handler_test.go` (package `simulatedgsap_test`) `TestCreateSimulatedGsapPurchaseOrder_Contract`: 200 with the PR, 403, 404, 409 envelopes validated against the contract.

### Implementation

- [x] T010 [US1] [BE] `backend/internal/simulatedgsap/purchase_order.go` (`Orders`, `NewOrders`, `CreatePurchaseOrder`), `purchase_order_numbers.go` (formats, `Released`), `purchase_order_ports.go` (`Numbers`, `Ledger`, `BookedOrder`, `BookedLine`, `PurchaseRequests`), `purchase_order_handler.go` (route, `SimulatedGsapOrderRequest`) — package doc comment if the package is new — T006, T009 pass.
- [x] T011 [US1] [BE] `backend/internal/purchaserequest/gsap_order.go` (`CreateGSAPOrder`, data-model §2.2), `access.go` (`gsapOrder`), views (`GSAPPending` per line, `access.gsapOrder`), messages of plan "Messages" — T007 passes.
- [x] T012 [US1] [BE] `backend/internal/app/simulated_gsap_orders.go`: the `Ledger` adapter (`BookedLine` → `financials.Fields`, data-model §2.4) over `RecordSimulatedGSAP`, `prs.UseGSAP(simulatedgsap.NewOrders(…))`, the handler's registration; remove the operation from `notYetServed` — T008 passes.

### Frontend (tests first)

- [X] T013 [P] [US1] [FE] `frontend/src/features/purchase-requests/GsapOrderDialog.test.tsx` and `GsapOrderButton.test.tsx` (`describe("033 FR-001/FR-006/FR-010: Create GSAP PR and PO (simulated)")`): shown when `access.gsapOrder` is available, explained when unavailable with a reason, hidden when the reason is empty; the confirmation lists the lines with `gsapPending` and their amounts as sent; Create posts the PR's version; pending blocks a second click; 403, 404 and 409 answers shown; axe.
- [X] T014 [P] [US1] [FE] `GsapFields.test.tsx` and `PurchaseRequestLinesTable` test (`describe("033 FR-004/FR-011: GSAP answer labelled Simulated GSAP")`): GSAP PR number and date with "Simulated GSAP", PO number, line, status, date and value per line, "Filled by GSAP" tips; `features/financials` import history test for the source label.
- [X] T015 [US1] [FE] `GsapOrderButton.tsx`, `GsapOrderDialog.tsx` and `useCreateGsapOrder` in `api.ts` (invalidates the PR, the list, the line-item and activity sections and the `['financials']` queries), placed on `PurchaseRequestPage.tsx` as a secondary action.
- [X] T016 [US1] [FE] Source labels in `GsapFields.tsx`, `PurchaseRequestLinesTable.tsx` and `features/financials`.

**Checkpoint**: the PR→PO journey works in the API and on screen.

---

## Phase 4: User Story 2 — Top-up after an approved change (P2)

- [x] T017 [P] [US2] [BE] Tests first: in `gsap_order_test.go` `TestCreateGSAPOrder_TopUp` (033 FR-007: after an approved change only the changed and the added lines are sent; GSAP PR and PO numbers kept; unchanged lines untouched; a line added gets PO line 20; log entry "top-up"); in `purchase_order_test.go` `TestOrders_TopUp` (existing PO and lines kept, new line after the highest); extend `TestSimulatedGSAPOrder_EndToEnd` (033 SC-002: after the top-up still one transaction per PO line, Committed equals the new amount).
- [x] T018 [US2] [BE] Make T017 pass in `gsap_order.go` and `purchase_order.go` (sending only `gsapPending` lines; numbering what is missing).
- [X] T019 [P] [US2] [FE] Test first (`describe("033 FR-007: top-up")`): the dialog says "The simulated GSAP will update PO SIM-…" and lists the changed and added lines; then the text in `GsapOrderDialog.tsx`.

---

## Phase 5: Polish

- [x] T020 [FE] End-to-end `frontend/e2e/pr-to-po.spec.ts` (033 FR-012, P1, axe on every page; ports 18133/14133, database `mapp_e2e_033`; needs the [BE] track merged): Priya creates an activity in P-1001 with a line item "Gift boxes" (10,000.00 USD, Request SWBS code on) and submits it; Daniel approves it; Lea runs the simulated SWBS batch (030); Priya raises the PR (MoA approver Daniel, Email group DE_LUBES) and submits it; Daniel approves it from the Approvals page; Lea opens Purchase requests › Waiting for GSAP, downloads the PR form, runs Create GSAP PR and PO (simulated); the PR shows SIM- numbers labelled "Simulated GSAP"; the line item's Financials shows Committed 10,000.00 USD; the Financials page lists the PO line.
- [x] T021 [BE] Product documents: `docs/product/glossary.md` (Simulated GSAP purchase order, GSAP PR number, PO line), `docs/product/domain-model.md` (the PR's GSAP fields filled by the simulated GSAP; top-up), `docs/product/demo-users-and-data.md` (the requisitioner's run).
- [x] T022 [BE] Gates: `gofmt -l`, `golangci-lint run --new ./...`, `sqlc generate && sqlc vet`, `go vet ./...`, `go test -race ./...` with `TEST_DATABASE_URL`, `govulncheck ./...`; comment budget and file length of every touched file; ≥ 80% statement coverage on new code.
- [X] T023 [FE] Gates: `pnpm run api:check`, `typecheck`, `lint`, `format:check`, `test:coverage`, `build`.
- [x] T024 [BE] Run quickstart.md's walkthrough on port 18133 and record each answer.

---

## Dependencies and execution order

T001 first; [BE]: T003, T004 → T005 → US1 (T006–T012) → US2 (T017–T018) → T021, T022, T024. [FE]: T002 → US1
(T013–T016) → US2 (T019) → T023; T020 last, after the [BE] track is merged. US2 depends on US1.

## Parallel examples

After T001: T003, T004 [BE] with T002 [FE]; then T006–T009 [BE] with T013, T014 [FE].

## Implementation strategy

MVP = US1 (with 032's US1–US3 it completes the product owner's PR→PO journey); US2 adds the top-up.

**Backend note (implementation).** T007–T009 and T017 run through `apitest` in
`backend/internal/purchaserequest/handler_gsap_test.go` (end to end, top-up, refusals 403/404/409, not routed);
the numbering and ledger refusal are unit-tested in `simulatedgsap/purchase_order_test.go`, the import in
`financials/simulated_test.go`. T024 ran on port 18132 with the 032 database (research R9).

## Converge (frontend track)

- [x] T025 [FE] T020: run `e2e/pr-to-po.spec.ts` once the [BE] track is merged (the PO number is read from the page, so the SIM- sequence may be at any value). — Run on 2026-10-29 against the real API and PostgreSQL (database `mapp_e2e_032`, ports 18222/14222): passes unchanged (with 032 T100).
