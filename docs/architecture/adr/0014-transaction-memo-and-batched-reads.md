# ADR-0014: Transaction memo and batched reads

**Status:** Accepted · **Date:** 2026-10-27

## Context
The services were written when every read was a map lookup (the JSON file store), so they read related
records one at a time inside loops. On PostgreSQL each of those reads is a round trip: on a database with
256 plans, 25,000 activities and 100,000 line items the Plans list sent more than a million statements
(feature 028, `specs/028-query-efficiency/research.md`). Constitution B5 forbids a query per row on list
endpoints; feature 025 reshapes the store interfaces later, so the fix must keep today's interfaces and
services.

## Decision
- **One memo per transaction** (`internal/platform/memo`), carried in the context by `database.DB.Read`
  and `Write` alike and dropped with the transaction: nothing read is ever kept across transactions or
  requests.
- **By area**: a name is `<area>:<key>`. Each PostgreSQL store keeps what it reads under its own area and
  forgets the area before any change it makes (`change(ctx)`), so a save always reads its own changes.
  Values worked out from several areas (an activity's completeness, a line item summary) live in the area
  `Derived`, which any change forgets.
- **Reference tables whole** (`memo.Table`): Settings, people, teams, roles and approval workflows are read
  once per transaction and found by id in memory; rows are kept, and each call decodes a fresh value.
- **Planning records by id and by parent** (`memo.Records`): plans, campaigns, initiatives, activities and
  line items read for many parents at once are found again without a statement; a missing id is kept as
  missing.
- **Batch reads and writes** on the existing store interfaces: lists by many ids (`= ANY(…)`), inserts of
  many new records by COPY, deletes by many ids, ID ranges (`ids.Generator.NextIDs`). The JSON stand-in
  implements them one by one.
- **The request log counts statements** (`queries`), through a pgx tracer and `internal/platform/querycount`,
  so a new N+1 is visible the day it appears.

## Consequences
- A store that keeps rows must route every write through its `change`; a write that bypasses it would let a
  later read in the same transaction see stale data. The end-to-end suite runs every write on PostgreSQL.
- Pages whose statement count is now constant can still be slow at large volumes: they compute roll-ups in
  Go over every record they show (no roll-up is stored, Principle IV). Making them faster is computation
  work, out of this decision's scope.
- Feature 025 may turn the batch methods into the store interfaces' normal shape; the memo stays.
