# Feature Specification: Query Efficiency

**Feature Branch**: `feature/028-query-efficiency`

**Created**: 2026-10-26

**Status**: Draft

**Input**: User description: "Query efficiency (feature map row 028): feature 023 moved every store to
PostgreSQL, but the services still load related records one at a time inside loops, as they did when every
read was a map lookup. Loading the Plans list once sends about 600 SQL queries, mostly the same person,
finance budget, approval workflow or exchange rates read again and again, and the count grows with the data.
Meet the scale targets the specs already set, with behaviour and the API contract unchanged. Cover every
N+1: list pages, single-record pages, Settings, people and teams, and the operations that change many
records at once (copy, move, delete, restore, bulk actions, imports, approval decisions). Count the queries
of every request in the request log."

**References**: Constitution 4.1.0 — Principle IV (PostgreSQL in every environment), V (simplicity; no
premature optimisation: every optimisation backed by a measurement), VI (only end-to-end and scope-isolation
tests use a database), VII (scope isolation), B5 (no query per row on list endpoints), B7 (observability),
B9 (a benchmark for every optimisation) · `backend/docs/GO_STANDARDS.md` §6, §11 Performance, §12 · feature
023 research R11 (the per-transaction memo, the open N+1) · the scale targets 003 SC-002, 004 SC-009, 009
SC-004, 010 SC-004, 011 SC-006, 012 SC-007, 015 SC-004, 016 SC-002 and SC-006 · feature map rows 023
(PostgreSQL storage) and 025 (backend layout)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Lists and overviews stay fast with a real year of planning (Priority: P1)

As a planner or a manager I open the Plans list, a plan, Home, All work, the Approvals board, the Activity
log, the recycle bin, Financials or the global budgets with a full year of data — 50 plans, 5,000
activities and 20,000 line items in my teams, hundreds of plans and tens of thousands of log entries in the
whole of MAPP — and every page answers within the time its feature promised, with exactly the content it
shows today.

**Why this priority**: the specs set these times when every read was free; on PostgreSQL each repeated read
is a round trip to the database, and the pages slow down as the data grows.

**Independent Test**: build the volume of the targets in a benchmark database, request each page through
the real API and compare its time with the target; the same pages on the demo data and on the volume send
the same number of queries.

**Acceptance Scenarios**:

1. **Given** 200 or more plans, **When** a person searches or filters the Plans list, **Then** the result
   answers in under 1 second (004 SC-009); **and** a plan with 500 activities opens — overview, tree,
   financials and budget by month — in under 2 seconds.
2. **Given** a person whose scope holds 50 plans and 5,000 activities, **When** they open Home, **Then**
   every card answers within 3 seconds (009 SC-004).
3. **Given** 5,000 activities and 20,000 line items in one year, **When** a person changes a filter, the
   search or the view of All work, **Then** the result answers in under 2 seconds (010 SC-004); a 5-year
   range with 25,000 activities shows its first page in the same time (016 SC-006).
4. **Given** 50,000 entries in the activity log, **When** a person opens the Activity log or changes a
   filter, **Then** the first entries answer in under 2 seconds (011 SC-006).
5. **Given** 1,000 people, **When** a person types in a people picker, **Then** the matches answer in under 1
   second (003 SC-002).
6. **Given** any list or overview page, **When** it is requested on the demo data and on the volume,
   **Then** it sends the same number of queries: the count does not grow with the records it shows.

---

### User Story 2 - Changes to many records stay fast and stay one change (Priority: P1)

As a planner I copy a plan with everything in it, copy a campaign, move an activity or an initiative,
delete a plan and undo it, restore from the recycle bin or empty it, run a bulk action on 500 records,
import a file, or submit and decide approvals, and each operation finishes quickly — still all or nothing,
in one transaction, with the same activity-log entries as today.

**Why this priority**: these operations touch hundreds of records at once, and a read or an insert per
record makes their time grow with the plan.

**Independent Test**: run each operation through the real API on the volume and count its queries and
time; run the service, contract and end-to-end tests unchanged.

**Acceptance Scenarios**:

1. **Given** a plan of 100 activities and 400 line items, **When** it is copied with its contents, **Then**
   the copy's queries grow at most with the levels of the tree, not with its records, and the copy and its
   one "copied" log entry are saved in one transaction.
2. **Given** 500 selected records or a 1,000-row file, **When** the summary or the check is requested,
   **Then** it answers within 10 seconds (012 SC-007); a 5,000-row GSAP file is checked and imported in
   under 5 seconds (015 SC-004); a 1,000-row rate file shows its review within 10 seconds (016 SC-002).
3. **Given** a delete, a restore or an Empty recycle bin of a plan with everything below it, **When** it
   runs, **Then** it writes the same recycle-bin and log entries as today in one transaction.

---

### User Story 3 - The request log shows the database work of each request (Priority: P2)

