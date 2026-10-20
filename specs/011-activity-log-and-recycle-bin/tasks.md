---

description: "Task list for feature 011 — Activity Log and Recycle Bin"
---

# Tasks: Activity Log and Recycle Bin

**Input**: Design documents from `specs/011-activity-log-and-recycle-bin/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); features 001 and 004–008 built (kinds `plan`,
`campaign`, `initiative`, `activity`, `lineItem` registered; record pages exist)

**Tests**: required — constitution V. Within each group write the tests first and see them fail, then
implement. Test names carry the FR they prove.

**Organization**: Setup, then **Backend** and **Frontend** phases that both start from the merged
contract and do not depend on each other, then End-to-end (needs both), then Polish. Stories: US1 find and
restore (P1), US2 activity log (P1), US3 record history (P1), US4 restore with parent (P2), US5 delete
permanently and empty (P2). Every text comes from the plan's *User-visible messages* table.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 … US5
- Paths are relative to the repository root.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- Merged into the current Plan module on 2026-10-20 (D-222, branch lane-011b): the backend and both pages of lane-011 with
  the T005 preference, T021's "deleted it permanently" wording, Home's Recent changes (009 T013) and the end of the
  "Coming soon" sections. T029 was built as `LogFilterBar.tsx`.
- Completed on 2026-10-20 on the same branch (D-316): T019 (tests in `backend/seed/history_test.go`), T032 – T035
  (**View history** now brings the page's own History section into view; the side panels use `history-<recordId>`),
  T045 – T046, the cross-feature journeys of `frontend/e2e/activity-log-and-recycle-bin.spec.ts` (delete from the
  page, find and restore, delete permanently, the log and History with who and when, Home's Recent changes), T049
  (every end-to-end file twice green; the store now decodes each table once per Read, which ended the time-outs
  of Home, the Plans list and the restore dialog) and TC99 (`checklists/success-criteria.md`). Nothing of 011 is
  open.
- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [x] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [x] TC01 Contract extensions of 001 schemas are optional (D-191): T003 updates only the fixtures its tests need.
- [x] TC99 Check the success criteria SC-003, SC-004, SC-007, which no task above names, in the final review (timed end-to-end run or moderated session, as each criterion says) and record the result in `checklists/`.

## Phase 1: Setup

- [x] T001 Merge `specs/011-activity-log-and-recycle-bin/contracts/openapi.yaml` into `backend/api/openapi.yaml` following the fragment's merge rules: replace the operations `listActivityLog`, `listRecycleBin`, `restoreFromRecycleBin` and the schemas `ActivityLogEntry`, `ActivityLogPage`, `RecycleBinPage`; extend `Error.code`, `Resolution.how`, `Preferences`, `PreferencesUpdate` (keep properties other features added); add the paths `/api/activity-log/summary`, `/api/recycle-bin/empty`, `/api/recycle-bin/{entryId}` (DELETE), `/api/recycle-bin/{entryId}/restore-preview` and the new schemas *(Done at consolidation, D-192.)*
- [x] T002 Regenerate `frontend/src/api/schema.d.ts` with `npm run api:types` (in `frontend/`); `npm run api:check` passes *(Done at consolidation, D-192.)*
- [x] T003 [P] Update test fixtures that the contract change makes incomplete: entries typed `ActivityLogEntry` get `items: []` and `recordNow`, bin entries in `RecycleBinPage` get the `RecycleBinListItem` fields, and every `preferences` fixture gets `historyIncludeInside: false` — in `frontend/src/features/dev/SampleRecordsPage.test.tsx`, `frontend/src/test/` helpers and any other file found with `grep -rn "sidebarCollapsed:\|recycleBinEntry\|'/api/activity-log'" frontend/src`

**Checkpoint**: the contract is final; backend (Phase 2) and frontend (Phase 3) can start in parallel.

---

## Phase 2: Backend

### Foundation (blocks the story groups below)

- [x] T004 [P] Add `CodeParentDeletedPermanently = "parent_deleted_permanently"` and `ParentDeletedPermanently(level string, parent RecordRef, parentLevel string) *Error` (409, sets `Parent`, message "This <level> can't be restored because its <parent level> '<name>' was permanently deleted.") to `backend/internal/platform/errs/errs.go`; test `TestParentDeletedPermanently` in `backend/internal/platform/errs/errs_test.go`
- [x] T005 [P] [US3] Add the preference `historyIncludeInside` (stored `Preferences`, `PreferencesView`, `PreferencesUpdate.HistoryIncludeInside *bool`, default `false`, saved only when changed, not logged) in `backend/internal/people/model.go` and `backend/internal/people/service.go`; test `TestUpdatePreferences_HistoryIncludeInside_FR012` in `backend/internal/people/service_test.go`
- [x] T006 Add to `backend/internal/history/model.go` the types of plan *Types*: `ActionDeletedPermanently`, `ActionEmptiedRecycleBin`, `Entry.Items` (`json:"items"`), `LogItem`, `RecordState`, `RecordNow`, `EntryView`, `LogQuery`, the new `LogPage` fields (`Items []EntryView`, `UnfilteredTotal`, `Record *RecordRef`), `LogSummary`, `RecordTypeCount`, `LabelledRef`, `BinRecord`, `BinListItem`, `RecordChange`, `RestorePreview`, `PurgeResult`, `EmptyResult`, restore-state constants
- [x] T007 Refactor `backend/internal/history/service.go` (depends on T004, T006): `appendEntry` (id, seq, at, actor, `[]` for `Changes`/`Items`, the dry-run **log collector** read from the context); `Log` calls it; `putBack` split into `restoreEntry(ctx, a, entry, undo, now) ([]restoredPair, error)` returning every snapshot's before/after; `checkCanComeBack`'s missing-parent branch answers `RestoreParentFirst` when an unresolved bin entry holds the parent and `ParentDeletedPermanently` otherwise; `verbOf` accepts the two new actions. All existing tests in `backend/internal/history/service_test.go` stay green; add `TestRestore_ParentInBin_RestoreParentFirst_FR023` there

### US2 — Activity log (backend)

- [x] T008 [US2] Write `backend/internal/history/query_test.go` (fixed clock, fake kinds `fakePlan` › `fakeItem` › `fakeLeaf`): `TestQueryLog_FilterPerson_FR003`, `TestQueryLog_FilterRecordType_FR003`, `TestQueryLog_FilterAction_FR003` (deletedPermanently matches emptiedRecycleBin), `TestQueryLog_FilterDateRange_FR003`, `TestQueryLog_Search_NameIDPerson_FR003`, `TestQueryLog_CombinedFilters_TotalAndUnfiltered_FR003`, `TestQueryLog_PlanScope_IsOrWasInside_FR003` (deleted and purged records included), `TestQueryLog_MovedInRecordMatches_FR003`, `TestQueryLog_NewestFirstBySeq_FR004`, `TestQueryLog_BeforeSeqPaging_FR009`, `TestQueryLog_RecordNowStates_FR008` (exists with a new name, in bin inside another entry with its `binEntryId`, purged, none), `TestQueryLog_RecordRef_FR013`, `TestQueryLog_Validation_AllProblemsAtOnce`, `TestLogSummary_Counts_FR002`, `TestLogSummary_IgnoresFilters_FR002`, `TestLogSummary_Facets_FR003`, `TestLog_EntriesNeverChangeOrDisappear_FR010_FR030`, `TestQueryLog_50000Entries_SC006`
- [x] T009 [US2] Implement `backend/internal/history/query.go` (depends on T007, T008): `QueryLog` (validation, filters, *Action filter*, *Scope matching*, newest first, `Total`/`UnfilteredTotal`, `BeforeSeq`, `Limit`, `RecordNow`, `Record`), `LogSummary`; make `ListLog` (001) delegate to `QueryLog`
- [x] T010 [US2] Extend `backend/internal/history/handler.go`: `GET /api/activity-log` parses `actorId`, `recordType`, `action` (repeat), `planId`, `recordId`, `includeInside`, `from`, `to`, `q`, `beforeSeq`, `limit` (invalid → `validation_failed` with the messages of plan *Errors and messages*) and answers `LogPage`; new `GET /api/activity-log/summary` (`todayStart`, `weekStart` required); tests in `backend/internal/history/handler_test.go`: `TestListActivityLog_Filters_Contract`, `TestListActivityLog_InvalidParams_400`, `TestActivityLogSummary_Contract`, `TestActivityLogSummary_MissingParams_400`

### US3 — Record history (backend)

- [x] T011 [US3] In `backend/internal/history/query_test.go` add `TestQueryLog_RecordWithItemsInside_FR012_FR013` (record only vs with items inside, incl. a deleted child and a child moved in) and make it pass in `backend/internal/history/query.go` (depends on T009)

### US1 — Find and restore (backend)

- [x] T012 [US1] Write `backend/internal/history/bin_test.go`: `TestListBin_RestoreStates_FR017`, `TestListBin_RecordsAndInside_FR015_FR028`, `TestListBin_UnregisteredKindHidden`, `TestPreviewRestore_DestinationIsCurrentPlace_EdgeCase`, `TestPreviewRestore_StatusChanges_FR018_FR019` (fake kinds whose `PrepareRestore` resets `status`), `TestPreviewRestore_NameClash_FR022`, `TestPreviewRestore_OutsideRecordChange_FR020` (a fake hook logs a `statusChanged` of a stored parent through `Log`), `TestPreviewRestore_ChangesNothing_FR018` (store, counters and log unchanged), `TestRestore_SameIDsAndValues_FR021`, `TestRestore_NotBlockedByOtherRules_FR025`, `TestRestore_LogsActorAndStatusChanges_FR019_FR026`, `TestUndo_StillExact_FR027`, `TestBin_NothingLeavesAutomatically_FR031`
- [x] T013 [US1] Implement `backend/internal/history/bin.go` (depends on T007, T012): `ListBinItems` (restore state per data-model §5, chain, `InsideEntryIDs`, `Records`, `RecordTypePlural`, unregistered kinds left out), `PreviewRestore` (dry run in one `Write` with the log collector, returning `errDryRun`; `Destination` after the parents came back; `Changes` with `Restored`; `Notes`; no dry run for `parentDeletedPermanently`)
- [x] T014 [US1] Extend `backend/internal/history/handler.go`: `GET /api/recycle-bin` answers `BinPage{Items: []BinListItem}`; new `GET /api/recycle-bin/{entryId}/restore-preview`; tests `TestListRecycleBin_Items_Contract`, `TestRestorePreview_Contract`, `TestRestorePreview_UnknownAndResolved_404_409` in `backend/internal/history/handler_test.go`

### US4 — Restore with its parent (backend)

- [x] T015 [US4] In `backend/internal/history/bin_test.go` add `TestListBin_ChainParentsFirst_FR023`, `TestRestoreWithParents_ParentsFirstOneStep_FR023` (two levels: "Restore all 3"), `TestRestore_ParentPurged_Refused_FR024` (nothing changes); implement `RestoreWithParents` in `backend/internal/history/bin.go` (depends on T013)
- [x] T016 [US4] `POST /api/recycle-bin/{entryId}/restore` reads `withParents` (`true`/`false`, otherwise `validation_failed`) and calls `RestoreWithParents` or `Restore` in `backend/internal/history/handler.go`; tests `TestRestore_WithParents_Contract`, `TestRestore_ParentCodes_409` (both `restore_parent_first` and `parent_deleted_permanently`) in `backend/internal/history/handler_test.go` (depends on T014, T015)

### US5 — Delete permanently and empty (backend)

- [x] T017 [US5] In `backend/internal/history/bin_test.go` add `TestPurge_RemovesForGoodAndLogs_FR030` (resolution, no snapshots, one `deletedPermanently` entry with the stored path and details, IDs not reused), `TestPurge_InsideEntriesBecomeUnrestorable_FR028`, `TestEmptyBin_OneEntryWithItems_FR029_FR030`, `TestEmptyBin_SkipsResolvedAndUnlisted_FR032`, `TestEmptyBin_NothingLeft_NoEntry`, `TestActions_AlreadyResolved_FR032` (restore after purge, purge after restore, preview after purge — each names who and when); implement `Purge` and `EmptyBin` in `backend/internal/history/bin.go` (depends on T013)
- [x] T018 [US5] Add `DELETE /api/recycle-bin/{entryId}` and `POST /api/recycle-bin/empty` to `backend/internal/history/handler.go`; tests `TestPurge_Contract`, `TestPurge_404_409`, `TestEmptyRecycleBin_Contract`, `TestEmptyRecycleBin_EmptyList_400` in `backend/internal/history/handler_test.go` (depends on T016, T017 — same file)

### Demo data

- [x] T019 [US1] [US2] Write `backend/seed/history.go` with `addHistory(ctx, generator, now)` as plan *Demo data* (created entries 14–8 days back, published plans, 008's approval history as `statusChanged` entries, one planned-cost change by Priya Raman, the deleted initiative "Trade shows" with "Fleet expo stand" and "Stand build" as a bin entry 2 days back), entries sorted by time before they are appended; call it last in `backend/seed/seed.go`; test `TestSeed_HistoryAndRecycleBin_011` in `backend/seed/seed_test.go` (entries in time order, within 14 days, one unresolved bin entry with contents 1 activity and 1 line item, its records not stored, `GET /api/recycle-bin` restore state `restorable`)

**Checkpoint**: `go test ./...` passes, including 001's `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005`.

---

## Phase 3: Frontend

### Foundation (blocks the story groups below)

- [x] T020 [P] Make `RECYCLE_BIN` and `ACTIVITY_LOG` plain `NavItem`s and remove them from `COMING_SOON_SECTIONS` in `frontend/src/app/navigation.ts`; add the routes `/activity-log` (`ActivityLogPage`) and `/recycle-bin` (`RecycleBinPage`) with `handle.title` in `frontend/src/app/router.tsx`; update `frontend/src/app/navigation.test.ts` and `frontend/src/app/router.test.tsx` (the two sections no longer show "Coming soon") (FR-001, FR-014)
- [x] T021 [P] In `frontend/src/api/errors.ts` add the kind `parentDeletedPermanently` (`KIND_BY_CODE`) and the `deletedPermanently` wording in `resolvedMessage`; tests in `frontend/src/api/errors.test.ts` (FR-024, FR-032)
- [x] T022 [P] Add `formatWeekdayDate(date)` ("Monday 12 Oct 2026") to `frontend/src/lib/format.ts` with tests in `frontend/src/lib/format.test.ts`; when `formatRelativeTime` is not there yet (008/009), add it as 009's plan specifies, with its tests
- [x] T023 [P] When 009 has not been built yet: add `SummaryCard`'s `icon`, `detail`, `to` and `linkLabel` props (`frontend/src/components/SummaryCard/SummaryCard.tsx` + test) and `frontend/src/lib/recordLinks.ts` (`recordHref`, + test) exactly as 009's plan specifies; otherwise nothing to do

### US2 — Activity log page (frontend)

- [x] T024 [P] [US2] Write `frontend/src/features/activity-log/logText.ts` and `logText.test.ts`: sentence per entry (incl. emptied bin with item count), action word and tone table (FR-005, FR-006), detail lines (FR-007), record-link decision per `recordNow.state` and " — now '<name>'" (FR-008), day label "Today" / "Yesterday" / weekday date (FR-004)
- [x] T025 [P] [US2] Write `frontend/src/features/activity-log/logFilters.ts` and `logFilters.test.ts`: URL params ⇄ filter state ⇄ API query (`actorId`, `recordType`, `action`[], `planId`, `recordId`, `includeInside`, `from`, `to`, `q`), local date ranges (today, Monday week start, last 30 days, year, custom with exclusive end; tested in UTC and with a non-UTC offset), `todayStart`/`weekStart`, chip texts (FR-002, FR-003)
- [x] T026 [US2] Write `frontend/src/features/activity-log/activityLogApi.ts`: `useLogSummary(todayStart, weekStart)`, `useLogEntries(query, pageSize)` (`useInfiniteQuery`, next `beforeSeq` = last `seq`), query keys of the plan (depends on T002)
- [x] T027 [US2] Write `frontend/src/features/activity-log/LogEntry.tsx` (+ `LogEntry.module.css`, `LogEntry.test.tsx`): avatar, sentence, ID, path, time with full date in a tooltip, action `StatusChip`, record link states, **Show details** / **Hide details** with changes, comment, details and items; no edit or delete control (FR-005 – FR-008, FR-010) (depends on T024)
- [x] T028 [US2] Write `frontend/src/features/activity-log/LogEntryList.tsx` (+ css, `LogEntryList.test.tsx`): day groups with counts, "Showing the last <loaded> of <N>", **Load more** (FR-004, FR-009) (depends on T027)
- [x] T029 [US2] Write `frontend/src/features/activity-log/LogFilters.tsx` (+ css, `LogFilters.test.tsx`): Person, Record type, Plan selects, Action checkbox menu (`modal={false}`), date-range pills with Custom dates and their error, search (300 ms), chips with **Remove filter …** and **Clear all**, count line (FR-003) (depends on T025)
- [x] T030 [US2] Write `frontend/src/features/activity-log/ActivityLogPage.tsx` (+ css, `ActivityLogPage.test.tsx`): header (FR-001), four summary cards with tips and filter links (FR-002), filters from the URL, list, loading / error / "No changes yet" / "Nothing matches these filters" with **Clear filters** (FR-003, FR-004, FR-009, FR-010) (depends on T026, T028, T029)

### US3 — History section (frontend)

- [x] T031 [US3] Write `frontend/src/features/activity-log/HistorySection.tsx` (+ css, `HistorySection.test.tsx`) and `frontend/src/features/activity-log/showHistory.ts`: card `#history`, include checkbox label by level, preference saved with `PATCH /api/me/preferences` (failed save → "Your choice couldn't be saved." with **Try again**), 20 entries with **Load more**, **Open in activity log** URL, empty and error states (FR-011 – FR-013) (depends on T026, T028)
- [x] T032 [P] [US3] Add `<HistorySection level="plan" …/>` to the plan page of 004 (`frontend/src/features/plans/PlanPage.tsx`) and make its **⋯ More › View history** call `showHistory()`; extend its page test
- [x] T033 [P] [US3] Same for the campaign and initiative details of 005 (levels `campaign`, `initiative`) and their View history items; extend their tests
- [x] T034 [P] [US3] Same for the activity page of 006 (`frontend/src/features/activities/ActivityPage.tsx`, level `activity`, **Include line items**); extend its test
- [x] T035 [P] [US3] Same for the line item details of 007 (level `lineItem`, no include option); extend its test

