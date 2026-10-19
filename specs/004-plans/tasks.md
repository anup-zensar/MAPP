---

description: "Task list for feature 004 — Plans"
---

# Tasks: Plans

**Input**: Design documents from `specs/004-plans/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); features 001 (built), 002 and 003 (built before 004).

**Tests**: required — constitution V (a unit test for every rule, a test for every acceptance scenario, an
end-to-end test for every P1 story, a contract test for every endpoint). In every group, write the tests
first and see them fail, then implement. Test names carry the FR they prove.

**Organization**: Setup, then **Backend** and **Frontend** — independent of each other: both start from the
merged contract (T001–T002); the web app's tests use `mockApi()`, the API's tests use `apitest` — then
End-to-end (needs both) and Polish. Inside Backend and Frontend, tasks are grouped by user story (US1–US5
are P1, US6 is P2). Every user-visible text comes from the plan's *User-visible messages* table; every rule
from plan.md and data-model.md (when they seem to differ, plan.md wins and the difference is reported).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 … US6
- Paths are relative to the repository root.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [ ] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [ ] TC01 [P] Create `PillTabs`, `FormSection` and `FormFooter` in `frontend/src/components/` with tests and gallery entries (D-197).
- [ ] TC02 [P] Create `frontend/src/lib/recordLinks.ts` (`recordHref(type, id, planId?)`, plans entry) with a test (D-194).
- [ ] TC03 `CompletenessChip.itemsBelow` accepts a count or a list (D-199).
- [ ] TC04 `References`/`ReferencesFromSnapshot` include the plan name and recycle-bin state (003 `PlanUse`, 002 `settings.Reference`).
- [ ] TC05 Use `platform/date` instead of `ShiftDate` (D-198).

## Phase 1: Setup (shared contract)

- [x] T001 Merge `specs/004-plans/contracts/openapi.yaml` into `backend/api/openapi.yaml`: tag **Plans**, the 8 paths (11 operations), parameter `PlanId`, the 28 schemas; apply the shared-schema change from the fragment's header (`Error.code` += `not_ready`, `Error.missing: ReadinessGroup[]`, `Conflict` response description); confirm that the schemas expected from 002/003 (`Money`, `AttributeValue`, `OptionRef`, `ModelSummary`, `FinanceBudgetRef`, `PersonRef`, `TeamRef`) exist in the merged file — where 002/003 used another name or shape, reference theirs and record the difference in `specs/004-plans/plan.md` (*Expects*) *(Done at consolidation, D-192.)*
- [x] T002 Regenerate `frontend/src/api/schema.d.ts` with `npm run api:types` in `frontend/`; `npm run api:check` passes *(Done at consolidation, D-192.)*
- [ ] T003 [P] Check the backend pieces expected from 002/003 (plan *Expects*: `platform/money` functions incl. a grouped text, the waterfall function forwards **and** backwards, the settings and teams service functions behind `plans.Settings` / `plans.Directory`, a "Used by" hook); list what is missing in `specs/004-plans/plan.md` *Expects* — missing money/waterfall functions are added by T009, missing service functions by T033
- [ ] T004 [P] Check the web pieces expected from 002/003 (Breadcrumb, SearchField, PillFilter, SelectField, MultiChoiceField, TextAreaField, YesNoField, MoneyField, NumberField, AttributeFields, AttributeValueList, PersonPicker, TeamPills, `useModels`, `usePlanAttributes`, `useFinanceBudgets`, `useTeams`, `calculateWaterfall`); every missing one is created by T043

**Checkpoint**: contract merged and types generated — Backend (Phase 2) and Frontend (Phase 3) start in parallel.

---

## Phase 2: Backend (`backend/`)

### Foundation (in order: T005 → T006–T008 [P] → T009 → T010 → T011 → T012)

- [ ] T005 Extend `backend/internal/platform/errs/errs.go` and `errs_test.go`: `CodeNotReady = "not_ready"`, `MissingField{Field, Label}` (JSON `field`, `label`), `ReadinessGroup{Record RecordRef; Fields []MissingField}` (JSON `record`, `fields`), `Error.Missing []ReadinessGroup` (`json:"missing,omitempty"`), `NotReady(message string, missing []ReadinessGroup) *Error` (409); test the JSON shape and that other errors have no `missing`
- [ ] T006 [P] Write `backend/internal/plans/model.go`: `Plan` (stored, plan *Stored type*), `Status`/`GoalStage`/`Spread` constants, and every API type of the plan's table (`PlanView`, `PlanSummary`, `PlanList`, `ListSummary`, `Figures`, `Goals`, `GoalFigures`, `GoalTotals`, `Completeness`, `Contents`, `Defaults`, `PlanInput`, `PlanUpdate`, `MoneyInput`, `ListFilter`, `UnpublishRequest`, `CopyOptions`, `NextYearOption`, `CopyRequest`, `CopyResult`, `ChangeEffectsRequest`, `ChangeEffects`, `References`) with the contract's JSON names; `MissingField = errs.MissingField`
- [ ] T007 [P] Write `backend/internal/plans/deps.go`: `AttributeValues`, the `Settings` and `Directory` interfaces and `Model`, `FinanceBudget`, `Option`, `AttributeGroupChange` (JSON `unit`, `levels`, `attributes`), `Team`, `Person`, `Converter` exactly as plan *Dependencies as interfaces*
- [ ] T008 [P] Write `backend/internal/plans/levels.go` and `levels_test.go`: `Level`, `Facts`, `AttentionItem` (`BlocksPublish` `json:"-"`), `CopyJob`, `GoalTotals` use; `(*Service).RegisterLevel` refusing an empty or duplicate name and a missing `AddFacts` or `Copy`
- [ ] T009 Write `backend/internal/plans/goals_test.go` (`TestWaterfall_FromINQ_FR009`, `TestWaterfall_FromMQL_FR009`, `TestWaterfall_FromRevenue_FR009`, `TestWaterfall_RoundsHalfUp_FR009` incl. 2.5 → 3 and 2.4999 → 2, `TestWaterfall_ZeroRateGivesReason_EdgeCase` for each of r1, r2, deal size, `TestPlannedShare_FR016` 150,000 of 250,000 → 60, null goal → null), then make them pass: use 002's shared waterfall function; if it only goes forwards, add the backward stages **in 002's file** with these vectors (plan *Calculations*); write `backend/internal/plans/goals.go` (`planGoals`, `plannedShare`, reasons texts)
- [ ] T010 Write `backend/internal/plans/repository.go` (`Repository`) and `store_json.go` (`NewJSONRepository`, table `plans`, `store.Get/List/Put/Delete`)
- [ ] T011 Write `backend/internal/plans/fakes_test.go`: fakes of `Settings` and `Directory` (models Lubricants 1.0 2.00/5.00/10,000.00 and Mobility 1.0 3.00/4.00/5,000.00 plus an inactive one; teams Lubricants Europe [Priya, Daniel, Sara], Mobility China [Mei, Carlos, Sara], Global Marketing Ops [Alex, Carlos], one inactive team; people incl. one inactive; Business unit options; display converter USD or EUR 0.92), a fake `Level` with configurable facts, copy behaviour and call counters, and a fixture: `store.Open(t.TempDir())`, `ids` generator, `history.Service` with the plan kind registered, fixed clock 2026-03-12 09:00 UTC, actors Priya and Daniel
- [ ] T012 Write `backend/internal/plans/figures_test.go` (`TestFigures_RemainingBudgetNegative_FR012` 1,200,000 − 1,300,000 → −100,000.00 and over-allocated 100,000.00, `TestFigures_BudgetUsed_FR012` 960,000 of 1,200,000 → 80, 1,300,000 → 108 and over budget, no budget and budget 0 → null, `TestFigures_ApprovalProgress_FR012`, `TestFigures_DisplayCurrency_FR015` EUR 0.92 half up, `TestCompleteness_OwnFieldsAndItemsBelow_FR017` order Teams, Owner, attributes, ineligible owner counts, items below), then `backend/internal/plans/figures.go`

### User Story 1 — Create a plan (backend)

- [ ] T013 [US1] Write in `backend/internal/plans/service_test.go`: `TestDefaults_FR006`, `TestCreate_OnlyNameYearModelRequired_FR010`, `TestCreate_AllProblemsAtOnce_FR010` (empty name, no model, taken name in another case → three problems in one error), `TestCreate_NameRules_FR010`, `TestCreate_DescriptionBudgetGoalChecks_FR010`, `TestCreate_YearRange_FR006` (2025–2031 accepted, 2024 and 2032 refused), `TestCreate_ActiveModelAndYearFinanceBudget_FR007`, `TestCreate_OwnerNeedsTeams_FR007`, `TestCreate_OwnerMustBeInChosenTeam_FR021`, `TestCreate_AttributeProblemsCollected_FR008`, `TestCreate_DraftWithIDAndLog_FR010_FR031` (P-1001, Draft, one `created` entry), `TestCreate_SameNameTwice_EdgeCase`, `TestCreate_DeletedNameReusable_EdgeCase`
- [ ] T014 [US1] Write `backend/internal/plans/validate.go` (every check of plan *Validation*, all problems collected, messages from the plan's table) and in `backend/internal/plans/service.go`: `Service`, `NewService`, `Defaults`, `Create`, and the view assembly used by every answer (`view(ctx, a, plan, facts, converter)`: references, `ownerProblem`, `businessUnit`, figures, goals, completeness, contents; facts from every registered level in one `AddFacts` call)
- [ ] T015 [US1] Write `backend/internal/plans/handler.go` (`NewHandler`, `Routes` with `GET /api/plan-defaults`, `POST /api/plans`, `GET /api/plans/{planId}`) and in `backend/internal/plans/handler_test.go` (package `plans_test`): `TestPlanDefaults_FR006` (as Priya: year 2026, years 2025–2031, her team, owner Priya), `TestCreatePlan_201AndValidation_FR010` (201 Draft with figures; empty body values → 400 with every field), `TestPlansOperations_WithoutSession_FR005` (extended by later tasks to every operation)

### User Story 2 — Find and compare plans (backend)

- [ ] T016 [US2] Write in `service_test.go`: `TestList_FiltersSortAndSummary_FR001_FR003` (order year desc then name; status, year; total = scope; summary over scope + year; years newest first), `TestList_MyTeams_FR004`, `TestList_SearchNameOrID_FR003` ("MOBILITY", "p-1002", " 1002 "), `TestList_200PlansOneFactsCallPerLevel_SC009`
- [ ] T017 [US2] Implement `Service.List` in `backend/internal/plans/service.go`; add `GET /api/plans` to `handler.go` (query parsing; invalid `scope`, `status`, `year` → field problems); `TestListPlans_SeedOrderSummaryAndFilters_FR001_FR003` in `handler_test.go` (seed: 3 plans, 2 Draft, 1 Published; `status=draft&year=2026` → FY2026 Mobility China, total 3; `q=mobility`; `q=p-1002`; `status=open` → 400, sent with `apitest.SkipRequestValidation()` because the contract's enum refuses it first)

### User Story 3 — Understand a plan at a glance (backend)

- [ ] T018 [US3] Write `TestGetPlan_ViewAnd404_FR011` in `handler_test.go` (FY2026 Lubricants Europe: every stored field, model with rates, teams, owner, business unit, figures with no campaigns: remaining = budget, planned cost null, budget used 0, approvals 0 of 0, goals 250,000 → 5,000 → 250 → 2,500,000.00, completeness all filled in; FY2026 Mobility China: missing Business unit; unknown ID → 404 with the message) and in `service_test.go` `TestGet_FactsFromLevels_FR012_FR016_FR017` (fake level facts → figures, goals share, items below); make them pass in `service.go`

### User Story 4 — Edit a plan (backend)

- [ ] T019 [US4] Write in `service_test.go`: `TestUpdate_MissingVersion_FR023`, `TestUpdate_VersionConflict_FR023` (Daniel saved first → 409 naming Daniel and the time; nothing saved), `TestUpdate_NoChangeNoLog_FR031`, `TestUpdate_PublishedStaysPublishedAndLogsFields_FR022` (budget, model, teams and one attribute → one `changed` entry with readable old/new texts; status still Published), `TestUpdate_OwnerNotInTeams_FR021`, `TestUpdate_InactiveValuesKept_EdgeCase`, `TestUpdate_HiddenAttributeValuesKept_FR020`, `TestChangeEffects_DisallowedActivities_FR019` (fake level returns 2 activities), `TestChangeEffects_AttributeGroups_FR020`
- [ ] T020 [US4] Implement `Service.Update` (version, checks, R1, diff of the record without `attributes` + one change per changed attribute, no-change rule) and `Service.ChangeEffects` in `backend/internal/plans/service.go`; add `PATCH /api/plans/{planId}` and `POST /api/plans/{planId}/change-effects` to `handler.go`; `TestUpdatePlan_200_409_404_FR018_FR023` and `TestChangeEffects_FR019_FR020` in `handler_test.go`

### User Story 5 — Publish and unpublish (backend)

- [ ] T021 [US5] Write `backend/internal/plans/status_test.go`: `TestPublish_CompleteDraft_FR024` (Published, `statusChanged` "published"), `TestPublish_ListsMissingGrouped_FR024` (no business unit + fake campaign "Retail growth" missing Cluster and Region → `not_ready` "Publish needs 3 fields", groups Plan then Campaign 'Retail growth'; plan still Draft, no log), `TestPublish_WarningsDoNotBlock_FR024` (negative remaining budget, incomplete activities from the fake level with `BlocksPublish` false), `TestPublish_AlreadyPublished_NoChange`, `TestUnpublish_ReasonRules_FR025` ("" and 501 characters), `TestUnpublish_ReturnsToDraftAndLogsReason_FR025`
- [ ] T022 [US5] Write `backend/internal/plans/status.go` (`Publish`, `Unpublish`, readiness = own missing + blocking items); add `POST …/publish` and `POST …/unpublish` to `handler.go`; `TestPublish_NotReadyThenPublished_FR024` (FY2026 Mobility China → 409 `not_ready` with group Plan › Business unit; after an update adding Lubricants → 200 Published) and `TestUnpublish_FR025` in `handler_test.go`

### User Story 6 — Copy or delete a plan (backend)

- [ ] T023 [P] [US6] Write `backend/internal/plans/copy_test.go`: `TestCopyName_Suffixes_FR026` ("(copy)", then "(copy 2)" when taken in any case), `TestCopyName_ShortenedTo120_EdgeCase`, `TestCopyOptions_NextYearAvailability_FR028` (2026 plan → 2027 available; 2031 plan with current year 2026 → 2032 unavailable with reason), `TestCopy_PlanOnly_FR027` (all fields, Draft, new ID, creator = copier, one `copied` entry with details "from '…' (P-…)"), `TestCopy_WithContents_FR027` (fake levels called in order with `NewIDs` holding the plan, `Years` 0 or 1, counts in details), `TestCopy_AllOrNothing_FR027` (a level fails → no plan, no counter-visible record, no log), `TestCopy_NextYear_FR028` (2027, finance budget cleared, note), `TestShiftDate_FR028` (2026-03-12 → 2027-03-12, 2028-02-29 → 2029-02-28, "" → "")
- [ ] T024 [US6] Write `backend/internal/plans/copy.go` (`copyName`, `CopyOptions`, `Copy`, exported `ShiftDate`); add `GET …/copy-options` and `POST …/copy` to `handler.go`; `TestCopyOptions_FR026` and `TestCopy_201_FR027_FR028` in `handler_test.go` (copy FY2026 Lubricants Europe into 2027 → 201, Draft, year 2027, no default finance budget, note)
- [ ] T025 [P] [US6] Write `backend/internal/plans/kinds_test.go`: `TestDelete_MovesToBin_FR029` (one bin entry, `deleted` entry, plan gone; with a fake child kind registered under `plan`, the child goes too and the details count it), `TestUndo_RestoresExactly_FR030` (same ID, version, status Published), `TestUndo_NameTaken_EdgeCase`, `TestRestore_RenamesAndReturnsDraft_R14` (Published → Draft, " (restored)", notes), `TestDescribe_ReadableLog_FR031`
- [ ] T026 [US6] Write `backend/internal/plans/kinds.go` (`Kinds()`: snapshot, remove, put, describe, prepareRestore) and `Service.Delete`; add `DELETE /api/plans/{planId}` to `handler.go`; `TestDeletePlan_AndUndo_FR029_FR030` in `handler_test.go` (delete → `DeleteResult` with `recordTypeLabel` "plan"; `POST /api/recycle-bin/{id}/undo` → plan back with the same ID)

### Cross-cutting (backend)

- [ ] T027 Write in `service_test.go` `TestEveryChangeLogged_FR031` (create, edit, publish, unpublish, copy, delete → one entry each with the actor) and `TestNoRoleChecks_FR032` (Daniel edits, publishes, unpublishes, copies and deletes Priya's plan; no refusal)
- [ ] T028 Implement `Service.Lookup`, `Service.References` and `ReferencesFromSnapshot` in `backend/internal/plans/service.go` with `TestLookupAndReferences` in `service_test.go` (inside a caller's Write; a snapshot decodes to the same references)
- [ ] T029 Extend `TestPlansOperations_WithoutSession_FR005` in `handler_test.go` to all 11 operations (401 `sign_in_required`)

### Wiring and seed

- [ ] T030 Write `backend/internal/app/plans_deps.go` (`settingsForPlans`, `directoryForPlans` implementing `plans.Settings` / `plans.Directory` over the 002 and 003 services; display currency = the person's preference with 002's rate, USD when none) and `backend/internal/app/plans_deps_test.go` (with the seed: Lubricants 1.0 rates and deal size, Priya's teams, Business unit options, USD converter)
- [ ] T031 Wire `plans` in `backend/internal/app/app.go` (repository, service, `Kinds()` registered with history before any 005 kind, routes); `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005` in `app_test.go` pass
- [ ] T032 Add `addPlans` to `backend/seed/seed.go` (the three plans of plan *Seed*, IDs P-1001–P-1003, after the 002/003 seed, returning name → ID) and `TestSeed_Plans` to `backend/seed/seed_test.go`; update the seed message of `cmd/seed` if 002/003 changed it to list records ("… 3 plans")
- [ ] T033 If T003 found service functions missing in 002/003 that the adapters need, add them in those packages with tests (no behaviour change for 002/003)
- [ ] T034 Backend checks from `backend/`: `gofmt -l .` prints nothing; `go vet ./...` and `go test ./...` pass

**Checkpoint**: the API serves every operation of the contract with the seed.

---

## Phase 3: Frontend (`frontend/`)

### Foundation (T035–T041 and T043 in parallel; T042 after T035)

- [ ] T035 [P] Extend `frontend/src/api/errors.ts` and `errors.test.ts`: code `not_ready` → kind `notReady`; `ApiError.missing` (the `missing` groups, `[]` otherwise)
- [ ] T036 [P] Write `frontend/src/components/TreeTable/TreeTable.tsx`, `TreeTable.module.css`, `filterTree.ts`, `TreeTable.test.tsx`, `filterTree.test.ts` per plan *Shared components* (treegrid ARIA, chevron buttons "Expand <name>" / "Collapse <name>", keys ↑ ↓ → ← Home End Enter, level bar + LevelTag, `footer`, `emptyText`; `filterTree` keeps ancestors and returns the ids to open)
- [ ] T037 [P] Write `frontend/src/components/ProgressBar/` and `frontend/src/components/LevelTag/` (component, CSS Module on tokens, test: `role="progressbar"`, `aria-valuenow`, width capped at 100 %, red tone; five level words and colours)
- [ ] T038 [P] Write `frontend/src/components/FormDialog/` and `frontend/src/components/FilterChips/` (component, CSS, test: first field focused, Enter submits, busy ignores a second submit, `hideCancel`, focus returns to the opener; chips "Remove filter: <label>", Clear all)
- [ ] T039 [P] Extend `frontend/src/components/SummaryCard/` (`detail?: ReactNode` under the value) and `frontend/src/components/CompletenessChip/` (`itemsBelow?: {id, label, href}[]` → second chip "1 item below needs attention" / "<n> items below need attention" with a popover of links) with tests; export the new components in `frontend/src/components/index.ts` (+ `index.test.ts`)
- [ ] T040 [P] Write `frontend/src/features/plans/planText.ts` + `planText.test.ts` (status word and tone, spread and stage labels, approval progress words, counts joined "2 campaigns, 4 initiatives, 7 activities and 15 line items", "1 Jan – 31 Dec <year>")
- [ ] T041 [P] Write `frontend/src/features/plans/recordLinks.ts` + test (`RECORD_LINKS` with `plan`: record `/plans/<id>`, field `/plans/<id>/edit?focus=<field>`; group label "Plan" / "<Type> '<name>'") and `frontend/src/features/plans/listPreferences.ts` + test (key per person, try/catch, invalid values → defaults, address wins)
- [ ] T042 Write `frontend/src/features/plans/plansApi.ts` (query keys, hooks and mutations of plan *Data*; `fetchLatestPlan`)
- [ ] T043 For each web piece T004 found missing, create it in `frontend/src/components/<Name>/` (or `src/lib/waterfall.ts` for `calculateWaterfall`, with the Go vectors of T009) with a test, following design system §5 and the props in plan *Expects*

### User Story 1 — Create a plan (frontend)

- [ ] T044 [US1] Write `frontend/src/features/plans/PlanForm.test.tsx` and `GoalFields.test.tsx`: sections and field order, markings as words, default hints (US1#1, FR-005, FR-006), owner disabled with the reason and R1 message with the person's name (US1#4, FR-007, FR-021), INQ 250,000 → MQL 5,000 and Revenue 2,500,000.00 USD labelled Calculated with formulas, backwards from Revenue, "Choose a model to calculate", 0 % reason (US1#5, FR-009), SMP shown with "Shown because the plan's business unit includes Mobility" (FR-008), model hint and inactive model label (FR-007), finance budget cleared on a year change with the note (FR-007), API problems shown at once with the summary focused (US1#3, FR-010)
- [ ] T045 [US1] Write `frontend/src/features/plans/PlanForm.tsx`, `PlanForm.module.css`, `GoalFields.tsx` per plan *Plan form* (one column, sections, sticky footer, edit-form rule, `?focus=` support)
- [ ] T046 [US1] Write `frontend/src/features/plans/PlanCreatePage.test.tsx` (defaults from `GET /api/plan-defaults`, save → `/plans/P-1004` and toast "Plan '<name>' created — add campaigns next." (US1#2), unsaved-changes guard, defaults failing → ErrorState) and `PlanCreatePage.tsx`

### User Story 2 — Find and compare plans (frontend)

- [ ] T047 [US2] Write `frontend/src/features/plans/PlansPage.test.tsx`, `PlanCard.test.tsx`, `PlanTable.test.tsx`: three seed-like plans newest year first with summary 3 / 2 / 1 and money totals (US2#1, FR-001), every card fact incl. bar red above 100 %, "No budget set", "n of m activities approved", completeness link to the edit field (FR-002), Draft + 2026 → one card, chips, "1 of 3" (US2#2, FR-003), search sends `q` (US2#3), My teams hint (US2#4, FR-004), both empty states and Clear filters (US2#5), loading skeleton, ErrorState, Table layout, filters from the address and remembered per person, actions menu items
- [ ] T048 [US2] Write `frontend/src/features/plans/PlansPage.tsx`, `PlansPage.module.css`, `PlanCard.tsx`, `PlanCard.module.css`, `PlanTable.tsx`, `PlanTable.module.css` per plan *Plans list*

### User Story 3 — Understand a plan at a glance (frontend)

- [ ] T049 [US3] Write `frontend/src/features/plans/PlanPage.test.tsx` and `tree/PlanTree.test.tsx`: header facts, chips and actions; Publish on Draft, Unpublish on Published (US3#1, FR-011); remaining "−100,000.00 USD" red with "Over-allocated by 100,000.00 USD" (US3#2); budget used 80 % and red above 100 %, no budget "—" with "No plan budget set" (US3#3, FR-012); "1 field needed" linking to the business unit field (US3#4, FR-017) and items below; Goals with "60 % of the INQ goal planned" (US3#5, FR-016); Details "—", Calculated, (inactive), amounts label (FR-015); plan tree row cells, empty text, search keeps ancestors / no-match text, Expand all / Collapse all (FR-013, FR-014); View history link (FR-031); loading, error, not found
- [ ] T050 [US3] Write `frontend/src/features/plans/PlanPage.tsx` (+ `.module.css`; layout route with `<Outlet/>`), `PlanHeader.tsx`, `PlanSummaryCards.tsx`, `PlanDetails.tsx`, `PlanGoals.tsx` (+ CSS Modules), `tree/PlanTree.tsx` (+ CSS; props `toolbarActions`, `footer`) and `tree/usePlanTreeNodes.ts` (plan node only, `children: []`) per plan *Plan page*

### User Story 4 — Edit a plan (frontend)

- [ ] T051 [US4] Write `frontend/src/features/plans/PlanEditPage.test.tsx` and `dialogs/ChangeEffectsDialog.test.tsx`: year lock text (US4#1, FR-018); model change → change-effects → dialog with 2 activity links and the sentence, **Change model** / **Keep editing** (US4#2, FR-019); business unit removed → hidden-group sentence (US4#3, FR-020); owner R1 message until fixed (US4#4, FR-021); published banner and "Applies to line items created from now on." (US4#5, FR-022); `version_conflict` → conflict message, values kept, next save sends the new version (FR-023); `?focus=` focuses the field; not found; unsaved-changes guard
- [ ] T052 [US4] Write `frontend/src/features/plans/PlanEditPage.tsx` and `dialogs/ChangeEffectsDialog.tsx`

### User Story 5 — Publish and unpublish (frontend)

- [ ] T053 [US5] Write `frontend/src/features/plans/dialogs/PublishBlockedDialog.test.tsx`, `UnpublishDialog.test.tsx` and `usePlanActions.test.tsx`: publish success toast "Plan '<name>' published" (US5#1); `not_ready` → "Publish needs 3 fields" with groups Plan and "Campaign 'Retail growth'" and working links (US5#2, FR-024); unpublish asks "Why are you unpublishing this plan?", empty and 501-character reasons show the API messages, success toast (US5#3, FR-025)
- [ ] T054 [US5] Write `frontend/src/features/plans/dialogs/PublishBlockedDialog.tsx`, `UnpublishDialog.tsx` and `usePlanActions.ts` (publish, unpublish); use them in `PlanHeader.tsx` and the list's actions

### User Story 6 — Copy or delete a plan (frontend)

- [ ] T055 [US6] Write `frontend/src/features/plans/dialogs/CopyPlanDialog.test.tsx` and `DeletePlanDialog.test.tsx`: copy options name "(copy)", Same / Next year, next year unavailable with its reason, include checkbox with counts, "This plan has nothing inside yet.", next-year note, success → new plan page + toasts incl. notes (US6#1–2, FR-026–FR-028); delete dialog sentences for a published plan with 2 campaigns, 4 initiatives, 7 activities (4 approved) and 15 line items, Cancel focused, **Delete plan** red (US6#3, FR-029); after delete → `/plans` and Undo toast; Undo restores (US6#4, FR-030)
- [ ] T056 [US6] Write `frontend/src/features/plans/dialogs/CopyPlanDialog.tsx` and `DeletePlanDialog.tsx`; add copy and delete to `usePlanActions.ts` (delete uses `useUndoableDelete({ invalidate: [['plans']] })`)

### Navigation and checks

- [ ] T057 Update `frontend/src/app/navigation.ts` (Plans is a `NavItem`, out of `COMING_SOON_SECTIONS`) and `frontend/src/app/router.tsx` (the four routes with `handle.title`); update `navigation.test.ts` and `router.test.tsx` (Plans page instead of Coming soon; `/plans/new`, `/plans/P-1001`, `/plans/P-1001/edit`; Plans highlighted on nested addresses)
- [ ] T058 Web checks from `frontend/`: `npm run check` and `npm run build` pass

**Checkpoint**: every page works against `mockApi`.

---

## Phase 4: End-to-end (needs Phases 2 and 3)

- [ ] T059 Write `frontend/e2e/plans.spec.ts` US1 (as Priya: New plan → unique name → Lubricants 1.0 → business unit Lubricants → Save in ≤ 6 steps (SC-001) → Draft, "All filled in", toast), US2 (Draft + 2026 → FY2026 Mobility China, chips, count; search "mobility" and "P-1001"; My teams as Mei Lin with the hint; Clear filters) and US3 (FY2026 Lubricants Europe: header, five cards, tree row, Details, Goals match the seed)
- [ ] T060 Add to `frontend/e2e/plans.spec.ts` US4 (edit an own plan: year lock text, budget, teams with the R1 message, save; a published own plan stays Published) and US5 (FY2026 Mobility China → "Publish needs 1 field" without change; own plan without business unit → link → fill → publish → toast → unpublish with a reason → Draft)
- [ ] T061 Add to `frontend/e2e/plans.spec.ts` US6 (copy FY2026 Lubricants Europe into 2027 with a unique name → Draft 2027 page; delete it → dialog → list + toast → Undo → back with the same ID) and axe (WCAG 2.1 AA) on the list (cards, table), the create form with its error summary, the plan page and the copy, unpublish and delete dialogs
- [ ] T062 Run `npm run test:e2e` twice in a row from `frontend/` (smoke and plans files pass)

---

## Phase 5: Polish

- [ ] T063 [P] Documentation: `backend/README.md` (package `plans` in *Packages*; seed now loads plans), `frontend/README.md` (`features/plans` in *Folders*; the new shared components), `docs/sdd/decision-log.md` (the proposed decisions of plan.md with the numbers given by the consolidation step), `docs/architecture/repository-layout.md` if it lists packages
- [ ] T064 Walk through the demo with the seed (Plans list, create, plan page, edit, publish, unpublish, copy, delete, Undo) and compare every screen with `docs/product/design-system.md` and `docs/reference/zenreach-screenshots/`; record anything that does not match as a new task here
- [ ] T065 Run the SC-002 moderated session (at least 5 first-time users) and record the results, with SC-004 and SC-005, in `specs/004-plans/checklists/usability.md`
- [ ] T066 Set feature 004 to Done in `docs/sdd/feature-map.md` once the product owner accepts the demo

---

## Requirement coverage (every FR has at least one test task)

| FR / SC | Tasks |
|---|---|
| FR-001 | T016, T017, T047, T048, T059 |
| FR-002 | T047, T048 |
| FR-003 | T016, T017, T041, T047, T048, T059 |
| FR-004 | T016, T017, T047, T059 |
| FR-005 | T044, T045, T059 |
| FR-006 | T013, T014, T015, T044, T046 |
| FR-007 | T013, T014, T044, T045 |
| FR-008 | T013, T030, T044, T045 |
| FR-009 | T009, T043, T044, T045 |
| FR-010 | T013, T014, T015, T044, T046, T059 |
| FR-011 | T018, T049, T050, T059 |
| FR-012 | T012, T018, T049, T050 |
| FR-013, FR-014 | T036, T049, T050, T059 |
| FR-015 | T012, T049, T050 |
| FR-016 | T009, T018, T049, T050 |
| FR-017 | T012, T018, T039, T049, T050 |
| FR-018 | T019, T020, T051, T052, T060 |
| FR-019 | T019, T020, T051, T052 |
| FR-020 | T019, T020, T030, T051, T052 |
| FR-021 | T013, T019, T044, T051, T060 |
| FR-022 | T019, T020, T051, T060 |
| FR-023 | T019, T020, T051 |
| FR-024 | T005, T021, T022, T035, T053, T054, T060 |
| FR-025 | T021, T022, T053, T054, T060 |
| FR-026 | T023, T024, T055, T056, T061 |
| FR-027 | T023 (`TestCopy_PlanOnly_FR027`, `TestCopy_WithContents_FR027`, `TestCopy_AllOrNothing_FR027`), T024, T055, T061 |
| FR-028 | T023, T024, T055, T061 |
| FR-029, FR-030 | T025, T026, T055, T056, T061 |
| FR-031 | T013, T019, T025, T027, T049 |
| FR-032 | T027 |
| SC-001 | T059 |
| SC-002, SC-004, SC-005 | T065 (and T059 for SC-004: find by name or ID in 2 steps) |
| SC-003 | T013, T044, T053, T061 |
| SC-006 | T009, T012 |
| SC-007 | T023 |
| SC-008 | T025, T061 |
| SC-009 | T016 |

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup**: T001 → T002; T003 and T004 any time after T001.
- **Backend (Phase 2)**: T005 → T006–T008 (parallel) → T009 → T010 → T011 → T012; then US1 (T013 → T014 →
  T015) → US2 (T016 → T017) → US3 (T018) → US4 (T019 → T020) → US5 (T021 → T022) → US6 (T023 and T025 in
  parallel; T024 after T023; T026 after T025) → T027–T029 → T030 → T031 → T032 (T033 whenever T003 found
  gaps, before T030) → T034. `handler.go` and `service.go` are touched by several stories, so their tasks run
  in story order.
- **Frontend (Phase 3)**: T035–T041 in parallel, T043 alongside them; T042 after T035; then US1 (T044 →
  T045 → T046), US2 (T047 → T048), US3 (T049 → T050), US4 (T051 → T052, after T045), US5 (T053 → T054, after
  T050), US6 (T055 → T056, after T054); T057 after the pages; T058 last.
- **End-to-end (Phase 4)**: after T034 and T058; T059 → T060 → T061 → T062.
- **Polish (Phase 5)**: after Phase 4.

### Parallel opportunities

- Backend and Frontend phases run at the same time from the merged contract (T001–T002).
- Inside the backend foundation: T006, T007, T008. Inside US6: T023 and T025.
- Inside the frontend foundation: T035–T041 and T043 (different folders).

### Parallel example

```text
Backend:  T005 → (T006 | T007 | T008) → T009 → T010 → T011 → T012 → US1 … US6 → T027 … T034
Web app:  (T035 | T036 | T037 | T038 | T039 | T040 | T041 | T043) → T042 → US1 … US6 → T057 → T058
Both:     T059 → T060 → T061 → T062
```

---

## Implementation Strategy

### MVP first (User Stories 1–3)

1. Setup, then the backend foundation and US1–US3 with the web foundation and US1–US3.
2. Stop and validate: create a plan, find it in the list, open its page with correct figures.

### Incremental delivery

1. + US4 (edit) → + US5 (publish) → + US6 (copy, delete).
2. After each story: its tests and its part of `plans.spec.ts`; commit with the feature number and FRs,
   e.g. `feat(004): publish and unpublish plans (FR-024, FR-025)`.

### Notes

- Never edit `frontend/src/api/schema.d.ts` by hand: change `backend/api/openapi.yaml`, then
  `npm run api:types`.
- Keep the user-visible messages exactly as in the plan's table; tests quote them.
- Features 005–007 extend this feature only through `plans.Level`, history kinds, `usePlanTreeNodes`,
  `PlanTree`'s `toolbarActions`/`footer`, child routes of `/plans/:planId` and `RECORD_LINKS`.
- Mark each task `[X]` here when it is done.
