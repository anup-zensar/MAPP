# Tasks: Planning Rules Aligned with Shell's Process

**Input**: Design documents from `specs/034-planning-alignment/` — [plan.md](plan.md), [spec.md](spec.md),
[research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/),
[quickstart.md](quickstart.md)

**Tests**: required (constitution VI): every rule and acceptance scenario has a test; tests are written or
changed first and fail before the code changes; test names carry `034 FR-###` (Go: `_034_FR###` in the
function name or `034 FR-###` in the subtest name; Vitest: `describe("034 FR-###: …")`). Tests that
asserted the old rule are changed to the new one and cite D-322 (research R9), never dropped silently.

**Organization**: grouped by user story (US1 P1, US2 P1, US3 P2). No branch and no commits in this
delivery (product owner's choice), so the git hooks of Spec Kit are skipped.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1, US2, US3

---

## Phase 1: Setup

- [X] T001 Install the web app's dependencies offline in the worktree (`cd frontend && pnpm install --frozen-lockfile --offline`) and confirm the baseline: `cd backend && go test -short ./...` and `cd frontend && pnpm run test` pass before any change (record failures that exist before 034)

---

## Phase 2: Foundational

No blocking prerequisite: the three stories touch different modules (line items and approval texts; seed;
plans). The contract change (T010) belongs to US1.

---

## Phase 3: User Story 1 — Work on the line items of an approved activity without losing its approval (P1) 🎯 MVP

**Goal**: line-item add, duplicate, edit, delete and restore never change the activity's status or ask for
confirmation; the activity's own planned cost and dates still do.

**Independent test**: spec US1 "Independent Test"; quickstart scenarios 1–3.

### Tests for User Story 1 (change or write first; they fail before T008–T017)

- [X] T002 [P] [US1] Change `TestCreate_RestartNeedsConfirmation_FR031` into `TestCreate_InApprovalKeepsStatus_034_FR001` in backend/internal/lineitem/create_test.go: on the Approved fake activity a create without any flag succeeds, the activity stays `approved`, `fakeActivities` records no status change, one "created" entry; also on a Submitted one (D-322)
- [X] T003 [P] [US1] Add `TestCreate_NegativeBalanceKeepsApproval_034_FR004` in backend/internal/lineitem/create_test.go: a line item bigger than the Approved activity's unassigned planned cost is saved, the section's balance is negative, the activity stays `approved`
- [X] T004 [P] [US1] Change `TestDuplicate_RestartNeedsConfirmation_FR031` into `TestDuplicate_InApprovalKeepsStatus_034_FR001` in backend/internal/lineitem/duplicate_test.go (copy made without flag; status unchanged; 404 for an unknown id kept)
- [X] T005 [P] [US1] Change `TestUpdate_CostOrCurrencyRestart_FR031` into `TestUpdate_CostOrCurrencyKeepsStatus_034_FR001` in backend/internal/lineitem/update_test.go (cost and currency changes saved without flag; activity status unchanged; one "changed" entry each)
- [X] T006 [P] [US1] Change `TestDelete_RestartNeedsConfirmation_FR031` into `TestDelete_InApprovalKeepsStatus_034_FR001`, and the approval check of `TestDelete_MovesToBin_FR030`, in backend/internal/lineitem/delete_test.go; change `TestRestore_IntoApprovedRestarts_FR033` into `TestRestore_IntoApprovedKeepsStatus_034_FR002` (no note, status unchanged) and review `TestUndo_EndedApprovalStaysEnded_FR033` in backend/internal/lineitem/kinds_test.go
- [X] T007 [P] [US1] Change the handler contract tests in backend/internal/lineitem/handler_test.go: create, duplicate and delete on the Approved demo activity A-1002 answer 201/200 without the flag and the activity stays `approved` (034 FR-001); the deprecated flag (`restartApproval: true`, `?restartApproval=true`) is still accepted; drop the `?restartApproval=maybe` refusal (the parameter is no longer read) and the `restartApproval` problem from the expected 400 body
- [X] T008 [P] [US1] In backend/internal/approval/handler_changes_test.go change `TestLineItemCurrencyChangeRestartsApproval_FR022` into `TestLineItemCurrencyChangeKeepsApproval_034_FR001` (PATCH L-1003 to EUR without flag → 200; A-1002 approval status `approved`, round 1 unchanged, no new history entry) add `TestLineItemChangeOnSubmittedKeepsRound_034_FR001` (a line item added to and re-costed on the Submitted A-1004: it stays `submitted`, round AR-1004 stays open at its level, Daniel's `tab=waiting` still lists it — US1 scenario 2), and the recall sentence in `TestRecall_WaitingMobilityActivity_FR021`; change the sentence in backend/internal/approval/view_test.go and backend/internal/approval/rounds_recall_test.go to "Approved activities can't be recalled. Changing its planned cost or dates sends it back for approval." (034 FR-005)
- [X] T009 [P] [US1] Change the web tests to the new rule, `034 FR-001` in their names: frontend/src/features/line-items/LineItemCreatePage.save.test.tsx and LineItemEditPage.test.tsx (approved activity: Save sends at once, no "This activity will need approval again" dialog, body without `restartApproval`), LineItemPage.test.tsx (Duplicate sends `{}` at once; the delete dialog has no "This activity will need approval again." and the DELETE has no query), line-item-requests.test.ts and line-item-text.test.ts (remove the tests of the removed helpers); frontend/src/testing/fixtures/line-items.ts sets `restartsApproval: false`

### Implementation for User Story 1

- [X] T010 [US1] Contract: in backend/api/openapi.yaml change the descriptions of createLineItem, updateLineItem, deleteLineItem, duplicateLineItem; mark `restartApproval` of LineItemCreate, LineItemUpdate, LineItemDuplicate and the `RestartApproval` parameter `deprecated: true` ("Ignored since feature 034 (D-322)"), and `LineItemActivity.restartsApproval` deprecated and "always false" (contracts/openapi.yaml); then `cd frontend && pnpm run api:types`
- [X] T011 [US1] Line-item service: remove `confirmRestart`, `restart`, `restartMessage` and the reason texts from backend/internal/lineitem/service.go; remove the restart steps from `Create` (create.go), `Duplicate` (duplicate.go), `saveEdit`/`edit.restartConfirmed` (update.go) and `Delete` (delete.go, which loses its `restartApproval bool` parameter); update their doc comments (FR-001)
- [X] T012 [US1] Remove `RestartApproval` from the `Activities` port and `ActivityFacts.restartsApproval` in backend/internal/lineitem/deps.go; `LineItemActivity.RestartsApproval` stays false in backend/internal/lineitem/view.go and model.go (comment: kept for older clients, always false); `Input`, `Update`, `DuplicateInput` keep `RestartApproval` as an ignored wire field; `Update.input()` in validate.go stops copying it
- [X] T013 [US1] Drop `AfterRestore`, `restartedNote` and `restoredReason` from backend/internal/lineitem/kinds.go (FR-002); the handler's delete in backend/internal/lineitem/handler.go no longer reads `?restartApproval`
- [X] T014 [US1] Update the test fakes: remove `RestartApproval`/`restartCall` from backend/internal/lineitem/fakes_test.go and the flag uses in fixture_test.go and service_test.go
- [X] T015 [US1] Remove `Service.RestartApproval` from backend/internal/activity/reapproval.go and its two tests from reapproval_test.go (research R9); update the `ApprovalRestarted` comment in backend/internal/activity/hooks.go; remove the adapter `lineItemActivities.RestartApproval` from backend/internal/app/lineitems.go (FR-001, FR-003)
- [X] T016 [US1] Change the recall explanation in backend/internal/approval/messages.go and frontend/src/features/approvals/messages.ts to "Approved activities can't be recalled. Changing its planned cost or dates sends it back for approval." (FR-005)
- [X] T017 [US1] Web app: delete frontend/src/features/line-items/RestartApprovalDialog.tsx; `useLineItemSave.ts` sends at once (no `pending`, `confirmRestart`, `cancelRestart`, no restart refusal handling); `LineItemForm.tsx` drops the dialog; `useLineItemActions.tsx` duplicates and deletes without restart; `DeleteLineItemDialog.tsx` loses `restartsApproval`; `line-item-requests.ts` loses `needsRestartConfirmation`, `restartReasons`, `isRestartRefusal` and the `restartApproval` body field; `api.ts` duplicate sends `{}` and delete no query; `line-item-text.ts` loses the restart texts (FR-001, FR-005)
- [X] T018 [US1] End-to-end journey `test('034 US1 …')` in frontend/e2e/planning-alignment.spec.ts: an Approved activity prepared through the API (createPlanningRecords + approveThroughApi); on screen add a line item (no dialog, toast, still Approved), delete it (dialog without the approval line, still Approved, Undo); axe check

**Checkpoint**: US1 complete — the MVP.

---

## Phase 4: User Story 2 — Code line items and activities with Shell's own lists (P1)

**Goal**: the seed carries FA E3.0.2's lists; GL code has no parent; demo records stay valid.

**Independent test**: spec US2 "Independent Test"; quickstart scenario 4.

### Tests for User Story 2

- [X] T019 [P] [US2] Add `TestShellLists_034_FR006_FR010` in backend/seed/settings_test.go: the options of GL code (13), Spend type (25), Spend category (19), Contracted spend (5) and Zero-based cost hierarchy (10) equal the lists of spec FR-006 – FR-010 in order, character for character; GL code has no parent attribute and no option links (FR-011); labels, markings and sections unchanged (FR-012); change the dependency test that expected Spend type → GL code links (Cluster → Region stays)
- [X] T020 [P] [US2] Change backend/seed/lineitems_test.go, activities_test.go (if it lists the zero-based value) and seed_test.go (162 options, next ID SO-1163) to the new demo values of research R7 with the same completeness (034 FR-013)
- [X] T021 [P] [US2] Line-item form keeps a GL code whatever the spend type when GL code has no parent: confirm with a Vitest in frontend/src/features/line-items/LineItemCreatePage.test.tsx using an independent GL code attribute (`034 FR-011`)

### Implementation for User Story 2

- [X] T022 [US2] Seed lists: in backend/seed/settings.go replace the options of Zero-based cost hierarchy, Spend type, GL code (remove `parent`), Spend category and Contracted spend with the lists of data-model §2, and their help texts; update the comment counts (35 attributes, 162 options)
- [X] T023 [US2] Demo values: backend/seed/names.go constants and backend/seed/activities.go (Launch film's zero-based value "LUBES - Growth drivers") per research R7
- [X] T024 [US2] End-to-end journey `test('034 US2 …')` in frontend/e2e/planning-alignment.spec.ts: on the line-item form of an activity, Spend type and GL code offer Shell's values (GL code 13 options after choosing a spend type); save with "BTL - Event" and "7250150 - Sponsorship programmes"; the line item page shows them; axe check

---

## Phase 5: User Story 3 — Plan for this year and the next ten (P2)

**Goal**: new plans of the current year to + 10; past-year plans keep working.

**Independent test**: spec US3 "Independent Test"; quickstart scenario 5.

### Tests for User Story 3

- [X] T025 [P] [US3] In backend/internal/plan tests (create_test.go / service_test.go / copy_test.go): `034 FR-014` defaults list current … + 10 and the create check refuses current − 1 and current + 11 with "Choose a year from <current> to <current + 10>."; `034 FR-016` next-year copy available up to current + 10 and refused after with "Plans can only be created up to <current + 10>."; change existing tests that expected − 1 … + 5
- [X] T026 [P] [US3] Add `TestUpdate_PastYearPlan_034_FR015` in backend/internal/plan/edit_test.go: with the clock moved to the following year, a plan of the previous year is edited and saved, its year unchanged
- [X] T027 [P] [US3] Update web fixtures/tests that list plan years (frontend/src/testing/fixtures, PlanCreatePage/PlanForm tests) to current … + 10 where they assert the list

### Implementation for User Story 3

- [X] T028 [US3] In backend/internal/plan/validate.go set `firstYearBeforeCurrent = 0` and `lastYearAfterCurrent = 10`; update the comments in validate_fields.go, copy.go and model.go (`Years`), and the plan-defaults and PlanDefaults/NextYearOption descriptions in backend/api/openapi.yaml (FR-014 – FR-016)

---

## Phase 6: Polish & cross-cutting

- [X] T029 [P] Docs (FR-017): docs/product/domain-model.md §2.5 and R11; docs/product/field-catalogue.md §1 #2, §4.2 #21, §5 #8–#11 and the intro sentence on sample option values; docs/product/demo-users-and-data.md (spend type/GL code example)
- [X] T030 [P] Amendment notes (one line each, "Amended by 034 (D-322/D-326)") in specs/007-line-items/spec.md (US4, FR-031, FR-033), specs/008-activity-approvals/spec.md (US5 scenario 2, FR-021) and specs/004-plans/spec.md (FR-006, FR-028)
- [X] T031 Comment budget and file length of every touched file (≤ 5 % comment lines, ≤ 300 non-blank lines); no requirement IDs in source comments
- [X] T032 Backend gates: `gofmt -l`, `golangci-lint run --new ./...`, `sqlc generate && sqlc vet`, `go vet ./...`, `go test -race ./...` with TEST_DATABASE_URL, `govulncheck ./...`
- [X] T033 Frontend gates: `pnpm run api:check`, `typecheck`, `lint`, `format:check`, `test`, `build`
- [X] T034 E2E on mapp_e2e_034 (ports 18134/14134): `pnpm run test:e2e -- e2e/planning-alignment.spec.ts`, then the whole suite (`pnpm run test:e2e`) so every journey that passed before still passes (SC-004)
- [X] T035 Run the app on mapp_f034 (API 18134) and exercise quickstart scenarios 1–5 with curl; record what was seen

---

## Dependencies & execution order

- Phase 1 first. US1, US2 and US3 are independent of each other; each story's tests come before its
  implementation. T010 (contract) before T017 (web app uses the regenerated types).
- T018 and T024 share one e2e file: write T018 first, then add T024.
- Phase 6 after the stories.

## Parallel examples

- US1 tests: T002–T009 touch different files and can be written together.
- US2: T019, T020, T021 together; then T022, T023.
- US3: T025, T026, T027 together; then T028.

## Implementation strategy

MVP = US1 (the process blocker). Then US2 (lists for the SWBS and PR demos), then US3. Each story is
checked against its independent test before the next.

---

## Phase 7: Convergence

- [X] T036 Remove the stale doc comment about the restart refusal left above `notFound` in frontend/src/testing/fixtures/line-item-forms.ts per FR-005 (contradicts)
- [X] T037 Correct the doc comment of LineItemEditPage in frontend/src/features/line-items/LineItemEditPage.tsx, which still said a cost change asks to restart approval, per FR-005 (contradicts)
- [X] T038 Answer the copy page's Financials request in the duplicate test of frontend/src/features/line-items/LineItemPage.test.tsx (the copy now opens without a dialog, so its request always lands) per FR-001 (partial)