### US1 — Recycle bin page and restore (frontend)

- [x] T036 [P] [US1] Write `frontend/src/features/recycle-bin/binText.ts` and `binText.test.ts`: count line, contents text ("2 activities and 5 line items", "Nothing inside"), "Was in" path, restore-state lines, search matching over `records` with "Contains <id> '<name>'" (+ "and n more"), type pills from the data, destination sentence, the *Restore sentences* table, purge and empty dialog texts (FR-014 – FR-018, FR-022, FR-028, FR-029)
- [x] T037 [US1] Write `frontend/src/features/recycle-bin/recycleBinApi.ts`: `useRecycleBin()`, `useRestorePreview(entryId)`, `useRestore()` (`withParents`), `usePurge()`, `useEmptyBin()`; after success invalidate every query except `['session']` (depends on T002)
- [x] T038 [US1] Write `frontend/src/features/recycle-bin/BinRow.tsx` (+ css, `BinRow.test.tsx`): level tag, name, ID, "Was in", contents, deleted by and relative time with tooltip, state line, **Restore**, ⋯ **Delete permanently**, contents tree, highlight (FR-015, FR-017) (depends on T036)
- [x] T039 [US1] Write `frontend/src/features/recycle-bin/RestoreDialog.tsx` (+ `RestoreDialog.test.tsx`): loads the preview, title, destination, change sentences, notes, **Restore <type>**, loading and error states, toast "<Type> '<name>' restored — View", `already_resolved` → `resolvedMessage` + refetch (FR-018 – FR-022, FR-025, FR-026, FR-032) (depends on T036, T037)
- [x] T040 [US1] Write `frontend/src/features/recycle-bin/RecycleBinPage.tsx` (+ css, `RecycleBinPage.test.tsx`): header with count line and **Empty recycle bin** only when not empty, search, type pills, "Showing N of M", rows, `?record=` highlight and "That item is no longer in the recycle bin.", loading / error / empty / filtered-empty states (FR-014 – FR-017, SC-002) (depends on T038, T039)

