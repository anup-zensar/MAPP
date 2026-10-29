---

description: "Task list for feature 015 Financial Transactions"
---

# Tasks: Financial Transactions

> **Build state after the final merge** (`plan-module` at `a1ae8fe`, 21 October 2026): the backend is complete.
> The web tasks were built by the UI lanes, partly under other names: `features/plans/RecordFinancials.tsx`
> (plan, campaign and initiative), `features/line-items/GsapFigures.tsx` and `FinancialsSection.tsx` (activity
> and line item), `features/home/FinancialsCard.tsx` (Home) and `features/financials/` (page, details panel,
> import, import history, link). **Open:** T044 (the Home card has
> no no-access state yet); T045 (the linked-transactions sentence is in the line-item and activity delete
> dialogs only, not yet in the campaign, initiative and plan dialogs); T048 (product documents); T049, T050.

**Input**: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml)

**Tests**: Tests are required (constitution V). Write each test first and see it fail. Test names carry
the FR, for example `TestMatch_HXKeyBeforePO_FR007`.

**Lanes**: one backend agent (B) and one frontend agent (W) work in parallel from the merged contract
(D-200). Toolchain: Git Bash, `export PATH="$HOME/tools/node:/c/Program Files/Go/bin:$PATH"`. Write
files with the editor tools.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1–US7 of spec.md; Setup, Foundation and Polish tasks have none

---

## Phase 1: Setup

