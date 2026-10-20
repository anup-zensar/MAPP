# Implementation Plan: Application Shell and Demo Sign-in

**Branch**: `001-app-shell-and-sign-in` | **Date**: 2026-10-18 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-app-shell-and-sign-in/spec.md`

**Design artifacts**: [research.md](research.md) (decisions R1–R24) · [data-model.md](data-model.md) ·
[contracts/openapi.yaml](contracts/openapi.yaml) · [quickstart.md](quickstart.md) ·
[ADR-0007](../../docs/architecture/adr/0007-activity-log-and-recycle-bin-core.md)

## Summary

Feature 001 creates NorthStar's two runtime parts and everything later features reuse:

- **Go API** (`backend/`): standard-library HTTP server; shared platform packages (errors, JSON helpers
  and the route recorder, actor, clock, JSON file store with transactions, readable IDs behind a
  repository interface, record bookkeeping, contract-test helper); demo sign-in with HttpOnly-cookie
  sessions and 12-hour inactivity expiry; the current user and per-person preferences; the
  **activity-log and recycle-bin core** (one log entry per change with its path and readable values,
  delete to the bin with everything underneath, Undo, restore with the parent check); a developer-only
  **sample records** type that exercises that core; a demo-data loader.
- **React web app** (`frontend/`): sign-in page in Shell's design language (design system v2), application layout (sidebar with collapse
  and mobile drawer, account menu, notification bell, light/dark switch, demo banner, footer),
  getting-started Home, "Coming soon" and "Page not found" pages, the shared building blocks of design
  system §5, a typed API client generated from the OpenAPI contract, date and money formatting, shared
  delete-with-Undo and unsaved-changes helpers, and two developer pages (sample records, shared building
  blocks).

The approach favours the standard library, a handful of well-known libraries and one repeated pattern
(handler → service → repository; page → shared components → API client) so that a beginner can follow it
(constitution VII). The plan passed a four-reviewer review and the Spec Kit analyze step before tasks
were executed.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app); Node.js 24 LTS for tooling

**Primary Dependencies**: API — standard library only at run time; tests add
`github.com/pb33f/libopenapi-validator`. Web app — `react-router` 8 (data router), `@tanstack/react-query`
5, `radix-ui` 1.8, `lucide-react`, `openapi-fetch` (+ `openapi-typescript` to generate types); Vite 8;
ESLint 10. Full list and reasons: research R2, R12.

**Storage**: local JSON file store (`backend/data/`, git-ignored) behind repository interfaces and a
`store.Transactor` interface; the activity log is append-only JSON Lines; PostgreSQL later (ADR-0004,
ADR-0007, research R3)

**Testing**: Go `testing` + `httptest` + OpenAPI contract validation; Vitest 5 + Testing Library + jsdom;
Playwright 1.64 + axe for end-to-end and WCAG 2.2 AA checks (research R19)

**Target Platform**: developer laptops (Windows 11 first; macOS/Linux work the same); browsers: current
Chrome and Edge (Playwright Chromium in tests)

**Project Type**: web application — React single-page app + Go REST API

**Performance Goals**: sign in and reach Home in under 10 seconds (SC-001) — locally every API call
answers in well under 100 ms with demo-sized data

**Constraints**: servers listen on `127.0.0.1` only; no cloud services; no secrets in the repository;
demo sign-in only in `local`/`development`; WCAG 2.2 AA (SC-006); usable at 1280 × 720 and 390 px wide
(SC-007); deterministic tests

**Scale/Scope**: 6 demo people; demo-sized data (thousands of records, tens of thousands of log
entries). 8 signed-in routes + sign-in + not found + 2 developer pages; 16 API operations.

## Constitution Check

*GATE: checked before Phase 0 research and again after Phase 1 design.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to an FR, acceptance scenario, edge case or SC (tables below). The developer pages are in the spec's Assumptions (D-036); the other-tab sign-in, the refused Undo and the one-Undo-at-a-time rule are in the spec's edge cases and FR-026. No `[NEEDS CLARIFICATION]` in the spec, so clarify was skipped as allowed | Pass |
| II. Shell business, plain language | Navigation uses Plans, Approvals, All work, Settings, Recycle bin, Activity log; no Hive9 terms; every user-visible message is listed once in *User-visible messages* (spec wording kept verbatim) | Pass |
| III. Simple, friendly experience | ZenReach layout with the tokens of design system v2 (Shell's design language); delete with confirmation and Undo; unsaved changes protected; all errors at once; nothing disabled without a reason; WCAG 2.2 AA with automated checks of every screen and overlay and AA-safe colours (research R17). One recorded exception: the developer sample page is not a full CRUD page (Complexity Tracking) | Pass (1 exception) |
| IV. Simple, layered architecture | Two runtime parts. Go packages per area (`auth`, `people`, `history`, `samples`), each handler → service → repository; services own every rule; all storage — readable-ID counters included (`ids.Counters`) — goes through repository interfaces plus `store.Transactor`, so PostgreSQL needs no service change; REST + JSON with an OpenAPI 3.1 file; generated web types | Pass |
| V. Tested business rules | Unit tests for every rule (table *Business rules and the tests that prove them*); a test for every acceptance scenario; end-to-end browser tests for US1–US5; a contract test for every endpoint plus a route-coverage test; Go tests use a fixed clock and temporary data; Vitest runs in UTC with fixed time; end-to-end tests run on their own seeded data (the real-clock exception is in *Complexity Tracking*) | Pass |
| VI. Safe by default | Demo sign-in only in local/development (and no session works outside them); the demo password is public on purpose (D-004, ADR-0005: fictitious data, local only) and is the only credential in the repository; the actor is on every request and passed to every service call, reads included; the activity log is written in the same transaction as the change (display preferences are not business records and are not logged, D-038); the HttpOnly `SameSite=Strict` cookie carries a random token and the server stores only its SHA-256 hash; cross-origin, wrong-host and non-JSON requests are refused; input validated by the API; personal data never in server logs; timestamps UTC; deletes go to the recycle bin and restore parents first | Pass |
| VII. Keep it small and readable | Standard library first; one test-only dependency in Go; no CSS framework, component library, form library or global store in the web app; every dependency justified in research R2/R12; the history core is recorded in ADR-0007 | Pass |

**Post-design re-check (after Phase 1)**: Pass — no third runtime part, no extra layers, no unjustified
dependency. The choices that add code beyond the visible screens, and the one exception, are in
*Complexity Tracking*.

## Project Structure

### Documentation (this feature)

```text
specs/001-app-shell-and-sign-in/
├── spec.md              # what and why
├── checklists/requirements.md   # spec quality (and usability.md after the SC-002 session)
├── plan.md              # this file
├── research.md          # decisions R1–R24
├── data-model.md        # records, rules, life cycles
├── quickstart.md        # run and validate
├── contracts/openapi.yaml   # the API of this feature (copied to backend/api/openapi.yaml)
└── tasks.md             # created by /speckit-tasks
```

### Source code (repository root)

```text
backend/
├── go.mod                       # module "northstar", go 1.27
├── README.md                    # how to run, test and add a package
├── api/
│   ├── openapi.yaml             # THE contract (source of truth; every feature adds to it)
│   └── embed.go                 # package api: //go:embed openapi.yaml → var OpenAPI []byte
├── cmd/
│   ├── api/main.go              # reads config, opens the store, serves on 127.0.0.1:APP_PORT
│   └── seed/main.go · main_test.go   # loads the demo data (-data, -reset); logic in run(...)
├── seed/
│   ├── seed.go                  # the demo dataset (001: six demo people); later features add theirs
│   └── seed_test.go
└── internal/
    ├── app/
    │   ├── config.go · config_test.go   # APP_ENV, APP_PORT, DATA_DIR, DEMO_PASSWORD
    │   ├── app.go               # New(): wires repositories, services, kinds and routes
    │   └── app_test.go          # package app_test: health, unknown paths, headers, demo mode, coverage
    ├── platform/                # small, boring shared code
    │   ├── actor/actor.go       # Actor{ID, Name}, actor.With / actor.From, actor.System
    │   ├── clock/clock.go       # type Clock func() time.Time; clock.System
    │   ├── errs/errs.go         # Error and its parts, codes, constructors, Problems (+ errs_test.go)
    │   ├── web/web.go · router.go   # JSON in/out, errors, middleware, route recorder (+ web_test.go)
    │   ├── store/store.go       # JSON file store: DB, Transactor, tables, append tables (+ store_test.go)
    │   ├── ids/ids.go · store_json.go   # Generator + Counters interface + JSON counters (+ ids_test.go)
    │   ├── record/record.go     # Meta: version + created/updated; CheckVersion (+ record_test.go)
    │   └── apitest/apitest.go   # test helper: real app on a temp data folder, validates every call
    ├── auth/                    # demo sign-in, sessions, the sign-in middleware
    │   ├── model.go · repository.go · store_json.go · service.go · service_test.go
    │   └── middleware.go · handler.go · handler_test.go
    ├── people/                  # demo people, "me", preferences (feature 003 extends)
    │   ├── model.go · model_test.go · repository.go · store_json.go · service.go · service_test.go
    │   └── handler.go · handler_test.go
    ├── history/                 # activity log + recycle bin core (feature 011 adds the screens)
    │   ├── model.go · kinds.go · diff.go · repository.go · store_json.go · service.go · service_test.go
    │   └── diff_test.go · handler.go · handler_test.go
    └── samples/                 # developer sample folders and notes (demo mode only)
        ├── model.go · repository.go · store_json.go · service.go · service_test.go
        └── kinds.go · handler.go · handler_test.go

