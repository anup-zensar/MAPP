# Research: Query Efficiency

**Feature**: [spec.md](spec.md) · **Plan**: [plan.md](plan.md)

## R1 — Benchmarks on PostgreSQL and constitution 4.x

- **Question**: Principle VI and B9 allow a real database only in end-to-end and scope-isolation tests and
  forbid "database tests of queries, constraints or migrations"; B9 also says "Every optimisation MUST be
  backed by a benchmark". An N+1 exists only against a database, so its benchmark must run on one.
- **Decision**: Go benchmarks (`Benchmark…` with `b.Loop`) on PostgreSQL, in `internal/platform/apibench`,
  skipped under `-short` like the scope-isolation tests (`dbtest`).
- **Rationale**: a benchmark is not a test. It asserts nothing, never fails on a statement count or a query
  plan, and runs only with `-bench`, so `go test ./...` never executes it. It reports what B9 asks for:
  statements per request (`queries/op`) and time. No test checks a statement count.
- **Rejected**: counting statements in unit tests with fakes (fakes have no round trips); a test asserting
  "at most N statements" (the kind of database test 4.0.0 forbids, and brittle).

## R2 — How statements are counted

- **Decision**: `internal/platform/querycount` keeps a counter in the request's context; the pool's pgx
  tracer (`database.counter`, a `QueryTracer` and `CopyFromTracer`) adds one for every statement and every
  COPY sent with that context, `begin` and `commit` included. The request log record carries the total as
  `queries` (FR-009); the benchmarks read the same counter.
- **Rationale**: no new dependency (ADR-0011 names `otelpgx` as the follow-up for traces; a count needs no
  spans); one atomic add per statement; a request served in several transactions is still one count.

## R3 — The volume

- **Decision**: benchmark helpers (`apibench/volume*.go`, test support only, like `dbtest`) build a database
  once per migration set and keep it on the test server as a template (`mapp_perf_volume1_<migrations
  hash>`); every benchmark gets its own copy, dropped at the end. The volume is made through the API as a
  planner would — copies of a grown copy of the demo plan — plus people and log entries written through the
  stores. No product flag or seed option was needed.

| What | Volume | Target it serves |
|---|---|---|
| Plans | 253: the 4 demo plans, 49 more of Lubricants Europe in 2026, 200 of Global Marketing Ops in 2027–2030 | 004 SC-009 (200 plans) |
| Priya's scope (Lubricants Europe, 2026) | 50 plans · 5,307 activities · 21,206 line items | 009 SC-004, 010 SC-004 |
| One plan | 500 activities, 2,000 line items | 004 SC-009 |
| Five years 2026–2030 | 25,212 activities · 100,810 line items | 016 SC-006 |
| People | 1,000 | 003 SC-002 |
| GSAP transactions | 5,011 (5,000 imported) | 015 SC-004 |
| Activity log | 50,000 entries | 011 SC-006 |
| Recycle bin | 101 entries | 011 |
| Waiting for Daniel | 5 activities, 7 open rounds with the demo's (the board's cost depended on all activities, not on the waiting ones) | 008 |
| Global budget | allocated to 50 plans | 014 |

- **Not measured**: bulk actions and the plan import of feature 012 (012 SC-007) — 012 is a draft and the
  API has no such endpoints yet, only their access rules. The imports that exist are measured: the GSAP
  import (015 SC-004) and the rate and option imports (016 SC-002).

## R4 — Before

