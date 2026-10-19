---

description: "Task list for feature 003 — People and Teams"
---

# Tasks: People and Teams

**Input**: Design documents from `specs/003-people-and-teams/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); feature 001 built (history core, components, `apitest`)

**Tests**: required — constitution V. Within each group write the tests first and see them fail, then
implement. Test names carry the FR they prove (plan *Tests*). Every user-visible text comes from the plan's
*User-visible messages* table; every rule from the plan and data-model.md (when they seem to differ, the
plan wins and the difference is reported).

**Organization**: Setup (the contract) → **Backend** and **Frontend** phases, which are independent of each
other and both start from the contract (a backend agent and a frontend agent work in parallel) → End-to-end
(needs both) → Polish. Inside each phase, tasks are grouped by user story: US1 people (P1), US2 teams (P1),
US3 person picker (P1), US4 My profile (P2).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 … US4
- Paths are relative to the repository root. Toolchain in Git Bash:
  `export PATH="$HOME/tools/node:/c/Program Files/Go/bin:$PATH"`.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [X] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [X] TC01 Do not create `SelectField`, `FilterPills`, `SearchField`, `TextAreaField` or `PageHeader` breadcrumbs; use 002's `ChoiceField`, `SegmentedFilter`, `SearchField`, `TextAreaField` (`rows?`) and `Breadcrumb` in every task that names them (D-197).
- [ ] TC02 `CurrentUser.version`/`teams` and `Preferences.displayCurrency`/`dateFormat` are optional in the contract: always send them; web code defaults when absent (D-191).
- [ ] TC99 Check the success criteria SC-002, SC-007, which no task above names, in the final review (timed end-to-end run or moderated session, as each criterion says) and record the result in `checklists/`.

## Phase 1: Setup (the shared contract)

**Purpose**: the API contract both sides build from. Blocks Phases 2 and 3.

- [x] T001 Merge `specs/003-people-and-teams/contracts/openapi.yaml` into `backend/api/openapi.yaml`: add the tag "People and teams", the paths `/api/people`, `/api/people/{personId}`, `/api/people/{personId}/impact`, `/api/people/{personId}/status`, `/api/teams`, `/api/teams/{teamId}`, `/api/teams/{teamId}/impact`, `/api/teams/{teamId}/status`, `/api/me/display-currencies`; add the operation `patch` (updateMyProfile) to the existing path `/api/me` (its `get` stays); add the parameters `PersonId`, `TeamId` and every schema of the fragment; **replace** `CurrentUser`, `Preferences`, `PreferencesUpdate` with the fragment's versions; add `not_allowed` to `Error.code` and name it in the `Conflict` response description; add `ownerNeedsAttention` to `ImpactWarning.kind`. If feature 002's contract is not merged yet, also add 002's `StatusUpdate`, `ChangeImpact`, `ImpactWarning` and the code `in_use` exactly as in `specs/002-settings-and-shell-attributes/contracts/openapi.yaml`. Mention feature 003 in `info.description` *(Done at consolidation, D-192.)*
- [x] T002 Regenerate `frontend/src/api/schema.d.ts` with `npm run api:types` in `frontend/`; `npm run api:check` passes (type errors in tests that build `CurrentUser` literals are fixed by T022) *(Done at consolidation, D-192.)*

**Checkpoint**: contract merged — Phases 2 and 3 start in parallel.

---

## Phase 2: Backend (Go API)

### Backend foundation (in order T003 → T008; T003 and T004 may run in parallel)

- [X] T003 [P] Add to `backend/internal/platform/errs/errs.go` the code `CodeNotAllowed = "not_allowed"` with the constructor `NotAllowed(message)` and — only if feature 002 has not added them — `CodeInUse = "in_use"` with `InUse(message)` (both 409, `Fields` empty); tests in `backend/internal/platform/errs/errs_test.go` (status, code, message, `"fields":[]`)
- [ ] T004 [P] Extend `backend/internal/people/model.go` with the stored types `Team`, `Membership`, the new `Preferences` fields, `palette`, `dateFormats`, and the API types and inputs of plan *Backend design* (`PersonRef`, `TeamRef`, `PersonView`, `PersonDetails`, `UsageCount`, `PersonPage`, `TeamView`, `TeamPlanUse`, `TeamPage`, `DisplayRate`, `DisplayCurrencyOption`, `DisplayCurrencyPage`, `ChangeImpact`, `ImpactWarning`, `PersonInput`, `PersonUpdate`, `TeamInput`, `TeamUpdate`, `StatusUpdate`, `ProfileUpdate`; `CurrentUser` + `Version`, `Teams`; `PreferencesView`/`PreferencesUpdate` + `DisplayCurrency`, `DateFormat`) with the contract's JSON names and `Person.Ref()`, `Team.Ref()`; create `backend/internal/people/messages.go` with every text of the plan's messages table that the API says (validation, refusals, owner-impact sentences, deactivate/reactivate verbs, palette labels); add `TestPerson_InitialsOneWordName_FR009` to `backend/internal/people/model_test.go`
- [ ] T005 Extend `backend/internal/people/repository.go` and `backend/internal/people/store_json.go`: `GetTeam`, `ListTeams`, `PutTeam`, `DeleteTeam` (table `teams`), `GetMembership`, `ListMemberships`, `PutMembership`, `DeleteMembership` (table `team_memberships`), `ListPreferences`; no person delete (FR-008)
- [X] T006 Write `backend/internal/people/usage.go`: `PlanUse`, `PlanSource`, `PersonUsage`, `Currency`, `CurrencySource`, `OnlyUSD`; the lookup functions `PersonRefs`, `TeamRefs`, `MemberIDs`, `TeamIDsOf`, `IsMemberOfAny`, `DisplayRateOf`, `PeopleByDisplayCurrency` (no actor, joining the caller's transaction; plan *Lookup functions*); the usage count builder used by `GetPerson` (plan owners from the PlanSource without bin plans, then each `PersonUsage` in registration order, counts ≥ 1, owners before approvers, singular/plural labels); tests in `backend/internal/people/usage_test.go`: `TestPersonRefs_UnknownIdsLeftOut`, `TestTeamRefs`, `TestMemberIDs`, `TestIsMemberOfAny_InactiveTeamCounts_FR014`, `TestTeamIDsOf_FR026`, `TestDisplayRateOf_EURAndFallback_FR023` (EUR "0.92"; unknown code → USD "1"; inactive EUR kept), `TestPeopleByDisplayCurrency_EmptyIsUSD`, `TestOnlyUSD`
- [X] T007 Change `backend/internal/people/service.go`: `NewService(tx, repo, generator *ids.Generator, hist *history.Service, now clock.Clock)` (currencies default `OnlyUSD`), `UsePlans`, `AddPersonUsage`, `UseCurrencies`; write `backend/internal/people/kinds.go` (`Kinds()`: `person` — `Remove` returns "people are never deleted (FR-008)"; `team`; `teamMembership` with parent `team` and `ChildIDs`; snapshots; `Describe` of data-model §6; the team's `PrepareRestore` as `samples.prepareFolderRestore` with the team texts); update `newTestService` in `backend/internal/people/service_test.go` (temporary store, generator, history service with the people kinds registered, fixed clock 2026-03-12 09:00 UTC, optional fake PlanSource/PersonUsage/CurrencySource); wire `backend/internal/app/app.go` (history before people, `people.NewService(db, peopleRepo, generator, historyService, now)`, register `peopleService.Kinds()` before the samples, auth unchanged); `TestKinds_Register_FR028` and `TestPersonKind_RemoveRefused_FR008` in `backend/internal/people/kinds_test.go`
- [X] T008 Write `backend/internal/people/memberships.go`: `applyMemberships` (create `TM-` memberships, delete removed ones, never a duplicate pair, `Touch` + save + one `changed` log entry per other record whose membership set changed, built with `history.Diff` over the log views of data-model §6) and `ownerImpact` + the warning message builder (plan *Membership changes and owner impact*). Its tests (`backend/internal/people/memberships_test.go`) go through the operations that use it and are written with them: person side in T009, team side in T012

### Backend — User Story 1: people directory (P1)

- [X] T009 [US1] Write the tests in `backend/internal/people/people_test.go` (plan *Tests*, row `people_test.go`): `TestListPeople_EveryoneSortedWithTeams_FR001`, `TestCreatePerson_SavedActiveAndLogged_FR001_FR028`, `TestCreatePerson_AllProblemsAtOnce_FR002_SC007`, `TestCreatePerson_EmailTrimmedAndLowerCase_FR002`, `TestCreatePerson_DuplicateEmailNamesHolder_FR002`, `TestCreatePerson_WithTeamsShowsOnBothSides_FR012`, `TestCreatePerson_InactiveOrUnknownTeamRefused_FR011`, `TestUpdatePerson_ChangesLoggedOldAndNew_FR028`, `TestUpdatePerson_MissingVersion_FR002`, `TestUpdatePerson_VersionConflict_FR029`, `TestUpdatePerson_NoChangeNoVersionNoLog_FR028`, `TestUpdatePerson_KeepsInactiveTeam_FR014`, `TestSetPersonStatus_DeactivateKeepsMembershipsAndLogs_FR005_FR028`, `TestSetPersonStatus_Yourself_NotAllowed_FR006`, `TestSetPersonStatus_AlreadyInactiveUnchanged_FR005`, `TestSetPersonStatus_ReactivateMembershipsIntact_FR007`, `TestSetPersonStatus_MissingActiveAndVersion`, `TestGetPerson_UsageCountsFromHooks_FR004`, `TestGetPerson_NotFound_FR008`; and in `backend/internal/people/memberships_test.go`: `TestPersonImpact_OneAndTwoRemovedTeams_FR013` (one removed team "uses <team>", two "uses the removed teams"; nothing saved), `TestImpact_ValidatesLikeUpdate_SC007`
- [X] T010 [US1] Write `backend/internal/people/people.go`: `ListPeople`, `GetPerson`, `CreatePerson`, `UpdatePerson`, `PersonImpact`, `SetPersonStatus` exactly as the plan's operations table (all field problems in one pass inside the `Write`; then, for the status action, 404 and `not_allowed` for yourself; then the version; then save + `applyMemberships` + log; `PersonImpact` runs the update's field checks, skips the version and saves nothing); T009 passes
- [X] T011 [US1] Write `backend/internal/people/people_handler.go` (list, create 201, get, patch, impact, status; `web.ReadJSON` → `actor.From` → service → `web.WriteJSON`/`WriteError`) and register the six routes in `Routes` of `backend/internal/people/handler.go`; write `backend/internal/people/people_handler_test.go` (`package people_test`, `apitest`): one test per operation (200/201, 400 with every field problem, 404, 409 version_conflict) plus `TestNewPersonCanSignIn_FR003`, `TestDeactivatedPersonCannotSignInAndSessionEnds_FR005`, `TestReactivatedPersonCanSignIn_FR007_SC004`, `TestDeactivateYourself_409NotAllowed_FR006`, `TestDeleteNotOffered_FR008` (`DELETE /api/people/U-1002` with `SkipValidation()` → JSON 404), `TestAnyoneManagesPeople_FR027`, `TestPersonImpact_NoPlansNoWarnings_FR013`

### Backend — User Story 2: teams and memberships (P1)

- [X] T012 [US2] Write the tests in `backend/internal/people/teams_test.go`: `TestCreateTeam_SavedWithMembersAndLogged_FR010_FR028`, `TestCreateTeam_AllProblemsAtOnce_FR011`, `TestCreateTeam_ColourLeastUsed_D025`, `TestUpdateTeam_RenameUnique_FR011`, `TestUpdateTeam_KeepsInactiveMember_FR011`, `TestUpdateTeam_VersionConflict_FR029`, `TestTeamEdit_MembershipShowsOnPerson_FR012`, `TestPersonEdit_MembershipShowsOnTeam_FR012`, `TestSetTeamStatus_DeactivateKeepsMembersAndLogs_FR014`, `TestSetTeamStatus_Reactivate_FR014`, `TestDeleteTeam_UsedByPlanRefused_FR014` (a stored plan; a plan only in the bin), `TestDeleteTeam_UnusedMovesWithMemberships_FR015`, `TestListTeams_MembersAndPlans_FR010`; in `backend/internal/people/memberships_test.go`: `TestMembershipChange_BothSidesNewVersionAndLog_FR012_FR028`, `TestTeamImpact_OwnerLosesLastPlanTeam_Warns_FR013` (exact message, count, nothing saved), `TestTeamImpact_OwnerStillInAnotherPlanTeam_NoWarning_FR013`, `TestTeamImpact_InactiveTeamStillCounts_FR014`, `TestImpact_IgnoresPlansInRecycleBin_FR013`, `TestUpdateTeam_RemovingOwnerIsAllowed_FR013`; and in `backend/internal/people/kinds_test.go`: `TestTeamDelete_UndoRestoresMembershipsExactly_FR015`, `TestTeamUndo_NameTakenRefused_FR015`, `TestTeamRestore_NameTakenGetsSuffix_FR015`, `TestDescribe_TeamsMembersStatusColour_FR028`
- [X] T013 [US2] Write `backend/internal/people/teams.go`: `ListTeams`, `GetTeam`, `CreateTeam` (least-used colour), `UpdateTeam`, `TeamImpact` (field checks of `UpdateTeam`, no version check, saves nothing), `SetTeamStatus`, `DeleteTeam` (`in_use` guard incl. plans in the bin, then `history.MoveToBin(ctx, a, "team", id)`); T012 passes
- [X] T014 [US2] Write `backend/internal/people/teams_handler.go` (seven operations: list, create 201, get, patch, delete, impact, status; delete answers `history.DeleteResult{RecycleBinEntry: entry.View()}`) and register the routes in `Routes`; write `backend/internal/people/teams_handler_test.go`: one test per operation plus `TestDeleteTeam_ThenUndo_FR015` (through `POST /api/recycle-bin/{entryId}/undo`), `TestRestoreTeam_NameTaken_FR015` and `TestTeamImpact_NoPlansNoWarnings_FR013`

### Backend — User Story 3: person picker (P1)

- [X] T015 [US3] Add to `backend/internal/people/people_handler_test.go` `TestListPeople_GivesPickerData_FR017_FR018` (every person with `email`, `jobTitle`, `active`, `teams` — the data the picker filters on; inactive people included so fields holding them can label them)

### Backend — User Story 4: My profile and preferences (P2)

- [ ] T016 [US4] Write the tests in `backend/internal/people/service_test.go`: update `TestPreferences_Defaults_FR011_FR014` (adds USD and DD MMM YYYY), add `TestPreferences_Old001RecordGetsDefaults_FR024`, `TestUpdatePreferences_DisplayCurrencyActiveOnly_FR023` (inactive current kept; inactive other refused), `TestUpdatePreferences_DateFormatChoices_FR024`, `TestUpdatePreferences_NotLogged_D038`, `TestCurrentUser_VersionAndTeams_FR026`, `TestUpdateProfile_NameAndTitleLogged_FR022`, `TestUpdateProfile_EmptyName_FR022`, `TestDisplayCurrencies_ActivePlusCurrent_FR023`; update `TestCurrentUser_PersonWithPreferences_FR012` to the new `CurrentUser`
- [ ] T017 [US4] Implement in `backend/internal/people/service.go`: `CurrentUser` (version, teams, preference defaults), `UpdatePreferences` (display currency and date format rules), `UpdateProfile`, `DisplayCurrencies`; T016 passes
- [ ] T018 [US4] Register `PATCH /api/me` and `GET /api/me/display-currencies` in `backend/internal/people/handler.go`; update `backend/internal/people/handler_test.go` (expected bodies of `TestGetMe_SignedInPerson_FR012` and `TestPatchPreferences_*`) and add `TestPatchMe_UpdatesProfile_FR021_FR022`, `TestPatchMe_VersionConflict_FR022`, `TestPatchPreferences_DisplayCurrencyAndDateFormat_FR023_FR024`, `TestGetDisplayCurrencies_FR023`, `TestPreferences_SameInAnotherBrowser_FR025`
- [ ] T019 [US4] Update the expected sign-in answer body in `backend/internal/auth/handler_test.go` (line with `"preferences":{"theme":"system","sidebarCollapsed":false}`) to the new `CurrentUser`

### Backend — demo data (FR-029)

- [ ] T020 Add the teams and memberships of data-model §7 to `backend/seed/seed.go` (`addTeams` after `addPeople`, same `Write`, counters `T` and `TM`, `actor.System`); `TestLoad_TeamsAndMemberships_FR029` in `backend/seed/seed_test.go` (names, ids, colours, descriptions, members; next ids `T-1004`, `TM-1009`); change the summary in `backend/cmd/seed/main.go` to "Loaded the demo data into <folder>: 6 people, 3 teams." and its expectation in `backend/cmd/seed/main_test.go`

- [X] T021 Backend checkpoint in `backend/`: `gofmt -l .` prints nothing, `go vet ./...` clean, `go test ./...` green — including `backend/internal/app/app_test.go` `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005` for the 15 new operations

**Checkpoint**: the API of feature 003 is complete and contract-checked.

---

## Phase 3: Frontend (web app)

### Frontend foundation (T022 first; then the [P] tasks T023–T031; T032 after T023; T033 → T034)

- [ ] T022 Create `frontend/src/test/users.ts` with `testUser(overrides?: Partial<CurrentUser>): CurrentUser` (Priya Raman, U-1002, version 1, teams `[Lubricants Europe T-1001 blue active]`, preferences `{theme: 'system', sidebarCollapsed: false, displayCurrency: 'USD', dateFormat: 'DD MMM YYYY'}`) and replace the inline `CurrentUser` literals with it in: `frontend/src/app/AccountMenu.test.tsx`, `AppLayout.test.tsx`, `NavDrawer.test.tsx`, `RequireSignIn.test.tsx`, `router.test.tsx`, `SessionProvider.test.tsx`, `Sidebar.test.tsx`, `ThemeToggle.test.tsx`, `frontend/src/features/dev/ComponentGalleryPage.test.tsx`, `frontend/src/features/dev/SampleRecordsPage.test.tsx`, `frontend/src/features/home/HomePage.test.tsx`, `frontend/src/features/placeholders/NotFoundPage.test.tsx`, `frontend/src/features/sign-in/SignInPage.test.tsx`, `frontend/src/lib/useUndoableDelete.test.tsx`, `frontend/src/lib/useUnsavedChanges.test.tsx`, `frontend/src/test/render.test.tsx`; `npm run typecheck` and `npm test` green
- [ ] T023 [P] Extend `frontend/src/lib/format.ts`: `DateFormat`, `DATE_FORMATS` (value, example, label), `formatDate(date, format = 'DD MMM YYYY')`, `formatDateTime(timestamp, format = 'DD MMM YYYY')`, `parseDate(text, format)`; tests in `frontend/src/lib/format.test.ts` (each format; 1-digit parts; month names in any case; 31/02/2026 → null; 001's cases unchanged) (FR-024)
- [X] T024 [P] Write `frontend/src/features/people/OwnerImpactDialog.tsx` (+ `OwnerImpactDialog.test.tsx`): the owner confirmation of the plan's *Save flow* on 001's `ConfirmDialog` — title "Some plans will need a new owner", each warning's message as a list item, **Keep editing** (focused) / **Save changes**, busy while saving — and the helper `removedIds(before, after)` that tells a form whether the impact check is needed; tests: every message listed, focus, both buttons, `removedIds` (FR-013)
- [ ] T025 [P] Extend `frontend/src/api/errors.ts`: kinds `inUse`, `notAllowed` from the new codes; optional `dateFormat` argument of `conflictMessage` and `resolvedMessage`; tests in `frontend/src/api/errors.test.ts`
- [ ] T026 [P] Add the `--team-<colour>` tokens of the plan to `frontend/src/styles/tokens.css` (light and dark)
- [ ] T027 [P] Move `FormPanel`, `FailureMessage`, `problemOf` and `SaveFailure` from `frontend/src/features/dev/SamplePanels.tsx` to `frontend/src/components/FormPanel/FormPanel.tsx` (+ `FormPanel.module.css`, `FormPanel.test.tsx`; new optional `saveLabel`); `SamplePanels.tsx` imports it from `@/components`; `SamplePanels`' tests stay green
- [ ] T028 [P] Write `frontend/src/components/SelectField/` and `frontend/src/components/TextAreaField/` (TextField's label, marking, hint and error layout; native `<select>`; `<textarea rows>`) with tests
- [ ] T029 [P] Write `frontend/src/components/FilterPills/` (group of `aria-pressed` pill buttons) and `frontend/src/components/SearchField/` (`type="search"`, icon, **Clear search**) with tests
- [ ] T030 [P] Extend `frontend/src/components/PageHeader/` with `breadcrumbs` (`<nav aria-label="Breadcrumb">`, "›" separators) and `frontend/src/components/ActionsMenu/` items with `hint` (second line, `aria-describedby`); update their tests
- [ ] T031 [P] Write `frontend/src/components/PersonLabel/` (avatar, full name, " (you)", " (inactive)" as text, optional detail line) and `frontend/src/components/TeamPill/` (`TeamPill` and `TeamPills`: colour dot from `--team-<colour>`, name, " (inactive)", link when `to` / `linked`) with tests (SC-003)
- [ ] T032 Write `frontend/src/lib/useFormatters.ts` with `frontend/src/lib/useFormatters.test.tsx` (`dateFormat`, `date`, `dateTime`, `parseDate` bound to `useSession().user`; `money` and `moneyCompact` format the `{amount, currency}` the API answered with 001's `formatMoney` / `formatMoneyCompact` and never convert (P6); defaults when signed out; plan *Dates and money for the person*) (FR-023, FR-024, SC-006)
- [ ] T033 Write `frontend/src/components/Combobox/` (`Combobox.tsx`, `.module.css`, `.test.tsx`): the ARIA 1.2 combobox of the plan — every match rendered in a scrolling list, ↓ ↑ Enter Escape Backspace, chips with "Remove <label>", visible count + polite status region, disabled with its reason; tests for each key, the count, 300 options all reachable, the disabled reason (FR-017, FR-020)
- [X] T034 Write `frontend/src/features/people/peopleApi.ts` (queries and mutations of the plan *Data hooks*: changes invalidate `['people']`, `['teams']`, `['session']`, `['activity-log']`, `['recycle-bin']`; `useCheckPersonImpact` / `useCheckTeamImpact` invalidate nothing; `useSetPersonStatus` / `useSetTeamStatus` send `{active, version}`; `useUpdateProfile` sets the session cache; `fetchLatestPerson`, `fetchLatestTeam`) and `frontend/src/features/people/texts.ts` + `texts.test.ts` (usage line, deactivate sentences, delete-team sentences, counts — plan messages table)
- [ ] T035 Export every new component from `frontend/src/components/index.ts` (`Combobox`, `PersonPicker`, `TeamPicker`, `PersonLabel`, `TeamPill`, `TeamPills`, `SelectField`, `TextAreaField`, `FilterPills`, `SearchField`, `FormPanel`, `FailureMessage`, `problemOf` and their prop types) and update `frontend/src/components/index.test.ts` (after T036 and T037)

### Frontend — User Story 3: person picker (P1) — before US1 and US2, whose forms use it

- [X] T036 [US3] Write `frontend/src/components/PersonPicker/` (on Combobox; active people only; `teamIds` mode — "Only members of the selected teams" — with qualifying team names, disabled with `disabledReason` when `teamIds` is empty, `notMemberText` flag on a held value outside those teams; signed-in person first with "(you)"; sort by name then e-mail; second line job title or e-mail, both for shared names; inactive values kept with " (inactive)" and the tip) with `PersonPicker.test.tsx`: "finance" → Sara Novak (job title), "okoro@" → Daniel Okoro (e-mail), inactive Priya never offered and labelled when held, team mode only active members with their team, no-match texts (both modes), not-member flag, 300 people → "300 people", keyboard choice and Backspace removal (FR-016 – FR-020, SC-003, SC-005)
- [X] T037 [US3] Write `frontend/src/components/TeamPicker/` (on Combobox, multiple, active teams offered, `TeamPill` chips, inactive values kept with tip) with `TeamPicker.test.tsx`
- [X] T038 [US3] Write `frontend/src/features/people/PersonField.tsx` and `TeamField.tsx` (data-bound; skeleton while loading; "People couldn't be loaded." / "Teams couldn't be loaded." with **Try again**) with tests, and `frontend/src/features/people/index.ts` exporting `PersonField`, `TeamField`, `usePeople`, `useTeams`, `PersonRef`, `TeamRef`

### Frontend — User Story 1: people directory (P1)

- [X] T039 [US1] Write `frontend/src/features/people/PeoplePage.test.tsx` covering US1 scenarios 1–8 and the list states (plan *Tests* row, including: removing a team calls `POST /api/people/{personId}/impact` and shows its warning; **Keep editing** sends no PATCH, **Save changes** sends it) with `mockApi` and `testUser()`
- [X] T040 [US1] Write `frontend/src/features/people/PeoplePage.tsx` (+ `PeoplePage.module.css`): header with breadcrumbs and **New person**, `SearchField`, `FilterPills` (default All), count, table, `ActionsMenu` with hints and no Delete (**Reactivate** → `POST …/status {active: true, version}`), states of the plan *States* table, self-deactivation message
- [X] T041 [US1] Write `frontend/src/features/people/PersonPanels.tsx` (+ `.module.css`): `PersonDetailsPanel` (teams as links, usage line, record line through `useFormatters`, footer actions, self hint with `aria-disabled`), `PersonFormPanel` (`FormPanel`, `TeamField`, edit-form rule, conflict, the plan's *Save flow*: when teams were removed `useCheckPersonImpact` first — its field problems shown like a save's — then `OwnerImpactDialog` on warnings, PATCH on **Save changes**; toasts with **View**), `DeactivateDialog` (loads usage first; Cancel focused; `POST …/status {active: false, version}`); T039 passes
- [X] T042 [US1] Add the route `/settings/people/:personId?` (`handle: { title: 'People' }`) to `frontend/src/app/router.tsx`; extend `frontend/src/app/router.test.tsx`

### Frontend — User Story 2: teams (P1)

- [X] T043 [US2] Write `frontend/src/features/people/TeamsPage.test.tsx` covering US2 scenarios 1–7 (plan *Tests* row), including removing a member → impact warning → **Save changes** sends the PATCH, **Keep editing** sends nothing; the can't-delete dialog → **Set team inactive**; delete → Undo
- [X] T044 [US2] Write `frontend/src/features/people/TeamsPage.tsx` (+ `.module.css`): header, search on name and description, filters, count, table with member avatars and "Used by", actions incl. **Delete** last and red
- [X] T045 [US2] Write `frontend/src/features/people/TeamPanels.tsx` (+ `.module.css`): `TeamDetailsPanel` (members with search above 10, plans with bin label), `TeamFormPanel` (Name, Description, Colour select with dot, Members `PersonField multiple`; the *Save flow* with `useCheckTeamImpact` and `OwnerImpactDialog`), `DeleteTeamDialog` (used vs unused texts; **Set team inactive** → `POST …/status {active: false, version}`; `useUndoableDelete({ invalidate: [['teams'], ['people'], ['session']] })`; `in_use` from the API refreshes the team and switches to the used dialog); T043 passes
- [X] T046 [US2] Add the route `/settings/teams/:teamId?` (`handle: { title: 'Teams' }`) to `frontend/src/app/router.tsx`; write `frontend/src/features/people/settingsSections.ts` (`PEOPLE_SETTINGS_SECTIONS` with title, description, path and the hooks `useActivePeopleCount`, `useActiveTeamsCount`) for feature 002's Settings page; extend `frontend/src/app/router.test.tsx`

### Frontend — User Story 4: My profile (P2)

- [ ] T047 [US4] Write `frontend/src/features/people/ProfilePage.test.tsx` covering US4 scenarios 1–5 (details and lock text; Edit details → session updated at once, empty name error; Edit preferences saves both and refetches every query; date shown in DD/MM/YYYY afterwards; no-teams sentence)
- [ ] T048 [US4] Write `frontend/src/features/people/ProfilePage.tsx` and `ProfilePanels.tsx` (+ `.module.css`): cards Your details, Display preferences, My teams; `EditDetailsPanel` (`PATCH /api/me`, conflict noun "profile"), `EditPreferencesPanel` (`useDisplayCurrencies`, `DATE_FORMATS`; afterwards `invalidateQueries()`); T047 passes
- [ ] T049 [US4] In `frontend/src/app/navigation.ts` make `MY_PROFILE` a `NavItem` and remove it from `COMING_SOON_SECTIONS`; add the route `/profile` (`ProfilePage`, `handle: { title: 'My profile' }`) to `frontend/src/app/router.tsx`; update `frontend/src/app/router.test.tsx` and `frontend/src/app/AccountMenu.test.tsx` (My profile opens the page, not Coming soon)
- [ ] T050 [US4] Show every date of feature 001's screens in the person's format: `frontend/src/features/dev/SampleRecordsPage.tsx` (`useFormatters().dateTime`), `frontend/src/features/dev/SamplePanels.tsx` (`conflictMessage(…, dateFormat)`), `frontend/src/lib/useUndoableDelete.ts` (`resolvedMessage(…, dateFormat)`), `frontend/src/features/dev/ComponentGalleryPage.tsx` (money through `useFormatters().money`); their tests stay green (FR-024, SC-006)

- [ ] T051 Frontend checkpoint in `frontend/`: `npm run check` green (API types, type check, lint, format, unit and component tests)

**Checkpoint**: every screen of feature 003 works against mocked answers.

---

## Phase 4: End-to-end (needs Phases 2 and 3)

- [ ] T052 Update `frontend/e2e/fixtures.ts`: `signInAs` resets all four preferences (`theme: 'system'`, `sidebarCollapsed: false`, `displayCurrency: 'USD'`, `dateFormat: 'DD MMM YYYY'`); add `createPerson(page, {name, email, jobTitle?, teamIds?})` (as the signed-in person, through `POST /api/people`, using `uniqueName` for the e-mail) and `createTeam(page, {name, memberIds?})`
- [ ] T053 [US1] [US2] [US3] Write `frontend/e2e/people-and-teams.spec.ts`: the US1, US2 and US3 journeys of the plan *End-to-end* table (tests create their own people and teams; demo people are never deactivated or renamed), axe on every listed screen and overlay in light mode, no horizontal page scroll at 390 px (SC-001, SC-004, SC-005, SC-008)
- [ ] T054 [US4] Write `frontend/e2e/profile.spec.ts`: the US4 journey (a person created for the test; date format DD/MM/YYYY visible on the People page's record line; name change on the account button at once; same preferences in a second browser context) and axe on the page and both panels (SC-006, SC-008)
- [ ] T055 Run `npm run test:e2e` in `frontend/`: `smoke.spec.ts` and the two new files green

---

## Phase 5: Polish & Cross-Cutting Concerns

- [ ] T056 [P] Update `backend/README.md` (packages: people now holds teams and memberships; the hooks `UsePlans`, `AddPersonUsage`, `UseCurrencies` and the lookup functions for other packages) and `frontend/README.md` (folder `features/people`; new shared components; "format dates and money only through `useFormatters()`"; `testUser()`)
- [ ] T057 [P] Update `docs/product/design-system.md` §5 (person picker modes, team picker, PersonLabel, TeamPill and team colours) and `docs/architecture/repository-layout.md` (new files); record the plan's proposed decisions P1–P16 in `docs/sdd/decision-log.md` with the numbers the consolidation step gives them; set 003 to Done in `docs/sdd/feature-map.md`
- [ ] T058 [P] With the product owner, amend `specs/003-people-and-teams/spec.md` for P9 (gender-neutral texts) and P10 (FR-011: team colour) and note P8 in `specs/009-home-dashboard/spec.md`
- [ ] T059 Final check: `gofmt -l .`, `go vet ./...`, `go test ./...` in `backend/`; `npm run check` and `npm run test:e2e` in `frontend/`; load the seed (`go run ./cmd/seed -reset`), run the app and walk through US1–US4 with the keyboard only; compare the screens with `docs/product/design-system.md`

---

## Requirement coverage

| FR | Test tasks | Implementation tasks |
|---|---|---|
| FR-001 | T009, T039, T053 | T010, T011, T040 |
| FR-002 | T009, T011, T039 | T004, T010, T041 |
| FR-003 | T011, T053 | (001 auth) T010 |
| FR-004 | T006, T009, T039 | T006, T010, T041 |
| FR-005 | T009, T011, T039, T053 | T010, T041 |
| FR-006 | T009, T011, T039 | T010, T040, T041 |
| FR-007 | T009, T011, T053 | T010, T040, T041 |
| FR-008 | T007, T009, T011, T039 | T005, T007, T040 |
| FR-009 | T004, T031 | (001 `Initials`, `Avatar`) T031 |
| FR-010 | T012, T043, T053 | T013, T014, T044 |
| FR-011 | T009, T012, T043 | T013, T045 |
| FR-012 | T009, T012, T043, T053 | T008, T034 |
| FR-013 | T009, T011, T012, T014, T024, T039, T043 | T008, T010, T013, T024, T041, T045 |
| FR-014 | T006, T012, T043 | T013, T045 |
| FR-015 | T012, T014, T043, T053 | T007, T013, T045 |
| FR-016 – FR-020 | T015, T033, T036, T037, T038, T053 | T033, T036, T037, T038 |
| FR-021, FR-022 | T016, T018, T047, T054 | T017, T018, T048, T049 |
| FR-023 | T006, T016, T018, T032 | T006, T017, T032, T048 |
| FR-024 | T016, T023, T032, T047, T054 | T017, T023, T032, T050 |
| FR-025 | T018, T054 | T017 |
| FR-026 | T006, T016 | T006, T017 |
| FR-027 | T011 | (no permission checks) |
| FR-028 | T007, T009, T012 | T007, T008, T010, T013 |
| FR-029 | T020 | T020 |

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (T001–T002)**: first; blocks everything.
- **Backend (T003–T021)** and **Frontend (T022–T051)**: independent of each other after Setup — the frontend
  uses `mockApi` answers shaped by the contract.
- **End-to-end (T052–T055)**: needs both.
- **Polish (T056–T059)**: last; T056–T058 may start once their area is done.

### Inside the backend

T003, T004 → T005 → T006 → T007 → T008 → US1 (T009 → T010 → T011) → US2 (T012 → T013 → T014) → T015 → US4
(T016 → T017 → T018 → T019) → T020 → T021. US2 needs US1's `people.go` only for the person-side membership
tests; US4 (T016–T019) needs only the foundation and may run beside US1/US2 by another developer.

### Inside the frontend

T022 → [T023, T024, T025, T026, T027, T028, T029, T030, T031 in parallel] → T032 (after T023) → T033 →
T034 → US3 (T036, T037 → T038) → T035 → US1 (T039 → T040 → T041 → T042) → US2 (T043 → T044 → T045 → T046)
→ US4 (T047 → T048 → T049 → T050) → T051. US4 needs only the foundation and `peopleApi.ts`.

### Parallel opportunities

- Backend T003 ∥ T004; frontend T023–T031 (different files).
- Backend US4 (T016–T019) ∥ backend US1/US2; frontend US4 (T047–T050) ∥ frontend US1/US2 after US3.
- E2E T053 ∥ T054 (different files; run together by `npm run test:e2e`).

## Implementation Strategy

1. Setup → backend foundation and frontend foundation in parallel.
2. US3 (picker) first on the web, because both forms need it; US1 then US2 on both sides; demo the
   directory and teams against the seed.
3. US4 (profile, preferences, `useFormatters`) — the part later features depend on for every date and amount.
4. End-to-end, then polish. Commit after each task group: `feat(003): <what> (FR-0xx)`.

### Notes

- Never delete or rename a demo person in an end-to-end test; create one with `createPerson`.
- The `409 in_use` answer and owner warnings with content need plans: their handler tests arrive with
  feature 004 (plan *Tests*); here the service tests prove them.
- Until feature 002 is wired, the only display currency is USD. Conversion is the API's (P6):
  `TestDisplayRateOf_EURAndFallback_FR023` proves the rate here; converted amounts are proven by 004's tests.
