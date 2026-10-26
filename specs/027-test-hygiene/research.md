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
- Frontend (Vitest v8), before: statements 92.91% (12,145/13,071), **branches 85.26%
  (9,196/10,785)**, functions 91.91%, lines 93.08%. After: see R9.

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
  `onViewHistory` of `useActivityActions`.
- View-state gaps found by the review: GlobalBudgetsPage has no loading, error or forbidden test;
  ApprovalWorkflowsPage no loading or error test; WorkflowDetailsPanel no loading or non-404 error
  test. GlobalBudgetsPage's year tests would have failed from 2027 on (fixed in phase 1).
- Home gaps found by the review: FinancialsCard has no loading-state test; WaitingCard,
  RecentChangesCard and SummaryCards compute times from `Date.now()` instead of a fixed clock.
- Optional speed-ups: paste instead of typing long text where typing is not the point (about 15
  tests, 0.3–0.8 s each); fake timers past the 250–500 ms search and preview debounces (about 8
  tests); one app render for the plans and campaigns route tables.

## R9 — After phase 1

Filled in after the final runs and the pull request's CI.
