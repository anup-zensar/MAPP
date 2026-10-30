# Research: Performance Data from a Simulated Feed

Only decisions with a real alternative are recorded here, per the build guide §8.

## R1 — How a simulated feed produces rows without becoming a live connector

**Decision**: an adapter package `internal/performance/simulatedfeed` sits behind a port owned by the service.
It produces rows on demand for a source whose delivery route says so, and those rows enter the **same**
validation → matching → review path as an uploaded file. A pull is started only by an explicit action on the
source, behind `performance.import` (046-C13). Recorded as **ADR-0020**.

**Why**: constitution non-negotiable 13 and D-370 put live performance connectors out of scope, and ADR-0016
already set this shape for the simulated GSAP. Feature 033 is the precedent in working code: the user opens
the record and chooses "Create GSAP PR and PO (simulated)" — one operation, one permission, no schedule.

**Alternatives rejected**:
- *A background scheduler that pulls on the source's cadence.* It is a live connector in all but name, needs
  run-time state MAPP has nowhere to keep, and makes tests non-deterministic (constitution VI requires a fixed
  clock and seed).
- *Generating rows inside the seed only.* Then the pull path is never exercised outside demo data, and the
  cadence and freshness rules in FR-005 could not be demonstrated at all.

## R2 — Matching a file without a query per row

**Decision**: matching is **set-based**. Each batch collects its distinct keys by type and resolves them in
one query per key type (four at most), building an in-memory key → activity map before any row is decided.
Row decisions are then pure functions over that map, which also makes the precedence rule (R1 of the plan)
directly unit-testable without a database.

**Why**: feature 028 exists because per-record queries were the project's recurring performance fault, and
constitution B9 requires a benchmark for every optimisation. A 50,000-row file would otherwise issue 50,000
lookups.

**Alternatives rejected**:
- *Per-row lookup with a cache.* Still O(distinct keys) round trips on a cold cache and hides the cost behind
  a hit rate that varies by file.
- *A SQL join against a temporary table of the file's rows.* Fastest, but it pushes the precedence rule and
  the "keys disagree" decision into SQL, where constitution IV says business rules may not live and where it
  could not be unit-tested on fakes.

## R3 — What identifies a fact for replacement

**Decision**: the pair **(data source, source reference)** (046-C4), carried as a unique constraint in the
database rather than enforced only in the service.

**Why**: client 020 FR-015 requires an identical file to change nothing and a repeated reference to replace.
A constraint makes the invariant true even if a future code path forgets it, which constitution B5 requires
("constraints for every invariant").

**Alternatives rejected**: *(source, activity, date, stage)* — it cannot express a correction that moves a
fact to a different activity, and two legitimately distinct facts can share all four values.

## R4 — How held facts are released on approval

**Decision**: facts carry a `held` flag written at load time, and approval releases them by clearing the flag
for that activity inside the same transaction that approves it, through a port the approvals area calls.
Figures then filter on one indexed boolean.

**Why**: the alternative — deciding "is this held?" at read time by joining to the activity's current status —
puts the approval rule in every report query of wave B, where four other lanes would each have to repeat it
correctly. One write at approval is cheaper than the same join in a dozen reports, and it keeps the rule in
this feature.

**Alternatives rejected**:
- *Compute at read time.* Spreads FR-028's status set across 048, 049 and 050; any lane that forgets it shows
  unapproved data.
- *A nightly sweep.* No scheduler exists, and the spec requires release "automatically on approval", not
  eventually.

**Consequence to report**: the approvals area must call this port when an activity reaches an approved status.
That is a change outside this feature's folders, so it is described in the pull request for the integrator
rather than made here.

## R5 — Storing revenue

**Decision**: integer minor units plus an explicit currency column, with no conversion stored. Conversion to a
display currency happens where feature 039's frame asks for it; a fact with no rate is marked and excluded with
its count (FR-014, 046-C7).

**Why**: constitution non-negotiable 9 — exact decimal and currency, never a float. Storing a converted value
would freeze a rate into history and make a later rate correction unrepresentable.
