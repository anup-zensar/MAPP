---

description: "Task list for feature 007 — Line Items"
---

# Tasks: Line Items

**Input**: Design documents from `specs/007-line-items/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); features 001–006 merged (007 builds on their packages,
pages and seed — see plan *Dependencies*).

**Tests**: required — constitution V. Within each group write the tests first and see them fail, then
implement. Test names carry the FR they prove (plan *Business rules and their tests*). Every user-visible
text comes from the plan's *User-visible messages* and *Validation messages* tables.

**Organization**: Setup (the contract both sides start from) → Backend → Frontend → End-to-end → Polish.
Backend and Frontend phases are **independent of each other**: a backend agent and a frontend agent work in
parallel from T001–T002 (the frontend tests use `mockApi` with the contract's shapes). Story tags: US1 see
an activity's costs · US2 add a line item · US3 spread over time · US4 view, edit, duplicate, delete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 … US4
- Paths are relative to the repository root.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [X] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [X] TC01 Use `restartApproval` and `400 validation_failed` on `restartApproval` everywhere (already renamed in these tasks) (D-189).
- [X] TC02 Implement `LineItems.CopyAll(..., yearShift)` and the `lineItem` `plans.Level`; copy each line item once (D-195).
- [X] TC03 Reuse `money.ConvertSpread` and `(Amount).SplitEvenly` (002) instead of own month and even-split code.
- [ ] TC99 Check the success criteria SC-001, SC-002, SC-003, SC-004, SC-005, SC-006, SC-007, SC-008, which no task above names, in the final review (timed end-to-end run or moderated session, as each criterion says) and record the result in `checklists/`.

## Phase 1: Setup (the shared contract)

- [x] T001 Merge `specs/007-line-items/contracts/openapi.yaml` into `backend/api/openapi.yaml`: the tag "Line items", the 7 operations, parameters `LineItemId` and `RestartApproval`, and every schema of the fragment; check that `Money`, `AttributeValues` (002), `PersonRef` (003), `SpreadMethod`, `MissingField` (004), `ActivityStatus` and parameter `ActivityId` (006) exist under these names (align the `$ref`s if an earlier feature named them differently) and that `Error.code` has `validation_failed` on `restartApproval` (add it if 006 did not) *(Done at consolidation, D-192.)*
- [x] T002 Regenerate `frontend/src/api/schema.d.ts` with `npm run api:types` in `frontend/`; `npm run api:check` passes (depends on T001) *(Done at consolidation, D-192.)*

**Checkpoint**: backend (Phase 2) and frontend (Phase 3) can start in parallel.

---

## Phase 2: Backend

### Foundation (blocks the backend stories)

- [X] T003 [P] Write the failing tests `TestAfterRestore_RunsOnceForDeletedRecord_FR033`, `TestAfterRestore_Notes` and `TestAfterRestore_ErrorChangesNothing` in `backend/internal/history/service_test.go` (a parent kind and a child kind with hooks: deleting the child runs only the child's hook; deleting the parent runs only the parent's; notes appear in the `restored` entry's details and in `RestoreResult.details`; a hook error leaves the bin entry and storage unchanged)
- [X] T004 Add the optional `AfterRestore func(ctx context.Context, rc RestoreContext, top Snapshot) (notes []string, err error)` field to `history.Kind` in `backend/internal/history/kinds.go` and call it in `putBack` in `backend/internal/history/service.go` after `putSnapshots`, for the entry's own kind only, appending its notes (plan *History*); T003 passes (depends on T003)
- [x] T005 [P] Create `backend/internal/lineitems/model.go`: stored `LineItem`, `Spread`, `MonthAmount`, `SpreadMethod` (+ constants) exactly as data-model §1; API types `View`, `Details`, `Section`, `ActivityView` (`LineItemActivity`), `Balance`, `Grid`, `GridColumn`, `GridRow`, `FormOptions`, `Defaults`, `CurrencyView`, `BudgetItemRef`, `MissingField`, `SpreadView`, `SpreadMonthView`, `Input`, `Update` (`Input` + `Version *int`), `DuplicateInput`, `ActivitySummary`, `LineItemProblems`, `Usage`, `RateImpact` with the JSON names of `contracts/openapi.yaml`; lists never nil
- [X] T006 [P] Create `backend/internal/lineitems/deps.go` with `Deps`, `Activities`, `ActivityFacts`, `Currencies`, `Currency`, `FinanceBudgets`, `BudgetItem`, `Attributes`, `People` exactly as plan *Package `lineitems` and its dependencies*
- [x] T007 Create `backend/internal/lineitems/repository.go` (`Repository`: `Get`, `ListByActivity`, `List`, `Put`, `Delete`) and `backend/internal/lineitems/store_json.go` (`NewJSONRepository()`, table `line_items`, creation order = ID order) (depends on T005)
- [X] T008 Create `backend/internal/lineitems/fakes_test.go`: in-memory fakes of `Activities` (facts per id, records `RestartApproval` calls), `Currencies` (USD 1, EUR 0.92, GBP 0.79, CNY 7.10, INR 83.50 inactive), `FinanceBudgets` (2026 Lubricants Budget: Europe – Digital, Europe – Events, Europe – Brand; a budget with one item; one inactive item; 2027 empty), `Attributes` (configurable problems, missing list and changes), `People`; and `newTestService(t)` building the service on `store.Open(t.TempDir())`, a fixed clock (2026-03-12 09:00 UTC), a real `history.Service` with a fake `activity` kind registered before `lineItem` (depends on T006, T007)

### User Story 3 — spread over time (pure rules first; US1 and US2 use them)

- [X] T009 [P] [US3] Write the failing tests in `backend/internal/lineitems/spread_test.go`: `TestActivityMonths_CalendarMonths_FR017`, `TestActivityMonths_OngoingAndNoDates_FR021`, `TestSpread_EvenlyRemainderLast_FR017_SC002`, `TestSpread_FirstAndLastMonth_FR017`, `TestSpread_OngoingLikeFirst_FR021`, `TestSpread_NoDatesNotSpread_FR017`, `TestSpread_CustomOutsideMarked_FR018`, `TestSpread_FollowsDatesAndCost_FR022`, `TestUSDMonths_AddUp_FR020_SC002` (amounts × rates 1, 0.92, 0.79, 7.10, 83.50 × 1–36 months), `TestUSDMonths_CentInLastNonZeroMonth_FR020`, using the plan's shared example table
- [X] T010 [US3] Implement `backend/internal/lineitems/spread.go`: `ActivityMonths`, `SpreadMonths`, `OutsideActivityDates`, `USDMonths`, `PlannedCostUSD` (data-model §3.1–§3.4; exported for features 010 and 013); T009 passes (depends on T005, T009)

### User Story 1 — see an activity's costs

- [X] T011 [P] [US1] Write the failing tests in `backend/internal/lineitems/balance_test.go` (`TestPlannedCostUSD_HalfUp_FR011_SC003`, `TestBalance_Formula_FR002_SC004`, `TestBalance_States_FR002`, `TestBalance_NoPlannedCost_007P5`, `TestBalance_RoundingHint_EdgeCase`), `grid_test.go` (`TestGrid_Columns_FR003`, `TestGrid_NoDatesColumn_D032`, `TestGrid_Totals_FR003_FR020`) and `completeness_test.go` (`TestMissing_NeededToSubmit_FR028`, `TestMissing_Attributes_FR014`, `TestMissing_InactiveItem_FR016`, `TestMissing_CustomNotAddingUp_FR019`, `TestSwbsStatus_FR024`)
- [X] T012 [US1] Implement `backend/internal/lineitems/balance.go` (`Balance`, data-model §3.7), `grid.go` (`Grid`, §3.8) and `completeness.go` (`Missing`, `SwbsStatus`, §3.5–§3.6); T011 passes (depends on T010, T011)
- [X] T013 [P] [US1] Write the failing service tests `TestSection_RowsInCreationOrder_FR001`, `TestSection_ActivityDeleted_EdgeCase` and `TestSection_RateChange_FR013` (change the fake EUR rate: USD amounts and balance change, nothing written, no `RestartApproval`) in `backend/internal/lineitems/service_test.go`
- [X] T014 [US1] Implement `NewService` and `Section` in `backend/internal/lineitems/service.go` (the item view builder shared by every operation: currency view with rate, USD amount, spread view, owner, budget item, missing, SWBS status); T013 passes (depends on T008, T012, T013)

### User Story 2 — add a line item

- [X] T015 [P] [US2] Write the failing tests in `backend/internal/lineitems/service_test.go`: `TestFormOptions_Defaults_FR008`, `TestFormOptions_NameTaken_FR008`, `TestFormOptions_NothingLeft_EdgeCase`, `TestFormOptions_BudgetChoices_FR015` (years, order, pre-selection only with one active item — R12), `TestFormOptions_Currencies_FR011`, `TestCreate_OnlyRequired_FR009`, `TestCreate_AllProblemsAtOnce_FR009_SC005`, `TestCreate_NameUnique_FR009`, `TestCreate_RoundsToCent_EdgeCase`, `TestCreate_IDAndLog_FR010`, `TestCreate_CurrencyActive_FR011`, `TestCreate_AttributeProblems_FR014`, `TestCreate_ContractedAmount_FR014`, `TestCreate_FinanceBudgetChecks_FR015_FR016`, `TestCreate_CustomMonthChecks_FR018`, `TestCreate_CustomNotAddingUpSaves_FR019`, `TestCreate_RestartNeedsConfirmation_FR031`
- [X] T016 [US2] Implement `backend/internal/lineitems/validate.go` (every check and message of plan *Validation messages*, collected with `errs.Problems`, amounts with more than two decimals refused, D-190; the form rounds them first) and `FormOptions` and `Create` in `backend/internal/lineitems/service.go` (default name with " 2", " 3"…; confirmation rule 007-P7; `RestartApproval` reason; log `created`); T015 passes (depends on T014, T015)

### User Story 4 — view, edit, duplicate and delete

- [X] T017 [P] [US4] Write the failing tests in `backend/internal/lineitems/service_test.go`: `TestGet_GoneMessages_007P15`, `TestUpdate_Version_FR027`, `TestUpdate_NoChangeNoLog_FR027`, `TestUpdate_LogsOldAndNew_FR027`, `TestUpdate_KeepsInactiveValues_FR011_FR016`, `TestUpdate_CostOrCurrencyRestart_FR031`, `TestUpdate_OtherEditsKeepStatus_FR032_SC007`, `TestUpdate_LeavingCustomDropsMonths_FR018`, `TestUpdate_CostChangeKeepsCustom_FR022`, `TestDuplicate_CopiesEverything_FR029`, `TestDuplicate_RestartNeedsConfirmation_FR031`, `TestDelete_MovesToBin_FR030`, `TestDelete_RestartNeedsConfirmation_FR031`
- [X] T018 [US4] Implement `Get`, `Update`, `Duplicate` and `Delete` in `backend/internal/lineitems/service.go` (007-P15 messages through `history.ListBin`; `CheckVersion(…, "line item")`; Diff without `attributes` plus `Attributes.Changes`; " (copy)" names shortened to 120 characters; `history.MoveToBin`); T017 passes (depends on T016, T017)
- [X] T019 [P] [US4] Write the failing tests in `backend/internal/lineitems/kinds_test.go`: `TestUndo_RestoresExactly_FR030`, `TestUndo_EndedApprovalStaysEnded_FR033`, `TestRestore_IntoApprovedRestarts_FR033`, `TestUndo_NameTaken_D037`, `TestRestore_NameTakenSuffix_FR030`, `TestDeleteActivity_TakesLineItems`, `TestDescribe_LogTexts_FR027`
- [X] T020 [US4] Implement `backend/internal/lineitems/kinds.go`: `Kinds()` with the kind `lineItem` (parent `activity`), `Snapshot`, `ChildIDs`, `Remove`, `Put`, `Describe` (data-model §7.3), `PrepareRestore` (name rule), `AfterRestore` (FR-033 restart and note); T019 passes (depends on T004, T018, T019)

### Provided to other features

- [ ] T021 [P] Write the failing tests in `backend/internal/lineitems/provided_test.go`: `TestSummaries_ReadyToSubmit_FR023_FR028_SC005`, `TestCopyLineItems_SameYear` (new IDs, same values, no log entries), `TestCopyLineItems_NextYear_007P9`, `TestUsage_IncludesRecycleBin`, `TestRateChangeImpact_FR013`, `TestOwnerUsage`
- [x] T022 Implement `backend/internal/lineitems/provided.go`: `Summaries`, `CopyLineItems`, `CurrencyUsage`, `FinanceBudgetItemUsage`, `AttributeUsage`, `OwnerUsage`, `RateChangeImpact` (plan *Provided to other features*); T021 passes (depends on T018, T021)

### HTTP, wiring and demo data

- [X] T023 Implement `backend/internal/lineitems/handler.go`: `NewHandler`, `Routes` registering the 7 operations of the contract behind `signedIn`; each handler `web.ReadJSON` → `actor.From` → service → `web.WriteJSON` (201 for create and duplicate; delete answers `history.DeleteResult{RecycleBinEntry: entry.View()}`; the `restartApproval` query value `true`/`false`, anything else `400 bad_request`) (depends on T020, T022)
- [X] T024 Create `backend/internal/app/lineitems.go` with the adapters implementing `lineitems.Activities` (facts from 006's activity and 004's plan; `RestartApproval` → 006), `Currencies`, `FinanceBudgets`, `Attributes` (002), `People` (003); in `backend/internal/app/app.go` build the repository and service, register `Kinds()` after 006's `activity` kind, call `NewHandler(...).Routes(router, signedIn)`, and hand the provided functions to the other features through their seams as plan *Alignment with the designs of 002, 004 and 006* (002 `settings.ReferenceSource` for usage and rate impact, 003 owner count, 004 `plans.RegisterLevel` with `AddFacts` and `Copy`, 006 `UseLineItems` with `Facts`, `CopyAll` and a no-op `DatesChanged`, 005 through the same `CopyLineItems`); `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005` pass (depends on T023)
- [X] T025 [P] Write `backend/internal/lineitems/handler_test.go` (`package lineitems_test`, `apitest.New(t)` with the seed): `TestListActivityLineItems_Contract`, `TestGetLineItemFormOptions_Contract`, `TestCreateLineItem_Contract`, `TestGetLineItem_Contract`, `TestUpdateLineItem_Contract`, `TestDuplicateLineItem_Contract`, `TestDeleteLineItem_Contract` as plan *Contract tests* (needs T024 and T027 to pass)
- [x] T026 [P] Write the failing test `TestLoad_LineItems` in `backend/seed/seed_test.go` (eight line items L-1001…L-1008; Launch film balance 20,000.00 USD; the Mobility China activity exceeded by 500.00 USD; every reference found)
- [x] T027 Add `addLineItems(ctx, generator, createdAt)` to `backend/seed/seed.go`, called after the records of 002–006, creating data-model §8 (look-ups by name, *computed* amounts, an error naming any missing reference); T026 and T025 pass (depends on T024, T026)

**Checkpoint**: `go test ./...` green; the API serves the contract with the demo data.

---

## Phase 3: Frontend (starts after T002, in parallel with Phase 2)

### Foundation

- [X] T028 [P] Write the failing tests `frontend/src/features/line-items/lineItemMath.test.ts`: the plan's shared example table (conversion both ways, even split, USD months with the cent in the last non-zero month), `parseAmount` (rounding half-up, invalid input), `formatAmount`, `formatRate`, `formatMonth`, `activityMonths`, `spreadMonths` (ongoing, no dates), `quarterColumns`, `customGridMonths` (activity months; ongoing → December of the plan's year; no dates → the plan's twelve months; stored outside months kept), `spreadEffect`
- [X] T029 Implement `frontend/src/features/line-items/lineItemMath.ts` (BigInt cents, 007-P3); T028 passes (depends on T028)
- [X] T030 [P] Create `frontend/src/features/line-items/lineItemText.ts` with every text of plan *User-visible messages* (constants and small functions for plurals, balance messages, restart reasons and `spreadEffectText`) and `lineItemText.test.ts` (singular/plural, negative amounts, display-currency labels)
- [X] T031 Create `frontend/src/features/line-items/lineItemsApi.ts` (hooks and query keys of plan *Data*; invalidation of `['line-items']`, `['activities']`, `['plans']`, `['recycle-bin']`, `['activity-log']` after every change) and `frontend/src/features/line-items/testData.ts` (contract-shaped fixtures: Launch film section, form options, an approved activity, an inactive currency and item, a custom spread with an outside month) (depends on T002)

### User Story 1 — see an activity's costs

- [X] T032 [P] [US1] Write the failing tests `frontend/src/features/line-items/LineItemsSection.test.tsx` (US1 scenarios 1–5: rows with ID, currency, local and USD amounts, spread, chips; system row "20,000.00 USD", meter "100,000.00 of 120,000.00 USD assigned", red exceeded message, all-assigned and no-planned-cost texts, rounding hint; empty state with **Add line item** and the system row; SWBS chips by status; display currency EUR relabels the USD column and converts; highlight from `location.state`; ⋯ Actions items; loading skeleton; error with **Try again**) and `SpreadGrid.test.tsx` (Mar–May columns and totals, Quarterly → Q1/Q2 2026, outside label and tip, No dates yet column and note, sticky name column, choice remembered in `localStorage` with try/catch)
- [X] T033 [P] [US1] Create `frontend/src/features/line-items/SwbsChip.tsx` (status → `StatusChip` tone and text) and `UnassignedBalanceRow.tsx` (+ `.module.css`)
- [X] T034 [P] [US1] Create `frontend/src/features/line-items/SpreadGrid.tsx` (+ `SpreadGrid.module.css`) (depends on T029)
- [X] T035 [US1] Create `frontend/src/features/line-items/LineItemsSection.tsx` (+ `LineItemsSection.module.css`) composing rows, system row and grid; T032 passes (depends on T031, T033, T034)
- [X] T036 [US1] Render `<LineItemsSection activityId>` in feature 006's Line items slot of `frontend/src/features/activities/ActivityPage.tsx` (anchor `line-items`, pass `location.state.highlightLineItemId`), replacing 006's placeholder; 006's page tests still pass (depends on T035)

### User Story 2 — add a line item

- [X] T037 [P] [US2] Write the failing tests `frontend/src/features/line-items/lineItemFormState.test.ts`: `newFormState(options)` (defaults and sources), `editFormState(lineItem)`, `changeCurrency` (FR-012: 20,000.00 USD → 18,400.00 EUR with the note; a typed amount is not converted), leaving Custom note, year change marks the item to clear, `toCreateBody`/`toUpdateBody` (empty amounts → `null`, the cleared item → `null`, months only for Custom), `needsRestartConfirmation` and `restartReasons` (FR-031, FR-032)
- [X] T038 [US2] Implement `frontend/src/features/line-items/lineItemFormState.ts`; T037 passes (depends on T029, T030, T037)
- [X] T039 [P] [US2] Write the failing tests `frontend/src/features/line-items/FinanceBudgetFields.test.tsx` (year options; items grouped by budget with the plan's default first and searchable; pre-selection note; "listed first" hint; "No finance budget items for 2027 yet…"; year change → "will be cleared" note; stored inactive item "(inactive)" with its problem)
- [X] T040 [US2] Create `frontend/src/features/line-items/FinanceBudgetFields.tsx`; T039 passes (depends on T038, T039)
- [X] T041 [US2] Create `frontend/src/features/line-items/CostFields.tsx` (currency choice with the inactive option, `MoneyInput`, the "= … USD at … per USD" line with "Calculated" and ⓘ, the FR-012 note, rounding hint), `LineItemSummaryBar.tsx` and `FromFinanceSection.tsx` (+ `.module.css`) (depends on T038)
- [X] T042 [US2] Create `frontend/src/features/line-items/LineItemForm.tsx` (+ `LineItemForm.module.css`): breadcrumb, header, summary bar, the seven sections in order with markings and source notes (002's `ShellAttributeFields` for level `lineItem`, Contracted amount, `FinanceBudgetFields`, GSAP request, `FromFinanceSection`), sticky footer, error summary and field errors, `useUnsavedChanges`, `?field=` focus (depends on T040, T041, T044)
- [X] T043 [P] [US2] Write the failing tests `frontend/src/features/line-items/LineItemCreatePage.test.tsx` (US2 scenarios 1–7: defaults with their sources and the summary bar; EUR 10,000 → "= 10,869.57 USD at 0.92 EUR per USD" and "After saving 9,130.43 USD"; GL code hint, single-option fill and clear note through the mocked attribute definitions; API 400 with three problems → summary and field messages, nothing navigates; save → toast "Line item 'Agency fee' created", the `['line-items']`, `['activities']` and `['plans']` queries invalidated (FR-023), navigation to the activity with the highlight; **Save and add another** → new balance in the planned cost; "Nothing left to assign…"; restart dialog on an approved activity and `restartApproval: true` after confirming; 400 `validation_failed` on `restartApproval` reopens the dialog; activity deleted 404; FR-025 collapsed section; cancel asks before discarding)
- [X] T044 [P] [US3] Write the failing tests `frontend/src/features/line-items/SpreadEditor.test.tsx` (US3 scenarios 1, 2, 4, 6: Evenly/first/last previews "Mar 2026 · 3,333.33 EUR"; Custom grid from the current spread with **Add month before**/**after**; Feb marked outside; "12,000.00 of 10,000.00 EUR spread — 2,000.00 EUR too much", "— adds up", "— … left"; **Use month total as planned cost**; ongoing notes; no-dates sentence; leaving Custom note) and implement `frontend/src/features/line-items/SpreadEditor.tsx` (+ `.module.css`) until they pass (depends on T038)
- [X] T045 [US2] Create `frontend/src/features/line-items/LineItemCreatePage.tsx` and `RestartApprovalDialog.tsx`; add the route `/activities/:activityId/line-items/new` (`handle.title` "New line item") to `frontend/src/app/router.tsx`; T043 passes (depends on T042, T043)

### User Story 4 — view, edit, duplicate and delete

- [X] T046 [P] [US4] Write the failing tests `frontend/src/features/line-items/LineItemPage.test.tsx` (US4 scenario 1: breadcrumb, tag, chips, ID, owner, "50,000.00 USD", spread, created/changed line, "—" for empty values, "Filled by GSAP later", From finance collapsed; **More** has Duplicate, View history (link to `/activity-log?recordType=lineItem&recordId=L-1001`), Delete last; scenario 4: Duplicate → toast and navigation to the copy, the restart dialog first on an approved activity; scenario 5: delete dialog texts, **Cancel** focused, the re-approval line only when `restartsApproval`, then navigation to the activity and the Undo message; not found and deleted-activity messages)
- [X] T047 [US4] Create `frontend/src/features/line-items/LineItemPage.tsx` (+ `.module.css`) and `DeleteLineItemDialog.tsx`; add the route `/line-items/:lineItemId` (`handle.title` "Line item"); T046 passes (depends on T045, T046)
- [X] T048 [P] [US4] Write the failing tests `frontend/src/features/line-items/LineItemEditPage.test.tsx` (US4 scenario 2: GL code / finance item / spread / SWBS change on an approved activity saves without a dialog; scenario 3: planned cost 50,000.00 → 55,000.00 or a currency change opens "This activity will need approval again" listing the change, **Keep editing** keeps the form, **Save and restart approval** sends the confirmation; US3 scenario 3 on a saved custom spread; summary bar "Unassigned now"/"After saving" on edit; version conflict message with values kept and the refreshed version used next; 404 → the deleted-activity message; `?field=financeBudgetItemId` focuses the field; toast "Line item 'Agency fee' saved")
- [X] T049 [US4] Create `frontend/src/features/line-items/LineItemEditPage.tsx`; add the route `/line-items/:lineItemId/edit` (`handle.title` "Edit line item"); extend the shared `recordHref` with `lineItem → /line-items/<id>`; T048 passes (depends on T047, T048)

### User Story 3 — spreads follow the activity's dates

- [X] T050 [P] [US3] Write the failing tests `frontend/src/features/line-items/SpreadEffectNote.test.tsx` (FR-022: nothing when the dates are unchanged or there are no line items; "When you save, 2 line items are re-spread over the new dates and 1 custom spread is kept as typed."; outside sentence; no-dates sentence; singular forms)
- [x] T051 [US3] Create `frontend/src/features/line-items/SpreadEffectNote.tsx`; render it under the dates of feature 006's activity edit form and add `spreadEffectText(...)` to 006's re-approval dialog list (007-P11); T050 and 006's tests pass (depends on T029, T030, T031, T050)
- [x] T052 Add the three routes and their titles to `frontend/src/app/router.test.tsx` (depends on T045, T047, T049)

**Checkpoint**: `npm run check` green.

---

## Phase 4: End-to-end (needs Phases 2 and 3)

- [ ] T053 Add `openActivity(page, planName, activityName)` to `frontend/e2e/fixtures.ts` (Plans → plan → "Search name or ID in this plan" → the activity's page)
- [ ] T054 [US1] Write the test `US1: the Line items section shows costs, the unassigned planned cost and the grid` in `frontend/e2e/line-items.spec.ts` (plan *End-to-end tests*) (depends on T053)
- [ ] T055 [US2] Add `US2: add a line item with its defaults` to `frontend/e2e/line-items.spec.ts` (cleans up its line items) (depends on T054)
- [ ] T056 [US3] Add `US3: spread a line item over time` to `frontend/e2e/line-items.spec.ts` (cleans up) (depends on T055)
- [ ] T057 [US4] Add `US4: edit, duplicate and delete on an approved activity` to `frontend/e2e/line-items.spec.ts` (depends on T056)
- [ ] T058 Add `Accessibility: line-item screens pass axe` to `frontend/e2e/line-items.spec.ts` (section, create form with errors, custom grid, details, restart and delete dialogs) (depends on T057)

---

## Phase 5: Polish

- [ ] T059 [P] Add the proposed decisions 007-P1 … 007-P15 to `docs/sdd/decision-log.md` with their consolidated numbers, and mark 007 in `docs/sdd/feature-map.md`
- [x] T060 [P] Add `lineitems/` to the package list of `backend/README.md` and `features/line-items/` to the folder list of `frontend/README.md`
- [ ] T061 Run `gofmt -l .`, `go vet ./...`, `go test ./...` in `backend/` and `npm run check`, `npm run test:e2e` in `frontend/`; walk through plan *Quickstart checks* with the demo data; compare the screens with `docs/product/design-system.md` (§4 chips and money, §6 CRUD pattern, §6.6 read-only, §7 forms) at 1280 × 720 and 390 px

---

## Dependencies & Execution Order

- **Setup (T001–T002)** first; then **Backend (T003–T027)** and **Frontend (T028–T052)** run in parallel —
  neither waits for the other.
- **Backend order**: T003→T004; T005, T006 (parallel) → T007 → T008; T009→T010 → T011→T012 → T013→T014 →
  T015→T016 → T017→T018 → T019→T020 (needs T004) → T021→T022 → T023 → T024 → T026→T027 → T025.
- **Frontend order**: T028→T029; T030, T031 (parallel); US1: T032–T034 → T035 → T036; US2/US3 form: T037→T038
  → T039→T040, T041, T044 → T042 → T043→T045; US4: T046→T047 → T048→T049; T050→T051; T052 last.
- **End-to-end (T053–T058)** after both phases; one file, so in order.
- **Polish (T059–T061)** last.

### Parallel opportunities

- Backend tests T003, T009, T011, T013, T015, T017, T019, T021, T026 are in different files or test only
  their own functions and can be written together once T005–T008 exist.
- Frontend T028, T030, T032, T037, T039, T043, T044, T046, T048, T050 are separate test files.
- T033 and T034 are separate components.

---

## FR coverage (every FR has at least one test task)

| FR | Test tasks |
|---|---|
| FR-001 | T013, T025, T032, T054 |
| FR-002 | T011, T032, T054 |
| FR-003 | T011, T032, T054 |
| FR-004 | T032 |
| FR-005 | T032 |
| FR-006, FR-007 | T043, T055 |
| FR-008 | T015, T037, T043 |
| FR-009 | T015, T025, T043, T055 |
| FR-010 | T015, T043, T055 |
| FR-011 | T011, T015, T017, T028, T043 |
| FR-012 | T028, T037, T043 |
| FR-013 | T013, T021 |
| FR-014 | T011, T015, T043, T055 |
| FR-015 | T015, T037, T039 |
| FR-016 | T011, T015, T017, T039 |
| FR-017 | T009, T028, T044, T056 |
| FR-018 | T009, T015, T017, T028, T044, T056 |
| FR-019 | T011, T015, T056 |
| FR-020 | T009, T011, T028 |
| FR-021 | T009, T028, T044 |
| FR-022 | T009, T017, T028, T050 |
| FR-023 | T021, T043 |
| FR-024 | T011, T032, T054 |
| FR-025 | T043, T046 |
| FR-026 | T017, T025, T046, T057 |
| FR-027 | T017, T019, T025, T048 |
| FR-028 | T011, T021, T032 |
| FR-029 | T017, T025, T046, T057 |
| FR-030 | T017, T019, T025, T046, T057 |
| FR-031 | T015, T017, T025, T037, T043, T048, T057 |
| FR-032 | T017, T037, T048, T057 |
| FR-033 | T003, T019 |

## Implementation Strategy

1. Setup → the contract is fixed.
2. Backend and frontend in parallel, each story in order US3 rules → US1 → US2 → US4 (backend) and US1 →
   US2/US3 → US4 (frontend); each story's tests first.
3. End-to-end once both sides are merged on the branch, then Polish.
4. Commit after each task group, e.g. `feat(007): line items section and unassigned balance (FR-001–FR-005)`.