As a developer I read the request log and see, next to each request's route, status and duration, how many
SQL statements it sent, so a page whose statements grow with its records (N+1) is visible the day it
appears.

**Why this priority**: the problem of this feature went unnoticed because nothing showed it; the count is
cheap and needs no new dependency.

**Independent Test**: a handler that sends three statements logs `"queries": 3` in its request record.

**Acceptance Scenarios**:

1. **Given** a request, **When** it ends, **Then** its one request record carries `queries`, the number of
   SQL statements sent while serving it, beginning and committing transactions included.

### Edge Cases

- A reference record that is missing (a deleted person, an inactive workflow): the page shows it exactly as
  today (by its ID, or "inactive"); batching never turns a missing record into an error.
- A record read, then changed, then read again inside one save: the second read sees the change.
- Two requests at the same time: nothing one request read is reused by another; what is kept lives only as
  long as one transaction.
- A list asked for by IDs that include duplicates or IDs that no longer exist: each existing record is
  returned once, missing ones are left out, and the order of the answer is the order the page shows today.
- A person's scope: a batched read returns only what the person may see, exactly as the reads it replaces.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every endpoint MUST answer exactly as before — status, body and order — and the API contract
  MUST NOT change.
- **FR-002**: A list or overview endpoint (Plans list, plan overview, tree, activities, financials and
  budget by month, Home's cards, All work's views and options, the Approvals board and summary, the
  Activity log and its summary, the recycle bin, Financials' list and overview, global budgets) MUST send a
  number of queries that does not grow with the number of records it reads.
- **FR-003**: A single-record page (activity, line item, campaign and initiative panels, the activity's
  approval and history, Settings lists, people and teams) MUST send a number of queries that does not grow
  with the records below or around it.
- **FR-004**: An operation that changes many records (copy of a plan, campaign or initiative, a move, a
  delete with everything below, a restore, a permanent delete or Empty recycle bin, a bulk action and its
  undo, an import and its undo, an approval submission or decision) MUST read and write the records of
  each level together, so that its queries do not grow with the number of records, and MUST stay one
  transaction with the activity-log entries it writes today.
- **FR-005**: Reference data a request reads repeatedly (people, teams, Settings, approval workflows,
  currencies and rates) MUST be read at most once per transaction, and nothing read MUST be kept beyond the
  transaction that read it: never across requests.
- **FR-006**: A value kept inside a transaction MUST be read again after that transaction changes data, so
  a save always sees its own changes.
- **FR-007**: Scope filtering MUST stay in SQL; a read that replaces scoped reads returns only what the
  scope sees, and every new scoped read MUST have a scope-isolation test.
- **FR-008**: Every optimisation MUST be backed by a benchmark on PostgreSQL that reports the queries per
  request and the time, on a database holding the volume of the scale targets, built by benchmark helpers
  (test code) and skipped under `-short`; the before and after numbers are recorded in research.md.
- **FR-009**: The request log record MUST carry `queries`: the number of SQL statements the request sent.

### Key Entities

- **Volume database**: a benchmark database holding the demo data plus the volume of the scale targets,
  built once per migration set and copied for each benchmark.
- **Transaction memo**: what one transaction read or worked out, kept until it changes data or ends.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On the volume, the Plans list (with search and with each filter) answers in under 1 second and
  the four requests of the 500-activity plan page in under 2 seconds together (004 SC-009).
- **SC-002**: On the volume, Home's cards answer within 3 seconds together for Priya Raman, whose scope
  holds 50 plans and 5,000 activities (009 SC-004).
- **SC-003**: On the volume, every All work view and filter answers in under 2 seconds for one year, and its
  first page for five years (010 SC-004, 016 SC-006).
- **SC-004**: On the volume, the Activity log's first page and each filter answer in under 2 seconds (011
  SC-006); a people search with 1,000 people in under 1 second (003 SC-002).
- **SC-005**: On the volume, a bulk preview of 500 records and the check of a 1,000-row file answer within
  10 seconds (012 SC-007); a 5,000-row GSAP import within 5 seconds (015 SC-004).
- **SC-006**: For every endpoint and operation measured, the queries per request are the same on the demo
  data and on the volume, or research.md explains why a count that still grows is acceptable.
- **SC-007**: 100% of the existing service, handler, contract and end-to-end tests pass unchanged.

## Assumptions

- The volume is fictitious, like the demo data: copies of the demo plans made through the API, people,
  log entries and GSAP rows added by benchmark helpers. No product flag or seed option is added for it.
- Benchmarks measure; they are not tests. They assert nothing about queries, constraints or SQL, so the
  constitution's rule that only end-to-end and scope-isolation tests use a database (Principle VI) is kept;
  B9 requires them.
- Feature 025 (backend layout) comes later: this feature adds batch reads to the existing store interfaces
  and to the JSON stand-in, and renames or reshapes nothing.
- Times are measured on a developer laptop with PostgreSQL on the same machine, as the targets assume.
