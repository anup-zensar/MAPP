# Research: Application Shell and Demo Sign-in

**Feature**: `001-app-shell-and-sign-in` · **Date**: 2026-10-18 · **Plan**: [plan.md](plan.md)

Feature 001 fixes the shared technical choices for the whole Plan module (constitution, "Technology
Choices"). Every later feature reuses them; changing one later needs a reason in that feature's
`research.md` (or an ADR when it is hard to reverse). Each decision lists what was chosen, why, and what
else was considered.

The spec has no `[NEEDS CLARIFICATION]` markers, so `/speckit-clarify` was skipped (constitution I).
The plan was reviewed by four independent reviewers (spec coverage, consistency with the project
documents, backend, frontend) before tasks were written; their findings are folded in below.

---

## R1. Go version and HTTP server

- **Decision**: Go **1.27** (the toolchain installed on the team's laptops, current stable). The API uses
  only the standard library: `net/http` server and `http.ServeMux` method-and-path patterns
  (`mux.HandleFunc("POST /api/auth/sign-in", h.signIn)`, `r.PathValue("folderId")`), `encoding/json`,
  `log/slog`, `crypto/rand`, `crypto/sha256`.
- **Rationale**: since Go 1.22 the standard mux matches methods and path parameters, which is all
  NorthStar needs. Zero framework code means one less thing for beginners to learn (constitution VII).
- **Alternatives**: chi, gin, echo (rejected: extra concepts and dependencies for no gain).

## R2. Go dependencies

- **Decision**: one external module, used **only by tests**: `github.com/pb33f/libopenapi-validator`
  (v0.16.x, with `github.com/pb33f/libopenapi` v0.41.x) to check every request and response of the
  contract tests against `backend/api/openapi.yaml`. Tests use the standard `testing` and
  `net/http/httptest` packages only (no assertion library).
- **Money (fixed now, added later)**: `github.com/shopspring/decimal` will be added by the first feature
  that stores money (002). Money travels in JSON as a decimal **string** with a currency code
  (`{"amount": "12500.00", "currency": "USD"}`) and is never a float (constitution VI). No money is
  stored in 001.
- **Rationale**: the validator is already used successfully on the team's machines (module cache),
  checks OpenAPI 3.1 documents and needs one call per request/response.
- **Alternatives**: `getkin/kin-openapi` (fine, but needs a router adapter to find operations);
  hand-written schema checks (rejected: they would drift from the contract); `testify` (rejected: plain
  `if got != want` is idiomatic Go and one less dependency).

## R3. JSON file store (storage now)

- **Decision**: package `internal/platform/store`.
  - **Tables**: every record type is a **table** kept in memory as encoded JSON (`map[id]json.RawMessage`)
    and saved to `backend/data/<table>.json` — an object keyed by record id, pretty-printed so people
    can read it. `Get` and `List` decode a **fresh copy** and `Put` encodes, so callers never share
    memory with the store.
  - **Append-only tables**: the activity log is the one table that grows without limit (feature 011
    plans for 50,000 entries). It is stored as **JSON Lines** (`activity_log.jsonl`, one entry per line):
    `store.Append` adds a record and `store.Scan` reads them in order. A commit appends the new lines to
    the end of the file instead of rewriting it; a half-written last line (a crash during an append) is
    cut off the file (`os.Truncate`) when the store opens, so the next append starts on a fresh line.
  - **Transactions**: `db.Write(ctx, fn)` runs `fn` with exclusive access; `db.Read(ctx, fn)` gives a
    consistent read-only view (one `sync.RWMutex`, released with `defer`, so a panic in `fn` never leaves
    the store locked). A `Write` keeps an **overlay** of its changes per table; `Get`, `List` and `Scan`
    inside the transaction read the overlay first, so a transaction always **sees its own changes** (two
    `ids.Next` calls in one transaction give two different IDs). When `fn` returns an error, the overlay
    is thrown away and nothing changes. `Read` and `Write` return `fn`'s error **unchanged**.
  - **Transaction in the context**: `Read`/`Write` put the transaction in the `context.Context` they pass
    to `fn`; repositories find it there. A `Write` or `Read` called with a context that already holds a
    transaction **joins** it (so one service can call another without a deadlock); a `Write` inside a
    `Read` is an error (`ErrReadOnly`). Code inside `fn` never starts a transaction with a fresh context
    such as `context.Background()` — that would deadlock.
  - **Commit** (all-or-nothing as far as a laptop allows):
    1. encode every changed table; a `Write` that changed nothing writes no file;
    2. write each to `<table>.json.tmp` in the data folder, `Sync`, `Close` — on any error delete the
       temporary files and return the error; memory and files stay unchanged;
    3. rename `counters.json` first, then the other tables in name order; a failed rename is retried for
       up to 2 seconds on every operating system (on Windows, antivirus or indexing can hold a file open
       for a moment);
    4. append the new activity-log lines and `Sync`;
    5. only then apply the overlay to memory.
    If anything fails once files start changing (from the first rename on), memory and disk may differ:
    the store logs it, returns the error and refuses every later `Write` with "NorthStar could not save
    its data folder — restart the API." (reads keep working).
  - **Opening**: `Open(dir)` creates the folder when it is missing, treats a table without a file as
    empty, deletes leftover `*.tmp` files, cuts a half-written last line off an append-only file, and
    stops start-up on a file (`.json` or `.jsonl`) that is not valid JSON with
    "backend/data/people.json is not valid JSON (line N) — fix it or run go run ./cmd/seed -reset."
  - **Seam for PostgreSQL**: services depend on the interface `store.Transactor` (`Read`, `Write`) and on
    their own repository interfaces — never on the JSON store. PostgreSQL later adds a second
    `Transactor` (holding a `*sql.Tx` in the context) and a PostgreSQL implementation of each repository;
    services and the web app do not change (constitution IV, ADR-0004).
  - Typed helpers for JSON repositories: `store.Get[T]`, `store.List[T]` (sorted by id: shorter ids first,
    then alphabetical, so `P-999` comes before `P-1001`; never `nil`), `store.Put`, `store.Delete`,
    `store.Append`, `store.Scan[T]`. Table names are checked (`^[a-z][a-z0-9_]*$`) so no file outside the
    data folder can be touched.
  - **The API reads the data folder only when it starts.** The seed command therefore refuses to run while
    an API answers on the configured port (R22).
- **Rationale**: correct all-or-nothing behaviour (a delete with its children, its recycle-bin entry and
  its log entry is saved together) in about 300 readable lines, with no database server.
- **Alternatives**: one file per record (rejected: many files, harder to keep consistent); SQLite
  (rejected: needs CGO or a large pure-Go driver, and PostgreSQL is the real target); embedded key-value
  stores such as bbolt (rejected: binary files, harder for beginners to inspect).
- **Known limits (accepted for local demos)**: the whole dataset is in memory; the store is sized for
  demo data (thousands of records and tens of thousands of log entries). It is not for production
  (ADR-0004).

## R4. Readable IDs

- **Decision**: package `internal/platform/ids`: a `Generator` whose `Next(ctx, "P")` returns `"P-1001"`,
  then `"P-1002"`… The last number of each prefix is kept behind the repository interface
  `ids.Counters` (`Increment(ctx, prefix) (int, error)`); its JSON implementation uses the `counters`
  table, and PostgreSQL later adds its own (constitution IV: all storage goes through repository
  interfaces). Counters only ever increase, so IDs are never reused, even after deletion or permanent
  deletion (FR-027, domain model §1). A counter is read and increased inside the caller's write
  transaction (and the transaction reads its own writes), so two creates can never get the same ID.
  Services receive the `*ids.Generator` in their constructor. **Every** record with a readable ID — also
  seed records — gets it from the generator.
- **Prefixes**: `P`, `C`, `I`, `A`, `L` for planning records (features 004–007); `U` people, `LOG`
  activity-log entries, `BIN` recycle-bin entries, `SF`/`SN` the developer sample records. Settings
  records (002) are identified to users by type and name, but still have internal ids.
- **Alternatives**: UUIDs (rejected for planning records: users quote IDs, decision D-019).

## R5. Record bookkeeping and editing safely (optimistic concurrency, FR-029)

- **Decision**: every business record embeds `record.Meta`: `version` (starts at 1, +1 on every saved
  change), `createdAt`, `createdBy`, `updatedAt`, `updatedBy` (`{id, name}` of the actor). Every update
  request carries the `version` the user started from; a missing version or one below 1 answers
  `400 validation_failed` (field `version`: "Reload the page and try again."). When it differs from the
  stored version the service refuses with `409 version_conflict`; the error's `conflict` object names who
  changed the record and when (`changedBy`, `changedAt`) and the server message has no time ("Not saved
  — Daniel Okoro changed this sample folder"), because only the browser knows the user's time zone. A
  save that changes nothing is not a change: no new version and no log entry.
- **Web app**: every `version_conflict` is shown through the shared `conflictMessage()` helper: "Not saved
  — Daniel Okoro changed this sample folder at 14:32" in the user's local time, or "… on 12 Mar 2026 at
  14:32" when the change was not today. The form keeps the typed values, shows the message at the top,
  and refreshes the record's latest version, so **Save** again is a conscious overwrite and **Cancel**
  shows the latest values.
- **Rationale**: one rule for every record type; the actor's name is stored with the record, so the
  message needs no extra lookup and stays true even if a name changes later.
- **Alternatives**: ETag / `If-Match` headers (equivalent but less visible to beginners and harder to
  type in generated clients); last write wins (rejected by FR-029).

## R6. Demo sign-in and sessions

- **Decision**:
  - Sign-in creates a **session**: a random 32-byte token (`crypto/rand`, base64url) sent to the browser
    in an **HttpOnly** cookie `northstar_session` (`Path=/`, `SameSite=Strict`, `Secure` when served over
    HTTPS, no expiry date so it lasts for the browser session). The server stores only the token's
    **SHA-256 hash** (table `sessions`), so a copied data file cannot be used to sign in.
  - **Every signed-in request** is resolved by `auth.Authenticate` inside **one `Write` transaction**: no
    cookie or an unknown token → `401 sign_in_required`; expired (`now − lastUsedAt > 12 h 1 min`,
    because `lastUsedAt` is rewritten at most once a minute and may lag the real last request by that
    much) → the session is deleted and `401 session_expired`; the person missing or inactive → the
    session is deleted and `401 sign_in_required` (feature 003: "open sessions end at their next
    action"); otherwise `lastUsedAt` is updated when it is at least a minute old. The middleware then
    calls the handler with a context that holds the **actor** and **no transaction**, so the handler can
    open its own.
  - Sign-in compares the e-mail ignoring case and surrounding spaces, and the password exactly. Unknown
    e-mail, wrong password and inactive person all give the same `401 incorrect_credentials` "Incorrect
    e-mail or password." (FR-004). Empty fields give `400 validation_failed` listing **both** fields.
    Signing in deletes the session named by the request's cookie (if any) and every expired session.
  - Choosing a demo person on the page signs in with that person's e-mail and the demo password, so there
    is one sign-in endpoint for both ways (FR-003).
  - The demo password is `northstar` (configurable with `DEMO_PASSWORD`, which may not be blank) and is
    returned by `GET /api/auth/demo-people` so the page can show it (FR-002, D-004).
  - **Demo mode** (`APP_ENV` = `local`, the default, or `development`): `GET /api/auth/demo-people` and
    `POST /api/auth/sign-in` exist. In any other environment they are not registered (`404`) and the
    sign-in middleware answers `401 sign_in_required` to every request, even with an old demo session,
    because no sign-in method exists until Entra ID (FR-009). `POST /api/auth/sign-out` always exists:
    it clears the cookie first and answers `204` (`500` only when the store fails).
- **Protection against forged requests**: a browser "site" ignores the port, so `SameSite=Strict` alone
  does not stop another local web page on `127.0.0.1` from sending requests. The API therefore also:
  wraps every route in Go's `http.CrossOriginProtection` (it refuses unsafe requests whose
  `Sec-Fetch-Site`/`Origin` show they come from another origin); refuses requests whose `Host` is not
  `127.0.0.1`, `localhost` or `[::1]` (any port — protection against DNS rebinding); and accepts request
  bodies only with `Content-Type: application/json`. All three answer `400 bad_request`.
- **Rationale**: an HttpOnly cookie keeps the token away from page scripts (no XSS token theft), the
  browser sends it automatically (no token code in the web app) and it survives reloads and new tabs
  (FR-006). Entra ID later replaces only the sign-in page and the middleware that produces the actor
  (ADR-0005).
- **Alternatives**: bearer token in `localStorage` like ZenReach (rejected: readable by any script, more
  client code); JWT (rejected: cannot be revoked on sign-out without a server list anyway, more
  concepts); in-memory sessions (rejected: a restart of the API during development would sign everyone
  out).

## R7. The actor

- **Decision**: a tiny leaf package `internal/platform/actor` defines `actor.Actor{ID, Name}` and the
  context helpers `actor.With(ctx, a)` and `actor.From(ctx)`. The sign-in middleware puts the actor in the
  request context; handlers read it and pass it **explicitly** to every service method — reads included
  (`svc.ListFolders(ctx, a)`, `svc.UpdateFolder(ctx, a, id, input)`), as the architecture overview shows
  (FR-007, constitution VI). The only operations without an actor are the ones that work before an
  actor exists or produce it: `auth.SignIn`, `auth.SignOut`, `auth.Authenticate` and
  `people.ActivePeople` (`GET /api/auth/demo-people` answers `people.ActivePeople` with the password from
  `auth.DemoPassword()`, a setting).
- **Rationale**: services can always record who acted, and adding permission checks later is one line at
  the top of a service method. A leaf package avoids import cycles (`auth` needs `people`; `people` needs
  the actor).

## R8. Activity log and recycle bin core (FR-022 – FR-027)

- **Decision**: package `internal/history` provides the shared mechanism; feature 011 adds the screens.
  It stores, from the first entry on, everything feature 011 shows — entries written by features
  002–010 cannot be completed later (011 Key Entities and Assumptions).
  - **Kinds**: every business package registers each record type it owns as a `history.Kind`: name,
    mid-sentence singular and plural ("sample note", "Shell attribute" — capitalise only the first
    letter at a sentence start), parent kind, and four small functions (take a snapshot, list children of
    a parent, remove, put back), plus optional `Describe` (readable field labels and values) and
    `PrepareRestore` (adjust a record before it is put back). Registration happens once in
    `internal/app`; `Register` returns an error (and the API refuses to start) for an empty or duplicate
    name, a parent that is itself or not registered yet (parents register first, which also rules out
    cycles), or a missing required function. Children are visited in registration order with their ids
    sorted, so results never depend on map order. Later features add further optional hooks when they
    need them (`AfterRestore` 007/008, `BeforeDelete` 008, a restore preview 011) without changing
    existing registrations.
  - **Log**: `history.Log(ctx, actor, LogInput)` appends one **activity-log entry** inside the caller's
    transaction, so the change and its log entry are saved together. History fills in the id and `seq`
    (from the `LOG` counter, which keeps entries written in the same millisecond in order), the time, the
    actor, the kind's label, the **path** (the record's ancestors at that moment, found by walking
    `Kind.Parent` and each snapshot's parent id) and readable **labels and texts** for every change
    (from `Describe`, otherwise the field name in words and a default text, "Not filled in" when empty).
    There is no update or delete of entries anywhere (FR-023).
  - **Diff**: `history.Diff(before, after)` encodes both records as JSON objects, drops `version`,
    `createdAt`, `createdBy`, `updatedAt` and `updatedBy`, compares each top-level field by its compact
    JSON (nested objects and arrays as a whole), treats `null`, missing, `""`, `[]` and `{}` as the same
    empty value, and returns the changes sorted by field name.
  - **One entry per business change** (SC-004): deleting a folder with two notes writes one `deleted`
    entry for the folder, with details "with 2 sample notes".
  - **Recycle bin**: `history.MoveToBin(ctx, actor, kind, id)` snapshots the record and everything under
    it (parents before children), removes them (children first), stores one **recycle-bin entry** (who,
    when, path, parent, counts, snapshots) and logs the delete — all in one transaction.
  - **Undo** (`POST /api/recycle-bin/{entryId}/undo`) puts every snapshot back exactly (same IDs,
    versions and statuses — domain model R14) and is allowed for **2 minutes** after the delete. The toast
    offers it for 10 seconds (FR-026) and pauses while the pointer or keyboard focus is on it, so the
    server window leaves a margin. Later it answers `409 undo_expired` ("Undo is no longer available —
    restore it from the recycle bin"). Undo never changes what it puts back: when someone has meanwhile
    taken the name (a kind's name rule), the kind's `PrepareRestore` refuses the Undo with
    `409 name_taken` and the item stays in the bin (spec edge case, D-037).
  - **Restore** (`POST /api/recycle-bin/{entryId}/restore`) has no time limit and applies each kind's
    restore rules in `PrepareRestore`: a taken name gets the suffix " (restored)" (then " (restored 2)",
    …, as feature 011 FR-022), plans return as Draft and activities as Created when 004/006 register them
    (R14). Feature 011 adds the page, "Restore both" and the restore preview.
  - `PrepareRestore(ctx, restoreContext, snapshot) (notes, error)` runs inside the restore's transaction
    (it may read and change other records), receives the actor, the time and whether it is an Undo, may
    refuse with an error, must update `record.Meta` when it changes a record, and returns short notes for
    side effects ("Renamed to 'Ideas (restored)' because the name was taken."). The `restored` log entry
    lists the top record's changes and adds the notes to its details; the API returns them as
    `RestoreResult.details`.
  - Both refuse, and change nothing, with `409 restore_parent_first` "Restore '<parent name>' first"
    when the parent is missing (FR-025), and with `409 already_resolved` when the entry was already undone
    or restored — the error's `resolution` says how, by whom and when, so the web app can say "'Ideas' is
    no longer in the recycle bin — Priya Raman restored it at 14:05". An entry whose kind is not
    registered in this environment answers `404 not_found` "This item can't be restored here."
  - Resolved entries stay in the table with their `resolution` (without snapshots) and are left out of
    the recycle-bin list.
- **Rationale**: one implementation of a rule every feature needs; the transaction makes "delete, bin and
  log" all-or-nothing; the stored path and readable values keep the log true after renames, moves and
  deletes.
- **Alternatives**: a `deleted` flag on every record (rejected: every query in every feature would have to
  remember to filter it — easy to forget, FR-024); each package deleting its own children (rejected:
  import cycles between parent and child packages); leaving path and readable values to 011 (rejected:
  older entries could never be completed).

## R9. One error format

- **Decision**: every API failure is JSON: `{"code": "...", "message": "...", "fields": [...]}` plus
  `conflict` (only for `version_conflict`), `parent` (only for `restore_parent_first`) and `resolution`
  (only for `already_resolved`). `code`, `message` and `fields` are **always** present (`fields: []`
  when there are no field problems). Messages are plain language for users; codes are for the web app's
  logic. Unknown `/api` paths (and wrong methods) answer JSON `404 not_found`; panics answer
  `500 internal_error` ("Something went wrong on our side. Please try again.") and the details go only to
  the server log. An unreadable request answers `400 bad_request` with "NorthStar couldn't read this
  request. Reload the page and try again." and the decoder's detail goes only to the log.

| Code | HTTP | When |
|---|---|---|
| `validation_failed` | 400 | One or more fields are wrong; `fields` lists **all** of them (FR-020) |
| `bad_request` | 400 | Body not JSON, wrong content type, unknown fields, data after the JSON, too large; refused cross-site or wrong-host request |
| `sign_in_required` | 401 | No session, unknown session, person gone or inactive, demo mode off |
| `session_expired` | 401 | The session expired after 12 hours without activity |
| `incorrect_credentials` | 401 | Sign-in failed |
| `not_found` | 404 | Unknown record or path |
| `version_conflict` | 409 | Saved from an out-of-date copy; `conflict` says who changed it and when |
| `restore_parent_first` | 409 | The parent of a record to restore is missing; `parent` names it |
| `undo_expired` | 409 | Undo asked for more than 2 minutes after the delete |
| `already_resolved` | 409 | The recycle-bin entry was already undone or restored; `resolution` says how, by whom, when |
| `name_taken` | 409 | Undo refused because someone has meanwhile taken the record's name |
| `internal_error` | 500 | Anything unexpected |

- **Go rules**: `errs.Error` has `Fields []FieldError` with tag `json:"fields"` (constructors set an
  empty list), `Conflict`, `Parent` and `Resolution` with `omitempty`, `Status` with `json:"-"`.
  `web.WriteError` finds an `*errs.Error` with `errors.As`, so errors returned through a transaction
  keep their status.
- **Rationale**: the web app can show every field problem at once and pick the right friendly message
  from the code; one format is easy to remember.
- **Alternatives**: RFC 9457 problem details (good standard, but more fields than users need and less
  obvious for beginners).

## R10. API conventions

- REST + JSON under `/api`, plural resource names, `camelCase` fields.
- Lists answer `{"items": [...], "total": n}` (the sign-in page's `GET /api/auth/demo-people`, which is
  not a list of business records, answers `{"password", "people"}`); creates answer `201` with the
  record; updates answer the updated record including recalculated values; deletes of business records
  answer `200` with `{"recycleBinEntry": {...}}` so the toast can offer **Undo**.
- **Arrays are never `null`**: every slice in a stored record or an API answer is created non-nil
  (`store.List` returns `[]T{}`; builders use `make([]T, 0)`), because Go encodes a nil slice as `null`.
  A nullable object is written as `null` only where the contract says so (`RecycleBinEntry.parent`).
- **Stored records and API answers are separate types**: `model.go` holds the stored record
  (`samples.Folder`, `people.Person`) and, separately, what the API returns (`samples.FolderView` =
  folder + notes, `people.CurrentUser`, `people.DemoPerson`). Stored records never contain lists of
  their children.
- Updates use `PATCH` with the fields to change plus `version`. A `PATCH` decodes optional fields into
  pointers (`*string`, `*bool`), so "not sent" (unchanged) differs from `false` or `""`.
- Timestamps: RFC 3339 in UTC; fractional seconds have up to 3 digits and trailing zeros are dropped
  (Go's encoding). Never compare or sort timestamps as text — parse them, or use `seq` for the log.
  Calendar dates: `YYYY-MM-DD`. Money: decimal string plus currency code (from 002).
- Text lengths count Unicode characters after trimming: `utf8.RuneCountInString` in Go, `[...value].length`
  in the web app.
- Every `/api` response has `Cache-Control: no-store` (so the browser never shows signed-out users cached
  data) and `X-Content-Type-Options: nosniff`. Request bodies are limited to 1 MB.

## R11. Contract first: OpenAPI 3.1 and contract tests

- **Decision**: `backend/api/openapi.yaml` (OpenAPI **3.1.0**) is the single source of truth. Feature 001
  writes it from [contracts/openapi.yaml](contracts/openapi.yaml); each later feature adds its paths and
  schemas. The small Go package `northstar/api` embeds the file (`//go:embed`) for the tests.
  - **Contract-test helper** `internal/platform/apitest`: builds the real application (`app.New`) on a
    temporary data folder (seeded with the demo people unless `WithoutSeed()` is given or the folder
    already has data — seeding arrives with the seed package in US1), with a fixed clock that tests can
    move, and checks **every** request and response it sends with libopenapi-validator, created with
    `config.WithFormatAssertions()` (so
    `date-time` is checked) and `config.WithoutSecurityValidation()` (the API's own middleware enforces
    sign-in and has its own tests). Options: `SkipRequestValidation()` for deliberately invalid requests
    (missing cookie, bad body), `SkipValidation()` for paths or methods that are not in the contract,
    `RawBody([]byte)` for malformed JSON (always sent as `application/json`), `WithCookie`, `WithHost`
    (requests use `Host: 127.0.0.1` otherwise), `WithEnv` and `WithDataDir` (two test servers sharing one
    data folder, e.g. a session made in `local` and tried in `production`). A `Response` carries
    `Status`, `Header`, `Body`, `Cookie(name)` and `Decode(t, &v)`.
  - Only external test packages (`package <name>_test`) import `apitest`, because it builds the whole
    application. `app.New` keeps no package-level state (a new router and a new `history.Service` each
    time), so many test servers can exist in one test binary.
  - **Coverage**: every handler registers its routes through `web.Router`, which records `"METHOD /path"`;
    `app.New` returns the handler and that list (`App.Routes`). `app_test.go` compares the list with the
    contract's operations in both directions, and sends every operation that needs a session without a
    cookie, expecting `401 sign_in_required` (FR-005). Both tests are added once every package is wired
    (after US5).
  - **Web app**: `openapi-typescript` generates `frontend/src/api/schema.d.ts` (committed, never edited
    by hand; `npm run api:check` fails when it is out of date) and `openapi-fetch` gives a typed client in
    which paths, parameters and bodies are checked by TypeScript.
- **Rationale**: constitution IV ("the two sides cannot drift apart") and V ("every endpoint checked by a
  contract test").

## R12. Web app stack and versions

Exact versions are pinned by `frontend/package-lock.json`; the ranges below are what `package.json`
allows. Chosen 18 Oct 2026 from the npm registry, checking that peer dependencies match.

| Package | Version | Why |
|---|---|---|
| react, react-dom | ^19.3.1 | Product owner's choice (ADR-0002) |
| typescript | ~5.9.3 | Strict mode. TypeScript 7 is out but `typescript-eslint` (< 6.1) and `openapi-typescript` (^5) do not support it yet |
| vite | ^8.3.4 | Build and dev server (stable since March 2026) |
| @vitejs/plugin-react | ^6.1.2 | React support for Vite 8 |
| react-router | ^8.4.0 | Routing as a **data router** without loaders or actions (`createBrowserRouter`, `RouterProvider`, `NavLink`, `useBlocker` for unsaved changes); v8 only removed APIs we do not use; everything is imported from `react-router` (`RouterProvider` from `react-router/dom`) |
| @tanstack/react-query | ^5.104.4 | Server state: loading, errors, refresh, cache invalidation after saves |
| radix-ui | ^1.8.1 | Accessible, unstyled primitives: dialog, alert dialog, dropdown menu, popover, tooltip, toast |
| lucide-react | ^1.59.0 | Line icons (tree-shaken) |
| openapi-fetch | ^0.17.0 | Typed `fetch` client generated from the contract |
| openapi-typescript (dev) | ^7.13.0 | Generates `schema.d.ts` from the contract |
| vitest (dev) | ^5.0.5 | Unit and component tests (works with Vite 8) |
| jsdom (dev) | ^30.1.3 | Browser-like test environment |
| @testing-library/react, user-event, jest-dom, dom (dev) | ^16.3.3, ^14.6.8, ^7.0.1, ^10.4.2 | Tests that act like a user |
| @playwright/test (dev) | ^1.64.1 | End-to-end browser tests (Chromium already installed for this version) |
| @axe-core/playwright (dev) | ^4.14.0 | Automated WCAG 2.1 AA checks (SC-006) |
| eslint, @eslint/js (dev) | ^10.13.0, ^10.0.1 | Linting (ESLint 9 is no longer supported) |
| typescript-eslint, eslint-plugin-react-hooks, eslint-plugin-react-refresh, globals (dev) | ^8.71.2, ^7.1.1, ^0.5.7, ^17.13.0 | Recommended rule sets |
| prettier (dev) | ^3.9.10 | One code format |
| @types/react, @types/react-dom, @types/node (dev) | ^19.3.0, ^19.3.0, ^24 | Types (Node 24 is the team's runtime) |

- **No accessibility lint plugin**: `eslint-plugin-jsx-a11y` (last released 2024) supports only ESLint ≤ 9,
  which is no longer supported, and its fork is a young 0.x package. Accessibility is checked instead by
  axe on every screen and overlay (R17) and by component tests that find controls by role and
  accessible name (they fail when a control has no name).
- **Not used, on purpose**: no CSS framework, no component library, no form library, no global state
  library, no mock-server library. Each would add concepts a beginner must learn; feature 004 (the first
  long form) may propose a form library in its own `research.md`.
- **Node**: 24 LTS (`engines.node: ">=22.22"`, required by React Router 8 and Vitest 5).

## R13. Styling: CSS Modules and design tokens

- **Decision**: plain CSS. `src/styles/tokens.css` defines every token of design system §3–§4 (plus R17)
  as CSS custom properties for light (`:root`) and dark (`:root[data-theme="dark"]`);
  `src/styles/global.css` sets the base (font, colours, focus ring, reduced motion). Each component has
  its own `Name.module.css` (CSS Modules, built into Vite). Class names are combined with a 5-line
  `cx()` helper. No colour literals outside `tokens.css`.
- **Rationale**: CSS that reads like the design system, scoped per component, no utility-class soup and
  no runtime styling library. ZenReach's look (white, `#f5f5f7` sidebar, 16 px cards, blue eyebrow,
  brand gradient) is reproduced from its CSS values (`ZenReach/client/src/index.css`, `Layout.jsx`).
- **Alternatives**: Tailwind (ZenReach uses it, but long class strings are hard for beginners to read
  and review), Material UI (ZenReach uses it; heavy and opinionated), CSS-in-JS (runtime cost, extra
  library).

## R14. Accessible behaviour: Radix primitives

- **Decision**: build dialogs, the delete confirmation (alert dialog), the side panel (a dialog styled as
  a right-hand drawer), the mobile navigation drawer, the account menu (dropdown menu), the notification
  popover, the info tip (popover), tooltips and toasts on **Radix** primitives from the single
  `radix-ui` package, styled with our CSS.
- **Rationale**: focus trapping, Escape, outside click, `aria-*` wiring and screen-reader announcements
  are hard to get right by hand; SC-006 requires keyboard use and WCAG 2.1 AA.
- **Known quirks** (from the team's previous project): a Radix tooltip inside a dialog swallows the first
  Escape — so close buttons inside dialogs, side panels and the drawer have no tooltip, and footer
  controls inside the mobile drawer show text labels instead of tooltips; Radix menus take their
  accessible name from the trigger.

## R15. Server state, routing and session in the web app

- **Server state**: TanStack Query. Query keys are plain arrays (`['session']`, `['sample-folders']`).
  `retry: false` (the user gets a **Try again** button instead of a hidden wait); old data stays visible
  while refreshing (design system §5 "Skeleton").
- **Routing**: React Router 8 as a **data router** (`createBrowserRouter` + `RouterProvider`) without
  loaders or actions — the mode in which `useBlocker` works, which every edit form needs to protect unsaved
  changes (design system §6.4). The root route's `errorElement` shows `ErrorState`, so a render error
  never leaves a blank page. `RequireSignIn` wraps every signed-in route; an unsigned visitor goes to
  `/sign-in?` + `new URLSearchParams({ next: pathname + search })` and comes back after signing in
  (FR-005). Where to go after sign-in is decided in one place, `safeNext(raw)`: it parses `raw` with
  `new URL(raw, location.origin)` and returns `pathname + search + hash` only when the origin is
  NorthStar's own and the path is not `/sign-in`, otherwise `/` — so the sign-in page cannot send people
  to another site. Both the redirect to the sign-in page and the navigation after signing in **replace**
  the history entry.
- **Session**: `useSession()` reads `GET /api/me` (query `['session']`, `staleTime` 30 seconds, checked
  again on every navigation and when the window regains focus). Its status: a 401 means `signedOut`; any
  other failure while no person is cached means **`unreachable`** (every page then shows a full-page error
  in the public layout with **Try again**, never a redirect); a failed background check while a person is
  cached keeps `signedIn`. A `GET /api/me` that answers `session_expired` shows "Please sign in again" on
  the sign-in page, also after a reload.
  - **One way to end a session in the web app**, `endSession(reason)`, built inside `SessionProvider`
    from `useNavigate()` and `useQueryClient()`: (1) if the cached session is already empty, stop (several
    401s end it once); (2) navigate to `/sign-in` with `replace` and the notice, adding the current address
    as `next` when the session was lost; (3) set the cached session to empty; (4) remove all other cached
    queries — never `queryClient.clear()`. **Sign out** calls the API and then `endSession('signedOut')`
    ("You have been signed out.", no `next`); the API client's session-lost hook (any call except
    `/api/auth/*` and `GET /api/me` answering `sign_in_required` or `session_expired`) and a
    `GET /api/me` 401 while a person is cached call `endSession('sessionLost')` ("Please sign in again").
    Back after sign-out shows the sign-in page, never the previous data (US4 scenario 1).
  - **Another person in another tab** (spec edge case): when `GET /api/me` returns a different person than
    the one cached, the app removes all other cached data, goes to Home and shows the toast "You're now
    signed in as <name> — this was changed in another tab."
  - A failed preference save shows its toast only for errors other than `sign_in_required` and
    `session_expired`; those run the session-lost flow alone.
- **Unreachable server**: a network failure, or a non-JSON `5xx` answer (the development proxy answers
  that way when the API is not running), becomes the error kind `unreachable`, shown everywhere as
  "NorthStar can't reach its server right now. Check that it is running and try again." with
  **Try again** (edge case).
- **openapi-fetch details**: the client is created with `baseUrl: window.location.origin` (an absolute
  address — the test environment cannot build requests from a relative one) and with
  `fetch: (request) => globalThis.fetch(request)` (openapi-fetch keeps the `fetch` it was given, so tests
  could not replace it otherwise). Error bodies arrive already parsed when they are JSON; network
  failures throw.

## R16. Light and dark mode, and other preferences

- **Decision**: preferences are stored **per person** on the server (table `preferences`):
  `theme` = `system` (default: follow the computer), `light` or `dark`; `sidebarCollapsed` = `false`
  by default (FR-011, FR-014, D-038). `PATCH /api/me/preferences` changes only the fields it receives.
  The web app sets `data-theme="light|dark"` on `<html>`, follows the computer's setting live while the
  preference is `system`, and remembers the last theme in `localStorage` to avoid a flash of the wrong
  colours before the first answer from the API. The **sign-in and public "Page not found" pages** also
  have the light/dark switch; there it only changes the stored browser choice, and after signing in the
  person's saved preference takes over (US2 scenario 5 "any page"). Changing a preference updates the
  screen at once and saves in the background; if the save fails the screen keeps the new choice and a
  toast says "Your choice couldn't be saved." with **Try again**.
- **Rationale**: "remembered per user" — demo people often share one browser during demonstrations.
- **Alternatives**: `localStorage` only (ZenReach; rejected: per browser, not per user).

## R17. Accessibility decisions (SC-006)

Contrast calculations (WCAG 2.1 formula) show that some ZenReach colours are too light **as small
text** on their tints. The design tokens therefore add text variants; the design system's colours stay
for dots, bars, borders and tints.

| Token | Light | Dark | Use |
|---|---|---|---|
| `--status-neutral-text` | `#636366` | `#b0b0b5` | Chip text, neutral |
| `--status-review-text` | `#8f5400` | `#f0a640` | Chip text, amber |
| `--status-positive-text` | `#077307` | `#46d246` | Chip text, green |
| `--status-active-text` | `#0060c0` | `#5aa9ff` | Chip text, blue |
| `--status-done-text` | `#4d5c6a` | `#a9b8c6` | Chip text, slate |
| `--status-problem-text` | `#b42c2c` | `#ff7b7b` | Chip text and error messages, red |
| `--button-primary-bg` | `#0071e3` | `#0071e3` | White text on the primary button stays ≥ 4.5:1 in dark mode (`#2a91ff` would be 3.2:1) |
| `--link` | `#0066cc` | `#59a8ff` | Link text (ZenReach's link blue): ≥ 4.5:1 on white **and** on the grey sidebar, where `--blue` is only 4.3:1 |
| `--brand-gradient-strong` | brand gradient darkened by 12 % | same | Surfaces that carry white text (sign-in header, avatars) |

- All chip text tokens were checked at ≥ 4.5:1 on their 12 % tint over white, `#f5f5f7`, `#1c1c1e` and
  `#161618`.
- Small text uses `--ink`, `--ink-2` or `--link` only; `--muted` is for icons, placeholders and
  decoration (it is 3.6:1 on white). `--blue` text (the eyebrow) is used only on white.
- Focus is always visible: 2 px `--blue` outline with 2 px offset (`:focus-visible`).
- Icon-only buttons have a tooltip and an accessible name; the password eye keeps the name "Show
  password" and reports its state with `aria-pressed`.
- Status is never colour alone (word + colour); motion is removed when the user prefers reduced motion.
- Checks: component tests query by role and accessible name; axe (`@axe-core/playwright`, tags
  `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`) on every screen and every open overlay in light and dark
  mode in the end-to-end tests; keyboard-only journeys.

## R18. Dates and money on screen (FR-021)

- `formatDate("2026-03-12")` → `12 Mar 2026`. Calendar dates are split by hand, never passed to
  `new Date()`, which would shift them by the time zone. Month names come from a fixed list: browsers'
  `en-GB` short months now print "Sept", which the design system does not use.
- `formatDateTime(timestamp)` → `12 Mar 2026, 14:32` in the user's local time; `formatTime` → `14:32`.
- `formatMoney("12500", "USD")` → `$12,500.00`; other currencies → `12,500.00 EUR`; negative →
  `-$12,500.00` (the screen colours it red); empty → `—` (never 0). `formatMoneyCompact` → `$1.25M`,
  `12.5K EUR`, with the full value for the tooltip. Amounts are decimal strings and are formatted with
  `Intl.NumberFormat`, which accepts strings exactly, so no floating-point rounding happens.
- Until feature 003 adds personal preferences, these defaults apply to everyone.

## R19. Testing approach

| Level | Tools | Where |
|---|---|---|
| Go unit tests (every rule) | `testing`, fixed clock | `*_test.go` next to the code |
| Go contract tests (every endpoint) | `apitest` helper + libopenapi-validator | each package's `handler_test.go` (package `<name>_test`) and `internal/app/app_test.go` |
| Component and page tests | Vitest + Testing Library + jsdom, a small `mockApi()` helper that replaces `fetch` | `*.test.tsx` next to the component |
| End-to-end (every P1 story + US4) | Playwright (Chromium) + axe | `frontend/e2e/` |

- **Deterministic**: Go tests use a fixed clock and a fresh temporary data folder. End-to-end tests start
  their own API on `127.0.0.1:18080` with a freshly seeded temporary data folder and the built web app on
  `127.0.0.1:14173`, so a running development server is never touched; they run one at a time
  (`workers: 1`), create their own uniquely named records, reset the signed-in person's preferences
  before each test, and never assert exact times. The one exception to "no real clock": the end-to-end
  API runs on the real clock (recorded in the plan's Complexity Tracking).
- **Web-app tests are time-zone and clock independent**: `vite.config.ts` sets `process.env.TZ = 'UTC'`
  when Vitest runs (`process.env.VITEST`), and tests that depend on "now" fix it with
  `vi.useFakeTimers()` / `vi.setSystemTime(new Date('2026-03-12T09:00:00Z'))`.
- **Unmocked requests fail loudly**: `mockApi()` records any request without a canned answer and the
  test setup fails the test after it, so a missing mock never looks like an unreachable server.
- **Why no MSW**: a 40-line `mockApi()` helper is enough for the few endpoints of each page and is easier
  to read.
- `go test -race` needs a C compiler on Windows (not installed); run plain `go test ./...` locally.

## R20. Tooling and checks

- Go: `gofmt`, `go vet ./...`, `go test ./...`.
- Web app: `npm run check` = generated types up to date + `tsc -b` (app, tests and configs) + ESLint +
  Prettier check + Vitest. `npm run build` type-checks only the app code (`tsconfig.app.json` excludes the
  test files, which `tsconfig.test.json` covers), so the end-to-end build never fails on a test written
  before its code. `npm run test:e2e` runs Playwright.
- No CI pipeline yet: the repository has no remote (constitution: GitHub and CI later). Commands are the
  same that CI will run.

## R21. Running locally

- API: always `127.0.0.1` (no host setting in this phase — constitution, "Servers listen on `127.0.0.1`
  only"); port `APP_PORT` (default `8080`, a number from 1 to 65535); data folder `DATA_DIR` (default
  `data`, git-ignored). On Windows a busy port is recognised by error number 10048 (`WSAEADDRINUSE`) as
  well as `syscall.EADDRINUSE`, and the API prints "Port <APP_PORT> is already in use — stop the other
  program or set APP_PORT."
- Web app: Vite on `127.0.0.1:5173` with `/api` proxied to the API; `vite preview` on `127.0.0.1:4173`.
  The proxy target can be changed with `NORTHSTAR_API_URL`. Ports are strict: if a port is busy the
  program stops with a clear message instead of silently moving.
- `APP_ENV`: `local` (default) or `development` enable demo sign-in and the developer pages;
  `production` disables them. Any other value stops the API with a clear message.
- An empty environment variable counts as not set (its default applies). When several values are
  invalid, the API reports all of them together, one message per line.

## R22. Demo data loader (FR-028)

- **Decision**: `go run ./cmd/seed` (in `backend/`) loads the six demo people of
  `docs/product/demo-users-and-data.md` into an **empty** data folder; `-data <folder>` chooses the folder
  (default `DATA_DIR` or `data`) and `-reset` empties it first (only `*.json`, `*.jsonl` and `*.tmp`
  files). It refuses:
  - a folder that already has data, unless `-reset` is given;
  - to run when `APP_ENV=production` (the demo data never goes into production);
  - to run while an API answers `GET http://127.0.0.1:<APP_PORT>/api/health` within 1 second with
    `200 OK` and NorthStar's health answer (`{"status": "ok", …}`; another program on the port does not
    count, whatever it answers) — "The NorthStar API is running — stop it, then run the seed again." —
    because the running API would overwrite the new files from its memory;
  - an unexpected extra argument ("Unexpected argument '<arg>' — the options are -data <folder> and
    -reset.").
- Package `backend/seed` creates every record through the normal repositories and `ids.Next` inside one
  `Write`, with the time passed in by its caller (`seed.Load(ctx, db, now)`), so `counters.json` ends
  with `U = 1006` and tests can fix the time. Each later feature adds its own records there (and can
  back-date history with the clock).
- The sign-in page explains what to do when there are no demo people (FR-028): stop the API, run the
  seed, start the API again, reload.

## R23. Developer pages (local and development only)

- **Sample records** (`/dev/sample-records`): US5's independent test needs "a simple demonstration record
  type provided for testing", and constitution V needs an end-to-end **browser** test for this P1 story
  (spec Assumptions, D-036). Package `internal/samples` provides two tiny record types — **sample
  folders** and the **sample notes** inside them — that use the shared create / edit (with version
  check) / delete-to-bin / Undo / restore / activity-log behaviour exactly as plans and activities will.
  It is a test bench for these shared behaviours, not the reference for the full standard CRUD pattern:
  feature 004 builds the first complete list, details view and actions menu.
- **Shared building blocks** (`/dev/components`): shows every FR-019 component in its variants, so
  designers can review them in light and dark mode and the end-to-end tests can run axe over all of them.
- Both pages and the sample endpoints exist only in demo mode. The `/dev/*` routes ask
  `GET /api/health`: with `demoSignIn: true` they show the page, otherwise "Page not found". Vite's
  development flag is not used, because the end-to-end tests run the production build. Neither page is in
  the navigation.
- The sample page does not follow the full standard CRUD pattern (no search, no details view): sample
  records are a developer test bench, not records users manage. This exception is recorded in the plan's
  Complexity Tracking; feature 004 builds the first complete pattern.

## R24. Server logs

- `log/slog` text output: one line per request (method, path, status, duration) and errors with their
  details. Never e-mails, names, passwords, tokens or request bodies (constitution VI).
