# Data Model: Financial Transactions (015)

Two new stored record types in a new Go package `backend/internal/financials`. Both use `record.Meta`
(version, created and updated by and at), the JSON file store and readable IDs from `ids.Generator`. The
shape is relational: no nesting, foreign keys by ID. A later PostgreSQL repository needs no service
change (constitution IV).

| Table | ID prefix | Go type | Written by |
|---|---|---|---|
| `financial_transactions` | `FT` (`FT-1001`) | `financials.Transaction` | imports (feed), the demo seed, link and unlink |
| `financial_imports` | `FIM` (`FIM-1001`) | `financials.Import` | imports, the demo seed |

Add `PrefixFinancialTransaction = "FT"` and `PrefixFinancialImport = "FIM"` to
`internal/platform/ids/ids.go` and its `Prefixes` list.

Nothing else is stored. Every USD amount, figure, link state and roll-up is derived on read (D-041,
D-090).

---

## 1. `financial_transactions`

```go
// Type is the kind of GSAP transaction.
type Type string

const (
    TypePurchaseOrder Type = "purchaseOrder" // a PO line: committed
    TypeInvoice       Type = "invoice"       // invoiced; credit notes are negative
    TypePayment       Type = "payment"       // paid (actuals)
    TypeAccrual       Type = "accrual"       // accrued, for AccrualMonth
)

// MatchedBy says which rule linked a transaction to its line item (FR-007, FR-026).
type MatchedBy string

const (
    MatchedByLineItemID  MatchedBy = "lineItemId"
    MatchedByTrackingKey MatchedBy = "trackingKey"
    MatchedByPONumber    MatchedBy = "poNumber"
    MatchedBySwbsCode    MatchedBy = "swbsCode"
    MatchedByManual      MatchedBy = "manual"
    MatchedByNone        MatchedBy = "" // not linked
)

type Transaction struct {
    ID             string       `json:"id"`             // "FT-1001"
    Type           Type         `json:"type"`
    DocumentNumber string       `json:"documentNumber"` // GSAP document number; for POs the PO number. 1–20 chars [A-Za-z0-9/-]
    LineNumber     string       `json:"lineNumber"`     // PO line or invoice item, 1–6 digits; "" when none
    PONumber       string       `json:"poNumber"`       // POs: = DocumentNumber. Others: the PO referred to; "" when none
    POLineNumber   string       `json:"poLineNumber"`   // the PO line referred to; "" when none (always "" on POs)
    PostingDate    date.Date    `json:"postingDate"`
    AccrualMonth   string       `json:"accrualMonth"`   // accruals: "2026-09"; others ""
    Amount         money.Amount `json:"amount"`         // as GSAP sent it, in Currency; may be negative
    Currency       string       `json:"currency"`       // ISO code of a currency of feature 002
    VendorCode     string       `json:"vendorCode"`     // ≤ 20; "" when none
    VendorName     string       `json:"vendorName"`     // ≤ 120; "" when none
    Description    string       `json:"description"`    // GSAP short text, ≤ 200; "" when none
    SwbsCode       string       `json:"swbsCode"`       // ≤ 40; "" when none
    TrackingKey    string       `json:"trackingKey"`    // as sent, upper case, "HX1003"; "" when none
    SentLineItemID string       `json:"sentLineItemId"` // the line_item_id column as sent, "L-1003"; "" when none

    LineItemID      string    `json:"lineItemId"`      // the linked line item; "" when not linked
    MatchedBy       MatchedBy `json:"matchedBy"`       // "" when not linked
    NotLinkedReason string    `json:"notLinkedReason"` // "" when linked (§4.3)
    ManualLink      *ManualLink `json:"manualLink"`    // set only when MatchedBy == manual
    ImportID        string    `json:"importId"`        // the import that added or last changed the GSAP fields
    record.Meta
}

type ManualLink struct {
    By actor.Actor `json:"by"`
    At time.Time   `json:"at"` // UTC
}
```

**Invariants** (checked by the service and by `store_json_test.go`):

- The key `(Type, DocumentNumber, LineNumber)` is unique. `DocumentNumber` is compared as sent, after
  trimming spaces.
