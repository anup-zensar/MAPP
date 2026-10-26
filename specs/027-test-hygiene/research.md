# Research: Test Hygiene

Measurements and decisions for [spec.md](spec.md). Rule names in R6: **R1** restates the
implementation, **R2** tests the language, a framework, a library or generated code, **R3** only
checks that a mock or callback was called, **R4** repeats a test at the same level, **R5** snapshots
a DOM tree, **R6** only raises coverage (Principle VI; GO_STANDARDS §12 adds the handler test that
repeats a service rule, counted as R4).

## R1 — Method and environment

- Local runs: a 10-core Apple-silicon laptop shared with other agents; the load average went from
  about 10 to above 100 during the work, so local times are compared only within one run or between
  runs at similar load. `go test -race -count=1 -covermode=atomic -coverprofile=… -json ./...` with
  `TEST_DATABASE_URL` (PostgreSQL 18 on 127.0.0.1); `npm run test:coverage`; `npm run test:e2e`
  with `E2E_DATABASE_URL=…/mapp_e2e_027`, ports 18780 and 14773.
- CI reference (GitHub Actions, 4 vCPU), develop run 40348935692 (the 023 merge): backend *Test*
  step **7 min 52 s**, frontend *Test* step **5 min 18 s**, *End-to-end* step **2 min 43 s**. The
  pull request's CI run gives the after values (R9).

## R2 — Backend: where the time went

**Decision**: fix the API test helper first; it is in phase 1 and every handler package uses it.

- Every package that uses `internal/platform/apitest` spent its first test building the contract
  validator: `validator.NewValidator` compiles every request, response and parameter schema of
  `openapi.yaml` before the first request ("warming" its schema cache). Measured alone:
  **12.9 s with `-race`** (0.8 s without); 20–30 s in a full run on a busy machine. 18 test binaries
  paid it: auth's first test took 20.0 s, samples' 24.6 s, apitest's 25.5 s plus 21.8 s for a second
  full validator built only to validate the document.
- Every `apitest.New` loaded the whole seed into a fresh JSON data folder: **190 ms per server with
  `-race`** (135 ms without; 1 ms without the seed). 353 call sites, several in loops.
- Fix (phase 1, `apitest`): the validator's schema cache reports every schema as held while the
  validator is built, so nothing is compiled up front; each schema is compiled when a request first
  needs it and kept. Compiling renders the shared document model, which is not safe for concurrent
  use (the race detector found it), so requests and answers are checked one at a time with the
  synchronous request check. The demo data is seeded once per binary (with and without the demo
  history) and its files are copied into each server's folder. The contract-document test calls
  `schema_validation.ValidateOpenAPIDocument` instead of building a second validator.
- Effect, same machine and load: apitest 49.9 s → 6.1 s, samples 32.0 → 8.2, auth 24.2 → 8.6,
  people 49.7 → 17.6, plans 41.5 → 20.2, lineitems 32.1 → 17.4; the sum of package times
  **955 s → 676 s (−29%)** with the helper change alone. The demo-data copy alone, measured on
  settings and people: 42.5 → 35.4 s and 23.2 → 16.6 s.
- `t.Parallel` in the phase-1 packages: app alone 25 s → 11 s; auth and samples a few seconds each.
  The contract checks are serialised, which caps the gain for request-heavy packages.
- A/B of the phase-1 packages, develop's tree and this branch run back to back with CI's flags
  (load 28 and 37): sum of package times **276.5 s → 123.7 s (−55%)** — apitest 101.0 → 16.7, app
  77.2 → 41.3, samples 43.1 → 27.9, auth 41.6 → 22.2; tests 158 → 145 top-level, 63 → 58 subtests.

Per package (seconds; "Before" and "apitest fixed" at similar load; the "Phase 1" run was taken at
a higher load — history, untouched, went from 105 to 127 s in it; R9 has the CI values):

