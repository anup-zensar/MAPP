# Tasks: Access Control and Global Budgets

> **Build state after the final merge** (`plan-module` at `a1ae8fe`, 21 October 2026): the backend, Settings ›
> Roles and access (routed in `25ea214`, only with `access.manage`), Your access, Global budgets and allocations,
> the filtered sidebar and Home's no-role state are built and ticked. **Open:** T043 (record pages do not yet show
> `NoAccess` on a 403), T044–T051 (pages still show actions the reader's role does not allow; the API refuses
> them), T061 (end-to-end journeys), T063 (product documents), T064 (full check run).

**Input**: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml)

**Prerequisites**: features 001–010 built on `plan-module`; feature 011 merged (its log and bin pages are filtered
here). Feature 012 stays deferred: its operations get their rules in the route table (T007) and its enforcement is
built with 012.

**Tests**: required (constitution V). Every test name carries its FR, e.g. `TestSetPersonRoles_LastAdministrator_FR021`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1–US7 of spec.md; foundation and polish tasks have none
- Paths are from the repository root. Toolchain: `export PATH="$HOME/tools/node:/c/Program Files/Go/bin:$PATH"`;
  write files with the editor tools; LF line endings.

## Phase 1: Setup (the shared contract)

- [x] T001 (done by the consolidation, D-298) Merge `specs/014-access-control/contracts/openapi.yaml` into `backend/api/openapi.yaml`: the 22 operations,
  parameters `RoleId`, `GlobalBudgetId`, `AllocationId`, response `Forbidden`, every new schema, the two tags; apply
  every item of the fragment's "Changes to other features' components" (Error.code `forbidden`; `"403": $ref
  Forbidden` on every operation except getHealth, getDemoPeople, signIn, signOut; the `access`, `allocation` and
  `canApprove` properties; ImpactWarning kinds). Run `npm run api:types` in `frontend/`. Add the 22 operations to
  `notYetServed` in `backend/internal/app/app_test.go`. Check: `go test ./internal/app/...` and `npm run api:check`.

## Phase 2: Backend foundation (blocks every story)

- [x] T002 [P] `backend/internal/platform/errs/errs.go`: `CodeForbidden = "forbidden"`, `Forbidden(message) *Error`
  (403); test in `errs_test.go` (FR-010).
- [x] T003 [P] `backend/internal/platform/ids/ids.go`: `PrefixRole = "RL"`, `PrefixRoleAssignment = "RA"`,
  `PrefixGlobalBudget = "GB"`, `PrefixBudgetAllocation = "BA"`.
- [x] T004 New package `backend/internal/platform/authz` (`catalogue.go`, `grants.go`, `context.go`, `check.go`,
  `messages.go`, `api.go` with `MyAccess`, `MyPermission`, `Availability`) exactly as plan *platform/authz*; tests
  `grants_test.go`, `check_test.go`, `messages_test.go`: union of roles, scope merge, live My teams, Chosen teams,
  `InScope` incl. the team-less plan of its creator, `CanViewPlan` incl. owner, Administrator = every permission,
  no Grants in context = unrestricted, messages 1–4 word for word (FR-001–FR-006, FR-010).
- [x] T005 `backend/internal/access`: `model.go`, `repository.go`, `store_json.go` for tables `roles`,
  `role_assignments`, `access_rules` (data-model §1.1–§1.3); `store_json_test.go`.
- [x] T006 `backend/internal/access/service.go`: `NewService`, `Directory` interface, `GrantsFor` (data-model §2.1),
  `Summary` (`authz.MyAccess`: roles, merged permissions, administrators, rule); `service_test.go` (FR-002, FR-004,
  FR-024).
- [x] T007 `backend/internal/access/routes.go` (the complete Table C of plan.md as `routeRules`, 012's rows
  included) and `middleware.go` (`Authorize`: Grants into context, `Need`/`InPlan` pre-checks, unknown pattern
  refused and logged, technical log line `forbidden` with operation and person ID only); `middleware_test.go`
  (FR-009, FR-010, FR-039).
- [x] T008 `backend/internal/app/app.go`: build the access service after people; `signedIn` = `RequireSignIn ∘
  Authorize`; `app/access_adapters.go` (`Directory` over people); new `app/access_routes_test.go`: every pattern in
  `App.Routes` has a rule and every rule pattern is served (SC-001).
- [x] T009 [P] `backend/internal/platform/apitest/apitest.go`: `(*Server).GrantRole(t, email, builtIn, scope string,
  teamNames ...string)`; test.
- [x] T010 `backend/seed/access.go` (`addAccess` after `addTeams` in `seed.Load`): five roles, six people's roles,
  access rules (plan *Demo data*); `seed/access_test.go` (FR-040).
- [x] T011 Run `go test ./...`; for every failing existing test that is not about access, grant the role it needs
  with `GrantRole` at its start (plan *Tests*, existing tests); no assertion changes. Record the list of changed
  test files in the lane report.

**Checkpoint**: the API runs with roles; every request carries Grants; nothing is scoped yet beyond `Need` rules.

## Phase 3: Backend by user story

### US1 — local and global visibility (P1)

- [x] T012 [US1] `backend/internal/plans`: `AccessFacts(Plan) authz.PlanFacts`; `List` filtered by `CanViewPlan`
  (cards and "n of m" over visible plans); `Get`, `CopyOptions`, `ChangeEffects` call `RequireView`; tests US1
  scenarios 1–3, 6 (FR-005–FR-008).
- [x] T013 [P] [US1] `backend/internal/campaigns`: `RequireView` on `ListPlanCampaigns`, `GetCampaign`,
  `GetInitiative`; destination lists keep only plans where `campaigns.edit` is in scope; tests (FR-006, FR-014).
- [x] T014 [P] [US1] `backend/internal/activities`: view rule (plan, owner, named approver through new consumer
  interface `Approvers.NamedIn(ctx, activityID) ([]string, error)`, adapter in `app/approvals_adapter.go`) on
  `Get`, `ListForPlan`; destinations filtered with `activities.edit`; tests (FR-006, FR-014).
- [x] T015 [P] [US1] `backend/internal/lineitems`: view of the activity on `Section`, `Get`; tests (FR-006).
- [x] T016 [P] [US1] `backend/internal/approvals`: `Inbox` All open filtered by activity view; `Summary` counts after
  filtering; `GetActivityApproval` and `Preview` view checks; tests (FR-007).
- [x] T017 [P] [US1] `backend/internal/app/home_sources.go`: every Home source filters plans and activities by
  visibility before counting; recent changes through the history access adapter; tests: Priya's totals equal the
  Lubricants plan only (FR-007, SC-004).
- [x] T018 [P] [US1] `backend/internal/app/allwork_source.go`: rows, totals, Budget & cost, Calendar, options, record
  location (403 for an invisible record) and export filtered; tests incl. export content (FR-007, FR-016, SC-004).
- [x] T019 [US1] `backend/internal/history`: `Access` interface and `UseAccess`; `ListLog`, the log summary, record
  History and `ListBin` filter with it; `backend/internal/app/history_access.go` implements it (D-A16: plan from the
  entry's path, else the bin snapshot, else All markets only; administration kinds need `activityLog.viewAll`);
  tests US1 scenario 5 (FR-007, FR-038).

### US2 — view versus edit (P1)

- [x] T020 [US2] `backend/internal/plans`: `Create` (`plans.create` against the chosen teams), `Update`
  (`plans.edit`), `Delete` (`plans.delete`), `Publish`/`Unpublish` (`plans.publish`), `Copy` (view source,
  `plans.create` for the copy's teams), `Defaults`; FR-013 team rules with messages 20–21 on `teamIds`;
  `Plan.access` and `PlanSummary.access` (`PlanAccess` with `content`); tests US2 scenarios 3–4 (FR-009, FR-011,
  FR-013).
- [x] T021 [P] [US2] `backend/internal/campaigns`: `campaigns.edit` on create, update, move (source and destination),
  copy (destination); `campaigns.delete` on delete; `PlanCampaigns.access`, `Campaign.access`,
  `Initiative.access`; tests (FR-009, FR-014).
- [x] T022 [P] [US2] `backend/internal/activities`: `activities.edit` on `NewForm`, `EditForm`, `Create`, `Update`,
  `Move` (both ends), `Copy` (destination); `activities.delete` on `Delete`; `Activity.access` (incl. `seeFinance`
  from `finance.view`) and `ActivitySummaryPage.access`; tests US2 scenario 2 (FR-009, FR-015).
- [x] T023 [P] [US2] `backend/internal/lineitems`: `activities.edit` on `FormOptions`, `Create`, `Update`,
  `Duplicate`; `activities.delete` on `Delete`; `LineItemSection.access`, `LineItemDetails.access`; tests (FR-009,
  FR-015).
- [x] T024 [P] [US2] `backend/internal/history` + `app/history_access.go`: `CanRestore` on Undo, Restore and the
  restore preview (level's delete permission in the plan; managing permission for administration kinds, plan
  D-A16); `RecycleBinListItem.access`; tests (FR-009).
- [x] T025 [US2] `backend/internal/app/access_matrix_test.go`: one row per non-public operation of Table C (built
  from the seed): Mei on Lubricants records → 403 for View and InPlan rows; Sara → 403 for every change operation;
  Priya → 403 for administration `Need` rows; every answer validated against the contract; 012 rows skipped while
  012 is not served (SC-001, FR-009, FR-010).

### US3 — administrators give roles (P1)

- [x] T026 [US3] `backend/internal/access/assignments.go`: `ListPeopleAccess`, `PersonRoles`, `SetPersonRoles` (whole
  set, person version via `Directory.TouchPerson`, one `changed` entry "Roles" on the person), Administrator scope
  rule (message 12), teams rule (message 16), last-administrator guard (message 10), `CheckRolesImpact` (messages
  18–19; approver data through `access.ApprovalFacts` interface implemented in `app/access_adapters.go` over
  approvals); tests US3 scenarios 2, 4, 5 (FR-017, FR-021, FR-022, FR-037).
- [x] T027 [P] [US3] `backend/internal/people`: hooks `UseAccessSummary` (`CurrentUser.access`), `NewPerson`
  (default role, same Write, logged), `CanDeactivate` (last administrator), `TeamDeleted` (team leaves every
  scope); `kinds.go` `Describe` label "Roles"; tests US3 scenario 6 (FR-020, FR-021, FR-023, FR-024).
- [x] T028 [US3] `backend/internal/access/handler.go`: `listPermissions`, `listPeopleAccess`, `getPersonRoles`,
  `setPersonRoles`, `checkPersonRolesImpact`; contract-validated handler tests; remove them from `notYetServed`
  (FR-017, FR-022).

### US4 — roles and rules (P2)

- [x] T029 [US4] `backend/internal/access/roles.go`, `kinds.go` (`role`): list, get (people who hold it, `locks`),
  create, update (changed fields, permissions logged as added/removed), delete (409 `in_use`, Administrator 409
  `not_allowed`, bin + Undo + name rule); handler `listRoles`, `createRole`, `getRole`, `updateRole`, `deleteRole`;
  tests US4 scenarios 1–5, messages 11, 13–15 (FR-018, FR-037).
- [x] T030 [P] [US4] `backend/internal/access/rules.go`, `kinds.go` (`accessRules`): get (defaults with version 0 when
  missing), update (message 17), `newPersonRoleId` cleared when its role is deleted (logged); handler
  `getAccessRules`, `updateAccessRules`; tests (FR-019, FR-023, FR-037).

### US5 — approvals follow roles (P2)

- [x] T031 [US5] `backend/internal/approvals`: approve/decline in the plan's check order (messages 5, 6a, 6b,
  on-behalf history text), `noSelfApproval` read from Grants rules (owner and submitter); `Submission`, `Submit`
  need `activities.submit`; `Recall` for the owner or submitter with `activities.submit`, or
  `approvals.decideOnBehalf`; submit advice 7 when the owner or submitter is a level's only approver;
  `actions.*.reason` from the same checks; tests US5 scenarios 1–5, 7 (FR-025–FR-027).
- [x] T032 [P] [US5] `backend/internal/approvals`: consumer interface `Approvers.CanApprove(ctx, personIDs)` + `UseApprovers`
  (adapter over `access`); workflow create/update refuse new approvers without `approvals.decide` (message 8 on
  `levels.<n>.approverIds`); `ApprovalPerson.canApprove`; `access.RoleHolders` hook for 017; tests US5 scenario 6
  (FR-028).

### US6 — global budgets (P2)

- [x] T033 [US6] New package `backend/internal/budgets`: `model.go`, `repository.go`, `store_json.go`, `kinds.go`
  (`globalBudget`; `budgetAllocation` with parent `plan` and `PrepareRestore` note 31); tests (FR-035).
- [x] T034 [US6] `backend/internal/budgets/service.go`, `figures.go`: list (year filter, `years`, `create`
  availability), get (figures, rows filtered by plan visibility, Other markets row, plans without allocation),
  create/update (messages 24–27, owner must manage All markets, year lock), delete (409 `in_use` message 30, bin +
  Undo); tests US6 scenarios 1, 5 (FR-029, FR-032, FR-033, FR-035, FR-036).
- [x] T035 [US6] `backend/internal/budgets`: `AllocationOptions`, `Allocate`, `UpdateAllocation` (messages 28–29,
  `planVersion` conflict), `RemoveAllocation`; `backend/internal/plans`: `SetBudgetFromAllocation` (write budget,
  `Touch`, log with comment); tests US6 scenarios 2, 6 and the log rows of data-model §3 (FR-030, FR-034, FR-037).
- [x] T036 [P] [US6] `backend/internal/plans`: `Allocations` hook + `UseAllocations` (default none); `Plan.allocation`;
  `Update` refuses a changed budget on an allocated plan (409 `not_allowed`, message 23); `PlanAccess.editBudget`;
  `Copy` does not copy the allocation; tests US6 scenario 3 (FR-031, FR-035). (`PlanAccess` itself comes with
  T020: its `editBudget` is `plans.edit` in scope, then unavailable with `(*Service).budgetLock`, message 23.)
- [x] T037 [US6] `backend/internal/budgets/handler.go` (9 operations), `backend/internal/app/budgets_adapter.go`
  (`Plans`, `People`, `Managers`), wiring in `app.go` (kinds registered after the plan kind), `backend/seed/budgets.go`
  (`addGlobalBudgets` after `addPlans`); contract-validated handler tests; remove from `notYetServed`; seed test
  (FR-029–FR-036, FR-040).

### US7 — audit trail (P3)

- [x] T038 [US7] `backend/internal/access/changes.go` + handler `listAccessChanges`; one test per row of data-model §3
  checking the exact entry (kind, verb, field labels, texts, comment) for roles, people's roles, rules, global
  budgets, allocations and plan budgets; access-related entries hidden from Priya in the log (FR-037, FR-038).

## Phase 4: Frontend foundation (after T001; uses `mockApi`, can run beside Phase 3)

- [x] T039 `frontend/src/api/errors.ts` (`forbidden`), `frontend/src/app/SessionProvider.tsx` and the QueryClient
  set-up: on 403 refetch the session, invalidate the page's queries, toast the message for actions and render
  `NoAccess` for page loads; refetch the session on window focus; tests (FR-008, FR-020).
- [x] T040 [P] `frontend/src/lib/access.ts`: `useAccess`, `availabilityProps`, `permissionLabel`, `summariseAccess`;
  tests (FR-011, FR-017, FR-024).
- [x] T041 [P] `frontend/src/components/ViewOnlyNote/`, `frontend/src/components/NoAccess/` (+ `index.ts` exports,
  gallery entries, tests; texts 33–36 and FR-008) (FR-008, FR-011).
- [x] T042 `frontend/src/app/RequirePermission.tsx`; `frontend/src/app/navigation.ts` (`NavItem.permission`,
  `GLOBAL_BUDGETS` after Plans), `Sidebar.tsx`, `NavDrawer.tsx` filtering; `frontend/src/features/settings/sections.ts`
  (`SettingsSection.permission`, `ACCESS_SETTINGS_SECTIONS`); router entries for the 6 new routes with stub pages;
  tests (FR-012, FR-036). *Build state (lane 014 core):* `RequirePermission`, `NavItem.permission`,
  `SettingsSection.permission` and `itemsFor` (Sidebar, so also NavDrawer, and the Settings cards) are done with
  tests; `GLOBAL_BUDGETS`, `ACCESS_SETTINGS_SECTIONS` and the routes come with the pages' lanes (T052, T057).

## Phase 5: Frontend by user story

- [ ] T043 [US1] Record pages of 004–007 and 011 render `NoAccess` on a 403 page load; Home shows the no-role state
  (text 36) when `access.roles` is empty; tests (FR-008, Edge cases).
- [ ] T044 [P] [US2] `frontend/src/features/plans/`: New plan (`plans.create`, no-team hint 22), `PlanCard` menu from
  `PlanSummary.access`, plan header and tree toolbar from `Plan.access`/`PlanCampaigns.access`, `ViewOnlyNote`,
  `PlanForm` team options limited to the permission's teams with other teams fixed; tests (FR-011, FR-013).
- [ ] T045 [P] [US2] `frontend/src/features/campaigns/`: row menus, Add buttons, record pages from `access`; tests
  (FR-011, FR-014).
- [ ] T046 [P] [US2] `frontend/src/features/activities/`: rows from `ActivitySummaryPage.access`, page actions from
  `Activity.access`, "From finance" only with `seeFinance`; tests (FR-011, FR-015).
- [ ] T047 [P] [US2] `frontend/src/features/line-items/`: section and page actions from `access`,
  `FromFinanceSection` only with `seeFinance`; tests (FR-011, FR-015).
- [ ] T048 [P] [US2] `frontend/src/features/settings/` pages: `ViewOnlyNote area="settings"` and hidden change actions
  without `settings.manage` (currencies: `exchangeRates.manage`); tests US2 scenario 5 (FR-011, FR-012).
- [ ] T049 [P] [US2] `frontend/src/features/people/` and `features/approval-workflows/`: same with `people.manage` and
  `workflows.manage`; tests (FR-011).
- [ ] T050 [P] [US2] `frontend/src/features/all-work/`: Export disabled with its reason without `data.export`; tests
  (FR-016).
- [ ] T051 [P] [US2] `frontend/src/features/recycle-bin/` and `features/activity-log/`: Restore and Delete
  permanently from `RecycleBinListItem.access`, Empty only with `recycleBin.purge`, record-type filter gains Role,
  Access rules, Global budget, Budget allocation; tests (FR-007, FR-011).
- [x] T052 [US3] `frontend/src/features/access/`: `accessApi.ts`, `AccessPage` (tabs), People tab, `PersonRolesPanel`
  (scope choice, live summary, impact warnings, save), `settingsSections.ts`; tests US3 scenarios 1–2, 4–5 (FR-017,
  FR-021, FR-022).
- [x] T053 [P] [US3] `YourAccessCard` in `frontend/src/features/access/`, shown by
  `frontend/src/features/people/ProfilePage.tsx`; person panel "Roles" line and **Change roles** link for
  `access.manage`; tests (FR-024).
- [x] T054 [US4] Roles tab, `RolePage`, `RoleForm` (grouped permissions, locks, New role empty or copy, delete with
  server reason); tests US4 scenarios 1–5 (FR-018).
- [x] T055 [P] [US4] Rules tab (toggle, role for new people, `FormPanel` with version); tests (FR-019).
- [x] T056 [US5] `frontend/src/features/approvals/`: chain flag 9 for `canApprove: false`, submit advice 7, reasons from
  `actions.*`; `features/approval-workflows/` shows the `approverIds` field error; tests (FR-025–FR-028).
- [x] T057 [US6] `frontend/src/features/global-budgets/`: `globalBudgetsApi.ts`, `GlobalBudgetsPage` (year filter,
  cards, New global budget, empty state); tests (FR-036).
- [x] T058 [US6] `GlobalBudgetPage`, `GlobalBudgetForm`, `AllocationDialog` (figures, red over-allocation, rows, Other
  markets, plans without allocation, change, remove); tests US6 scenarios 1, 2, 5, 6 (FR-029–FR-034).
- [x] T059 [P] [US6] `frontend/src/features/plans/`: `PlanAllocationNote` on the budget card, `PlanForm` budget
  read-only with `LockedNote` when allocated; tests US6 scenarios 3–4 (FR-031, FR-033).
- [x] T060 [US7] `RecentAccessChanges` card on `AccessPage`; tests US7 scenario 3 (FR-038).

## Phase 6: End-to-end (needs Phases 3 and 5)

- [ ] T061 New `frontend/e2e/access-control.spec.ts`: journeys 1–5 of plan *Tests* with an axe check on every page
  (US1–US3, US5, US6; SC-002, SC-003, SC-007).
- [x] T062 Update `frontend/e2e/core-flow.spec.ts`, `module-rest.spec.ts`, `smoke.spec.ts`: each step signs in as the
  persona whose seeded roles allow it (plan *Tests*); only actor names in expected texts change.

## Phase 7: Documents and checks

- [ ] T063 [P] Prepare, for the consolidation step (do not edit `docs/sdd/decision-log.md`): ADR-0008 "Role-based
  access with team scopes" (`docs/architecture/adr/0008-role-based-access.md`); constitution 1.1.0 wording
  (D-239, done once by D-308); `docs/product/domain-model.md` §8 rewritten as current rules and §6 rows Role, Global budget;
  `docs/product/demo-users-and-data.md` Roles column and the global budget; `docs/product/glossary.md` (Permission,
  Role, Scope, Market, Global budget, Allocation, View only); `docs/product/design-system.md` (View only note,
  no-access page); one sentence pointing to 014 in 002 FR-008, 003 FR-026–FR-027, 004 FR-004 and FR-032, 005
  FR-031, 006 and 007 Assumptions, 008 FR-017 and FR-021, 009 and 011 Assumptions (FR-041);
  `docs/sdd/plan-module-build.md` rows for 014.
- [ ] T064 Checks: backend `gofmt -l .` prints nothing, `go vet ./...`, `go test ./...`; frontend `npm run check`;
  Playwright suite (all journeys) green; tick the tasks.

## Requirement coverage

| FR | Tasks |
|---|---|
| FR-001–FR-003 | T004, T006, T010, T029 |
| FR-004–FR-005 | T004, T006, T012 |
| FR-006 | T012–T015 |
| FR-007 | T012, T016–T019, T043, T051 |
| FR-008 | T012, T039, T041, T043 |
| FR-009–FR-010 | T002, T004, T007, T008, T020–T025 |
| FR-011 | T020–T023, T040, T041, T044–T051 |
| FR-012 | T042, T048 |
| FR-013 | T020, T044 |
| FR-014 | T013, T014, T021, T045 |
| FR-015 | T022, T023, T046, T047 |
| FR-016 | T018, T050 |
| FR-017 | T026, T028, T052 |
| FR-018 | T029, T054 |
| FR-019 | T030, T055 |
| FR-020 | T027, T039 |
| FR-021 | T026, T027, T052 |
| FR-022 | T026, T028, T052 |
| FR-023 | T027, T030 |
| FR-024 | T006, T027, T040, T053 |
| FR-025–FR-027 | T031, T056 |
| FR-028 | T032, T056 |
| FR-029–FR-036 | T033–T037, T057–T059 |
| FR-037 | T026, T029, T030, T035, T038 |
| FR-038 | T019, T038, T060 |
| FR-039 | T007 |
| FR-040 | T010, T037 |
| FR-041 | T063 |
| SC-001 | T008, T025 |
| SC-002, SC-003, SC-007 | T061 |
| SC-004 | T017, T018 |

## Dependencies & Execution Order

- **T001** first. **T002–T004** next (parallel), then **T005 → T006 → T007 → T008**; T009 and T010 beside T008;
  **T011** once T008 and T010 are in.
- Backend stories after T011, in three lanes:
  - **Lane A (access)**: T026 → T028, T027 (parallel with T026), T029, T030, T038.
  - **Lane B (enforcement)**: T012 → T020; T013/T021, T014/T022, T015/T023 in parallel per package; T016–T019, T024;
    T025 last (it needs every enforcement task).
  - **Lane C (approvals and budgets)**: T031, T032; T033 → T034 → T035 → T037, T036 beside T035.
- Frontend: T039–T042 after T001 (beside the backend, with `mockApi`); then T043–T060 by story, in two lanes
  (planning screens T043–T051, T056, T059; administration and budgets T052–T055, T057, T058, T060).
- **T061–T062** after the backend and frontend of US1–US6; **T063** any time after the plan is accepted; **T064**
  last.
