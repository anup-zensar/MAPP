# ADR-0016: A simulated GSAP for SWBS codes, GSAP PRs and purchase orders

**Status:** Accepted · **Date:** 2026-10-27

## Context
In Shell's process (Hive9 functional analysis D3–D5) an approved activity's line items ask GSAP for an
SWBS code in a daily file (FSSI 1523), and GSAP returns one code per line item within 24–48 hours. The
marketer then raises a purchase request (PR) on the line item; once approved it goes to a requisitioner,
who builds the GSAP PR (typing the HX key, "HX" + the line-item number), and GSAP turns it into a purchase
order (PO). GSAP's PR number, PO number and PO lines come back in a daily file (PGSI 941), matched on the HX
key.

GSAP integrations are frozen until about March 2027 and MAPP has no GSAP access. ADR-0009 covers only
inbound simulated feeds and says MAPP never sends anything to GSAP. On 2026-10-27 the product owner asked
for the whole PR→PO process to be demonstrable now, with GSAP's part mimicked (constitution 4.4.0, D-320,
D-321).

## Decision
- **A simulated GSAP, in process, behind adapter interfaces owned by the packages that need them.** It is
  not a third runtime part (constitution IV): it runs inside the API, only when an operation calls it, and
  holds no state of its own beyond what the owning package stores.
  - **SWBS codes (feature 030).** The package that owns SWBS requests declares a small interface for "send
    these requests, get these codes back". The simulated implementation answers every request at once with
    one code per line item, numbered in sequence per company code, and every simulated code carries the
    marker `SIM`, so nobody takes it for a real GSAP code. The batch is started by an API operation, which
    stands for GSAP's daily FSSI 1523 run and its response file; the screen offers it as a button.
  - **GSAP PR and purchase order (feature 033).** The purchase-request package (032) declares an interface
    for "this routed PR → its GSAP PR number and PO". The simulated implementation creates a GSAP PR number,
    a PO number and one PO line per PR line, keyed by the line's HX key, again marked as simulated. The PO
    lines are recorded as GSAP purchase-order transactions through the existing `financials` service
    (feature 015), so Committed figures, matching and the Financials page work unchanged.
- **The rules stay in the services.** Which line items need a code, when a request is sent, what happens
  when an activity leaves approval after a request was sent, which PRs may get a PO, reservations and
  figures all belong to the owning services; the simulated adapters only turn a request into GSAP's
  answer.
- **Who may run it:** `finance.import`, the permission that already loads simulated GSAP data (ADR-0009).
  Every run writes one activity-log entry naming the simulated GSAP as its source.
- **Labelled everywhere.** Screens and the PR form name the source "Simulated GSAP".

## Consequences
- **Live GSAP replaces a simulation by adding implementations of the same interfaces**: an FSSI 1523
  writer and a response reader for SWBS codes, and a PGSI 941 reader for PR and PO data. Services, rules,
  the API contract and the screens do not change. A scheduled job or SAP Cloud Integration connection is
  a new runtime part and gets its own ADR when MAPP is hosted.
- The simulated SWBS code format and the simulated GSAP PR and PO numbers are invented; Shell's real
  formats replace them when the integration document arrives (FA D9).
- ADR-0009's "inbound and read-only" still holds for the transaction and rate feeds; this ADR is the only
  place MAPP sends something to (a simulation of) GSAP.
- Real GSAP behaviour the simulation does not reproduce — rejected SWBS requests, unmatched HX keys, GSAP
  approvals and PO release thresholds — stays with live integration.