| Package | CI develop (s) | Before (s) | apitest fixed (s) | Phase 1 (s) | Coverage before → after |
|---|---:|---:|---:|---:|---|
| internal/history | 94.1 | 106.6 | 105.0 | 126.8 | 83.2 → 83.2 |
| internal/activities | 58.5 | 75.7 | 59.6 | 64.8 | 82.1 → 82.1 |
| internal/approvals | 49.6 | 74.5 | 56.0 | 64.8 | 84.4 → 84.4 |
| internal/app | 54.8 | 71.1 | 66.7 | 27.4 | 69.7 → 69.7 |
| internal/campaigns | 41.4 | 62.3 | 38.3 | 44.7 | 86.2 → 86.2 |
| internal/financials | 50.3 | 51.8 | 46.6 | 49.7 | 90.3 → 90.3 |
| internal/settings | 41.3 | 50.7 | 36.0 | 40.3 | 83.6 → 83.6 |
| internal/platform/apitest | 58.7 | 49.9 | 6.1 | 6.0 | 73.0 → 76.2 |
| internal/people | 40.2 | 49.7 | 17.6 | 19.3 | 81.4 → 81.4 |
| internal/allwork | 54.7 | 48.2 | 51.2 | 47.5 | 91.6 → 91.6 |
| internal/plans | 34.4 | 41.5 | 20.2 | 22.0 | 87.8 → 87.8 |
| seed | 32.2 | 41.2 | 27.7 | 31.9 | 81.1 → 81.1 |
| internal/home | 45.1 | 40.2 | 32.1 | 37.8 | 90.6 → 90.6 |
| internal/budgets | 38.4 | 37.3 | 28.9 | 29.8 | 82.4 → 82.4 |
| internal/access | 34.0 | 36.4 | 21.1 | 18.7 | 82.0 → 82.0 |
| internal/lineitems | 34.4 | 32.1 | 17.4 | 19.5 | 85.7 → 85.7 |
| internal/samples | 29.0 | 32.0 | 8.2 | 7.3 | 74.5 → 74.5 |
| internal/auth | 32.9 | 24.2 | 8.6 | 8.4 | 75.2 → 75.2 |
| 17 platform and cmd packages | 1.0–1.4 each | 1.4–2.1 each | 1.5–2.1 each | 0.6–2.2 each | see R5 |
| **Sum** | **842** | **955** | **676** | **696** | total 80.2% → 80.2% |

## R3 — Web app: where the time went

**Decision**: remove what proves nothing (fewer files means fewer jsdom start-ups) and replace the
fixed waits; keep the default pool.

- Before: 361 test files, 2,323 tests, **90.5 s** with coverage on the idle laptop. Vitest's own
  breakdown: tests 36%, **environment 28%** (jsdom created 361 times, 203 s of worker time), import
  19%, setup 9%, transform 6%. Most cost is per file, not per test: 1,480 tests take under 0.1 s.
- The slowest file was the developer gallery (`ComponentGalleryPage.test.tsx`): 51 renders of the
  whole gallery, 7.0 s; now one render.
- Three fixed real waits (F10): 600 ms in All work's saved views (past the 500 ms preferences
  debounce), 50 ms in the Tooltip test and 50 ms in the record financials 404 test. Replaced with
  fake time (`vi.advanceTimersByTimeAsync`) or waiting for the region to go
  (`waitForElementToBeRemoved`). A probe confirmed that a save the debounce lets through is recorded
  before the advanced fake time returns, so the "nothing saved" assertion still bites.
