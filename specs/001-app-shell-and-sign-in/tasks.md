---

description: "Task list for feature 001 — Application Shell and Demo Sign-in"
---

# Tasks: Application Shell and Demo Sign-in

**Input**: Design documents from `specs/001-app-shell-and-sign-in/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/openapi.yaml](contracts/openapi.yaml), [quickstart.md](quickstart.md)

**Tests**: required — constitution V (unit tests for every rule, a test for every acceptance scenario, an
end-to-end test for every P1 story, a contract test for every endpoint). Within each task group, write the
tests first and see them fail, then implement.

**Organization**: Setup and Foundational phases first, then one phase per user story in priority order
(US1, US2, US3, US5 are P1; US4 is P2), then Polish. Every user-visible text comes from the plan's
*User-visible messages* table; every rule from the plan, research.md and data-model.md (when they seem to
differ, the plan wins and the difference is reported).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 … US5 (user story phases only)
- Paths are relative to the repository root.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: create the two projects and the contract they share. Done while planning.

- [X] T001 Create the Go module `backend/go.mod` (module `northstar`, `go 1.27`) and the folders `backend/cmd/api`, `backend/cmd/seed`, `backend/seed`, `backend/internal/app`, `backend/internal/platform/{actor,clock,errs,web,store,ids,record,apitest}`, `backend/internal/{auth,people,history,samples}`
- [X] T002 Copy `specs/001-app-shell-and-sign-in/contracts/openapi.yaml` to `backend/api/openapi.yaml` and create `backend/api/embed.go` (`package api`, `//go:embed openapi.yaml`, `var OpenAPI []byte`)
- [X] T003 Add the test-only dependency `github.com/pb33f/libopenapi-validator@v0.16.1` (with `github.com/pb33f/libopenapi@v0.41.3`) to `backend/go.mod` / `backend/go.sum` (do not run `go mod tidy` before code imports it — T018)
- [X] T004 [P] Create `frontend/package.json` (scripts `dev`, `build` = `tsc -b tsconfig.app.json && vite build`, `preview`, `typecheck` = `tsc -b`, `lint`, `format`, `format:check`, `test`, `test:watch`, `test:e2e`, `api:types`, `api:check`, `check`) with the dependencies of research R12 (ESLint 10, no jsx-a11y plugin); `npm install` → `frontend/package-lock.json`
- [X] T005 [P] Create `frontend/tsconfig.json` (references), `frontend/tsconfig.app.json` (`src/` without tests), `frontend/tsconfig.test.json` (`src/` with tests) and `frontend/tsconfig.node.json` (`vite.config.ts`, `playwright.config.ts`, `e2e/`)
- [X] T006 [P] Create `frontend/vite.config.ts` (React plugin; alias `@`; `server`/`preview` on `127.0.0.1` ports 5173/4173 with `strictPort`; `/api` proxy to `NORTHSTAR_API_URL` or `http://127.0.0.1:8080`; Vitest `jsdom`, `include: ['src/**/*.test.{ts,tsx}']`, `setupFiles: ['src/test/setup.ts']`; `TZ=UTC` when Vitest runs)
- [X] T007 [P] Create `frontend/eslint.config.js` (ESLint 10 flat config: @eslint/js, typescript-eslint, react-hooks and react-refresh recommended; ignores `dist`, `coverage`, `test-results`, `playwright-report`, `src/api/schema.d.ts`), `frontend/.prettierrc.json`, `frontend/.prettierignore`
- [X] T008 [P] Create `frontend/index.html` (theme script in `<head>`), `frontend/public/favicon.svg`, `frontend/src/main.tsx` (imports `@/styles/tokens.css`, `@/styles/global.css` and `{ App }` from `@/app/App`; created by T021 and T038) and `frontend/src/vite-env.d.ts`
- [X] T009 Generate `frontend/src/api/schema.d.ts` with `npm run api:types`; `npm run api:check` passes

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: the shared platform every story uses. No story work starts before this phase is done.

### Backend platform (in order: T010 → T011 → T012 → T013 → T014 → T015 → T016 → T017 → T018 → T019)