- `PONumber == DocumentNumber` and `POLineNumber == ""` for purchase orders.
- `AccrualMonth != ""` exactly when `Type == accrual`.
- `LineItemID == ""` ⇔ `MatchedBy == ""` ⇔ `NotLinkedReason != ""`.
- `ManualLink != nil` ⇔ `MatchedBy == manual`.
- Transactions are never deleted. There is no history `Kind` that moves them to the recycle bin (§7).

**Readable name** (used in the activity log, the panel title and messages):
`<type word> <DocumentNumber>[ / <LineNumber>]`. The type words are PO, Invoice, Payment and Accrual.
Examples: "PO 4500001001 / 10", "Invoice 5100002002".

**Repository** (`repository.go`):

```go
type Repository interface {
    ListTransactions(ctx context.Context) ([]Transaction, error)   // all; the service indexes in memory
    GetTransaction(ctx context.Context, id string) (Transaction, bool, error)
    PutTransaction(ctx context.Context, t Transaction) error       // insert or replace by ID
    ListImports(ctx context.Context) ([]Import, error)
    PutImport(ctx context.Context, i Import) error
    GetImport(ctx context.Context, id string) (Import, bool, error)
}
```

These are demo-sized tables. The service builds an in-memory `index` per request: by key, by line item, by
PO (number, line) and by SWBS code. PostgreSQL later adds the matching SQL indexes in its repository.

---

## 2. `financial_imports`

```go
type Source string

const (
    SourceFile Source = "file" // the simulated GSAP extract, uploaded through the API
    SourceDemo Source = "demo" // the seed
)

type Import struct {
    ID        string `json:"id"`        // "FIM-1001"
    Source    Source `json:"source"`
    FeedName  string `json:"feedName"`  // Feed.Name(): "Simulated GSAP extract", "Demo data (simulated GSAP)"
    FileName  string `json:"fileName"`  // "" for the demo
    Rows      int    `json:"rows"`      // data rows read
    New       int    `json:"new"`
    Updated   int    `json:"updated"`
    Unchanged int    `json:"unchanged"`
    Refused   int    `json:"refused"`   // Rows = New + Updated + Unchanged + Refused
    NotLinked int    `json:"notLinked"` // accepted rows of this delivery left not linked
    Relinked  int    `json:"relinked"`  // stored not-linked transactions this import linked (FR-007 retry)
    record.Meta                          // CreatedBy / CreatedAt = who imported and when
}
```

**GSAP data as of** = `CreatedAt` of the newest import (any source). With no import it is `null`, and the
screens say "No GSAP data loaded yet".

---

## 3. Incoming rows (the feed seam)

```go
// Feed turns one delivery from a source system into incoming rows (FR-003). The simulated extract
// (feed_csv.go) and the demo data (feed_demo.go) are feeds. A PGSI 941 feed replaces the CSV feed later
// without changes to the service, the handler or the screens.
type Feed interface {
    Name() string
    Source() Source
    // Read parses a delivery. A problem with the whole delivery is an *errs.Error (validation_failed on
    // "content"). Row problems are returned with their row numbers; those rows are not in the result.
    Read(ctx context.Context, d Delivery) ([]Incoming, []RowProblem, error)
}

type Delivery struct {
    FileName string
    Content  []byte // decoded file; nil for the demo feed
}

type Incoming struct {
    Row            int // 2 = the first data row, as a spreadsheet numbers it
    Type           Type
    DocumentNumber, LineNumber, PONumber, POLineNumber string
    PostingDate    date.Date
    AccrualMonth   string
    Amount         money.Amount
    Currency       string
    VendorCode, VendorName, Description, SwbsCode, TrackingKey, SentLineItemID string
}

type RowProblem struct {
    Row     int
    Column  string // CSV column name, e.g. "amount"; "" for a whole-row problem
    Message string
}
```

The service validates what a feed cannot know: whether the currency exists in Settings, and whether a key
is repeated in the delivery. It adds those problems to the feed's problems, which keeps one list and one
message style.

---

## 4. Matching (FR-007)

### 4.1 Order of application in one delivery

1. Validate every row (feed + service checks). Problem rows are **refused**.
2. Sort the accepted rows by type, in the order purchaseOrder, invoice, payment, accrual, keeping file
   order within a type.
3. Process them in that order. New rows get IDs in that order.
4. For each row, find the stored transaction with its key:
   - none: **new**;
   - the GSAP fields (all of §1 above `LineItemID`) differ: **updated** (Touch, `ImportID` = this import);
   - otherwise: **unchanged** (no write; its link is kept).