### US4 — Restore with its parent (frontend)

- [x] T041 [US4] Extend `frontend/src/features/recycle-bin/RestoreDialog.tsx` and its test: parent block, **Restore both** / **Restore all <N>** sending `withParents=true`; blocked dialog "This <type> can't be restored" whose **Delete permanently** opens the purge dialog; `restore_parent_first` / `parent_deleted_permanently` toasts + refetch (FR-023, FR-024) (depends on T039)

### US5 — Delete permanently and empty (frontend)

- [x] T042 [P] [US5] Write `frontend/src/features/recycle-bin/PurgeDialog.tsx` (+ `PurgeDialog.test.tsx`): title, contents, "This can't be undone.", inside-entry warning (1 / n), **Cancel** focused, red **Delete permanently**, toast, `already_resolved` (FR-028, FR-030, FR-032) (depends on T036, T037)
- [x] T043 [P] [US5] Write `frontend/src/features/recycle-bin/EmptyBinDialog.tsx` (+ `EmptyBinDialog.test.tsx`): counts and contents totals of all listed entries, sends their IDs, toasts for n / 1 / 0 (FR-029, FR-030) (depends on T036, T037)
- [x] T044 [US5] Wire `PurgeDialog` (row menu and blocked restore) and `EmptyBinDialog` (header button) into `frontend/src/features/recycle-bin/RecycleBinPage.tsx`; extend `RecycleBinPage.test.tsx` (depends on T040 – T043)