- [X] T010 Write `backend/internal/platform/errs/errs.go` and `errs_test.go` as the plan's `errs` row: `Error`, `FieldError`, `Conflict`, `RecordRef`, `Resolution` with their JSON tags; the code constants of research R9 (incl. `already_resolved`, `name_taken`); every constructor with the messages-table texts and `Fields` set to an empty list; `Problems`; tests for statuses, messages and that `fields` encodes as `[]`
- [X] T011 Write `backend/internal/platform/actor/actor.go` (`Actor{ID, Name}` with JSON `id`/`name`, `System`, `With`, `From`) and `backend/internal/platform/clock/clock.go` (`type Clock func() time.Time`, `System()` = UTC truncated to milliseconds)
- [X] T012 Write `backend/internal/platform/web/web.go`, `router.go` and `web_test.go` as the plan's `web` row: `ReadJSON`, `WriteJSON`, `WriteError` (`errors.As`), `NotFound`, `Middleware`, `Wrap`, `Router` (`NewRouter`, `Handle`, `Patterns`, `ServeHTTP`); tests: cross-origin POST (`Sec-Fetch-Site: same-site`) → 400, wrong Host → 400, `text/plain` body → 400, data after the JSON → 400, unknown field → 400, panic → 500 JSON, an `*errs.Error` wrapped with `%w` keeps its status, every error body has `"fields": []`, `Cache-Control: no-store` and `X-Content-Type-Options: nosniff` on every answer, the router records patterns
- [X] T013 Write `backend/internal/platform/store/store.go` and `store_test.go` exactly as research R3 and the plan's `store` row (incl. `IsEmpty`, no file handle open between commits); tests for each behaviour: reads-own-writes (Put then Get, Delete then List in one Write), a failed Write keeps the old value, returned values are copies, reopen, joined transactions, Write inside Read → `ErrReadOnly`, parallel Writes are serialised, commit order and an injected rename failure (memory and files unchanged; `ErrFailed` afterwards), missing folder, missing table, leftover `*.tmp` removed, invalid JSON message, append-only table with a partial last line, `List` order (`P-999` before `P-1001`) and never nil, table-name check
- [X] T014 Write `backend/internal/platform/ids/ids.go`, `store_json.go` and `ids_test.go`: `Counters` interface, `NewJSONCounters()` (table `counters`), `Generator`, `NewGenerator`, `Next`, `NextNumber`; tests `TestNext_StartsAt1001_FR027`, `TestNext_TwiceInOneTransaction_FR027` (P-1001, P-1002), `TestNext_NeverReused_FR027` (after a delete and after reopening the store)
- [X] T015 Write `backend/internal/platform/record/record.go` and `record_test.go`: `Meta` (JSON `version`, `createdAt`, `createdBy`, `updatedAt`, `updatedBy`), `NewMeta` (version 1), `Touch` (version + 1, updated at/by), `CheckVersion(expected, noun)` → `errs.VersionConflict` with the API message "Not saved — <name> changed this <noun>" and `conflict {changedBy, changedAt}`
- [X] T016 Write `backend/internal/app/config.go` and `config_test.go`: `APP_ENV` (`local` default, `development`, `production`), `APP_PORT` (1–65535, default 8080), `DATA_DIR` (default `data`), `DEMO_PASSWORD` (default `northstar`, blank refused), `DemoMode()`; the messages-table texts for invalid values; no host setting
- [X] T017 Write `backend/internal/app/app.go`: `New(cfg, db, now) (*App, error)` returning `App{Handler, Routes}`; in this task it builds the `ids.Generator`, a `web.Router` with `GET /api/health` (`{status: "ok", environment, demoSignIn}`) and the `/api/` catch-all JSON 404, and `web.Middleware`; no package-level state (later tasks add packages)
- [X] T018 Write `backend/internal/platform/apitest/apitest.go` as the plan's `apitest` row **without seeding and without `SignIn`** (both added by T045): `New` with `WithoutSeed()` (accepted, no effect yet), `WithEnv`, `WithDataDir`, fixed clock and `Advance`, `Do` with request/response validation (`libopenapi.NewDocument(api.OpenAPI)`, `validator.NewValidator(doc, config.WithFormatAssertions(), config.WithoutSecurityValidation())`), options `SkipRequestValidation`, `SkipValidation`, `RawBody`, `WithCookie`, `WithHost`, `Response{Status, Header, Body}` with `Cookie` and `Decode`; then `go mod tidy`; write `backend/internal/app/app_test.go` (package `app_test`): health shape, unknown path → JSON 404 with `fields: []`, headers, `WithHost("evil.example")` refused
- [X] T019 Write `backend/cmd/api/main.go`: config (exit 1 with the message on error), `store.Open`, `app.New`, listen on `127.0.0.1:APP_PORT` (busy port: Windows error 10048 or `syscall.EADDRINUSE` → the messages-table text, exit 1), the start-up line, graceful shutdown on Ctrl+C

### Web-app foundation (in order: T020 first; T024 before T031–T032; T025 before T027; T030 before T036)

