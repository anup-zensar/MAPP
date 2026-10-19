# backend — NorthStar API

The NorthStar API: one Go program (module `northstar`, Go 1.27, standard library only at run time) with
one package per business area. Every area follows the same pattern, **handler → service → repository**
(ADR-0003), and keeps its data in a local JSON file store for now (ADR-0004); PostgreSQL comes later.

A request goes through `web.Middleware` → the route's handler (behind `auth.RequireSignIn` when it needs a
signed-in person) → the service, which holds every rule → a repository interface → the JSON store.
The API contract is [api/openapi.yaml](api/openapi.yaml); the web app's types are generated from it.

## Commands

Run them in this folder (`backend/`).

| Command | What it does |
|---|---|
| `go run ./cmd/seed` | Loads the demo data (people, teams, Settings, plans with their campaigns, initiatives, activities and line items, approvals) into an empty data folder — stop the API first |
| `go run ./cmd/seed -reset` | Deletes the data files (`*.json`, `*.jsonl`, `*.tmp`) and loads the demo data again |
| `go run ./cmd/seed -data <folder>` | Uses another data folder |
| `go run ./cmd/api` | Starts the API on <http://127.0.0.1:8080>; Ctrl+C stops it |
| `gofmt -l .` | Lists files that are not formatted (prints nothing when all is well; `gofmt -w .` fixes them) |
| `go vet ./...` | Finds suspicious code |
| `go test ./...` | Runs every unit and contract test (one package: `go test ./internal/samples/`) |

The API reads the data folder only when it starts and would later overwrite new files from its memory, so
the seed refuses to run while an API answers on its `APP_PORT` (if the API runs on another port, give the
seed the same `APP_PORT`). `go test -race` needs a C compiler on Windows, which is
not installed.

## Settings

Environment variables; an empty one counts as not set. Invalid values stop the API with a message for each.

| Variable | Meaning | Default |
|---|---|---|
| `APP_ENV` | `local`, `development` or `production`. Demo sign-in and the developer sample records exist only in `local` and `development` | `local` |
| `APP_PORT` | Port number, 1–65535. There is no host setting: the API listens on `127.0.0.1` only | `8080` |
| `DATA_DIR` | Folder of the JSON store (`data` is git-ignored) | `data` |
| `DEMO_PASSWORD` | The password every demo person shares; may not be blank | `northstar` |

Example, PowerShell: `$env:APP_PORT = '18081'; go run ./cmd/api` · Git Bash: `APP_PORT=18081 go run ./cmd/api`

## Packages

```
api/                  openapi.yaml — the contract, source of truth for every endpoint (embed.go: for the tests)
cmd/api/              the API program
cmd/seed/             the demo-data loader (logic in run(), tested in main_test.go)
seed/                 the demo dataset; every feature adds its own records here
internal/
  app/                config.go (the settings above) and app.go (wiring) — the only package that knows all others
  platform/           small shared helpers
    actor/            who is acting: Actor{ID, Name}; actor.With and actor.From on the context
    clock/            "now" as a function, so tests can fix the time
    errs/             the one error format, its codes and constructors; errs.Problems collects field errors
    web/              ReadJSON, WriteJSON, WriteError, the middleware, and the Router that records every route
    store/            the JSON file store: transactions (Read, Write) and table helpers (Get, List, Put, …)
    ids/              readable IDs such as SF-1001, from counters that only ever go up
    record/           record.Meta — version, created and updated by and at — with Touch and CheckVersion
    apitest/          test helper: the real API on a temporary data folder, every call checked against the contract
    money/            exact money: Amount, Money, Input, Rate, Rates, Percent; half-up rounding (owner: 002)
    date/             calendar dates: date.Date ("2026-03-01" in JSON), AddYears, Today, Format (owner: 005)
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
```

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