5. For a new or updated row whose `MatchedBy != manual`, run `match` (§4.2) and store the result.
6. **Retry**: run `match` on every stored transaction with `MatchedBy == ""` that this delivery did not
   touch, in the order of step 2, then by ID. Each one that becomes linked counts in `Relinked`.

The index used by `match` includes the rows already processed in this delivery, so an invoice finds a PO
of the same file.

### 4.2 `match(t, idx) (lineItemID, MatchedBy, reason)`

Try the rules in order. The first rule that gives exactly one line item wins.

| # | Rule | Applies when | Gives |
|---|---|---|---|
| 1 | Line-item ID | `SentLineItemID` matches `^L-[0-9]+$` and that line item is stored (live or in the recycle bin) | that ID, `lineItemId` |
| 2 | HX key | `TrackingKey` matches `^HX([0-9]+)$` and `L-<digits>` is stored (live or in the bin) | `L-<digits>`, `trackingKey` |
| 3 | PO number | not a PO; `PONumber != ""`; the linked PO transactions with `DocumentNumber == PONumber` (and `LineNumber == POLineNumber` when that is given) point to exactly one line item | that line item, `poNumber` |
| 4 | SWBS code | `SwbsCode != ""`; the linked transactions with that `SwbsCode` (other than `t`) point to exactly one line item | that line item, `swbsCode` |

When no rule gives a line item, the transaction is not linked. Its reason is that of the **first key it
carries**, in the order of the table:

| First key carried | Reason (`notLinkedReason`) |
|---|---|
| line-item ID | No line item L-1099 — check the line item ID in GSAP. |
| HX key | No line item for HX key HX1099 — check the tracking key in GSAP. |
| PO number, no linked PO | PO 4500001009 isn't linked to a line item yet. |
| PO number, several line items | PO 4500001003 has lines on several line items — the row needs its PO line. |
| SWBS code, none | SWBS code GB01-M26-0099 isn't on any linked transaction yet. |
| SWBS code, several | SWBS code GB01-M26-0006 is on transactions of several line items. |
| nothing | No line item reference (line item ID, HX key, PO number or SWBS code). |

"Stored" in rules 1–2 means a live line item, or one inside a recycle-bin entry that is not yet resolved
(`Planning.LineItemState`, plan §Backend). A purged line item does not count.

### 4.3 Link state (derived, `FinancialLink.state`)

| State | When |
|---|---|
| `linked` | `LineItemID` is a live line item |
| `lineItemDeleted` | `LineItemID` is set but the line item is in the recycle bin (or later purged) |
| `notLinked` | `LineItemID == ""` |

---

## 5. Figures (FR-010 – FR-013)

```go
// Facts are a line item's figures in USD, from its transactions. Line items without transactions have
// zero facts.
type Facts struct {
    Committed, Invoiced, Paid, Accrued money.Amount // USD
    Count int                                        // transactions linked
}
```

- `usd(t) = money.ToUSD(t.Amount, rates[t.Currency])`, half up to the cent, per transaction (D-052).
  Rates are the current Settings rates of every currency, active or not (`settings.Rates`).
- **Accrual month** = the greatest `AccrualMonth` among all stored accrual transactions, whatever their
  link state. Only accruals of that month add to `Accrued`. With no accruals it is `null` and Accrued is
  0.00.
- Line-item facts: the sum of `usd(t)` per type over `t.LineItemID == id`.
- Activity and container facts: the sum of the facts of their **live** line items (FR-012).
- `plannedCost`: a line item's USD planned cost (007), an activity's typed planned cost (006), a
  container's rolled-up planned cost (005/004, D-090). `null` when it has none.
- `notYetCommitted = plannedCost − committed`; `null` without a planned cost.
- `committedPercent = committed.cents × 100 ÷ plannedCost.cents`, half up to a whole number. It is `null`
  without a planned cost, or when the planned cost is 0.00. It may exceed 100.
- `committedAbovePlanned = plannedCost != null && committed > plannedCost` (exact comparison).
- `lineItemsAbovePlanned` (activities and containers): the number of live line items below with
  `committedAbovePlanned`.
- An **activity is committed above planned cost** when its own `committedAbovePlanned` is true or its
  `lineItemsAbovePlanned ≥ 1`. This drives the All work filter and the Home count.
