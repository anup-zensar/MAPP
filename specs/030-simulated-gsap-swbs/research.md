# Research: SWBS Codes from a Simulated GSAP

Each entry: the decision, why, and what was rejected. Sources: client specs 005 and 015, Hive9 functional
analysis (FA) §B7 and §D3–§D7, ADR-0009, ADR-0016, decisions D-255, D-304 and D-321.

## R1 — Where SWBS requests live

**Decision**: a new business package `internal/swbs` owns SWBS requests and runs; package `lineitem` keeps
the line item's own SWBS fields, its status rule, its HX key and the rule that stores a received code.
The two meet through a consumer-defined port (`swbs.LineItems`) implemented in `internal/app`.

**Why**: runs are records of their own (ID, who, when, codes) with a lifecycle unlike a line item's
(constitution B2: one package per business area); ADR-0016 asks for the adapter interface in "the package
that owns SWBS requests". Package `lineitem` already has some fifty files; putting runs there would mix two
record types. The port keeps the service testable on hand-written fakes.

**Rejected**: everything in `lineitem` (lower cohesion, its Store and both fakes would grow run methods);
`swbs` importing `lineitem` directly (acyclic today, but the repository's rule is ports through
`internal/app`, and fakes would then need the whole line-item service).

## R2 — The adapter interface

**Decision**: `swbs.GSAP { RequestCodes(ctx, []Request) ([]Answer, error) }`; `Request` carries the line
item's ID, name and company code; the simulated implementation answers every request at once.

**Why**: the simulation needs nothing more; a live FSSI 1523 writer plus response reader will implement the
same interface (ADR-0016). Answers that arrive later in live GSAP would add a "sent, waiting" status then —
the statuses of brief §8.1 have none now.

**Rejected**: a request carrying the whole "Record Content" of client 015 (amount, GL code, cost centre…):
nothing reads it now (constitution V, YAGNI); it is added with the live writer.

## R3 — Numbering the simulated codes

**Decision**: `SIM-<company code>-<nnnn>`, `nnnn` = codes already issued with that prefix + position in the
run, counted from the stored run codes (`CountCodes(prefix)`); `0000` stands in for a missing company code,
and a company code is kept to ten letters and digits, upper case; the run sends line items in ID order.

**Why**: FA §D7 — codes are sequential per company code and never duplicate; ADR-0016 — every simulated
code carries `SIM`; the simulation keeps no state of its own beyond what the owning package stores.
Runs are serialised by `database.Write`'s lock, and `swbs_run_codes.swbs_code` and
`line_items.swbs_code` are unique, so a number is never given twice.

**Rejected**: the shared ID counters (`counters.prefix` allows letters only, `last ≥ 1001`); a random code
(not reproducible in tests or the demo).

## R4 — The company code

**Decision**: the activity's Shell attribute labelled "Company code" (activity level), the label of the
chosen option (GB01, NL02, …); read in the port implementation in `internal/app` (one read of attributes
and options per run).

**Why**: FA §D5 — the company code is an activity attribute; Settings has no role marker for it, and adding
one is Settings' business (feature 002). The approval preview already finds line-item coding by label.

**Assumption for the product owner**: the attribute keeps the label "Company code".

## R5 — Status order and locking

**Decision**: received wins over every other status; a code is never removed; the request flag cannot be
switched off once a code exists (API field problem, UI locked switch with the reason; database check
`swbs_code = '' OR request_swbs`).

**Why**: client 005 FR-015 (editable until the request is sent), client 015 FR-039 ("Sent and confirmed
line items keep their status"), D-321 ("the activity leaving approval does not undo it").

## R6 — Named line items

**Decision**: `lineItemIds` limits a run; an ID the actor cannot see answers 404 `not_found` (scope rule:
indistinguishable from missing); an ID that is not ready answers 409 `not_allowed` naming the line items;
nothing is sent then. An empty list is a validation problem; leaving the field out sends everything ready.

**Why**: "no surprises" (constitution IX): the line item page's action must say when its line item was
already sent rather than record an empty run silently.

## R7 — Empty runs

**Decision**: a run that finds nothing ready is kept and logged ("Nothing was ready to send").

**Why**: client 015 FR-007 logs a run with nothing to send; the panel offers the action only when something
is ready, so empty runs come only from races.

## R8 — Who sees runs

**Decision**: running and the history need `finance.import`; the log entry is visible with `finance.import`
or `activityLog.viewAll` (as GSAP imports, D-304).

**Why**: ADR-0016. `finance.import` is not a scoped permission (014 Table A), so its scope is All markets
and the history needs no scope filter; the selection of line items still goes through the SQL scope
filter of `ActivityStore.ListInScope`.

## R9 — Demo data

**Decision**: the seed's financials step first records run `SWR-1001` (source `demoData`, by MAPP at the
seed time, no log entry per D-188) through `swbs.LoadDemo`, the same send path as a run, giving Venue hire
(L-1003, DE03) `SIM-DE03-0001`; Stand build (L-1006, GB01) stays Ready to send. The demo GSAP transactions
of these two line items (feature 015's demo feed) carry `SIM-DE03-0001` and `SIM-GB01-0001`.

**Why**: brief — "give a few approved demo line items received codes … so the demo shows every status";
only two approved line items request a code, so one of each status remains. In Shell's process a PO
carries the line item's SWBS code (FA §D5); aligning the demo transactions keeps the line item and its
GSAP data consistent after the demo's first run. Matching by SWBS code (015 rule 4) still links the
Stand build payment, because its PO and payment share the code.

**Rejected**: adding a new demo line item (changes many features' demo figures and tests); leaving the
transactions' invented codes (`DE03-M26-0003`, `GB01-M26-0006`) (the line item and its GSAP data would
disagree on screen).

**Decision for the log**: "The demo GSAP transactions of Venue hire and Stand build carry the simulated
codes SIM-DE03-0001 and SIM-GB01-0001 (feature 030); D-255 is superseded by D-321."

## R10 — The Financials section's SWBS line

**Decision**: the line item page's Financials section no longer shows "SWBS code: … (from GSAP data)";
the SWBS request card shows the line item's own code.

**Why**: D-255 ("the code shown comes from the transactions") is superseded by D-321 ("the line item now
stores it"); one page must not show two sources for one value.

## R11 — Where the panel and the actions sit

**Decision**: the Simulated GSAP panel is a card after the transactions on the Financials page, shown only
to people with `finance.import`; its action is a secondary button ("Send SWBS requests to simulated
GSAP"), so the page keeps at most one yellow primary; the confirmation dialog's confirm button is the
dialog's one primary. The line item page's "Send to simulated GSAP now" is a secondary button inside the
SWBS request card; Edit stays the page's primary.

## R12 — Comment budget and file length

Every touched file is brought within 5 % comments (constitution VIII); `line-item-text.ts` (296 lines)
loses its SWBS texts to the new `swbs-text.ts` so it stays under 300 lines.

## Dependencies

None added (constitution B1, F1).
