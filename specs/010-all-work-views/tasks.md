---

description: "Task list for feature 010 — All Work Views"
---

# Tasks: All Work Views

**Input**: Design documents from `specs/010-all-work-views/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); features 001–008 built (002's `money` package and
`Money` schema, 003's display currency and teams, 004–007's records and calculations, 004's `LevelTag`
and `PillTabs`).

**Tests**: required (constitution V). In every task, write the tests first and see them fail. Test names
carry the FR they prove. Every user-visible text comes from the plan's *User-visible messages* table.

**Organization**: Setup, then **Backend** and **Frontend** phases that two agents run in parallel — both
start from the contract (T001) and never wait for each other — then End-to-end and Polish. Story tags:
US1 Grid · US2 Filters and search · US3 Columns and sort · US4 Budget & cost · US5 Calendar · US6 Saved
views · US7 Export.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- Paths are relative to the repository root.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [X] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [X] TC01 Reuse 004's `PillTabs`, `FilterChips`, `TreeTable` and `recordHref` (D-194, D-197).
- [ ] TC99 Check the success criteria SC-001, SC-002, SC-003, SC-005, SC-006, SC-007, which no task above names, in the final review (timed end-to-end run or moderated session, as each criterion says) and record the result in `checklists/`.

## Phase 1: Setup

- [x] T001 Merge `specs/010-all-work-views/contracts/openapi.yaml` into `backend/api/openapi.yaml` (tag, 7 paths, parameters `AllWork*`, `ViewId`, schemas `AllWork*`, `SavedView*`; `Money` already there from 002) and run `npm run api:types` in `frontend/` to regenerate `frontend/src/api/schema.d.ts`; `npm run api:check` passes *(Done at consolidation, D-192.)*
- [X] T002 [P] Create the folders `backend/internal/allwork/`, `backend/internal/platform/sheet/` and `frontend/src/features/all-work/`

**Checkpoint**: the contract is final; backend and frontend start in parallel.

---

## Phase 2: Backend (Go)

### Foundation

- [X] T003 [P] Write `backend/internal/platform/sheet/sheet.go` (`Workbook`, `Sheet{Name, Rows}`, `Cell` constructors `Text`, `Number`, `Money`, `Date`, `Percent`, `Bold`), `xlsx_write.go` (`WriteXLSX(w io.Writer, wb Workbook) error`: `[Content_Types].xml`, `_rels/.rels`, `xl/workbook.xml`, `xl/_rels/workbook.xml.rels`, `xl/styles.xml` with formats `#,##0.00` and `dd mmm yyyy` and a bold font, one `xl/worksheets/sheetN.xml` per sheet with inline strings) and `csv_write.go` (`WriteCSV(w, notes [][2]string, rows [][]Cell)`: UTF-8 BOM, notes, empty line, table; numbers "12500.00", dates `YYYY-MM-DD`) with `xlsx_write_test.go` (`TestXLSX_OpensAsZipWithWorkbookSheetsStrings`, `TestXLSX_NumbersDatesBoldHeadings`, `TestXLSX_EscapesTextAndSheetNames`) and `csv_write_test.go` (`TestCSV_BOMNotesThenTable`)
- [X] T004 Write `backend/internal/allwork/model.go`: the API types of the contract (`Result`, `Row`, `Cell`, `Column`, `ColumnGroup`, `Period`, `Summary`, `Count`, `Totals`, `LeftOut`, `Options`, `RecordLocation`, `Preferences`, `Sort`, `SavedView`, `SavedViewPage`, `SavedViewCreate`, `SavedViewUpdate` with pointer fields, `ViewState`, `Filters`, `FilterValue`, `Expansion`), the stored `SavedView` (with `PersonID`) and `PreferencesRecord`; slices never nil; pointer money fields encode as the contract's null or absence exactly (data-model §1, contract)
- [X] T005 Write `backend/internal/allwork/source.go` (`Source` interface, `YearData`, `Node`, `Currency`, `Location` — plan *allwork.Source*) and the test helper `backend/internal/allwork/fixture_test.go`: a builder of hand-built years (plans, campaigns, initiatives, activities, line items with months, owners, attributes) dated 2026-10-18 and an in-memory `fakeSource`, reproducing the seed story of `docs/product/demo-users-and-data.md` (2026 plans; EUR and GBP line items; one non-zero balance; one negative balance; one activity without dates; one ongoing)
- [X] T006 Write `backend/internal/allwork/query.go` (`Query`, `ParseQuery(url.Values) (Query, error)` reporting every problem at once with the plan's validation messages; repeated parameters; `column` order kept; `sort` split into column and direction) and `query_test.go`: `TestParseQuery_AllParameters_FR004`, `TestParseQuery_AllProblemsAtOnce_FR004`, `TestFilter_OnlyFR004Filters_FR010` (an attribute filter name is an unknown parameter and changes nothing)

### US1 — See all planned work in one grid (P1)

- [X] T007 [US1] Write `backend/internal/allwork/tree.go` (build the tree from `YearData`: plans by name ignoring case, children by `order`, line items by creation, an unassigned row under every activity whose balance is not zero; flatten in tree order with depth and `parentKey`) and `tree_test.go`: `TestTree_PlansByNameChildrenInPlanTreeOrder_FR011`, `TestUnassignedRows_OnlyNonZeroBalances_FR012`
- [X] T008 [US1] Write `backend/internal/allwork/rollup.go` (data-model §5: container roll-ups from matching activities shown, `unfiltered` values, positive-only unassigned + negative count, budget/remaining/budget used, summary cards, totals, conversion to the display currency after summing in USD) and `rollup_test.go`: `TestRollup_ContainersSumMatchesShown_FR013`, `TestRollup_UnfilteredValueWhenRecordsHidden_FR013`, `TestSummary_SixCards_FR003`, `TestSummary_UnassignedPositiveOnlyAndNegativeCount_FR003`, `TestRollup_TotalsEqualSummaryCards_FR014_SC003`, `TestMoney_DisplayCurrency_SumInUSDThenConvert_FR017`, `TestRollup_EmptyPlannedCostCountsZeroAllEmptyIsDash_EdgeCase`
- [X] T009 [US1] Write `backend/internal/allwork/columns.go` — the **Standard** group of data-model §6.1, the default columns, and `cellsFor(row, columns)` (null on other levels, `empty` on own level, money/percent tones, `unfiltered`) — and `columns_test.go`: `TestColumns_DefaultColumns_FR019`, `TestCells_StandardColumnsPerLevel_FR011_FR017` (status chip per level, line item status = completeness, unassigned row amount)
- [X] T010 [US1] Write `backend/internal/allwork/service.go` (`NewService(tx, repo, src, ids, now)`; `Query` for views `grid` and `calendar` without filters yet; `Locate` with the recycle-bin message) and `service_test.go`: `TestQuery_DefaultYearIsCurrentYear_FR004`, `TestQuery_NeverWrites_FR002` (store files and activity log unchanged), `TestQuery_CalendarViewHasNoLineItemRows_FR025`, `TestQuery_TodayAndDatesForCalendar_FR026`, `TestLocate_DeletedRecordInRecycleBin_EdgeCase`, `TestLocate_UnknownRecord_EdgeCase`
- [X] T011 [US1] Write `backend/internal/allwork/source_planning.go` (`NewPlanningSource(...)` on the packages listed in the plan's *Dependencies*: years, nodes with their owning features' statuses, completeness, balances, USD months and goals, display currency, options, locate, recycle-bin lookup through `history.Service.ListBin`) and `source_planning_test.go` (`TestPlanningSource_ReadsSeedYear_FR001`: run on `apitest` seed data; node counts and one value per level equal the seed)
- [X] T012 [US1] Write `backend/internal/allwork/handler.go` (`NewHandler`, `Routes` for `GET /api/all-work` and `GET /api/all-work/records/{recordId}`, both behind `signedIn`) and wire `allwork` in `backend/internal/app/app.go` (repository, planning source, service, routes); `handler_test.go` (`package allwork_test`): `TestGetAllWork_Contract` (default grid on the seed: both 2026 plans, cards, totals), `TestLocate_Contract`

### US2 — Filters and search (P1)

- [X] T013 [US2] Extend `backend/internal/allwork/tree.go` with matching (data-model §3: own level, nearest tested ancestor, any matching descendant; container-status groups; context rows), search, Show levels and the count; tests in `tree_test.go`: `TestFilter_EachFilterOnItsLevel_FR004` (table, one case per filter), `TestFilter_ActivityStatusShowsOnlyBranchesToMatches_FR005`, `TestFilter_OwnerParentsAreContext_FR005`, `TestFilter_HigherLevelFilterInheritedFromNearestAncestor_FR005`, `TestFilter_ContainerStatusOrWithinGroupAndAcross_FR005`, `TestSearch_MatchesNameIdExternalNameDescription_IgnoringCase_FR006`, `TestSearch_LineItemHitMakesActivityMatch_FR006`, `TestSearch_MatchingContainerShowsEverythingInside_FR006`, `TestShowLevels_CompleteBranchesOnly_FR007`, `TestShowLevels_CampaignsWithoutInitiatives_FR007`, `TestShowLevels_InitiativesWithoutActivities_FR007`, `TestCount_ShowingNOfMWithNoun_FR008`
- [X] T014 [US2] Add `Service.Options` and `GET /api/all-work/options` (years with plans + current year, plans, owners, teams with `mine`, business units, activity types, column groups, default columns); tests `TestOptions_YearsOwnersTeamsMineFirst_FR004` in `service_test.go` and `TestGetOptions_Contract`, `TestGetAllWork_InvalidParameters_FR004` in `handler_test.go`

### US3 — Columns and sort (P2)

- [X] T015 [US3] Extend `backend/internal/allwork/columns.go` with the Plan, Campaign, Initiative, Activity and Line item groups and every Shell attribute column (data-model §6.2; inactive ones only when still named; no "later" fields); tests `TestColumns_StandardGroupAndLevelGroups_FR018`, `TestCells_BlankOnOtherLevels_DashOnOwnLevel_SplitShares_FR020`, `TestCells_InactiveValuesLabelled_EdgeCase`
- [X] T016 [P] [US3] Write `backend/internal/allwork/sorting.go` and `sorting_test.go`: `TestSort_ChildrenOfEveryParentAscDescDefault_FR015`, `TestSort_EmptyLast_StatusLifecycle_UnassignedStays_FR015`; use it in `Query`
- [X] T017 [US3] Write `backend/internal/allwork/repository.go` (`GetPreferences`, `PutPreferences`, `GetView`, `ListViews(personID)`, `PutView`, `DeleteView`) and `store_json.go` (tables `all_work_preferences`, `saved_views`); add `Service.Preferences` / `UpdatePreferences` (rules in `views.go`) and `GET`/`PUT /api/all-work/preferences`; tests `TestPreferences_DefaultsAndRemembered_FR019_FR021`, `TestPreferences_RefusesDuplicatesAndTooMany_FR021` in `views_test.go`, `TestPreferences_GetPut_Contract` in `handler_test.go`; `TestQuery_UnknownIdsAndColumnsLeftOut_FR031` in `service_test.go`

### US4 — Budget & cost by month (P2)

- [X] T018 [US4] Write `backend/internal/allwork/periods.go` (data-model §8) and use it for `view=budget` in `Query`; `periods_test.go`: `TestPeriods_EvenlyFirstLastCustom_FR023`, `TestPeriods_PositiveUnassignedSpreadEvenly_OngoingFirstMonth_FR023`, `TestPeriods_NegativeBalanceNotSpread_FR023`, `TestPeriods_OtherYearsAndNoDatesOnlyWhenUsed_FR022`, `TestPeriods_QuarterIsSumOfMonths_FR022`, `TestPeriods_ContainersAndFooter_GrandTotalEqualsPlannedCostCard_FR024`

### US5 — Calendar (P2)

- [X] T019 [US5] Extend `TestGetAllWork_Contract` with `view=calendar` (no line-item or unassigned rows; dates, ongoing, owner, status and planned cost on every row; `today`) — the server side of FR-025 – FR-027 is T010; FR-027 and FR-028 have no server part

### US6 — Saved views (P2)

- [X] T020 [US6] Add the saved-view rules to `backend/internal/allwork/views.go` (name, uniqueness per person, state validation, one default, own views only), `Service.ListViews` / `CreateView` / `UpdateView` / `DeleteView`, routes `GET`/`POST /api/saved-views`, `PATCH`/`DELETE /api/saved-views/{viewId}`; tests `TestCreateView_NameRequiredMax60UniquePerPersonIgnoringCase_FR029`, `TestViews_PersonalOnly_FR030`, `TestViews_OneDefaultPerPerson_FR031`, `TestViews_RenameUpdateStateDelete_FR031`, `TestViews_NotLogged_D_i` in `views_test.go`; `TestSavedViews_CRUD_Contract` in `handler_test.go`
- [X] T021 [US6] Add the two saved views of Priya Raman to `backend/seed/seed.go` (plan *Demo data*) and check them in `backend/seed/seed_test.go` (`TestSeed_SavedViews_FR029`: two views for U-1002, none default, none for U-1003)

### US7 — Export (P2)

- [X] T022 [US7] Write `backend/internal/allwork/export.go` (`Service.Export`, notes, columns per view, Level/Path/"Shown for context"/Currency columns, totals row, file name and `Content-Disposition`) and `GET /api/all-work/export`; `export_test.go`: `TestExport_GridHasEveryRowLevelPathColumnsTotals_FR032`, `TestExport_CalendarAndBudgetColumns_FR032`, `TestExport_MoneyNumbersCurrencyHeadingsDatesStatusWords_FR033`, `TestExport_NotesBeforeData_AboutSheet_FileName_FR034`, `TestExport_UnassignedFollowsSwitch_FR012`; `handler_test.go`: `TestExport_XlsxAndCsvContentTypeAndDisposition_FR034`, `TestEveryAllWorkOperationRequiresSignIn`

### Performance

- [X] T023 Write `backend/internal/allwork/stress_test.go`: `TestQuery_StressYear_SC004` (20 plans, 5,000 activities, 20,000 line items built with the fixture builder; grid with all default columns, budget and calendar each answer in under 1 second; skipped with `-short`)

**Checkpoint**: `go test ./...` green; `TestRoutesMatchContract` green.

---

## Phase 3: Frontend (web app)

### Foundation

- [X] T024 [P] Write `frontend/src/lib/download.ts` (`saveBlob(blob, fileName)`, `fileNameFromDisposition(header)`, `downloadFromApi(request)` turning failures into `ApiError` like `call()`) and `download.test.ts`
- [X] T025 [P] Write `frontend/src/features/all-work/viewState.ts` (`standardState(prefs, year)`, `toQueryParams`, `fromAddress`, `sameState`, `isExpanded`, `visibleRows`, `pageOf`, `chipsOf`, `clearAll`, `leftOutOf`) and `viewState.test.ts` covering FR-006, FR-008, FR-009, FR-011, FR-018, FR-030, FR-031 and D-122 (plan *Web tests*)
- [X] T026 [P] Write `frontend/src/features/all-work/allWorkStore.ts` (per-person store, `useAllWorkState()` with `useSyncExternalStore`) and `allWorkStore.test.ts` (kept between visits, per person, page/size/scroll kept — FR-009, FR-016)
- [X] T027 Write `frontend/src/features/all-work/allWorkApi.ts`: `useAllWork(params)` (`staleTime: Infinity`, no refetch on focus, `keepPreviousData`), `useAllWorkOptions(year)`, `useAllWorkPreferences()`, `useUpdatePreferences()` (debounced 500 ms), `useSavedViews()`, `useCreateView()`, `useUpdateView()`, `useDeleteView()`, `locateRecord(id)`
- [X] T028 [P] Write `frontend/src/features/all-work/recordHref.ts` (addresses per level, plan *Dependencies*) with `recordHref.test.ts`, and `Highlight.tsx` with `Highlight.test.tsx`

### US1 — Grid (P1)

- [X] T029 [US1] Write `CellValue.tsx`, `GridRow.tsx`, `Pager.tsx` and `AllWorkGrid.tsx` (+ `AllWorkGrid.module.css`) in `frontend/src/features/all-work/` per the plan's *Grid*; `AllWorkGrid.test.tsx`: level tags and links, AOP tag, status chip in the name when the Status column is hidden (D-118), context rows labelled (FR-005), expand/collapse by mouse and keyboard and Expand/Collapse all (FR-011), unassigned rows and the switch (FR-012), Filtered mark and tip (FR-013), totals row (FR-014), row click → `locateRecord` → navigate, and 404 → toast + refetch (FR-016, edge case), money in display currency and local column (FR-017), long text tooltip and "(inactive)" (edge cases), pager and "Continued:"
- [X] T030 [P] [US1] Write `frontend/src/features/all-work/SummaryCards.tsx` and `SummaryCards.test.tsx`: six cards, tips, compact value with full value, negative-balance line (FR-003)
- [X] T031 [US1] Write `frontend/src/features/all-work/AllWorkPage.tsx` (+ css): header, cards, tabs (`PillTabs`), toolbar (Expand/Collapse all, unassigned switch, Refresh, "Updated at"), the grid, every state of the plan's *States* table; replace the Coming-soon route in `frontend/src/app/router.tsx` and remove `ALL_WORK` from `COMING_SOON_SECTIONS` in `frontend/src/app/navigation.ts` (update `navigation`/`router` tests that list Coming-soon pages); `AllWorkPage.test.tsx`: FR-001 (header, tabs keep filters, search and expansion), loading, error + Try again, "No plans in <year> yet" + Go to Plans, "Nothing matches these filters" + Clear filters, Refresh

### US2 — Filters and search (P1)

- [X] T032 [US2] Write `frontend/src/features/all-work/FilterMenu.tsx` and `FilterBar.tsx` (+ css): search box with debounce, the eleven FR-004 filters, My teams first, chips with remove buttons, Clear all (keeps the year), the count with `aria-live`; `FilterMenu.test.tsx`, `FilterBar.test.tsx`: exactly the FR-004 filters (FR-010), immediate apply (FR-009), chips and Clear all (FR-008), count text (FR-008)
- [X] T033 [US2] Connect the filter bar and the address filters (D-122) in `AllWorkPage.tsx`; first visit applies the standard state; extend `AllWorkPage.test.tsx` (FR-004, FR-009, D-122)

### US3 — Columns and sort (P2)

- [X] T034 [US3] Write `frontend/src/features/all-work/ColumnChooser.tsx` (+ css) and add sortable headings (`aria-sort`, three-step cycle) to `AllWorkGrid.tsx`; `ColumnChooser.test.tsx`: groups and search, Name locked with its reason, adding a Campaign column, Move up/down and drag, Reset, preferences saved only without an applied view (FR-018, FR-019, FR-021); sort cases in `AllWorkGrid.test.tsx` (FR-015)

### US4 — Budget & cost (P2)

- [X] T035 [US4] Write `frontend/src/features/all-work/BudgetView.tsx` (+ css) and add it to the page with the Monthly · Quarterly switch; `BudgetView.test.tsx`: Plan budget column, months and quarters, Other years / No dates, negative-balance text, footer, read-only note, row opens its record (FR-022 – FR-024)

### US5 — Calendar (P2)

- [X] T036 [US5] Write `frontend/src/features/all-work/CalendarView.tsx` (+ css) and add it to the page; `CalendarView.test.tsx` with the date fixed at 2026-10-18: bars placed by dates, colour + word + legend, Today line, Ongoing open end, Continues from/into, "No dates yet (N)" with links, tooltip lines, no drag handles (FR-025 – FR-027)
- [X] T037 [US5] Write `frontend/src/features/all-work/MonthCalendar.tsx` (+ css) and the Timeline · Month switch; `MonthCalendar.test.tsx`: Monday-first grid, ‹ › and Today, activities on their days, "+ n more" (FR-028, P3)

### US6 — Saved views (P2)

- [X] T038 [US6] Write `ViewsMenu.tsx`, `SaveViewDialog.tsx`, `ManageViewsPanel.tsx` in `frontend/src/features/all-work/` and the first-visit default-view logic in `AllWorkPage.tsx`; tests `ViewsMenu.test.tsx`, `SaveViewDialog.test.tsx`, `ManageViewsPanel.test.tsx`: save with name rules and the replace question, active view and Modified with Update view / Save as new view, default set/unset and applied on first visit, rename, delete with Undo re-creating it, left-out toast (FR-009, FR-029 – FR-031)

### US7 — Export (P2)

- [X] T039 [US7] Write `frontend/src/features/all-work/ExportMenu.tsx` and `ExportMenu.test.tsx`: both formats call the export with the current query and `unassigned`, the file is saved under the server's name, "Preparing your file…", "Exported '<file name>'", "Nothing to export — clear a filter first." without a call (FR-032 – FR-034)

**Checkpoint**: `npm run check` green.

---

## Phase 4: End-to-end

- [ ] T040 Write `frontend/e2e/all-work.spec.ts` with the plan's journeys 1–5 (US1, US2, US4, US6, US7) against the seeded API (`signInAs` from `fixtures.ts`; unique view names; no global counts asserted)
- [ ] T041 Add to `frontend/e2e/all-work.spec.ts` the axe checks (light and dark) of the Grid, Calendar, Budget & cost, the column chooser and the Views menu, and a keyboard-only journey: search, open a filter, expand a row, sort a column, open a record (SC-008)

---

## Phase 5: Polish

- [ ] T042 [P] Update `backend/README.md` (package list: `allwork`, `platform/sheet`) and `frontend/README.md` (folders: `features/all-work`, `lib/download.ts`); set 010 to Done in `docs/sdd/feature-map.md`; add this plan's proposed decisions to `docs/sdd/decision-log.md` as numbered by the consolidation step and close Q-009 with its default
- [ ] T043 Run `gofmt -l .`, `go vet ./...`, `go test ./...` (and once without `-short` for T023) in `backend/`, and `npm run check` and `npm run test:e2e` in `frontend/`; fix what fails
- [ ] T044 Demonstrate on the seed data as Priya Raman: every scenario of the spec, the screens checked against `docs/product/design-system.md` (layout, tokens, chips, empty states, 1280 × 720 without page-wide horizontal scroll)

---

## Dependencies & Execution Order

- T001 → everything. T002 any time.
- **Backend**: T003, T004, T005 → T006 → T007 → T008 → T009 → T010 → T011 → T012; then T013 → T014; T015, T016 [P], T017; T018; T019; T020 → T021; T022; T023 last.
- **Frontend**: T024, T025, T026, T028 [P] → T027 → T029, T030 [P] → T031 → T032 → T033; then T034, T035, T036 → T037, T038, T039 in any order.
- Backend and frontend phases never depend on each other (frontend tests use `mockApi` with contract-shaped answers).
- **End-to-end** (T040, T041) needs both phases. **Polish** last.

## FR coverage (every FR has a test task)

| FR | Tasks |
|---|---|
| FR-001 | T011, T012, T031, T040 |
| FR-002 | T010 |
| FR-003 | T008, T030 |
| FR-004 | T006, T013, T014, T032, T033 |
| FR-005 – FR-007 | T013, T029, T040 |
| FR-008 | T013, T025, T032 |
| FR-009 | T025, T026, T033, T038 |
| FR-010 | T006, T032 |
| FR-011 | T007, T009, T025, T029 |
| FR-012 | T007, T022, T029 |
| FR-013, FR-014 | T008, T029 |
| FR-015 | T016, T034 |
| FR-016 | T010, T026, T029, T040 |
| FR-017 | T008, T009, T029 |
| FR-018 – FR-020 | T009, T015, T025, T034 |
| FR-021 | T017, T034 |
| FR-022 – FR-024 | T018, T035, T040 |
| FR-025 – FR-027 | T010, T019, T036 |
| FR-028 | T037 |
| FR-029 – FR-031 | T017, T020, T021, T025, T038, T040 |
| FR-032 – FR-034 | T003, T022, T039, T040 |
| SC-004 | T023 |
| SC-008 | T041 |

## Implementation Strategy

### MVP first (User Stories 1 and 2)

1. Setup; backend foundation, US1 and US2 (T003 – T014); frontend foundation, US1 and US2 (T024 – T033).
2. Stop and validate: on the seed data, All work shows both 2026 plans, the totals row equals the
   **Planned cost** card, an activity is found by its ID, and a filter shows its chip and the count
   (US1 and US2 independent tests).

### Incremental delivery

1. + US3 (columns and sort) → + US4 (Budget & cost) → + US5 (Calendar; the Month layout of FR-028 last,
   it is P3) → + US6 (saved views) → + US7 (export).
2. After each story: its tests and its journey in `all-work.spec.ts`; commit with the feature number and
   FRs, e.g. `feat(010): budget and cost by month (FR-022 – FR-024)`.

### Notes

- Never edit `frontend/src/api/schema.d.ts` by hand: change `backend/api/openapi.yaml`, then
  `npm run api:types`.
- Keep every user-visible text exactly as in the plan's *User-visible messages* table; tests quote it.
- All work never adds money in the web app: every sum, roll-up, period amount and conversion comes from
  `GET /api/all-work` (plan, D-114).
- Feature 012 extends this feature only through `AllWorkGrid`'s `selection` prop, the pager's visible
  rows, `platform/sheet` and `lib/download.ts`; keep those seams small and documented.
- Mark each task `[X]` here when it is done.
