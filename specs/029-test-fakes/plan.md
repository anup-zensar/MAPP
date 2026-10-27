# Implementation Plan: Test Fakes

**Branch**: `feature/029-test-fakes` | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

## Summary

The JSON file store, kept since feature 023 only as the tests' stand-in, is replaced by hand-written
in-memory fakes and deleted. A small shared package, `internal/platform/fake`, gives the fakes their
transaction (`fake.DB`), their tables (`fake.Table[T]`) and the readable-ID counters (`fake.Counters`).
Each business area gets a fake of its store in `internal/<area>/<area>test`; `internal/stores/storestest`
gathers them into an in-memory `stores.Set`; `apitest.DemoDB` loads the demo data into it through `seed.Load`.
Behaviour, SQL and the API contract don't change.

## Technical Context

Go (pinned in `go.mod`), Gin, pgx/v5, sqlc, goose; tests with `testing`, `httptest`, `go-cmp`,
`libopenapi-validator`; PostgreSQL 18 for the scope-isolation and end-to-end tests only. No new dependency.

## Constitution Check

- Principle VI: the feature's goal. Only scope-isolation and end-to-end tests touch PostgreSQL; no new
  database test (no shared fake-versus-PostgreSQL suite: research R4). Every test keeps its name, its
  requirement ID and its assertions; the tests of the JSON store itself go with it (research R6).
- B2, B9: each fake implements a consumer-defined interface (the area's `Store`, or the wiring's
  `stores.<X>Store`); no mocking framework; no test only checks that a fake was called.
- Principle V: one shared, generic table type removes the duplication of the per-area fakes; files within
  300 non-blank lines.
- Principle VIII: 029 is not exempt; every file the feature touches is brought within the 5% comment
  budget as a whole.
- Governance → Alignment: production code changes only where the JSON store's types are named
  (`store.Transactor`, `store.Memo`); no package migrates its stack.

## The fake design (worked example: history)

```go
// internal/platform/fake — shared by every fake
db := fake.NewDB()                          // a Transactor: Read, Write (one at a time, undone on error)
entries := fake.TableOf(db, "activity_log", func(e history.Entry) string { return e.ID })
entries.Put(ctx, e)                         // inside db.Write: undone if the Write fails; in a Read: ErrReadOnly
entries.List()                              // ordered by length(id), id like the SQL; never nil
generator := ids.NewGenerator(fake.NewCounters(db))
copied := db.Copy()                         // a test's own copy of the loaded demo data
```

```go
// internal/history/historytest/store.go — the fake every other package uses
type Store struct {
	entries *fake.Table[history.Entry]
	bin     *fake.Table[history.BinEntry]
}

func NewStore(db *fake.DB) Store { ... }
func (s Store) AppendEntry(ctx context.Context, e history.Entry) error { return s.entries.Put(ctx, e) }
func (s Store) GetBinEntry(_ context.Context, id string) (history.BinEntry, bool, error) { return s.bin.Get(id) }
// ... one short method per Store method; the area's queries and scope filters as Go predicates
```

```go
// internal/history/store_fake_test.go — the same fake for history's own internal tests (research R1)
type fakeStore struct { ... }
func newFakeStore(db *fake.DB) fakeStore { ... }

// internal/history/fixture_test.go
f := &fixture{db: fake.NewDB(), clock: &testClock{now: start}}
f.service = NewService(f.db, newFakeStore(f.db), ids.NewGenerator(fake.NewCounters(f.db)), f.clock.Now)
```

```go
// internal/history/store.go — the transactor the service needs, next to its Store (GO_STANDARDS §6)
type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}
```

Per area:

1. `internal/<area>/<area>test/store.go`: `Store` and `NewStore(db *fake.DB)`, implementing the wiring's
   `stores.<X>Store` (the area's `Store` and the wiring-only methods). It must not import
   `internal/stores` (that package imports every area, so another area's internal tests could not import
   the fake); `storestest` checks the interface when it builds the set.
2. `internal/<area>/store_fake_test.go`: `fakeStore` and `newFakeStore(db *fake.DB)`, implementing the
   area's `Store`, for the area's own internal tests only.
3. `store.go` declares `Transactor`; `service.go` (and any file naming `store.Transactor`) uses it.
4. Fixtures and tests use `fake.DB`, the fakes and other areas' `<area>test` fakes.

## The whole API in memory

`internal/stores/storestest`: `New(db *fake.DB) stores.Set` builds every area's fake on one DB; `apitest.DemoDB`
loads the demo data once per test binary with `seed.Load` and returns a copy (`fake.DB.Copy`) per call.
`apitest.New` builds the API with `app.New(cfg, set, clock)` on it, so the access matrix and every contract
check run as before.

## Order of work

1. Lead: spec documents; `internal/platform/fake`; history (the worked example).
2. Helpers, at most three at a time, each in its own worktree and branch `029-<area>`: `<area>test`, the
   internal fake, the `Transactor`, the fixtures; only their packages' tests and lint. The lead merges
   each one (merge, not rebase) and runs that area's tests.
3. Lead: `storestest`; `apitest`, `seed`, `internal/app`, `ids` and activity's demo-data fixture on it.
4. Lead: delete the JSON store; gates; measurements; pull request.

| Round | Helpers |
|---|---|
| 1 | settings · people · access |
| 2 | plan · campaign · approval |
| 3 | lineitem · budget · financials |
| 4 | sample, auth · allwork, home · activity |

## Verification

Gates (FR-009): `gofmt -s`, `goimports -local mapp`, `go vet ./...`, golangci-lint on new lines,
`sqlc generate` (no diff) and `sqlc vet`, `go test -race ./...` with `TEST_DATABASE_URL`,
`go test -short ./...`, CI's timing tests without `-race`, govulncheck, `pnpm run api:check` and the
end-to-end suite on PostgreSQL. Test counts and times before and after are recorded in research R5.