frontend/
├── package.json · package-lock.json · vite.config.ts · eslint.config.js · playwright.config.ts
├── tsconfig.json (references) · tsconfig.app.json (src without tests) · tsconfig.test.json (src with
│   tests) · tsconfig.node.json (configs, e2e/)
├── .prettierrc.json · .prettierignore · index.html (with the theme script) · README.md
├── public/favicon.svg
├── e2e/                         # Playwright
│   ├── fixtures.ts              # signInAs(page, email) via the API + preference reset; axe helper
│   └── sign-in · navigation · home · session · history · accessibility (.spec.ts)
└── src/
    ├── main.tsx                 # imports tokens.css + global.css, mounts <App/>
    ├── vite-env.d.ts
    ├── app/                     # router, layout, session, theme
    │   ├── App.tsx              # QueryClientProvider, Tooltip.Provider, ToastProvider, RouterProvider
    │   ├── router.tsx           # exports routes and router (createBrowserRouter)
    │   ├── RootLayout.tsx · RouteErrorPage.tsx
    │   ├── navigation.ts        # sidebar items, My profile, and the "Coming soon" sentences
    │   ├── SessionProvider.tsx · useSession.ts      # session state and endSession
    │   ├── RequireSignIn.tsx · PublicLayout.tsx · DevOnly.tsx
    │   ├── SessionCheckPage.tsx (+ .module.css)   # full page while the session is checked or unreachable
    │   ├── NotFoundRoute.tsx    # "Page not found" in the app layout or the public layout
    │   ├── theme.ts · useTheme.ts
    │   ├── AppLayout.tsx · Sidebar.tsx · MobileTopBar.tsx · NavDrawer.tsx (+ .module.css)
    │   ├── AccountMenu.tsx · NotificationBell.tsx · ThemeToggle.tsx (+ .module.css)
    │   └── *.test.tsx           # also App, router, RouteErrorPage and PublicLayout
    ├── components/              # FR-019 shared building blocks — one folder per component
    │   ├── index.ts             # features import only from '@/components' (+ index.test.ts)
    │   └── Button/ PageHeader/ SummaryCard/ Card/ StatusChip/ CompletenessChip/ InfoTip/ Tooltip/
    │       Avatar/ Logo/ SidePanel/ ConfirmDialog/ Toast/ EmptyState/ Skeleton/ FormErrorSummary/
    │       TextField/ PasswordField/ ErrorState/ DemoBanner/ ActionsMenu/
    │       (Name.tsx, Name.module.css, Name.test.tsx; Toast also useToast.ts)
    ├── features/
    │   ├── sign-in/SignInPage.tsx (+ css, test)
    │   ├── home/HomePage.tsx (+ css, test)
    │   ├── placeholders/ComingSoonPage.tsx · NotFoundPage.tsx (+ tests)
    │   └── dev/SampleRecordsPage.tsx · SamplePanels.tsx · ComponentGalleryPage.tsx · sampleApi.ts
    │       (+ css, tests)
    ├── api/
    │   ├── schema.d.ts          # GENERATED from backend/api/openapi.yaml — never edit by hand
    │   ├── client.ts            # openapi-fetch client, call(), session-lost hook
    │   └── errors.ts            # ApiError and its kinds; conflictMessage(); resolvedMessage()
    ├── lib/                     # every file with its test (cx.test.ts, format.test.ts, …)
    │   ├── format.ts            # formatDate, formatDateTime, formatTime, formatMoney, formatMoneyCompact
    │   ├── safeNext.ts          # the only place that decides where to go after sign-in
    │   ├── useUndoableDelete.ts # shared delete message with Undo (US5)
    │   ├── useUnsavedChanges.tsx # Keep editing / Discard changes guard (useBlocker + beforeunload)
    │   ├── usePageTitle.ts      # "<Page> · NorthStar"
    │   └── cx.ts                # join class names
    ├── styles/
    │   ├── tokens.css           # design tokens, light + dark (values below)
    │   └── global.css           # base styles, focus ring, reduced motion, .visually-hidden
    └── test/                    # the helpers have tests too (mockApi.test.ts, render.test.tsx)
        ├── setup.ts             # jest-dom; stubs; fails a test that made an unmocked request
        ├── mockApi.ts           # replaces fetch with canned answers per "METHOD /path"
        └── render.tsx           # renderWithProviders(ui, { route, user, path }) and
                                 # renderRoutes(routes, { route, user }) → { router, queryClient, user };
                                 # findMessages(): the toast area, once it takes clicks
```

**Structure decision**: the web-application layout of `docs/architecture/repository-layout.md` (updated in
this feature). The `@/` import alias points to `frontend/src` (Vite, TypeScript and Vitest).

## Backend design

### Request flow

```
HTTP request
  → web.Middleware: recover panics → log the request (no personal data) → API headers (no-store,
    nosniff) → refuse a Host that is not 127.0.0.1 / localhost / [::1] → http.CrossOriginProtection
  → ServeMux pattern ("PATCH /api/sample-folders/{folderId}")
  → auth.RequireSignIn (signed-in routes): Authenticate in one Write → actor in a fresh context (or 401)
  → handler: web.ReadJSON, actor.From(ctx), call the service, web.WriteJSON / web.WriteError
  → service: validate (errs.Problems), run the rule inside tx.Write(ctx, fn), history.Log in the same fn
  → repository (interface) → JSON store implementation (store.Get / store.Put …)
