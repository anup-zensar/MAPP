# ADR-0021: One active source per model stage is a service rule, not a database constraint

**Status:** Proposed (awaiting the product owner's approval) · **Date:** 2026-10-30

## Context
Constitution B5 requires a relational schema "with constraints for every invariant", and Governance →
Exceptions requires an approved ADR for a deviation from a MUST. Feature 046 carries an invariant that cannot
be expressed as one: client 020 FR-003 — within a model, each funnel stage and revenue is supplied by **at most
one active data source**; a conflicting mapping is refused, naming the source that already supplies it.

The invariant spans two tables. The stage and the model live on `performance_source_mappings`; whether the
source counts lives on `performance_sources.status`. Three routes were tried:

1. **A partial unique index** on `(model_id, stage)` whose predicate selects only mappings of active sources.
   PostgreSQL rejects it: an index predicate may not contain a subquery. This was written, run, and failed.
2. **A plain unique index** on `(model_id, stage)` with no status condition. It is wrong rather than merely
   strict: FR-004 keeps a deactivated source's mappings and facts intact, and FR-002 offers Deactivate
   precisely so a source can be stood down without losing history. A plain index would refuse a new active
   mapping for a stage whose only other supplier is inactive — the normal way of replacing a feed.
3. **Denormalising** the source's status onto every mapping row, kept in step by a trigger, so a partial index
   can see it.

## Decision
The rule is enforced in the service, in `backend/internal/performance/mapping.go`, which refuses a conflicting
mapping and names the source already supplying that stage. The database carries supporting indexes on
`(model_id, stage)` and `(model_id)` for revenue so the check is a single indexed read, and the reason it is
not a constraint is written in migration 00030 beside them.

Option 3 is rejected. The duplicated status column can drift from its source of truth, and a trigger puts
behaviour in the database that constitution IV reserves for the service layer. The machinery costs more than
the invariant is worth, and it would be the project's only such trigger.

## Consequences
**Good.** The refusal can name the conflicting source, which a unique-constraint violation cannot do without
a second query — and FR-003 requires that name. The rule is unit-testable on fakes, as constitution VI
prefers, rather than only on a real database.

**Bad.** A write that bypasses the service could create a second active supplier for a stage. Nothing in MAPP
writes outside the service, and the seed goes through the same path, but the guarantee is now conventional
rather than structural. Every new write path for mappings must call the same check.

**Neutral.** This is the second deviation of its kind in the project: ADR-0019 records the overspend guard's
unscoped budget read for comparable reasons. Both share a shape — an invariant whose scope is wider than one
row — and both stay in the service with the reasoning written down.

## Related
Constitution B5, IV and Governance → Exceptions · ADR-0019 (the overspend guard's unscoped read) ·
client 020 FR-002, FR-003, FR-004 · migration `00030_performance_sources.sql` ·
`specs/046-performance-data/plan.md` rule R9.