- [x] T001 (done by the consolidation, D-298) Merge `specs/015-financial-transactions/contracts/openapi.yaml` into `backend/api/openapi.yaml`
  following its four merge rules. Add the 10 operations to `notYetServed` in
  `backend/internal/app/app_test.go`. Run `go test ./internal/app/...` and the web app's `npm run
  api:check` to regenerate `frontend/src/api/` types.
- [x] T002 [P] Add `PrefixFinancialTransaction = "FT"` and `PrefixFinancialImport = "FIM"` to
  `backend/internal/platform/ids/ids.go` and to `Prefixes`, with tests in `ids_test.go`.

---

## Phase 2: Foundation (blocks every story)

### Backend (B)

- [x] T003 [P] Write `backend/internal/financials/model.go`: the types of data-model §1–§2, the API view
  structs named as the contract schemas, and the readable name function. Write `repository.go` and
  `store_json.go` (tables `financial_transactions`, `financial_imports`). Write `store_json_test.go`,
  which checks a round trip and the invariants of data-model §1.
- [x] T004 [P] Write `backend/internal/financials/feed.go` (`Feed`, `Delivery`, `Incoming`,
  `RowProblem`) and `feed_demo.go`, which holds the 11 rows of data-model §6, in file order, as
  `Incoming`. Write `feed_demo_test.go`.
- [x] T005 [P] Write `backend/internal/financials/ports.go`: `Planning`, `Node`, `LineItemState`,
  `Rates`, `Displays`, `Access` and `AllowAll{}`, as in plan *Ports*.
- [x] T006 Write the tests `backend/internal/financials/match_test.go` and the code `match.go` + `index.go`
  (FR-007, data-model §4):
  - the rule order and every reason;
  - POs first, IDs in processing order;
  - in-delivery references;
  - retry of not-linked transactions and the `Relinked` count;
  - manual links kept;
  - a bin line item still matches, an unknown one does not;
  - ambiguous PO and SWBS.
- [x] T007 Write the tests `backend/internal/financials/figures_test.go` and the code `figures.go`
  (FR-010–FR-014, data-model §5):
  - `usd` per transaction, half up;
  - credit notes;
  - the latest accrual month;
  - line-item facts and roll-up over `Planning.Subtree` (live only);
  - not yet committed, % half up and every `null` case;
  - above planned (record and line items below), `firstLineItemAbovePlanned`;
  - a missing currency left out and flagged;
  - one display conversion per figure.
- [x] T008 Write `backend/internal/financials/kinds.go` + `kinds_test.go`: the top-level kinds
  `financialImport` and `financialTransaction` (data-model §7), with `Describe` for `lineItemId`.
- [x] T009 Write `backend/internal/financials/service.go` (constructor, `apply` shared by Check, Import and
  LoadDemo, `Record`, `FactsByLineItem`, `DataAsOf`) and `texts.go` (every message of plan *User-visible
  messages* and data-model §4.2). Write `service_test.go` for `Record` against the data-model §6.1 values,
  using fakes for the ports.
- [x] T010 Write `backend/internal/app/financials.go`:
  - `planningAdapter` over plans, campaigns, activities and lineitems (same readers as
    `allwork_source.go`; 005 `rollup` for container planned cost; `history.SnapshotsInBin` for
    `LineItemState`);
  - adapters for `settings.Rates` and the people display currency.
  - Wire the service, kinds and routes in `app.go`.
  - Tests in `financials_test.go`: the adapter answers the seed's tree with the planned costs of
    data-model §6.1.
- [x] T011 Write `backend/seed/financials.go`: call `financials.LoadDemo` after approvals and before
  011's history step, in `seed/seed.go` (D-202), with no log entries. Add
  `TestSeed_FinancialsStory_015` in `seed/seed_test.go` for every figure and count of data-model §6.1
  (FR-031).

### Web (W)

- [x] T012 [P] Write `frontend/src/features/financials/financialsApi.ts` (queries, mutations and query keys
  of plan *Query keys*), `financialsText.ts` + `financialsText.test.ts` (every text of plan *User-visible
  messages*, plurals, type words, matched-by words), `financialsLinks.ts` (`financialsHref`) and
  `testData.ts` (the demo data's API answers for A-1002, A-1003, L-1003, L-1004, L-1006, P-1001, C Helix
  launch, the transactions page and the overview).
- [x] T013 [P] Write `frontend/src/features/financials/FinancialFigures.tsx` (+ `.module.css`,
  `.test.tsx`): tiles, ⓘ texts, the meter, red states with words, `null` states and a responsive wrap
  (FR-011, FR-013, FR-014).
- [x] T014 Write `frontend/src/features/financials/FinancialsCard.tsx` (+ `.module.css`, `.test.tsx`):
  - the variants `page` and `panel`, with breakdown headings per level;
  - latest transactions (activities);
  - the View all link and the source line;
  - the loading, error, empty and no-data states.

**Checkpoint**: the API answers `GET /api/financials/records/{id}` on the seed, and the card renders
from `testData`.

---

## Phase 3: User Story 1 - See an activity's and a line item's money (P1) 🎯 MVP

**Goal**: the activity page's second section and the line-item page show GSAP figures, open; the Line
items section shows Committed and Invoiced; no "From finance (coming later)" remains.

**Independent Test**: spec US1.

- [x] T015 [US1] (B) In `backend/internal/financials/handler.go` + `handler_test.go`, serve `GET
  /api/financials/records/{recordId}` with `apitest` contract checks, the 404 and the `not_allowed`
  answer (fake `Access`). Remove the operation from `notYetServed`.
- [x] T016 [US1] (B) Serve `GET /api/financials/transactions` with the `recordId` filter and paging (the
  rest comes in US3), so the line-item page can list its transactions. Add handler tests.
- [x] T017 [US1] (W) Activity page (FR-015, changes 006 FR-020):
  - in `frontend/src/features/activities/ActivityPage.tsx`, render `<FinancialsCard recordId={activity.id}
    variant="page" id="financials" />` right after `ActivityDetails`;
  - remove `FromFinance` from `ActivitySections.tsx` and its uses;
  - update `ActivityPage.test.tsx` and `ActivitySections.test.tsx`: the card is the 2nd section, open,
    and has no "From finance" region.
- [x] T018 [US1] (W) In `frontend/src/features/line-items/LineItemsSection.tsx` (+ `.module.css`,
  `.test.tsx`), add the Committed and Invoiced columns from the activity's record view (shared query
  key), with the red marker and the loading and error cells (FR-016).
- [x] T019 [US1] (W) Write `frontend/src/features/financials/LineItemFinancials.tsx` (+ `.test.tsx`):
  figures, the Transactions table (paged by 50) and the SWBS code line (FR-017). Use it in
  `LineItemPage.tsx` in place of `FromFinanceSection`.
- [x] T020 [US1] (W) Line-item form (FR-017, changes 007 FR-007/FR-025):
  - remove `FromFinanceSection` from `LineItemForm.tsx`;
  - delete `frontend/src/features/line-items/FromFinanceSection.tsx` and the three `FROM_FINANCE` texts
    in `lineItemText.ts`;
  - update `LineItemCreatePage.test.tsx`, `LineItemEditPage.test.tsx` and `LineItemPage.test.tsx`.
- [x] T021 [US1] (W) e2e journey 1 in `frontend/e2e/financials.spec.ts` (with axe on the activity page).
  - Done 2026-10-28. The line item's "SWBS code: …" line of scenario 6 is now 030's SWBS request card, so it is
    not checked here (030's e2e covers it).

**Checkpoint**: US1 works on the seed.

---

## Phase 4: User Story 2 - Money rolls up to initiative, campaign and plan (P1)

**Independent Test**: spec US2.

- [x] T022 [US2] (B) Add to `service_test.go` the roll-up equality for every container of data-model
  §6.1, the display-currency conversion (EUR reader), and the move of an activity to another initiative
  (its figures move) (FR-012, SC-002).
- [x] T023 [P] [US2] (W) In `frontend/src/features/plans/PlanPage.tsx` (+ test), add `<FinancialsCard
  recordId={plan.id} variant="page" />` right after `PlanSummaryCards` (FR-018).
- [x] T024 [P] [US2] (W) In `frontend/src/features/campaigns/CampaignPanel.tsx` and `InitiativePanel.tsx`
  (+ tests), add `<FinancialsCard variant="panel" />` after Budget and cost (FR-019).
- [x] T025 [US2] (W) e2e journey 2 in `frontend/e2e/financials.spec.ts`.
  - Done 2026-10-28. Two bugs found and fixed, each with a component test that failed first
    (`RecordFinancials.test.tsx`): **View all n transactions** on the plan, campaign and initiative linked to
    `/financials?record=…`, which the Financials page ignores, so it showed every transaction (scenario 5) — it
    now links to `?recordId=…`; and a breakdown row showed no marker when one of its line items was committed
    above its planned cost (scenario 3) — it now says "Committed above planned cost on 1 line item" (on the
    campaign and initiative rows too, as their line items include it).

---

## Phase 5: User Story 3 - Find POs, invoices, payments and accruals in one place (P1)

**Independent Test**: spec US3.

- [x] T026 [US3] (B) Complete `Transactions` in `service.go`:
  - type, q, year (plan year / posting year), planId and link filters;
  - counts per tab, totals without type, sort and choices.
  - Serve `GET /api/financials/transactions/{transactionId}`.
  - Tests: `service_test.go` for each filter and the seed's tab counts and totals (spec US3 scenario 1);
    `handler_test.go` for every parameter refusal (FieldError.field = name).
- [x] T027 [P] [US3] (W) In `frontend/src/app/navigation.ts` (+ its test), add the `FINANCIALS` item
  (`ReceiptText`) after All work. In `frontend/src/app/router.tsx`, add `/financials` and
  `/financials/transactions/:transactionId` (FR-020). In `frontend/src/lib/recordLinks.ts` (+ test), add
  `financialTransaction`.
- [x] T028 [US3] (W) Write `frontend/src/features/financials/FinancialsPage.tsx` (+ `.module.css`,
  `.test.tsx`) and `TransactionsTable.tsx` (+ test):
  - header, source line, tabs with counts, search and filters kept in the address;
  - record chip, summary cards, sortable table, pager, phone card layout and empty result (FR-021).
- [x] T029 [US3] (W) Write `frontend/src/features/financials/TransactionPanel.tsx` (+ test): fields,
  linked path, matched-by, import line and timestamps; actions hidden in this story (FR-022).
- [x] T030 [US3] (W) e2e journey 3, with axe on the Financials page and the panel (`frontend/e2e/financials-page.spec.ts`).
  - Done 2026-10-28. Bug found and fixed (test first in `FinancialsPage.test.tsx`): a not-linked row showed the
    chip but not its reason (scenario 4); `TransactionsTable` now shows the reason under the chip. The tab counts
    of scenario 1 are left to the component test: other journeys add transactions to the shared demo data.

**Checkpoint**: the three P1 stories are demonstrable; the MVP of BUS.FIN.080 is met.

---

## Phase 6: User Story 4 - Load GSAP data (simulated GSAP extract) (P2)

**Independent Test**: spec US4.

- [x] T031 [US4] (B) Write the tests `backend/internal/financials/feed_csv_test.go` (table-driven: every
  column rule and message of plan *Simulated GSAP extract*, BOM, quoting, header case, unknown columns,
  empty lines, row numbers, every whole-file refusal) and the code `feed_csv.go` (FR-004, FR-005).
- [x] T032 [US4] (B) Implement `Check` (discarded `Write`) and `Import`:
  - the currency and duplicate-key checks, counts and lists capped at 200;
  - the import record and one log entry (FR-027).
  - Tests in `service_test.go`: Check saves nothing; counts add up (SC-003); importing twice changes
    nothing; refused rows are skipped; the retry relinks; SC-006 with a second fake feed.
- [x] T033 [US4] (B) Write `template.go` and serve `GET /api/financials/imports`, `POST
  /api/financials/imports/check`, `POST /api/financials/imports` and `GET
  /api/financials/imports/template`, each with handler tests (FR-008, FR-009).
- [x] T034 [US4] (W) Write `frontend/src/features/financials/ImportDialog.tsx` (+ test):
  - choose, check, result and done steps; Download sample file;
  - the problem and not-linked lists; Import disabled with a reason;
  - toast and invalidation.
  - Write `ImportHistoryPanel.tsx` (+ test).
  - Wire both from `FinancialsPage`.
- [x] T035 [US4] (W) e2e journey 4 (generated 3-row file), with axe on the dialog (`frontend/e2e/financials-page.spec.ts`).
  - Done 2026-10-28: the file links to a line item the test creates (its HX key), not to Grand opening Shanghai,
    so the demo figures the other journeys read stay as seeded.

---

## Phase 7: User Story 5 - Financial columns and a filter in All work (P2)

**Independent Test**: spec US5.

- [x] T036 [US5] (B) Write the tests first, then change `backend/internal/allwork`:
  - `source.go`: `Node.Finance` with `POCount` and `InvoiceCount`;
  - `columns.go`: group `financials`, five columns, the new `DefaultColumns`, the group left out without
    access;
  - `rollup.go`: activity and container sums, not yet committed, above planned;
  - `query.go`: the `finance` filter;
  - `options.go`; `export.go`: the note.
  - Tests in `columns_test.go`, `rollup_test.go`, `query_test.go`, `export_test.go` and `handler_test.go`
    (FR-023, FR-024). The totals equal `financials` for the same plans (SC-002).
- [x] T037 [US5] (B) In `backend/internal/app/allwork_source.go` (+ test), fill `Node.Finance` from
  `financials.FactsByLineItem` for the year's line items.
- [x] T038 [P] [US5] (W) In `frontend/src/features/all-work/FilterMenu.tsx`, `FilterBar.tsx`,
  `viewState.ts`, `allWorkText.ts` and `allWorkApi.ts` (+ tests), add the Financials filter, its chips
  and its address parameter `finance`. Check that the column chooser shows the `financials` group from
  the options (no hard-coded list).

---

## Phase 8: User Story 6 - A Financials card on Home (P2)

**Independent Test**: spec US6.

- [x] T039 [US6] (B) Implement `Overview` (current year from the API clock, Home scope through
  `MyTeamPlanIDs`, activities above planned, not-linked count). Serve `GET /api/financials/overview`.
  Tests: seed values for Priya Raman (My teams) and Alex Morgan (Everything), and handler contract
  tests (FR-025).
- [x] T040 [US6] (W) Write `frontend/src/features/financials/FinancialsHomeCard.tsx` (+ test): figures,
  the two links, Open Financials, the source line, the empty state and `SectionError`. Place it right
  after `SummaryCards` in `frontend/src/features/home/HomePage.tsx` and update `HomePage.test.tsx`.

---

## Phase 9: User Story 7 - Link a transaction GSAP could not match (P3)

**Independent Test**: spec US7.

- [x] T041 [US7] (B) Implement `Link` and `Unlink`:
  - only not-linked / only manual;
  - a bin or unknown line item refused;
  - version conflict; one log entry each.
  - Serve both operations.
  - Tests in `service_test.go` and `handler_test.go`, including an import after a manual link (the link
    is kept) (FR-026, FR-027).
- [x] T042 [US7] (W) Write `frontend/src/features/financials/LinkTransactionDialog.tsx` (+ test): the ID
  field, the preview through `GET /api/line-items/{id}`, the refusals and `conflictMessage`. Add Link
  and Unlink (with `ConfirmDialog`) to `TransactionPanel.tsx` and its test.

---

## Phase 10: Polish and cross-cutting

- [x] T043 [P] (B) Access seam (FR-028): in `service_test.go`, a fake `Access` denying view, import and
  link, and `VisiblePlanIDs` scoping lists, counts, totals, the overview and `FactsByLineItem` users (All
  work hides the group).
- [x] T044 [P] (W) No-access states: the "You don't have access to financial data" text in
  `FinancialsCard`, `FinancialsPage`, `FinancialsHomeCard` and `LineItemFinancials`, shown on a 409
  `not_allowed` (tests with `mockApi`).
  - Done 2026-10-29. The refusal is **403 `forbidden`** (Consolidation, D-299), so every state answers that
    (`FinancialsSection` also keeps 409 `not_allowed`). Already built: the Financials page (tested), the activity
    and line-item sections (`FinancialsSection.tsx`; the line item's test added) and `RecordFinancials` (test
    added). Built: Home's card (`features/home/FinancialsCard.tsx`) shows the note in place of its figures,
    without Open Financials. Bug found by the new test and fixed: `RecordFinancials` retried a refusal twice, so
    the note came after about 3 s of skeleton; it now follows the app's rule (no retry).
- [x] T045 [P] (W) Delete dialogs (FR-029): add the linked-transactions sentence to
  `frontend/src/features/line-items/DeleteLineItemDialog.tsx`,
  `frontend/src/features/activities/DeleteActivityDialog.tsx`,
  `frontend/src/features/campaigns/DeleteRecordDialog.tsx` and the plan delete dialog in
  `frontend/src/features/plans/dialogs/`, from the record view's `transactionCount` (+ tests).
  - Done 2026-10-29. The line-item and activity dialogs already said it (tests in `LineItemPage.test.tsx`,
    `LineItemsSection.actions.test.tsx`, `DeleteActivityDialog.test.tsx`). Added to
    `features/plans/campaigns/DeleteRecordDialog.tsx` (campaign and initiative) and
    `features/plans/dialogs/DeletePlanDialog.tsx`, from the record view (the Financials card's query, so an open
    panel or page answers from its cache), with a test each; the product name on screen is SP&A One (D-332). The
    shared test fixtures `plan-actions.tsx` and `campaign-tree.tsx` answer the record view with no transactions.
- [x] T046 [P] (B) In `service_test.go`, test that copying a plan, campaign, activity or line item copies
  no transaction and the copy's figures are 0.00 (FR-030). Test that deleting a line item drops its
  amounts from every figure and restoring brings them back (FR-012).
- [x] T047 [P] (B) Stress tests (SC-004) in `backend/internal/financials/stress_test.go`: a 5,000-row
  import under 5 s and a 20,000-transaction list under 1 s, skipped by default as in 010.
- [ ] T048 [P] Documentation (no decision-log edits; the decisions are D-242–D-257):
  - `docs/product/domain-model.md`: rewrite §4.6 as "Financial figures from GSAP" (data-model §5) and add
    the two records to §7;
  - `docs/product/field-catalogue.md` §4.6 and §5 #16–17;
  - `docs/product/glossary.md`: Committed, Invoiced, Paid (actuals), Accrued, Purchase order, Invoice,
    Accrual, GSAP extract, HX key, Not linked;
  - `docs/sdd/feature-map.md` (015) and `docs/sdd/plan-module-build.md` §2, §4 and §5 rows for 015;
  - `docs/product/vision-and-scope.md` scope line.
  - Hand the constitution 1.1.0 amendment text of plan *Constitution Check* to the product owner.
- [ ] T049 Run every check: backend `gofmt -l .` (prints nothing), `go vet ./...`, `go test ./...`;
  frontend `npm run check`. Check that `notYetServed` no longer lists the 10 operations. Run the e2e
  files `financials.spec.ts`, `core-flow.spec.ts`, `module-rest.spec.ts` and `smoke.spec.ts`. Update
  existing expectations only where 015 changes them: the activity page sections, the line-item form
  sections, All work default columns and the sidebar order.
- [ ] T050 Demonstrate with the seed: the US1–US7 independent tests. Tick the tasks.

---

## Dependencies & Execution Order

- **T001–T002** first. **T003–T011** (B) and **T012–T014** (W) run in parallel after T001.
- **US1** needs T009–T011 (B) and T012–T014 (W). **US2** reuses US1's card. **US3** needs T016.
- **US4** needs only the Foundation (the demo data already feeds US1–US3). **US5** needs T009
  (`FactsByLineItem`). **US6** needs T009 and T010. **US7** needs US3's panel (T029).
- **Polish** last. T049 closes the feature.
- Outside dependencies:
  - **Roles and access** (BUS.ADM.020) later replaces `AllowAll`; no task here depends on it;
  - **dated exchange rates** (BUS.BUD.050) later change `usd` to the posting-date rate (Proposed 015-G);
  - **011** may be merged or not (log entries are written either way).

## Parallel opportunities

- B: T003, T004 and T005 together. Then T006 and T007 together. T031 (CSV feed) can start right after T004.
- W: T012 and T013 together. Then T014. T023, T024 and T027 together. T038 alongside B's T036.
- Polish T043–T048 in parallel.
