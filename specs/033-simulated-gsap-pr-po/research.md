# Research: GSAP PR Number and Purchase Order from a Simulated GSAP (033)

"Decision for the log" marks what the orchestrator records in `docs/sdd/decision-log.md` (brief §4).

## R1. Where the code lives

- **Decision**: three small pieces.
  1. **The use case** — `purchaserequest.(*Service).CreateGSAPOrder(ctx, a, id, version)` in
     `internal/purchaserequest/gsap_order.go`: checks (scope, permission, Purchase Support, status, routed,
     change pending, waiting for GSAP, version), builds the `RoutedPR` of brief §8.3, calls the `GSAP` port
     declared by 032, stores the answer and writes the log entry, in one `Write`.
  2. **The simulation** — package `internal/simulatedgsap` (shared with 030's SWBS run if 030 created it; 033
     adds only `purchase_order*.go` files): `Orders` implements `purchaserequest.GSAP`; it numbers the GSAP
     PR, the PO and the PO lines and records the PO lines through a `Ledger` port; and a handler for the
     operation `POST /api/v1/simulated-gsap/purchase-requests/{purchaseRequestId}/purchase-order`, which calls
     the use case through a consumer interface `PurchaseRequests` (satisfied by `*purchaserequest.Service`).
  3. **The ledger** — `financials.(*Service).RecordSimulatedGSAP(ctx, a, d SimulatedDelivery)`: the rows go
     through the same applier as an import (`apply`, data model 015 §4.1) with a new in-memory feed
     (`rowsFeed`, source `simulatedGsap`), one import record and one log entry.
- **Why**: ADR-0016 — "the rules stay in the services" (which PRs may get a PO, what changed: 032's package),
  "the simulated adapter only turns a request into GSAP's answer" (simulatedgsap), "PO lines are recorded
  through the existing financials service" (Committed, matching and the Financials page unchanged). Live
  GSAP later adds a PGSI 941 reader; the use case and the screens stay.
- **Alternatives**: the whole flow inside `simulatedgsap` (it would need 032's store and rules); writing
  `financial_transactions` directly (bypasses 015's matching and import record).

## R2. Numbering

- **Decision**: counters in the existing `counters` table (`platform/ids`, prefixes `GPR` and `GPO`, uppercase
  letters only as the table's check requires): `NextNumber` gives 1001, 1002…; GSAP PR number =
  `fmt.Sprintf("SIM-%010d", 10_000_000+n)` (SIM-0010001001), PO number = `fmt.Sprintf("SIM-%010d",
  4_500_000_000+n)` (SIM-4500001001). PO lines: 10 × (position among the PR's ordered lines), continuing after
  the highest existing PO line for new lines. All within 015's document form `^[A-Za-z0-9/-]{1,20}$`, so a
  later simulated invoice file can refer to them.
- **Why**: the counters already give gap-free, never-reused numbers inside the `Write`; the "SIM-" marker is
  ADR-0016's rule (Decision for the log: formats).

## R3. When a run is allowed (the gate is 032's "waiting for GSAP")

- **Decision**: allowed when Purchase Support is on, status Approved, no change pending, at least one routing,
  and 032's `waitingForGsap` rule holds (no PO line on some line, or a line's amount/currency differs from its
  PO value). Refusals: 409 `not_allowed` with the texts of plan "Messages"; 404 out of scope; 403 without
  `finance.import`; 409 `version_conflict` when the PR changed since the page was opened.
- **Why**: one rule decides both the queue and the action, so a PR is in the queue exactly when it can be run
  (spec SC-003).

## R4. Top-up

- **Decision**: the use case passes the stored GSAP PR number, PO number and each line's PO line in the
  `RoutedPR`; `Orders` keeps them and numbers only what is missing; the ledger rows carry the same keys
  (type purchaseOrder + PO number + PO line), so 015's applier updates changed rows and leaves unchanged ones
  ("a row sent again updates or replaces what is there", ADR-0009). Only changed or new lines are sent, so
  the import counts read "1 updated" or "1 new".
- **Why**: no new rule in financials; SC-002 (no duplicates) follows from 015's key.

## R5. The import source

- **Decision**: `financials.SourceSimulatedGSAP = "simulatedGsap"`; migration `00021_simulated_gsap_orders.sql`
  re-creates the `financial_imports.source` check with `'simulatedGsap'`; contract `FinancialImport.source`
  enum gains `simulatedGsap`; feed name "Simulated GSAP purchase order"; file name = the PR's ID.
- **Why**: the Financials page and import history name where the data came from (FR-011, ADR-0009 "screens name
  the source").

## R6. Permission (Decision for the log, with 032-E)

- **Decision**: route rule `need(authz.FinanceImport)` (added to 032's `access/routes_purchase_requests.go`, not
  to a file 030 may own); the use case also requires that the person sees the PR (404). The seeded
  Requisitioner role holds `finance.import` (032 R9).
- **Why**: brief §8.3 and ADR-0016 name `finance.import`; the requisitioner stands in for the GSAP side.

## R7. One transaction

- **Decision**: the handler calls one use case inside one `Write`; `Orders` and the ledger join it (their
  `Write` calls join the caller's, B5). An error anywhere rolls everything back (FR-009); the counters advance
  inside the same transaction, so a failed run uses no number.

## R8. No new dependency; no new table

Everything 033 stores lives in 032's columns, 015's tables and the counters.

## R9. Implementation notes (backend, recorded while building)

- The port `purchaserequest.GSAP` takes a `RoutedPR` (lines with HX key, SWBS code, currency, amount and
  their PO line if any, plus `LastPOLine`) and answers a `GSAPOrder`; `simulatedgsap.Orders` implements it
  with `Numbers` (the readable-ID counters `GPR` and `GPO`) and a `Ledger` (`app/simulated_gsap_orders.go`
  over `financials.RecordSimulatedGSAP`). Everything runs in one `Write`, so a refused row books nothing.
- Fixed while testing: the run answered the PR as read before the GSAP fields were stored (stale
  version); `storeOrder` now answers the stored PR.
- The quickstart walkthrough ran on the 032 database and port (18132), the API of this worktree.

## Decisions for the log (proposed)

| # | Decision |
|---|---|
| 033-A | Simulated GSAP numbers: GSAP PR SIM-0010001001…, PO SIM-4500001001…, PO lines 10, 20…; status "Released"; dated the run's day; one PO per PR. |
| 033-B | A run is per PR, by a `finance.import` holder, allowed exactly while the PR is waiting for GSAP; a second run without a change is refused; after an approved change it tops up (same numbers, updated and new PO lines). |
| 033-C | PO lines are recorded through 015's applier as one import per run (source simulatedGsap, feed "Simulated GSAP purchase order"); no direct writes to transactions. |
| 033-D | Closing a PR never changes its PO. |
