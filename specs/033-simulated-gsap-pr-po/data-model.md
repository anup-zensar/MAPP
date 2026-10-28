# Data Model: GSAP PR Number and Purchase Order from a Simulated GSAP (033)

033 adds no table. It fills 032's GSAP columns, adds feature 015 transactions and imports, and uses two
counters.

## 1. Storage

| Where | Change | Migration |
|---|---|---|
| `purchase_requests.gsap_pr_number`, `gsap_pr_date`, `gsap_source` (032) | set at the first run: `SIM-0010001001`, the run's date, `simulatedGsap`; never changed afterwards | — |
| `purchase_request_lines.po_number`, `po_line`, `po_line_status`, `po_date`, `po_value_minor` (032) | set per line at each run that creates or updates its PO line | — |
| `financial_imports` (015) | one row per run: source `simulatedGsap`, feed name "Simulated GSAP purchase order", file name the PR ID, counts | `00021_simulated_gsap_orders.sql`: re-create the `source` check as `source IN ('file', 'demo', 'simulatedGsap')` |
| `financial_transactions` (015) | one row per PO line (new) or updated (top-up), type `purchaseOrder`, key (type, PO number, PO line) | — |
| `counters` (`platform/ids`) | prefixes `GPR` (GSAP PR numbers) and `GPO` (PO numbers) | — (rows created on first use) |

`platform/ids`: `PrefixSimulatedGSAPPR = "GPR"`, `PrefixSimulatedPO = "GPO"`, appended to `Prefixes`.

## 2. Types

### 2.1 Declared by 032 (`purchaserequest/ports.go`, brief §8.3) — used as is

`GSAP` (`CreatePurchaseOrder(ctx, RoutedPR) (GSAPOrder, error)`), `RoutedPR`, `RoutedLine`, `GSAPOrder`,
`OrderLine` (032 data-model §2.3). 033 fills `RoutedPR.RequestedBy`, `GSAPPRNumber`, `PONumber` and each
`RoutedLine.POLine` from what is stored, and sends only the lines to create or update.

### 2.2 `internal/purchaserequest/gsap_order.go` (added by 033)

```go
// CreateGSAPOrder asks GSAP (the simulated GSAP now) for the GSAP PR and the purchase order of an approved,
// routed purchase request waiting for GSAP, and stores the answer (033 FR-001 – FR-010).
func (s *Service) CreateGSAPOrder(ctx context.Context, a actor.Actor, id string, version int) (View, error)
```

Steps, inside one `Write`: `GetInScope` (404) → `authz.Require(ctx, authz.FinanceImport)` (403) → Purchase
Support on → version (409 `version_conflict`) → status Approved, no change pending, routed, waiting for GSAP
(409 `not_allowed` with the reason) → `RoutedPR` of the lines to order (PO line empty, or amount/currency ≠ PO
value) with their facts (HX key, SWBS code, cost centre, GL code) → `gsap.CreatePurchaseOrder` → apply: header
GSAP fields only when empty; each answered line's PO fields → `Put` → log entry (kind `purchaseRequest`,
action `gsapOrder`, details "Simulated GSAP created GSAP PR SIM-0010001001 and PO SIM-4500001001 (1 line)" or
"Simulated GSAP updated PO SIM-4500001001: 1 line changed, 1 line added (top-up)").

`access.go` gains the availability `gsapOrder` (hidden without `finance.import`; otherwise the first failing
reason of the steps above). `View.Access.GSAPOrder` in the contract (`PurchaseRequestAccess.gsapOrder`); each `LineView` gains
`GSAPPending` (`PurchaseRequestLine.gsapPending`), the per-line part of 032's waiting-for-GSAP rule.

### 2.3 `internal/simulatedgsap/` (added by 033)