- [ ] T020 Write `frontend/src/test/setup.ts` (jest-dom via `@testing-library/jest-dom/vitest`; cleanup; stubs for `matchMedia` with add/removeEventListener, `ResizeObserver`, `Element.prototype.scrollIntoView`, `hasPointerCapture`, `releasePointerCapture`; an `afterEach` that fails the test when `mockApi` saw an unmocked request) and `frontend/src/test/mockApi.ts` (`vi.stubGlobal('fetch', …)` matching `"METHOD /path"` on the `Request`, recording request bodies and unmocked requests, `vi.unstubAllGlobals()` after each test)
- [ ] T021 [P] Write `frontend/src/styles/tokens.css` (every token of the plan's *Tokens* table, light on `:root` with `color-scheme: light`, dark on `:root[data-theme="dark"]` with `color-scheme: dark`) and `frontend/src/styles/global.css` (box-sizing, body font and colours on `--page-bg`, focus ring 2 px `--blue` with 2 px offset on `:focus-visible`, `prefers-reduced-motion`, `.visually-hidden`, links in running text underlined, `input[type=password]::-ms-reveal { display: none }`)
- [ ] T022 [P] Write `frontend/src/lib/cx.ts`; `frontend/src/lib/format.ts` + `format.test.ts` (FR-021: `formatDate('2026-09-12')` = `12 Sep 2026`, no time-zone shift for date strings, `formatDateTime`, `formatTime`, `formatMoney('12500','USD')` = `$12,500.00`, `formatMoney('12500','EUR')` = `12,500.00 EUR`, negative `-$12,500.00`, empty `—`, `formatMoneyCompact` `$1.25M` / `12.5K EUR`); `frontend/src/lib/usePageTitle.ts` + test ("<Page> · NorthStar"); `frontend/src/lib/safeNext.ts` + `safeNext.test.ts` (every case of the plan's rules table)
- [ ] T023 Write `frontend/src/api/client.ts` and `frontend/src/api/errors.ts` with `errors.test.ts` as the plan's *API client and errors* section (`call()`, `ApiError` kinds incl. `nameTaken`, `setSessionLostHandler`, `conflictMessage`, `resolvedMessage`); tests with fake timers at 2026-03-12T09:00:00Z: empty 500, text/plain 502, JSON 500, JSON 401 `session_expired`, 204, TypeError, AbortError re-thrown, conflict message today and not today, resolved message undone and restored, today and not today
- [ ] T024 Write `frontend/src/components/Button/` (`Button`, `LinkButton`; variants and shape of the plan) with test
- [ ] T025 Write `frontend/src/components/Tooltip/` and `frontend/src/components/InfoTip/` (Radix Popover: hover, focus, click, tap; Escape, blur, outside click) with tests
- [ ] T026 [P] Write `frontend/src/components/PageHeader/` with test (eyebrow, `<h1>`, highlight, description, actions)
- [ ] T027 Write `frontend/src/components/Card/` and `frontend/src/components/SummaryCard/` with tests (focusable full value when it differs)
- [ ] T028 [P] Write `frontend/src/components/StatusChip/` and `frontend/src/components/CompletenessChip/` with tests (six tones; "All filled in", "1 field needed", "3 fields needed"; popover with the fields)
- [ ] T029 [P] Write `frontend/src/components/Avatar/` and `frontend/src/components/Logo/` with tests
- [ ] T030 [P] Write `frontend/src/components/Toast/` (`ToastProvider`, `useToast.ts`: 4 s default, 10 s with an action, action `{label, altText, onClick}`, type foreground, pause on hover and focus, viewport label "Messages ({hotkey})") with test
- [ ] T031 Write `frontend/src/components/SidePanel/` and `frontend/src/components/ConfirmDialog/` with tests (Close button without tooltip; `aria-describedby={undefined}` without description; Cancel focused)
- [ ] T032 Write `frontend/src/components/EmptyState/`, `frontend/src/components/Skeleton/` (`role="status"` + visually hidden "Loading…" + `aria-hidden` shapes) and `frontend/src/components/ErrorState/` (messages by error kind; **Try again**) with tests
- [ ] T033 [P] Write `frontend/src/components/TextField/`, `frontend/src/components/PasswordField/` (eye button named "Show password" with `aria-pressed`) and `frontend/src/components/FormErrorSummary/` (messages-table title; links focus fields; takes focus) with tests
- [ ] T034 [P] Write `frontend/src/components/DemoBanner/` and `frontend/src/components/ActionsMenu/` (DropdownMenu `modal={false}`, "Actions for <label>", danger item last after a separator, dialogs open after the menu closes, focus returns to the button) with tests
- [ ] T035 Write `frontend/src/components/index.ts` exporting every shared component
- [ ] T036 Write `frontend/src/test/render.tsx`: `renderWithProviders(ui, { route = '/', user = null }) → { router, queryClient, user: userEvent.setup() }` — a new QueryClient (`retry: false`, `gcTime: Infinity`, `['session']` preset from `user`), `Tooltip.Provider` (`delayDuration={0}`), `ToastProvider`, `createMemoryRouter` mounting `ui` at `path: '*'` with a stub `/sign-in` route (T049 adds `SessionProvider`)
- [ ] T037 Write `frontend/src/app/theme.ts` and `frontend/src/app/useTheme.ts` with test (stores the preference only; resolves `system` with `matchMedia` and follows changes; sets `data-theme`)
- [ ] T038 Write `frontend/src/app/App.tsx` (QueryClientProvider → Tooltip.Provider → ToastProvider → RouterProvider), `frontend/src/app/router.tsx` (exports `routes` and `router`; root route `RootLayout` with `errorElement` `RouteErrorPage`; a temporary index route with a placeholder until T053), `frontend/src/app/RootLayout.tsx` (`ScrollRestoration`, `Outlet`; T049 adds `SessionProvider`), `frontend/src/app/RouteErrorPage.tsx` (`ErrorState` kind `other`, Try again reloads) and `frontend/src/app/PublicLayout.tsx` (skip link first, full-width banner, logo, theme-switch slot, `<main id="main" tabIndex={-1}>`)

**Checkpoint**: `go test ./...` and `npm run check` pass; the foundation is ready.

---

## Phase 3: User Story 1 — Sign in with a demo account (Priority: P1) 🎯 MVP

**Goal**: a single ZenReach-style sign-in page; typed or one-click demo sign-in; sessions that survive
reloads and new tabs; Home greets the person.

**Independent test**: start with the seed data, sign in as each demo person in turn (a new private window
for each), and confirm Home greets that person by name.

### Tests for User Story 1

- [X] T039 [P] [US1] Write `backend/internal/people/model_test.go` (`TestPerson_FirstNameAndInitials_FR001`: "Priya Raman" → "PR"/"Priya", "Mary Ann Smith" → "MA", "Cher" → "CH") and `backend/internal/people/service_test.go` (package `people`; never imports `northstar/seed` or apitest; adds people through the repository): `TestActivePeople_ExcludesInactive_FR001` (sorted by name), `CurrentUser` with default preferences, `TestPreferences_Defaults_FR011_FR014`, `TestUpdatePreferences_ChangesOnlySentFields_FR011`, `TestUpdatePreferences_InvalidTheme_FR014` (field `theme`, "Choose system, light or dark.")
- [X] T040 [P] [US1] Write `backend/internal/auth/service_test.go` with every auth rule of the plan's rules table, using a fixed clock and a temporary store; boundaries: `lastUsedAt` is rewritten at exactly 1 minute; a session is accepted 12 h 1 min after the last request and refused (and deleted) after more than 12 h 1 min

### Implementation for User Story 1

- [X] T041 [US1] Write `backend/internal/people/model.go` (stored `Person{ID, Name 1–120 chars, Email ≤ 254 lower case unique, JobTitle 0–120, Active, record.Meta}`, stored `Preferences{PersonID, Theme system|light|dark, SidebarCollapsed}`, API types `CurrentUser`, `DemoPerson`, `PreferencesView`, `FirstName()`, `Initials()` per data-model §1–§2), `repository.go`, `store_json.go` (tables `people`, `preferences`) and `service.go` (`ActivePeople`, `CurrentUser`, `UpdatePreferences` with `PreferencesUpdate{Theme *string; SidebarCollapsed *bool}` in one Write)
- [X] T042 [US1] Write `backend/seed/seed.go` (`Load(ctx, db, now)`: the six demo people of data-model §1 in order, through `people.NewJSONRepository()` and an `ids.Generator` in one Write; `ErrNotEmpty` when `db.IsEmpty()` is false) and `backend/seed/seed_test.go` (six active people with the documented e-mails and titles; next id `U-1007`; `createdAt` = the given time; refuses a store with data)
- [X] T043 [US1] Write `backend/internal/auth/model.go` (`Session{ID = hex SHA-256 of the token, PersonID, CreatedAt, LastUsedAt}`, `CookieName = "northstar_session"`, idle limit 12 h 1 min, touch interval 1 min), `repository.go`, `store_json.go` (table `sessions`), `service.go` (`DemoPeople`, `SignIn` with old-token removal and expired-session pruning of every person, `SignOut`, `Authenticate` in one Write) and `middleware.go` (`RequireSignIn`: fresh context with the actor and no transaction; demo mode off → 401 `sign_in_required` always)
- [X] T044 [US1] Write `backend/internal/auth/handler.go` (`GET /api/auth/demo-people` and `POST /api/auth/sign-in` only in demo mode; `POST /api/auth/sign-out` always, 204; cookie `HttpOnly`, `SameSite=Strict`, `Path=/`, `Secure` under TLS; cleared with `MaxAge: -1`) and `backend/internal/people/handler.go` (`GET /api/me`, `PATCH /api/me/preferences`); wire people and auth into `backend/internal/app/app.go` with the constructors of the plan
- [X] T045 [US1] Add default seeding (`seed.Load` unless `WithoutSeed()`) and `(*Server).SignIn(t, email) *Client` to `backend/internal/platform/apitest/apitest.go`; write the contract tests `backend/internal/auth/handler_test.go` (package `auth_test`: demo-people seeded and `WithoutSeed()` → `people: []` with the password `TestDemoPeople_EmptyStore_FR028`; sign-in 200 + cookie; 400 with both fields; 401 `incorrect_credentials`; sign-out 204 with and without a session; `TestRequireSignIn_HandlerCanWrite_FR007`) and `backend/internal/people/handler_test.go` (package `people_test`: `GET /api/me` 200, 401 `sign_in_required`, 401 `session_expired` after `Advance(12*time.Hour + 2*time.Minute)`; `PATCH /api/me/preferences`: `{}` → 200 unchanged, `{"theme":"dark"}` keeps `sidebarCollapsed`, invalid theme → 400 with `SkipRequestValidation()`)
- [X] T046 [US1] Extend `backend/internal/app/app_test.go`: `TestDemoMode_RoutesOnlyInLocalAndDevelopment_FR009` (404 for both demo routes and `demoSignIn: false` from health in production) and `TestDemoModeOff_ExistingSessionRefused_FR009` (sign in with `New(t, WithDataDir(dir))`, then `New(t, WithEnv("production"), WithDataDir(dir))` with `WithCookie`)
- [X] T047 [US1] Write `backend/cmd/seed/main.go` (logic in `run(args []string, getenv func(string) string, out io.Writer) error`: `-data`, `-reset` deleting only `*.json`, `*.jsonl`, `*.tmp`; creates a missing folder; refuses with the messages-table texts when the folder has data, when `APP_ENV=production`, or when `GET http://127.0.0.1:<APP_PORT>/api/health` answers within 1 second; prints the loaded line) and `backend/cmd/seed/main_test.go`
- [ ] T048 [P] [US1] Write `frontend/src/app/SessionProvider.test.tsx`, `frontend/src/app/RequireSignIn.test.tsx` (loading placeholder; `unreachable` → ErrorState and Try again refetches; signed out → `/sign-in?next=` with the encoded path and query, `replace`) and `frontend/src/features/sign-in/SignInPage.test.tsx` (people listed; a click signs in and goes to `safeNext(next)`; typed sign-in; "Incorrect e-mail or password." keeps the e-mail and clears the password; both empty fields + summary focused; no-demo-people texts; demo-off sentence with form and list hidden; unreachable; notices from router state; clicks ignored while signing in; a signed-in visitor goes on; tab title)
- [ ] T049 [US1] Write `frontend/src/app/SessionProvider.tsx` and `frontend/src/app/useSession.ts` exactly as the plan's *Session* section (statuses incl. `unreachable`; `staleTime` 30 s and checks on focus; `endSession` built from `useNavigate()` and `useQueryClient()`; the API client's session-lost hook; another person in another tab → toast and Home); render `SessionProvider` in `RootLayout.tsx` and wrap `ui` with it in `frontend/src/test/render.tsx`
- [ ] T050 [US1] Write `frontend/src/features/sign-in/SignInPage.tsx` and `SignInPage.module.css` as the plan's *Sign-in page* section, with `usePageTitle('Sign in')`
- [ ] T051 [US1] Write `frontend/src/app/navigation.ts` (label, path, lucide icon and Coming-soon sentence of every section and of My profile, as the plan)
- [ ] T052 [US1] Write the first version of `frontend/src/app/AppLayout.tsx` and `frontend/src/app/Sidebar.tsx` (skip link, banner in the content column, sidebar with logo, the navigation items from `navigation.ts` and the account button with avatar + first name) and `frontend/src/features/home/HomePage.tsx` with the greeting `PageHeader` ("Welcome back, <first name>.") and `usePageTitle('Home')`
- [ ] T053 [US1] Write `frontend/src/app/RequireSignIn.tsx` (checks the session again on every navigation) and put the `/sign-in` route and the signed-in root with `/` (HomePage) in `frontend/src/app/router.tsx`, replacing the temporary index route
- [ ] T054 [US1] Write `frontend/playwright.config.ts` (as the plan's *End-to-end tests* configuration) and `frontend/e2e/fixtures.ts` (`signInAs(page, email)` through the API + preference reset, axe helper, unique-name helper)
- [ ] T055 [US1] Write `frontend/e2e/sign-in.spec.ts` (the plan's row for this file)

**Checkpoint**: US1 works on its own — sign in as every demo person and see the greeting.

---

## Phase 4: User Story 2 — Find my way around the application (Priority: P1)

**Goal**: the ZenReach layout with collapse, drawer, account menu, bell, light/dark, banner, tab titles,
Coming soon and Page not found.

**Independent test**: sign in, visit every navigation item, collapse and expand the sidebar, switch to
dark mode and back, and resize to a phone width.

- [ ] T056 [P] [US2] Write `frontend/src/app/Sidebar.test.tsx` (items in order with the More label; `aria-current="page"`; collapsed: labels visually hidden and links still named; collapse saves the preference), `frontend/src/app/NavDrawer.test.tsx` (opens; one Escape closes with focus on a footer control; closes on an item and when the window widens to 768 px; full name wrapped), `frontend/src/app/AccountMenu.test.tsx` (header label with name, e-mail, job title; 100-character name and title: ellipsis on the button, full text in the tooltip on focus; **My profile**, **Sign out**), `frontend/src/app/NotificationBell.test.tsx`, `frontend/src/app/ThemeToggle.test.tsx` (signed in → PATCH; signed out → localStorage only; failed save → toast with **Try again**; a 401 → no toast), `frontend/src/features/placeholders/ComingSoonPage.test.tsx`, `frontend/src/features/placeholders/NotFoundPage.test.tsx` (inside and outside the app layout)
- [ ] T057 [US2] Complete `frontend/src/app/AppLayout.tsx` and `Sidebar.tsx`, and write `MobileTopBar.tsx` and `NavDrawer.tsx` with their `.module.css`, exactly as the plan's *Layout* section
- [ ] T058 [US2] Write `frontend/src/app/AccountMenu.tsx` (with **Sign out** → `useSession().signOut()` and **My profile** → `/profile`), `frontend/src/app/NotificationBell.tsx` and `frontend/src/app/ThemeToggle.tsx` with their CSS (plan *Layout* and *Preferences and theme*); add the theme switch to `PublicLayout.tsx`
- [ ] T059 [US2] Write `frontend/src/features/placeholders/ComingSoonPage.tsx` and `NotFoundPage.tsx`; add every section route, `/profile` and the `*` route to `frontend/src/app/router.tsx`; call `usePageTitle` on every page
- [ ] T060 [US2] Write `frontend/e2e/navigation.spec.ts` (the plan's row for this file)

**Checkpoint**: US1 and US2 work together; every navigation item opens a page.

---

## Phase 5: User Story 3 — See a getting-started Home page (Priority: P1)

**Goal**: Home greets the person and shows the five getting-started steps.

**Independent test**: sign in as any demo person and follow each step link.

- [ ] T061 [P] [US3] Write `frontend/src/features/home/HomePage.test.tsx` (greeting with the first name; five steps in order with their links and explanations from the plan; number circles `aria-hidden`)
- [ ] T062 [US3] Complete `frontend/src/features/home/HomePage.tsx` and `HomePage.module.css` (Getting started card of the plan)
- [ ] T063 [US3] Write `frontend/e2e/home.spec.ts` (US3 scenario 1 and every step link)

**Checkpoint**: Home is useful for beginners.

---

## Phase 6: User Story 5 — Every change is recorded and every delete can be undone (Priority: P1)

**Goal**: the activity-log and recycle-bin core, proven with the developer sample records.

**Independent test**: on the sample page create, edit and delete a record; the log holds the entries
with the actor and the changed fields; Undo after the delete brings the record back unchanged.

### Tests for User Story 5

- [X] T064 [P] [US5] Write `backend/internal/history/diff_test.go` (`TestDiff_SortedByField_FR022`, `TestDiff_IgnoresMeta_FR022`, `TestDiff_NullAndEmptyAreEqual_FR022`, `TestDiff_NestedValuesAsAWhole_FR022`)
- [X] T065 [P] [US5] Write `backend/internal/history/service_test.go` with test kinds (parent, child, grandchild): `TestRegister_RefusesBadKinds`, `TestLog_StoresPathAndTexts_FR022`, `TestLog_EveryAction_FR022`, `TestLog_SameMillisecondKeepsOrder_EdgeCase`, `TestLog_TwoEntriesInOneWrite_FR022`, `TestLog_EntriesNeverChangeOrDisappear_FR023`, `TestMoveToBin_WithChildren_FR024` (path, contents only with counts ≥ 1, snapshots parents first, children removed first), `TestMoveToBin_OneNoteSaysSingular_FR024`, `TestUndo_RestoresExactly_FR026`, `TestUndo_ExpiresAfterTwoMinutes_FR026`, `TestRestore_ParentMissing_FR025`, `TestUndo_AlreadyResolved_FR025`, `TestUndo_KindNotRegistered_FR025`, `TestRestore_AppliesPrepareRestore_FR025` (notes in details, changes logged), `TestListLog_FilterLimitTotal`, `TestListBin_UnresolvedNewestFirst`
- [X] T066 [P] [US5] Write `backend/internal/samples/service_test.go`: `TestCreateFolder_Validation_FR020` (required "Enter a name.", trimmed, ≤ 80 Unicode characters "Use 80 characters or fewer.", unique ignoring case and surrounding spaces "There is already a sample folder called '<name>'. Choose another name."), `TestCreateNote_Validation_FR020` (required "Enter a title.", ≤ 120 "Use 120 characters or fewer.", missing folder → not found), `TestUpdateFolder_MissingVersion_FR029`, `TestUpdateFolder_VersionConflict_FR029`, `TestUpdateFolder_NoChangeNoLog_FR022`, `TestFolderLifecycle_OneLogEntryEach_FR022_SC004`, `TestUndo_NameTakenRefused_FR026`, `TestRestore_NameClashGetsSuffix_FR025` (" (restored)", " (restored 2)", version bumped, note in details)

### Implementation for User Story 5

- [X] T067 [US5] Write `backend/internal/history/model.go` (Entry, Change, Action, BinEntry with `Parent *RecordRef json:"parent"` (no omitempty), Snapshot, RestoreResult, LogInput, LogFilter; `RecordRef` and `Resolution` as aliases of the `errs` types; every slice non-nil), `kinds.go` (Kind, RestoreContext, registration rules), `diff.go`, `repository.go`, `store_json.go` (append-only `activity_log`, table `recycle_bin`) and `service.go` (Register, Log with path, verb and texts, MoveToBin, Undo with the 2-minute window, Restore, ListLog, ListBin) per research R8 and data-model §5–§7
- [X] T068 [US5] Write `backend/internal/history/handler.go` (`GET /api/activity-log` with `recordType`, `recordId`, `limit` 1–200 default 50 — other values → `validation_failed` on `limit`; `GET /api/recycle-bin`; `POST /api/recycle-bin/{entryId}/undo`; `POST /api/recycle-bin/{entryId}/restore`), wire history into `backend/internal/app/app.go`, and write `backend/internal/history/handler_test.go` (package `history_test`: empty lists `items: []` and `total: 0`, `limit` 400, unknown entry 404, 401s without a session)
- [X] T069 [US5] Write `backend/internal/samples/model.go` (stored `Folder{ID, Name, record.Meta}`, `Note{ID, FolderID, Title, record.Meta}`; API `FolderView` with `Notes` never nil; inputs with `Version *int`), `repository.go`, `store_json.go` (tables `sample_folders`, `sample_notes`), `service.go` (operations of the plan's samples row) and `kinds.go` (`sampleFolder` with the `PrepareRestore` name rule; `sampleNote` with parent `sampleFolder`)
- [X] T070 [US5] Write `backend/internal/samples/handler.go` (the six `/api/sample-*` operations; not-found texts of the plan) and `handler_test.go` (package `samples_test`: list with `notes: []`, create 201, update 200/400/404/409, delete with `contents: []` and with notes, note create/delete, and through `/api/recycle-bin`: undo 200, `undo_expired` after `Advance(2*time.Minute + time.Millisecond)`, `restore_parent_first`, `already_resolved`, `name_taken`, restore 200 with details); wire samples (routes and kinds) into `backend/internal/app/app.go` only in demo mode
- [X] T071 [US5] Extend `backend/internal/app/app_test.go` with `TestRoutesMatchContract` (the recorded routes and the contract's operations match in both directions) and `TestEveryProtectedOperationRequiresSignIn_FR005`
- [ ] T072 [P] [US5] Write `frontend/src/lib/useUndoableDelete.test.tsx` (text, 10 s, a new delete replaces the message, Undo calls the endpoint and shows "restored", refused-Undo messages incl. `name_taken`, invalidation), `frontend/src/lib/useUnsavedChanges.test.tsx` (blocker dialog; beforeunload) and `frontend/src/features/dev/SampleRecordsPage.test.tsx` (list and empty state; create and edit panel with all errors at once; the PATCH keeps the version the form started from after a refetch; conflict keeps the typed name, shows the message and refreshes the version; Back with typed changes shows "Discard changes?"; delete dialogs for a folder (with and without notes) and a note, through the actions menu; recently deleted restore refusal toast; activity-log entries rendered)
- [ ] T073 [US5] Write `frontend/src/lib/useUndoableDelete.ts` and `frontend/src/lib/useUnsavedChanges.ts` as the plan's *Shared hooks*
- [ ] T074 [US5] Write `frontend/src/app/DevOnly.tsx` (asks `GET /api/health`; NotFoundPage unless `demoSignIn`), `frontend/src/features/dev/sampleApi.ts` (queries and mutations with the edit-form rule) and `frontend/src/features/dev/SampleRecordsPage.tsx` + `SampleRecordsPage.module.css` as the plan's *Developer pages*; add `/dev/sample-records` (inside `DevOnly`) to `frontend/src/app/router.tsx`
- [ ] T075 [US5] Write `frontend/e2e/history.spec.ts` (the plan's row for this file)

**Checkpoint**: every change is logged once and every delete can be undone; the core is ready for 002–012.

---

## Phase 7: User Story 4 — Manage my session (Priority: P2)

**Goal**: account menu with sign out; a lost session sends the person to sign in again and back.

**Independent test**: sign in as one person, sign out, sign in as another and confirm the name changes.

- [ ] T076 [P] [US4] Extend `frontend/src/app/SessionProvider.test.tsx` and `frontend/src/app/AccountMenu.test.tsx`: sign out → exactly one notice "You have been signed out." and `/sign-in` without `next`; several 401s at once → one redirect with "Please sign in again" and the encoded `next`; a reload after expiry (`GET /api/me` → `session_expired`) shows "Please sign in again"; another person in another tab → toast and Home
- [ ] T077 [US4] Make the tests of T076 pass, changing `frontend/src/app/SessionProvider.tsx`, `useSession.ts` or `AccountMenu.tsx` only where they fail
- [ ] T078 [US4] Write `frontend/e2e/session.spec.ts` (the plan's row for this file)

**Checkpoint**: all five stories work.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T079 [P] Write `frontend/src/features/dev/ComponentGalleryPage.tsx` (+ CSS and test) as the plan's *Developer pages*; add `/dev/components` (inside `DevOnly`) to `frontend/src/app/router.tsx`
- [ ] T080 Write `frontend/e2e/accessibility.spec.ts` (every state and keyboard journey in the plan's row for this file)
- [ ] T081 [P] Write the documentation: `README.md` (what NorthStar is, prerequisites, run, test, URLs, demo password, developer pages), `backend/README.md` (packages, commands, how to add a business package: model, repository, store_json, service, handler, kind, routes, contract), `frontend/README.md` (structure, commands, how to add a page and use shared components, regenerating API types)
- [ ] T082 Run the backend checks from `backend/`: `gofmt -l .` prints nothing, `go vet ./...` and `go test ./...` pass
- [ ] T083 Run the web-app checks from `frontend/`: `npm run check`, `npm run build` and `npm run test:e2e` pass (twice in a row for the end-to-end tests)
- [ ] T084 Walk through `specs/001-app-shell-and-sign-in/quickstart.md` §4 steps 1–23 with the seed data; compare the screens with `docs/product/design-system.md` and the ZenReach screenshots (`docs/reference/zenreach-screenshots/`); record anything that does not match as a new task
- [ ] T085 Run the SC-002 moderated session (at least 5 people who have not used NorthStar) with the product owner and record the results in `specs/001-app-shell-and-sign-in/checklists/usability.md`
- [ ] T086 Set feature 001 to Done in `docs/sdd/feature-map.md` once the product owner accepts the demo

---

## Requirement coverage

| FR / SC | Tasks |
|---|---|
| FR-001, FR-002 | T039, T041, T044, T045, T048, T050, T055, T080 |
| FR-003, FR-004 | T040, T043, T044, T045, T048, T050, T055 |
| FR-005 | T012, T022, T048, T053, T055, T071 |
| FR-006 | T040, T043, T044, T055, T058, T076, T077, T078 |
| FR-007 | T040, T043, T045, T066, T075 |
| FR-008 | T075 (Daniel renames a folder Priya created; no permission message) |
| FR-009 | T016, T043, T046, T048, T050 |
| FR-010, FR-011, FR-013, FR-014 | T037, T039, T041, T044, T052, T056, T057, T058, T060 |
| FR-012 | T056, T058, T076, T077, T078 |
| FR-015, FR-016, FR-017 | T022, T034, T038, T051, T052, T057, T059, T060 |
| FR-018 | T052, T061, T062, T063 |
| FR-019 | T024 – T035, T079, T080 |
| FR-020 | T010, T012, T023, T032, T033, T048, T049, T066, T078 |
| FR-021 | T022 |
| FR-022 | T064, T065, T066, T067, T075 |
| FR-023 | T065 (`TestLog_EntriesNeverChangeOrDisappear_FR023`), T067 |
| FR-024 – FR-026 | T065 – T070, T072 – T075 |
| FR-027 | T014, T065 |
| FR-028 | T042, T045, T047, T048, T050, T084 |
| FR-029 | T015, T023, T066, T069, T072, T074, T075 |
| SC-001, SC-003, SC-007 | T055, T060, T075 |
| SC-002 | T085 |
| SC-004, SC-005 | T065, T066, T075 |
| SC-006 | T080 |

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (Phase 1)**: done.
- **Foundational (Phase 2)**: blocks all stories. Backend tasks in the listed order. Web app: T020 first;
  T024 before T031 and T032; T025 before T027; T030 before T036; T036 before any component test that
  needs providers (component tests of T024–T034 render without the router where possible).
- **US1 (Phase 3)**: backend T039/T040 → T041 → T042 → T043 → T044 → T045 → T046 → T047; web T048 →
  T049 → T050 → T051 → T052 → T053; end-to-end T054 → T055 after both (the fixture needs T044).
- **US2 (Phase 4)**: web only; needs US1's web tasks. T060 after T056–T059.
- **US3 (Phase 5)**: needs T052. T063 after T062.
- **US5 (Phase 6)**: backend T064–T066 → T067 → T068 → T069 → T070 → T071 (needs every package wired);
  web T072 → T073 → T074; T075 after T070 and T074.
- **US4 (Phase 7)**: needs US1 and US2. T078 after T077.
- **Polish (Phase 8)**: T079 after T074; T080 after T079 and every story; T082–T086 last.

### Parallel opportunities

- The backend tasks and the web-app tasks of every phase are independent: the web app's unit tests use
  `mockApi()`, and both sides follow `backend/api/openapi.yaml`. Only the end-to-end tasks (T054, T055,
  T060, T063, T075, T078, T080) need both sides.
- `npm run build` type-checks only the app code, so an end-to-end run is never blocked by a test file
  written before its code.

### Parallel example: Phase 6 (US5)

```text
Backend:  T064 + T065 + T066 (tests) → T067 → T068 → T069 → T070 → T071
Web app:  T072 (tests) → T073 → T074
Both:     T075 (end-to-end)
```

---

## Implementation Strategy

### MVP first (User Story 1)

1. Phase 2 Foundational.
2. Phase 3 (US1) → stop and validate: every demo person can sign in and is greeted.

### Incremental delivery

1. + US2 → the full frame; + US3 → Getting started; + US5 → the history core; + US4 → session care.
2. After each story: run its tests and its end-to-end file; commit with the feature number, e.g.
   `feat(001): demo sign-in page (FR-001–FR-004)`.

### Notes

- Never edit `frontend/src/api/schema.d.ts` by hand: change `backend/api/openapi.yaml`, then
  `npm run api:types`.
- Keep the user-visible messages exactly as in the plan's table; tests quote them.
- Mark each task `[X]` here when it is done.
