# Research: Test Fakes

## R1. The fake shape and where it lives (decision a)

- **Decision**: one shared package, `internal/platform/fake`, with `DB` (the transaction), `Table[T]` (the
  records of one type by id) and `Counters` (the readable-ID counters). Each area's fake is a thin type
  over its tables: `internal/<area>/<area>test.Store`, implementing the wiring's `stores.<X>Store`, for
  every other package (`apitest`, `seed`, `internal/app`, other areas' fixtures, the area's external
  tests). The area's own internal tests (`package <area>`) use `fakeStore` in `store_fake_test.go`,
  implementing the area's `Store`. Built: 14 `<area>test` packages and `storestest` (with `fake`, 1,863
  non-blank lines), 13 `store_fake_test.go` files (817 lines); the JSON store they replace was about 4,600.
- **Behaviour** (FR-002), as the PostgreSQL store, not as the JSON stand-in where the two differ:
  - records by id; a missing one is `(zero, false, nil)`; deleting a missing one changes nothing;
  - lists ordered as the SQL's `ORDER BY length(id), id`, never nil; other orders (by date, by number,
    newest first) sort as the store's query does;
  - the same scope filters: the Go predicates the services and the JSON stand-in already share with the
    SQL (`authz.Scope.Sees`, `SeesActivity`, …);
  - records are kept JSON-encoded inside the table, so a caller never shares memory with what is stored,
    as with rows read from a database (a service that changes a slice of a record it read cannot change
    the stored record);
  - normalisation on save where the PostgreSQL store normalises: plans (`Plan.Normalised`), campaigns
    and initiatives (owner, attributes, phasing), activities and line items (attributes `{}`), access
    assignments (teams `[]`), Settings (empty child lists), approval workflows and rounds (feature 017's
    defaults); where the JSON store differed, the fake follows PostgreSQL (a workflow without levels reads
    `[]`; `PurgeBinEntries` skips a missing id instead of stopping);
  - no conflict of its own: version conflicts are the services' rule on the stored `record.Meta`, and no
    PostgreSQL store maps a constraint violation to an answer today; no fake enforces a foreign key.
- **Why two fakes per area**: Go refuses an import cycle in tests. `plantest` imports `plan`, so the
  internal tests of `plan` (20 of its 27 test files) cannot import `plantest`. The alternatives:
  - a fake in the production package (`plan/store_memory.go`) — one copy, but a test double shipped in the
    API, which the brief rules out;
  - turning about 130 internal test files into external ones — each needs its identifiers qualified and
    the unexported ones it uses exported for tests, and every touched file must meet the comment budget;
    far beyond this feature, for no behaviour gained.
  The cost is kept small: both fakes are one-line methods over `fake.Table`; the logic they could repeat
  (ordering, copying, rollback) is in `fake`, and the scope predicates are the model's own functions.
  activity's tests are external and home has no store, so they have no internal fake.
- **Rationale**: B9 asks for small hand-written fakes of consumer-defined interfaces; the internal fake is
  exactly that for the service's `Store`; `<area>test` is the same for the wiring's interface. No fake
  imports `internal/stores` (it imports every area, so another area's internal tests could not import the
  fake); `storestest` checks every fake against `stores.<X>Store` when it builds the set.

## R2. The whole API without the JSON store (decision b)

- **Decision**: `internal/stores/storestest.New(db)` builds a `stores.Set` from every `<area>test` fake on
  one `fake.DB`, with `fake.Counters`, `Ready` nil and `IsEmpty` "no ID given out yet" (as
  `stores.Postgres`). `apitest.DemoDB(tb, noHistory)` loads the demo data once per test binary through
  `seed.Load` into a `fake.DB` and returns a `DB.Copy` per call (map copies; the stored encodings are never
  changed in place, so copies share them). `apitest.New` passes the set to `app.New`, so the access matrix
  in `internal/app` and the contract validation of every request and answer run unchanged. Options:
  `WithDB(db)` replaces `WithDataDir` (two servers share data), `WithStorageDown()` makes `/readyz` fail
  (it replaces removing the data folder), `WithoutSeed`, `WithoutDemoHistory`, `WithEnv` are unchanged.
- **Rationale**: the brief's choice; a fake behind each service of each handler test would duplicate the
  wiring of `internal/app` in 80 test files and lose the access matrix, which needs the whole API.

## R3. The transaction stand-in (decision c)

- **Decision**: `fake.DB` implements `Read` and `Write` as `database.DB` does: Writes one at a time (a
  mutex), Reads share it; a Write or Read inside a Write joins it (the context carries it); a change or a
  Write inside a Read returns `fake.ErrReadOnly`; a Write whose function fails or panics puts back every
  change it made, in reverse order, counters included. A change outside any transaction stands at once,
  as on a pool connection.
- **The memo**: a fake Read starts a `memo` as `database.DB` does; nothing changes inside a Read, so
  nothing kept there can go stale. A fake Write keeps none, so no fake has to forget what a change
  outdates (the JSON store kept `store.Memo` values in a Write and forgot them all on any change; the
  effect is the same). Measured: without the Read memo the Settings form was rebuilt for every record a
  page shows, and the race-detector suite took 61.6 s against 41.2 s before (one Home test 50.5 s); with
  it, 30.9 s (that test 16.8 s).
