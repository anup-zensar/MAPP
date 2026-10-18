# ADR-0007: Activity log and recycle bin core

**Status:** Accepted · **Date:** 2026-10-18

## Context
Every business change must be logged with the actor, and every delete must go to a recycle bin from which
the record and everything under it can be restored (constitution VI, feature 001 FR-022 – FR-027). Plans,
campaigns, initiatives, activities and line items live in different packages, and a delete must cascade
across them. Feature 011 builds the activity-log and recycle-bin screens later, but it can only show what
the entries written by features 002–010 already contain: the record's path at the time, readable values
and any comment. The log grows with every change (011 plans for 50,000 entries).

## Decision
- Package `internal/history` owns the mechanism. Each business package registers a **kind** per record
  type: names, its parent kind, and small functions to snapshot, list children, remove and put back a
  record, plus optional hooks to describe field values and to adjust a record before it is restored.
  Only `internal/app` wires packages together, so there are no import cycles.
- `history.Log` writes one entry per business change inside the same transaction as the change, with the
  path, the verb, readable labels and values, the comment and plain-text details.
- `history.MoveToBin` keeps the deleted record and its descendants as snapshots in one recycle-bin entry;
  Undo (2 minutes) restores exactly; Restore applies the kinds' restore rules. Resolved entries are kept
  (how, by whom, when) without their snapshots.
- In the JSON store the activity log is an append-only JSON Lines file; in PostgreSQL it will be an
  append-only table.

## Consequences
- Every feature from 002 on registers its kinds and logs through `history.Log`; feature 011 adds screens,
  "Restore both", permanent delete and the restore preview without changing stored entries.
- Optional hooks needed later (`AfterRestore`, `BeforeDelete`, a restore preview) are added as new
  optional fields, so existing registrations do not change.
- Entries are never updated or deleted.
