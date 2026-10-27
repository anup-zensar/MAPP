# Research: Test Fakes

## R1. The fake shape and where it lives (decision a)

- **Decision**: one shared package, `internal/platform/fake`, with `DB` (the transaction), `Table[T]` (the
  records of one type by id) and `Counters` (the readable-ID counters). Each area's fake is a thin type
  over its tables: `internal/<area>/<area>test.Store`, implementing the wiring's `stores.<X>Store`, for
  every other package (`apitest`, `seed`, `internal/app`, other areas' fixtures, the area's external
  tests). The area's own internal tests (`package <area>`) use `fakeStore` in `store_fake_test.go`,
  implementing the area's `Store`.
- **Behaviour** (FR-002), as the PostgreSQL store, not as the JSON stand-in where the two differ:
  - records by id; a missing one is `(zero, false, nil)`; deleting a missing one changes nothing;
  - lists ordered as the SQL's `ORDER BY length(id), id`, never nil; other orders (by date, by number,
    newest first) sort as the store's query does;
  - the same scope filters: the Go predicates the services and the JSON stand-in already share with the
    SQL (`authz.Scope.Sees`, `SeesActivity`, …);
  - records are kept JSON-encoded inside the table, so a caller never shares memory with what is stored,
    as with rows read from a database (a service that changes a slice of a record it read cannot change
    the stored record);
  - normalisation on save where the PostgreSQL store normalises (`Plan.Normalised`);
  - no conflict of its own: version conflicts are the services' rule on the stored `record.Meta`, and no
    PostgreSQL store maps a constraint violation to an answer today.
- **Why two fakes per area**: Go refuses an import cycle in tests. `plantest` imports `plan`, so the
  internal tests of `plan` (20 of its 27 test files) cannot import `plantest`. The alternatives:
  - a fake in the production package (`plan/store_memory.go`) — one copy, but a test double shipped in the
    API, which the brief rules out;
  - turning about 130 internal test files into external ones — each needs its identifiers qualified and
    the unexported ones it uses exported for tests, and every touched file must meet the comment budget;
    far beyond this feature, for no behaviour gained.
  The cost is kept small: both fakes are one-line methods over `fake.Table`; the logic they could repeat
  (ordering, copying, rollback) is in `fake`, and the scope predicates are the model's own functions.
- **Rationale**: B9 asks for small hand-written fakes of consumer-defined interfaces; the internal fake is
  exactly that for the service's `Store`; `<area>test` is the same for the wiring's interface.

## R2. The whole API without the JSON store (decision b)

- **Decision**: `internal/stores/storestest.New(db)` builds a `stores.Set` from every `<area>test` fake on
  one `fake.DB`, with `fake.Counters` and `Ready`/`IsEmpty` answering from the fakes. `storestest.Demo`
  loads the demo data once per test binary through `seed.Load` (as `apitest.CopyDemoData` loaded it once
  into a folder) and returns a `DB.Copy` per call, a few map copies instead of a folder copy. `apitest.New`
  passes the set to `app.New`, so the access matrix in `internal/app` and the contract validation of
  every request and answer keep working unchanged.
- **Rationale**: the brief's choice; a fake behind each service of each handler test would duplicate the
  wiring of `internal/app` in 80 test files and lose the access matrix, which needs the whole API.

## R3. The transaction stand-in (decision c)

- **Decision**: `fake.DB` implements `Read` and `Write` as `database.DB` does: Writes one at a time (a
  mutex), Reads share it; a Write or Read inside a Write joins it (the context carries it); a change or a
  Write inside a Read returns `fake.ErrReadOnly`; a Write whose function fails or panics puts back every
  change it made, in reverse order, counters included. A change outside any transaction stands at once,
  as on a pool connection.
- **The memo**: `fake.DB` does not start a `memo` per transaction, so `memo.Get` works its value out on
  every call, as it did on the JSON stand-in (whose transactions carried no `memo` either). The memo is
  proved where it runs, on PostgreSQL: feature 028's statement counts and the end-to-end suite.
- **Transactor**: the interface moves out of the deleted package to its consumers (GO_STANDARDS §6):
  each service declares `Transactor` next to its `Store`; `history.Tools.Tx` is `history.Transactor`;
  `stores.Set.Tx` is `stores.Transactor`. `database.DB` and `fake.DB` satisfy all of them.
  `settings.formOf` uses `memo.Get` in place of `store.Memo`, which already fell back to it on PostgreSQL.

## R4. Keeping the fakes honest (decision d)

- **Decision**: the scope-isolation tests and the end-to-end suite stay on PostgreSQL and are the only
  database tests. No shared behaviour check runs one suite against a fake and the PostgreSQL store: that
  would be a database test of queries, which Principle VI forbids outside scope isolation.
- **Risks, recorded**:
  - A fake and its PostgreSQL store can drift (an order, a normalisation, a default). A service test can
    then pass on the fake and the API fail on PostgreSQL. The end-to-end suite runs every P1 story on
    PostgreSQL and catches what users see; a drift outside those paths is caught only in review.
  - The scope filter exists twice: in SQL (proved by the scope-isolation tests) and as the Go predicate
    the fakes use (proved by the service tests). A change to one must change the other.
  - The area's two fakes (R1) can drift from each other. Both are compiled against the interfaces they
    implement, so a changed signature fails the build; a changed behaviour must be made in both.
  - The fakes keep no memo (R3): a memo that serves a stale value is not caught by service tests.

## R5. Measurements (SC-002, SC-003)

Measured on the lead's machine (Apple silicon), `go test -count=1 -json ./...` from `backend/`, after a
warm build; times are the wall clock of the whole command and the sum of the packages' own times.

| Suite | Tests before | Tests after | Wall before | Wall after | Package time before | Package time after |
|---|---|---|---|---|---|---|
| `-race` with `TEST_DATABASE_URL` | 1,528 top-level (2,039 with subtests) | | 41.2 s | | 302.9 s | |
| `-short` | 1,528 top-level (2,014 with subtests) | | 30.2 s | | 221.6 s | |

## R6. Tests removed with the JSON store (FR-007)

To be listed as they are removed, each with the rule of Principle VI that removes it: a test of code that
no longer exists (the JSON store's files, transactions and data folder).
