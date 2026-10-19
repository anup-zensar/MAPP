---

description: "Task list for feature 009 — Home Dashboard"
---

# Tasks: Home Dashboard

**Input**: Design documents from `specs/009-home-dashboard/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); features 001–008 built; feature 011's history query
(011 tasks T006, T007, T009) built — or built first by this feature's backend agent (plan *Dependencies*)

**Tests**: required — constitution V. Within each group write the tests first and see them fail, then
implement. Test names carry the FR they prove.

**Organization**: Setup, then **Backend** and **Frontend** phases that both start from the merged
contract and do not depend on each other, then End-to-end, then Polish. Stories: US1 my work at a glance
(P1), US2 my teams' plans (P1), US3 what needs me (P1), US4 my activities and recent changes (P2), US5
getting started (P1). Every text comes from the plan's *User-visible messages* table.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 … US5
- Paths are relative to the repository root.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [ ] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [ ] TC01 Drop `lib/teamColor.ts` and the `--team-1…8` tokens from T019/T040; use 003's `TeamPill` and `TeamRef.colour` (D-201).
- [ ] TC02 Replace the USD fall-back test with `TestSummary_InactiveDisplayCurrencyStaysInUse`; `unavailableCurrency` is always null (D-201).
- [ ] TC03 Reuse 008's `formatRelativeTime`/`formatWaitingTime` and 004's `recordHref`; add only missing entries (D-194, D-197).

## Phase 1: Setup

- [x] T001 Merge `specs/009-home-dashboard/contracts/openapi.yaml` into `backend/api/openapi.yaml`: the tag `Home`, the six `/api/home/*` paths, the parameter `HomeScope`, the schemas, and `homeScope` added to `Preferences` (required) and `PreferencesUpdate` (keeping the properties other features added) *(Done at consolidation, D-192.)*
- [x] T002 Regenerate `frontend/src/api/schema.d.ts` with `npm run api:types` (in `frontend/`); `npm run api:check` passes *(Done at consolidation, D-192.)*
- [ ] T003 [P] Add `homeScope: 'myTeams'` to every test fixture typed as `Preferences`/`CurrentUser` (find them with `grep -rn "sidebarCollapsed:" frontend/src frontend/e2e`)

**Checkpoint**: the contract is final; backend (Phase 2) and frontend (Phase 3) can start in parallel.

---

## Phase 2: Backend

### Foundation

- [ ] T004 [P] [US2] Add the preference `homeScope` (`myTeams` default, `everything`; other values → `validation_failed` on `homeScope` "Choose My teams or Everything."; saved only when changed; not logged) to `backend/internal/people/model.go` and `backend/internal/people/service.go`; test `TestUpdatePreferences_HomeScope_FR003` in `backend/internal/people/service_test.go`
- [ ] T005 Create `backend/internal/home/sources.go` (`Sources` and the fact types `Team`, `Plan`, `Container`, `Activity`, `Decline`, `Waiting`, `Currency`, `MissingField` exactly as plan *Inputs*) and `backend/internal/home/model.go` (`Scope` with `ScopeMyTeams`/`ScopeEverything` and `ParseScope`, and the API views `SummaryView`, `PlanRow`, `PlanPage`, `HomeAttentionReason`, `HomeAttentionItem`, `HomeAttentionPage`, `WaitingItem`, `WaitingPage`, `MyActivityRow`, `MyActivityGroup`, `MyActivitiesView`, `RecentPage` with the contract's JSON names; lists never nil)
- [ ] T006 Create `backend/internal/home/fakes_test.go` (a `fakeSources` builder: teams, plans, containers, activities, waiting list, currency, a fixed-rate `FromUSD`, a recording `Log`) and `backend/internal/home/service.go` with `NewService(src Sources, now clock.Clock)`, the effective-currency helper `display` and `plansInScope`; tests `TestPlansInScope_MyTeamsAndEverything_FR003_FR004` and `TestScope_Invalid` in `backend/internal/home/service_test.go` (depends on T005)

### US1 — My work at a glance (backend)

- [ ] T007 [US1] In `backend/internal/home/service_test.go` write `TestSummary_Counts_FR005`, `TestSummary_OwnedActivitiesCoverAllPlans_EdgeCase`, `TestSummary_BudgetUsed_FR006` (incl. 100.4 % → 100 and over budget), `TestSummary_PlansWithoutBudget_FR006`, `TestSummary_NoBudgetAtAll_FR006`, `TestSummary_TotalsConvertedOnce_FR017`, `TestSummary_InactiveCurrencyFallsBackToUSD_EdgeCase`; implement `Summary` in `backend/internal/home/service.go` (depends on T006)

### US5 — Getting started (backend)

- [ ] T008 [US5] Write `TestSummary_GettingStarted_FR020` (no team → `notInTeam`; teams without plans → `noTeamPlans`; a team plan → null) and implement `gettingStarted` in `backend/internal/home/service.go` (depends on T007)

### US2 — Plans at a glance (backend)

- [ ] T009 [US2] Write `TestPlansAtAGlance_RowFigures_FR008_FR009`, `TestPlansAtAGlance_OwnTeamsFirst_FR008`, `TestPlansAtAGlance_OrderAndLimit_FR010` and implement `PlansAtAGlance` in `backend/internal/home/service.go` (depends on T008)

### US3 — What needs me (backend)

- [ ] T010 [P] [US3] Write `backend/internal/home/attention_test.go` (`TestAttention_EachReason_FR011`, `TestAttention_OnlyOwnedRecords_FR011`, `TestAttention_OneItemAllReasons_FR012`, `TestAttention_Order_FR012`, `TestAttention_LineItemsThroughActivity_Assumption`) and implement `Attention` in `backend/internal/home/attention.go` (depends on T006)
- [ ] T011 [US3] Write `TestWaiting_SameListOldestFirst_FR013` and implement `Waiting` in `backend/internal/home/service.go` (depends on T009)

### US4 — My activities and recent changes (backend)

- [ ] T012 [US4] Write `TestMyActivities_Groups_FR014`, `TestMyActivities_UsesDisplayedStatus_EdgeCase` and implement `MyActivities` in `backend/internal/home/service.go` (depends on T011)
- [ ] T013 [US4] Write `TestRecentChanges_Scope_FR015` (record types, `PlanIDs` of the plans in scope, nil for everything, no call and an empty page for `myTeams` without plans, limit 10) and implement `RecentChanges` in `backend/internal/home/service.go` (depends on T012; uses 011's `history.LogQuery`)
- [ ] T014 [US1] Write `TestHome_LargeScope_SC004` (50 plans, 5,000 activities, each operation < 300 ms) in `backend/internal/home/service_test.go` (depends on T013, T010)

### API, wiring and seed

- [ ] T015 Write `backend/internal/home/handler.go` (`NewHandler`, `Routes(r, signedIn)`: the six `GET /api/home/*` routes; `scope` parsed with `ParseScope`, invalid → `validation_failed`) (depends on T013, T010)
- [ ] T016 Write `backend/internal/app/home_sources.go` (`homeSources(...) home.Sources` built from the services of 002–008 and 011 named in plan *Dependencies*) and wire `home.NewService` / `home.NewHandler(...).Routes` in `backend/internal/app/app.go` (depends on T015)
- [ ] T017 [US1] [US2] [US3] [US4] Write `backend/internal/home/handler_test.go` (`package home_test`, `apitest` with the seed): `TestHomeSummary_Contract`, `TestHomePlans_Contract`, `TestHomeAttention_Contract`, `TestHomeWaiting_Contract`, `TestHomeMyActivities_Contract`, `TestHomeRecentChanges_Contract`, `TestHome_InvalidScope_400`, `TestHome_EqualsPlanAndApprovalPages_SC002_FR017` (every seeded person, both scopes, against 004's plan figures, 004's list totals and 008's Waiting for me total), `TestHome_DeletedRecordsNeverCount_FR019` (depends on T016)
- [ ] T018 [US1] [US3] Write `TestSeed_HomeStory_009` in `backend/seed/seed_test.go` checking the facts of plan *Demo data* through the Home endpoints (Priya's declined activity by Daniel Okoro with a comment; an activity of Priya's whose line items exceed its planned cost by exactly 1,200.00 USD; Alex Morgan's over-allocated FY2027 campaign; an activity waiting for Daniel at level 1 of 1; the teams); add any missing fact to `backend/seed/seed.go` (for example one extra line item) (depends on T016)

**Checkpoint**: `go test ./...` passes, including 001's `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005`.

---

## Phase 3: Frontend

### Foundation (shared parts; skip a task whose part an earlier feature already added)

- [ ] T019 [P] Add `--team-1` … `--team-8` (plan *Shared parts*) to `frontend/src/styles/tokens.css` and write `frontend/src/lib/teamColor.ts` (`teamColor(id)` → `var(--team-n)`, `n = 1 + (sum of UTF-16 code units) mod 8`) with `frontend/src/lib/teamColor.test.ts` (stable, spread over the palette)
- [ ] T020 [P] Create `frontend/src/components/TeamPill/TeamPill.tsx` (+ `TeamPill.module.css`, `TeamPill.test.tsx`; props `name`, `teamId`; dot `aria-hidden`, 12 % tint, `--ink` text) and export it from `frontend/src/components/index.ts` (+ `index.test.ts`) (depends on T019)
- [ ] T021 [P] Extend `frontend/src/components/SummaryCard/SummaryCard.tsx` with optional `icon`, `detail` and `to` (link card, accessible name "<label>: <value> — <linkLabel>" via a `linkLabel` prop) and their styles; tests in `frontend/src/components/SummaryCard/SummaryCard.test.tsx` (old usage unchanged)
- [ ] T022 [P] Write `frontend/src/lib/recordLinks.ts` (`recordHref(type, id)`, routes of plan *Shared parts*) with `frontend/src/lib/recordLinks.test.ts`
- [ ] T023 [P] Add `formatRelativeTime` and `formatWaitingTime` to `frontend/src/lib/format.ts` (rules of plan *Shared parts*) with tests in `frontend/src/lib/format.test.ts` (fake timers, UTC; "yesterday" by local day)
- [ ] T024 Write `frontend/src/features/home/homeApi.ts` (`useHomeSummary(scope)`, `useHomePlans(scope)`, `useHomeAttention()`, `useHomeWaiting()`, `useHomeMyActivities()`, `useHomeRecentChanges(scope)`, `useCurrencies()`, `useSaveHomePreference()`; `staleTime: 0`, `refetchOnMount: 'always'`; keys of the plan) and `frontend/src/features/home/homeText.ts` (+ `homeText.test.ts`: card details and plurals, reason texts, waiting texts, dates line, status word and tone, recent-change sentence with field-label lower-casing and "and n more fields") (depends on T002, T023)
- [ ] T025 [P] Write `frontend/src/features/home/SectionError.tsx` (+ test: text and **Try again** calls `onRetry`) and move 001's steps into `frontend/src/features/home/GettingStartedCard.tsx` (+ test: five steps and links unchanged; `notInTeam` adds the no-team sentence with its Teams page link)

### US1 — My work at a glance (frontend)

- [ ] T026 [P] [US1] Write `frontend/src/features/home/CurrencyPicker.tsx` (+ `CurrencyPicker.test.tsx`): active currencies, value = effective currency, save `displayCurrency`, invalidate `['session']` and `['home']`, fallback notice, failed save toast with **Try again** (FR-002) (depends on T024)
- [ ] T027 [P] [US1] Write `frontend/src/features/home/SummaryCards.tsx` (+ `.module.css`, `SummaryCards.test.tsx`): the four cards with icons, values, details, tones, `ProgressBar`, links per scope, the All work fallback to `#my-activities` while All work is in `COMING_SOON_SECTIONS`, loading and `SectionError` (FR-005 – FR-007, FR-018) (depends on T021, T024, T025)
- [ ] T028 [US1] [US5] Rewrite `frontend/src/features/home/HomePage.tsx` (+ `HomePage.module.css`, `HomePage.test.tsx`): header with the picker, `GettingStartedCard` from `summary.gettingStarted`, the summary cards, the responsive grid with a slot for each later card; tests FR-001, FR-019 (refetch on return), FR-020, FR-022 (a failing query shows `SectionError` only in its card) (depends on T025 – T027)

### US2 — My teams' plans (frontend)

- [ ] T029 [US2] Add the scope switch (004's `PillFilter`, label "Scope") and its hint to `frontend/src/features/home/HomePage.tsx`; the value starts from `preferences.homeScope`, changes at once, saves `PATCH /api/me/preferences {homeScope}` (failed save → toast with **Try again**); tests in `HomePage.test.tsx` (FR-003) (depends on T028)
- [ ] T030 [P] [US2] Write `frontend/src/features/home/PlansAtAGlanceCard.tsx` (+ `.module.css`, `PlansAtAGlanceCard.test.tsx`): table, `TeamPill`s, approval and budget bars with their texts, "—" / "No budget", "No activities", **View all <n> plans**, empty states with **Switch to Everything** / **Go to Plans**, loading and `SectionError` (FR-008 – FR-010, FR-021) (depends on T020, T022, T024, T025)

### US3 — What needs me (frontend)

- [ ] T031 [P] [US3] Write `frontend/src/features/home/AttentionCard.tsx` (+ `.module.css`, `AttentionCard.test.tsx`): `LevelTag`, name link, ID, plan, one line per reason with its link and accessible name, 8 + **Show all <n>** / **Show fewer**, empty sentence, loading and `SectionError` (FR-011, FR-012, FR-021) (depends on T022, T024, T025)
- [ ] T032 [P] [US3] Write `frontend/src/features/home/WaitingCard.tsx` (+ `.module.css`, `WaitingCard.test.tsx`): rows, "Level <n> of <m>", "waiting <time>", **Review** link with its accessible name, **View all <n>**, empty sentence, loading and `SectionError` (FR-013, FR-021) (depends on T022 – T025)

### US4 — My activities and recent changes (frontend)

- [ ] T033 [P] [US4] Write `frontend/src/features/home/MyActivitiesCard.tsx` (+ `.module.css`, `MyActivitiesCard.test.tsx`): group toggles with status chips and counts (Complete collapsed), 5 rows + **Show all <n>**, dates line, planned cost, `CompletenessChip`, empty state, loading and `SectionError` (FR-014, FR-021) (depends on T022, T024, T025)
- [ ] T034 [P] [US4] Write `frontend/src/features/home/RecentChangesCard.tsx` (+ `.module.css`, `RecentChangesCard.test.tsx`): avatar, sentence, links by `recordNow.state` with "(in recycle bin)" / "(permanently deleted)", relative time with the full date in a tooltip, **View activity log**, subtitle by scope, empty sentence, loading and `SectionError` (FR-015, FR-016, FR-021) (depends on T022 – T025)
- [ ] T035 Place `PlansAtAGlanceCard`, `AttentionCard`, `WaitingCard`, `MyActivitiesCard` (`id="my-activities"`) and `RecentChangesCard` in `frontend/src/features/home/HomePage.tsx` in the plan's order and grid; extend `HomePage.test.tsx` (all cards render with mocked answers; scope change refetches the three scope cards only) (depends on T029 – T034)
- [ ] T036 [P] Show `TeamPill` and the `SummaryCard` link variant in `frontend/src/features/dev/ComponentGalleryPage.tsx` (+ its test) (depends on T020, T021)

**Checkpoint**: `npm run check` (types, lint, format, unit tests) passes.

---

## Phase 4: End-to-end (needs Phases 2 and 3)

- [ ] T037 [US1] [US2] Write `frontend/e2e/home.spec.ts` part 1: Priya — header, picker, four cards with the seed figures and their links, EUR → every amount labelled EUR and kept after reload, back to USD (SC-005); Sara — both FY2026 plans once with pills and bars, **Everything** shows FY2027 Global Brand and is kept after reload, back to My teams (SC-002 spot check)
- [ ] T038 [US3] [US4] [US5] Extend `frontend/e2e/home.spec.ts`: Priya's declined item first with "Declined by Daniel Okoro on …" and its link, the "Line items exceed the planned cost by $1,200.00" reason; Daniel — **Review** opens Approvals with the preview (SC-003); Priya changes a planned cost through 006's API and Recent changes starts with her sentence and "just now"; a new person without a team (003's API) sees Getting started with the no-team sentence and every card's empty state (SC-006); axe (WCAG 2.1 AA) in light and dark mode and a keyboard-only pass (SC-007) (depends on T037)
- [ ] T039 [P] Update `frontend/e2e/smoke.spec.ts`: the Home check for Priya expects the "Plans at a glance" card instead of "Getting started"

---

## Phase 5: Polish

- [ ] T040 [P] Document the team colours (`--team-1` … `--team-8`, derived from the team ID) in `docs/product/design-system.md` §3 and set feature 009's status in `docs/sdd/feature-map.md`
- [ ] T041 Run `go test ./...` (in `backend/`), `npm run check` and `npm run test:e2e` (in `frontend/`); fix any failure; demonstrate Home as Priya, Daniel, Sara and a new person with the seed (SC-001 walkthrough)

---

## Requirement coverage

| FR | Test tasks | Implementation tasks |
|---|---|---|
| FR-001 | T028, T037 | T028 |
| FR-002 | T007, T026, T037 | T007, T026 |
| FR-003, FR-004 | T004, T006, T029, T037 | T004, T006, T029 |
| FR-005 – FR-007 | T007, T017, T021, T027, T037 | T007, T021, T027 |
| FR-008 – FR-010 | T009, T017, T030, T037 | T009, T030 |
| FR-011, FR-012 | T010, T017, T031, T038 | T010, T031 |
| FR-013 | T011, T017, T032, T038 | T011, T032 |
| FR-014 | T012, T017, T033 | T012, T033 |
| FR-015, FR-016 | T013, T017, T034, T038 | T013, T034 |
| FR-017 | T007, T017 | T007, T016 |
| FR-018 | T023, T027, T030 | T023, T027 |
| FR-019 | T017, T028 | T024, T028 |
| FR-020 | T008, T025, T028, T038 | T008, T025, T028 |
| FR-021 | T030 – T034, T038 | T030 – T034 |
| FR-022 | T025, T028 | T025, T028, T035 |
| SC-002, SC-004 | T017, T014 | T016, T007 – T013 |
| Seed facts | T018 | T018 |

## Dependencies & Execution Order

- **Setup** (T001 → T002 → T003) first.
- **Backend** and **Frontend** start after T001/T002 and do not wait for each other.
  - Backend: T004 and T005 in parallel → T006 → T007 → T008 → T009 → T011 → T012 → T013 (all edit
    `service.go`, so in this order); T010 (`attention.go`) in parallel after T006; T014 and T015 after
    T013 and T010 → T016 → T017, T018.
  - Frontend: T019 – T023 in parallel (T020 after T019) → T024 → T025 → T026, T027 → T028 → T029; the
    cards T030 – T034 in parallel once T024/T025 are done; T035 after the cards; T036 any time after
    T020/T021.
- **End-to-end** after both phases; **Polish** last.
- Cross-feature: needs 002–008 (sources, schemas `Money`, `PlanStatus`, `MissingField`, `ActivityStatus`,
  components `PillFilter`, `ProgressBar`, `LevelTag`) and 011's `QueryLog` (T013, T016).

## Notes

- `[P]` tasks touch different files. Commit after each group, e.g. `feat(009): needs your attention card (FR-011)`.
- Keep every text identical to the plan's messages table; tests quote it.
