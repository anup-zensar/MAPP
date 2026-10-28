# Implementation Plan: GSAP PR Number and Purchase Order from a Simulated GSAP

**Branch**: none (no Git branches or commits, product owner 2026-10-27) | **Date**: 2026-10-27 |
**Spec**: [spec.md](spec.md)

**Input**: `specs/033-simulated-gsap-pr-po/spec.md`; brief §8.3 (the `GSAP` port, `RoutedPR`, `GSAPOrder`
and the PR's GSAP columns that 032 provides); ADR-0016; feature 015's applier.

## Summary

One operation, `POST /api/v1/simulated-gsap/purchase-requests/{purchaseRequestId}/purchase-order`
(`finance.import`), runs the simulated GSAP for one approved, routed PR that is waiting for GSAP. The use
case `purchaserequest.(*Service).CreateGSAPOrder` checks the PR, sends the lines to order through 032's `GSAP`
port and stores the answer; the port's implementation `simulatedgsap.Orders` numbers the GSAP PR
(SIM-0010001001…), the PO (SIM-4500001001…) and the PO lines (10, 20…), and books the PO lines through
`financials.(*Service).RecordSimulatedGSAP` — feature 015's applier with an in-memory feed — so Committed, Not
yet committed and the Financials page follow, linked by the HX key. A second run is refused unless an approved
change made a line differ (top-up: same numbers, updated and new PO lines). The web app adds the action, its
confirmation and the "Simulated GSAP" labels.

## Technical Context

**Language/Version**: Go (pinned); TypeScript 5.9, React 19.
**Primary Dependencies**: as 032; no new dependency.
**Storage**: PostgreSQL; migration `00021_simulated_gsap_orders.sql` (one check re-created); counters `GPR`,
`GPO`.
**Testing**: Go unit tests with fakes (`Numbers`, `Ledger`, `GSAP`, 032's store fake); an `apitest` integration
test through the whole API in memory; contract test; Vitest + MSW + axe; Playwright e2e on ports 18133/14133,
database `mapp_e2e_033`.
**Target Platform / Project Type**: as 032.
**Performance Goals**: a run is one `Write` of a few statements per PR line (≤ 200 lines per PR).
**Constraints**: constitution 4.4.0 (as 032).
**Scale/Scope**: 1 operation, 3 contract schema changes, 1 migration, ~6 Go files, ~3 web files.

## Constitution Check

| Principle | How 033 meets it |
|---|---|
| I | Every behaviour traces to spec FR-001–FR-012; test names carry them (`TestCreateGSAPOrder/033 FR-006 …`). |
| II | Constitution 4.4.0 allows the simulated GSAP to create GSAP PRs and POs behind adapters (ADR-0016); everything it makes is labelled "Simulated GSAP" and marked "SIM-". |
| III | research.md, data-model.md, contracts/, this plan; ADR-0016 already records the decision; no new ADR. |
| IV | Rules in the services (032's use case decides what may be ordered; 015 decides how transactions link and count); the adapter only answers. Live GSAP replaces `simulatedgsap.Orders` behind the same port. |
| V | No new table; numbering through the existing counters; one feed type added to 015. |
| VI | Unit tests for every rule (below), a test per acceptance scenario, a contract test, an e2e of the whole P1 journey; no new scoped store read (the use case reuses 032's `GetInScope`, already isolation-tested). |
| VII | `need(finance.import)` in the route table; 404 for PRs the person cannot see; the version guards stale actions; the request body is one integer. |
| VIII–X | As 032; the action is a secondary button; no new colours. |

### Threat model (STRIDE)

| Threat | Where | Response |
|---|---|---|
| Spoofing | the operation | session actor; `finance.import` from the actor's grants only |
| Tampering: forging GSAP numbers or PO values | request | the body carries only `version`; numbers come from counters, values from the stored PR lines |
| Tampering: duplicate commitments by repeated runs | run | refused when nothing changed; top-up uses the same transaction keys, so 015 updates rather than adds (SC-002) |
| Repudiation | run | one PR log entry and one import record with who and when, in the same `Write` |
| Information disclosure | run | out-of-scope PR → 404; the answer is the PR the person already sees |
| Denial of service | run | ≤ 200 lines per PR (032); one `Write` |
| Elevation of privilege | run | Planner and Approver roles lack `finance.import`; the action is hidden for them |

### Risk model

| Risk | Response | Owner |
|---|---|---|
| Users take simulated numbers for real GSAP numbers | "SIM-" prefix, "Simulated GSAP" labels, import named "Simulated GSAP purchase order" | Product owner |
| 030 created `internal/simulatedgsap` with other conventions | 033 adds only `purchase_order*.go`; if the package does not exist, 033 creates it with these files | Team |
| Demo line items already carry demo POs (015 seed) | The e2e journey raises its PR on a new line item, so Committed starts at 0.00 | Team |

## Project Structure

```text
specs/033-simulated-gsap-pr-po/  spec.md · plan.md · research.md · data-model.md · quickstart.md · tasks.md
                                 contracts/openapi.yaml · contracts/routes.md · checklists/requirements.md
backend/
├── api/openapi.yaml                                         # fragment merged (rules 1–4)
├── internal/purchaserequest/gsap_order.go (+ _test.go)       # use case; access.go gsapOrder; views gsapPending
├── internal/simulatedgsap/purchase_order.go, purchase_order_numbers.go, purchase_order_ports.go,
│   purchase_order_handler.go (+ _test.go files)              # Orders (GSAP port), handler
├── internal/financials/feed_rows.go, simulated.go (+ tests)  # SourceSimulatedGSAP, RecordSimulatedGSAP
├── internal/app/simulated_gsap_orders.go (+ _test.go)        # Ledger adapter, UseGSAP, routes
├── internal/access/routes_purchase_requests.go              # + the rule
├── internal/platform/ids/ids.go                             # + GPR, GPO
└── internal/platform/database/migrations/00021_simulated_gsap_orders.sql
frontend/
├── src/features/purchase-requests/GsapOrderDialog.tsx, GsapOrderButton.tsx (+ tests); api.ts; GsapFields.tsx
├── src/features/financials/ (source label)
└── e2e/pr-to-po.spec.ts
```

## Backend design

- **Use case** (`gsap_order.go`): data-model §2.2; the `GSAP` port set by `UseGSAP` in `internal/app`; without
  it `access.gsapOrder` says "GSAP is not connected." and the operation answers 409.
- **Orders** (`simulatedgsap`): numbers what is missing (`Numbers.NextNumber(GPR|GPO)`), answers each sent line
  with `Released`, today, value = amount; builds one `BookedOrder` and calls `Ledger.BookPurchaseOrder`; uses
  the actor in `RoutedPR.RequestedBy`.
- **Ledger adapter** (`app/simulated_gsap_orders.go`): maps `BookedLine` → `financials.Fields` (data-model
  §2.4) and calls `RecordSimulatedGSAP`.
- **Financials** (`simulated.go`): `RecordSimulatedGSAP` = `Access.Import` + `apply` with `rowsFeed` + import
  record + log entry, inside the caller's `Write`.
- **Handler** (`purchase_order_handler.go`): binds `SimulatedGsapOrderRequest`, calls
  `PurchaseRequests.CreateGSAPOrder`, answers the PR.
- **Wiring**: `prs.UseGSAP(simulatedgsap.NewOrders(idsGenerator, ledgerAdapter, clock))`;
  `simulatedgsap.NewPurchaseOrderHandler(prs).RegisterRoutes(r)`.

### Messages

| Key | Text |
|---|---|
| notRoutedApproved | The simulated GSAP processes approved purchase requests that were routed to the requisitioner. |
| changePending | Wait until the change of this purchase request is approved or declined. |
| alreadyOrdered | The simulated GSAP already created the PO for this purchase request. Run it again after an approved change of its amounts or lines. |
| notConnected | GSAP is not connected. |
| logCreated | Simulated GSAP created GSAP PR %s and PO %s (%s). |
| logTopUp | Simulated GSAP updated PO %s: %d changed, %d added (top-up). |
| feedName | Simulated GSAP purchase order |
| sourceLabel | Simulated GSAP |

### Rules and their tests

| Rule | FR | Test |
|---|---|---|
| Run allowed, with the first failing reason | FR-001, FR-006, FR-010 | `purchaserequest`: `TestCreateGSAPOrder_Gate` (table: Created, Submitted, In review, Declined, Closed, Approved not routed, change pending, already ordered, switched off, not connected, version conflict, 403 without finance.import, 404 out of scope) and `TestAccess_GSAPOrder` |
| Only lines to order are sent; answer stored; header set once | FR-002–FR-004, FR-007 | `TestCreateGSAPOrder_StoresAnswer` (first run; top-up keeps header; unchanged line untouched) with a fake `GSAP` |
| All or nothing | FR-009 | `TestCreateGSAPOrder_GSAPErrorChangesNothing` |
| Log entry | FR-008 | `TestCreateGSAPOrder_LogEntry` |
| Numbering and answer | FR-002, FR-003 | `simulatedgsap`: `TestOrders_Numbers` (SIM-0010001001, SIM-4500001001, lines 10/20, existing kept, new after the highest), `TestOrders_Answer` (Released, today, value), `TestOrders_Booking` (one booked line per sent line, HX key, SWBS code, vendor, short text) with fake `Numbers` and `Ledger` |
| Booking through 015 | FR-005, FR-007, FR-011 | `financials`: `TestRecordSimulatedGSAP` (new rows linked by tracking key; same key with a new amount updates; import source simulatedGsap and feed name; finance.import required) |
| Committed follows, no duplicates | FR-005, FR-007, SC-001, SC-002 | `app`: `TestSimulatedGSAPOrder_EndToEnd` through `apitest` (approve and route a PR, run, read the PR, the line item's financial record and the transactions; second run refused; approved change then top-up; still one transaction per PO line) |
| Contract | — | `simulatedgsap`: `TestCreateSimulatedGsapPurchaseOrder_Contract` (200, 403, 404, 409 envelopes) |

### JavaScript budget

Inside the existing `purchase-requests` route chunk (+ ≤ 3 kB gzip); the initial route is unchanged.

### Shared files touched

`backend/api/openapi.yaml`; `backend/internal/access/routes_purchase_requests.go`; `backend/internal/app/*`
(new file, wiring, `app_test.go` `notYetServed`); `backend/internal/platform/ids/ids.go`;
`backend/internal/financials` (new files, `model.go` Source constant); `docs/product/glossary.md` and
`domain-model.md` (GSAP fields of the PR now filled by the simulated GSAP), `demo-users-and-data.md`
(the requisitioner's run); `frontend/src/api/generated/*`; `frontend/src/features/financials` (label).

## Complexity Tracking

None.