- `pool: 'vmThreads'` (Vitest's suggestion against the jsdom cost) was tried: every test file that
  imports the API client failed to load in a VM context, so it is not used. `isolate: false` would
  share module state (stores, MSW server, localStorage) across files and is not used either.
- Debounced searches (250–500 ms) and `user.type` of long text add 0.3–1 s to about 20 tests; they
  are left as they are (R8), because they wait for what the user sees, not a fixed time.

## R4 — End-to-end tests

40 tests in 15 spec files, **146 s** locally (2.4 min of tests; CI step 2 min 43 s). The slowest
journey is Home for a planner, an approver and a global planner (15.5 s); every journey covers a P1
story or an acceptance scenario no component test covers end to end (sign-in, navigation, the core
plan → line item flow, approvals, the activity log and recycle bin, Settings, the 014–017 gaps,
026's first-page checks). No fixed waits (`waitForTimeout`) were found. The one overlap is 011 US5:
"Delete permanently: the activity leaves the bin for good…" (activity-log-and-recycle-bin.spec.ts)
and "US5: Delete permanently and Empty recycle bin…" (recycle-bin.spec.ts) both purge a new activity
through the same dialog; the first alone checks that the activity's page is gone and that its log
entries lose their link, the second alone empties the bin. **Decision**: no end-to-end test removed;
every P1 story keeps its journey (SC-005); merging the two US5 journeys (about 3.5 s) is left to a
change that can move the log helper they would share.

## R5 — Coverage before and after

- Backend statement coverage, total: **80.2% → 80.2%**. Per phase-1 package: apitest 73.0 → 76.2
  (the lazy cache and demo-data code are exercised), errs 97.6 → 95.1 (`Error()`'s one line, whose
  restating test went), actor 100 → no test file (its two functions are run by every handler test
  of other packages; the package's own figure is 0), record, ids, clock, auth, samples, app
  unchanged; cmd/api 14.3 → 16.3 (develop's listener change).
- Frontend (Vitest v8): statements 92.91% → 92.77% (12,145 → 12,127 of 13,071), **branches 85.26% →
  85.20% (9,196 → 9,189 of 10,785)**, functions 91.91% → 91.58%, lines 93.08% → 92.94%. Every branch
  and function no longer reached was reached only by a removed test, and none by an assertion of a
  rule: dead code (`permissionLabel`, the default `onViewHistory` of `useActivityActions`, PageHeader's
  `tag` prop, which no page passes), the developer gallery's buttons (13 removed duplicates), the
  UnpublishDialog's Cancel (its R3 test), and three branches that tests passed through without
  asserting them: TextAreaField's "1 character left" while 50 characters were typed one by one,
  FieldHelpCount's "All default" in the removed Settings routes test, PlanForm's onSave arrow. Two
  branches of ActivityEditor are newly reached (the spread note merged into ActivityEditPage).

## R6 — Removed and merged tests, by rule

Backend (phase 1):

| Package | Test | Rule | Still proven by |
|---|---|---|---|
| platform/apitest | TestResponse_CookieAndDecode | R2, R4 | net/http's ParseSetCookie and json.Unmarshal; every handler test reading a cookie or body |
| platform/errs | TestCodes_MatchTheContract | R1 | the contract's code enum and the exact error bodies of the handler tests |
| platform/errs | TestError_FoundThroughWrapping | R2 | (errors.As through %w is the standard library) |
| platform/errs | TestError_Text | R1 | — (one-line format of `Error()`) |
| platform/actor | the whole file (3 tests) | R2, R1 | auth's RequireSignIn test (the actor reaches the handler); every body with `createdBy` |
| platform/record | TestNewMeta_StartsAtVersion1 | R1 | TestTouch starts from NewMeta and checks created/updated |
| platform/record | TestMeta_FieldsAreFlattenedIntoTheRecord | R2 | every handler body with `version` and `createdBy` |
| platform/ids | TestPrefixes_UniquePerRecordType_FR027: the table of 27 constants and the count | R1 | form and uniqueness checks stay |
| auth | TestSignIn_IncorrectCredentials_FR004: the unknown e-mail case | R4 (§12) | TestSignIn_SameErrorForUnknownWrongOrInactive_FR004 (service) |
| samples | TestUndo_NameTaken_FR026 (handler) | R4 (§12) | TestUndo_NameTakenRefused_FR026 (service) |
| samples | TestRestore_RenamesWhenTheNameIsTaken_FR025 (handler) | R4 (§12) | TestRestore_NameClashGetsSuffix_FR025 (service); restore contract in people, budgets, app |
| samples | TestCreateFolder_CreatedOrRefused_FR020: the taken-name case | R4 (§12) | TestCreateFolder_Validation_FR020 (service) |
| app | TestNew_RoutesListTheOperations | R4 | TestRoutesMatchContract |
| app | TestEnvironment_EnvironmentAndDemoSignIn_FR009 | R4, merged | TestDemoMode_RoutesOnlyInLocalAndDevelopment_FR009 now checks the environment name |

Web app (the commits list each test by name):

| Area | Removed or merged | Rules |
|---|---|---|
| `ui/` design system | 6 files (DemoBanner, EmptyState, FormFooter, LevelTag, LockedNote, StatusChip) and 21 tests: Avatar initials and sizes; Button action, icon name, icon-only name, variants, LinkButton; Card actions; Logo name; PageHeader actions, tag and facts; SidePanel focus trap; SummaryCard label, value and detail; Tooltip hover and Escape; usePageTitle first title; ActionsMenu button name | R2, R6, R1, R4 |
| `api/`, `session/`, `testing/` | 2 rows of the error-kind table; the role-page label of a permission; renderWithProviders' starting address, signed-out default and router state; PublicLayout's theme click trimmed to the switch's place | R4, R1, R2 |
| `app/` | navigation.test.ts (file); root layout outlet; RequireSignIn positive case; AppLayout icon rail; TopBar menu button; NavDrawer outside click; second sign-out test merged | R1, R4, R2 |
| Home | SectionError (file); HomePage US3 and US4 card contents; GettingStartedCard no-team sentence; greeting, "View all" and link-name templates; budget-line templates trimmed | R4, R1 |
| All work | FilterMenu count and group headings | R4 |
| Plans | PlanForm: every problem at once, typed values, owner outside teams, the edit-page describe (merged into PlanCreatePage and PlanEditPage); UnpublishDialog (file); PlanTable pinned star; plan-text statuses, spreads and goal labels; TimingNote words; campaign routes' activities in the initiative panel | R4, R3, R1 |
| Activities, line items, financials | ReapprovalDialog without line items; the form's dependent-attribute lock; the hook's unused default View history; routes of line items, approval and the spread note (merged into ActivityEditPage); activity-text and line-item-text templates; one hand-picked Financials round trip | R4, R1 |
| Approvals, people, activity log | ApprovalPreviewPanel decline comment and first decision; ApprovalsTable preview from the name; two ApprovalStatusAction button tests (names folded into the submit tests); OwnerImpactDialog callbacks; display-currency label; the nine action chips; LogEntryList card headings; ActivityLogPage grouping | R4, R3, R1 |
| Approval workflows, global budgets | nothing removed; WorkflowDetailsPanel's three bare `toHaveBeenCalled()` now check the workflow passed; GlobalBudgetPage's allocation test checks the DELETE sent instead of its own fixture's id; GlobalBudgetsPage and WorkflowFormPanel's conflict test read the real clock and now run on a fixed date | R3, R1, F10 |
| Settings (second pass), sign-in | SettingDeleteDialog: 4 tests (unused record with Undo, Set inactive for a record in use, inactive in use with Close only, the API's in-use refusal) — the pages' own tests (ModelsPage.status, OptionsCard.status, ActivityTypesPage, ShellAttributesPage.actions) drive the same dialog, and the currency's Set-inactive sentence moved into CurrenciesPage's test; UsageLine: the used-by line and links (ModelsPage's details test) and "Not used yet" (prints the API's summary); SettingsPage: opening a section from its card (the settings-sections test checks every link and address). RateHistorySection read the real clock and would have failed from 1 Jan 2027; it now runs on a fixed date and checks exact dates. Long names are pasted in FinanceBudgetItemsPage.save, FieldHelpPage, SignInForm and CurrenciesPage.save | R4, R2, F10 |
| `ui/` and `format/` (second pass) | intl's list of formats (ProfilePage.preferences); Combobox's limit (ChoiceField's SC-007 test, the only user of a limit); ConfirmDialog's Cancel focus and its close on Cancel and Escape (merged into the question test and the focus-return test; Escape is Radix); SidePanel's name and content (FormPanel) and its Close button (merged into the focus-return test); CompletenessChip's list (merged into its choose-a-field test); FormSection's footer part; PersonLabel's name with initials (PersonPicker, Avatar); Toast's message area (its 4-second test); useUndoableDelete's message (its replace test). Combobox counts its 300 options without the accessibility walk, TextAreaField pastes the middle 50 characters, InfoTip waits on fake time for its 150 ms close delay | R4, R2, F10 |
| Settings, recycle bin, dev | ComponentGalleryPage: 13 detail tests and 37 single-card renders (one render left); PlanModuleGallery (file); BinRow restore/purge callbacks, search line, highlighted row; EmptyBinDialog and PurgeDialog success (merged into the page tests); WaterfallExample ⓘ tip and the Settings section list (merged) | R4, R1, R3 |

### Kept despite doubt

- TreeTable "tints the highlighted row" and LinkSummaryCard "colours the value": a CSS-module class
  is the only proof jsdom can give of a visual state the design system requires; the same for the
  right-aligned money of PlanTable, RateHistorySection and ImportRatesPage and the red of PlanCard,
  PlanTable and BudgetByMonthCard (they also assert the words).
- The error-kind table of `api/errors.test.ts` and ActivityStatusChip's six words: they copy a table,
  but the API-code-to-kind mapping and the status words are contracts TypeScript cannot check.
- ApprovalStatusAction's tests: no screen uses the component (ActivityHeader composes SubmitButton
  and the decision buttons itself), so its switch tests prove dead code; its submit tests are the only
  proof of SubmitButton's FR-010, FR-011 and D-030 behaviour. Recorded in R8.
- `testing/` harness tests (`mock-api.test.ts`, `render.test.tsx`): they guard that an unanswered
  request fails a test and that the providers are present, which the rest of the suite relies on.
- All work's page tests, the plans and campaigns routes tests and the dialogs that overlap their
  hosts' tests: each proves wiring (queries, preferences, the route table, panels over the page) that
  no smaller test covers.
- From the second passes: `settings/api.test`'s query keys (other modules refresh data by those
  literal keys) and its list reads (the only check of the year `useFinanceBudgets` sends); ImpactDialog's
  list order; ImportOptionsDialog's 1,214-row set (the scenario's own numbers); ChoiceField's,
  PersonPicker's and TeamPicker's "no match" sentences; MoneyField's typed amounts; the ThemeToggle and
  PasswordField tooltips; RuleConditionFields' level-prefixed checkbox ids.
- Every end-to-end journey (R4).

## R7 — Phase 2: the other backend packages (after 028)

Measured causes and recommendations, in order of time saved:

1. **Stress tests under the race detector.** `history` TestQueryLog_50000Entries_SC006 takes 58 s
   (idle) to 105 s (busy) and is the critical path of the whole suite; `allwork`
   TestQuery_StressYear_SC004 7–24 s; `financials` TestImport_FiveThousandRows_SC004 3–8 s and
   TestTransactions_TwentyThousand_SC004 4 s; `settings` TestImport_RowAndSizeLimits_FR031 3 s; `home`
   TestHome_EqualsPlanAndApprovalPages_SC002_FR017 6–15 s. Under `-race` they already skip their
   timing assertion, so they only re-prove results smaller tests prove. Skip them when
   `race.Enabled` (or turn them into 028's benchmarks) and run them without `-race` in their own CI
   step.
2. **The JSON test stand-in.** Every `Write` of `internal/platform/store` syncs each changed file to
   disk (`file.Sync`), which dominates the service tests' fixtures; feature 025 replaces it with
   in-memory fakes (CLAUDE.md), which removes the cost.
3. **`t.Parallel`.** Ten of 244 test files call it today. The business packages' tests each build
   their own server or fixture; making them parallel cuts each package's time roughly in half on
   a laptop (app: 25 → 11 s), less in CI where the cores are already busy.
4. **Handler tests that repeat service rules** (§12): the same review as phase 1's auth and samples,
   package by package; the large handler files (`settings/handler_test.go`, 42 servers;
   `history/handler_test.go`, 20) are the likely places.
5. **`seed`** (28–41 s): seeds a full data folder per test; share one seeded folder per binary as
   `apitest` now does.
6. `internal/platform/web` and `internal/platform/database` (moved to phase 2 by the coordinator):
   reviewed for restating tests only; both are fast.

## R8 — Follow-ups outside this feature

- Coverage gate: SonarQube's 80% new-code coverage once SonarQube exists (US4, FR-011).
- Axe checks in every component test of a screen or dialog (F10; deferred to 027 by 026): most
  screen and dialog tests have none (activities, line items, financials, plans, approvals). It adds
  tests and time, so it is a feature of its own.
- Dead code found by the review: `ApprovalStatusAction` (no screen uses it; ActivityHeader's
  decide/owner-may-approve logic of functional review A-01 has no test of its own — the Approval
  section's A-01 test covers the section), `permissionLabel` in `session/access.ts`, the default
  `onViewHistory` of `useActivityActions`, PageHeader's `tag` prop, and the placeholders' BuildingPage, BuildingPanel and
  BuildingSection (no file imports them; their tests stay until the code goes).
- View-state gaps found by the review: GlobalBudgetsPage has no loading, error or forbidden test;
  ApprovalWorkflowsPage no loading or error test; WorkflowDetailsPanel no loading or non-404 error
  test. GlobalBudgetsPage's year tests would have failed from 2027 on (fixed in phase 1).
- Home gaps found by the review: FinancialsCard has no loading-state test; WaitingCard,
  RecentChangesCard and SummaryCards compute times from `Date.now()` instead of a fixed clock.
- Optional speed-ups: paste instead of typing long text where typing is not the point (about 15
  tests, 0.3–0.8 s each); fake timers past the 250–500 ms search and preview debounces (about 8
  tests); one app render for the plans and campaigns route tables.

## R9 — After phase 1

| Suite | Before | After |
|---|---|---|
| Web-app unit and component tests | 361 files, 2,323 tests; 90.5 s on the idle laptop | 351 files, 2,142 tests (−181); 272 s at load 30 (not comparable; CI below) |
| End-to-end | 40 tests, 146 s | 40 tests, all passing; 221 s at a higher load (no test changed) |
| Backend, phase-1 packages (A/B, back to back) | 158 tests, 63 subtests; 276.5 s summed | 145 tests, 58 subtests; 123.7 s summed (−55%) |
| Backend, whole suite (similar load) | 1,591 tests; 955 s summed, 179 s wall | 1,578 tests; 676 s summed after the apitest change (−29%) |
| Backend coverage | 80.2% | 80.2% |
| Frontend branch coverage | 85.26% | 85.20% (R5) |

CI (4 vCPU, the reference of R1), pull request #12 against the last two develop runs:

| CI step | develop 40363452779 | develop 40377956625 | PR #12 (40410583981) |
|---|---|---|---|
| backend Test (`go test -race -cover ./...`) | 8 min 29 s | 5 min 52 s | **4 min 48 s** |
| frontend Test (`npm run test:coverage`) | 6 min 46 s | 7 min 00 s | **6 min 17 s** |
| End-to-end | 2 min 10 s | 2 min 32 s | 2 min 36 s |

Per package in CI (develop's 023-merge run → PR #12): apitest 58.7 → 3.5 s, auth 32.9 → 4.6, samples
29.0 → 4.9, app 54.8 → 26.0, people 40.2 → 10.7, plans 34.4 → 13.2, lineitems 34.4 → 9.1, access 34.0
→ 9.7, settings 41.3 → 23.0, budgets 38.4 → 13.2, campaigns 41.4 → 20.3, home 45.1 → 22.2, approvals
49.6 → 30.2, financials 50.3 → 28.5, activities 58.5 → 39.1, allwork 54.7 → 40.3, seed 32.2 → 16.1,
history 94.1 → 74.1 — history is now the critical path, almost all of it one stress test (R10).

## R10 — Phase 2 plan, per package (read-only review of develop at 6c10428)

Prepared before feature 028 merges; 028 changes these packages, so every entry is re-checked against
the merged code before it is applied. Times: CI after phase 1 (PR #12) and the busy-laptop run
(`after-be1`). "§12" marks a handler test that repeats a rule the service test proves; for each, another
handler test still validates the operation's answer against the contract (checked by METHOD + path).

### Cross-cutting changes (largest gain first)

1. **Stress tests under `-race`** (`if race.Enabled { t.Skip(…) }`, plus a CI step `go test -run 'SC00'`
   without `-race` for the success-criteria timings, which CI never checks today because it runs only
   `go test -race`):
   - history `TestQueryLog_50000Entries_SC006`: 58–105 s, about 60 of history's 74 s in CI. Under `-race`
     its 2-second check is off; it then only proves that a 50,000-entry Write succeeds and each of 4
     queries returns one item — every query shape is proven exactly by small tests
     (`TestQueryLog_NewestFirstBySeq_FR004`, `…FilterPerson…`, `…PlanScope…`, `…RecordWithItemsInside…`).
     It runs each query three times (12 full decodes of the log); stop once a try is under 2 s.
   - allwork `TestQuery_StressYear_SC004` (7–24 s; 4 queries × 3 over 5,000 activities),
     financials `TestImport_FiveThousandRows_SC004` (6.4 s; it also runs on the JSON store, not the
     in-memory repository its comment names) and `TestTransactions_TwentyThousand_SC004` (2.2 s),
     home `TestHome_LargeScope_SC004` (0.7 s).
2. **`t.Parallel`** in every business package: safe for all top-level tests except
   access `TestAuthorize_RefusalsInTheTechnicalLogOnly_FR039` and web
   `TestEngine_OneRequestRecordPerRequest_024_FR007` (`slog.SetDefault`), the timed SC tests above,
   and store `TestList_ARead_DecodesEachTableOnce` (a package-level counter). Subtests that share one
   fixture stay sequential; approvals' `TestNotify_Recipients_FR035` and
   `TestDecide_EndedRoundExplains_FR018` first pass the subtest's `t` to their helpers.
3. **Seeded folders per test** outside apitest: activities' `newFixture` seeds the full demo data for
   60 service tests (about 39 of the package's 62 s locally) and `seed`'s tests make about 49 full
   loads (13 of 29 s). Share one seeded folder per binary, as `apitest/demo_data.go` does (export it as
   a helper; B2 applies as in plan.md Complexity Tracking).
4. **Big data sets that a smaller one replaces**: settings `TestImport_RowAndSizeLimits_FR031` (5.2 s,
   4,999 options decoded for each of 9 checks) and `TestListOptions_5000Paged_FR018` (1.7 s): keep one
   4,999-option fixture for the two cap rules, page and refuse on about 60 options (6.9 s → about 1 s).
   Repeated reads: home's three SC-002 sweeps (14.7, 4.4, 2.5 s) compare against pages read with
   contract checks — read the comparison pages with `apitest.SkipValidation()` (they have contract
   tests of their own) and compare row figures under "everything" only; seed's
   `TestSeed_FinancialsStory_015` reads each child card twice; `TestSeed_HomeStory_009` reads three
   people twice.
5. **The JSON stand-in's `file.Sync`** on every Write dominates the service fixtures (F_FULLFSYNC on
   macOS doubles local times); feature 025's in-memory fakes remove it.
6. **Real sleeps** left: store `TestRenameWithRetry_WaitsForAFileThatIsBrieflyInUse` (300 ms) and
   `TestStore_KeepsNoFileOpen` (50 ms in a retry loop); B9 forbids them — drive them from a
   `testing/synctest` bubble, or they go with the JSON store (025).

### Removals and merges

| Package (CI s) | Remove or merge | Rule | Still proven by |
|---|---|---|---|
| history (74.1) | `TestEmptyLogAndBin_AnswerEmptyLists_FR022_FR024`; `TestListActivityLog_FiltersNewestFirst_FR022`; `TestEmptyRecycleBin_EmptyList_400`; `TestRestorePreview_UnknownAndResolved_404_409` (its BIN-9999 404 moves into `TestRestorePreview_Contract`) | R4, §12 | `TestListActivityLog_InvalidLimit_FR020`, `…_Filters_Contract`, `…EntriesAsTheContractSays_FR022`, `TestEmptyBin_NothingLeft_NoEntry`, `TestActions_AlreadyResolved_FR032`, `TestPurge_404_409` |
| allwork (40.3) | `TestEveryAllWorkOperationRequiresSignIn`; `TestGetAllWork_InvalidParameters_FR004`; `TestGetAllWork_Calendar_Contract`; `TestGetAllWork_Budget_Contract` (its `period=quarter` call into `…RangeYearlyCompare_Contract_FR034`); `TestFinance_ParseQuery_FR024` (its refusal into `TestParseQuery_AllProblemsAtOnce_FR004`) and the matching `refused` block of `TestGetAllWork_FinancialsFilterAndExport_FR024` | R4, §12 | app `TestEveryProtectedOperationRequiresSignIn_FR005`; `TestParseQuery_*`; the FR-023–FR-026 period and calendar tests |
| activities (39.1) | `TestKinds_RegisterAfterInitiative`; `TestJSONRepository_RoundTrip`; `TestCreate_PlannedCostRules_FR011/another_currency`; the 116.67 % readiness lines of `TestCreate_AttributeValuesChecked_FR016_FR017`; `TestCreate_InitiativeGone_404` folded into `TestCreate_OnlyNameAndTypeRequired_FR003_FR004` | R1, R6, R4 | history `TestRegister_RefusesBadKinds`; `TestUndo_RestoresExactly_FR031`; `TestPlannedCost_MustBePlanCurrency_FR011`; `TestUpdate_SplitNot100SavesButIsListed_FR017` |
| approvals (30.2) | 12 handler tests (`TestCreateApprovalWorkflow_201_FR002`, `…AllProblems400_FR003`, `TestGetApprovalWorkflow_404`, `TestUpdateApprovalWorkflow_OpenRoundUnchanged_FR006`, `TestDeleteApprovalWorkflow_InUse409_FR008`, `TestSubmit_Problems400_FR010`, `TestApprove_Twice409_FR018`, `TestDecline_WithoutComment400_FR016`, `TestApprove_OnBehalfOfDaniel_FR017`, `TestApprove_AlreadyApprovedAllLevel403_FR020`, `TestMarkRead_OtherPersons404_FR034`, `TestWorkflowFieldLockedWhileSubmitted_FR013`) and `TestNotify_OwnerNotSubmitter_Assumption`; rename the handler `TestRecall_FR021`, whose name clashes with the service test's | §12, R4 | the service tests of the same rules; every endpoint keeps a contract test; 39 → 27 servers |
| financials (28.5) | `TestOperations_NeedSignIn_FR028`; `TestTransactions_Contract_FR021`, `TestImport_Contract_FR004` and `TestRecord_Contract_FR015` trimmed to the translation (five unique checks of the first move into the service table first); the round-trip half of `TestJSONRepository_RoundTripAndInvariants_FR001` | R4, §12, R6 | app's sign-in sweep; `TestTransactions_FiltersCountsAndTotals_FR021`; `TestImport_CountsAddUpAndTwiceChangesNothing_FR006`; `TestRecord_DemoFiguresAndRollUp_FR012` |
| settings (23.0) | `TestActivityTypesAPI_ModelsEditedFromType_FR013`; `TestCurrenciesAPI_USDLocked_FR028`; `TestFormsAPI_Get_FR024` (its unknown-level 400 into `TestFormsAPI_Check_FR025`); the `taken` request of `TestFinanceAPI_CreateGetUpdate_FR014`; the `usd` request of `TestRatesAPI_AddUpdateDelete_FR003_FR005`; `TestJSONRepository_RoundTrip`; `TestJSONShapes_MoneyAndValues_D048`; the `Name()` line of `TestGSAPRateFile_Header_FR006` | §12, R6, R2, R1 | the service tests named in the review; each operation keeps a contract test |
| home (22.2) | `TestHomeSources_ConvertsPerPlan_FR014` (file); its P-1004 and A-1011 figures merged into `TestHomePlans_Contract` and `TestHomeMyActivities_Contract` | R4 | `TestHomeSummary_Contract` and the SC-002 sweeps |
| campaigns (20.3) | `TestGetCampaign_404`, `TestGetInitiative_404`, `TestCreateCampaign_AllProblems400`, `TestCreateCampaign_TotalsFollow_FR012`, `TestGetCampaign_Contract`, `TestGetInitiative_Contract`; `TestMoveCampaign_Refused400` (its C-9999 404 into `TestMoveCampaign_Contract`); `TestUpdateInitiative_Conflict409` (one stale PATCH into `TestUpdateInitiative_Contract`); `TestCreateCampaign_SameNameTwice_SecondRefused_EdgeCase` (merged into the name-unique test); `TestJSONRepository_RoundTrip`; `TestNoActivities_NothingBelow` (and the unused `NoActivities`) | R4, §12, R6, R1 | `TestDeleteCampaign_ContractAndUndo` and the service tests named in the review; 30 → 22 servers |
| seed (16.1) | `TestLoad_GlobalBudgetIDsComeFromTheCounters_FR040` merged into `TestLoad_IDsComeFromTheCounters_FR027`'s table | R4 | the merged table |
| plans (13.2) | `TestPlansOperations_WithoutSession_FR005`; `TestListPlans_MyTeams_FR004`; `TestPlanDefaults_CurrencyType`; `TestAttentionItem_BlocksPublishNotInJSON`; `TestJSONRepository_RoundTrip`; `TestCreate_SameNameTwice_EdgeCase` (merged into `TestCreate_NameRules_FR010`) | R4, §12, R2, R6 | app's sign-in sweep; `TestList_MyTeams_FR004`; `TestDefaults_FR006`; `TestGetPlan_ViewAnd404_FR011` |
| budgets (13.2) | — | | |
| people (10.7) | `TestGetMe_WithoutSession_FR005`; `TestPatchPreferences_NothingSent_FR011`; `TestPatchPreferences_InvalidTheme_FR014`; `TestPatchMe_VersionConflict_FR022`; `TestPatchMe_AllProblemsAtOnce_FR022`; `TestDeactivateYourself_409NotAllowed_FR006`; `TestRestoreTeam_NameTaken_FR015`; `TestKinds_Register_FR028`; `TestJSONRepository_TeamsMembershipsPreferences`; `TestRefs_PersonAndTeam` | R4, §12, R1, R6 | the auth and people service tests named in the review; 36 → 29 servers |
| access (9.7) | `TestListPermissions_TableA_FR001` (its literal group sizes into the service test); `TestListPeopleAccess_FR017`; `TestStore_RolesAssignmentsRules_FR001` (its empty-store defaults into `TestUpdateRules_FirstSaveFromDefaults_FR019`) | §12, R6 | `TestPermissions_GroupedAsTableA_FR001`; `TestListPeopleAccess_EveryoneWithRoles_FR017` |
| lineitems (9.1) | `TestJSONRepository_RoundTrip`; `TestListActivityLineItems_PlanCurrency_Contract_FR015` (its display figures into `TestBalance_PlanCurrency_FR015` first); two dead lines in `TestSpread_EvenlyRemainderLast_FR017_SC002` | R6, §12 | the service tests named in the review |
| rollup, platform | rollup `TestPhasing_None`'s literal line; date `TestCompareAndToday_D091`'s getter lines; authz `TestCatalogue_TableA_FR001`'s copied counts; web `TestRespondError_FieldsAlwaysPresent_FR020`; store `TestGetAndList_ReturnCopies` | R1, R2, R4 | errs `TestError_FieldsEncodeAsEmptyList_FR020`; the rest are the language or the standard library |

Kept despite doubt (the full lists go into the phase-2 commits): every handler test that is the only
contract-checked 4xx of its operation, even when it repeats a service rule; every scope-isolation
test (`store_postgres_test.go`); the cross-package acceptance tests (people's sign-in after create,
deactivate and reactivate; budgets' plan delete and restore; access's hooks on team delete and new
people); home's SC-002 sweeps.

### Found by the review, outside 027

- Test files over 400 non-blank lines, which must be split when phase 2 changes them: lineitems
  `service_test.go` 1,165, budgets `service_test.go` 662, allwork `fixture_test.go` 517, home
  `handler_test.go` 483, home `service_test.go` 428.
- No fuzz test exists in the backend, although B9 requires one for the GSAP importers (financials' CSV
  feed, settings' rate file and option import).
- 004 FR-032 and 005 FR-031 ("no role checks") are superseded by D-223; their tests only show that a
  context without grants is not refused — for the spec owner to decide.
- Dead code: campaigns `NoActivities`, access `AssignmentsOfPerson`.
