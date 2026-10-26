# backend — MAPP API

The MAPP API: one Go program (module `mapp`, Go 1.27, standard library only at run time) with
one package per business area. Every area follows the same pattern, **handler → service → repository**
(ADR-0003), and keeps its data in PostgreSQL 18 (ADR-0013): pgx, queries compiled by sqlc, goose migrations.

A request goes through `web.Middleware` → the route's handler (behind `auth.RequireSignIn` when it needs a
signed-in person) → the service, which holds every rule → a repository interface → the area's
`store_postgres.go`. The JSON file store remains only as the stand-in of the service, handler and contract
tests, until feature 025 replaces it with fakes.
The API contract is [api/openapi.yaml](api/openapi.yaml); the web app's types are generated from it.

## Commands

Run them in this folder (`backend/`).

| Command | What it does |
|---|---|
| `go run ./cmd/seed` | Creates the database of `DATABASE_URL` when it is missing, applies the migrations, and loads the demo data (people, teams, roles and access rules, Settings with dated exchange rates, plans with their campaigns, initiatives, activities and line items, approval workflows with rules and a delegation, a global budget with its allocations, simulated GSAP transactions, activity-log history and a recycle-bin entry, two saved All work views) into an empty database — stop the API first |
| `go run ./cmd/seed -reset` | Applies the migrations, empties every table (`TRUNCATE`) and loads the demo data again |
| `go run ./cmd/seed -if-empty` | Applies the migrations and loads the demo data only into an empty database; a database with data is left as it is, and the seed still succeeds |
| `go run ./cmd/api` | Starts the API on <http://127.0.0.1:8080>, logging JSON lines on standard output; Ctrl+C or SIGTERM stops it once the requests in progress finish |
| `gofmt -l .` | Lists files that are not formatted (prints nothing when all is well; `gofmt -w .` fixes them) |
| `go vet ./...` | Finds suspicious code |
| `go test -short ./...` | Runs every unit and contract test; no database needed (one package: `go test -short ./internal/samples/`) |
| `go test ./...` | Also runs the scope-isolation tests on the PostgreSQL server of `TEST_DATABASE_URL` |
| `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate` | Regenerates `internal/platform/database/sqlcdb` after a change to `queries/` or `migrations/` (`… vet` checks the queries) |

The seed refuses to run while an API answers on its `APP_PORT` (if the API runs on another port, give the
seed the same `APP_PORT`). The API never migrates: it stops with a message when the database is behind
the latest migration, and `go run ./cmd/seed` applies it. `go test -race` needs a C compiler on Windows,
which is not installed.

### PostgreSQL

PostgreSQL 18 on `127.0.0.1:5432`: `docker compose up -d postgres` from the repository root, or a native
install with a role that may create databases (`createuser --createdb mapp`). The scope-isolation tests
need the same: each copies its own database from a migrated template (`mapp_template_<hash>`) and drops it
when it ends; migration `00013` creates the `btree_gist` extension, which a database's owner may create.

```sh
export DATABASE_URL='postgres://mapp@127.0.0.1:5432/mapp?sslmode=disable'          # the API and the seed
export TEST_DATABASE_URL='postgres://mapp@127.0.0.1:5432/postgres?sslmode=disable' # the scope-isolation tests
```

A test run that was killed can leave `mapp_test_*` databases behind, and templates of older migration sets
stay too. To drop them all, with no test running:

```sh
psql "$TEST_DATABASE_URL" -Atc "SELECT format('DROP DATABASE %I WITH (FORCE);', datname) FROM pg_database WHERE datname LIKE 'mapp\_template\_%' OR datname LIKE 'mapp\_test\_%'" | psql "$TEST_DATABASE_URL"
```

## Settings

Environment variables; an empty one counts as not set. Invalid values stop the API with a message for each.