**Checkpoint**: `npm run check` (types, lint, format, unit tests) passes.

---

## Phase 4: End-to-end (needs Phases 2 and 3)

- [x] T045 [US1] [US4] [US5] Write `frontend/e2e/recycle-bin.spec.ts`: seeded "Trade shows" row with its contents and deleter; search the activity's ID → "Contains …"; Restore → dialog text → toast → the plan page shows it, the activity is Created (SC-001: ≤ 3 steps); delete an approved activity, let the Undo message close, restore it from the bin → Created and "must be submitted for approval again" in the dialog; delete an activity then its initiative → "Its initiative … is also in the recycle bin" → **Restore both**; Delete permanently and Empty recycle bin with their texts, then the activity log shows "emptied the recycle bin"; axe (WCAG 2.1 AA) on the page and on each dialog (SC-008)
- [x] T046 [US2] [US3] Write `frontend/e2e/activity-log.spec.ts`: cards and a card link; filter by person and action → chips and "Showing N of M entries"; expand an entry; follow a record link and a "(in the recycle bin)" link to the highlighted row; change an activity's planned cost twice and a line item once → its History without and with **Include line items** (choice kept after reload) → **Open in activity log** shows the Record chip (SC-005: ≤ 3 steps); axe on the page and the History section; keyboard-only: filter, expand, Load more (SC-008)