- **Display**: each USD figure is converted once with the reader's `money.Display` (D-061, D-220). Sums are
  taken in USD before conversion, so a total may differ by a cent from the sum of its converted rows.

---

## 6. Demo data (FR-031)

Written by the seed step `seed/financials.go`, after approvals (008) and before 011's history step (D-202).
It goes through the demo feed and the same `apply` code as an import, with logging off (D-188: seed steps
write no log entries). It writes one import `FIM-1001`:
- source `demo`, feed name "Demo data (simulated GSAP)";
- 11 rows: 11 new, 0 updated, 0 unchanged, 0 refused · 1 not linked;
- created by `actor.System` at the seed time.

Vendors, document numbers and SWBS codes are fictitious.

| ID | Type | Document / line | PO ref | Posting date | Accrual month | Amount | Vendor | SWBS code | HX key | Links to | Matched by |
|---|---|---|---|---|---|---|---|---|---|---|---|
| FT-1001 | PO | 4500001001 / 10 | — | 2026-01-12 | — | 15,800.00 GBP | V-100101 Northlight Events GmbH | DE03-M26-0003 | HX1003 | L-1003 Venue hire | trackingKey |
| FT-1002 | PO | 4500001002 / 10 | — | 2026-05-04 | — | 5,000.00 USD | V-100102 Keynote Speakers Ltd | — | HX1004 | L-1004 Speaker fees | trackingKey |
| FT-1003 | PO | 4500001003 / 10 | — | 2026-01-08 | — | 9,800.00 EUR | V-100103 Outdoor Frames Ltd | GB01-M26-0006 | HX1006 | L-1006 Stand build | trackingKey |
| FT-1004 | PO | 4500001004 / 10 | — | 2026-09-15 | — | 12,000.00 USD | V-100104 Brightside Media BV | — | HX1099 | — (not linked) | — |
| FT-1005 | Invoice | 5100002001 | 4500001001 | 2026-02-20 | — | 15,800.00 GBP | V-100101 Northlight Events GmbH | — | — | L-1003 | poNumber |
| FT-1006 | Invoice | 5100002002 | 4500001002 | 2026-09-10 | — | 2,500.00 USD | V-100102 Keynote Speakers Ltd | — | — | L-1004 | poNumber |
| FT-1007 | Invoice | 5100002003 | 4500001003 | 2026-03-02 | — | 9,800.00 EUR | V-100103 Outdoor Frames Ltd | GB01-M26-0006 | — | L-1006 | poNumber |
| FT-1008 | Payment | 1500003001 | 4500001001 | 2026-03-20 | — | 15,800.00 GBP | V-100101 Northlight Events GmbH | — | — | L-1003 | poNumber |
| FT-1009 | Payment | 1500003002 | — | 2026-04-01 | — | 9,800.00 EUR | V-100103 Outdoor Frames Ltd | GB01-M26-0006 | — | L-1006 | swbsCode |
| FT-1010 | Accrual | 9100004001 | 4500001002 | 2026-08-31 | 2026-08 | 5,000.00 USD | V-100102 Keynote Speakers Ltd | — | — | L-1004 | poNumber |
| FT-1011 | Accrual | 9100004002 | 4500001002 | 2026-09-30 | 2026-09 | 2,500.00 USD | V-100102 Keynote Speakers Ltd | — | — | L-1004 | poNumber |

IDs follow the processing order of §4.1, so POs come first.

- FT-1004's reason: "No line item for HX key HX1099 — check the tracking key in GSAP."
- The descriptions are "Venue hire Berlin", "Keynote speakers", "Stand build", "Retail display
  production", and so on, as written in `feed_demo.go`.

### 6.1 Expected figures (USD display; seeded rates EUR 0.92, GBP 0.79)

`usd`: 15,800.00 GBP → 20,000.00; 9,800.00 EUR → 10,652.17 (10,652.1739…). The accrual month is 2026-09.