```go
// Orders is the simulated GSAP's purchasing side (ADR-0016): it answers a routed purchase request with a GSAP
// PR number and a purchase order, and books the PO lines as GSAP transactions.
type Orders struct {
	numbers Numbers     // *ids.Generator
	ledger  Ledger
	now     clock.Clock
}

func NewOrders(numbers Numbers, ledger Ledger, now clock.Clock) *Orders
func (o *Orders) CreatePurchaseOrder(ctx context.Context, pr purchaserequest.RoutedPR) (purchaserequest.GSAPOrder, error)

// Numbers gives the next number of a counter (platform/ids.Generator.NextNumber).
type Numbers interface {
	NextNumber(ctx context.Context, prefix string) (int, error)
}

// Ledger books purchase-order lines as GSAP transactions (financials, feature 015).
type Ledger interface {
	BookPurchaseOrder(ctx context.Context, a actor.Actor, order BookedOrder) error
}

type BookedOrder struct {
	PurchaseRequestID     string // the import's file name
	PONumber              string
	VendorCode, VendorName, Description string
	Lines                 []BookedLine
}

type BookedLine struct {
	POLine, Currency, SwbsCode, HXKey string
	Amount                            money.Amount
	PostingDate                       date.Date
}

// PurchaseRequests is the use case the handler calls (*purchaserequest.Service).
type PurchaseRequests interface {
	CreateGSAPOrder(ctx context.Context, a actor.Actor, id string, version int) (purchaserequest.View, error)
}
```

Number formats (`numbers.go`): `gsapPRNumber(n) = fmt.Sprintf("SIM-%010d", 10_000_000+n)`,
`poNumber(n) = fmt.Sprintf("SIM-%010d", 4_500_000_000+n)`, PO line = `strconv.Itoa(10*k)`; status constant
`"Released"`.

### 2.4 `internal/financials` (added by 033)

```go
const SourceSimulatedGSAP Source = "simulatedGsap"

// SimulatedDelivery is GSAP data the simulated GSAP produced (ADR-0016), applied like an import.
type SimulatedDelivery struct {
	FeedName string   // "Simulated GSAP purchase order"
	FileName string   // the purchase request's ID
	Rows     []Fields // purchase-order lines
}

func (s *Service) RecordSimulatedGSAP(ctx context.Context, a actor.Actor, d SimulatedDelivery) (Import, error)
```

`rowsFeed` (`feed_rows.go`) implements `Feed` over the rows (Source `simulatedGsap`, no row problems); the
service checks `Access.Import`, runs the applier inside the caller's `Write`, stores the import and writes one
`financialImport` log entry, as `Import` does. Unlike a file import, any row the applier refuses (for example a
currency no longer set up) is an error for the whole delivery, so nothing is booked and the caller's `Write`
rolls back (FR-009). Mapping `BookedLine` → `Fields` (in the `internal/app`
adapter): `Type purchaseOrder`, `DocumentNumber = PONumber = order.PONumber`, `LineNumber = POLine`,
`PostingDate`, `Amount`, `Currency`, `VendorCode`, `VendorName`, `Description` (short text, cut to 015's
limit), `SwbsCode`, `TrackingKey = HXKey`, `SentLineItemID = ""`.

## 3. Rules (each with a test, plan "Rules and their tests")

1. **Run allowed** (FR-001, FR-006, FR-010): all of — Purchase Support on; Approved; no change pending; routed;
   some line without a PO line or with amount/currency ≠ PO value. The reason of the first that fails.
2. **Numbers** (FR-002, FR-003): GSAP PR number and date once; PO number once; PO lines 10 × k in line order,
   new lines after the highest; never reused.
3. **Answer** (FR-003, FR-004): each sent line gets status Released, the run's date and value = amount in its
   currency; lines not sent keep their PO fields.
4. **Booking** (FR-005, FR-007): one booked line per sent line; same key on top-up → 015 updates the
   transaction; Committed follows from 015's figures (linked by tracking key).
5. **All or nothing** (FR-009): one `Write`.

## 4. Web app (`frontend/src/features/purchase-requests/`)

- `useCreateGsapOrder(id)` in `api.ts`: POST with the PR's version; on success it invalidates the PR, the
  list, the line item and activity sections of its lines and every financials query (`['financials']` root
  key of feature 015).
- `GsapOrderDialog` state: open/closed, pending; the text lists the lines whose `gsapPending` is true (the server's flag: no PO line
  yet, or amount or currency differ from the PO value) with their amounts as sent, never a sum worked out in
  the browser, and says create or top-up from whether the PR already has a PO number.
- `FinancialImport.source` gains `simulatedGsap` → label "Simulated GSAP" in `features/financials`.