```

### Shared platform packages (contracts other features rely on)

| Package | Public API (names fixed; bodies up to the implementer) |
|---|---|
| `actor` | `type Actor struct{ ID string \`json:"id"\`; Name string \`json:"name"\` }`; `var System = Actor{ID: "system", Name: "NorthStar"}`; `With(ctx, Actor) context.Context`; `From(ctx) (Actor, bool)` |
| `clock` | `type Clock func() time.Time`; `func System() time.Time` (UTC, truncated to milliseconds) |
| `errs` | Owns the error parts every package shares: `Error{Status int \`json:"-"\`; Code string \`json:"code"\`; Message string \`json:"message"\`; Fields []FieldError \`json:"fields"\`; Conflict *Conflict \`json:"conflict,omitempty"\`; Parent *RecordRef \`json:"parent,omitempty"\`; Resolution *Resolution \`json:"resolution,omitempty"\`}` implementing `error`; `FieldError{Field \`json:"field"\`; Message \`json:"message"\`}`; `Conflict{ChangedBy string \`json:"changedBy"\`; ChangedAt time.Time \`json:"changedAt"\`}`; `RecordRef{Type \`json:"type"\`; ID \`json:"id"\`; Name \`json:"name"\`}`; `Resolution{How string \`json:"how"\`; By actor.Actor \`json:"by"\`; At time.Time \`json:"at"\`}`. Code constants of research R9 (incl. `already_resolved`, `name_taken`). Constructors (each sets `Fields` to an empty list and uses the messages table): `Validation(...FieldError)`, `BadRequest(msg)`, `SignInRequired()`, `SessionExpired()`, `IncorrectCredentials()`, `NotFound(msg)`, `VersionConflict(noun, changedBy, changedAt)`, `RestoreParentFirst(parent RecordRef)`, `UndoExpired()`, `AlreadyResolved(name string, r Resolution)`, `NameTaken(msg)`, `Internal()`. `Problems.Add(field, message)` / `Problems.Err()` (nil when empty) |
| `web` | `ReadJSON(w, r, dst) error` — only `Content-Type: application/json`, 1 MB limit, unknown fields rejected, nothing after the JSON value; failure → `bad_request` with the messages-table text (detail only in the log); `WriteJSON(w, status, v)`; `WriteError(w, r, err)` — `errors.As` finds an `*errs.Error`, anything else is logged and becomes `500 internal_error`; `NotFound(w, r)`; `Middleware(next) http.Handler` (as in the request flow; refused cross-origin or wrong-host requests answer `400 bad_request` "This request came from another site and was refused."); `type Wrap func(http.HandlerFunc) http.HandlerFunc`; `type Router` (`NewRouter()`, `Handle(pattern string, h http.HandlerFunc)` registering on its `ServeMux` and recording the pattern, `Patterns() []string`, `ServeHTTP`) |
| `store` | `type Transactor interface{ Read(ctx, fn func(ctx) error) error; Write(ctx, fn func(ctx) error) error }`; `type DB` implementing it; `Open(dir) (*DB, error)`; `(*DB).IsEmpty() bool` (no `*.json` or `*.jsonl` table yet); helpers for JSON repositories `Get[T](ctx, table, id) (T, bool, error)`, `List[T](ctx, table) ([]T, error)`, `Put(ctx, table, id, v) error`, `Delete(ctx, table, id) error`, `Append(ctx, table, v) error`, `Scan[T](ctx, table) ([]T, error)`; errors `ErrNoTransaction`, `ErrReadOnly`, `ErrFailed`. Behaviour exactly as research R3: encoded tables; an overlay the transaction reads (own writes visible); joining an existing transaction; `fn`'s error returned unchanged; commit order (temp files + Sync → `counters` first → other tables by name with a 2 s rename retry → log lines appended + Sync → memory); **no file handle stays open between commits**; failed state from the first rename on; `Open` creates the folder, missing table = empty, removes leftover `*.tmp`, stops on invalid JSON, cuts a partial last log line off the file; `List` sorted (shorter ids first, then alphabetical) and never nil |
| `ids` | `type Counters interface{ Increment(ctx context.Context, prefix string) (int, error) }` (the repository of the last number per prefix); `NewJSONCounters() Counters` (table `counters`); `type Generator`; `NewGenerator(c Counters) *Generator`; `(*Generator).Next(ctx, prefix) (string, error)` → `"P-1001"`; `(*Generator).NextNumber(ctx, prefix) (int, error)` (research R4) |
| `record` | `type Meta struct{ Version int \`json:"version"\`; CreatedAt time.Time \`json:"createdAt"\`; CreatedBy actor.Actor \`json:"createdBy"\`; UpdatedAt time.Time \`json:"updatedAt"\`; UpdatedBy actor.Actor \`json:"updatedBy"\` }` (embedded, JSON flattened); `NewMeta(by, now) Meta`; `(*Meta).Touch(by, now)`; `(Meta).CheckVersion(expected int, noun string) error` |
| `apitest` | `New(t, opts...) *Server`: temporary data folder (or `WithDataDir(dir)`), demo-people seed unless `WithoutSeed()` or the folder already has data (seeding arrives with the seed package in US1), fixed clock 2026-03-12 09:00 UTC with `(*Server).Advance(d)`, demo mode unless `WithEnv("production")`; `(*Server).Do(t, method, path, body any, opts...) *Response` validates request and response with libopenapi-validator (`config.WithFormatAssertions()`, `config.WithoutSecurityValidation()`) and sends `Host: 127.0.0.1` unless `WithHost(h)`; request options `SkipRequestValidation()`, `SkipValidation()`, `RawBody([]byte)` (sent as `application/json`), `WithCookie(*http.Cookie)`; `Response{Status int; Header http.Header; Body []byte}` with `Cookie(name) *http.Cookie` and `Decode(t, &v)`; `(*Server).SignIn(t, email) *Client` (same `Do`, sending that person's cookie). Only external test packages import it |

### Constructors and wiring

Repositories find the transaction in the context, so their constructors take nothing. Handlers register
their routes on the shared `web.Router`; `signedIn` is `auth.RequireSignIn` bound to the auth service.

| Package | Constructors | Repository interface (JSON implementation in `store_json.go`) |
|---|---|---|
| `people` | `NewJSONRepository() Repository`; `NewService(tx store.Transactor, repo Repository) *Service`; `NewHandler(s *Service) *Handler`; `(*Handler).Routes(r *web.Router, signedIn web.Wrap)` | `Get(ctx, id)`, `FindByEmail(ctx, email)`, `List(ctx)`, `Put(ctx, Person)`, `GetPreferences(ctx, personID)`, `PutPreferences(ctx, Preferences)` |
| `auth` | `NewJSONRepository() Repository`; `NewService(tx store.Transactor, repo Repository, people people.Repository, demoMode bool, demoPassword string, now clock.Clock) *Service`; `(*Service).RequireSignIn(next http.HandlerFunc) http.HandlerFunc`; `NewHandler(s *Service, people *people.Service) *Handler`; `Routes(r, signedIn)` | `Get(ctx, id)`, `Put(ctx, Session)`, `Delete(ctx, id)`, `List(ctx)` |
| `history` | `NewJSONRepository() Repository`; `NewService(tx store.Transactor, repo Repository, ids *ids.Generator, now clock.Clock) *Service`; `NewHandler(s *Service) *Handler`; `Routes(r, signedIn)` | `AppendEntry(ctx, Entry)`, `Entries(ctx)`, `PutBinEntry(ctx, BinEntry)`, `GetBinEntry(ctx, id)`, `BinEntries(ctx)` |
| `samples` | `NewJSONRepository() Repository`; `NewService(tx store.Transactor, repo Repository, ids *ids.Generator, hist *history.Service, now clock.Clock) *Service`; `(*Service).Kinds() []history.Kind`; `NewHandler(s *Service) *Handler`; `Routes(r, signedIn)` | `GetFolder`, `ListFolders`, `PutFolder`, `DeleteFolder`, `GetNote`, `ListNotes`, `NotesInFolder`, `PutNote`, `DeleteNote` |
| `seed` | `Load(ctx context.Context, db *store.DB, now clock.Clock) error` (returns `ErrNotEmpty` when `db` already has data) | uses `people.NewJSONRepository()` and an `ids.Generator` |
| `app` | `New(cfg Config, db *store.DB, now clock.Clock) (*App, error)` with `App{Handler http.Handler; Routes []string}` | — |

`app.New`: one `ids.NewGenerator(ids.NewJSONCounters())`; people, auth (demo mode and password from
`cfg`), history; in demo mode also samples, whose kinds are registered with history (an error stops
start-up); then a `web.NewRouter()`, `GET /api/health`, each handler's `Routes`, the `/api/` catch-all,
and `web.Middleware` around the router. No package-level state: everything is built on each call.

### Feature packages

Every package keeps **stored records** and **API types** separate in `model.go` (`samples.Folder` vs
`samples.FolderView`; `people.Person` vs `people.CurrentUser`, `people.DemoPerson` and
`people.PreferencesView`; `history.BinEntry` vs `history.BinEntryView`); handlers return only API
types; stored records never contain lists of their children. `history.Entry` and `samples.Note` serve
as both: the API returns them exactly as stored. Every service method takes the actor
(`ctx, a actor.Actor, …`), reads included. The exceptions work before an actor exists or produce it:
`auth.SignIn`, `auth.SignOut`, `auth.Authenticate` and `people.ActivePeople` (`auth.DemoMode()` and
`auth.DemoPassword()` only return settings).

| Package | Responsibility | Key operations |
|---|---|---|
| `auth` | Demo sign-in and sessions (research R6) | `DemoMode()`, `DemoPassword()`, `SignIn(ctx, email, password, oldToken) (token, people.Person, error)`, `SignOut(ctx, token)`, `Authenticate(ctx, token) (actor.Actor, error)` — one `Write`: missing → `sign_in_required`; expired (`now − lastUsedAt > 12 h 1 min`) → delete + `session_expired`; person missing or inactive → delete + `sign_in_required`; otherwise update `lastUsedAt` when it is at least 1 minute old. `RequireSignIn(next)` lets that transaction finish and calls the handler with `r.WithContext(actor.With(r.Context(), a))`; with demo mode off it answers `sign_in_required` to everything. Handler: `GET /api/auth/demo-people` (answers `people.ActivePeople` — a `[]DemoPerson` — with `DemoPassword()`) and `POST /api/auth/sign-in` **only in demo mode**; `POST /api/auth/sign-out` always: it clears the cookie first (`MaxAge: -1`), then deletes the session, and answers `204` (`500` only when the store fails) |
| `people` | People, "me", preferences | `ActivePeople(ctx)` (sorted by name, as `[]DemoPerson`), `CurrentUser(ctx, a)` (`sign_in_required` when the person no longer exists), `UpdatePreferences(ctx, a, PreferencesUpdate)` with `PreferencesUpdate{ Theme *string; SidebarCollapsed *bool }` — nil means unchanged; one `Write`: read the stored record or the defaults, apply the sent fields, validate, save only when a value changed. Handler: `GET /api/me`, `PATCH /api/me/preferences` |
| `history` | Activity log and recycle bin (research R8, data-model §5–§7) | `Register(Kind) error`; `Log(ctx, a, LogInput) (Entry, error)`; `Diff(before, after any) ([]Change, error)`; `MoveToBin(ctx, a, kind, id) (BinEntry, error)`; `Undo(ctx, a, entryID) (RestoreResult, error)`; `Restore(ctx, a, entryID) (RestoreResult, error)`; `ListLog(ctx, a, LogFilter) ([]Entry, int, error)`; `ListBin(ctx, a) ([]BinEntry, error)`. API types: `BinEntryView` (from `BinEntry.View()`), `DeleteResult{RecycleBinEntry}` (the answer of every delete), `RestoreResult`, and `LogPage` / `BinPage` (`{items, total}`). Handler: `GET /api/activity-log` (`limit` 1–200, 50 when not sent; any other value, an empty `limit=` included, → `validation_failed` on `limit`), `GET /api/recycle-bin`, `POST /api/recycle-bin/{entryId}/undo`, `POST /api/recycle-bin/{entryId}/restore` |
| `samples` | Developer sample folders and notes | `ListFolders(ctx, a)` (sorted by name ignoring letter case; each folder's notes oldest first; the handler answers `FolderPage`, `{items, total}`), `CreateFolder(ctx, a, FolderInput)`, `UpdateFolder(ctx, a, id, FolderUpdate)` (`Version *int`: missing or < 1 → `validation_failed` on `version`; then the version check, unique name, `history.Diff` → `changed` entry; no change → nothing), `DeleteFolder(ctx, a, id)` (→ `MoveToBin`), `CreateNote(ctx, a, folderID, NoteInput)`, `DeleteNote(ctx, a, id)`, `Kinds()` — the folder kind's `PrepareRestore`: when another folder has the name, an **Undo** is refused with `NameTaken` and a **Restore** renames to " (restored)", " (restored 2)", …, calls `Meta.Touch` and returns the note "Renamed to '<new name>' because the name was taken.". Handler: the `/api/sample-*` operations; missing records answer `404 not_found` with the messages-table texts |

`history.Kind`, `LogInput` and the types `history` shares with `errs`:

```go
type RecordRef = errs.RecordRef   // {type, id, name}
type Resolution = errs.Resolution // {how, by, at}

type Kind struct {
    Name     string // "sampleNote" — stored in log and bin entries
    Singular string // "sample note" — mid-sentence; capitalise the first letter at a sentence start
    Plural   string // "sample notes"
    Parent   string // parent kind name, "" for top-level records (parents register first)

    Snapshot func(ctx context.Context, id string) (Snapshot, bool, error) // record + name + parent id
    ChildIDs func(ctx context.Context, parentID string) ([]string, error) // required when Parent != ""
    Remove   func(ctx context.Context, id string) error
    Put      func(ctx context.Context, s Snapshot) error

    // Optional. Describe gets the stored value decoded (nil, bool, json.Number, string, []any or
    // map[string]any); an empty label or text means "use the default".
    Describe       func(ctx context.Context, field string, value any) (label, text string, err error)
    PrepareRestore func(ctx context.Context, rc RestoreContext, s *Snapshot) (notes []string, err error)
}

type RestoreContext struct { Actor actor.Actor; Now time.Time; Undo bool }

type LogInput struct {
    Kind, RecordID, RecordName string
    ParentID string   // where the path starts when the record itself is gone (then required for a child kind)
    Action   Action   // created, changed, statusChanged, moved, copied, deleted, restored
    Verb     string   // optional; required for statusChanged ("published")
    Changes  []Change // from Diff; history fills label, oldText, newText
    Comment, Details string
}
```

`Register` errors: empty or duplicate name, `Parent` equal to the name or not yet registered, missing
`Snapshot`/`Remove`/`Put`, missing `ChildIDs` when `Parent` is set. Children are visited in registration
order with sorted ids; `contents` lists only kinds with a count of at least 1, in that order; `Remove` runs
children first. `Log` fails, writing nothing, for an unknown kind or action, a status change without its
verb, or a gone child-kind record without `ParentID`. Undo and Restore check in this order: unknown entry
→ `404 not_found`; a kind not registered here → `404 not_found`; already undone or restored →
`already_resolved`; Undo more than 2 minutes after the delete → `undo_expired`; parent missing →
`restore_parent_first`; then each kind's `PrepareRestore` (which refuses an Undo of a taken name with
`name_taken`). `MoveToBin`, `Undo`, `Restore` and `Log` join the caller's transaction when there is one.
Details texts: deletes "with 2 sample notes"; restores "Undo" or "Restored from the recycle bin", then
", with 2 sample notes" when it had contents, then each note as its own sentence (". Renamed to …").

### Configuration and commands

- `Config` (environment): `APP_ENV` (`local` default | `development` | `production`), `APP_PORT` (1–65535,
  default 8080), `DATA_DIR` (default `data`), `DEMO_PASSWORD` (default `northstar`; blank refused);
  `DemoMode()` = `local` or `development`. No host setting: the API always listens on `127.0.0.1`.
  An empty variable counts as not set (its default applies). Invalid values stop the program with the
  messages-table texts — all of them together, one per line.
- `GET /api/health` → `{"status": "ok", "environment": "...", "demoSignIn": true|false}` (public).
  Everything else under `/api/` that is not registered answers JSON `404 not_found` (wrong methods too).
- `cmd/api`: reads the config, opens the store, listens on `127.0.0.1:APP_PORT`; a busy port (Windows
  error 10048 or 10013, or `syscall.EADDRINUSE`) prints the messages-table text and exits 1; Ctrl+C shuts down
  gracefully; prints the start-up line.
- `cmd/seed`: `run(args []string, getenv func(string) string, out io.Writer) error` holds the logic (tested
  in `main_test.go`) and reads the settings through `app.LoadConfig`; `-data` (default `DATA_DIR` or
  `data`, created when missing), `-reset` (deletes only `*.json`, `*.jsonl` and `*.tmp`); refuses an
  extra argument, a folder with data (`IsEmpty`) without `-reset`, `production`, and a running API —
  `GET http://127.0.0.1:<APP_PORT>/api/health` answering `200 OK` within 1 second (another program on the
  port does not count); prints "Loaded the demo data into <folder>: 6 people."

## Frontend design

### Router (research R15)

A **data router** without loaders or actions: `router.tsx` exports `routes` (for tests) and `router`
(`createBrowserRouter(routes)`); `App.tsx` renders `QueryClientProvider` → `Tooltip.Provider` →
`ToastProvider` → `RouterProvider` (from `react-router/dom`). The root route renders `RootLayout`
(`SessionProvider`, `ScrollRestoration`, `Outlet`); its `errorElement` is `RouteErrorPage`, which shows
`ErrorState` (kind `other`; **Try again** reloads the page), so an unexpected render error never leaves a
blank page. The data router is what makes `useBlocker` (unsaved changes) work. Tests use
`createMemoryRouter`.

| Path | Page | Signed in? | Tab title |
|---|---|---|---|
| `/sign-in` | SignInPage (public layout) | public; a signed-in visitor goes on to `safeNext(next)` | Sign in · NorthStar |
| `/` | HomePage | yes | Home · NorthStar |
| `/plans` | ComingSoonPage (Plans) | yes | Plans · NorthStar |
| `/approvals` | ComingSoonPage (Approvals) | yes | Approvals · NorthStar |
| `/all-work` | ComingSoonPage (All work) | yes | All work · NorthStar |
| `/settings` | ComingSoonPage (Settings) | yes | Settings · NorthStar |
| `/recycle-bin` | ComingSoonPage (Recycle bin) | yes | Recycle bin · NorthStar |
| `/activity-log` | ComingSoonPage (Activity log) | yes | Activity log · NorthStar |
| `/profile` | ComingSoonPage (My profile; not in the sidebar, reached from the account menu) | yes | My profile · NorthStar |
| `/dev/sample-records` | SampleRecordsPage inside `DevOnly` | yes | Sample records · NorthStar |
| `/dev/components` | ComponentGalleryPage inside `DevOnly` | yes | Shared building blocks · NorthStar |
| anything else | NotFoundPage — inside the app layout when signed in, the public layout otherwise | public | Page not found · NorthStar |

- `DevOnly` asks `GET /api/health` (query `['health']`): `demoSignIn: true` shows the page, otherwise
  NotFoundPage.
- US3 scenario 2: the route `/` renders only `features/home/HomePage.tsx`; the sign-in flow navigates to
  `safeNext(next)` (default `/`) and never imports `HomePage`, so feature 009 replaces one component.
- "Coming soon" sentences (FR-015), kept in `navigation.ts`:

| Section | Sentence |
|---|---|
| Plans | Create yearly marketing plans, give them budgets and publish them when they're ready. |
| Approvals | Approve or decline the activities waiting for you, and follow the ones you sent. |
| All work | See every campaign, initiative, activity and line item across plans as a grid, a calendar or a budget view. |
| Settings | Set up people and teams, models, activity types, Shell attributes, approval workflows, currencies and finance budget items. |
| Recycle bin | Bring back anything that was deleted, together with everything that was under it. |
| Activity log | See who changed what, and when, in every plan and setting. |
| My profile | See your details and teams, and choose how dates and money are shown to you. |

  The page: `PageHeader` (eyebrow and title = the section name, description = the sentence) and an
  `EmptyState` card with the Coming-soon texts and **Go to Home**. NotFoundPage: `<h1>` and text from the
  messages table, **Go to Home** (to `/`).

### Session (research R15)

- `useSession()` → `{ status: 'loading' | 'signedIn' | 'signedOut' | 'unreachable', user, error, theme,
  refetch, signIn(email, password), signOut(), updatePreferences(patch) }` (`theme`: the colours shown,
  `light` or `dark`, for the theme switch; `useCurrentUser()` returns the person inside `RequireSignIn`,
  where someone is always signed in), built on the `['session']` query
  (`GET /api/me`, `staleTime` 30 s, checked again when the window regains focus and by `RequireSignIn`
  on every navigation). Rules: a 401 means `signedOut`; any other failure while no person is cached
  means `unreachable`; a failed background check while a person is cached keeps `signedIn`. A 401
  `session_expired` from `GET /api/me` puts "Please sign in again" on the sign-in page even when no person
  was cached (a reload after expiry).
- While `loading`, `RequireSignIn`, the sign-in page and the not-found route show a full-page loading
  placeholder (`SessionCheckPage`: banner only, no sidebar). While `unreachable`, they show a full-page
  `ErrorState` (public layout) whose **Try again** calls `refetch()`; the tab title stays the requested
  page's title (each signed-in route names its page in `handle: { title }` in `router.tsx`); there is no
  redirect.
- **One way to end a session in the web app** — `endSession(reason: 'signedOut' | 'sessionLost')`,
  created inside `SessionProvider` from `useNavigate()` and `useQueryClient()` (it never imports the
  router, so there is no import cycle):
  (1) if the cached `['session']` is already `null`, do nothing (several 401s at once end it once);
  (2) `queryClient.setQueryData(['session'], null)`;
  (3) navigate to `/sign-in` with `replace`, `state: { notice: reason }` and `flushSync: true`, adding
  `'?' + new URLSearchParams({ next: pathname + search })` of the page on screen for `sessionLost` (a
  session found expired on the sign-in page itself keeps that page's address and its `next`). Steps 2
  and 3 show in the same moment, so neither the page on screen nor the sign-in page sees the old state
  in between and sends people on a second time, without the notice;
  (4) `queryClient.removeQueries({ predicate: q => q.queryKey[0] !== 'session' })` — never
  `queryClient.clear()`.
  Sign out calls `POST /api/auth/sign-out` and then `endSession('signedOut')`; when that call fails, the
  person stays signed in and a message says why, with **Try again**. The API client's
  session-lost hook (any call except `/api/auth/*` and `GET /api/me` answering `sign_in_required` or
  `session_expired`) calls `endSession('sessionLost')`; so does a `GET /api/me` 401 while a person is
  cached. The sign-in page reads the notice from `useLocation().state`.
- **Another person in another tab** (spec edge case): when `GET /api/me` returns a person with another id
  than the cached one, the app removes all other cached data, navigates to `/` and shows the toast from
  the messages table.
- **Where to go after sign-in**: `safeNext(raw)` parses `raw` with `new URL(raw, location.origin)` and
  returns `pathname + search + hash` only when the origin is NorthStar's and the path is not `/sign-in`;
  otherwise `/`. `RequireSignIn` redirects (with `replace`) to `/sign-in?` +
  `new URLSearchParams({ next: pathname + search })`. After a typed or demo-person sign-in, and for a
  signed-in visitor of `/sign-in`, the app navigates to `safeNext(next)` with `replace`.

### API client and errors (research R15)

- `client.ts`: `createClient<paths>({ baseUrl: window.location.origin, fetch: (request) =>
  globalThis.fetch(request) })`.
- `call(promise)`: network failure (anything thrown except an `AbortError`, which is re-thrown) →
  `ApiError('unreachable')`; `response.ok` → `data` (`undefined` for 204); a JSON body with `code` →
  `ApiError.fromBody(status, body)`; anything else → `unreachable` for status ≥ 500, otherwise `other`.
- `ApiError.kind`: `validation`, `badRequest`, `conflict`, `notFound`, `signInRequired`,
  `sessionExpired`, `incorrectCredentials`, `restoreParentFirst`, `undoExpired`, `alreadyResolved`,
  `nameTaken`, `unreachable` or `other`; plus `message`, `fields`, `conflict`, `parent`, `resolution`.
- Forms show `fields` next to inputs and in `FormErrorSummary`; `conflict` always through
  `conflictMessage(error, noun)`; `already_resolved` through `resolvedMessage(error, name)`; everything
  else with `ErrorState` or a toast showing the API's message; `unreachable` always with the spec
  sentence. Technical details never reach the screen (FR-020).

### Preferences and theme (research R16)

- `index.html` has a small blocking script in `<head>`, before any stylesheet, that reads
  `localStorage['northstar.theme']` (`system` | `light` | `dark`, default `system`) and sets
  `data-theme="light|dark"` on `<html>` (falling back to light on any error).
- `theme.ts` stores the **preference** (never the resolved colour), resolves `system` with
  `matchMedia('(prefers-color-scheme: dark)')` and follows its changes live. `tokens.css` sets
  `color-scheme: light` / `dark` so scrollbars and native controls match.
- Signed in: the person's `theme` and `sidebarCollapsed` drive the screen; a change updates the screen at
  once and saves with `PATCH /api/me/preferences` (only the changed field). A failed save keeps the new
  choice and shows the toast "Your choice couldn't be saved." with **Try again** (re-sends the PATCH) —
  except for `sign_in_required` / `session_expired`, which only run the session-lost flow.
- Signed out (sign-in and public not-found pages): the theme switch changes only
  `localStorage['northstar.theme']`; after sign-in the person's saved preference takes over and the
  visitor's choice is forgotten. After sign-out the sign-in page keeps this browser's last theme (the
  person's own), as a reload would.

### Layout (FR-010 – FR-017, design system §2)

- **Skip link** "Skip to main content" (to `#main`) is the first focusable element of every page —
  `AppLayout` and `PublicLayout` both have it; every page has `<main id="main" tabIndex={-1}>`.
- **Demo banner** (FR-016): signed-in layout — the first element of the content column (right of the
  sidebar, above the page; on phones above the sticky top bar), not sticky; public layout — full width at
  the top. Text from the messages table, `--banner-bg` / `--banner-text`.
- **Sidebar** (≥ 768 px): fixed from top to bottom, `--gray-bg`, 244 px, right hairline. Header 56 px:
  28 px logo tile + "NorthStar" (link to Home, accessible name "NorthStar home"). `<nav aria-label="Main">`:
  Home, Plans, Approvals, All work; the label **More** (10.5 px semibold, upper case, letter-spacing
  .05em, `--ink-2`); Settings, Recycle bin, Activity log. Items: 19 px icon + 13.5 px label, radius 12 px;
  hover moves 2 px right with a 5 % black fill (white 6 % in dark mode; no movement with reduced motion);
  the current item is semibold with a blue icon, a soft blue gradient background and a 3 px blue bar on the
  left (ZenReach), and has `aria-current="page"` (set by `NavLink`; the CSS styles the current item
  through `[aria-current='page']`). Footer (top hairline): **Collapse** (chevron + text);
  then the account button, the bell and the theme switch in a row.
- **Collapsed** (FR-011): 72 px icon rail. Labels stay in the DOM with `.visually-hidden` (never
  `display: none`), so links keep their names; each item has a Radix Tooltip on the right (200 ms delay)
  that is visual only. The More label becomes a hairline. The footer stacks vertically: expand chevron
  (tooltip "Expand sidebar"), avatar-only account button, bell, theme switch (8 px gaps).
- **Narrow screens** (< 768 px — the spec's "narrower than a tablet"): no sidebar; a sticky top bar with
  **Open menu** (tooltip), logo, bell and theme switch. The drawer (Radix Dialog from the left, overlay,
  focus trapped) has a visually hidden title "Menu", `aria-describedby={undefined}`, a **Close menu**
  button without tooltip, always shows the **expanded** sidebar without the Collapse button, shows the
  account button with avatar and the **full name, wrapped** (no ellipsis, no tooltip; accessible name
  "<full name> — account menu"), and shows footer
  controls with text labels ("Notifications", "Dark mode" / "Light mode") instead of tooltips, so one
  Escape always closes it. It closes on Escape, on an overlay click, after choosing an item, and when the
  window becomes 768 px or wider.
- **Main area**: content column offset by the sidebar width, content max width 1120 px with 20 px side
  padding (FR-010); footer "© NorthStar — demo data" (small, `--ink-2`, on `--gray-bg`).
- **Account button** (FR-012, US4, edge case): avatar + first name (ellipsis when long); accessible name
  "<first name> — account menu"; a Radix Tooltip "<full name> — <job title>" (on hover and focus). Menu:
  Radix DropdownMenu with **`modal={false}`** (every DropdownMenu in NorthStar); header as
  `DropdownMenu.Label` (not focusable) with the large avatar, full name, e-mail and job title, which wrap
  instead of being cut (no Tooltip inside menu content); separator; **My profile** (→ `/profile`);
  **Sign out**.
- **Notification bell** (FR-013): icon button "Notifications" with tooltip; Radix Popover titled
  "Notifications" with "You're all caught up."
- **Theme switch** (FR-014): icon button "Switch to dark mode" / "Switch to light mode" (moon / sun, same
  tooltip). It sets the preference to `dark` or `light`.

### Sign-in page (US1, FR-001 – FR-004, FR-009, FR-028)

Public layout: skip link, banner, theme switch top right, then a centred card (max width 440 px, radius
20 px, large soft shadow) on the grey page background — ZenReach's auth modal is the model
(`ZenReach/client/src/components/auth/AuthModal.jsx`). Tab title "Sign in · NorthStar".

1. **Brand header** on `--brand-gradient-strong` with white text: logo tile (white mark on 15 % white) +
   "NorthStar"; `<h1>` "Welcome back" (22 px semibold, as ZenReach); the subtitle.
2. **Notice** (after sign-out or a lost session): a `role="status"` region rendered empty and filled in an
   effect after the page appears (so screen readers announce it).
3. **Demo notice** (blue-tinted box): its two texts, with the password from the API (FR-002).
4. **Form** (`noValidate`, `aria-busy` while signing in): **E-mail** (`type="email"`,
   `autocomplete="username"`), **Password** (`autocomplete="current-password"`) with an eye button whose
   name stays "Show password" and whose state is `aria-pressed` (Edge's own reveal button is hidden with
   `::-ms-reveal`), and the full-width **Sign in** button (`--radius-pill`; "Signing in…" while waiting).
   Empty fields are checked before calling the API: every empty field gets its message and
   `FormErrorSummary` lists them and takes focus. A failed sign-in shows the alert "Incorrect e-mail or
   password." above the form, keeps the e-mail and clears the password. While any sign-in request is in
   flight, clicks on **Sign in** and on every demo person are ignored (nothing is greyed out).
5. **Explore as a demo person** (`<h2>`) with its hint, then one button per demo person: avatar, name, job
   title; accessible name "Sign in as Priya Raman, Marketer". While loading: skeleton rows. States:
   - no demo people (FR-028): the two texts of the messages table, the form still shown;
   - demo sign-in off (`404` from `/api/auth/demo-people`): the messages-table sentence, form and list
     hidden;
   - API unreachable: `ErrorState` with **Try again**.

### Home (US3, FR-018)

Tab title "Home · NorthStar". `PageHeader`: eyebrow "Home", the greeting and description of the messages
table. Then a `Card` "Getting started" (subtitle "The Plan module in five steps") with an ordered list.
Each step: its number in a small `--brand-gradient-strong` circle (`aria-hidden`, because the list already
numbers it), its title as a link (a standalone link, underlined on hover), and one line:

| # | Link (title) | Goes to | Explanation |
|---|---|---|---|
| 1 | Set up Settings | `/settings` | Add the models, activity types, Shell attributes and currencies your plans use. |
| 2 | Create a plan | `/plans` | Give the plan its year, budget, owner and teams. |
| 3 | Add campaigns and initiatives | `/plans` | Split the plan into campaigns, and each campaign into initiatives with their own budgets. |
| 4 | Add activities and line items | `/plans` | Describe each piece of work and plan its costs as line items. |
| 5 | Submit for approval | `/approvals` | Send activities to their approvers and follow their status. |

### Shared building blocks (FR-019, design system §5)

All live in `src/components/`, are exported from `src/components/index.ts`, use tokens only (no colour
literals outside `tokens.css`) and have a component test.

| Component | Props (essentials) | Behaviour |
|---|---|---|
| `Button`, `LinkButton` | `variant: 'primary' \| 'secondary' \| 'danger' \| 'ghost'`, `size: 'md' \| 'sm'`, `icon?` | ZenReach `.btn-pill` shape: radius `--radius-control` (10 px), padding 8 × 16 px, 13.5 px medium. Primary: solid `--button-primary-bg`, hover `--button-primary-bg-hover`, white text; secondary: outlined; danger: `--status-problem-text` text and outline (the confirm button of a delete dialog is filled `--danger-bg` with white text); ghost: text only. Never disabled without a reason next to it |
| `PageHeader` | `eyebrow`, `title`, `highlight?`, `description?`, `actions?` | Blue upper-case 12 px eyebrow; 22 px (26 px from 640 px) bold `<h1>` with an optional `--ink-2` highlight phrase; 13.5 px `--ink-2` description; actions on the right |
| `SummaryCard` | `label`, `value`, `tip?`, `tone?`, `fullValue?` | White card, label 13 px `--ink-2` with InfoTip, value 27 px bold; colour only from `tone`; when `fullValue` differs, the value is a focusable Tooltip trigger whose accessible name is the full value |
| `Card` | `title?`, `subtitle?`, `actions?`, `children` | White, radius 16 px, `--card-border`, `--shadow-sm`, 20–24 px padding; 15 px semibold title, 12.5 px subtitle |
| `StatusChip` | `tone: 'neutral' \| 'review' \| 'positive' \| 'active' \| 'done' \| 'problem'`, `children` | 11–12 px semibold `--status-<tone>-text` on a 12 % tint of `--status-<tone>`, radius 8 px; the word is always shown |
| `CompletenessChip` | `missing: { id: string; label: string }[]`, `onSelectField?` | "All filled in" (positive) or "1 field needed" / "3 fields needed" (problem); a click opens a Popover listing the fields as buttons |
| `InfoTip` | `text`, `label` | Muted ⓘ button, accessible name "About <label>"; Radix **Popover** opened on hover, focus, click and tap; closes on Escape, blur, outside click and when the mouse leaves (after 150 ms, so the mouse can move onto the bubble — WCAG 1.4.13) |
| `Tooltip` | `content`, `side?`, `children` | Radix Tooltip, dark bubble, 12 px; never inside menus or as the only name of a control |
| `ActionsMenu` | `label`, `items: { label, onSelect, tone?: 'danger', separatorBefore?: boolean }[]` | "⋯" button with accessible name "Actions for <label>", Radix DropdownMenu `modal={false}`; Delete last and separated; an item that opens a dialog waits until the menu has closed; focus returns to the "⋯" button when the dialog closes |
| `Avatar` | `initials`, `size?: 'sm' \| 'md' \| 'lg'` (28, 32, 40 px) | Circle with initials on `--brand-gradient-strong`; `aria-hidden` because the name is always next to it |
| `Logo` | `size?: 'md' \| 'lg'` (28 or 32 px tile), `withWordmark?` | NorthStar mark (white four-point star on the brand gradient tile) |
| `SidePanel` | `open`, `onOpenChange`, `title`, `description?`, `children`, `footer` | Radix Dialog from the right, 480–560 px (full width below 640 px), header with a **Close** button (no tooltip), scrolling body, sticky footer; `aria-describedby={undefined}` when there is no description; on closing, focus returns to the control that opened it |
| `ConfirmDialog` | `open`, `onOpenChange`, `title`, `children`, `confirmLabel`, `cancelLabel?` (default "Cancel"), `onConfirm`, `tone?: 'danger'`, `busy?` | Radix AlertDialog; **Cancel** has focus; the confirm button names the action; the dialog stays open until the caller closes it, showing `busy` meanwhile (a second click is then ignored); on closing, focus returns to the control that opened it |
| `ToastProvider` + `useToast()` | `show({ message, action?: { label, altText, onClick }, durationMs? })`, which returns a function that closes the message early | Radix Toast, bottom centre; several messages can be on screen at once; 4 s by default, 10 s with an action (FR-026); pauses on hover and focus; type foreground (announced at once); viewport label "Messages ({hotkey})" |
| `EmptyState` | `icon?`, `title`, `text?`, `action?` | Centred icon, one sentence, the create button |
| `Skeleton` | `width?`, `height?`, `lines?` | `<div role="status">` with a visually hidden "Loading…" and an `aria-hidden` block of grey shimmering shapes (still with reduced motion) |
| `FormErrorSummary` | `errors: { field, message }[]`, `fieldId?(field)` | `--status-problem` tinted box with the messages-table title, links that focus each field; it takes focus when it appears |
| `TextField` / `PasswordField` | `label`, `name`, `value`, `onChange`, `error?`, `hint?`, `marking?: 'Required' \| 'Needed to publish' \| 'Needed to submit'`, `maxLength?` | Label above, marking as a word, error under the field (`aria-invalid`, `aria-describedby`); password with the eye button |
| `ErrorState` | `error`, `onRetry` | Friendly message by error kind (messages table) + **Try again** |
| `DemoBanner` | — | The FR-016 banner |

### Shared hooks (`src/lib`)

- `useUndoableDelete({ invalidate, altText? })` → `afterDelete(result: DeleteResult)`: shows "<Record type>
  '<name>' moved to the recycle bin" with **Undo** for 10 seconds (alt text "Restore it from the recycle
  bin" unless given), **replacing any earlier Undo message** (FR-026, D-037). Undo calls
  `POST /api/recycle-bin/{entryId}/undo` and shows "<Record type> '<name>' restored"; `undo_expired`,
  `restore_parent_first`, `already_resolved` and `name_taken` show their messages. It invalidates the given
  query keys afterwards.
- `useUnsavedChanges(isDirty)`: `useBlocker` while the form is dirty → ConfirmDialog "Discard changes?"
  with **Keep editing** (focused) and **Discard changes**; plus a `beforeunload` listener for reloads and
  tab closes.
- **Edit-form rule for every feature**: when a form opens, it copies the record — including `version` —
  into form state once. Background refetches never change the form. Only the conflict handler replaces
  `version` with the refreshed one, after showing `conflictMessage`, and keeps the typed values.

### Developer pages (research R23; demo mode only)

**Sample records** (`/dev/sample-records`) — the test bench for the shared behaviours of US5:

- `PageHeader`: eyebrow "Developer", title "Sample records", description "Try the shared create, edit,
  delete, Undo and activity-log behaviour. Not part of the Plan module." Primary action **New sample
  folder**.
- Three `SummaryCard`s: Sample folders, Sample notes, In the recycle bin.
- Card "Sample folders": a table — Name (and ID), Notes (each note with an icon button "Delete note
  '<title>'" with tooltip), Last changed ("Priya Raman, 18 Oct 2026, 14:32"), and an `ActionsMenu` per row:
  **Edit**, **Add note**, separator, **Delete**. Empty state from the messages table with **New sample
  folder**.
- Create and edit in a `SidePanel` ("New sample folder" / "Edit sample folder"; **Name** marked Required;
  **Save**, **Cancel**), following the edit-form rule, `useUnsavedChanges`, all errors at once, and the
  conflict handling above. **Add note** in a `SidePanel` ("Add note", **Title**).
- Delete a folder: `ConfirmDialog` with the messages-table texts (with and without notes), **Cancel** and
  **Delete sample folder** (danger). Delete a note: `ConfirmDialog` "Delete sample note '<title>'?" with
  "It goes to the recycle bin. You can restore it.", **Cancel** and **Delete sample note**. Then
  `useUndoableDelete` (alt text "Restore it from Recently deleted on this page").
- Card "Recently deleted": each unresolved bin entry — "Sample folder 'Ideas' (SF-1001), with 2 sample
  notes — deleted by Priya Raman, 18 Oct 2026, 14:32" — with **Restore** (accessible name "Restore sample
  folder Ideas"); a refused restore shows the API's message in a toast; a successful one shows "<Record
  type> '<name>' restored" plus the API's details.
- Card "Activity log": the latest 20 entries, each "Priya Raman created sample folder 'Ideas' (SF-1001)",
  the path when there is one ("in Ideas"), the time, and for changes "Name: Ideas → Ideas 2".
- Toasts: the sample texts of the messages table.

**Shared building blocks** (`/dev/components`): `PageHeader` (eyebrow "Developer", title "Shared building
blocks", description "Every shared component of design system §5 in all its variants."), then one `Card`
per component of the table above showing every variant — all button variants, the six status tones
(Draft, Submitted, Approved, In progress, Complete, Declined), both completeness states, four summary cards
with tips, avatar sizes, info tip, actions menu, empty state, skeleton, a text field with an error and the
error summary, error state — and buttons that open the side panel, the confirm dialog, a toast and a toast
with Undo.

### Tokens (design system §3 – §4 plus research R17)

`src/styles/tokens.css`, light on `:root` (with `color-scheme: light`) and dark on
`:root[data-theme="dark"]` (with `color-scheme: dark`):

| Token | Light | Dark |
|---|---|---|
| `--page-bg` · `--surface` · `--gray-bg` | `#ffffff` · `#ffffff` · `#f5f5f7` | `#0b0b0d` · `#1c1c1e` · `#161618` |
| `--ink` · `--ink-2` · `--muted` | `#1d1d1f` · `#6e6e73` · `#86868b` | `#f5f5f7` · `#a1a1a6` · `#8a8a8e` |
| `--hairline` · `--card-border` · `--chip` | `#d2d2d7` · `rgba(0,0,0,.07)` · `rgba(0,0,0,.05)` | `#38383a` · `rgba(255,255,255,.1)` · `rgba(255,255,255,.09)` |
| `--hover` (background of a hovered item) | `rgba(0,0,0,.05)` | `rgba(255,255,255,.06)` |
| `--blue` · `--link` | `#0071e3` · `#0066cc` | `#2a91ff` · `#59a8ff` |
| `--button-primary-bg` · `--button-primary-bg-hover` · `--danger-bg` | `#0071e3` · `#0062c4` · `#d03b3b` | same |
| `--on-accent` (text and icons on blue, red and brand-gradient surfaces) | `#ffffff` | same |
| `--brand-gradient` | `linear-gradient(135deg, #2a78d6, #9f4bd6)` | same |
| `--brand-gradient-strong` | `linear-gradient(135deg, #256abc, #8c42bc)` | same |
| `--status-neutral` / `-text` | `#86868b` / `#636366` | `#86868b` / `#b0b0b5` |
| `--status-review` / `-text` | `#c77700` / `#8f5400` | `#c77700` / `#f0a640` |
| `--status-positive` / `-text` | `#0ca30c` / `#077307` | `#0ca30c` / `#46d246` |
| `--status-active` / `-text` | `#0071e3` / `#0060c0` | `#0071e3` / `#5aa9ff` |
| `--status-done` / `-text` | `#5b6b7a` / `#4d5c6a` | `#5b6b7a` / `#a9b8c6` |
| `--status-problem` / `-text` | `#d03b3b` / `#b42c2c` | `#d03b3b` / `#ff7b7b` |
| `--banner-bg` · `--banner-text` | `#fff4ce` · `#1d1d1f` | `#3a2f00` · `#f5f5f7` |
| `--overlay` (behind dialogs, side panels and the drawer) · `--tooltip-bg` · `--tooltip-text` | `rgba(0,0,0,.3)` · `#1d1d1f` · `#ffffff` | `rgba(0,0,0,.6)` · `#3a3a3c` · `#f5f5f7` |
| `--level-plan` … `--level-line-item` | `#2a78d6`, `#4a3aa7`, `#1baf7a`, `#e87ba4`, `#86868b` (used from 004) | same |
| shadows | `--shadow-sm`, `--shadow-md`, `--shadow-lg` as ZenReach `index.css` | ZenReach dark values |
| shape and size | `--radius-card` 16 px, `--radius-control` 10 px, `--radius-chip` 8 px, `--radius-pill` 999 px, `--content-max` 1120 px, `--sidebar` 244 px, `--sidebar-collapsed` 72 px | same |
| motion and font | `--ease-out` `cubic-bezier(0.22, 1, 0.36, 1)`, `--dur` 0.24 s; `--font` `-apple-system, BlinkMacSystemFont, "Segoe UI", Inter, system-ui, sans-serif` | same |

White text on a gradient is checked by hand (axe skips gradients): on `--brand-gradient-strong` it is
5.4:1 and 5.8:1.

### Deviations from ZenReach (on purpose)

| ZenReach | NorthStar | Why |
|---|---|---|
| Drawer below 1024 px | Drawer below 768 px | The spec says "narrower than a tablet" |
| Inter from Google Fonts | System font stack (Segoe UI on Windows) | Design system §3; no external font download |
| Gradient primary button `#1a86ff → #0071e3` | Solid `#0071e3`, hover `#0062c4` | White text on `#1a86ff` is only 3.6:1 |
| `--muted` for small text and the More label | `--ink-2` | `--muted` is 3.3–3.6:1 |
| Theme in `localStorage` | Per person on the server | D-038 |
| Sign-in in a modal over a landing page, with sign-up | A sign-in page, no sign-up | FR-001, D-004 |
| MUI components | Radix primitives + CSS Modules | Research R13, R14 |

## User-visible messages

Every message appears here once; code, tests and the quickstart quote this table. `<…>` marks a value
filled in at run time.

| Where | Text |
|---|---|
| Demo banner | Demo environment — sign-in and data are for demonstration only |
| Footer | © NorthStar — demo data |
| Skip link | Skip to main content |
| Sign-in heading / subtitle | Welcome back / Sign in to plan your marketing work. |
| Demo notice | Demo sign-in — Shell single sign-on comes later · All demo people share the password `<password>`. |
| Demo-person hint | Choose a person to sign in as them — no typing needed. |
| Sign-in button while waiting | Signing in… |
| Empty e-mail / empty password | Enter your e-mail / Enter the password |
| Failed sign-in | Incorrect e-mail or password. |
| After sign-out / after a lost session | You have been signed out. / Please sign in again |
| No demo people (FR-028) | No demo users found — load the demo data first · Stop NorthStar's API, run `go run ./cmd/seed` in the backend folder, start the API again with `go run ./cmd/api`, then reload this page. |
| Demo sign-in off | Demo sign-in is turned off in this environment. Shell single sign-on comes later. |
| Loading (visually hidden) | Loading… |
| Server unreachable | NorthStar can't reach its server right now. Check that it is running and try again. · **Try again** |
| Any other failure without an API message | Something went wrong. Please try again. · **Try again** |
| Error summary title | 1 problem needs your attention / <n> problems need your attention |
| Bell | Notifications · You're all caught up. |
| Coming soon | Coming soon · This part of NorthStar isn't built yet. · **Go to Home** |
| Not found | Page not found · There's no page at this address. Check the address, or go back to Home. · **Go to Home** |
| Home | Welcome back, <first name>. · Here's how planning works in NorthStar — follow the steps in order. · Getting started · The Plan module in five steps |
| Preference not saved | Your choice couldn't be saved. · **Try again** |
| Another tab switched person | You're now signed in as <name> — this was changed in another tab. |
| Conflict (FR-029), web | Not saved — <person> changed this <noun> at <HH:mm> (when not today: … on <DD MMM YYYY> at <HH:mm>) · Your changes are still here. Save again to replace their changes, or Cancel to see theirs. |
| Conflict, API message | Not saved — <person> changed this <noun> |
| Unsaved changes | Discard changes? · Your changes haven't been saved. · **Keep editing** · **Discard changes** |
| Delete message / Undo done | <Record type> '<name>' moved to the recycle bin · **Undo** (alt text: Restore it from the recycle bin) / <Record type> '<name>' restored |
| Undo too late | Undo is no longer available — restore it from the recycle bin |
| Undo refused, name taken | Can't undo — there is now another <record type> called '<name>'. Restore it from the recycle bin instead. |
| Parent missing | Restore '<parent name>' first |
| Already undone or restored, API message | '<name>' is no longer in the recycle bin. |
| Already undone or restored, web | '<name>' is no longer in the recycle bin — <person> restored it at <HH:mm> (undone: … <person> undid the delete at <HH:mm>; when not today: … on <DD MMM YYYY> at <HH:mm>) |
| Unknown bin entry / kind not available | This item is not in the recycle bin. / This item can't be restored here. |
| Restore note | Renamed to '<new name>' because the name was taken. |
| Sample page | Sample folder '<name>' created · Sample folder '<name>' saved · Sample note '<title>' added · No sample folders yet — create your first sample folder · Delete sample folder '<name>'? · It goes to the recycle bin together with its <n> sample notes. You can restore it. (one note: "its sample note") · It goes to the recycle bin. You can restore it. · Delete sample note '<title>'? · alt text: Restore it from Recently deleted on this page |
| Sample folder / note gone | This sample folder no longer exists — someone may have deleted it. / This sample note no longer exists — someone may have deleted it. |
| Sample validation | Enter a name. · Use 80 characters or fewer. · There is already a sample folder called '<name>'. Choose another name. · Enter a title. · Use 120 characters or fewer. |
| Validation, API message / missing version / invalid theme / invalid limit | Some fields need your attention. / Reload the page and try again. / Choose system, light or dark. / Choose a number from 1 to 200. |
| API: unreadable request / other site / unknown path / internal | NorthStar couldn't read this request. Reload the page and try again. / This request came from another site and was refused. / This page or record doesn't exist. / Something went wrong on our side. Please try again. |
| API: not signed in / session expired / store failed | Please sign in to continue. / Please sign in again / NorthStar could not save its data folder — restart the API. |
| Store: invalid file (`<table>.json`, or `<table>.jsonl` for the activity log) | <folder>/<file> is not valid JSON (line <n>) — fix it or run go run ./cmd/seed -reset. |
| Seed | Loaded the demo data into <folder>: 6 people. · The data folder <folder> already has data. Run go run ./cmd/seed -reset to replace it (this deletes the current data). · The NorthStar API is running — stop it, then run the seed again. · The demo data is never loaded when APP_ENV is production. · Unexpected argument '<arg>' — the options are -data <folder> and -reset. |
| API start-up and config | NorthStar API listening on http://127.0.0.1:<port> (environment: <env>, demo sign-in on\|off) · Port <port> is already in use — stop the other program or set APP_PORT. · APP_ENV must be local, development or production (it is '<value>'). · APP_PORT must be a number from 1 to 65535 (it is '<value>'). · DEMO_PASSWORD can't be blank. |

## Shared files this feature creates and later features extend

| File | Later features add |
|---|---|
| `backend/api/openapi.yaml` | Their paths and schemas |
| `backend/internal/app/app.go` | Their repositories, services, kinds and routes |
| `backend/seed/seed.go` | Their demo records |
| `frontend/src/app/router.tsx`, `frontend/src/app/navigation.ts` | Their pages (replacing "Coming soon") |
| `frontend/src/components/`, `frontend/src/lib/` | New shared parts only with a reason in their plan |
| `frontend/src/api/schema.d.ts` | Regenerated by `npm run api:types` |

## Business rules and the tests that prove them (constitution V)

| Rule | Proven by |
|---|---|
| Sign-in ignores e-mail case and surrounding spaces; the password is exact | `auth/service_test.go` `TestSignIn_EmailIgnoresCaseAndSpaces_FR003`, `TestSignIn_WrongPassword_FR004` |
| Unknown e-mail, wrong password and inactive person give the same answer | `TestSignIn_SameErrorForUnknownWrongOrInactive_FR004` |
| Both empty fields are reported at once | `TestSignIn_EmptyFieldsAllReported_FR004` |
| Sign-in deletes the browser's old session and every expired session | `TestSignIn_ReplacesOldSessionAndPrunesExpired_FR006` |
| Only the token hash is stored | `TestSignIn_StoresOnlyTokenHash_FR006` |
| A session slides; `lastUsedAt` is rewritten at exactly 1 minute; never ends before 12 full hours idle; expires after 12 h 1 min and is deleted | `TestAuthenticate_SlidingExpiry_FR006`, `TestAuthenticate_TouchesAtMostOncePerMinute_FR006`, `TestAuthenticate_NeverExpiresBefore12HoursIdle_FR006`, `TestAuthenticate_ExpiredIsDeleted_FR006` |
| Authenticate does not bring back a signed-out session | `TestAuthenticate_DoesNotBringBackSignedOutSession_FR006` |
| A missing or inactive person's session ends at the next request | `TestAuthenticate_InactivePersonSessionEnds_FR007` |
| Two sessions of one person are independent | `TestSessions_IndependentPerBrowser_EdgeCase` |
| A handler behind the middleware can write | `auth/handler_test.go` `TestRequireSignIn_HandlerCanWrite_FR007` |
| Demo routes and the sample records only in local/development; no session works in production | `app/app_test.go` `TestDemoMode_RoutesOnlyInLocalAndDevelopment_FR009`, `TestSampleRecords_OnlyInLocalAndDevelopment_D036`, `TestDemoModeOff_ExistingSessionRefused_FR009` |
| Every contract operation is served and every route is in the contract; every signed-in operation refuses a request without a session, and only health and the three `/api/auth/*` operations work without one | `TestRoutesMatchContract`, `TestEveryProtectedOperationRequiresSignIn_FR005` |
| Config: APP_ENV values, port range, blank demo password | `app/config_test.go` |
| Requests from another origin, with another Host, with a non-JSON content type or with data after the JSON are refused | `platform/web/web_test.go` |
| Errors always carry `fields` (an empty list); errors returned through a transaction keep their status | `web_test.go`, `errs_test.go` |
| Active people only, sorted by name | `people/service_test.go` `TestActivePeople_ExcludesInactive_FR001` |
| Preferences default to system / expanded; only sent fields change; invalid theme refused | `TestPreferences_Defaults_FR011_FR014`, `TestUpdatePreferences_ChangesOnlySentFields_FR011`, `TestUpdatePreferences_InvalidTheme_FR014` |
| First name and initials ("Cher" → "CH", "Mary Ann Smith" → "MA") | `people/model_test.go` `TestPerson_FirstNameAndInitials_FR001` |
| Store: all-or-nothing; reads its own writes; failed Write leaves the old value; returned values are copies; survives reopen; joins nested transactions; Write inside Read fails; commit order and rename failure; no open file handles; missing folder and table; invalid JSON message; append-only log with a partial last line | `platform/store/store_test.go` |
| IDs start at 1001, increase, two in one transaction differ, never reused (also after delete) | `platform/ids/ids_test.go` `TestNext_StartsAt1001_FR027`, `TestNext_TwiceInOneTransaction_FR027`, `TestNext_NeverReused_FR027` |
| Version check: missing version refused; out-of-date save refused naming who and when; no change → no version, no log | `platform/record/record_test.go`, `samples/service_test.go` `TestUpdateFolder_MissingVersion_FR029`, `TestUpdateFolder_VersionConflict_FR029`, `TestUpdateFolder_NoChangeNoLog_FR022` |
| One log entry per change with actor, time, type, ID, name, path, action, verb, readable changed fields | `samples/service_test.go` `TestFolderLifecycle_OneLogEntryEach_FR022_SC004`; `history/service_test.go` `TestLog_StoresPathAndTexts_FR022`, `TestLog_EveryAction_FR022` (also `moved`, `copied`, `statusChanged`) |
| Entries are never changed or removed | `TestLog_EntriesNeverChangeOrDisappear_FR023` (after delete, Undo, Restore and a reopen) |
| Diff: sorted, bookkeeping ignored, empty values equal | `history/diff_test.go` `TestDiff_*_FR022` |
| Log entries keep their order even in the same millisecond; two entries in one Write | `TestLog_SameMillisecondKeepsOrder_EdgeCase`, `TestLog_TwoEntriesInOneWrite_FR022` |
| Register refuses bad kinds | `TestRegister_RefusesBadKinds` |
| Delete moves the record and its children to one bin entry with path and contents, removes them and logs once | `TestMoveToBin_WithChildren_FR024`, `TestMoveToBin_OneNoteSaysSingular_FR024` |
| Undo restores exactly (same IDs, data, versions) within 2 minutes; refused later; refused when the name was taken | `TestUndo_RestoresExactly_FR026`, `TestUndo_ExpiresAfterTwoMinutes_FR026`, `samples` `TestUndo_NameTakenRefused_FR026` |
| Restore refuses when the parent is missing, naming it, and changes nothing | `TestRestore_ParentMissing_FR025` |
| Undo/Restore of a resolved entry says who and when; an unknown kind answers not found | `TestUndo_AlreadyResolved_FR025`, `TestUndo_KindNotRegistered_FR025` |
| `PrepareRestore` runs on Restore and Undo, its changes and notes are logged | `TestRestore_AppliesPrepareRestore_FR025`, `samples` `TestRestore_NameClashGetsSuffix_FR025` |
| Sample folder name required, ≤ 80 Unicode characters, unique ignoring case and spaces; note title required, ≤ 120 | `samples/service_test.go` `TestCreateFolder_Validation_FR020`, `TestCreateNote_Validation_FR020` |
| Seed: six active people through the counter (next is U-1007), with the given time; refuses a folder with data, production and a running API | `seed/seed_test.go`, `cmd/seed/main_test.go` (FR-028) |
| Dates `DD MMM YYYY` (never "Sept"), date-only strings not shifted; money formats | `frontend/src/lib/format.test.ts` (FR-021) |
| `safeNext` keeps internal paths and queries, refuses everything else | `frontend/src/lib/safeNext.test.ts`: `//evil.com`, `/\evil.com`, `https://evil.com`, `javascript:alert(1)`, `/sign-in?next=/plans` → `/`; the decoded forms of `/%5Cevil.com` and `/%09/evil.com` → `/`; `/plans?status=draft&year=2026` unchanged (FR-005) |
| `call()` error kinds | `frontend/src/api/errors.test.ts` (empty 500, text/plain 502, JSON 500, JSON 401 `session_expired`, 204, TypeError, AbortError) (FR-020) |
| Conflict and resolved messages in local time (fake timers, UTC) | `errors.test.ts` (FR-029) |
| One session end, one notice | `SessionProvider.test.tsx` (sign out → "You have been signed out." only; several 401s → one redirect) |
| Edit form keeps the version it started from | `SampleRecordsPage.test.tsx` (FR-029) |

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001, FR-002 | `SignInPage`, `GET /api/auth/demo-people`, `people.ActivePeople` | `SignInPage.test.tsx`, `auth/handler_test.go`, people tests, e2e `sign-in.spec.ts` |
| FR-003, FR-004 | `auth.Service.SignIn`, `SignInPage` | auth unit tests, `SignInPage.test.tsx`, e2e |
| FR-005 | `RequireSignIn` (web) + `auth.RequireSignIn` (API), `safeNext` | `RequireSignIn.test.tsx`, `safeNext.test.ts`, `TestEveryProtectedOperationRequiresSignIn_FR005`, e2e |
| FR-006 | sessions, cookie, sign out | auth unit tests, `SessionProvider.test.tsx`, e2e (reload, new tab, sign out) |
| FR-007 | `auth.RequireSignIn`, `actor`, services | `auth/handler_test.go`, samples log tests |
| FR-008 | no permission checks anywhere | e2e `history.spec.ts`: Daniel renames a folder Priya created; no permission message appears |
| FR-009 | `cfg.DemoMode()`, middleware | `app_test.go`, `SignInPage.test.tsx` (demo off) |
| FR-010 – FR-017 | `AppLayout`, `Sidebar`, `NavDrawer`, `AccountMenu`, `NotificationBell`, `ThemeToggle`, `DemoBanner`, `usePageTitle`, placeholder pages, preferences API | component tests, `people/handler_test.go`, e2e `navigation.spec.ts`, `accessibility.spec.ts` |
| FR-018 | `HomePage` | `HomePage.test.tsx`, e2e `home.spec.ts` |
| FR-019 | `src/components/*`, `/dev/components` | one test per component, e2e axe on the gallery |
| FR-020 | `api/errors.ts`, `FormErrorSummary`, `ErrorState`, `web.WriteError`, session `unreachable` | `errors.test.ts`, component tests, `web_test.go`, e2e `session.spec.ts` (API unreachable via `page.route`) |
| FR-021 | `lib/format.ts` | `format.test.ts` |
| FR-022, FR-024 – FR-026 | `history`, `samples`, `useUndoableDelete`, sample page | history and samples unit + contract tests, e2e `history.spec.ts` |
| FR-023 | append-only log, no update or delete anywhere | `TestLog_EntriesNeverChangeOrDisappear_FR023` |
| FR-027 | `ids` | `ids_test.go`, `TestUndo_RestoresExactly_FR026` (same IDs) |
| FR-028 | `cmd/seed`, `seed`, sign-in empty state | `seed_test.go`, `main_test.go`, `TestDemoPeople_EmptyStore_FR028`, `SignInPage.test.tsx`, quickstart step 21 |
| FR-029 | `record.Meta.CheckVersion`, `conflictMessage`, edit-form rule | record and samples tests, `errors.test.ts`, `SampleRecordsPage.test.tsx`, e2e two-person conflict |

Business-level moves, copies and status changes are written by features 004–008 with `history.Log`;
001 proves the mechanism for every action (`TestLog_EveryAction_FR022`).

## End-to-end tests (`frontend/e2e/`, research R19)

**Configuration** (`playwright.config.ts`): `testDir: './e2e'`, `testMatch: '**/*.spec.ts'`,
`workers: 1`, `use.baseURL: 'http://127.0.0.1:14173'`, Chromium only, `dataDir =
path.join(os.tmpdir(), 'northstar-e2e')`, and two web servers (both `reuseExistingServer: false`,
`timeout: 180000`):

- API: `go run ./cmd/seed -reset && go run ./cmd/api` in `../backend` with
  `env: { DATA_DIR: dataDir, APP_PORT: '18080', APP_ENV: 'local' }`, ready at
  `http://127.0.0.1:18080/api/health`;
- web app: `npm run build && npx vite preview --host 127.0.0.1 --port 14173 --strictPort` with
  `env: { NORTHSTAR_API_URL: 'http://127.0.0.1:18080' }`, ready at `http://127.0.0.1:14173`.

Vitest only includes `src/**/*.test.{ts,tsx}`, so the two never pick up each other's files.

**Isolation rules**: `fixtures.ts` `signInAs(page, email)` signs in through `POST /api/auth/sign-in` with
`page.request` and then resets that person's preferences (`PATCH /api/me/preferences {theme: 'system',
sidebarCollapsed: false}`). Tests that change preferences use a person no other file uses (navigation →
Sara Novak; per-person dark mode → Carlos Mendes, then Mei Lin) and restore the defaults afterwards.
Dark-mode scans use `page.emulateMedia({ colorScheme: 'dark' })` with the `system` theme. Tests never
assert global counts — only their own rows or before/after differences. An unreachable API is simulated
per test with `page.route('**/api/**', …)` (abort, or an empty `text/plain` 500).

| File | Covers |
|---|---|
| `sign-in.spec.ts` | US1 scenarios 1–6: redirect and return (URL `/` after sign-in, not Home's text — kept green by feature 009), each of the six demo people greeted by first name, typed sign-in with odd case and spaces, demo-person click (SC-001), wrong password and unknown e-mail, empty fields, reload and a new tab; banner and tab title on `/sign-in` |
| `navigation.spec.ts` | US2 scenarios 1–7: every nav item, highlight and tab title (SC-003); collapse remembered; drawer at 390 px (Escape once even with focus on a footer control, overlay, item); dark mode remembered per person; first visit follows the computer (`colorScheme: 'dark'` → `data-theme="dark"`); not found (signed in and out, with banner and title); bell "You're all caught up."; the banner does not overlap the sidebar at 1280 × 720; no horizontal scroll at 1280 × 720 and 390 px on Home and `/sign-in` (SC-007) |
| `home.spec.ts` | US3 scenario 1 and every step link |
| `session.spec.ts` | US4: account menu content (long names in a tooltip), sign out + Back, switch to another person (the account button shows the new first name), lost session (cookie removed, then the theme switch) → "Please sign in again" → back on the same page; another tab signs in as someone else → toast; API unreachable → spec sentence and **Try again** recovers |
| `history.spec.ts` | US5 scenarios 1–4 on the sample page: create, rename, add a note, delete through the ⋯ menu → log entries with actor, path and readable changes; the rows and summary cards drop; Undo within 10 s brings back the same IDs and logs `restored`; delete a note then its folder → one Undo message, Undo brings back the folder; restore refused naming the parent; FR-008: Daniel renames a folder Priya created, no permission message; two-person conflict (Priya's page refetches before she saves and still gets the message); unsaved-changes guard on Back; the page stays usable after a menu-opened dialog; no horizontal scroll at 1280 × 720 and 390 px (SC-007) |
| `accessibility.spec.ts` | axe (WCAG 2.2 A/AA) in light and dark mode on: sign-in (plain, with the error summary, with "Incorrect e-mail or password.", with no demo people, demo off, unreachable); Home; a Coming-soon page; Not found (signed in and out); collapsed sidebar; 390 px with the drawer open; account menu open; bell popover open; sample page with the side panel and its errors, the confirm dialog, the Undo message; the gallery. Keyboard-only journeys: sign in, every sidebar item, collapse and expand, theme, bell, account menu → Sign out; on the sample page create, rename, delete and reach **Undo** with F8 within 10 s (SC-006) |

**Written so far (decision D-040)**: `smoke.spec.ts` only — the main journey of US1, US2, US4 and US5, that
Home shows the Getting started card, and axe on the sign-in page and Home in light mode. The six files in
the table are deferred (tasks T055, T060, T063, T075, T078 and T080 stay open).

## Documentation updates in this feature

- `README.md` (root), `backend/README.md`, `frontend/README.md`: how to run, test and where things are.
- `CLAUDE.md` "Active technologies"; `docs/architecture/repository-layout.md`, `overview.md`, ADR-0007;
  `docs/product/design-system.md` §3 – §4 and §9; `docs/sdd/decision-log.md` D-036 – D-039; specs 002 and
  006 conflict wording: done with this plan.
- `docs/sdd/feature-map.md`: 001 status.

## Complexity Tracking

| Choice or exception | Why needed | Simpler alternative rejected because |
|---|---|---|
| **Exception to constitution III**: the developer sample page has no search and no details view | Sample records are a developer test bench for the shared behaviours of US5, not records users manage; feature 004 builds the first full standard CRUD pattern (list, search, details view, actions menu) | Building search and a details view for test data would be unrequested work |
| Developer-only sample records (`samples` package and page) | US5's independent test asks for "a simple demonstration record type provided for testing", and constitution V needs a browser test for this P1 story before any planning record exists | Waiting for plans (004) would leave FR-022 – FR-026 untested in the feature that owns them |
| `history.Kind` registry with path and readable values (ADR-0007) | Deletes and restores must cascade across packages (plan → … → line item) without import cycles; 011 needs path, readable values and comments on entries written by 002–010 | Each package deleting its own children creates import cycles; adding the fields later cannot complete old entries |
| Append-only activity-log file | The log grows with every change (011 plans for 50,000 entries); rewriting it on every save would slow every feature | A normal table rewrites the whole file on every change |
| End-to-end tests use the real clock | They run the real API as a user would; Go tests cover every time rule with a fixed clock, and end-to-end tests never assert exact times | A test clock endpoint would add an unsafe API |
