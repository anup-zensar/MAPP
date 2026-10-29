---

description: "Task list for feature 016 — Planning and Currency Extensions"
---

# Tasks: Planning and Currency Extensions

> **Build state after the final merge** (`plan-module` at `a1ae8fe`, 21 October 2026): the backend is complete.
> Shared web parts were built under other names: `RateTip`, `MoneyPair` and `rates.ts` in `features/line-items/`,
> `AmountPair` in `features/plans/`, `YearRangeMenu` in `features/all-work/`, `useFieldHelp` and `FieldHelpTip`
> in `features/settings/field-help/`. **Open:** T028 (the Import GSAP rates page is built, but
> `/settings/currencies/import` is not in `frontend/src/app/router.tsx` — `SETTINGS_016_ROUTES` is unused — so
> the Currencies page's **Import GSAP rates** button opens Not found); T074–T077 (attribute filters, Breakdown,
> Planned cost by, saved views and export of them); T084–T088 (record screens do not read Field help yet);
> T090–T093 (product documents); T094, T095.

**Input**: Design documents from `specs/016-planning-and-currency-extensions/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); features 001–010 merged on `plan-module` (011 optional).

**Tests**: required — constitution V. In each group write the tests first, see them fail, then implement. Test
names carry the FR (`TestX_Y_FR020`). Texts come from plan.md *User-visible messages*.

**Organization**: Setup → Foundation (blocks everything) → one phase per user story (lanes R, C, P, A, H of
plan.md *Build order and lanes*; backend and frontend tasks of a story run in parallel against the contract) →
End-to-end, docs and checks. Story tags: US1 rates · US2 plan currency · US3 phasing · US4 split · US5 plan type
and multi-year · US6 categories · US7 field help.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- Paths are relative to the repository root. Backend checks: `gofmt -l .`, `go vet ./...`, `go test ./...`;
  web checks: `npm.cmd run check` in `frontend/`.

---

## Phase 1: Setup (the shared contract)

- [x] T001 (done by the consolidation, D-298) Merge `specs/016-planning-and-currency-extensions/contracts/openapi.yaml` into `backend/api/openapi.yaml`: the 21 new operations (17 paths), new parameters and schemas, and every `x-amendments` entry (added properties into the named schemas' `properties` and `required`, added parameters into the named operations, enum values added); keep 001's required lists untouched (D-191)
- [x] T002 (done by the consolidation) Regenerate `frontend/src/api/schema.d.ts` (`npm.cmd run api:types`); `npm.cmd run api:check` passes (depends on T001)
- [x] T003 (done by the consolidation) Add the 21 new operations to `notYetServed` in `backend/internal/app/app_test.go`; every lane removes its own when served (D-192) (depends on T001)

**Checkpoint**: backend and frontend can start.

---

## Phase 2: Foundation (blocks all stories)

### Backend

- [x] T004 [P] Write `backend/internal/platform/money/convert_test.go`: `TestConvert_Table_FR015` (same rate → unchanged; 23,000.00 EUR 0.92 → GBP 0.79 = 19,750.00; 5,000.00 USD → GBP = 3,950.00; 600,000.00 GBP → USD = 759,493.67; half up and negative half away from zero; overflow-safe with 1,000,000 rates), `TestConvert_MatchesToUSDAndFromUSD`, `TestConvertParts_AddsUp_FR015`, `TestRatesConvert_UnknownCurrency`
- [x] T005 Implement `Convert`, `ConvertParts`, `(Rates).Convert` in `backend/internal/platform/money/convert.go`; rewrite `ToUSD`, `FromUSD`, `ConvertSpread` on top; all existing money tests stay green (depends on T004)
- [x] T006 [P] Write `backend/internal/rollup/phasing_test.go`: `TestPhasing_None`, `TestPhasing_Evenly_FR020` (600,000.00 → 50,000.00 × 12; 100,000.00 → 8,333.33 × 11 + 8,333.37), `TestPhasing_Custom_MissingMonthsAreZero`, `TestCheckPhasing_Rules_FR020` (no budget, month outside year, negative, three decimals, duplicate month), `TestPhasingProblem_Mismatch_FR022` (text of data-model §3.3), `TestShiftPhasing_NextYear_FR025`
- [x] T007 Implement `backend/internal/rollup/phasing.go` (types and functions of plan *rollup — phasing.go*) (depends on T006)
- [x] T008 Add `ExchangeRate`, `RateSource`, `RateImport`, `FieldHelpEntry` to `backend/internal/settings/model.go` (data-model §2.1, §2.3, §2.5), mark `Currency.Rate`, `RateChangedAt`, `RateChangedBy` as legacy `omitempty`; add the repository methods of plan *settings* and the JSON tables `exchange_rates`, `rate_imports`, `field_help` in `repository.go` and `store_json.go`; register ID prefixes `XR`, `RI` in `backend/internal/platform/ids`
- [x] T009 [P] Write `backend/internal/settings/rates_test.go`: `TestMigrateRates_CreatesOneRatePerCurrency`, `TestMigrateRates_SecondRunDoesNothing`, `TestRates_LatestOnOrBeforeToday_FR002`, `TestRates_FutureRateNotUsed_FR002`, `TestRatesOn`, `TestCurrentRates_DateAndSource_FR009` (fixed clock 2026-10-20)
- [x] T010 Implement `MigrateRates`, the new `Rates(ctx)` over `exchange_rates`, `RatesOn`, `CurrentRates`, `CurrentRate` in `backend/internal/settings/rates.go` and `readers.go`; call `MigrateRates` from `app.New` in `backend/internal/app/app.go` (depends on T008, T009)
- [x] T011 [P] Add `Currency`, `PlanType` (constants `operational`, `strategic`, `crossMarket`) and `BudgetPhasing rollup.Phasing` to `Plan` in `backend/internal/plans/model.go`, with read normalisation (data-model §4.5) in `store_json.go`; test `TestPlanStore_NormalisesOlderRows` (depends on T007)
- [x] T012 [P] Add `BudgetPhasing rollup.Phasing` to `Campaign` and `Initiative` in `backend/internal/campaigns/model.go` with read normalisation; add `SplitFromID` to `Activity` in `backend/internal/activities/model.go`; tests for both stores (depends on T007)
- [x] T013 Seed (data-model §8) in `backend/seed/settings.go` (RI-1001 and 30 days of rates per currency ending with today's seeded rate), `plans.go` (types; P-1004), `campaigns.go`, `activities.go`, `lineitems.go` (P-1004's records, appended after the existing ones); extend `backend/seed/*_test.go`: existing IDs and 2026 totals unchanged, current rates EUR 0.92 / GBP 0.79 / CNY 7.10 / INR 83.50, P-1004 facts (depends on T010–T012)

### Frontend

- [x] T014 [P] Create `frontend/src/api/ratesQueries.ts` (`useCurrentRates`, `currentRatesKey`, invalidation helper) and `frontend/src/components/RateTip/` (`RateTip.tsx`, test, gallery entry) per plan *Shared parts*
- [x] T015 [P] Create `frontend/src/components/AmountPair/` (component, test: secondary hidden when same currency, RateTip present, negative red), export in `components/index.ts`, gallery entry
- [x] T016 [P] Create `frontend/src/components/YearRangeField/` (component, test: from ≤ to kept, max span message "Choose at most 5 years.", "All years"), export, gallery entry
- [x] T017 [P] Create `frontend/src/api/fieldHelp.ts` (`useFieldHelp()` → `help(key)`; empty text → undefined) with tests using `mockApi`

**Checkpoint**: Foundation ready; lanes start in parallel.

---

## Phase 3: User Story 1 — Dated exchange rates and GSAP rate import (P1, lane R) 🎯

**Goal**: rate history, current rate, simulated GSAP import. **Independent test**: spec US1.

### Backend

- [x] T018 [P] [US1] Write `backend/internal/settings/rates_crud_test.go`: `TestAddRate_Rules_FR001` (date taken, rate rules, USD refused), `TestUpdateRate_SetsManual_FR003`, `TestDeleteRate_LastCurrentRefused_FR003`, `TestDeleteRate_FutureAllowed`, `TestRestoreRate_DateTakenRefused`, `TestCreateCurrency_FirstRateToday_FR003`, `TestUpdateCurrencyRate_UpsertsToday_FR004`, `TestRateImpact_LineItemsAndPlans_FR005` (EUR 0.92 → 0.95 with a EUR plan of 1,200,000.00: 1,304,347.83 → 1,263,157.89 USD), `TestRateImpact_FutureRateNoWarning`, each change logged with the actor (FR-049)
- [x] T019 [US1] Implement rate CRUD, impact and currency create/update changes in `backend/internal/settings/rates.go` and `service_currencies.go`; history kind `exchangeRate` under `currency` with `PrepareRestore` in `kinds.go`; API `Currency` computed fields (`rate`, `inverseRate`, `rateChangedAt/By`, `rateDate`, `rateSource`, `rateImportFileName`) in `views.go` (depends on T018)
- [x] T020 [US1] Usage: plans' `ReferenceSource` fills `Currency` with the plan currency in `backend/internal/app/references.go`; currencies count "Plan currency of n plans" and lock delete in `backend/internal/settings/usage.go`; test `TestCurrencyUsage_CountsPlans_FR010`
- [x] T021 [P] [US1] Write `backend/internal/settings/ratefeed_test.go`: `TestGSAPRateFile_Header_FR006` (any order, case, BOM; missing column message), one test per outcome row of data-model §2.4, `TestGSAPRateFile_DateFormats`, `TestGSAPRateFile_TooLarge`
- [x] T022 [US1] Implement `RateFeed`, `FeedRow`, `GSAPRateFile` in `backend/internal/settings/ratefeed.go` (`encoding/csv`) (depends on T021)
- [x] T023 [P] [US1] Write `backend/internal/settings/rates_import_test.go`: `TestCheckRateImport_Review_FR006` (the 92-row example of spec US1-2), `TestApplyRateImport_SameCountsAsCheck_FR006`, `TestApplyRateImport_RecordAndLog_FR007` (one `rateImport` entry, verb "imported"), `TestApplyRateImport_InactiveCurrencyImported`, `TestApplyRateRows_SharedCore_FR008`, `TestImportSample_TodayRates`
- [x] T024 [US1] Implement `CheckRateImport`, `ApplyRateImport`, `ApplyRateRows`, `ImportSample` in `backend/internal/settings/rates_import.go`; history kind `rateImport` (log only) (depends on T022, T023)
- [x] T025 [US1] Handlers and routes for the 11 rate operations in `backend/internal/settings/handler.go` and `backend/internal/app/app.go` (`GSAPRateFile` wired); `handler_test.go` checks each against the contract; remove them from `notYetServed` (depends on T019, T024)

### Frontend

- [x] T026 [P] [US1] Currency details in `frontend/src/features/settings/currencies/`: current rate line with date and source, `RateHistorySection` (table, "Current", "From <date>", Add rate, row menu Edit/Delete with `useUndoableDelete`); tests
- [x] T027 [P] [US1] `RateDialog` (add/edit, date + rate, impact confirmation via the two impact operations, "Keep editing" / "Save rate"); invalidate `useCurrentRates`; tests
- [ ] T028 [P] [US1] `ImportRatesPage` at `/settings/currencies/import` (upload, review grouped by outcome with counts and row numbers, Import n rates, Cancel, sample link) and `RecentImports` on the Currencies list; route in `frontend/src/app/router.tsx`; tests with `mockApi`

**Checkpoint**: US1 works alone (spec US1 scenarios).

---

## Phase 4: User Story 2 — Plan currency (P1, lane C) 🎯

**Goal**: plan in a local currency with USD beside. **Independent test**: spec US2.

### Backend

- [x] T029 [P] [US2] Write `backend/internal/plans/currency_test.go`: `TestCreatePlan_CurrencyAndType_FR011_FR032` (defaults, inactive refused, unknown type), `TestUpdatePlan_CurrencyLock_FR012` (lock text with count; allowed with only a plan budget), `TestPlanFigures_PlanCurrencyAndDisplay_FR013` (P-1004: 600,000.00 GBP; display USD 759,493.67), `TestPlanList_PlanCurrencyAmounts_FR014`, `TestCopyPlan_KeepsCurrencyTypePhasing_FR018_FR025`, `TestPlanDefaults_CurrencyType`
- [x] T030 [US2] Implement in `backend/internal/plans/` (`validate.go`, `service.go`, `figures.go`, `list.go`, `copy.go`, `handler.go`): currency and type rules, `AmountsSet` facts from the level registry, `currencyLock`, `planFigures` beside `figures`, PlanSummary plan-currency amounts, copy, plan defaults (depends on T029)
- [x] T031 [P] [US2] Write `backend/internal/campaigns/currency_test.go`: `TestCampaignBudget_MustBePlanCurrency_FR011`, `TestContainerDisplay_FromPlanCurrency_FR013`, `TestDestinations_OtherCurrencyUnavailable_FR017` (move and copy)
- [x] T032 [US2] Implement in `backend/internal/campaigns/` (`validate.go`, `display.go`, `moves.go`) (depends on T031)
- [x] T033 [P] [US2] Write `backend/internal/activities/currency_test.go`: `TestPlannedCost_MustBePlanCurrency_FR011`, `TestActivityDisplay_FromPlanCurrency_FR013`, `TestActivityForm_TypeDefaultConverted_FR016` (5,000.00 USD → 3,950.00 GBP), `TestGoals_RevenueStaysUSD_FR019` (in a GBP plan)
- [x] T034 [US2] Implement in `backend/internal/activities/` (`service.go`, `display.go`, `form.go`) (depends on T033)
- [x] T035 [P] [US2] Write `backend/internal/lineitems/currency_test.go`: `TestPlannedCostPlan_FR015` (23,000.00 EUR → 19,750.00 GBP), `TestBalance_PlanCurrency_FR015` (A-1012: 5,250.00 GBP), `TestGrid_PlanCurrency_FR015`, `TestRoundingHint_PlanCurrency`, `TestLineItemCurrency_RateDate_FR009`
- [x] T036 [US2] Implement in `backend/internal/lineitems/` (`balance.go`, `grid.go`, `display.go`, `service.go`) with `ActivityFacts.PlanCurrency` filled in `backend/internal/app/lineitems.go` (depends on T035)
- [x] T037 [US2] Convert plan-currency amounts to USD in the adapters: `backend/internal/app/currency.go` (new helpers), `allwork_source.go` (Node `Budget`, `PlannedCost`, `MonthsUSD`, `UnspreadUSD`, plus `BudgetPlan`, `PlannedCostPlan`, `PlanCurrency`, `PlanType`, `PlanYear`), `home_sources.go`, `approvals_adapter.go`; tests `TestAllWorkSource_ConvertsPlanCurrency_FR014`, `TestHomeSources_ConvertsPerPlan_FR014`, `TestApprovals_DisplayFromPlanCurrency` (depends on T005, T030, T034, T036)
- [x] T038 [US2] Handler and contract tests with the seed for plans, campaigns, activities and line items of P-1004 (`handler_test.go` of each package): every changed answer validates against the merged contract (depends on T030–T037)

### Frontend

- [x] T039 [P] [US2] `frontend/src/features/plans/PlanForm.tsx`: Plan type (Basics) and Plan currency (Budget, active currencies, lock note from `currencyLock`), budget `MoneyField` in the plan currency with "≈ X USD" from the API's figures after save; `planFormValues.ts`; tests (FR-011, FR-012, FR-032)
- [x] T040 [P] [US2] Plan page and list: `PlanHeader` chips (type, "Planned in GBP"), `PlanSummaryCards` from `planFigures` with `figures` beside (`AmountPair`), `PlanCard`/`PlanTable` type chip and `AmountPair`; plan tree rows (`features/plans/tree`); tests (FR-013, FR-014, FR-032)
- [x] T041 [P] [US2] Campaigns: `RecordForm` budget in the plan currency, `BudgetCost`, `RecordDetails`, `campaignRows` with `AmountPair`; `Destinations` shows the API's reason; tests (FR-011, FR-013, FR-017)
- [x] T042 [P] [US2] Activities: `ActivityForm` planned cost in the plan currency with the type-default note, `ActivityHeader` and `activityTreeRows` with `AmountPair`; tests (FR-011, FR-013, FR-016)
- [x] T043 [P] [US2] Line items: `LineItemsSection` columns (local, plan currency, display), `UnassignedBalanceRow` and `SpreadGrid` in the plan currency, `CostFields` conversion line "= 19,750.00 GBP at 0.92 EUR and 0.79 GBP per USD (rates of 19 Oct 2026)", `LineItemPage`; tests (FR-009, FR-015)

**Checkpoint**: US2 works alone; existing 002–010 tests green.

---

## Phase 5: User Story 3 — Budget by month (P2, lane P)

**Goal**: phasing and phased vs planned. **Independent test**: spec US3.

### Backend

- [x] T044 [P] [US3] Write `backend/internal/plans/phasing_test.go`: `TestSetPlanPhasing_SavesAndLogs_FR020_FR023` (log text "Budget by month: Mar 2027 50,000.00 → 60,000.00 GBP"), `TestSetPlanPhasing_PublishedStaysPublished_FR023`, `TestSetPlanPhasing_VersionConflict`, `TestPlanBudgetRemoved_ResetsPhasing_FR022`, `TestPublish_PhasingMismatchBlocks_FR022`, `TestPlanCompleteness_PhasingProblem_FR022`
- [x] T045 [US3] Implement `SetBudgetPhasing` in new `backend/internal/plans/phasing.go`, the completeness/publish problem in `status.go`, budget-removal reset in `service.go`; route `PUT /api/plans/{planId}/budget-phasing`; remove it from `notYetServed` (depends on T044)
- [x] T046 [P] [US3] Write `backend/internal/campaigns/phasing_test.go`: the same rules for campaigns and initiatives, `TestCampaignCompleteness_PhasingProblem_FR022`, `TestMoveCopy_KeepPhasing_FR025`
- [x] T047 [US3] Implement new `backend/internal/campaigns/phasing.go`, completeness in `views.go`, routes for the two `PUT …/budget-phasing` operations (depends on T046)
- [x] T048 [P] [US3] Write `backend/internal/allwork/budget_by_month_test.go`: `TestPlannedByMonth_PlanCurrency_FR024` (line items converted with `ConvertParts`, positive balance spread, negative not, no dates, other years), `TestBudgetByMonth_Rows_FR024` (budget, difference, allocated, over flags, "2 campaigns are not phased"), `TestBudgetByMonth_EqualsBudgetAndCost_SC004`
- [x] T049 [US3] Implement `backend/internal/allwork/budget_by_month.go`, `Source.PlanRates` in `source.go` and `backend/internal/app/allwork_source.go`, route `GET /api/plans/{planId}/budget-by-month` (depends on T048, T037)

### Frontend

- [x] T050 [P] [US3] `frontend/src/features/budget-phasing/phasingMath.ts` (BigInt cents: even split of 12, quarter split of 3, totals, left/too much) with tests from data-model §3.3
- [x] T051 [P] [US3] `PhasingEditorPanel.tsx`: method choice, 12-month grid with Monthly · Quarterly entry, Split evenly, Copy from planned cost (from `budget-by-month`, with the left-out note), Use month total as budget, running total, Save via the level's `PUT`; unsaved-changes guard; tests (FR-020, FR-021, FR-023)
- [x] T052 [P] [US3] `BudgetByMonthCard.tsx`: rows Budget, Planned cost, Difference, Allocated, Monthly · Quarterly switch, red "Over by X", Other years / No dates columns, "Not phased"; **Edit** opens the panel; tests (FR-024)
- [x] T053 [US3] Place the card on `PlanPage` (after the summary cards) and in the campaign and initiative panels (`RecordDetails`); completeness links to it (`focusOnArrival`); budget field note "Budget by month will be removed" in `PlanForm` and `RecordForm` (depends on T051, T052)

**Checkpoint**: US3 works alone.

---

## Phase 6: User Story 4 — Split activity (P2, lane P)

**Goal**: split one activity into two. **Independent test**: spec US4.

### Backend

- [x] T054 [P] [US4] Write `backend/internal/lineitems/move_test.go`: `TestMoveToActivity_KeepsIDsValuesSpreads_FR028`, `TestMoveToActivity_LogsMoved_FR030`, `TestMoveToActivity_SwbsFollowsNewActivity_FR029`
- [x] T055 [US4] Implement `MoveToActivity` in new `backend/internal/lineitems/move.go`; add it to `activities.LineItems` in `backend/internal/activities/deps.go` and the adapter in `backend/internal/app/activities.go` (depends on T054)
- [x] T056 [P] [US4] Write `backend/internal/activities/split_test.go`: one test per rule of data-model §3.6 (`TestSplit_NameUnique_FR026`, `TestSplit_DestinationRules_FR026`, `TestSplit_NothingChosen_FR027`, `TestSplit_AmountTooHigh_FR026`, `TestSplit_NoPlannedCostMovesLineItemsOnly`), `TestSplitPreview_MatchesSplit_FR027` (spec US4-1 numbers), `TestSplit_CreatesCopyOfFieldsWithoutGoals_FR028`, `TestSplit_ApprovedNeedsRestart_FR029`, `TestSplit_Logs_FR030`, `TestSplit_SumsUnchanged_SC005`, `TestSplitLinks_DeletedOriginal_FR031`, `TestCopy_DoesNotCarrySplitFrom_FR031`
- [x] T057 [US4] Implement new `backend/internal/activities/split.go` (`SplitOptions`, `PreviewSplit`, `Split`), `splitFrom`/`splitInto` in `view.go`, `ActivitySummary.splitFromId`; routes for the three split operations; contract tests; remove them from `notYetServed` (depends on T055, T056)

### Frontend

- [x] T058 [P] [US4] `frontend/src/features/activities/SplitActivityDialog.tsx`: name, initiative (`DestinationList`), line-item checklist with plan-currency amounts, amount (`MoneyField`, follows ticks until typed), debounced preview of both activities, re-approval line, problems at once; toast with **View**; tests (FR-026 – FR-029)
- [x] T059 [P] [US4] `SplitLinks.tsx` under the activity header ("Split from …", "Split into …", "(deleted)"); tests (FR-031)
- [x] T060 [US4] Add **Split activity** to the activity page's More menu and the activity row's ⋯ menu in `useActivityActions.tsx` and `activityTreeRows.tsx`; tests (depends on T058)

**Checkpoint**: US4 works alone.

---

## Phase 7: User Story 5 — Plan type and multi-year views (P2, lane A)

**Goal**: types, year ranges, yearly and quarterly views. **Independent test**: spec US5.

### Backend

- [x] T061 [P] [US5] Write `backend/internal/plans/list_range_test.go`: `TestListPlans_YearRange_FR033`, `TestListPlans_YearAlias`, `TestListPlans_TypeFilter_FR033`, `TestListPlans_SummaryOverRange_FR033`
- [x] T062 [US5] Implement `yearFrom`, `yearTo`, `type` in `backend/internal/plans/list.go` and `handler.go` (depends on T061)
- [x] T063 [P] [US5] Write `backend/internal/allwork/range_test.go`: `TestQuery_YearRange_FR034` (max 5 years, refused with parameter name), `TestTree_PlansOfEveryYear_FR034`, `TestPeriods_Yearly_FR035`, `TestPeriods_QuarterKeysAcrossYears_FR035`, `TestPeriods_OtherYearsOutsideRange_FR035`, `TestFilters_PlanTypeAndCurrency_FR037`, `TestColumns_YearTypeCurrencyPlanAmounts_FR037`, `TestCompare_PhasedBudgetAndOverPeriods_FR038`, `TestViews_OldYearReadAsRange_FR043`
- [x] T064 [US5] Implement in `backend/internal/allwork/` (`query.go`, `service.go`, `periods.go`, `tree.go`, `columns.go`, `views.go`, `result.go`) the range (`Source.Year` per year), Yearly period, plan type and currency filters, new columns, `phasedBudget`/`overPeriods`, view-state migration; `export.go` states the range and the rates used (FR-009) (depends on T063, T037)

### Frontend

- [x] T065 [P] [US5] `PlansPage.tsx`: Years (`YearRangeField`, allowAll), Plan type filter chips, remembered in `listPreferences.ts` (old `year` read as a one-year range); tests (FR-033)
- [x] T066 [P] [US5] All work `FilterBar.tsx` / `FilterMenu.tsx`: Years range (max 5), Plan type, Plan currency; `allWorkApi.ts` parameters; `viewState.ts` and `allWorkStore.ts` with the new state and migration; tests (FR-034, FR-037, FR-043)
- [x] T067 [P] [US5] `BudgetView.tsx`: Monthly · Quarterly · Yearly, opens on Yearly for a range above one year, **Compare with phased budget** sub-rows and red cells; tests (FR-035, FR-038)
- [x] T068 [P] [US5] `CalendarView.tsx` and `calendar.ts`: timeline over the range with year and quarter bands, "Continues from / into" at the range's edges; tests (FR-036)
- [x] T069 [P] [US5] `ColumnChooser.tsx`: the five new Standard columns; plan rows show their year; tests (FR-037)

**Checkpoint**: US5 works alone.

---

## Phase 8: User Story 6 — Budget categories (P2, lane A)

**Goal**: attribute filters, Breakdown, plan card. **Independent test**: spec US6.

### Backend

- [x] T070 [P] [US6] Write `backend/internal/allwork/filters_attr_test.go`: `TestAttrFilter_TestedOnLevel_FR039` (D-113 rules per level), `TestAttrFilter_NotFilledIn_FR039`, `TestAttrFilter_SplitShareAboveZero_FR039`, `TestAttrFilter_HiddenCountsAsNotFilled_FR039`, `TestAttrFilter_YesNo`, `TestQuery_AttrParameterRefused`
- [x] T071 [US6] Implement new `backend/internal/allwork/filters_attr.go`, the `attr` parameter in `query.go`, `Node.Attrs` filled in `backend/internal/app/allwork_source.go`, filterable attributes in `GET /api/all-work/options` (`options.go`) (depends on T070, T037)
- [x] T072 [P] [US6] Write `backend/internal/allwork/breakdown_test.go`: one test per rule of data-model §3.5 (`TestBreakdown_SingleChoice_FR041`, `_MultipleCountsInEach_FR041`, `_SplitRemainderOnLast_FR041`, `_LineItemAttributeAndNotOnALineItem_FR041`, `_NoPlannedCostNote_FR041`, `_HiddenIsNotFilled`, `_ThenBy_FR040`, `_BudgetGroupings_FR040`, `_BuiltIns_FR040`, `_TotalsEqualCard_SC007`, `_PlanCurrencyForOnePlan_FR042`)
- [x] T073 [US6] Implement new `backend/internal/allwork/breakdown.go` and `GET /api/all-work/breakdown` (`handler.go`); Breakdown in `export.go` (group table, FR-043); remove the operation from `notYetServed` (depends on T072, T071)

### Frontend

- [ ] T074 [P] [US6] `AttributeFilterMenu.tsx` under **More filters** (attributes by level, searchable options, Not filled in, Yes/No) and chips "Brands: Helix, Not filled in"; tests (FR-039)
- [ ] T075 [P] [US6] `BreakdownView.tsx` and the **Breakdown** pill tab: "Break down by" / "then by" pickers, table with expandable groups, notes, totals row, rows open the Grid filtered; tests (FR-040, FR-041)
- [ ] T076 [P] [US6] `frontend/src/features/plans/PlannedCostByCard.tsx` on `PlanPage` (Spend type default, picker, plan currency); tests (FR-042)
- [ ] T077 [US6] Saved views and export menu carry the Breakdown, attribute filters and range (`SaveViewDialog`, `ViewsMenu`, `ExportMenu`); tests (FR-043) (depends on T066, T075)

**Checkpoint**: US6 works alone.

---

## Phase 9: User Story 7 — Field help (P1, lane H) 🎯

**Goal**: admin-maintained ⓘ on every field. **Independent test**: spec US7.

### Backend

- [x] T078 [P] [US7] Create `backend/internal/settings/fieldhelp_catalogue.go` with every key, level, group, label and default of data-model §6; test `TestCatalogue_KeysUniqueAndTextsWithinLimit`
- [x] T079 [P] [US7] Write `backend/internal/settings/fieldhelp_test.go`: `TestListFieldHelp_DefaultsAndEdited_FR045`, `TestUpdateFieldHelp_SavesAndLogs_FR046`, `TestUpdateFieldHelp_ResetDeletesRow_FR046`, `TestUpdateFieldHelp_TooLong_FR046`, `TestUpdateFieldHelp_Version0Conflict_FR046`, `TestUpdateFieldHelp_UnknownKey404`, `TestListFieldHelp_AttributesWithoutHelp_FR048`
- [x] T080 [US7] Implement `backend/internal/settings/fieldhelp.go`, history kind `fieldHelp` (log only), routes `GET /api/field-help`, `PATCH /api/field-help/{helpKey}`; `help=missing` filter on the Shell attributes list; remove the operations from `notYetServed` (depends on T078, T079)
- [x] T081 [US7] Seed a help text for every seeded Shell attribute in `backend/seed/settings.go`; test that none is empty (FR-048)

### Frontend

- [x] T082 [P] [US7] `frontend/src/features/settings/field-help/FieldHelpPage.tsx` and `FieldHelpPanel.tsx` (search, level tabs, chips, Edit with counter, Reset to default with confirmation, version conflict message, note with the Shell-attribute link); route `/settings/field-help`; Settings card in `sections.ts`; tests (FR-045, FR-046, FR-048)
- [x] T083 [P] [US7] Shell attributes list: "No help yet" chip and `?help=missing` filter in `features/settings/shell-attributes/`; tests (FR-048)
- [ ] T084 [P] [US7] Tips on plan form, details, header and summary cards (`PlanForm`, `PlanDetails`, `PlanFacts`, `PlanSummaryCards`, `GoalFields`, `PlanGoals`) from `help()`; fixed tip literals removed; tests assert one ⓘ per labelled field (FR-047)
- [ ] T085 [P] [US7] Tips on campaign and initiative forms, panels and rows (`RecordForm`, `RecordDetails`, `BudgetCost`, `StatusChips`); tests (FR-047)
- [ ] T086 [P] [US7] Tips on the activity form, page and sections (`ActivityForm`, `ActivityHeader`, `ActivitySections`, `GoalsFields`, `SplitLinks`); tests (FR-047)
- [ ] T087 [P] [US7] Tips on line-item form, page and section (`LineItemForm`, `CostFields`, `FinanceBudgetFields`, `SpreadEditor`, `UnassignedBalanceRow`, `SpreadGrid`, `LineItemPage`); tests (FR-047)
- [ ] T088 [P] [US7] Tips on Budget by month (`phasing.*` keys) and All work: `SummaryCards`, `ColumnChooser` via new `frontend/src/features/all-work/columnHelp.ts`, Breakdown share; tests (FR-047)

**Checkpoint**: US7 works alone; SC-008.

---

## Phase 10: End-to-end, documentation and checks

- [x] T089 Write `frontend/e2e/planning-currency.spec.ts` (journeys of plan *Tests* › End-to-end, records prepared through the API as `e2e/planning.ts` does, axe on Currencies with rate history, Import rates, Field help, plan page with Budget by month, Split dialog, All work Breakdown and Budget & cost Yearly); existing `core-flow`, `module-rest`, `smoke` stay green (depends on Phases 3–9)
  - Done 2026-10-29 for the screens built today: the GSAP rate import with its review (axe on Import rates and on
    Currencies › EUR with its rate history), a GBP plan made on the plan form with USD beside its amounts, its
    budget by month evenly and then custom by quarter (axe on the editor and the plan page), a GBP activity
    with a EUR line item split off into a new activity (axe on the Split dialog), and All work over two years
    in Budget & cost Yearly with Compare with phased budget (axe). Every journey uses records of its own. Not
    covered here because another feature lane is finishing them: attribute filters, Breakdown and Field help
    (T074–T077, T084–T088). The plan form shows "≈ … USD" only after saving, as T039 decided, so the journey
    checks it on the plan page.
  - Bug found and fixed: All work's pager ("Rows 1–50 of …", shown from 51 rows) lies on the grey page in
    Text Subtle, which fails WCAG 1.4.3 contrast (axe `color-contrast`); it is now Text Strong, as constitution X
    asks on grey. The journey's axe check found it once the full suite had added enough rows; no component test
    can see it, because jsdom computes no colours.
- [ ] T090 [P] Update `docs/product/domain-model.md`: §4 money rules (plan currency, dated rates, `Convert`, unassigned in the plan currency), §4.1 Budget by month, §2.5 split as a re-approval change, §6 Currency (rate history), §7 records (exchange rate, rate import, field help)
- [ ] T091 [P] Update `docs/product/field-catalogue.md`: §1 Plan type, Plan currency, Budget by month; §1 #8, §2 #5, §3 #6, §4.1 #8 "money (plan currency)"; §2–§3 Budget by month; §4 Split from / Split into; §5 #6 "Planned cost (plan currency)"
- [ ] T092 [P] Update `docs/product/glossary.md` (Plan currency, Current rate, Rate history, GSAP rate import, Budget by month, Phased budget, Split activity, Plan type, Breakdown, Field help) and `docs/product/design-system.md` §4–§5 (AmountPair, RateTip, YearRangeField, Budget by month card)
- [ ] T093 [P] Update `docs/sdd/plan-module-build.md` (016 row in §2, owners of the new shared parts in §4, routes in §5, seed order note in §6) and `docs/sdd/feature-map.md` (016 row and scope boundaries)
- [ ] T094 Run the backend checks (`gofmt -l .` prints nothing, `go vet ./...`, `go test ./...`) and the web checks (`npm.cmd run check`); `notYetServed` holds none of 016's operations (depends on T089)
- [ ] T095 Check SC-001 – SC-009 (timed runs for SC-002, SC-003, SC-006; axe for SC-009) and record the results in `specs/016-planning-and-currency-extensions/checklists/requirements.md`; tick the tasks (depends on T094)

---

## Backend build state (2026-10-20, branch `lane-b016`)

- Every backend task is done (ticked above): the 21 operations are served and out of `notYetServed`; every
  changed operation has an `apitest` handler test against the merged contract; `gofmt`, `go vet` and the
  backend tests pass (the SC-004 timing tests of 010 and 009 can miss their limit on a heavily loaded machine).
- Line-item section and details: as the contract states (`UnassignedBalance`, `SpreadGrid`), the balance and the
  spread grid are worked out in the plan currency and answered in the reader's display currency; the line items
  carry `plannedCostPlan` and `LineItemActivity.plannedCost` is in the plan currency.
- Built on `plan-module` before 014's access control: the route rules of D-301 (exchangeRates.manage,
  settings.manage, plans.edit, campaigns.edit, activities.edit), 014's plan visibility on the Plans list, All work
  and Breakdown, and D-303's allocation lock of the plan currency come with the merge onto 014.