---

## Phase 5: Polish

- [x] T047 [P] Update the package comment of `backend/internal/history/model.go` (011 adds the queries, restore preview, purge and empty) and the "History" tag description in `backend/api/openapi.yaml`
- [x] T048 [P] Set feature 011's status in `docs/sdd/feature-map.md`
- [x] T049 Run `go test ./...` (in `backend/`), `npm run check` and `npm run test:e2e` (in `frontend/`); fix any failure; verify the demo with the seed: the bin shows one item, the log shows two weeks of history

---

## Requirement coverage

| FR | Test tasks | Implementation tasks |
|---|---|---|
| FR-001 | T020, T030, T046 | T020, T030 |
| FR-002 | T008, T010, T025, T030, T046 | T009, T010, T030 |
| FR-003 | T008, T010, T025, T029, T030, T046 | T009, T010, T029, T030 |
| FR-004 | T008, T024, T028 | T009, T028 |
| FR-005 – FR-007 | T024, T027, T046 | T027 |
| FR-008 | T008, T024, T027, T046 | T009, T027 |
| FR-009 | T008, T028, T030 | T009, T028 |
| FR-010 | T008, T027, T030 | (no write route) |
| FR-011 – FR-013 | T005, T011, T031, T032 – T035, T046 | T005, T011, T031 – T035 |
| FR-014 – FR-017 | T012, T014, T036, T038, T040, T045 | T013, T014, T038, T040 |
| FR-018 – FR-022 | T012, T014, T036, T039, T045 | T013, T039 |
| FR-023 | T007, T015, T016, T041, T045 | T015, T016, T041 |
| FR-024 | T004, T015, T016, T021, T041 | T004, T007, T041 |
| FR-025 – FR-027 | T012, T039 | T007, T013 |
| FR-028 – FR-030 | T008, T017, T018, T042, T043, T045 | T017, T018, T042 – T044 |
| FR-031 | T012 | (no timer) |
| FR-032 | T017, T021, T039, T042 | T017, T021, T039, T042 |
| SC-006 | T008 | T009 |
| Seed (US1, US2) | T019 | T019 |

