---

description: "Task list for feature 002 — Settings and Shell Attributes"
---

# Tasks: Settings and Shell Attributes

**Input**: Design documents from `specs/002-settings-and-shell-attributes/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); feature 001 built (history core, record.Meta, components,
`apitest`, `mockApi`, e2e fixtures)

**Tests**: required — constitution V. Within each group write the tests first and see them fail, then
implement. Test names carry the FR they prove (plan *Tests*). Every user-visible text comes from the plan's
*User-visible messages* table; every rule from the plan and data-model.md (when they seem to differ, the
plan wins and the difference is reported).

**Organization**: Setup (the contract) → **Backend** and **Frontend** phases, which are independent of each
other and both start from the merged contract (a backend agent and a frontend agent work in parallel; the
frontend uses `mockApi` answers shaped by the contract) → End-to-end (needs both) → Polish. Inside each
phase, tasks are grouped by user story: US1 models (P1), US2 Shell attributes and options (P1), US3
dependent options and business-unit conditions (P1), US4 currencies (P1), US5 activity types and finance
budget items (P2), US6 option import (P3).

**Built in parallel with 003**: 003 also adds `errs.InUse` / `CodeInUse`, the web error kind `inUse`,
`SearchField` and `TextAreaField`, and has look-alikes of `SegmentedFilter` (`FilterPills`), `Breadcrumb`
(`PageHeader` `breadcrumbs`) and `ChoiceField`'s searchable mode (`Combobox`) — plan *Overlap with feature
003*. Before T003, T034, T038 and T039, look at what is already on the branch: when 003 merged first,
reuse its definition and add only the missing props, never a second copy.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 … US6
- Paths are relative to the repository root. Toolchain in Git Bash:
  `export PATH="$HOME/tools/node:/c/Program Files/Go/bin:$PATH"`.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [X] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [ ] TC01 Build `Combobox` only if 003 has not (one component, 003's design) and base `ChoiceField`'s searchable mode on it (D-197).
- [X] TC02 `MissingField.reason` is optional: Go `json:"reason,omitempty"`, tests accept its absence (D-193).

## Phase 1: Setup (the shared contract)

**Purpose**: the API contract both sides build from. Blocks Phases 2 and 3.

- [x] T001 Merge `specs/002-settings-and-shell-attributes/contracts/openapi.yaml` into `backend/api/openapi.yaml`: the six tags (Settings, Models, Activity types, Shell attributes, Currencies, Finance budgets), the 31 paths (47 operations), the parameters `ModelId`, `ActivityTypeId`, `AttributeId`, `OptionId`, `CurrencyCode`, `BudgetId`, `ItemId`, `LevelPath` and every schema of the fragment; add `in_use` and `cannot_restore` to `Error.code` and name them in the `Conflict` response description; mention feature 002 in `info.description`. No name clash with 001 (checked: the fragment only refers to 001's `BadRequest`, `Unauthorized`, `NotFound`, `Conflict`, `Error`, `DeleteResult`, `RecordRef`, `ActorRef`, `FieldError`) *(Done at consolidation, D-192.)*
- [x] T002 Regenerate `frontend/src/api/schema.d.ts` with `npm run api:types` in `frontend/`; `npm run api:check` passes *(Done at consolidation, D-192.)*

**Checkpoint**: contract merged — Phases 2 and 3 start in parallel.

---

## Phase 2: Backend (Go API)

### Backend foundation (T003, T004, T006 in parallel; T005 after T004; then T007 → T011)

- [ ] T003 [P] Add to `backend/internal/platform/errs/errs.go` `CodeInUse = "in_use"`, `CodeCannotRestore = "cannot_restore"` and the constructors `InUse(message string) *Error`, `CannotRestore(message string) *Error` (409, `Fields` empty; keep 003's `InUse` if it is already there); `TestInUseAndCannotRestore_409` in `backend/internal/platform/errs/errs_test.go`
- [ ] T004 [P] Write the money tests (plan *Tests*, row `platform/money`) in `backend/internal/platform/money/money_test.go`, `rate_test.go`, `percent_test.go`: `TestParseAmount_FormatsAndLimits_D010`, `TestAmountJSON_StringWithTwoDecimals_D010`, `TestToUSD_RoundsHalfUp_FR029`, `TestFromUSD_RoundsHalfUp_D010`, `TestRateInverse_SixDecimals_FR028`, `TestParseRate_Limits_FR028`, `TestConvertSpread_LastMonthTakesDifference_D024`, `TestSplitEvenly_RemainderLast_D024`, `TestPercentSplitEvenly_R10`, `TestOfWhole_HalfUp_FR010`, `TestMulInt_OutOfRange_D010`, `TestRates_UnknownCurrency_FR029`
- [ ] T005 Write `backend/internal/platform/money/money.go` (`Amount`, `Money`, `Input`, `ParseAmount`, `ParseInput`, `FromCents`, `Cents`, `String`, `Format`, `Add`, `Sub`, `Neg`, `Cmp`, `Sign`, `IsZero`, `MulInt`, `SplitEvenly`, `Sum`, JSON as a string, the errors `ErrInvalidAmount`, `ErrOutOfRange`, `ErrInvalidCurrency`, `ErrUnknownCurrency`, unexported `roundDiv` with `math/big`), `rate.go` (`Rate`, `One`, `ParseRate`, `String`, `Format`, `Inverse`, `IsOne`, `ToUSD`, `FromUSD`, `ConvertSpread`, `Rates`, `Rates.ToUSD`, `Rates.FromUSD`, `ErrInvalidRate`) and `percent.go` (`Percent`, `Hundred`, `ParsePercent`, `String`, `Format`, `OfWhole`, `PercentSplitEvenly`, `SumPercent`, `ErrInvalidPercent`) exactly as plan *Package `money`* and data-model §1; package doc comment says it is owned by 002 and used by every feature; T004 passes
- [X] T006 [P] Add `(*Service).SnapshotsInBin(ctx, kindName string) ([]Snapshot, error)` to `backend/internal/history/service.go` (snapshots of that kind in unresolved recycle-bin entries, the deleted records and their contents, oldest entry first; joins the caller's transaction); `TestSnapshotsInBin_KindAndContents_FR003` in `backend/internal/history/service_test.go`
- [ ] T007 Create `backend/internal/settings/model.go` (the exported stored types and enums of plan *How later features read Settings*: `Level` + constants, `AttributeType`, `Marking`, `Model`, `Goals`, `ActivityType`, `Attribute` + `IsChoice`, `Option`, `Currency`, `FinanceBudget`, `FinanceBudgetItem`; JSON names of data-model §2–§8), `views.go` (API types with the contract's JSON names: views, inputs, updates, `StatusUpdate`, `MoveInput`, `OptionFilter`, pages, `ChangeImpact`, `Usage`, `UsagePart`, `Lock`, `SettingRef`, `SettingsOverview`, `FormView`, `CheckRequest`), `values.go` (`AttributeValues`, the value shapes of data-model §9, `AttributeInput`, `AttributeResult`, `ClearedValue`, `MissingField`), `repository.go` and `store_json.go` (`NewJSONRepository()`; tables `models`, `activity_types`, `shell_attributes`, `shell_attribute_options`, `currencies`, `finance_budgets`, `finance_budget_items`; `OptionsOf`, `ItemsOf`; lists sorted as `store.List`, never nil)
- [X] T008 Write `backend/internal/settings/service.go`: `Service`, `NewService(tx, repo, ids, hist, now)`, `AddReferenceSource`, the shared helpers of plan *Shared rules* (`sameName`, trimming, Unicode lengths, version check, status change with `deactivated` / `reactivated`, the delete guard), `Overview`; in `backend/internal/settings/service_test.go` the helper `newTestService(t, sources ...ReferenceSource)` (temporary store, ID generator, history service with the settings kinds registered, fixed clock 2026-03-12 09:00 UTC, actors Alex and Daniel), a `fakeSource` builder for `Reference` lists (live and in the bin), and `TestOverview_ActiveCounts_FR001`
- [X] T009 Write `backend/internal/settings/usage.go` (`Reference`, `ReferenceSource`, `buildUsage` reading every source once plus the Settings tables and their bin snapshots; `Usage` with parts in data-model §10 order, " (<n> in the recycle bin)" suffixes, `summary`, `records` ≤ 50 for models and budgets; the counting functions impact statements reuse, live references only) and `backend/internal/settings/usage_test.go`: `TestUsage_SummaryAndRecycleBin_FR003`, `TestUsage_SettingsInBinCount_EdgeCase`
- [X] T010 Write `backend/internal/settings/kinds.go` (`(*Service).Kinds()` — the 7 kinds of data-model §11 in plan order with singular/plural, parent kinds, snapshot names, `Describe` texts of data-model §12, `PrepareRestore` rules of §11 using 001's `restoredName` approach, `cannot_restore` and `name_taken` texts of the plan's *Restore (API)* row) and `backend/internal/settings/kinds_test.go`: `TestKinds_DescribeTexts_FR006`
- [X] T011 Write `backend/internal/settings/handler.go` (`NewHandler`, `Routes(r, signedIn)` with `GET /api/settings/overview`; later stories add their routes) and wire `backend/internal/app/app.go` (build the settings repository and service in every environment, before the demo-only samples; register `settingsService.Kinds()` with history; `settingsHandler.Routes`; a commented place for the `AddReferenceSource` calls of 003–007); `TestSettingsKindsRegistered_FR006` in `backend/internal/app/app_test.go`; `TestOverview_API_FR001` in `backend/internal/settings/handler_test.go` (`package settings_test`, `apitest`). `TestRoutesMatchContract` stays red until T031 registers the last route

### Backend — User Story 1: models and the goal waterfall (P1)

- [X] T012 [US1] Write the tests in `backend/internal/settings/waterfall_test.go` (`TestWaterfall_SeededModels_FR010_SC006` with the data-model §2.1 table, `TestWaterfall_RoundsHalfUp_FR010`, `TestWaterfall_TooLarge_FR010`) and `backend/internal/settings/service_models_test.go` (the 18 tests of plan *Tests*, row `service_models_test.go`, from `TestCreateModel_ListedActive_FR009` to `TestModelRestore_NameTaken_FR004`; activity types put directly through the repository; plans as fake references)
- [X] T013 [US1] Write `backend/internal/settings/waterfall.go` (`Waterfall(inq int64, m Model) (Goals, error)`) and `backend/internal/settings/service_models.go` (`ListModels`, `GetModel`, `CreateModel`, `UpdateModel` with the in-use details, `ModelImpact`, `SetModelActive`, `DeleteModel`) per plan *Models*; T012 passes
- [X] T014 [US1] Register the 7 model routes in `backend/internal/settings/handler.go` (list, create 201, get, patch, impact, status, delete → `history.DeleteResult`); add to `backend/internal/settings/handler_test.go` `TestModelsAPI_ListCreateGet_FR009`, `TestModelsAPI_UpdateConflict_FR007`, `TestModelsAPI_Impact_FR011`, `TestModelsAPI_Status_FR004`, `TestModelsAPI_DeleteAndUndo_FR004` (Undo through `POST /api/recycle-bin/{entryId}/undo`) and `TestSettings_AnySignedInPersonMayChange_FR008` (Daniel edits Alex's model)

### Backend — User Story 2: Shell attributes and options (P1)

- [X] T015 [US2] Write the tests in `backend/internal/settings/service_attributes_test.go` (the 20 tests of plan *Tests* row, `TestCreateAttribute_ListedLastInSection_FR019` … `TestDeleteAttribute_WithOptions_Undo_FR004`), `backend/internal/settings/order_test.go` (the 7 order tests) and `backend/internal/settings/service_options_test.go` (the 11 option tests, `TestAddOption_UniqueLabel_FR018` … `TestOptionUsage_PerOption_FR020`; 5,000 options generated in the test)
- [X] T016 [US2] Write `backend/internal/settings/service_attributes.go` (`ListAttributes`, `GetAttribute`, `CreateAttribute` with options, links and condition — all problems together, `UpdateAttribute` with locks, type change removing options permanently (D-049), becoming dependent with `optionLinks`, `AttributeImpact`, `SetAttributeActive`, `DeleteAttribute` with its options) and `backend/internal/settings/order.go` (display order of data-model §4.2, `MoveAttribute` for attribute and section moves with the dependent and Business-unit checks); the attribute tests and order tests of T015 pass
- [X] T017 [US2] Write `backend/internal/settings/service_options.go` (`ListOptions` with `q`, `status`, `limit`, `offset` and their messages, `CreateOption`, `UpdateOption`, `SetOptionActive`, `MoveOption`, `DeleteOption` with the last-option lock); the option tests of T015 pass
- [X] T018 [US2] Register the 8 attribute routes and the 6 option routes (list, create, patch, status, move, delete) in `backend/internal/settings/handler.go`; add `TestShellAttributesAPI_ListByLevel_FR019`, `…_Create_FR016`, `…_Get_FR020`, `…_Update_FR021`, `…_Impact_FR021`, `…_Status_FR023`, `…_Move_FR019`, `…_Delete_FR004`, `TestOptionsAPI_ListSearch_SC007`, `…_Create_FR018`, `…_Update_FR018`, `…_Status_FR018`, `…_Move_FR018`, `…_Delete_FR004` to `backend/internal/settings/handler_test.go`

### Backend — User Story 3: dependent options, conditions and the reader API (P1)

- [X] T019 [US3] Write the tests in `backend/internal/settings/form_test.go` (the 13 tests of plan *Tests*, row `form_test.go`, `TestForm_SectionsInOrderOfFirstAttribute_FR019_FR024` … `TestDescribeAttribute_Texts_FR006`) and `backend/internal/settings/readers_test.go`: `TestReaders_SortedWithoutBinRecords_FR007` (every reader of plan *Readers*; a deleted record is not returned; called inside a caller's `Write`), `TestCanChoose_KeepsInactiveNeverChoosesNew_FR004`, `TestVisible_ConditionAnyBusinessUnit_FR023_FR026`
- [X] T020 [US3] Write `backend/internal/settings/form.go` (`Form`, `CheckAttributes`, `CheckValues`, `MissingAttributes`, `BusinessUnit`, `DescribeAttribute`, `Visible`, `allowedOptions`, `noOptionsForParent`, `midSentence`), `Diff` in `backend/internal/settings/values.go`, and `backend/internal/settings/readers.go` (`Model`, `Models`, `Allows`, `ActivityType`, `ActivityTypes`, `Attributes`, `Options`, `Option`, `Currency`, `Currencies`, `Rates`, `FinanceBudget`, `FinanceBudgetsOfYear`, `FinanceBudgetItem`, `FinanceBudgetItems`, `CanChoose` — no actor, inside `s.tx.Read`); T019 passes
- [X] T021 [US3] Register `GET /api/shell-attribute-forms/{level}` and `POST /api/shell-attribute-forms/{level}/check` in `backend/internal/settings/handler.go`; add `TestFormsAPI_Get_FR024` and `TestFormsAPI_Check_FR025` (Region auto-fill case, a cleared Region after a Cluster change, a hidden Lubricants attribute not missing) to `backend/internal/settings/handler_test.go`

### Backend — User Story 4: currencies and exchange rates (P1)

- [X] T022 [US4] Write the 11 tests of plan *Tests*, row `service_currencies_test.go`, in `backend/internal/settings/service_currencies_test.go` (`TestCreateCurrency_JPY_FR028` … `TestRestoreCurrency_CodeTaken_FR028`; line items and people as fake references; the US4-3 figures 14 line items, 6 activities, 52,173.91 → 50,526.32 USD)
- [X] T023 [US4] Write `backend/internal/settings/service_currencies.go` (`ListCurrencies` USD first, `GetCurrency`, `CreateCurrency`, `UpdateCurrency` with `rateChangedAt/By` and the log details, `CurrencyImpact`, `SetCurrencyActive`, `DeleteCurrency`, the USD locks); T022 passes
- [X] T024 [US4] Register the 7 currency routes in `backend/internal/settings/handler.go`; add `TestCurrenciesAPI_ListCreateGet_FR028`, `…_UpdateConflict_FR007`, `…_Impact_FR030`, `…_Status_FR004`, `…_DeleteAndUndo_FR004`, `…_USDLocked_FR028` to `backend/internal/settings/handler_test.go`

### Backend — User Story 5: activity types and finance budget items (P2)

- [X] T025 [US5] Write the tests of plan *Tests*, rows `service_activity_types_test.go` (9 tests) and `service_finance_test.go` (11 tests), in `backend/internal/settings/service_activity_types_test.go` and `backend/internal/settings/service_finance_test.go`
- [X] T026 [US5] Write `backend/internal/settings/service_activity_types.go` (`ListActivityTypes`, `GetActivityType`, `CreateActivityType` writing each listed model with its own `changed` entry, `UpdateActivityType` with `OptionalMoney` default cost and the complete `modelIds` list, `ActivityTypeImpact`, `SetActivityTypeActive`, `DeleteActivityType`); the activity-type tests of T025 pass
- [X] T027 [US5] Write `backend/internal/settings/service_finance.go` (`ListFinanceBudgets` with the default year and `years`, `CreateFinanceBudgetItem` finding or creating the budget, `GetFinanceBudgetItem`, `UpdateFinanceBudgetItem` moving between budgets of the same year, `RenameFinanceBudget`, `SetFinanceBudgetItemActive`, `DeleteFinanceBudgetItem`); the finance tests of T025 pass; now that every kind has its service, add `TestEveryChangeLogged_FR006` (create, change, inactivate, reactivate, delete, restore for each kind) to `backend/internal/settings/service_test.go`
- [X] T028 [US5] Register the 7 activity-type routes, `GET /api/finance-budgets`, `PATCH /api/finance-budgets/{budgetId}` and the 5 finance-budget-item routes in `backend/internal/settings/handler.go`; add `TestActivityTypesAPI_ListCreateGet_FR013`, `…_UpdateConflict_FR007`, `…_Impact_FR012`, `…_Status_FR004`, `…_DeleteAndUndo_FR004`, `…_ModelsEditedFromType_FR013`, `TestFinanceAPI_ListYear_FR015`, `…_Rename_FR015`, `…_CreateGetUpdate_FR014`, `…_Status_FR004`, `…_Delete_FR004` to `backend/internal/settings/handler_test.go`

### Backend — User Story 6: option import (P3)

- [X] T029 [US6] Write the tests in `backend/internal/settings/importfile_test.go` (`TestReadRows_CSVCommaSemicolonBOM`, `TestReadRows_XLSXSharedAndInlineStrings` — the `.xlsx` built in the test with `archive/zip`, `TestReadRows_RefusesXLS`) and `backend/internal/settings/service_import_test.go` (`TestImportCheck_RowOutcomes_FR031` with 1,217 generated rows and 3 faulty, `TestImport_AddsOnlyNewValid_OneLogEntry_FR031`, `TestImport_NeverRenamesOrReactivates_FR031`, `TestImport_DependentParentColumn_FR031`, `TestImport_RowAndSizeLimits_FR031`, `TestImport_NothingToAdd_FR031`)
- [X] T030 [US6] Write `backend/internal/settings/importfile.go` (`readRows`: zip signature → first worksheet columns A and B with shared and inline strings and the file's row numbers; otherwise CSV with BOM removal and `;` detection; `.xls` refused) and `backend/internal/settings/service_import.go` (`CheckImport`, `Import` per plan *Options*; the file is decoded in memory, never written to disk or logged); T029 passes
- [X] T031 [US6] Register `POST …/options/import-check` and `POST …/options/import` in `backend/internal/settings/handler.go`; add `TestImportAPI_CheckAndImport_FR031` (a base64 CSV over 700 KB → `validation_failed` on `content`) to `backend/internal/settings/handler_test.go`

### Backend — demo data (FR-032)

- [ ] T032 Add `addSettings(ctx, generator, createdAt) (SettingsSeed, error)` to `backend/seed/seed.go` (data-model §13 in its order — currencies, activity types, models, finance budgets and items, Shell attributes with options, links and conditions; `actor.System`; no log entries; inside `Load`'s `Write` after the people) and the exported `SettingsSeed` (ids by name, keys of plan *Demo data*) that `Load` keeps for the planning seed of 004–007; `TestSeed_Settings_FR032` in `backend/seed/seed_test.go` (5 currencies, 6 types, 2 models, 2 budgets with 6 items, 35 attributes, 110 options; Region → Cluster and GL code → Spend type links; Business unit `role`; markings and conditions; ids `MOD-1001`, `SA-1001`); the `cmd/seed` message does not change
- [X] T033 Backend checkpoint in `backend/`: `gofmt -l .` prints nothing, `go vet ./...` clean, `go test ./...` green — including `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005` in `backend/internal/app/app_test.go` for the 47 new operations

**Checkpoint**: the API of feature 002 is complete and contract-checked; later features can call the readers.

---

## Phase 3: Frontend (web app)

### Frontend foundation (T034–T039 in parallel; T040 after T035; T041 → T042 → T043; T044 → T045)

- [ ] T034 [P] Extend `frontend/src/api/errors.ts` with the kinds `inUse` and `cannotRestore` (the API's message shown; keep 003's `inUse` if present) and `frontend/src/lib/useUndoableDelete.ts` to show the `cannot_restore` message; update `frontend/src/api/errors.test.ts` and the `useUndoableDelete` test
- [ ] T035 [P] Write `frontend/src/lib/decimal.ts` (`parseMoneyInput` — `12500`, `12,500.00`, `12.5k`; `parseDecimalInput`, `formatPercent`, `formatRate`, `inverseRate`, `formatWhole`, `parseWholeInput`, `toCents`, `formatCents`; BigInt, never `number` for money) and `frontend/src/lib/text.ts` (`midSentence`, `plural`) with `decimal.test.ts` and `text.test.ts`
- [ ] T036 [P] Write `frontend/src/lib/waterfall.ts` (`waterfall(inq, model)` → `{mql, deals, revenue}` or `'tooLarge'`, BigInt, half-up) with `frontend/src/lib/waterfall.test.ts` using the same table as `waterfall_test.go` (FR-010, SC-006)
- [ ] T037 [P] Write `frontend/src/api/settingsQueries.ts` (`SETTINGS_KEYS`, `useModels`, `useActivityTypes`, `useCurrencies`, `useFinanceBudgets(year)`, `useAttributeForm(level)` with `staleTime: 0` and refetch on mount, `useSettingsOverview`) with `settingsQueries.test.tsx` (keys, refetch of the form on a new mount — SC-008)
- [ ] T038 [P] Write `frontend/src/components/Breadcrumb/`, `SearchField/`, `SegmentedFilter/` and `LockedNote/` (each `.tsx`, `.module.css`, `.test.tsx`; props and behaviour of plan *New shared components*; accessible names)
- [ ] T039 [P] Write `frontend/src/components/CheckboxField/` and `TextAreaField/` (with "n characters left") and add `suggestions?: string[]` (`<datalist>`) to `frontend/src/components/TextField/`, each with tests
- [ ] T040 Write `frontend/src/components/MoneyField/` (currency inside the field, `parseMoneyInput`, format on blur) and `DecimalField/` (`decimals: 0 | 2 | 6`, suffix) with tests
- [ ] T041 Write `frontend/src/components/ChoiceField/` (`<select>` / check-box group up to 10 options; ARIA combobox in a Radix Popover above 10 with search by any part, the 50-row cap message, chips with **Remove <label>**, inactive chosen values "(inactive)", `disabledReason`) with `ChoiceField.test.tsx` (switch at 10, "0220" among 1,217, cap message, inactive chosen value, keyboard) (FR-024, SC-007)
- [ ] T042 Write `frontend/src/components/SplitEditor/` (values through `ChoiceField` multiple, Value · Share table with `DecimalField`, running total line, **Split evenly**, re-split note) with `SplitEditor.test.tsx` (FR-027, R10)
- [ ] T043 Export the new components from `frontend/src/components/index.ts` and add a gallery entry for each to `frontend/src/features/dev/ComponentGalleryPage.tsx`; update `ComponentGalleryPage.test.tsx`
- [X] T044 Write `frontend/src/features/settings/shared/` — `SettingDeleteDialog.tsx` (unused / in use / already inactive states; switches to in use on a `409 in_use`), `ImpactDialog.tsx`, `UsageLine.tsx`, `MetaLine.tsx`, `useListFilters.ts` (`q`, `status` in the URL), `useSettingStatus.ts`, `settingsNouns.ts`, `SettingsTable.module.css` — with tests for the delete dialog states, the impact dialog and the usage line (FR-003, FR-004, SC-004, SC-005)
- [X] T045 Write `frontend/src/features/settings/sections.ts` and `SettingsPage.tsx` (+ `.module.css`) with `SettingsPage.test.tsx` (cards with counts, Coming soon cards, loading and error states — FR-001); in `frontend/src/app/navigation.ts` make Settings a `NavItem` and remove it from `COMING_SOON_SECTIONS`; add `/settings` (`handle: { title: 'Settings' }`) to `frontend/src/app/router.tsx`; update `router.test.tsx` and the navigation/sidebar tests

### Frontend — User Story 1: models (P1)

- [X] T046 [US1] Write `frontend/src/features/settings/models/ModelsPage.test.tsx`, `ModelForm.test.tsx` and `WaterfallExample.test.tsx` covering US1 scenarios 1–7 with `mockApi` (list and filters, details panel, create → toast with **View**, three problems at once, impact dialog before saving, conflict message, Duplicate "(copy)", delete unused vs in use → **Set inactive**, example "250,000 INQ → 5,000 MQL → 250 deals → 2,500,000.00 USD revenue")
- [X] T047 [US1] Write `frontend/src/features/settings/models/modelsApi.ts`, `ModelsPage.tsx`, `ModelDetails.tsx`, `ModelForm.tsx`, `WaterfallExample.tsx` (+ `.module.css`) per plan *Pages* and the save flow of *Shared Settings parts*; add `/settings/models` (`title: 'Models'`) to `frontend/src/app/router.tsx`; T046 passes (FR-002, FR-009 – FR-012)

### Frontend — User Story 2: Shell attributes and options (P1)

- [X] T048 [US2] Write `frontend/src/features/settings/shell-attributes/ShellAttributesPage.test.tsx` (level filter, grouping by section, move buttons with keyboard, focus kept and live-region text, order buttons explained while filtering — SC-009), `ShellAttributeEditorPage.test.tsx` (all problems together, markings by level, yes/no hint, locks with `LockedNote`, impact dialog, dependent option links), `ShellAttributePage.test.tsx` (Settings and Options cards, not-found card) and `OptionsCard.test.tsx` (server search, paging "Showing 100 of 1,217", option actions)
- [X] T049 [US2] Write `frontend/src/features/settings/shell-attributes/shellAttributesApi.ts` and `ShellAttributesPage.tsx` (+ `.module.css`); route `/settings/shell-attributes` (`title: 'Shell attributes'`)
- [X] T050 [US2] Write `ShellAttributeEditorPage.tsx` and `OptionsEditor.tsx` (+ `.module.css`) in `frontend/src/features/settings/shell-attributes/` (create and edit, sticky footer, unsaved-changes guard); routes `/settings/shell-attributes/new` and `/settings/shell-attributes/:attributeId/edit`
- [X] T051 [US2] Write `ShellAttributePage.tsx`, `OptionsCard.tsx` and `OptionPanel.tsx` (+ `.module.css`) in `frontend/src/features/settings/shell-attributes/`; route `/settings/shell-attributes/:attributeId` (title "<label> · Shell attributes"); T048 passes (FR-016 – FR-023)

### Frontend — User Story 3: attribute fields, dependencies and conditions (P1)

- [X] T052 [US3] Write `frontend/src/components/ShellAttributeFields/rules.test.ts` (the cases of `form_test.go`: visibility, allowed options, values to clear, missing, auto-fill), `ShellAttributeFields.test.tsx` (sections and markings, help tip, dependent disabled → filtered → auto-fill message, D-034 note, hidden section keeps values, condition note, no-BU note with plan link, inactive read-only), `ShellAttributeValues.test.tsx`, `ClearValuesDialog.test.tsx`, and `frontend/src/features/settings/shell-attributes/AttributePreview.test.tsx` (business-unit check boxes, live completeness, **Try saving** answers)
- [X] T053 [US3] Write `frontend/src/components/ShellAttributeFields/` (`ShellAttributeFields.tsx`, `rules.ts`, `ShellAttributeValues.tsx`, `ClearValuesDialog.tsx`, `.module.css`) with the props of plan *New shared components*; export them from `frontend/src/components/index.ts` and add gallery entries (FR-024 – FR-027)
- [X] T054 [US3] Write `frontend/src/features/settings/shell-attributes/AttributePreview.tsx` (+ `.module.css`) and add the **Preview — <Level> form** card to `ShellAttributePage.tsx`; T052 passes (FR-020)

### Frontend — User Story 4: currencies (P1)

- [X] T055 [US4] Write `frontend/src/features/settings/currencies/CurrenciesPage.test.tsx` covering US4 scenarios 1–5 (rates both ways live, USD locks, code lock on edit, impact dialog, three problems at once, delete in use)
- [X] T056 [US4] Write `frontend/src/features/settings/currencies/currenciesApi.ts`, `CurrenciesPage.tsx`, `CurrencyDetails.tsx`, `CurrencyForm.tsx` (+ `.module.css`); route `/settings/currencies` (`title: 'Currencies'`); T055 passes (FR-028 – FR-030)

### Frontend — User Story 5: activity types and finance budget items (P2)

- [X] T057 [US5] Write `frontend/src/features/settings/activity-types/ActivityTypesPage.test.tsx` (create with models, default-cost note, impact when models are removed, delete in use) and `frontend/src/features/settings/finance-budget-items/FinanceBudgetItemsPage.test.tsx` (year filter, grouping by budget, budget-name suggestions, year lock, **Rename budget**, delete in use, Undo)
- [X] T058 [US5] Write `frontend/src/features/settings/activity-types/activityTypesApi.ts`, `ActivityTypesPage.tsx`, `ActivityTypeDetails.tsx`, `ActivityTypeForm.tsx` (+ `.module.css`); route `/settings/activity-types` (`title: 'Activity types'`) (FR-013)
- [X] T059 [US5] Write `frontend/src/features/settings/finance-budget-items/financeBudgetsApi.ts`, `FinanceBudgetItemsPage.tsx`, `FinanceBudgetItemDetails.tsx`, `FinanceBudgetItemForm.tsx`, `RenameBudgetDialog.tsx` (+ `.module.css`); route `/settings/finance-budget-items` (`title: 'Finance budget items'`); T057 passes (FR-014, FR-015)

### Frontend — User Story 6: option import (P3)

- [X] T060 [US6] Write `frontend/src/features/settings/shell-attributes/ImportOptionsDialog.test.tsx` (file type and 700 KB checks in the browser, review counts and three lists with row numbers, "and <n> more", **Add <n> options**, nothing to add → **Cancel** only, success toast) and `importTemplate.test.ts` (file name, header with and without "Parent option")
- [X] T061 [US6] Write `ImportOptionsDialog.tsx` (+ `.module.css`) and `importTemplate.ts` in `frontend/src/features/settings/shell-attributes/`; add **Import from a spreadsheet** and **Download template** to `OptionsCard.tsx`; T060 passes (FR-031)
- [X] T062 Frontend checkpoint in `frontend/`: `npm run check` green (API types, type check, lint, format, unit and component tests)

**Checkpoint**: every screen of feature 002 works against mocked answers; 004–007 can use the shared components.

---

## Phase 4: End-to-end (needs Phases 2 and 3)

- [ ] T063 Add to `frontend/e2e/fixtures.ts` `optionsCsv(rows: number, faulty: number): Buffer` (the US6 file: 1,217 cost-centre codes including "CN04000220", 3 faulty rows) for `setInputFiles`
- [ ] T064 [US1] [US2] [US3] Write `frontend/e2e/settings.spec.ts` journeys 1–3 of plan *End-to-end* (Alex; records created with `uniqueName`; demo settings never renamed or deleted)
- [ ] T065 [US4] [US5] [US6] Add journeys 4–6 to `frontend/e2e/settings.spec.ts` (EUR's rate restored to 0.92 at the end of journey 4; axe light and dark on the listed screens; keyboard-only reorder — SC-009)
- [ ] T066 Run `npm run test:e2e` in `frontend/` after `go run ./cmd/seed -reset` in `backend/`: `smoke.spec.ts` and `settings.spec.ts` green

---

## Phase 5: Polish & Cross-Cutting Concerns

- [ ] T067 [P] Update `backend/README.md` (packages `platform/money` and `settings`; reference sources; the reader API of plan *How later features read Settings*) and `frontend/README.md` (`features/settings`, the new shared components, `settingsQueries.ts`, `lib/decimal.ts`, `lib/waterfall.ts`)
- [ ] T068 [P] Update `docs/product/design-system.md` §5 (ChoiceField, SplitEditor, MoneyField, DecimalField, ShellAttributeFields, LockedNote, Breadcrumb, SegmentedFilter) and `docs/architecture/repository-layout.md` (new folders); record the plan's D-041 … D-055 in `docs/sdd/decision-log.md` with the next free numbers, and the one-name-per-component choice of plan *Overlap with feature 003*
- [ ] T069 Final check: `gofmt -l .`, `go vet ./...`, `go test ./...` in `backend/`; `npm run check` and `npm run test:e2e` in `frontend/`; load the seed, run the app and walk US1–US6 with the keyboard only; confirm no personal data in the API log; run `/speckit-analyze` and fix what it reports

---

## Requirement coverage

| FR / SC | Test tasks | Implementation tasks |
|---|---|---|
| FR-001 | T008, T011, T045, T064 | T008, T011, T045 |
| FR-002 | T012, T015, T022, T025, T046, T048, T055, T057, T064 | T013, T016, T023, T026, T027, T038, T044, T047, T049, T056, T058, T059 |
| FR-003 | T006, T009, T012, T015, T022, T025, T044 | T006, T009, T044 |
| FR-004 | T003, T012, T014, T015, T019, T022, T025, T034, T044, T046 | T003, T008, T010, T013, T016, T017, T020, T023, T026, T027, T034, T044 |
| FR-005 | T012, T015, T022, T025, T046 | T008 |
| FR-006 | T010, T011, T012, T015, T019, T022, T025, T029 | T008, T010, T020 |
| FR-007 | T012, T014, T019, T024, T028, T037, T046 | T008, T020, T037, T044 |
| FR-008 | T014 | T011 (no permission checks) |
| FR-009 | T012, T014, T046 | T013, T047 |
| FR-010 | T012, T036, T046, T064 | T013, T036, T047 |
| FR-011 | T012, T014, T046 | T013, T044, T047 |
| FR-012 | T012, T025, T028, T046, T057 | T013, T026 |
| FR-013 | T025, T028, T057 | T026, T058 |
| FR-014 | T025, T028, T057, T065 | T027, T059 |
| FR-015 | T025, T028, T057, T065 | T027, T059 |
| FR-016 – FR-019 | T015, T018, T019, T048, T064 | T016, T017, T049, T050, T051 |
| FR-020 | T015, T018, T021, T048, T052 | T016, T017, T051, T054 |
| FR-021 | T015, T018, T048 | T016, T050 |
| FR-022 | T015, T018, T048 | T016, T017, T050 |
| FR-023 | T015, T018, T019, T052 | T016, T020, T053 |
| FR-024 – FR-027 | T019, T021, T041, T042, T052, T064 | T020, T041, T042, T053 |
| FR-028 | T004, T022, T024, T055, T065 | T005, T023, T056 |
| FR-029 | T004, T019, T022 | T005, T020 |
| FR-030 | T022, T024, T055, T065 | T023, T056 |
| FR-031 | T029, T031, T060, T065 | T030, T061 |
| FR-032 | T032 | T032 |
| SC-003 | T012, T015, T022, T046, T048, T055 | T008, T044 |
| SC-004 | T012, T015, T022, T025, T044 | T008, T044 |
| SC-005 | T012, T015, T022, T046, T048, T055 | T013, T016, T023, T044 |
| SC-006 | T012, T036 | T013, T036 |
| SC-007 | T015, T018, T041, T065 | T017, T041 |
| SC-008 | T037 | T037 |
| SC-009 | T048, T065 | T049 |
| SC-001, SC-002 | usability measures with first-time administrators — not automated; T069 walkthrough | all pages |

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (T001–T002)**: first; blocks everything.
- **Backend (T003–T033)** and **Frontend (T034–T062)**: independent of each other after Setup.
- **End-to-end (T063–T066)**: needs both.
- **Polish (T067–T069)**: last; T067 and T068 may start once their area is done.

### Inside the backend

[T003, T004, T006] → T005 → T007 → T008 → T009 → T010 → T011 → US1 (T012 → T013 → T014) → US2 (T015 → T016 →
T017 → T018) → US3 (T019 → T020 → T021) → US4 (T022 → T023 → T024) → US5 (T025 → T026 → T027 → T028) → US6
(T029 → T030 → T031) → T032 → T033. After T011, US1, US2, US4 and US5 touch different files except
`handler.go` / `handler_test.go` (add routes one story at a time); US3 needs US2's attributes and options;
US6 needs US2's options; T032 needs every service.

### Inside the frontend

[T034, T035, T036, T037, T038, T039] → T040 (after T035) → T041 → T042 → T043 → T044 → T045 → US1 (T046 →
T047) → US2 (T048 → T049 → T050 → T051) → US3 (T052 → T053 → T054) → US4 (T055 → T056) → US5 (T057 → T058 →
T059) → US6 (T060 → T061) → T062. US4 and US5 need only the foundation and may run beside US1–US3.

### Parallel opportunities

- Backend T003 ∥ T004 ∥ T006; frontend T034–T039.
- After the foundations: backend US4 ∥ US5 (separate service files) and frontend US4 ∥ US5 (separate folders).
- Backend and frontend agents work from the contract at the same time.

## Implementation Strategy

1. Setup → backend foundation (money first: 003–012 depend on it) and frontend foundation in parallel.
2. P1 stories US1 → US2 → US3 → US4 on both sides; after US3 the reader API and `ShellAttributeFields` are
   ready for 004–007 — tell those agents.
3. US5 (P2), then US6 (P3); demo data; end-to-end; polish. Commit after each task group:
   `feat(002): <what> (FR-0xx)`.

### Notes

- Usage counts, in-use deletes and impact statements that need plans, activities or line items are proven
  with fake `ReferenceSource`s and mocked answers; 004–007 add the real sources and their e2e steps.
- `Waterfall`, `money` and `rules.ts` have twin implementations in Go and TypeScript: change both with the
  shared test table.
- Never log a request body of the import endpoints (files may hold business codes).