The code before this feature, on the volume, one request per page (`-benchtime=1x`), Apple M5, PostgreSQL 18
in Docker on the same machine. A request still running after a minute is stopped; its row shows the
statements it had sent by then, so its real count is higher. On the demo data the same pages sent 20–760
statements; on the volume the Plans list alone sent more than a million (one uncapped run: 1,028,561
statements, 663 s). Almost all of them read one reference record (a person, a Settings item, a workflow) or
one parent (an activity's initiative, campaign and plan) per record shown. The tables are in R7.

## R5 — What changed

1. **Transaction memo, by area** (ADR-0014). `database.DB.Write` carries the memo as `Read` already did.
   Each PostgreSQL store keeps what it reads under its area and forgets the area before every change
   (`change(ctx)`); values worked out from several areas live in `memo.Derived`, which any change forgets.
   Nothing is kept beyond the transaction (FR-005, FR-006).
2. **Reference tables whole** (`memo.Table`): models, activity types, currencies, rates, rate imports,
   finance budgets and items, Shell attributes and options, people, teams, memberships, preferences, roles,
   assignments, access rules, approval workflows. Rows are kept and decoded per call, with slices cloned.
3. **Planning records by id and by parent** (`memo.Records`): plans, campaigns, initiatives, activities and
   line items. A list of children read for many parents answers every later read of one of them; the list
   queries take many ids (`id = ANY(…)`).
4. **One read per level for many records**: plan roll-ups build every plan's tree at once
   (`campaigns.planTrees`), and the plan levels ask once; All work reads every plan's tree before its rows;
   line items read their activities' plans once (`FactsOf`); the Approvals board reads only activities with
   a round, in scope (`activities.ListInScopeOf`, with a scope-isolation test), and their parents once;
   workflow usage reads only the activities' workflow fields; financials reads the planning nodes one level
   at a time, and a card or Home only its plans; the activity log, an activity's history and the recycle bin
   read where each entry's record is now kind by kind; Budget by month reads its plan alone; an activity
   reads the activities split off it only; a person's page counts what they own in SQL.
5. **Writes per level** (FR-004): `ids.Generator.NextIDs` takes a range of IDs with one counter raise;
   `AddCampaigns`, `AddInitiatives` and `AddMany` insert a level with COPY, `DeleteMany` deletes it with one
   statement. Copies of plans, campaigns, initiatives and activities, delete with everything below, Undo,
   Restore and Empty recycle bin walk the tree one level at a time; rate and option imports save their rows
   at once. IDs, names, notes, counts and log and bin entries are the same as before (the existing service
   tests, new order tests and the end-to-end suite prove it). A GSAP import takes the IDs of its new
   transactions at once and adds them with COPY (`AddTransactions`); a restore reads the open approval
   rounds once (`OpenRounds`) instead of the rounds of each activity it puts back.
6. **Derived values once**: an activity's completeness and an activity's line-item summary are worked out
   once per transaction (the plan levels and the campaign tree asked for the same ones).

## R6 — Decisions and alternatives

- **Forget by area, not on every statement.** Clearing the whole memo on every write (as the JSON store
  did) would undo the memo inside write loops; detecting writes in the tracer would hide the rule in SQL
  text. Each store knows its tables, so it forgets its own area; cascades only reach a record's own child
  tables, which the same store owns.
- **Keep rows, decode per call** (reference tables): a caller that changes a value it got cannot corrupt
  what the transaction keeps. Decoded workflows are kept, with their level list copied per call.
- **Batch methods on today's interfaces**, implemented one by one in the JSON stand-in, so the service tests
  run unchanged; feature 025 may reshape them.
- **No migration**: no index was needed for the measured counts; an index on `activities.split_from_id` is
  a follow-up (R8).
- **Scope**: N+1 removal and scoped reads by ID only; no computation tuning (product owner, 2026-10-27).

## R7 — Before and after

Statements per request and time, one request each, on the volume (R3); "demo" is the same page on the demo
data, to show the count no longer grows with the records; for record pages and changes it is the demo plan
on the same volume. Before: the code at `origin/develop` 6c10428; after: this branch. Times are single cold
requests on a laptop; statement counts are exact. "Stopped" is a request cut off after a minute (R4).

| Lists and overviews | Statements before | Time before | Statements after | Time after | Statements after, demo data |
|---|---|---|---|---|---|
| plans | > 128,551 (stopped) | > 60 s | 35 | 1.47 s | 35 |
| plans search | > 71,610 (stopped) | > 60 s | 35 | 1.37 s | 35 |
| plans draft | > 41,057 (stopped) | > 60 s | 35 | 1.44 s | 35 |
| plans my teams | > 50,412 (stopped) | > 60 s | 35 | 0.31 s | 35 |
| home summary | > 80,297 (stopped) | > 60 s | 49 | 6.14 s | 45 |
| home plans | > 10,340 (stopped) | > 60 s | 35 | 1.94 s | 35 |
| home attention | > 33,614 (stopped) | > 60 s | 46 | 7.72 s | 42 |
| home waiting | > 36,651 (stopped) | > 60 s | 37 | 35 ms | 37 |
| home my activities | > 52,815 (stopped) | > 60 s | 44 | 6.71 s | 44 |
| home recent changes | > 127,180 (stopped) | > 60 s | 37 | 2.08 s | 38 |
| home financials | > 47,956 (stopped) | > 60 s | 29 | 1.78 s | 29 |
| all work grid | > 48,016 (stopped) | > 60 s | 38 | 2.28 s | 38 |
| all work search | > 63,136 (stopped) | > 60 s | 38 | 2.17 s | 38 |
| all work calendar | > 73,132 (stopped) | > 60 s | 38 | 2.11 s | 38 |
| all work budget | > 40,545 (stopped) | > 60 s | 38 | 2.19 s | 38 |
| all work options | > 60,695 (stopped) | > 60 s | 41 | 2.11 s | 41 |
| all work five years | > 27,237 (stopped) | > 60 s | 65 | 11.25 s | 47 |
| approvals waiting | > 17,884 (stopped) | > 60 s | 41 | 36 ms | 41 |
| approvals open | > 20,196 (stopped) | > 60 s | 41 | 35 ms | 41 |
| approvals decided | > 18,962 (stopped) | > 60 s | 41 | 36 ms | 41 |
| approvals summary | > 33,766 (stopped) | > 60 s | 36 | 36 ms | 36 |
| notifications | 96 | 1.03 s | 26 | 22 ms | 26 |
| activity log | > 56,217 (stopped) | > 60 s | 21 | 0.11 s | 24 |
| activity log filtered | > 76,268 (stopped) | > 60 s | 21 | 0.12 s | 21 |
| activity log summary | > 83,989 (stopped) | > 60 s | 18 | 0.11 s | 18 |
| recycle bin | 422 | 1.88 s | 22 | 21 ms | 19 |
| financials | > 21,353 (stopped) | > 60 s | 25 | 6.32 s | 25 |
| financials overview | > 213,961 (stopped) | > 60 s | 26 | 1.85 s | 26 |
| global budgets | > 19,181 (stopped) | > 60 s | 34 | 1.80 s | 34 |
| global budget | > 5,569 (stopped) | > 60 s | 37 | 1.80 s | 37 |
| people | 20 | 4.66 s | 18 | 20 ms | 18 |
| teams | 22 | 1.83 s | 20 | 18 ms | 20 |
| me | 30 | 0.48 s | 22 | 15 ms | 22 |
| settings overview | 22 | 0.24 s | 20 | 10 ms | 20 |
| models | 1,033 | 49.19 s | 27 | 2.44 s | 27 |
| activity types | 1,033 | 32.87 s | 27 | 2.42 s | 27 |
| shell attributes | 1,033 | 57.31 s | 26 | 2.41 s | 26 |
| currencies | 1,034 | 25.45 s | 29 | 2.43 s | 29 |
| finance budgets | 1,033 | 28.03 s | 28 | 2.46 s | 28 |
| approval workflows | > 602 (stopped) | > 60 s | 28 | 32 ms | 28 |
| roles | 25 | 17.17 s | 21 | 15 ms | 21 |
| access people | 25 | 6.14 s | 21 | 16 ms | 21 |

| Record pages (plan of 500 activities) | Statements before | Time before | Statements after | Time after | Statements after, demo plan |
|---|---|---|---|---|---|
| plan | 27,049 | 47.33 s | 39 | 54 ms | 40 |
| plan tree | 7,392 | 9.05 s | 43 | 48 ms | 43 |
| plan activities | 7,187 | 3.16 s | 43 | 48 ms | 43 |
| plan financials | > 74,228 (stopped) | > 60 s | 29 | 38 ms | 29 |
| plan budget by month | > 7,130 (stopped) | > 60 s | 38 | 65 ms | 38 |
| plan copy options | > 12,997 (stopped) | > 60 s | 36 | 51 ms | 36 |
| campaign | > 1,446 (stopped) | > 60 s | 45 | 45 ms | 45 |
| initiative | > 198 (stopped) | > 60 s | 47 | 45 ms | 47 |
| activity | 72 | 45.47 s | 49 | 25 ms | 49 |
| activity form | > 314 (stopped) | > 60 s | 43 | 0.44 s | 43 |
| activity line items | 54 | 20.97 s | 36 | 17 ms | 36 |
| activity approval | 69 | 8.47 s | 42 | 29 ms | 42 |
| activity destinations | 34 | 2.33 s | 32 | 20 ms | 32 |
| activity history | > 99,095 (stopped) | > 60 s | 18 | 0.11 s | 21 |
| line item | 56 | 5.54 s | 38 | 18 ms | 38 |
| line item form options | 54 | 2.53 s | 35 | 17 ms | 35 |
| all work record | 28 | 5.09 s | 26 | 12 ms | 26 |

| Changes (plan of 500 activities, template of 100) | Statements before | Time before | Statements after | Time after | Statements after, demo plan |
|---|---|---|---|---|---|
| approve | 123 | 79 ms | 55 | 42 ms | — |
| copy plan | 12,033 | 3.81 s | 63 | 45 ms | 63 |
| copy campaign | 16,215 | 5.24 s | 62 | 60 ms | 62 |
| copy initiative | 15,753 | 5.14 s | 61 | 56 ms | 61 |
| copy activity | 113 | 0.15 s | 59 | 27 ms | 59 |
| move activity | 99 | 0.16 s | 56 | 26 ms | 56 |
| move initiative | 15,594 | 5.24 s | 58 | 56 ms | 58 |
| delete plan | 1,170 | 0.69 s | 38 | 59 ms | 38 |
| undo delete | 1,883 | 0.69 s | 40 | 59 ms | 40 |
| restore | 1,998 | 0.73 s | 61 | 67 ms | 45 |
| delete permanently | 23 | 4.02 s | 21 | 84 ms | 21 |
| submit and recall | 310 | 0.40 s | 120 | 0.14 s | — |
| empty recycle bin | 326 | 4.20 s | 21 | 0.12 s | 21 |

| Approvals | Statements before | Time before | Statements after | Time after | Statements after, demo data |
|---|---|---|---|---|---|
| approval monitoring | > 72,730 (stopped) | > 60 s | 37 | 37 ms | — |
| approval workflow | > 83,362 (stopped) | > 60 s | 35 | 0.57 s | — |
| approval delegations | 35 | 57 ms | 22 | 17 ms | — |
| activity approval | 86 | 0.11 s | 43 | 33 ms | — |
| approval preview | 148 | 0.34 s | 50 | 43 ms | — |
| decline | 123 | 0.13 s | 55 | 42 ms | — |
| mark notifications read | 25 | 21 ms | 18 | 11 ms | — |

| Settings | Statements before | Time before | Statements after | Time after | Statements after, demo data |
|---|---|---|---|---|---|
| settings people | 30 | 0.72 s | 26 | 32 ms | 26 |
| settings team | 23 | 23 ms | 20 | 17 ms | 20 |
| display currencies | 20 | 12 ms | 18 | 10 ms | 18 |
| access changes | 21 | 0.14 s | 19 | 0.10 s | 19 |
| access rules | 25 | 22 ms | 21 | 16 ms | 21 |
| activity form | 19 | 16 ms | 17 | 11 ms | 17 |
| current rates | 20 | 13 ms | 18 | 10 ms | 18 |
| euro rates | 20 | 12 ms | 18 | 10 ms | 18 |
| rate imports | 18 | 11 ms | 16 | 10 ms | 16 |
| model impact | 1,026 | 0.98 s | 24 | 0.32 s | 24 |
| activity type impact | 1,027 | 0.95 s | 25 | 0.33 s | 25 |
| attribute impact | 1,033 | 1.08 s | 26 | 0.42 s | 26 |
| currency impact | 1,028 | 0.93 s | 25 | 0.34 s | 25 |
| rate impact | 19 | 13 ms | 17 | 10 ms | 17 |
| person impact | 23 | 23 ms | 20 | 19 ms | 20 |
| team impact | 23 | 23 ms | 20 | 18 ms | 20 |
| roles impact | 25 | 22 ms | 21 | 15 ms | 21 |
| rate import check | 19 | 14 ms | 17 | 12 ms | 17 |

| Imports | Statements before | Time before | Statements after | Time after | Rows in the file |
|---|---|---|---|---|---|
| exchange rates (016) | 1,901 | 0.39 s | 26 | 24 ms | 1,000 |
| Shell attribute options (016) | 4,023 | 0.80 s | 23 | 20 ms | 1,000 |
| GSAP check (015) | > 183,075 (stopped) | > 60 s | 29 | 0.52 s | 5,000 |
| GSAP import (015) | > 182,825 (stopped) | > 60 s | 32 | 0.52 s | 5,000 |

## R8 — Still above a spec target at the target volume, or still growing (follow-ups)

No list or record endpoint measured sends more statements on the volume than on the demo data, except those
listed here. The remaining time on the large pages is Go computation over every record shown — roll-ups,
completeness and the JSON of answers of 10–25 MB — which this feature does not touch (R6).

**Counts that still grow, and why**

| Where | After | Why it grows | Follow-up |
|---|---|---|---|
| All work, five years | 65 statements (47 for one year) | a few reads per year of the range, at most five years | none needed: bounded by the range limit |
| Recycle bin | 22 (19 on the demo) | one read per kind of record in the bin, at most the number of kinds | none |
| Home summary, attention | 49 and 46 (45 and 42 on the demo) | cards with data on the volume read their sources once; the count does not grow further | none |
| Restore | 61 for the template of 100 activities (45 for the demo plan) | 4 statements for each approval round open at the delete: end it, its history entry and ID, and the read of the open rounds after the change | end the open rounds of a restore together |
| GSAP import of known rows | constant for new rows | an updated or relinked transaction is still saved by itself (`PutTransaction`) | save updated rows together |
| Approval escalations (scheduler) | one round at a time | each escalated round is saved with its notifications | batch when escalations are measured |

**Still above a spec target on the volume** (statements constant; the time is computation and encoding)

| Page | Time after | Target |
|---|---|---|
| Plans list (all, search, draft) | 1.37–1.47 s | 004 SC-009: under 1 s with 200 plans |
| Home summary · attention · my activities | 6.14 s · 7.72 s · 6.71 s | 009 SC-004: 3 s for 50 plans and 5,000 activities |
| All work grid, search, calendar, budget, options | 2.11–2.28 s | 010 SC-004: under 2 s |
| All work, five years | 11.25 s | 016 SC-006: the one-year time |
| Financials list | 6.32 s with 5,011 transactions | 015 SC-004: under 1 s with 20,000 |

Within their targets now: a plan of 500 activities opens in 54 ms (004 SC-009: 2 s), the Activity log in
0.11–0.12 s with 50,000 entries (011 SC-006: 2 s), a 5,000-row GSAP file is checked and imported in 0.52 s
each (015 SC-004: 5 s), a 1,000-row rate file in 24 ms (016 SC-002: 10 s), the people list in 20 ms
(003 SC-002).

**Other follow-ups**

- Settings lists (models, activity types, Shell attributes, currencies, finance budgets): 2.41–2.46 s with
  27–29 statements; their usage counts work through every activity and line item, and read the recycle
  bin's entries five times (a constant).
- Approval workflow details: 0.57 s, an answer of 7.9 MB.
- `ListLineItemsInScope` still uses correlated subqueries; an index on `activities.split_from_id` would
  serve the "split from" read (no migration in this feature).
- Access changes reads the 50,000 log entries in one statement (0.10 s); plan copy options read every plan.
- PostgreSQL's JIT on the largest reads, and `otelpgx` traces per statement (ADR-0011).

## R9 — Running the benchmarks

```sh
cd backend
export TEST_DATABASE_URL='postgres://mapp@127.0.0.1:5432/postgres?sslmode=disable'
go test -run '^$' -bench 'BenchmarkLists|BenchmarkRecords|BenchmarkChanges|BenchmarkApprovals|BenchmarkSettings|BenchmarkGSAPImport' \
  -benchtime 1x -timeout 3h ./internal/platform/apibench/
```

The first run builds the volume (about five minutes); later runs copy it. The templates stay on the server;
drop `mapp_perf_*` databases to free the space.