- **Transactor**: the interface moves out of the deleted package to its consumers (GO_STANDARDS §6):
  each service declares `Transactor` next to its `Store` (home only `Read`); `history.Tools.Tx` is
  `history.Transactor`; `stores.Set.Tx` is `stores.Transactor`, which `internal/app`'s adapters also use.
  `database.DB` and `fake.DB` satisfy all of them. `settings.formOf` uses `memo.Get` in place of
  `store.Memo`, which already fell back to it on PostgreSQL.

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
  - Rules written twice: `approvaltest` repeats feature 017's level defaults (the production normaliser
    is unexported; the internal fake calls it), and `fake.Counters` starts at 1001 with its own constant
    (`ids.firstNumber` is unexported). A change to either rule must change the fake.
  - A Write keeps no memo in the fakes (R3): a memo that serves a stale value inside a Write is caught
    only on PostgreSQL, by the end-to-end suite.

## R5. Measurements (SC-002, SC-003)

Measured on the lead's machine (Apple silicon), `go test -count=1 -json ./...` from `backend/`, after a
warm build; times are the wall clock of the whole command and the sum of the packages' own times.

| Suite | Tests before | Tests after | Wall before | Wall after | Package time before | Package time after |
|---|---|---|---|---|---|---|
| `-race` with `TEST_DATABASE_URL` | 1,528 top-level (2,039 with subtests) | 1,500 (2,008) | 41.2 s | 30.9 s | 302.9 s | 209.0 s |
| `-short` | 1,528 top-level (2,014 with subtests) | 1,500 (1,983) | 30.2 s | 7.4 s | 221.6 s | 24.6 s |

The 28 fewer top-level tests are the 32 tests of the deleted JSON store (R6) less the 4 tests of `fake`.
CI's timing tests, without the race detector (before on `develop`, after on this branch):

| Test | Before | After |
|---|---|---|
| `TestQueryLog_50000Entries_SC006` | 1.07 s | 1.13 s |
| `TestQuery_StressYear_SC004` | 0.42 s | 0.37 s |
| `TestImport_FiveThousandRows_SC004` | 0.18 s | 0.06 s |
| `TestTransactions_TwentyThousand_SC004` | 0.06 s | 0.05 s |
| `TestHome_LargeScope_SC004` | 0.02 s | 0.02 s |

## R6. Tests removed, moved or adapted (FR-007)

Removed with the JSON store; each tested code that no longer exists (Principle VI: a test proves a
behaviour someone relies on):

- the 27 tests of `internal/platform/store` (files, the data folder, its transactions and table helpers);
- `TestPlanStore_NormalisesOlderRows` and `TestJSONRepository_PhasingOfOlderRows` (the JSON store reading
  rows saved before 016) and `TestJSONRepository_RoundTrip` (the JSON store's tables and orders);
- `TestJSONCounters_StoreTheLastNumberPerPrefix` (the JSON counters' table);
- `TestNext_NeedsAWrite`: it asserted the JSON store's `ErrNoTransaction`, which PostgreSQL does not have;
  its Read half repeats `fake`'s read-only test (Principle VI: a second test of what another proves).

Moved, with name and assertions unchanged: `TestLegacyRecords_ReadWithDefaults_017P19` (now an internal
test on the fake, which saves through the production normaliser as PostgreSQL does);
`TestOptionalMoney_LeftOutNullOrAmount_D203`, `TestJSONRepository_Invariants_FR001` and
`TestDemoFeed_LoadsDataModelSix_FR031` (their files were named `store_json_test.go` but never tested it).

Adapted, assertion unchanged, where the setup named the data folder: `TestReadyz_DataFolderGone_Answers503_022_FR008`
(`WithStorageDown`), `TestDemoModeOff_ExistingSessionRefused_FR009` and
`TestDisplayCurrencies_SettingsAndPeople_FR003_D061` (`WithDB`), `TestQuery_NeverWrites_FR002` and
`TestSignIn_StoresOnlyTokenHash_FR006` (they compared files; now the stored records),
`TestNext_NeverReused_FR027` and history's restart test (a new service on the same data),
`TestLoad_RefusesAStoreWithData_FR028` (the stored folder now takes its ID from the counters first: like
PostgreSQL, the fakes count a store as empty until an ID was given out).

Left as found, outside this feature's files: the package comments of `seed/seed.go` and six `model.go`
files (settings, plan, campaign, activity, lineitem, approval) still name `repository.go` and
`store_json.go` (stale since 025); each file is far over the comment budget, so changing that line would
oblige rewriting its documentation (Principle VIII). Four functions unused on `develop` already
(`history.canSeeEntry`, financials' `ancestor`, plan's `assertJSON`, allwork's `strategic`) stay;
`history.invalidLimit` and `settings.withoutFields`, in files this feature touched, are removed.
