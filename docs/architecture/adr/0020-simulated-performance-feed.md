# ADR-0020: Performance data arrives only by upload or a simulated feed behind an adapter

**Status:** Proposed (awaiting the product owner's approval) · **Date:** 2026-10-30

## Context
The Measure module's every actual figure — INQ, MQL, deals, revenue, web metrics — resolves to a performance
fact matched to an activity (`docs/product/measure-metrics.md` §2). Hive9 populated those facts from live
connectors to web analytics, marketing automation and CRM systems (functional analysis §E7.2), and its own
product guide says the module "only adds real value when connectors populate the performance data".

MAPP cannot have those connectors. Constitution 4.8.0 non-negotiable 13 puts live performance connectors out
of scope, and D-370 states that performance data comes only from a simulated feed behind an adapter. The same
problem was already solved once for Shell's GSAP: ADR-0009 put the GSAP feeds behind adapters, and ADR-0016
built the simulated GSAP that returns SWBS codes, GSAP PR numbers and purchase orders on demand.

Feature 046 must therefore decide how rows reach MAPP at all, and client 020 FR-008 constrains the answer:
"any future automated feed passes through the same rules" as an uploaded file.

## Decision
Performance rows reach MAPP by exactly two routes, and both end in the same code path.

1. **File upload.** A CSV or spreadsheet for a chosen data source, uploaded by someone holding
   `performance.import`.
2. **A simulated pull.** An adapter package, `internal/performance/simulatedfeed`, sits behind a port the
   service owns. It produces rows for a source whose delivery route is `simulated`.

A pull is started **only by an explicit action on the data source**, behind the same `performance.import`
permission — one operation, no schedule, no background worker, no external credential. This mirrors feature
033, where the requisitioner opens a purchase request and chooses "Create GSAP PR and PO (simulated)".

Rows from either route enter the identical pipeline: whole-file personal-data refusal, per-row validation
reporting every fault at once, set-based matching on the four keys in their strict order, and the five-outcome
summary that must reconcile against rows read. The service cannot tell which route a batch came from, and
nothing downstream branches on it; only the import record keeps an `origin` for the audit trail.

Everything the adapter produces is labelled as simulated on screen, as ADR-0016 requires of the simulated GSAP.

## Consequences
**Good.** The upload path is the only path, so it gets all the testing: there is no second, less-exercised
route that a live integration would later have to match. Replacing the adapter with a real connector means
implementing one port, and the validation, matching and review behaviour it must satisfy is already pinned by
tests. Determinism holds — a fixed clock and seed are enough, because nothing runs on a timer (constitution
VI). Nothing new needs to be kept secret, so constitution non-negotiable 14 is untouched.

**Bad.** Freshness (client 020 FR-005: *Current · Stale · Failed · Inactive*) is driven by a cadence that
nothing automatically honours, so a source goes *Stale* in the demo unless somebody pulls it. That is accurate
rather than misleading — it is exactly what a late feed looks like — but it does mean the demo data must be
pulled deliberately to look current.

**Bad.** A real connector will eventually need retry, partial-batch and back-pressure behaviour that this
decision gives no place to. That is accepted: designing those now would be speculative, and the port is the
seam where they will go.

**Neutral.** The adapter's generated values must not collide with Hive9's historical ones; media-code
numbering therefore starts from a configurable number (client 020 FR-040), as the simulated GSAP does with its
`SIM-` prefix.

## Related
ADR-0009 (simulated GSAP feeds behind adapters) · ADR-0016 (the simulated GSAP round trip) ·
D-370 · constitution 4.8.0 non-negotiable 13 · feature 046 clarification 046-C13 ·
`specs/046-performance-data/research.md` R1.
