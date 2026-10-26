# Go Backend Standards

This guide explains how to meet the constitution
([`.specify/memory/constitution.md`](../../.specify/memory/constitution.md)) in the Go backend. The
constitution states the rules; this guide shows how to follow them, with code. References such as
**B3** or **Principle VI** point to the constitution. Where the two disagree, the constitution wins
and this guide is corrected.

The guide is adapted from the Go standards of the separate NorthStar backend repository: SQLite
instead of PostgreSQL, no Docker, access scopes instead of tenants, and the comment budget
(Principle VIII). The samples show the target patterns. They have not yet been compiled against
this repository, whose code is aligned to constitution 2.x by dedicated features (Governance →
Alignment); the first of those features compiles the samples and corrects this guide where needed.
Blocks marked **✗ Avoid** show what not to do.

## Contents

1. [Toolchain and daily commands](#1-toolchain-and-daily-commands)
2. [Folder structure](#2-folder-structure)
3. [Architecture and dependencies](#3-architecture-and-dependencies)
4. [Context, concurrency, and lifecycle](#4-context-concurrency-and-lifecycle)
5. [Errors](#5-errors)
6. [Data access](#6-data-access)
7. [HTTP handlers and the API contract](#7-http-handlers-and-the-api-contract)
8. [Security](#8-security)
9. [Logging, tracing, and health](#9-logging-tracing-and-health)
10. [Configuration](#10-configuration)
11. [Code conventions and complexity](#11-code-conventions-and-complexity)
12. [Testing](#12-testing)
13. [What to avoid](#13-what-to-avoid)
14. [Pull request checklist](#14-pull-request-checklist)
15. [References](#15-references)

---

## 1. Toolchain and daily commands

| Tool | Purpose |
|---|---|
| Go, pinned by `go` and `toolchain` in `go.mod` | Language and standard tooling |
| Gin | HTTP routing, handler layer only |
| SQLite through a pure-Go `database/sql` driver (no cgo) | The database: one local file. The driver is chosen in the SQLite feature's `research.md` |
| sqlc | Generates type-safe Go from SQL (engine `sqlite`) |
| goose | SQL migrations, embedded in the binary |
| OpenTelemetry Go (`otelgin` and a `database/sql` tracer) | Traces and metrics |
| libopenapi-validator (through `internal/platform/apitest`) | Contract tests against `api/openapi.yaml` |
| go-cmp | Test comparisons |
| golangci-lint v2 | Linting and formatting, configured by `.golangci.yml` |
| govulncheck | Known-vulnerability scan of reachable code |
| SonarQube | Quality gate; configured when CI is set up |

There is no Docker in this phase (constitution, Current Phase & Scope). Every test, the store tests
included, runs against SQLite files the test creates itself (§12).

Install the command-line tools once:

```sh
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
# golangci-lint: use the official binary installer (or Homebrew on macOS), not go install
brew install golangci-lint
```

Run these from `backend/`. They are the same checks CI runs (constitution, *Quality gates*):

```sh
golangci-lint fmt ./...            # gofmt + goimports
golangci-lint run ./...            # all linters in .golangci.yml
sqlc generate && sqlc vet          # regenerate queries; must leave no diff
go test -race ./...                # every test, store tests on SQLite included
go test -race -coverprofile=cover.out ./...   # the same, with coverage
govulncheck ./...                  # reachable known vulnerabilities
goose -dir internal/platform/database/migrations create add_campaigns sql   # new migration
go run ./cmd/seed                  # create and migrate the local database, load the demo data
go run ./cmd/api                   # start the API on 127.0.0.1
```

---

## 2. Folder structure

This layout follows the Go team's [Organizing a Go module](https://go.dev/doc/modules/layout)
guidance: commands under `cmd/`, everything else under `internal/`, and packages organised **by
business area, not by technical layer** (constitution B2).

```text
backend/
├── cmd/
│   ├── api/
│   │   └── main.go             # main → run: signals, exit code; nothing else
│   └── seed/
│       └── main.go             # creates and migrates the local database, loads the demo data
├── internal/
│   ├── app/                    # wiring: config, dependencies, routes, server lifecycle
│   ├── budget/                 # one package per business area (plan/, campaign/, ...)
│   │   ├── model.go            # domain types, statuses, invariants and the area's errors
│   │   ├── store.go            # the Store interface the service needs
│   │   ├── store_sqlite.go     # SQLiteStore: sqlc queries ↔ domain types
│   │   ├── service.go          # use cases and business rules (no Gin, no SQL)
│   │   ├── handler.go          # Gin handlers and request/response types
│   │   ├── service_test.go     # unit tests with an in-memory fake Store
│   │   ├── handler_test.go     # HTTP and contract tests with httptest
│   │   └── store_sqlite_test.go  # store tests against a real SQLite file
│   └── platform/               # cross-cutting infrastructure, one package per concern
│       ├── actor/              # the signed-in person, from the request context
│       ├── authz/              # permissions and the teams an actor may act in
│       ├── clock/              # the injectable clock
│       ├── errs/               # failure kinds shared by all areas
│       ├── web/                # error envelope, JSON decoding, pagination, middleware, probes
│       ├── logging/            # slog setup and request-scoped attributes
│       ├── telemetry/          # OpenTelemetry setup
│       └── database/
│           ├── database.go     # Open (with the SQLite settings), Write, Queries, Migrate
│           ├── migrations/     # goose SQL files, embedded with //go:embed
│           ├── queries/        # sqlc input: one .sql file per area
│           ├── sqlcdb/         # sqlc output: generated, committed, never edited
│           └── dbtest/         # creates a migrated SQLite file per test
├── api/
│   └── openapi.yaml            # the API contract (B6)
├── docs/
│   └── GO_STANDARDS.md         # this guide
├── sqlc.yaml
├── .golangci.yml
├── CLAUDE.md                   # loads this guide for coding agents
├── go.mod
└── go.sum
```

Today's packages (plural names, `internal/platform/store` with the JSON stores, `net/http`
handlers) move to this layout feature by feature (constitution, Governance → Alignment).
Architecture decision records live at the repository root in `docs/architecture/adr/`, because
they span backend and frontend (Principle III).

**Rules for the layout:**

- **Package by business area.** Everything about budgets is in `internal/budget`. Adding a
  feature touches one package; deleting it removes one directory. Top-level `models/`,
  `services/`, `controllers/` or `repositories/` directories split one feature across many
  packages and lead to import cycles. An area has at least the files above; a large one gives a
  use case its own file (`copy.go`).
- **`internal/` for everything except `cmd/`.** The compiler stops other modules importing it,
  so internal APIs can change freely. Do not create `pkg/`; nothing here is a public library.
- **No grab-bag packages.** `util`, `common`, `helpers`, `misc`, `base` and `shared` are
  forbidden (B2). Name a package after what it provides (`poll`, `web`, `errs`).
- **No shared `models` package.** Each area owns its domain types. Code generated by sqlc
  (`sqlcdb`) is the only cross-area data package, and only stores import it.
- **Generated code is committed** so that builds need no generator, and CI runs
  `sqlc generate` to prove it is current.
- **Tests sit next to the code** they test. Use the external test package
  (`package budget_test`) so tests exercise the public API, as callers do; handler tests always do
  (B9).
- **`cmd/*/main.go` stays tiny.** It converts `run`'s error into an exit code. Wiring goes in
  `internal/app`, so it can be tested without signals or `os.Exit`.

---

## 3. Architecture and dependencies

Dependencies point inward: **handler → service → store**. The service holds the business rules
and knows nothing about HTTP or SQL. The handler translates HTTP. The store translates SQL.

`depguard` in `.golangci.yml` fails the build if a `service.go` imports Gin, `net/http`,
`database/sql` or `internal/platform/database`.

### Define interfaces where they are consumed

The service declares the `Store` it needs, in `store.go`. The SQLite store satisfies it
implicitly, and tests substitute a fake (B2). Every method that reads plan-owned records takes the
caller's scope, so a record outside it behaves exactly like a missing one (Principle VII).

```go
type Store interface {
	Get(ctx context.Context, scope authz.Scope, id string) (Budget, error)
	List(ctx context.Context, scope authz.Scope, query ListQuery) ([]Budget, error)
	Create(ctx context.Context, in CreateInput, at time.Time) (Budget, error)
	ActivateDraft(ctx context.Context, scope authz.Scope, id string, at time.Time) (bool, error)
}

// Scoper resolves the teams in which an actor holds a permission.
type Scoper interface {
	Scope(ctx context.Context, who actor.Actor, perm authz.Permission) (authz.Scope, error)
}

type Service struct {
	store  Store
	access Scoper
	clock  clock.Clock
}

func NewService(store Store, access Scoper, now clock.Clock) *Service {
	return &Service{store: store, access: access, clock: now}
}
```

Keep interfaces small. Do not create an interface for a type with only one implementation and
no test fake. Return concrete types (`*Service`, `*SQLiteStore`), and accept interfaces.

### Wire dependencies in one place

Constructors receive their dependencies. There are no globals, no `init()` side effects, and no
service locators. All wiring lives in `internal/app`:

```go
func mountBudgets(r gin.IRoutes, db *database.DB, access budget.Scoper, now clock.Clock) {
	service := budget.NewService(budget.NewSQLiteStore(db), access, now)
	budget.NewHandler(service).RegisterRoutes(r)
}
```

---

## 4. Context, concurrency, and lifecycle

### `context.Context` everywhere I/O happens

Anything that performs I/O or may block takes `ctx context.Context` as its first parameter and
passes it on (B3). Handlers take it from `c.Request.Context()`. Never store a context in a
struct, never pass `nil`, and never call `context.Background()` on a request path: that drops the
caller's deadline, cancellation and trace.

### `main` and `run`

`main` only turns an error into an exit code. `run` owns the process lifetime and stops on
SIGINT or SIGTERM.

```go
func main() {
	if err := run(context.Background(), os.Getenv); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// run keeps all process wiring in app.New, so tests can exercise it without
// signals or os.Exit.
func run(ctx context.Context, getenv func(string) string) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := app.New(ctx, getenv)
	if err != nil {
		return err
	}
	defer server.Close(ctx)

	return server.Serve(ctx)
}
```

### Server timeouts and graceful shutdown

`gin.Engine.Run` and `http.ListenAndServe` set no timeouts, so a slow client can hold a
connection open forever. Always build an `http.Server`, and listen on `127.0.0.1` only (constitution,
Current Phase & Scope):

```go
// Without these timeouts a slow or malicious client can hold a connection
// open indefinitely.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

func (a *App) Serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:              net.JoinHostPort(a.cfg.Host, a.cfg.Port),
		Handler:           a.routes(),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ListenAndServe() }()
	slog.InfoContext(ctx, "listening", "addr", srv.Addr)

	select {
	case err := <-serveErr:
		return fmt.Errorf("listen on %s: %w", srv.Addr, err)
	case <-ctx.Done():
	}
	if err := shutdown(ctx, srv, a.cfg.ShutdownTimeout); err != nil {
		return err
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// shutdown runs after ctx is cancelled, so its deadline comes from
// context.WithoutCancel: values such as trace IDs carry over, the
// cancellation does not.
func shutdown(ctx context.Context, srv *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shut down http server: %w", err)
	}
	return nil
}
```

Two details matter. The serving goroutine has an owner that waits for it (`<-serveErr`), and the
channel is buffered, so the goroutine can always finish its send and exit. Splitting out
`shutdown` also keeps `Serve` within the fan-out limit (§11). `App.Close` closes the database
(B3).

### Goroutines

Every goroutine needs an owner that waits for it, a way to stop, and a path for its error (B3).

- To run a group of tasks and stop at the first error, use `golang.org/x/sync/errgroup`. Use
  `errgroup.SetLimit` to bound concurrency.
- For a fixed set of tasks that cannot fail, use `sync.WaitGroup.Go` (Go 1.25+).
- Never start an unbounded number of goroutines per request (for example, one per row).
- Any goroutine that sends on a channel must be able to complete or abandon the send when the
  receiver has gone; see [§13](#13-what-to-avoid).
- Guard shared mutable state with a mutex, or confine it to one goroutine. Only the sender closes
  a channel. CI runs every test with `-race`.

A reusable, cancellable polling loop:

```go
// Until calls check at once and then every interval until check reports
// done, check fails, or ctx ends; then it returns check's error or the
// context's cause.
func Until(ctx context.Context, interval time.Duration, check func(context.Context) (bool, error)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		done, err := check(ctx)
		if done || err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-ticker.C:
		}
	}
}
```

---

## 5. Errors

### Classify failures once, map them to HTTP once

Domain packages classify failures with `errs.Kind`. The transport maps kinds to HTTP status codes
in exactly one function, so no handler ever chooses a status for an error (B4).

```go
type Kind int

// KindInternal is the zero value, so an unclassified error is never mistaken
// for a client error.
const (
	KindInternal Kind = iota
	KindInvalid
	KindNotFound
	KindConflict
	KindUnauthenticated
	KindForbidden
	KindTooLarge
)

// Error is a classified failure whose Code, Message and Fields are safe to
// return to API clients.
type Error struct {
	Kind    Kind
	Code    string       // stable and documented, e.g. "budget_not_found"
	Message string       // plain language, free of internal detail
	Fields  []FieldError // for KindInvalid: every field at fault, reported together
}

type FieldError struct {
	Field   string
	Message string
}
```

Each area declares its own errors next to the code that returns them, in `model.go`:

```go
var (
	// ErrNotFound also covers a budget outside the caller's scope, so its
	// existence is never disclosed (Principle VII).
	ErrNotFound = &errs.Error{Kind: errs.KindNotFound, Code: "budget_not_found", Message: "budget not found"}
	ErrNotDraft = &errs.Error{Kind: errs.KindConflict, Code: "budget_not_draft", Message: "only a draft budget can be activated"}
)
```

Exported `Err…` variables are the one accepted form of package-level variable: the
`gochecknoglobals` linter allows them, and the `reassign` linter blocks reassignment.

### Wrap with context; inspect with `errors.Is` and `errors.AsType`

Add what *this* function was doing, and wrap with `%w` so callers can still match the cause:

```go
func (s *Service) Get(ctx context.Context, who actor.Actor, id string) (Budget, error) {
	scope, err := s.access.Scope(ctx, who, authz.ViewBudgets)
	if err != nil {
		return Budget{}, fmt.Errorf("scope of %s: %w", who.ID, err)
	}
	found, err := s.store.Get(ctx, scope, id)
	if err != nil {
		return Budget{}, fmt.Errorf("get budget %s: %w", id, err)
	}
	return found, nil
}
```

- Use `errors.Is(err, ErrNotFound)` to match a value and `errors.AsType[*errs.Error](err)`
  (Go 1.26+) to extract a type. Never compare error strings.
- Error strings are lowercase, with no trailing punctuation: `"get budget B-7: budget not found"`.
- Translate infrastructure errors at the boundary that understands them. The store turns
  `sql.ErrNoRows` into `ErrNotFound`, so nothing above it knows about SQL.

### Handle each error exactly once

Either return an error or log it, never both. Errors are logged once, at the edge: `WriteError`
logs unexpected failures and returns a generic 500, so SQL text, driver messages and stack traces
never reach a client. Validation failures carry every field at once (B6).

```go
// WriteError is the single place where request errors are logged. An
// unclassified or internal error reaches the client only as a generic 500.
func WriteError(c *gin.Context, err error) {
	appErr, ok := errors.AsType[*errs.Error](err)
	if !ok || appErr.Kind == errs.KindInternal {
		slog.ErrorContext(c.Request.Context(), "request failed", "error", err)
		appErr = errInternal
	}
	body := errorBody{Code: appErr.Code, Message: appErr.Message, Fields: map[string]string{}}
	for _, field := range appErr.Fields {
		body.Fields[field.Field] = field.Message
	}
	c.AbortWithStatusJSON(statusOf(appErr.Kind), errorEnvelope{Error: body})
}
```

### Panics

Do not panic for expected failures. `panic`, `log.Fatal` and `os.Exit` are allowed only in
`main` (the `deep-exit` linter checks this). `panic` is also allowed for a state that can only
come from a programming error. A recovery middleware turns any panic into a logged 500, with the
stack trace in the log and never in the response:

```go
func Recover() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		WriteError(c, fmt.Errorf("panic: %v\n%s", recovered, debug.Stack()))
	})
}
```

---

## 6. Data access

### Opening the database

The API and the seed command open the SQLite file once, through `database.Open`, and pass the
handle to every store. The settings of constitution B5 go in the driver's connection string, so
that every connection the pool opens gets them, not only the first:

- `foreign_keys = ON` — SQLite ignores foreign keys without it;
- `journal_mode = WAL` — readers no longer block the writer;
- a `busy_timeout` (for example 5 s) — a second writer waits instead of failing at once;
- the driver's immediate-transaction option, so every transaction starts with `BEGIN IMMEDIATE`.
  SQLite allows one writer at a time; taking the write lock at the start means two writers queue
  on `busy_timeout` instead of one failing midway with `SQLITE_BUSY`.

The exact connection-string syntax depends on the driver chosen in `research.md`.

```go
func Open(ctx context.Context, path string) (*DB, error) {
	sqlDB, err := sql.Open(driverName, connectionString(path))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err = sqlDB.PingContext(ctx); err != nil {
		return nil, errors.Join(fmt.Errorf("ping database: %w", err), sqlDB.Close())
	}
	return &DB{sql: sqlDB}, nil
}
```

The database file is never committed (`.gitignore`); `go run ./cmd/seed` creates it.

### Migrations (goose)

Schema changes are versioned SQL files in `internal/platform/database/migrations`. A merged
migration is never edited; fix it with a new one. Migrations are forward-only (B5).

```sql
-- +goose Up
CREATE TABLE budgets (
    id            TEXT    PRIMARY KEY,
    team_id       TEXT    NOT NULL REFERENCES teams (id),
    name          TEXT    NOT NULL,
    fiscal_year   INTEGER NOT NULL,
    status        TEXT    NOT NULL DEFAULT 'draft'
                          CHECK (status IN ('draft', 'active', 'closed')),
    planned_minor INTEGER NOT NULL DEFAULT 0,
    currency      TEXT    NOT NULL CHECK (length(currency) = 3),
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL
) STRICT;

CREATE INDEX budgets_team_id_idx ON budgets (team_id, id);

-- +goose Down
DROP TABLE budgets;
```

- Let the database enforce invariants it can check: `NOT NULL`, `CHECK`, foreign keys and unique
  constraints. Application validation gives friendly messages; the database guarantees
  correctness.
- Tables are `STRICT`, so a value of the wrong type fails instead of being stored silently.
- Money is integer minor units (`planned_minor` is in cents), or a canonical decimal string where
  more scale is needed (exchange rates). Never `REAL`, and never a column with `NUMERIC` affinity,
  which may turn a decimal into a float.
- Calendar dates are `TEXT` in `YYYY-MM-DD`; timestamps are `TEXT` in RFC 3339, in UTC. Times
  come from the injected clock, not from `CURRENT_TIMESTAMP`, so tests are deterministic.
- Keep SQL portable: standard SQL, ids and foreign keys, nothing that leans on SQLite's dynamic
  typing or `rowid`. The PostgreSQL move then rewrites queries mechanically (Principle IV).
- `cmd/seed` applies migrations locally, and a deploy step will later. The API never migrates at
  startup; it stops with a clear message if the database is not at the latest migration.

```go
//go:embed migrations/*.sql
var embeddedMigrations embed.FS

func Migrate(ctx context.Context, db *DB) error {
	files, err := fs.Sub(embeddedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db.sql, files)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	if _, err = provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
```

### Queries (sqlc)

All SQL lives in `internal/platform/database/queries/*.sql`. `sqlc` reads the goose migrations
as the schema and generates typed Go into `internal/platform/database/sqlcdb`. A typo in a column
name fails at `sqlc generate`, not in production.

```sql
-- name: GetBudget :one
SELECT * FROM budgets
WHERE id = sqlc.arg(id) AND team_id IN (sqlc.slice(team_ids));

-- name: ListBudgets :many
SELECT * FROM budgets
WHERE team_id IN (sqlc.slice(team_ids))
ORDER BY
    CASE WHEN sqlc.arg(sort_key) = 'name' THEN name END,
    created_at,
    id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: ActivateDraftBudget :execrows
UPDATE budgets
SET status = 'active', updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND status = 'draft' AND team_id IN (sqlc.slice(team_ids));
```

`sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: sqlite
    schema: internal/platform/database/migrations
    queries: internal/platform/database/queries
    gen:
      go:
        package: sqlcdb
        out: internal/platform/database/sqlcdb
        emit_empty_slices: true
```

**Scope isolation is in every query.** Each query on plan-owned data filters by the teams of the
caller's scope, in SQL, and each store has a test proving a record outside the scope is invisible
(§12). `authz.Scope` always lists teams explicitly: "All markets" is resolved to every team, so
queries need one form only.

**Dynamic sorting and filtering** never interpolate client input into SQL. The handler maps
public names to a closed set of keys (`sort_key` above), and the query branches on the key.

**Pagination:** offset pagination (above) is fine for short lists. For large or append-heavy
tables, use keyset pagination (`WHERE (created_at, id) > (?, ?) ORDER BY created_at, id LIMIT ?`),
which stays fast at any depth.

### The store translates between SQL and the domain

```go
func (s *SQLiteStore) Get(ctx context.Context, scope authz.Scope, id string) (Budget, error) {
	row, err := s.db.Queries(ctx).GetBudget(ctx, sqlcdb.GetBudgetParams{ID: id, TeamIds: scope.TeamIDs})
	if errors.Is(err, sql.ErrNoRows) {
		return Budget{}, ErrNotFound
	}
	if err != nil {
		return Budget{}, fmt.Errorf("select budget: %w", err)
	}
	return fromRow(row), nil
}
```

### Avoid check-then-act races

Reading a row, checking it in Go, then updating it lets two concurrent requests both pass the
check. Put the condition in the `UPDATE` and act on the affected-row count:

```go
// Activate returns ErrNotFound if the caller cannot see the budget and
// ErrNotDraft if it is not a draft.
func (s *Service) Activate(ctx context.Context, who actor.Actor, id string) error {
	scope, err := s.access.Scope(ctx, who, authz.EditBudgets)
	if err != nil {
		return fmt.Errorf("scope of %s: %w", who.ID, err)
	}
	// One conditional update makes check-and-change atomic, so two requests
	// cannot both activate the same budget.
	activated, err := s.store.ActivateDraft(ctx, scope, id, s.clock.Now())
	if err != nil {
		return fmt.Errorf("activate budget %s: %w", id, err)
	}
	if activated {
		return nil
	}
	if _, err = s.store.Get(ctx, scope, id); err != nil {
		return err
	}
	return ErrNotDraft
}
```

When the invariant spans rows (for example, "one active budget per fiscal year"), enforce it with
a constraint such as a partial unique index, and map the unique violation to `KindConflict`.

### Transactions

The service owns transaction boundaries (B5), and a change and its activity-log entry commit
together. `database.DB.Write` runs a function in one transaction that it carries in the context,
so every store called with that context — the area's own and the activity log's — joins it:

```go
// Write commits if fn returns nil and rolls back otherwise. Inside a
// transaction it runs fn in the same transaction.
func (d *DB) Write(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := txFrom(ctx); ok {
		return fn(ctx)
	}
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// After a successful Commit, Rollback is a no-op, so this also covers an
	// error or a panic in fn.
	defer func() { _ = tx.Rollback() }()
	if err = fn(withTx(ctx, tx)); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (d *DB) Queries(ctx context.Context) *sqlcdb.Queries {
	if tx, ok := txFrom(ctx); ok {
		return sqlcdb.New(tx)
	}
	return sqlcdb.New(d.sql)
}
```

The service declares the transactor it needs (`Write(ctx, fn) error`) next to its `Store`, as it
does today with `store.Transactor`. Keep transactions short. Never call external services or wait
on user input inside one: SQLite has a single writer, so a long write transaction stalls every
other write.

---

## 7. HTTP handlers and the API contract

### Handlers translate; they do not decide

A handler takes the signed-in actor, decodes, calls one service method, and encodes. It contains
no business rules. Access is checked twice: the route table refuses an operation without its
permission (fail closed), and the service applies the scope (Principle VII).

```go
func (h *Handler) create(c *gin.Context) {
	who, err := actor.Require(c.Request.Context())
	if err != nil {
		web.WriteError(c, err)
		return
	}
	var req createRequest
	if err = web.BindJSON(c, &req); err != nil {
		web.WriteError(c, err)
		return
	}
	created, err := h.service.Create(c.Request.Context(), who, CreateInput(req))
	if err != nil {
		web.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toResponse(created))
}
```

- **Separate request and response types** from domain types, so the JSON contract and the domain
  model can change independently. `json` tags are `camelCase`.
- **Decode strictly.** Unknown fields are rejected, and an oversized body maps to 413:

```go
func BindJSON(c *gin.Context, dst any) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(dst)
	if err == nil {
		return nil
	}
	if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
		return ErrBodyTooLarge
	}
	return ErrMalformedBody
}
```

- **Limit request bodies** for every route, using middleware:

```go
func LimitBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
```

- **Allowlist query parameters** that select behaviour:

```go
// parseSortKey accepts only the listed names, so client input never reaches SQL.
func parseSortKey(raw string) (SortKey, error) {
	switch raw {
	case "", "createdAt":
		return SortByCreatedAt, nil
	case "name":
		return SortByName, nil
	default:
		return "", errs.Invalid("sort", "sort must be one of: createdAt, name")
	}
}
```

### Response envelopes

Every error has the same shape, with a stable `code` that clients branch on, and every invalid
field reported at once:

```json
{ "error": { "code": "validation_failed", "message": "Some fields need attention.",
             "fields": { "name": "Name must be 1 to 200 characters." } } }
```

Every list response uses one generic envelope with an enforced maximum page size:

```go
type ListResponse[T any] struct {
	Items  []T   `json:"items"`
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}
```

### Contract and versioning

- `api/openapi.yaml` (OpenAPI 3.1) is the contract. Update it in the same pull request as the
  handler, then run `npm run api:types` in `frontend/`.
- Routes live under `/api/v1`. Adding fields or endpoints is backward compatible. Removing or
  renaming fields, or changing their meaning, requires `/api/v2`.
- Use the right status codes: `201` with the created resource, `204` for no body, `400` for
  validation, `401` without a session, `403` for an action the actor's roles do not allow on a
  record they can see, `404` for a missing record *or one outside the actor's scope*, `409` for
  state and version conflicts, `413` for oversized bodies.
- For a create that clients may retry, accept an `Idempotency-Key` header and store the key with
  the result, so a retried request returns the original outcome instead of a duplicate.

---

## 8. Security

### The scope comes only from the signed-in actor

MAPP serves one organisation, Shell. What multi-tenant systems call tenant isolation applies
here to access scopes: the teams (markets) in which an actor holds a permission (Principle VII).

```go
// Scope lists the teams in which an actor holds a permission. "All markets"
// is resolved to every team.
type Scope struct {
	TeamIDs []string
}
```

- Never read a scope from a path, query string, body or header. A team id in a request (for
  example the team of a new plan) is input to validate against the scope, never a grant.
- Services pass `scope` explicitly to the store, so forgetting it is a compile error, not a data
  leak.
- A record outside the scope answers 404, exactly like a missing one; 403 only refuses an action
  on a record the actor can see, and names nothing they cannot see.

### Sign-in and sessions

- Sign-in is a demo sign-in until Microsoft Entra ID single sign-on (D-004), switched on only by
  an explicit development setting (B7).
- The session is an opaque random token (`crypto/rand`) in an `HttpOnly`, `SameSite` cookie,
  `Secure` outside local development. The server stores only the token's SHA-256 hash and
  compares with `subtle.ConstantTimeCompare`.
- Middleware answers a bare 401 and never says *why* a session failed.
- When Entra ID arrives, verify its tokens with OIDC discovery (`coreos/go-oidc`), pinning the
  issuer, the audience and the accepted algorithms.

### Cryptography

| Need | Use | Never |
|---|---|---|
| Random tokens, IDs, nonces | `crypto/rand` (`rand.Text()`, `rand.Read`) | `math/rand` (banned by `depguard`) |
| Password storage | `golang.org/x/crypto/argon2` (argon2id) or `golang.org/x/crypto/bcrypt` | Fast hashes (SHA-*, MD5), encryption |
| Storing session, refresh or reset tokens | SHA-256 of the token; compare with `subtle.ConstantTimeCompare` | The raw token |
| Transport | TLS 1.2+ (`tls.Config{MinVersion: tls.VersionTLS12}`) | Plain HTTP outside the developer's own machine |
| Anything else | A vetted library, chosen in the design's threat model | Hand-written cryptography |

### Other rules

- **Secrets** come from environment variables. They never appear in Git, logs, error messages or
  API responses; configuration errors name the variable, never its value.
- **Validate at the boundary:** type, length, range, format and allowlists. Run `govulncheck` and
  `gosec` (via golangci-lint) on every build.
- **Don't trust client-supplied request IDs** or other values that end up in logs; generate them
  server-side (§9).

---

## 9. Logging, tracing, and health

### Structured logs with request context

Use `log/slog` only (`depguard` bans `log`, logrus, zap and zerolog). Logs are JSON on stdout. A
small handler adds request-scoped attributes and the trace ID to every record:

```go
func New(w io.Writer, level slog.Leveler) *slog.Logger {
	json := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(contextHandler{next: json})
}

func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	existing, _ := ctx.Value(attrsKey{}).([]slog.Attr)
	// Clip forces append to copy, so a child never writes through the
	// parent context's slice.
	return context.WithValue(ctx, attrsKey{}, append(slices.Clip(existing), attrs...))
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	out := record.Clone()
	if attrs, ok := ctx.Value(attrsKey{}).([]slog.Attr); ok {
		out.AddAttrs(attrs...)
	}
	if span := trace.SpanContextFromContext(ctx); span.HasTraceID() {
		out.AddAttrs(slog.String("trace_id", span.TraceID().String()))
	}
	return h.next.Handle(ctx, out)
}
```

Middleware attaches `request_id`, and sign-in attaches `actor_id`:

```go
// RequestID ignores any client-supplied ID, so callers cannot inject content
// into the logs.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := rand.Text()
		c.Header(requestIDHeader, id)
		ctx := logging.WithAttrs(c.Request.Context(), slog.String("request_id", id))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
```

**Logging rules** (`sloglint` enforces the first three):

- Use the `…Context` variants (`slog.InfoContext`) whenever a `ctx` is in scope, so request
  attributes are attached.
- Messages are constant strings; data goes in attributes: `slog.InfoContext(ctx, "budget
  activated", "budget_id", id)`.
- Attribute keys are `snake_case`.
- Never log tokens, passwords, secrets, full request bodies or personal data. `actor_id` is an
  id, never a name or an e-mail address.
- Levels: `ERROR` means someone must act; `WARN` means degraded but self-healing; `INFO` marks
  lifecycle and business events; `DEBUG` is off outside development.

### Tracing and metrics

`telemetry.Start` installs the OpenTelemetry providers, configured by the standard `OTEL_*`
environment variables. In the local phase no exporter sends data off the machine (B7). `otelgin`
traces every request, and a `database/sql` tracer traces every query. Add your own spans only
around significant work that is not already an HTTP or database call. Export request rate, error
rate and duration (RED) metrics for every route.

```go
// newEngine installs, in order: panic recovery, tracing, request ID, body limit.
func newEngine(maxBodyBytes int64) *gin.Engine {
	engine := gin.New()
	engine.Use(
		web.Recover(),
		otelgin.Middleware(serviceName),
		web.RequestID(),
		web.LimitBody(maxBodyBytes),
	)
	return engine
}
```

### Health probes

`/healthz` (liveness) checks only that the process serves; if it checked the database, a database
outage would make an orchestrator restart healthy processes. `/readyz` (readiness) pings the
database with a short timeout.

```go
func RegisterHealth(r gin.IRoutes, db Pinger) {
	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			slog.WarnContext(ctx, "readiness check failed", "error", err)
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
```

---

## 10. Configuration

Configuration is read once at startup into a typed struct and validated. Every problem is
reported together, and the process stops. Code never calls `os.Getenv` outside `internal/app` and
`cmd`.

```go
// Load reports every invalid variable in one error and never includes a
// value, because some are secrets.
func Load(getenv func(string) string) (Config, error) {
	env := reader{getenv: getenv}
	cfg := Config{
		Host:            env.text("HOST", defaultHost),
		Port:            env.text("PORT", defaultPort),
		DatabasePath:    env.required("DATABASE_PATH"),
		DemoSignIn:      env.flag("DEMO_SIGN_IN", false),
		LogLevel:        env.level("LOG_LEVEL", slog.LevelInfo),
		MaxBodyBytes:    env.bytes("MAX_BODY_BYTES", defaultMaxBodyBytes),
		ShutdownTimeout: env.duration("SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
	}
	return cfg, env.err()
}
```

Passing `getenv` instead of calling `os.Getenv` makes `Load` testable without mutating the process
environment. Defaults are the safe choice: `defaultHost` is `127.0.0.1`, and development-only
behaviour (demo sign-in, seeding) needs an explicit setting (B7).

---

## 11. Code conventions and complexity

### Naming

| Do | Don't | Why |
|---|---|---|
| `budget.Service`, `budget.NewService` | `budget.BudgetService` | The package name is part of the name; avoid stutter |
| `userID`, `apiURL`, `HTTPClient` | `userId`, `apiUrl`, `HttpClient` | Initialisms keep one case |
| `func (s *Service)` | `func (this *Service)`, `func (self *Service)` | Short, consistent receivers |
| `b.Owner()` | `b.GetOwner()` | Go getters omit `Get` |
| `IsValid()`, `HasAccess()`, `CanEdit()` | `Valid()`, `CheckAccess()` | Boolean functions read as questions (Principle VIII) |
| `budgets []Budget`, `budget Budget` | `budgetList`, `budgetsArr` | Plural for collections; name by meaning, not type |
| `ErrNotFound`, `ValidationError` | `NotFoundErr`, `ErrValidation` type | Error values are `Err…`; error types are `…Error` (`errname`) |
| `maxNameLength = 200` | `200` repeated inline | Named constants for repeated literals (`goconst`) |

### Files and comments

- **One primary concept per file**, named after it. Order the file as: package doc, imports,
  constants, variables, types, constructors, methods, functions.
- **Comment budget (Principle VIII).** At most 5% of a file's non-blank lines hold a comment,
  rounded up: one in a 20-line file, five in a 100-line file. Tests count; generated code does
  not, and neither do tool directives (`//go:`, `//nolint`, goose's `-- +goose` and sqlc's
  `-- name:` annotations).
- **A package** opens with a one- or two-line package comment (`revive: package-comments`).
- **An exported identifier** gets a doc comment only when its name and signature do not already
  state its contract — for example which errors it returns, or what it requires of the caller.
  The comment starts with the identifier's name, is one to three lines, and ends with a period
  (`godot`). `// NewService returns a new Service.` and `// Get implements Store.` are not
  written.
- **Inside a function**, a comment says only what the code cannot: why a choice was made, a
  constraint, a non-obvious behaviour, or the measurement behind an optimisation. When you want
  to explain *what* the code does, rename or extract instead.
- **Requirement IDs** go in test names and the pull request, not in comments.
- No commented-out code (`gocritic`) and no `TODO`/`FIXME`/`XXX`/`HACK`/`BUG` markers (`godox`).
  Track future work in the board. Git holds the history, so don't write changelogs in comments.
- Imports form three groups: standard library, third-party, then this module (`goimports` with
  `local-prefixes: mapp`).

### Types and functions

- Prefer concrete types to `any`. Use generics only to remove real duplication across two or more
  types (`ListResponse[T]` is a good fit). Don't use them for single-use abstraction.
- Do not modify parameters, including through a slice or map argument, unless the contract says
  so (`revive: modifies-parameter`). Build a new value instead:

```go
// normalized trims surrounding spaces and upper-cases the currency code.
func (in CreateInput) normalized() CreateInput {
	return CreateInput{
		TeamID:     in.TeamID,
		Name:       strings.TrimSpace(in.Name),
		FiscalYear: in.FiscalYear,
		Currency:   strings.ToUpper(strings.TrimSpace(in.Currency)),
	}
}
```

- A constructor returns a value that is ready to use, or an error. Never return a value that
  needs a second `Init` call.
- Value receivers for small immutable types; pointer receivers when the method mutates or the
  type holds a mutex or is large. Don't mix the two on one type.
- No naked returns in functions longer than five lines (`nakedret`). No dot imports, `unsafe`,
  `//go:linkname` or cgo without an ADR. The SQLite driver is pure Go for this reason.

### Complexity metrics (Principle V)

| Metric | Acceptable | Needs justification | Unacceptable | How it is checked |
|---|---|---|---|---|
| Parameters | 0–4 | 5–6 | 7+ | `revive argument-limit` at 4 |
| Cyclomatic complexity | 1–8 | 9–14 | 15+ | `gocyclo` at 9 (soft), `cyclop` at 14 (hard) |
| Nesting depth | 1–4 | 5–6 | 7+ | `revive max-control-nesting` at 4 |
| Fan-out | 0–7 | 8–10 | 11+ | Code review |
| Function length (statements, excluding tests) | 1–40 | 41–60 | 61+ | `funlen` at 40; a `//nolint:funlen` states its count, and review rejects 61+ |
| Terms per expression | 1–5 | 6–8 | 9+ | Code review |

**How to count the review-only metrics:**

- **Fan-out:** the number of *distinct* functions and methods a function calls, including
  standard-library calls and calls inside its closures. Built-ins (`len`, `append`, `make`,
  `new`, `cap`, `copy`, `delete`) don't count.
- **Terms:** the operands in one expression: identifiers, literals and calls. For example,
  `err != nil || limit < 1 || limit > max` has six terms.

**How to bring a value down:**

- **Too many parameters:** group related ones into a struct (`ListQuery`, `CreateInput`). If
  they don't belong together, the function is doing too much.
- **High complexity or nesting:** return early with guard clauses, use `switch` instead of `if`
  chains, and extract a well-named helper.
- **High fan-out:** split by responsibility. `Serve` delegates to `shutdown`; `routes` delegates
  to `newEngine` and `mountBudgets`.
- **Long expressions:** name the parts.

```go
		limitOK := page.Limit >= 1 && page.Limit <= web.MaxPageSize
		if !limitOK || page.Offset < 0 {
```

**Justifying an exception:** when a value in the "needs justification" band is the right call,
suppress the specific linter on that line and say why. `nolintlint` rejects a bare `//nolint`, or
one without a reason:

```go
		FiscalYear: int32(in.FiscalYear), //nolint:gosec // G115: Service.Create validates 2000..2100 first.
```

Review-only metrics are justified in the pull request description. Values in the "unacceptable"
band are never merged: a `//nolint:cyclop` is always rejected, and so is a `//nolint:funlen`
whose stated count is above 60.

### Performance

Clear code first. Optimise only for a measured need: a benchmark proves the gain, and a comment
records what changed and why (Principle V). Profile with `pprof` before guessing. Benchmarks use
`b.Loop()`:

```go
func BenchmarkParsePage(b *testing.B) {
	for b.Loop() {
		_, _ = web.ParsePage("50", "100")
	}
}
```

---

## 12. Testing

A test exists to catch a defect that could really happen (Principle VI). In the backend that
means:

- unit tests for every business rule and calculation, with edge and negative cases (money,
  roll-ups, spreads, budgets, status transitions, required fields);
- a test for every acceptance scenario of the spec;
- a contract test for every endpoint (§ HTTP tests);
- a scope-isolation test for every store method that reads scoped records;
- for every bug fix, a test that reproduces the bug and fails before the fix;
- a fuzz test for every parser of untrusted input.

The coverage floor is SEMS's: at least 80% statement coverage on new or changed code, **plus** a
test for each outcome of every decision that carries a rule, a validation, an access check or a
user-visible error. Plain error propagation (`if err != nil { return fmt.Errorf("...: %w", err) }`)
needs no test of its own.

### Tests not to write

- A test that `NewService` returns a non-nil service, that a getter returns its field, or that
  sqlc maps a column.
- A test whose only assertion is that a fake was called.
- A test per `if err != nil` that only passes an error up.
- A handler test that repeats a rule the service test already proves: the handler test checks the
  translation (status, envelope, contract), not the rule.
- Any test written only to raise coverage. Close a coverage gap by testing behaviour that matters,
  or by deleting code nobody needs.

### Table-driven unit tests

Name each case after the behaviour and the requirement it proves, compare with `go-cmp`, and run
cases in parallel:

```go
func TestServiceActivate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  budget.Status
		teamID  string
		wantErr error
	}{
		{name: "FR-012 a draft becomes active", status: budget.StatusDraft, teamID: teamA},
		{name: "FR-012 an active budget is refused", status: budget.StatusActive, teamID: teamA, wantErr: budget.ErrNotDraft},
		{name: "FR-013 another team's budget looks missing", status: budget.StatusDraft, teamID: teamB, wantErr: budget.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &fakeStore{}
			existing := store.add(budget.Budget{TeamID: tt.teamID, Status: tt.status})
			service := budget.NewService(store, scopeOf(teamA), fixedClock)

			err := service.Activate(t.Context(), planner, existing.ID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Activate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
```

- Use `t.Context()` (cancelled when the test ends) instead of `context.Background()`
  (`usetesting`).
- Comparison helpers call `t.Helper()` (`thelper`).
- On failure, show what you called, what you got and what you wanted:
  `Activate() error = %v, want %v`; use `cmp.Diff` output in the form `(-want +got)`.
- Use `t.Fatalf` when the test cannot continue, and `t.Errorf` to report more.

### Fakes, not mocking frameworks

A hand-written fake of the consumer's interface is short, readable and behaves like the real
thing. This one limits every method to the scope's teams, as the SQLite store does:

```go
type fakeStore struct {
	budgets []budget.Budget
	err     error // returned by every method when set
}

func (f *fakeStore) Get(_ context.Context, scope authz.Scope, id string) (budget.Budget, error) {
	if f.err != nil {
		return budget.Budget{}, f.err
	}
	for _, b := range f.budgets {
		if b.ID == id && slices.Contains(scope.TeamIDs, b.TeamID) {
			return b, nil
		}
	}
	return budget.Budget{}, budget.ErrNotFound
}
```

### HTTP tests

Test handlers through a real router with `httptest`, covering status codes and the error
envelope, and validate every response against `api/openapi.yaml` through
`internal/platform/apitest`. That validation is the contract test every endpoint needs:

```go
		{name: "oversized body is refused", body: `{"name":"` + strings.Repeat("x", 2048) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge, wantBody: `"code":"body_too_large"`},
```

`gin.SetMode(gin.TestMode)` changes global state, so call it once in `TestMain`, before any
parallel test starts.

### Store tests against a real SQLite database

Mocks cannot prove that SQL is right, so stores are tested against the engine the application
runs on: SQLite now, PostgreSQL once it replaces SQLite (Principle VI). `dbtest.New(t)` creates a
database file in `t.TempDir()`, applies every migration, and closes it when the test ends. No
Docker is involved.

```go
func TestSQLiteStoreScopeIsolation(t *testing.T) {
	store := budget.NewSQLiteStore(dbtest.New(t))
	ctx := t.Context()
	owned, err := store.Create(ctx, budget.CreateInput{TeamID: teamA, Name: "FY27", FiscalYear: 2027, Currency: "EUR"}, fixedTime)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	t.Run("FR-013 get", func(t *testing.T) {
		if _, err := store.Get(ctx, authz.Scope{TeamIDs: []string{teamB}}, owned.ID); !errors.Is(err, budget.ErrNotFound) {
			t.Errorf("Get() outside the scope error = %v, want ErrNotFound", err)
		}
	})
}
```

Every store method that reads scoped records needs an isolation test like this one (B5).

### Fuzz tests for parsers

Any code that parses untrusted input (query parameters, the GSAP file imports) gets a fuzz test
that asserts an invariant:

```go
// FuzzParsePage checks what every caller relies on: whatever the input,
// ParsePage either fails or returns a page inside the limits.
func FuzzParsePage(f *testing.F) {
	f.Add("", "")
	f.Add("10", "20")
	f.Add("0", "-1")
	f.Add("201", "9999999999")
	f.Add("abc", " 1")

	f.Fuzz(func(t *testing.T, rawLimit, rawOffset string) {
		page, err := web.ParsePage(rawLimit, rawOffset)
		if err != nil {
			return
		}
		limitOK := page.Limit >= 1 && page.Limit <= web.MaxPageSize
		if !limitOK || page.Offset < 0 {
			t.Fatalf("ParsePage(%q, %q) = %+v, outside the limits", rawLimit, rawOffset, page)
		}
	})
}
```

The seeds run as ordinary tests. Explore locally with
`go test -run=^$ -fuzz=FuzzParsePage -fuzztime=30s ./internal/platform/web/`.

### Time without sleeping

`time.Sleep` in tests is forbidden because it is slow and flaky. Inject a clock
(`internal/platform/clock`), or run the test in a `testing/synctest` bubble, where a fake clock
advances instantly once every goroutine is blocked:

```go
func TestUntilStopsAtDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()

		calls := 0
		err := poll.Until(ctx, 7*time.Second, func(context.Context) (bool, error) {
			calls++
			return false, nil
		})

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Until() error = %v, want %v", err, context.DeadlineExceeded)
		}
		// Checks run at 0s, 7s, ... 56s; the deadline at 60s stops the next.
		const wantCalls = 9
		if calls != wantCalls {
			t.Errorf("check called %d times, want %d", calls, wantCalls)
		}
	})
}
```

### Coverage

```sh
go test -race -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1      # total
go tool cover -html=cover.out                 # find untested branches
```

Generated code (`sqlcdb`) and `cmd/` wiring count toward the total. Judge a change by the coverage
of the lines it touched; SonarQube's "coverage on new code" reports this once CI is set up.

---

## 13. What to avoid

These are the mistakes that most often damage Go services in production.

### 1. Returning a typed nil as an `error`

An interface holding a nil pointer is **not** nil, so callers see a failure that never happened.

```go
// ✗ Avoid
func validateBad(name string) error {
	var verr *ValidationError
	if name == "" {
		verr = &ValidationError{Field: "name"}
	}
	return verr // non-nil interface holding a nil pointer
}
```

```go
func validateGood(name string) error {
	if name == "" {
		return &ValidationError{Field: "name"}
	}
	return nil
}
```

### 2. Appending to a slice you were given

`append` may write into the caller's backing array and silently change their data.

```go
// ✗ Avoid
func withDefaultBad(tags []string) []string { return append(tags, "default") }
```

```go
func withDefaultGood(tags []string) []string { return append(slices.Clip(tags), "default") }
```

### 3. Goroutines that can never finish

If the caller gives up, a send on an unbuffered channel blocks forever. Each leaked goroutine
holds memory until the process dies.

```go
// ✗ Avoid
	result := make(chan string)
	go func() { result <- fetch() }() // blocks forever if ctx wins the select
```

```go
	result := make(chan string, 1) // the send never blocks, so the goroutine always exits
	go func() { result <- fetch() }()
```

### 4. Outbound HTTP without a timeout or context

`http.Get` uses `http.DefaultClient`, which has **no timeout**. One slow dependency can exhaust
every goroutine and connection.

```go
// ✗ Avoid
	resp, err := http.Get(url) // no timeout, no context, body never closed
```

```go
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := client.Do(req) // client is created once with Timeout set
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
```

Create one `*http.Client` with `Timeout` set, inject it, and reuse it so connections are pooled.
Always close the body, and cap how much you read.

### 5. `defer` inside a loop

Deferred calls run when the *function* returns, not when the iteration ends, so every file or
connection stays open until the loop finishes.

```go
// ✗ Avoid
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close() // every file stays open until processAllBad returns
```

```go
	for _, path := range paths {
		if err := processFile(path); err != nil {
			return err
		}
	}
```

### 6. Logging an error and returning it

The same failure gets logged at every layer, five lines per incident with no extra information.

```go
// ✗ Avoid
	if err != nil {
		slog.Error("query failed", "error", err)
		return err
	}
```

```go
	if err != nil {
		return Budget{}, fmt.Errorf("get budget %s: %w", id, err)
	}
```

### More to avoid

| Area | Avoid | Do instead |
|---|---|---|
| Errors | `_ = f()` or ignoring an `err` | Handle it, or explain in a comment why it can't fail (`errcheck`) |
| Errors | `err.Error() == "not found"`, `strings.Contains(err.Error(), …)` | `errors.Is` / `errors.AsType` |
| Errors | `panic` for bad input; `log.Fatal` in a library | Return an error; exit only in `main` |
| Errors | Returning raw database errors to clients | `WriteError` with a classified `errs.Error` |
| Context | `context.Background()` inside a request | Pass the request's `ctx` down |
| Context | Storing `ctx` in a struct field | Pass `ctx` as the first parameter |
| Context | Using context values for optional parameters | Explicit parameters; context values only for request-scoped data (the actor, the transaction) |
| Concurrency | Writing a map from several goroutines | A mutex, or confine the map to one goroutine (a concurrent map write is fatal) |
| Concurrency | Copying a struct that contains a `sync.Mutex` | Pointer receivers; `go vet` (copylocks) catches this |
| Concurrency | A goroutine per item for unbounded input | `errgroup` with `SetLimit`, or a fixed worker pool |
| Concurrency | `time.Ticker` or `time.Timer` without `Stop` | `defer ticker.Stop()` |
| State | Package-level mutable variables, singletons, `init()` that does I/O | Constructor injection (`gochecknoglobals`, `gochecknoinits`) |
| Database | Building SQL with `+` or `fmt.Sprintf` | sqlc queries with parameters; allowlists for sort keys |
| Database | A query on plan-owned data without the scope's teams in its `WHERE` | Scoped queries plus an isolation test |
| Database | A connection without the SQLite settings, or a cgo driver | `database.Open` with the pure-Go driver (§6) |
| Database | Read, check in Go, then update | One conditional `UPDATE`, or a constraint |
| Database | Long transactions, or network calls inside a transaction | Short transactions; do external I/O before or after |
| Database | Unbounded `SELECT` for list endpoints | Pagination with a maximum page size |
| Database | N+1: one query per row of a list | A join or `WHERE id IN (sqlc.slice(ids))` |
| Database | Editing a merged migration; migrating at API startup | A new migration; `cmd/seed` (or a deploy step) applies it |
| Data | `float64` for money; `REAL` or `NUMERIC` columns | `int64` minor units, or a canonical decimal string, in a `STRICT` table |
| Data | `t1 == t2` for `time.Time`; local time zones; `time.Now()` in a service | `t1.Equal(t2)`; UTC; the injected clock |
| HTTP | `router.Run()` or `http.ListenAndServe` without timeouts | `http.Server` with all four timeouts, on `127.0.0.1` |
| HTTP | `io.ReadAll(r.Body)` with no limit | `http.MaxBytesReader` (the `LimitBody` middleware) |
| HTTP | Trusting a team, user ID or role from the request | The signed-in actor and the scope resolved from it |
| Design | `util`, `common`, `helpers`, `models` packages | Packages named for what they provide, organised by business area |
| Design | An interface for every struct, declared next to its implementation | Small interfaces declared by the consumer, only when needed |
| Design | `any` or reflection where a type is known; generics for one use | Concrete types |
| Design | Premature optimisation (`sync.Pool`, `unsafe`, hand-rolled caches) | Measure first (`pprof`, benchmarks), then justify in a comment |
| Comments | Doc comments that repeat the name; more comments than the budget | Names that explain; comments only for why (§11) |
| Logging | `fmt.Println`, `log.Printf`, logging secrets or personal data | `slog` with attributes; redact at the source |
| Tests | `time.Sleep`, shared state between tests, order-dependent tests | Injected clocks or `synctest`, fresh fixtures, `t.Parallel()` |
| Tests | Testing only the happy path, or testing lines just for coverage | A case for each outcome of every rule, validation and access check (§12) |
| Tooling | Ignoring `go vet`, the race detector or `govulncheck` findings | Fix them, or justify with a specific `//nolint` |

---

## 14. Pull request checklist

Copy this into the pull request description and tick each item.

```markdown
- [ ] Requirement IDs addressed: …
- [ ] `golangci-lint fmt` and `golangci-lint run` are clean
- [ ] `sqlc generate` produces no diff; `sqlc vet` passes
- [ ] New migrations apply to an empty SQLite database; no merged migration edited
- [ ] `go test -race ./...` passes, store and contract tests included
- [ ] Every changed rule, validation and access check has a test for each outcome; new-code coverage ≥ 80%; no test written only for coverage
- [ ] `govulncheck ./...` is clean
- [ ] `api/openapi.yaml` updated for any API change, and `npm run api:types` run
- [ ] Every new query on plan-owned data filters by the scope's teams and has an isolation test
- [ ] Every changed file is within the comment budget (≤ 5% of non-blank lines)
- [ ] Fan-out ≤ 7 and expression terms ≤ 5, or justified below
- [ ] Every //nolint names its linter and gives a reason
- [ ] Design docs / ADRs updated for structural or cross-cutting changes
- [ ] No secrets, tokens, or personal data in code, logs, or test fixtures

Metric exceptions and justifications: …
```

---

## 15. References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Google Go Style Guide](https://google.github.io/styleguide/go/)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Organizing a Go module](https://go.dev/doc/modules/layout)
- [Go Concurrency Patterns: Context](https://go.dev/blog/context) and
  [Pipelines and cancellation](https://go.dev/blog/pipelines)
- [sqlc documentation](https://docs.sqlc.dev/) and [goose](https://pressly.github.io/goose/)
- SQLite: [STRICT tables](https://www.sqlite.org/stricttables.html),
  [write-ahead logging](https://www.sqlite.org/wal.html),
  [foreign keys](https://www.sqlite.org/foreignkeys.html),
  [transactions](https://www.sqlite.org/lang_transaction.html)
- [OWASP ASVS](https://owasp.org/www-project-application-security-verification-standard/) and
  [OWASP API Security Top 10](https://owasp.org/API-Security/)
- [golangci-lint linters](https://golangci-lint.run/docs/linters/)
