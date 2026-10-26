---

description: "Tasks for 026 Web-App Structure"
---

# Tasks: Web-App Structure

**Input**: Design documents from `specs/026-web-app-structure/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md)

**Tests**: behaviour does not change, so today's unit, component and end-to-end tests are the
reference and must pass after every task (FR-001, FR-017). New tests only where new code appears:
the form adapters (T028), property tests for every URL and storage parser (T029–T043) and the
first-page check (T055).

**Done after each task**: `npm run check`, `npm run build` and `npm run test:e2e` green; files moved
with `git mv`; no hand-written file the task touches over 300 non-blank lines.

## Format: `[ID] [P?] [Story] [after …] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 layout, US2 forms and outside data, US3 splitting, US4 types and test doubles
- **[after 020] / [after 022]**: touches files that 020 (Shell palette) or 022 (API v1) changes;
  start only when that feature is merged into `plan-module` and this branch is rebased (R12)

---

## Phase 1: Setup (can start now)

- [ ] T001 Add `react-hook-form` 7.91.0, `zod` 4.6.7, `@hookform/resolvers` 5.9.2 (dependencies) and `msw` 3.0.3, `openapi-msw` 2.1.0, `fast-check` 4.10.2, `eslint-plugin-import-x` 4.17.1, `eslint-plugin-jsx-a11y` 6.10.2, `axe-core` 4.14.0 (dev) to `frontend/package.json` and `package-lock.json`, with `"overrides": { "eslint-plugin-jsx-a11y": { "eslint": "$eslint" } }`; `npm ci` and `npm audit --omit=dev --audit-level=high` clean (FR-015, R11)
- [ ] T002 [P] `frontend/tsconfig.base.json` with every F5 option except `noUncheckedIndexedAccess`; `tsconfig.app.json`, `tsconfig.test.json` and `tsconfig.node.json` extend it (FR-008, R6)
- [ ] T003 [P] `frontend/bundle-budget.ts`: a Vite plugin that sums the gzip size of the entry chunk and its static imports, prints a table and fails the build over budget (420 KiB for the single chunk until Phase 7: today's 374 KiB plus the form libraries); wired in `vite.config.ts`, included in `tsconfig.node.json` (FR-013, R10)

---

## Phase 2: Folders (US1) — after 020 and 022

**Goal**: the F2 layout, one barrel per feature, an acyclic feature graph. The `@/` alias keeps
pointing at `src/` (`tsconfig*.json`, `vite.config.ts`); each move rewrites the `@/…` and relative
imports of the files it moves, in the same commit.

- [ ] T004 [US1] [after 022] `src/api/schema.d.ts` → `src/api/generated/schema.d.ts`; update the `api:types` and `api:check` scripts, the ESLint ignore, the coverage exclude in `vite.config.ts` and every `@/api/schema` import (FR-002)
- [ ] T005 [US1] [after 020] `src/styles/tokens.css`, `global.css` → `src/ui/`; the style and font imports move from `main.tsx` to `app/App.tsx`, so `main.tsx` only mounts `<App />` (FR-002)
- [ ] T006 [US1] [after 020] [after 022] `src/components/<Name>/` → `src/ui/<Name>/`; delete `components/index.ts`; a one-off codemod rewrites the 285 importers to `@/ui/<Name>/<Name>` (FR-002, FR-003, R4)
- [ ] T007 [US1] [after 022] `app/SessionProvider`, `useSession`, `RequireSignIn`, `RequirePermission`, `SessionCheckPage` (with tests and CSS) → `src/session/`; `app/PublicLayout` → `src/ui/PublicLayout/` (R2)
- [ ] T008 [US1] [after 022] Empty `src/lib/` (R2): `decimal.ts` → `format/money.ts`, `format.ts` → `format/dates.ts`, `text.ts` and `useFormatters.ts` → `format/`; `cx.ts`, `usePageTitle.ts`, `useUndoableDelete.ts`, `useUnsavedChanges.tsx`, `recordLinks.ts` (→ `record-links.ts`) → `ui/`; `access.ts`, `safeNext.ts` (→ `safe-next.ts`) → `session/`; `download.ts` → `api/`; `waterfall.ts` → `features/settings/models/`; tests move with their modules
- [ ] T009 [US1] [after 022] `src/test/` → `src/testing/`; `features/settings/shared/axeCheck.ts` → `testing/axe.ts`; fixtures used by more than one feature (the people of `approvals/testData.ts`) → `testing/`; update `setupFiles` and the coverage exclude in `vite.config.ts` and the exclude in `tsconfig.app.json` (FR-002)
- [ ] T010 [US1] [after 022] `src/api/settingsQueries.ts` → `features/settings/api.ts` (R2)
- [ ] T011 [US1] [after 020] [after 022] Dissolve `features/settings/shared/`: `ListToolbar`, `SettingsCount`, `useListFilters`, `useStatusFlow`, `useCloseAfterSave` → `ui/`; the rest into `features/settings/`; remove `settings/field-help/index.ts` (FR-002, R3)
- [ ] T012 [US1] [after 020] [after 022] Move shared pieces down (R3): `activities/ActivityStatusChip` and its status words → `ui/ActivityStatusChip/`; `activities/FormRail` and `formProgress` → `ui/`; `plans/AmountPair` → `ui/AmountPair/`; `plans/gateAction.ts` → `session/gate-action.ts`
- [ ] T013 [US1] [after 020] [after 022] Merge `features/access/` into `features/people/access/` (R3)
- [ ] T014 [US1] [after 020] [after 022] Merge `features/campaigns/` into `features/plans/campaigns/` and `features/budget-phasing/` into `features/plans/phasing/` (R3)
- [ ] T015 [US1] [after 022] `app/settings-sections.ts` assembles the Settings hub's sections (settings, people, approval workflows) and the router passes them to `SettingsPage`; the `ComingSoonSection` type moves to `features/placeholders/` (R3)
- [ ] T016 [US1] [after 020] [after 022] One `index.ts` per feature (one or two lines of purpose, named exports of what other code uses); every cross-feature import (139) and every `app/` import of a feature file (52) goes through it (FR-003)
- [ ] T017 [US1] [after 022] Data access into hooks (FR-005): `SessionProvider`, `DevOnly` → `session/api.ts`; `SignInPage` → `features/sign-in/api.ts`; `LineItemEditPage` → `line-items/api.ts`; `ExportMenu` → `all-work/api.ts`; `LinkTransactionDialog` → `financials/api.ts`; `CurrencyForm`, `FinanceBudgetItemForm`, `RenameBudgetDialog` read the latest record through a hook using `queryClient.fetchQuery`
- [ ] T018 [US1] [after 020] [after 022] F9 names, one feature per commit: `*Api.ts` → `api.ts` (a second resource `<resource>-api.ts`) and the other camelCase modules (125) → kebab-case (FR-014)

**Checkpoint**: 0 forbidden folders, 0 imports into another feature's files (SC-001).

---

## Phase 3: Lint rules (US1, US4)

- [ ] T019 [US1] [after 020] [after 022] `eslint.config.js`: `import-x` with `createNodeResolver` and the `@` alias; `no-restricted-paths` (lower layers may not import `features` or `app`; `features` may not import `app`; zones built from the `src/features/*` folder list so another feature is reachable only through `index.ts`), `no-cycle`, `order` (auto-fix the 33 findings) (FR-004, R5)
- [ ] T020 [US1] `no-restricted-imports` (`@/api/client` in `*.tsx`; `msw`, `openapi-msw`, `@/testing/*` outside tests and `src/testing/`) and `no-restricted-globals` (`fetch` outside `src/api/`) (FR-005, FR-011)
- [ ] T021 [US4] [after 020] `jsx-a11y` recommended config; fix the 48 findings without a visible change (FR-010)
- [ ] T022 [US4] [after 022] `@typescript-eslint/no-non-null-assertion`; narrow the 13 `!` in 8 files instead (FR-009)
- [ ] T023 [US4] [after 020] [after 022] Remove the 96 `as T` of app code: `instanceof` for DOM elements, parsers or type guards for unions (FR-009)
- [ ] T024 [US4] [after 022] Remove the 85 `as T` of tests and the one in `e2e/fixtures.ts`; switch on `@typescript-eslint/consistent-type-assertions` (`assertionStyle: never`), with one `eslint-disable-next-line` and its reason on the response line of `src/api/client.ts` (FR-009)

---

## Phase 4: Compiler options (US4)

- [ ] T025 [US4] [after 022] Fix the 79 `noUncheckedIndexedAccess` errors of app code (35 files), counted with `npx tsc -p tsconfig.test.json --noEmit --noUncheckedIndexedAccess` (R6)
- [ ] T026 [US4] [after 022] Add a `testing/` helper that returns a list element or fails the test with a message; fix the 159 errors in 48 test files
- [ ] T027 [US4] [after 022] Fix the 9 errors in 4 `e2e/` files; switch `noUncheckedIndexedAccess` on in `tsconfig.base.json` (FR-008, SC-003)

---

## Phase 5: Forms and outside data (US2)

**Goal**: every form on React Hook Form + Zod with today's messages and timing; URL and storage
values parsed with Zod; every file of the feature ≤ 300 lines. A feature's existing tests pass
unchanged.

- [ ] T028 [US2] [after 020] `ui/FormField/` (React Hook Form `Controller` → `value`, `onChange`, `error`), `ui/form-problems.ts` (form errors → `FieldProblem[]` for `FormErrorSummary`; a 400's `fields` → `setError`), `useUnsavedChanges` from `isDirty`; validation on Save (R7); unit tests of both adapters; split `ui/ShellAttributeFields/ShellAttributeFields.tsx` (360) (FR-006)
- [ ] T029 [P] [US2] [after 022] `session/safe-next.ts` (`?next=`) and `app/theme.ts` (`mapp.theme`) parse with Zod, with property tests; split `SessionProvider.test.tsx` (363) (FR-007)
- [ ] T030 [P] [US2] [after 020] [after 022] sign-in: `SignInPage` with `features/sign-in/schemas.ts`; split `SignInPage.test.tsx` (338)
- [ ] T031 [P] [US2] [after 020] [after 022] settings — models and activity types: `ModelForm` (352), `ActivityTypeForm`; split `ModelsPage.test.tsx` (427), `ActivityTypesPage.test.tsx` (395)
- [ ] T032 [P] [US2] [after 020] [after 022] settings — currencies and finance budget items: `CurrencyForm`, `RateDialog`, `ImportRatesPage` (318), `FinanceBudgetItemForm`, `RenameBudgetDialog`; split `FinanceBudgetItemsPage.tsx` (349) and its test (443)
- [ ] T033 [P] [US2] [after 020] [after 022] settings — Shell attributes and field help: `ShellAttributeEditorPage` (682), `OptionPanel`, `FieldHelpPanel`; list filters in the URL (`ui/useListFilters`) with Zod and a property test; split `ShellAttributesPage.tsx` (518), the attributes API module (373) and the tests over 300 (487, 446, 379)
- [ ] T034 [P] [US2] [after 020] [after 022] people: `PersonPanels` (330), `TeamPanels` (444), `ProfilePanels`; split `PeoplePage.test.tsx` (514), `TeamsPage.test.tsx` (470), `ProfilePage.test.tsx` (304)
- [ ] T035 [P] [US2] [after 020] [after 022] people/access: `RoleFormPage`, `PersonRolesPanel` (306), `RulesTab`; the access tabs' URL state; split `AccessPage.test.tsx` (350)
- [ ] T036 [P] [US2] [after 020] [after 022] plans: `PlanForm` (666, one file per section), `CopyPlanDialog`, `UnpublishDialog`; plan-list preferences (storage) and plans URL state with property tests; split `PlansPage.tsx` (357), `PlanPage.test.tsx` (394), `usePlanActions.test.tsx` (304)
- [ ] T037 [P] [US2] [after 020] [after 022] plans/campaigns and phasing: `PanelFrame`, `RecordEdit`, `NewCampaignPanel`, `NewInitiativePanel`, `CopyDialog`, `MoveDialog`, `PhasingEditor`; split `campaignRows.tsx` (314), `BudgetCost.tsx` (306), `BudgetByMonthCard.tsx` (307), `CampaignPanel.test.tsx` (534), `campaignRows.test.tsx` (305), `useCampaignTreeContent.test.tsx` (307)
- [ ] T038 [P] [US2] [after 020] [after 022] activities: `ActivityForm` (400) and `activityFormState` (361) become a schema and smaller files; `ActivityFormPage` (378), `CopyActivityDialog`, `MoveActivityDialog`, `SplitActivityDialog` (301); activity tabs in the URL with a property test; split `testData.ts` (360)
- [ ] T039 [P] [US2] [after 020] [after 022] line items: `LineItemForm` (440) and `lineItemFormState` (393) become a schema and smaller files; grid period (storage) with a property test; split `LineItemPage.tsx` (386), `lineItemText.ts` (331), `lineItemMath.ts` (330), `FinancialsSection.tsx` (311), `testData.ts` (556), `LineItemCreatePage.test.tsx` (372)
- [ ] T040 [P] [US2] [after 020] [after 022] approvals and approval workflows: `DelegationForm`, `WorkflowFormPanel` (305); the approvals tabs' URL state; split `ApprovalWorkflowsPage.tsx` (347), `messages.ts` (407), `ApprovalSection.test.tsx` (447)
- [ ] T041 [P] [US2] [after 020] [after 022] all work: `SaveViewDialog`, `ManageViewsPanel`; `viewState.ts` (595) becomes Zod schemas for URL and saved-view state with property tests; split `AllWorkPage.tsx` (316), `CalendarView.tsx` (368), `testData.ts` (632), `AllWorkPage.test.tsx` (659), `viewState.test.ts` (342), `AllWorkGrid.test.tsx` (336)
- [ ] T042 [P] [US2] [after 020] [after 022] financials and global budgets: `ImportDialog` (334), `LinkTransactionDialog`, `GlobalBudgetForm`, `AllocationDialog`; financials links in the URL with a property test; split `FinancialsPage.tsx` (356), `GlobalBudgetPage.tsx` (466), `testData.ts` (371), `FinancialsPage.test.tsx` (303)
- [ ] T043 [P] [US2] [after 020] [after 022] activity log, home, recycle bin, notifications, dev: log filters (303) and home links in the URL with property tests; split `homeText.ts` (365), `binText.ts` (318); dev forms in `SamplePanels` and `PlanModuleGallery` (491); split `SampleRecordsPage.tsx` (445), its test (729), `ComponentGalleryPage.tsx` (384)

**Checkpoint**: 38/38 forms with input fields on React Hook Form + Zod (SC-005); 0 files over 300
lines (SC-007).

---

## Phase 6: Mock Service Worker (US4) — after 022

- [ ] T044 [US4] `testing/server.ts`: MSW `setupServer` started by `testing/setup.ts`; `onUnhandledRequest` records the request so the test fails afterwards, as today; `mockApi()` rebuilt on `server.use()` with the same signature and frame answers — all 121 test files unchanged and green (FR-011, R9)
- [ ] T045 [US4] `testing/http.ts`: `createOpenApiHttp<paths>()` from `openapi-msw`; typed frame answers; the render helpers accept typed handlers
- [ ] T046 [P] [US4] Replace `vi.mock` with network answers in `app/NotificationBell.unread.test.tsx`, `app/Sidebar.badge.test.tsx` and `features/settings/SettingsPage.test.tsx`
- [ ] T047 [P] [US4] Replace `vi.mock` with network answers in the plans tests (`PlanPage`, `routes`, campaigns `InitiativePanel`, `campaignRows`, `routes`) and activities tests (`ActivityPage`, `routes`) (SC-006)
- [ ] T048 [P] [US4] Typed handlers instead of `mockApi` in the settings and people tests
- [ ] T049 [P] [US4] Typed handlers in the plans, activities and line-items tests
- [ ] T050 [P] [US4] Typed handlers in the approvals, approval-workflows, notifications, all-work, financials and global-budgets tests
- [ ] T051 [US4] Typed handlers in the remaining tests (home, activity log, recycle bin, dev, sign-in, `app/`, `session/`, `ui/`); delete `mockApi` and its test (SC-006)

---

## Phase 7: Per-route splitting (US3) — after 020 and 022

- [ ] T052 [US3] [after 022] Every route in `app/routes.tsx`, including today's `settingsRoutes` and `accessRoutes` (those features export pages instead); `app/router.tsx` keeps only `createBrowserRouter`; each file ≤ 300 lines (FR-012)
- [ ] T053 [US3] [after 020] [after 022] `app/pages.ts`: a `lazyPage(load, name)` helper and one lazy declaration per page, importing the feature's `index.ts`; `<Suspense>` with the page skeleton in `RootLayout` and `AppLayout`; render helpers and page tests wait for the page with `findBy…` (FR-012, R10)
- [ ] T054 [US3] `bundle-budget.ts`: chunks named after their feature; budgets 220 KiB (first page) and 80 KiB (each feature); then lower both to the measured size plus 10%, rounded up to 10 KiB, in `bundle-budget.ts` and the plan's budget table (FR-013, SC-004)
- [ ] T055 [US3] `frontend/e2e/first-page.spec.ts`: opening `/sign-in` downloads no other feature's chunk; `/plans` draws the shell before the page; a blocked plans chunk shows the page error with a reload (US3 scenarios 2–3)

---

## Phase 8: Documentation and close

- [ ] T056 [P] `frontend/README.md` (folders, form and test patterns), `docs/architecture/repository-layout.md` (frontend tree, merged features), `CLAUDE.md` (the web app is aligned) (FR-016)
- [ ] T057 [P] ADR-0010: gate 12 enforced with the recorded budget; `docs/sdd/feature-map.md` 026 row (FR-016)
- [ ] T058 Final check: `npm run check`, `npm run build`, `npm run test:coverage`, `npm run test:e2e`; SC-001–SC-008 measured as in research R1

---

## Dependencies and order

- Phase 1 needs nothing and can merge before 020 and 022.
- Phases 2–7 start when 020 and 022 are in `plan-module`; 021 does not run meanwhile (R12).
- Phase 2 runs in task order (each task rewrites imports). T019 needs T016; T020 needs T017.
- Phase 4 follows Phase 3; T027 needs T025 and T026.
- T029–T043 need T028 and can run in parallel, one feature each.
- Phase 6 follows Phase 5; T046–T050 need T045; T051 last.
- Phase 7 follows Phase 6; T054 needs T053. Phase 8 last.
- Merge each phase into `plan-module` when its checkpoint holds.

## Requirement coverage

| Requirement | Tasks |
|---|---|
| FR-001, FR-017 | every task; T058 |
| FR-002 | T004–T011 |
| FR-003 | T006, T016 |
| FR-004 | T012–T016, T019 |
| FR-005 | T017, T020 |
| FR-006 | T028, T030–T043 |
| FR-007 | T029, T033, T035–T043 |
| FR-008 | T002, T025–T027 |
| FR-009 | T022–T024 |
| FR-010 | T021 |
| FR-011 | T020, T044–T051 |
| FR-012 | T052, T053 |
| FR-013 | T003, T054 |
| FR-014 | T018, T028–T043, T052 |
| FR-015 | T001 |
| FR-016 | T056, T057 |