| Record | Planned cost | Committed | Invoiced | Paid | Accrued | Not yet committed | % | Above planned |
|---|---|---|---|---|---|---|---|---|
| L-1003 Venue hire | 20,000.00 | 20,000.00 | 20,000.00 | 20,000.00 | 0.00 | 0.00 | 100 | no |
| L-1004 Speaker fees | 5,000.00 | 5,000.00 | 2,500.00 | 0.00 | 2,500.00 | 0.00 | 100 | no |
| L-1006 Stand build | 10,000.00 | 10,652.17 | 10,652.17 | 10,652.17 | 0.00 | −652.17 | 107 | **yes** |
| A-1002 Social always-on | 80,000.00 | 25,000.00 | 22,500.00 | 20,000.00 | 2,500.00 | 55,000.00 | 31 | no |
| A-1003 Billboard winter wave | 45,000.00 | 10,652.17 | 10,652.17 | 10,652.17 | 0.00 | 34,347.83 | 24 | 1 line item |
| A-1001 Launch film | 120,000.00 | 0.00 | 0.00 | 0.00 | 0.00 | 120,000.00 | 0 | no |
| Initiative Digital & social | 200,000.00 | 25,000.00 | 22,500.00 | 20,000.00 | 2,500.00 | 175,000.00 | 13 | no |
| Initiative Print & outdoor | 75,000.00 | 10,652.17 | 10,652.17 | 10,652.17 | 0.00 | 64,347.83 | 14 | 1 line item |
| Campaign Helix launch | 275,000.00 | 35,652.17 | 33,152.17 | 30,652.17 | 2,500.00 | 239,347.83 | 13 | 1 line item |
| Campaign Rimula fleet | 185,000.00 | 0.00 | 0.00 | 0.00 | 0.00 | 185,000.00 | 0 | no |
| Plan P-1001 FY2026 Lubricants Europe | 460,000.00 | 35,652.17 | 33,152.17 | 30,652.17 | 2,500.00 | 424,347.83 | 8 | 1 line item |
| Financials page, year 2026, All | — | 47,652.17 | 33,152.17 | 30,652.17 | 2,500.00 | — | — | — |

Counts: 11 transactions; 4 POs, 3 invoices, 2 payments and 2 accruals; 1 not linked. A-1002 has 7
transactions, A-1003 has 3, L-1004 has 4. Home, Priya Raman, My teams: committed 35,652.17 of 460,000.00
(8 %), 1 activity committed above planned cost, 1 transaction not linked.

Note: the seed keeps line items L-1001 to L-1008 unchanged (D-209). Only line items of approved
activities carry transactions.

---

## 7. History kinds and activity-log entries (FR-027)

`financials` registers two **top-level** kinds (no `Parent`), so that no recycle-bin sweep ever reaches a
transaction. `Snapshot`, `Remove` and `Put` are implemented over the repository, as `history.Register`
requires, but no feature calls `MoveToBin` for them.

| Kind `Name` | Singular / plural | Entries |
|---|---|---|
| `financialImport` | "GSAP extract" / "GSAP extracts" | `ActionCreated`, verb **imported**; `RecordID` the FIM ID; `RecordName` the file name; `Details` "<rows> rows: <new> new, <updated> updated, <unchanged> unchanged, <refused> refused · <notLinked> not linked[ · <relinked> earlier transactions are now linked]". Reads "Alex Morgan imported GSAP extract 'gsap-2026-10-19.csv'". The demo load writes no entry (D-188) |
| `financialTransaction` | "GSAP transaction" / "GSAP transactions" | `ActionChanged`, verb **linked** or **unlinked**; one `Change` on field `lineItemId`, whose `Describe` gives "Line item" and "<name> (L-1007)", or "Not linked" |

Example: "Alex Morgan linked GSAP transaction 'PO 4500001004 / 10'" with "Line item: Not linked → Station
signage (L-1007)".

---

## 8. Changes to other features' data

| Feature | Change |
|---|---|
| 010 `allwork.Node` | New field `Finance *FinanceFacts` (line items: USD committed, invoiced, paid and accrued, and `AbovePlanned bool`), filled by `internal/app/allwork_source.go` from `financials.Service.FactsByLineItem`. Containers and activities sum them in `rollup.go`, as planned cost. |
| 010 `all_work_preferences`, `saved_views` | No migration. Stored column lists stay as they are. Only `DefaultColumns` changes. |
| 007 `line_items` | None. The SWBS code shown comes from the transactions (§5, FR-017). |
| 002 `currencies` | None. If a currency used by transactions is deleted later in Settings, those transactions cannot be converted. `usd(t)` is then unknown, they are left out of every figure, and they carry the flag `currencyMissing` ("Currency CHF is no longer set up in Settings — this amount is left out of the totals"). Restoring the currency brings them back. No `settings.ReferenceSource` is added: its usage counts are per planning level (D-042). |