| Variable | Meaning | Default |
|---|---|---|
| `APP_ENV` | `local`, `development` or `production`. Demo sign-in and the developer sample records exist only in `local` and `development` | `local` |
| `APP_PORT` | Port number, 1–65535. There is no host setting: the API listens on `127.0.0.1` only | `8080` |
| `DATABASE_URL` | The PostgreSQL database of the API and the seed; never written to logs or errors | `postgres://mapp@127.0.0.1:5432/mapp?sslmode=disable` |
| `DEMO_PASSWORD` | The password every demo person shares; may not be blank | `mapp` |
| `LOG_LEVEL` | `debug`, `info`, `warn` or `error` | `info` |
| `SHUTDOWN_TIMEOUT` | How long the requests in progress may take to finish when the API stops, 1s–60s | `10s` |
| `OTEL_TRACES_EXPORTER`, `OTEL_METRICS_EXPORTER` | `none`, or `console` to write traces or metrics to standard error; nothing is ever sent off this computer | `none` |

Example, PowerShell: `$env:APP_PORT = '18081'; go run ./cmd/api` · Git Bash: `APP_PORT=18081 go run ./cmd/api`

## Packages

```
api/                  openapi.yaml — the contract, source of truth for every endpoint (embed.go: for the tests)
cmd/api/              the API program
cmd/seed/             the demo-data loader (logic in run(), tested in main_test.go)
seed/                 the demo dataset; every feature adds its own records here
internal/
  app/                config.go (the settings above) and app.go (wiring) — the only package that knows all others
  stores/             the store of every area and their Transactor: Postgres for the API and the seed, JSON for tests
  platform/           small shared helpers
    actor/            who is acting: Actor{ID, Name}; actor.With and actor.From on the context
    clock/            "now" as a function, so tests can fix the time
    errs/             the one error format, its codes and constructors; errs.Problems collects field errors
    web/              the Gin engine and its middleware; BindJSON, Respond and RespondError for handlers
    logging/          JSON log lines that carry the request ID, trace ID and actor ID
    telemetry/        OpenTelemetry tracing and metrics, exported to the console at most
    database/         PostgreSQL: Open, Migrate, Write (one at a time), Read (snapshot); migrations/, queries/, sqlcdb/, dbtest/
    memo/             values worked out once per read-only transaction, such as Settings' forms
    store/            the JSON file store, the tests' stand-in until feature 025: transactions and table helpers
    ids/              readable IDs such as SF-1001, from counters that only ever go up
    record/           record.Meta — version, created and updated by and at — with Touch and CheckVersion
    apitest/          test helper: the real API on a temporary data folder, every call checked against the contract
    money/            exact money: Amount, Money, Input, Rate, Rates, Percent; half-up rounding (owner: 002)
    date/             calendar dates: date.Date ("2026-03-01" in JSON), AddYears, Today, Format (owner: 005)
    sheet/            spreadsheet files with the standard library: .xlsx workbooks and CSV (owner: 010)
  auth/               demo sign-in, sessions and the RequireSignIn middleware
  people/             people, "me" (GET /api/me), display preferences; teams and memberships (003)
  history/            activity log and recycle bin core: Log, MoveToBin, Undo, Restore and the Kind registry
  samples/            developer sample folders and notes (local and development only)
  settings/           models, activity types, Shell attributes and options, currencies, finance budgets (002)
  plans/              plans and the registry of the levels below them (004)
  campaigns/          campaigns and initiatives (005)
  activities/         activities (006)
  lineitems/          line items (007)
  approvals/          approval workflows, rounds, history entries and notifications (008)
  home/               the Home dashboard, read-only over 003–008 through Sources (009)
  allwork/            All work (010): the read-only Grid, Calendar and Budget & cost of a year, export, saved
                      views; it reads 004–007 through allwork.Source (internal/app/allwork_source.go)
  budgets/            global budgets and their allocations to plans (014): an allocation sets the plan budget
                      through plans.SetBudgetFromAllocation (internal/app/budgets_adapter.go)
  access/             roles, role assignments with scopes (All markets, My teams, Chosen teams), access rules
                      and the route table that gives every operation its permission, failing closed (014)
  financials/         GSAP transactions (POs, invoices, payments, accruals): the simulated GSAP extract
                      (CSV import with a check step), matching to line items, figures and manual links (015)
  rollup/             how plans, campaigns and initiatives add up what is below them, and budget phasing (005, 016)
```