## Dependencies & Execution Order

- **Setup** (T001 → T002 → T003) first.
- **Backend** (Phase 2) and **Frontend** (Phase 3) start after T001/T002 and do not wait for each other.
  - Backend: T004, T005, T006 in parallel → T007 → story groups. US2 (T008 → T009 → T010, then T011),
    US1 (T012 → T013 → T014), then US4 (T015 → T016) and US5 (T017 → T018); T010, T014, T016, T018
    edit `handler.go` in that order. T019 after T013 and T009.
  - Frontend: T020 – T023 in parallel → US2 (T024, T025 in parallel → T026 → T027 → T028 → T029 →
    T030) and US1 (T036 → T037 → T038 → T039 → T040) can run side by side; US3 (T031 → T032 – T035 in
    parallel) after T028; US4 (T041) after T039; US5 (T042, T043 in parallel → T044) after T040.
- **End-to-end** after both phases; **Polish** last.
- Cross-feature: needs 004–008 built (kinds, hooks, record pages, seed). 009's Recent changes card uses
  T009 (`QueryLog`); when 009 is built first, its backend agent runs T006, T007 and T009 from this list.

## Notes

- `[P]` tasks touch different files. Commit after each group, e.g. `feat(011): recycle bin restore preview (FR-018)`.
- Keep every text identical to the plan's messages table; tests quote it.