`platform/authz` holds the fixed permissions, the five built-in roles and `Grants` (what a person may do,
where); `internal/access` fills them from the stored roles. Settings also holds dated exchange rates, the
simulated GSAP rate-file import (`ratefeed.go`) and field help (`fieldhelp.go`) of 016; `approvals` holds
017's rules, delegations, reminders and escalation, and monitoring. GSAP data and rates come from files and
demo data behind adapters (ADR-0009), so a real feed replaces them without changing the services.

**Demo roles** (`seed/access.go`): Alex Morgan Administrator (all markets); Carlos Mendes Global planner
(all markets); Daniel Okoro Planner and Approver, Priya Raman and Mei Lin Planner, Sara Novak Viewer (each for
their own teams). A refused operation answers with the reason ("You don't have access to this …"); the
access matrix test (`internal/app/access_matrix_test.go`) checks every served operation against each persona.

The Plan module's packages (002–008) started from a shared foundation: each has its stored types and API
types (`model.go`), the hook types other features implement (`hooks.go`, `usage.go`, `levels.go`), its
`Repository` with the JSON implementation, and a round-trip test. Their features add the services,
history kinds and handlers (docs/sdd/plan-module-build.md).

## How to add a business package

`internal/samples` is the smallest complete example: copy its shape. For a new package `plans`:

1. **`model.go`** — the stored type, which embeds `record.Meta`, and separately the API types the handler
   returns and receives (for example `PlanView`, `PlanInput`, and `PlanUpdate` with `Version *int`).
   Lists are never `nil`.
2. **`repository.go`** — the `Repository` interface. **`store_json.go`** — `NewJSONRepository()`, built on
   `store.Get`, `store.List`, `store.Put` and `store.Delete`.
3. **`service.go`** — `NewService(tx store.Transactor, repo Repository, …)`. In every method:
   - take the actor (`ctx, a actor.Actor, …`), also when it only reads;
   - check the input and report every problem at once with `errs.Problems` (`Add`, then `Err()`);
   - do the work inside one `s.tx.Write(ctx, func(ctx context.Context) error { … })` (`s.tx.Read` when
     it only reads); on an update, check the version with `CheckVersion` and call `Touch`;
   - write one `history.Log` entry per change in that same `Write`;
   - delete only with `history.MoveToBin`, and describe each record type as a `history.Kind`
     (samples does this in `kinds.go`), so the recycle bin can bring it back.
4. **`handler.go`** — `NewHandler(s)` and `Routes(r *web.Router, signedIn web.Wrap)`; wrap every route that
   needs a signed-in person in `signedIn`. Each handler: `web.ReadJSON` → `actor.From` → service →
   `web.WriteJSON` or `web.WriteError`. A delete answers
   `history.DeleteResult{RecycleBinEntry: entry.View()}`, so the web app can offer Undo.
5. **Tests** — `service_test.go` inside the package (`package plans`) for every rule;
   `handler_test.go` as `package plans_test` with `apitest.New(t)` and `server.SignIn(t, email)`, for every
   endpoint. Test names mention the requirement they prove (`TestCreateFolder_Validation_FR020`).
6. **Contract** — add the paths and schemas to `api/openapi.yaml`, then run `npm run api:types` in
   `frontend/`.
7. **Wiring** — in `internal/app/app.go`: build the repository and the service, register the kinds with
   `historyService.Register` (parents first), and call `NewHandler(…).Routes(router, signedIn)`.
   `TestRoutesMatchContract` fails until the routes and the contract agree.

Demo records for the new package go in `seed/`: `seed.go` holds `Load` and the shared helpers, and each
feature has its own file (`settings.go`, `teams.go`, `plans.go`, …) with a test file beside it. The reasons behind all of this are in
`specs/001-app-shell-and-sign-in/research.md`; the folder rules in `docs/architecture/repository-layout.md`.
