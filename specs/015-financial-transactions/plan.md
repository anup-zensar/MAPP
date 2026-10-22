# Implementation Plan: Financial Transactions

**Branch**: `015-financial-transactions` | **Date**: 2026-10-20 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/015-financial-transactions/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (fragment) ·
[tasks.md](tasks.md) · conventions of [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md)

## Summary

This feature closes BRD MVP gap **BUS.FIN.080**. A new Go package `internal/financials` stores GSAP
transactions (PO lines, invoices, payments, accruals) read-only. They arrive through a **`Feed`
interface**, which has two implementations now:
- the **simulated GSAP extract**: a CSV file sent to `POST /api/financials/imports`, after a dry-run
  `…/check`;
- the **demo data** loaded by the seed.

A PGSI 941 feed replaces the CSV feed later with no change elsewhere.

Each transaction is linked to a line item by line-item ID, HX key, PO number or SWBS code. Figures
(committed, invoiced, paid, accrued, not yet committed, % committed) are calculated **on read** for any
plan, campaign, initiative, activity or line item, in the reader's display currency.

The web app shows them where people work. Every one of these places is open, never collapsed:
- the activity page's second section;
- the line-item page;
- Committed and Invoiced columns in the Line items section;
- the plan page under the summary cards;
- the campaign and initiative panels;
- two default All work columns, plus a filter;
- a Home card;
- a new **Financials** page in the sidebar, with search, tabs, a details panel and the import.

The collapsed "From finance (coming later)" sections are removed.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app), as 001.

**Primary Dependencies**: none new. CSV is read with the standard `encoding/csv`, as 002's option import
(D-051). Web: `react-router`, `@tanstack/react-query`, `openapi-fetch` and `lucide-react` (icon
`ReceiptText`) from 001.

**Storage**: JSON file store, two new tables `financial_transactions` (FT) and `financial_imports` (FIM)
(data-model §1–§2). Money uses `internal/platform/money`. Dates use `internal/platform/date`.

**Testing**: Go `testing` with the fixed clock and in-memory fakes for the ports; `apitest` contract tests
on the seed; Vitest + Testing Library with `mockApi`; Playwright + axe (`frontend/e2e/financials.spec.ts`).

**Target Platform / Project Type**: as 001 (React SPA + Go REST API, 127.0.0.1 only).

**Performance Goals**: SC-004.
- One request loads all transactions once and indexes them in memory: at most 20,000 rows.
- Figures come from one pass over the transactions plus 010's hierarchy facts.
- An import is one store transaction (`Write`).

**Constraints**:
- No personal data in technical logs. Vendor names and descriptions are never logged.
- Exact money (never floats).
- The request body is at most 1 MB (`web.ReadJSON`), so files are at most 700 KB as base64, as in D-051.
- WCAG 2.2 AA.

**Scale/Scope**:
- 10 new API operations; 2 changed (`GET /api/all-work`, `GET /api/all-work/export`) and the All work
  options' column catalogue.
- 1 new page with a panel; 6 embedded sections or cards; 1 Home card.

## Constitution Check

*GATE: checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to FR-001–FR-031 (traceability table). No `[NEEDS CLARIFICATION]`; the open points are defaults with questions for Shell (spec, *Questions for Shell*) | Pass |
| II. Shell business, plain language | Shell's words (purchase order, invoice, accrual, SWBS code, HX key, GSAP); every text is in *User-visible messages*. "MAPP owns plans; GSAP owns the money": transactions are read-only and nothing is sent to GSAP | Pass, **with amendment** (below) |
| III. Simple, friendly experience | Figures are open by default and shown as words plus colour. Empty, loading and error states for each card. Search plus tabs on one page. Problems are listed all at once, with row numbers | Pass |
| IV. Simple, layered architecture | One package, handler → service → repository. Ports to other features are consumer interfaces with adapters in `internal/app` (D-196). No third runtime part: the import is an API call, not a job | Pass |
| V. Tested business rules | Unit tests for matching, figures, rounding, CSV parsing and access. Contract tests per operation. Seed-wide equality test (SC-002). Playwright for the P1 stories | Pass |
| VI. Safe by default | The actor is on every call. An access seam (`Access`) is on every operation. Imports and links are logged. Exact decimals. UTC timestamps | Pass |
| VII. Keep it small | No dependency. Matching is 4 rules. No document storage, no invoice approval, no PRs (later BRD rows) | Pass |

**Amendment needed (Complexity Tracking #1).** The constitution's *Current Phase & Scope* lists
"Finance (… transactions, … accruals)" and "GSAP integration" as out of scope. Principle II says "GSAP
integration is out of scope for the current phase". The product owner's decision of 2026-10-20 brings
read-only transactions and a **simulated** GSAP feed into the MVP.

Proposed constitution **1.1.0 (MINOR)**:
- in *Current Phase & Scope*, move "financial transactions (read-only, from a simulated GSAP feed)" into
  scope;
- keep "live GSAP integration, purchase requests, accrual calculation" out of scope;
- in II, replace the last sentence with "Live GSAP integration is out of scope for the current phase;
  GSAP data is loaded through a simulated feed."

**Post-design re-check**: Pass, with the exceptions in Complexity Tracking.

## Proposed decisions

Numbered by the consolidation step: D-242–D-257 in docs/sdd/decision-log.md. Each one is a reversible default for Shell to confirm.
It replaces or reopens the decision named.

| # | Proposed decision | Reason | Reopens |
|---|---|---|---|
| D-242 (015-A) | Read-only GSAP financial transactions are in the MVP. They are loaded through a simulated feed (CSV import endpoint + demo seed) behind `financials.Feed`; a PGSI 941 feed replaces it later. MAPP never creates, edits, deletes or sends transactions | PO decision 2026-10-20; BUS.FIN.080 (MVP); GSAP frozen until about March 2027 (FA §D1 file-exchange option) | D-008; constitution II and *Current Phase & Scope* (1.1.0) |
| D-243 (015-B) | Four types: PO line (Committed), Invoice (Invoiced; credit notes negative), Payment (Paid = actuals), Accrual (Accrued, one accrual month). No purchase requests | BRD text names these four; PRs are BUS.FIN.020 (later); workshop: actuals are payments | Field catalogue §4.6, §5 #17 |
| D-244 (015-C) | Accrued is the sum of the accruals of the **latest accrual month in the data**. MAPP calculates no accruals | Accruals are month-end positions; BUS.FIN.040 (calculation) is later | — |
| D-245 (015-D) | Matching order: line-item ID → HX key ("HX" + line-item number) → PO number (+ PO line) → SWBS code (unique) → Not linked with a reason. Rows are applied POs first; each import retries every not-linked transaction; manual links are never changed by an import | Workshop D5: HX key is the match key; D-019; makes invoices without HX keys link through their PO | — |
| D-246 (015-E) | A row's key is type + document number + line. A re-sent row updates in place. Imports never delete. Refused rows are skipped, the rest imported. Two-step check → import, with the file sent twice (stateless, as D-051 and D-136) | GSAP is the source of truth; idempotent re-imports; nothing saved without a check | — |
| D-247 (015-F) | Simulated extract format: UTF-8 CSV, header row, the columns of *Simulated GSAP extract*, at most 700 KB and 5,000 rows, base64 in JSON | API body limit 1 MB (001 R10); same pattern as 002's option import | — |
| D-248 (015-G) | Transaction amounts are stored as sent, in their currency. USD is calculated on read with the current Settings rate, half up per transaction (D-027, D-041, D-052). With dated rates (BUS.BUD.050) this becomes the posting-date rate | One rule with line items; nothing to recalculate | Decided again with the BUS.BUD.050 work |
| D-249 (015-H) | Figures are calculated on read for every level from the live tree. Transactions linked to a line item in the recycle bin stay linked and leave the figures until it is restored. Copies never copy transactions | D-041, D-090, R13 | — |
| D-250 (015-I) | Placement, all open by default:<br>- activity page: second section, "Financials";<br>- Line items section: Committed and Invoiced columns;<br>- line-item page: Financials section (the form has none);<br>- plan page: under the summary cards, by campaign;<br>- campaign and initiative panels: after Budget and cost;<br>- All work: column group "Financials", with Committed and Invoiced default, and filter "Financials";<br>- Home: card under the summary cards;<br>- sidebar: "Financials" after All work.<br>The plan tree gets no new columns | BUS.FIN.080 "prominent … main working views", "simple to access"; design-system card layout | 006 FR-020, 007 FR-007/FR-025, 010 FR-018/FR-019, D-086 (adds a section) |
| D-251 (015-J) | Every embedded Financials card reads one endpoint, `GET /api/financials/records/{recordId}`, owned by `financials`. Home's card reads `GET /api/financials/overview` from `financials`, not from `home` (an exception to D-171). The card component lives in `features/financials/` and host pages import it (as D-187) | One owner per rule; the host packages 004–009 stay untouched on the server | D-171 (exception) |
| D-252 (015-K) | Access seam `financials.Access` (view, import, link), checked on every operation. The default allows everyone (D-004). Roles and access (BUS.ADM.020) supplies the real one and the visible plan scope. A refusal answers `409 not_allowed` | BRD BUS.ADM.020 names POs and invoices; keeps today's rule | D-004 for financial data, when roles exist |
| D-253 (015-L) | Manual link and unlink are allowed only on transactions that the rules could not link, by line-item ID with a preview, logged, with `version`. Rule-based links are corrected in GSAP | The HX key is typed by hand in GSAP; MAPP must not fight GSAP data | — |
| D-254 (015-M) | New readable-ID prefixes `FT` (transactions) and `FIM` (imports). History kinds `financialImport` and `financialTransaction` are top-level (never in the recycle bin) | Module convention; transactions are GSAP's records | — |
| D-255 (015-N) | The SWBS code shown on a line item is taken from its transactions ("(from GSAP data)"). No stored SWBS code on line items yet | No SWBS response feed in this phase; avoids a 007 storage change | 007 FR-024 (display only) |
| D-256 (015-O) | The Financials page Year filter is the linked plan's year; not-linked transactions use their posting year | Planners think in plan years; invoices arrive after year end | — |
| D-257 (015-P) | A transaction whose currency was later deleted in Settings is listed but left out of the figures, flagged | Avoids a cross-package usage hook (D-042 counts per level) | — |

## Project Structure

### Documentation (this feature)

```text
specs/015-financial-transactions/
├── spec.md · plan.md · data-model.md · contracts/openapi.yaml · tasks.md
```

### Source code (repository root)

```text
backend/
├── api/openapi.yaml                         # merge the fragment (10 paths; AllWorkFinance; AllWorkColumn.group += financials)
├── internal/platform/ids/ids.go             # + PrefixFinancialTransaction "FT", PrefixFinancialImport "FIM"
├── internal/financials/                     # NEW package
│   ├── model.go                             # Transaction, Import, Type, MatchedBy, Source, API views (data-model §1–§2)
│   ├── repository.go · store_json.go (+ _test)
│   ├── feed.go                              # Feed, Delivery, Incoming, RowProblem (data-model §3)
│   ├── feed_csv.go (+ _test)                # CSVFeed: the simulated GSAP extract
│   ├── feed_demo.go (+ _test)               # DemoFeed: the 11 demo rows (data-model §6)
│   ├── template.go                          # the sample file (FR-009)
│   ├── index.go                             # in-memory indexes per request
│   ├── match.go (+ _test)                   # match(), apply order, retry (data-model §4)
│   ├── figures.go (+ _test)                 # Facts, Figures, roll-up over Planning's tree, accrual month (§5)
│   ├── ports.go                             # Planning, Rates, Displays, Access, AllowAll (below)
│   ├── service.go (+ _test)                 # Check, Import, Record, Transactions, Transaction, Link, Unlink, Overview, Imports, FactsByLineItem, LoadDemo
│   ├── texts.go                             # every message of *User-visible messages*
│   ├── kinds.go (+ _test)                   # history kinds financialImport, financialTransaction
│   └── handler.go (+ _test)                 # the 10 operations
├── internal/app/financials.go (+ _test)     # NEW: adapters Planning/Rates/Displays over 004–007, 002, 003; wiring
├── internal/app/app.go                      # wire financials; routes
├── internal/app/allwork_source.go           # Node.Finance from financials.FactsByLineItem
├── internal/app/app_test.go                 # remove the 10 operations from notYetServed when served
├── internal/allwork/{source.go,columns.go,rollup.go,query.go,options.go,export.go} (+ tests)  # columns, filter, totals, export note
└── seed/financials.go (+ seed_test.go)      # demo step after approvals, before 011's history

frontend/src/
├── app/navigation.ts · app/router.tsx       # Financials item and routes
├── lib/recordLinks.ts                       # + financialTransaction → /financials/transactions/:id
├── features/financials/                     # NEW
│   ├── FinancialsPage.tsx (+ .module.css, .test.tsx)      # tabs, search, filters, tiles, table, pager, import button, data-as-of
│   ├── TransactionsTable.tsx (+ .test.tsx)                # table; card list under 640 px
│   ├── TransactionPanel.tsx (+ .test.tsx)                 # /financials/transactions/:transactionId (SidePanel)
│   ├── LinkTransactionDialog.tsx (+ .test.tsx)            # FR-026
│   ├── ImportDialog.tsx (+ .test.tsx)                     # check → import (FR-004–FR-009)
│   ├── ImportHistoryPanel.tsx (+ .test.tsx)
│   ├── FinancialFigures.tsx (+ .module.css, .test.tsx)    # figure tiles + meter + warnings (shared by every card)
│   ├── FinancialsCard.tsx (+ .module.css, .test.tsx)      # embedded card: variant "page" | "panel"; breakdown; latest
│   ├── LineItemFinancials.tsx (+ .test.tsx)               # line-item page section with all transactions + SWBS code
│   ├── FinancialsHomeCard.tsx (+ .test.tsx)               # Home card
│   ├── financialsApi.ts                                   # queries, mutations, keys
│   ├── financialsText.ts (+ .test.ts)                     # words, plurals, messages
│   ├── financialsLinks.ts                                 # financialsHref(filters)
│   └── testData.ts
├── features/activities/ActivityPage.tsx · ActivitySections.tsx (+ tests)   # FinancialsCard 2nd; FromFinance removed
├── features/line-items/LineItemsSection.tsx · LineItemPage.tsx · LineItemForm.tsx · lineItemText.ts (+ tests); FromFinanceSection.tsx DELETED
├── features/plans/PlanPage.tsx (+ test)                    # FinancialsCard under PlanSummaryCards
├── features/campaigns/CampaignPanel.tsx · InitiativePanel.tsx (+ tests)
├── features/campaigns/DeleteRecordDialog.tsx · features/activities/DeleteActivityDialog.tsx · features/line-items/DeleteLineItemDialog.tsx · plans delete dialog  # FR-029 sentence
├── features/home/HomePage.tsx (+ test)                     # FinancialsHomeCard under SummaryCards
└── features/all-work/{FilterBar,FilterMenu,allWorkText,viewState,allWorkApi}.ts(x) (+ tests)   # Financials filter

frontend/e2e/financials.spec.ts (new)
```

## Backend design

### Ports (`internal/financials/ports.go`)

```go
// Planning reads the planning tree. internal/app implements it over plans, campaigns, activities and
// lineitems (the same service reads as allwork_source.go). Every method runs in the caller's transaction.
type Planning interface {
    // Locate returns a live plan, campaign, initiative, activity or line item by its readable ID.
    Locate(ctx context.Context, recordID string) (Node, bool, error)
    // Subtree returns the live records under recordID (itself included), each with parent, plan, name,
    // level and planned cost (USD; nil when none). Line items carry their USD planned cost.
    Subtree(ctx context.Context, recordID string) ([]Node, error)
    // Year returns the live plans of year and everything under them (for the overview and list filters).
    Year(ctx context.Context, year int) ([]Node, error)
    // LineItemState says whether a line item ID is live, in an unresolved recycle-bin entry, or unknown.
    LineItemState(ctx context.Context, id string) (LineItemState, error) // live | inBin | unknown
    // Paths returns, for line item IDs, the live path plan → … → line item (for rows and search).
    Paths(ctx context.Context, lineItemIDs []string) (map[string][]Node, error)
    // MyTeamPlanIDs returns the IDs of plans that have one of the person's teams (Home "My teams").
    MyTeamPlanIDs(ctx context.Context, personID string) ([]string, error)
}

type Node struct {
    Level       Level        // plan, campaign, initiative, activity, lineItem
    ID, ParentID, PlanID, Name string
    PlanYear    int
    PlannedCost *money.Amount // USD; activities as typed; containers rolled up (D-090); line items converted
}

type Rates interface{ Rates(ctx context.Context) (money.Rates, error) }               // settings.(*Service).Rates
type Displays interface{ Display(ctx context.Context, personID string) (money.Display, error) } // people

// Access decides who may see financial data, import and link (FR-028). AllowAll is the default until
// Roles and access; it then provides scoping through VisiblePlanIDs.
type Access interface {
    CanView(ctx context.Context, a actor.Actor) bool
    CanImport(ctx context.Context, a actor.Actor) bool
    CanLink(ctx context.Context, a actor.Actor) bool
    // VisiblePlanIDs returns nil for "every plan", otherwise the plans the person may see.
    VisiblePlanIDs(ctx context.Context, a actor.Actor) ([]string, error)
}
```

When `CanView` is false, every read operation answers `403 forbidden` (D-299) with "You don't have access to
financial data — ask an administrator." `CanImport` and `CanLink` refuse their operations the same way,
with the import or link wording. Records outside `VisiblePlanIDs` answer `403` (014 message 4), not `404`.
The adapter in `internal/app` implements `Access` over 014's `authz` (D-252, D-301); `AllowAll` stays only for
unit tests. Lists, counts and
totals then leave out transactions linked to other plans; not-linked transactions are visible only to
people with `CanImport`.

### Service operations (`service.go`)

| Method | Rule |
|---|---|
| `Check(ctx, a, Delivery) (ImportCheck, error)` | `CSVFeed.Read` + currency and duplicate-key checks + `plan(…)`, the same code as Import, inside a `Write` returning a sentinel error so nothing is saved (D-177 pattern). Returns counts, problems and not-linked rows (first 200 each) |
| `Import(ctx, a, Delivery) (Import, error)` | Same, committed. Writes the import record and one log entry (FR-027). Refuses `content` problems before any write |
| `LoadDemo(ctx) error` | `DemoFeed` + the same apply code, actor `actor.System`, logging off (seed only) |
| `Record(ctx, a, recordID) (RecordView, error)` | Figures of the record, breakdown of its children (plan → campaigns, campaign → initiatives, initiative → activities, activity → line items, line item → none), latest 5 transactions (posting date desc, then ID desc), count, SWBS codes (line items), source |
| `Transactions(ctx, a, Query) (TransactionPage, error)` | Filters (type, q, year, planId, recordId, link), counts per tab, totals for all tabs' filters except type, sort, paging. Choices: years and the plans of the year |
| `Transaction(ctx, a, id) (TransactionView, error)` | Row plus details, import and timestamps |
| `Link(ctx, a, id, lineItemID, version)` / `Unlink(ctx, a, id, version)` | FR-026 rules; version check (D-030); one log entry each |
| `Overview(ctx, a, scope) (Overview, error)` | Current year (API clock, D-091) and Home scope (`MyTeamPlanIDs`); figures over the scope's plans; the count of activities committed above planned cost; not-linked count |
| `Imports(ctx, a) (ImportPage, error)` | Newest first |
| `FactsByLineItem(ctx, ids) (map[string]Facts, error)` | For All work's source adapter. No access check (the caller checks) |

### Simulated GSAP extract (`feed_csv.go`, FR-004, Proposed 015-F)

UTF-8, with an optional byte-order mark. Comma separated, `"` quoting (`encoding/csv`). Row 1 names the
columns in any order. Names are matched ignoring case and surrounding spaces. Unknown columns are ignored
and named in the check result ("Columns not used: cost_centre"). Empty lines are skipped. Data rows are
numbered from 2.

| Column | Required | Format and rule | Problem message (`Row n · <column>: …`) |
|---|---|---|---|
| `type` | yes | `PO`, `INVOICE`, `PAYMENT`, `ACCRUAL` (any case; `PURCHASE ORDER` also accepted) | Type must be PO, INVOICE, PAYMENT or ACCRUAL. |
| `document_number` | yes | 1–20 characters `[A-Za-z0-9/-]` | Document number is missing. / Use 20 letters, digits, "-" or "/" at most. |
| `line` | no | 1–6 digits | Line must be a number of up to 6 digits. |
| `po_number` | no | as document_number; ignored on PO rows | PO number: use 20 letters, digits, "-" or "/" at most. |
| `po_line` | no | 1–6 digits; ignored without `po_number` and on PO rows | PO line must be a number of up to 6 digits. |
| `posting_date` | yes | `YYYY-MM-DD`, a real date | Posting date must be a date like 2026-03-31. |
| `accrual_month` | ACCRUAL rows | `YYYY-MM`; ignored on other types | Accrual month must be a month like 2026-09. |
| `amount` | yes | `-?[0-9]{1,15}(\.[0-9]{1,2})?` (no thousands separators) | Amount must be a number like 1234.50 (a minus sign for credits). |
| `currency` | yes | ISO code of a currency in Settings (active or not) | Currency 'XYZ' isn't set up in Settings — an administrator adds it under Currencies. |
| `vendor_code` | no | ≤ 20 | Use 20 characters or fewer. |
| `vendor_name` | no | ≤ 120 | Use 120 characters or fewer. |
| `description` | no | ≤ 200 | Use 200 characters or fewer. |
| `swbs_code` | no | ≤ 40, no spaces | SWBS code: use 40 characters or fewer, without spaces. |
| `tracking_key` | no | `HX` + 1–9 digits (any case, stored upper case) | Tracking key must look like HX1003. |
| `line_item_id` | no | `L-` + digits | Line item ID must look like L-1003. |

Service checks: a duplicate key in the file gives "Row 12 repeats PO 4500001001 / 10 from row 3 — keep
one." on the later row (column `document_number`).

Whole-file refusals: `400 validation_failed`, field `content`, one message.

| Case | Message |
|---|---|
| File name does not end in `.csv` | Choose a .csv file — GSAP extracts are imported as CSV. |
| Content is not base64 | The file couldn't be read — choose it again. |
| More than 700 KB (decoded) | The file is larger than 700 KB — split it into smaller files. |
| Not UTF-8 | The file isn't UTF-8 text — save it as "CSV UTF-8" and try again. |
| No data rows | The file has no rows. |
| More than 5,000 data rows | The file has more than 5,000 rows — split it into smaller files. |
| Required columns missing | The first row must name the columns. Missing: type, amount. |

The sample file `GET /api/financials/imports/template` is named `mapp-gsap-extract-sample.csv`. It
holds the header in table order and three rows: a PO for `HX1001`, an invoice for its PO, and an accrual
of month `2026-09`.

### API operations

All are under `RequireSignIn`. Answers use the display currency of the actor (D-061).

| Operation | Service | Errors |
|---|---|---|
| `GET /api/financials/records/{recordId}` | Record | 404 "This <level> no longer exists — someone may have deleted it." · 409 not_allowed |
| `GET /api/financials/transactions` | Transactions | 400 on a bad parameter (`FieldError.field` = parameter name, as D-219) · 409 not_allowed |
| `GET /api/financials/transactions/{transactionId}` | Transaction | 404 "This GSAP transaction doesn't exist." |
| `POST /api/financials/transactions/{transactionId}/link` | Link | 400 `lineItemId` ("No line item L-9999." / "Line item L-1005 is in the recycle bin — restore it first.") · 404 · 409 version_conflict · 409 not_allowed ("Linked by GSAP data (HX key HX1006) — correct it in GSAP." / no access) |
| `POST /api/financials/transactions/{transactionId}/unlink` | Unlink | 404 · 409 version_conflict · 409 not_allowed ("Only links made by a person can be undone here.") |
| `GET /api/financials/overview?scope=` | Overview | 400 scope |
| `GET /api/financials/imports` | Imports | — |
| `POST /api/financials/imports/check` | Check | 400 content / fileName |
| `POST /api/financials/imports` | Import | 400 content / fileName · 409 not_allowed |
| `GET /api/financials/imports/template` | template | — (text/csv, `Content-Disposition: attachment`) |

### Changes to All work (feature 010, FR-023, FR-024)

- `source.go`: `Node.Finance *FinanceFacts{Committed, Invoiced, Paid, Accrued money.Amount; AbovePlanned
  bool}`, filled for line items by `internal/app/allwork_source.go` with
  `financials.FactsByLineItem`. It is nil when `Access.CanView` is false.
- `columns.go`:
  - a new group `GroupFinancials = "financials"`, after the level groups, with the money columns
    `committed`, `invoiced`, `paid`, `accrued` and `notYetCommitted` on levels plan, campaign,
    initiative, activity and lineItem;
  - all five are sortable and added to `rolledColumns`;
  - `DefaultColumns` becomes `… "unassigned", "committed", "invoiced"`;
  - FR-018's comment changes from "Fields that Finance or GSAP fill later are not offered" to
    "Financial figures come from feature 015";
  - with no access, the group is left out of the options.
- `rollup.go`: activities sum their line items' facts. Containers sum their matching activities. Not yet
  committed = rolled planned cost − rolled committed. An activity's `AbovePlanned` follows data-model §5.
- `query.go`: a new filter `finance` (multi). Its values are `hasPurchaseOrder` (committed ≠ 0 or at
  least one PO; counted from the facts), `noPurchaseOrder`, `invoiced` (invoiced ≠ 0) and
  `committedAbovePlanned`. It is tested on activities (D-113: lower levels hold by descendant).
  - To tell "has a PO" from "committed 0.00 after a credit", `FinanceFacts` carries `POCount` and
    `InvoiceCount`, so "has a PO" means `POCount ≥ 1`.
- `export.go`: when a financial column is in the view, add the note "Financial figures: GSAP data as of
  <DD MMM YYYY HH:mm UTC>" (from `financials.DataAsOf`).
- Web: the filter "Financials" in `FilterMenu` with the four values; `viewState` parses `finance` from
  the address; and `allWorkText` gets labels "Has a PO", "No PO yet", "Invoiced", "Committed above planned
  cost".

### Wiring (`internal/app`)

- `financials.go`:
  - `planningAdapter` over plans, campaigns, activities and lineitems (reuse the readers of
    `allwork_source.go`; activities' planned cost; 005's `rollup` for containers; 007's USD planned cost);
  - `history.SnapshotsInBin` for `LineItemState`;
  - people for `MyTeamPlanIDs` and `Display`;
  - `settings.Rates`;
  - `financials.AllowAll{}` for Access.
- `app.go`: `financials.NewService(store, repo, ids, clock, planning, rates, displays, access,
  history)`; `RegisterKinds`; routes; the All work source gets `financials`.
- Register the history kinds after 008's kinds (they are top level, so their order is free).

### Demo data (`seed/financials.go`)

`seed.Load` calls `financials.LoadDemo` after approvals (008) and before 011's history step (D-202). The
11 rows and the expected figures are in data-model §6. `seed_test.go` adds `TestSeed_FinancialsStory_015`,
which asserts every figure of data-model §6.1 through `Service.Record` and `Overview`.

## Frontend design

### Route and navigation

- `navigation.ts`: `FINANCIALS = { label: 'Financials', path: '/financials', icon: ReceiptText,
  alsoCurrentOn: [] }`, inserted in `MAIN_ITEMS` after `ALL_WORK`. The header comment names 015.
- `router.tsx`: `/financials` → `FinancialsPage`, with the child route `transactions/:transactionId` →
  `TransactionPanel` (an outlet panel, like 005's panels).
- Address parameters: `type`, `q`, `year`, `planId`, `recordId`, `link`, `sort`, `page` and `pageSize`.
  The page reads them and writes them back on every change (`replace: true` for typing, push for tabs). The
  address is the state. `financialsHref(filters)` builds links for the cards and Home.

### `FinancialFigures` (shared by every card)

- Tiles: Committed (POs), Invoiced, Paid, Accrued (<Mon YYYY>), Not yet committed.
- Committed has a `ProgressBar` "<n> % of planned cost committed". The bar is red above 100 %; at "—" the
  bar is hidden and "No planned cost yet" is shown.
- Each tile has an ⓘ (`InfoTip`) with the text of *User-visible messages*.
- Negative Not yet committed shows "Committed above planned cost by X" in red with a warning icon.
  `lineItemsAbovePlanned ≥ 1` shows "Committed above planned cost on n line items", linking to the first.
- Money is formatted with `useFormatters().money` and its currency label (D-143).
- Narrow screens: the tiles wrap 2 per row, 1 per row under 400 px.

### `FinancialsCard` (record pages and panels)

Props: `recordId: string`, `variant: 'page' | 'panel'`, `id?: string` (anchor `financials`).

It fetches `GET /api/financials/records/{recordId}` with the query key `['financials', 'record',
recordId, displayCurrency]` and shows:
- the title "Financials" with the ⓘ "Purchase orders, invoices, payments and accruals from GSAP for this
  <level>'s line items.";
- `FinancialFigures`;
- the breakdown table, headed "By campaign", "By initiative", "By activity" or "By line item". Rows are
  name links (`recordHref`) with the four figures and the red marker;
- for activities: Latest transactions (at most 5: type chip, number, date, vendor, amount);
- the footer "View all n transactions in Financials" → `financialsHref({recordId})`, and "GSAP data as of
  <date time> · <source>".

States:
- loading: `Skeleton`;
- error: `ErrorState` inside the card only;
- empty: the FR-014 sentence;
- no data at all: "No GSAP data loaded yet".

Hosts:

| Host (owner) | Place | Variant |
|---|---|---|
| `ActivityPage` (006) | Right after `ActivityDetails`; `FromFinance` and its test removed | page, `id="financials"` |
| `PlanPage` (004) | Right after `PlanSummaryCards`, before the plan tree | page |
| `CampaignPanel`, `InitiativePanel` (005) | After the Budget and cost section | panel (breakdown: name + Committed + Invoiced only; table scrolls inside the panel) |

### Line items (007)

- `LineItemsSection`: it reads the activity's record view (the same query key as the card, so there is one
  request) and adds the columns Committed and Invoiced after the USD planned cost. A line item over its
  planned cost shows a red marker with an accessible name. While loading, the cells show "…"; on an error
  they show "—" with the title "Financial figures couldn't load".
- `LineItemPage`: `LineItemFinancials` replaces `FromFinanceSection`. It holds `FinancialFigures`, a
  **Transactions** table (`GET /api/financials/transactions?recordId=L-…&pageSize=50`, with a pager), and
  "SWBS code" from the record view's `swbsCodes`: codes joined by ", " + " (from GSAP data)", or "Filled by
  GSAP later".
- `LineItemForm`: remove `FromFinanceSection`. Delete `FromFinanceSection.tsx` and the
  `SECTION_FROM_FINANCE`, `FROM_FINANCE_FIELDS` and `FROM_FINANCE_NOTE` texts, and update their tests
  (create, edit, page).

### Financials page (`FinancialsPage`)

- `PageHeader`: the title "Financials", the subtitle "Purchase orders, invoices, payments and accruals from
  GSAP" and the actions **Import GSAP extract** (secondary) and **Import history** (link button).
- A source line under the header: "GSAP data as of 19 Oct 2026, 02:00 · Demo data (simulated GSAP)" or
  "No GSAP data loaded yet".
- `PillTabs`: All · Purchase orders · Invoices · Payments · Accruals · Not linked, each with its count.
- Filter row:
  - `SearchField` "Search number, vendor, SWBS code, HX key, line item or activity";
  - Year (`ChoiceField`, years from `choices.years`);
  - Plan (`ChoiceField`, `choices.plans`, "All plans");
  - the record filter as a `FilterChips` chip with a remove button;
  - **Clear all**.
- Four `SummaryCard`s: Committed, Invoiced, Paid, Accrued (month). They follow the filters, ignoring the tab.
- `TransactionsTable`:
  - columns: Type (chip) · Number · Posting date · Vendor · Amount (own currency, then the display
    currency in muted text when different) · Linked to (line item link + activity link, or the chip
    "Not linked" / "Line item in the recycle bin") · SWBS code;
  - sortable headings (`aria-sort`); the row opens the panel; the pager offers 50 / 100 / 200;
  - under 640 px it becomes one card per row.
- Empty results: "No transactions match — clear a filter or search for something else", with **Clear
  all**.

### Transaction panel (`TransactionPanel`)

- A `SidePanel` titled with the readable name (for example "PO 4500001003 / 10") and the type chip.
- Definition list: document number, line, PO number / PO line, posting date, accrual month, amount (own
  currency), USD amount and display amount, vendor (code · name), description, SWBS code, HX key, line-item
  ID as sent.
- **Linked to**: a breadcrumb of links, plan › campaign › initiative › activity › line item.
- **Matched by**: the method words of *User-visible messages*, or the not-linked reason.
- **From import**: "<feed> '<file>' on <date time> by <person>".
- Added and last updated.
- Actions:
  - **Link to a line item** (not linked; `CanLink`);
  - **Unlink** (manual only), with a `ConfirmDialog` "Unlink 'PO 4500001004 / 10' from 'Station signage
    (L-1007)'? It goes back to Not linked.";
  - otherwise the note "Linked by GSAP data (<method>) — correct it in GSAP."

### Import dialog (`ImportDialog`, FormDialog)

1. **Choose a file**: the format summary (required columns), **Download sample file**, and a file input
   (`accept=".csv"`).
2. **Check**: the file is read as base64 in the browser and sent to `/check`, with a spinner "Checking
   <file>…". Whole-file errors show at the input.
3. **Result**: the counts sentence, "Problems (n)" as a list "Row 3 · amount: …", and "Not linked (n)" as
   "Row 7 · PO 4500001004 / 10: No line item for HX key HX1099 — check the tracking key in GSAP". Buttons:
   **Import n rows** (primary; disabled with a reason when every row is refused: "Nothing to import — fix
   the rows and choose the file again") and **Choose another file**.
4. **Done**: the dialog closes; the toast says "Imported 'gsap.csv': 2 new, 1 updated · 1 not linked".
   The queries `['financials']`, `['all-work']` and `['home']` are invalidated.

### Link dialog (`LinkTransactionDialog`)

- A text field "Line item ID" (`L-1007`).
- On a valid pattern, `GET /api/line-items/{id}` is called for a preview: "<name> (L-1007) · <activity> ·
  <plan>". On 404: "No line item L-1007."
- **Link** posts with the version. The API's 400 and 409 messages show at the field or as
  `conflictMessage`.

### Home (009)

`FinancialsHomeCard` sits right after `SummaryCards` in `HomePage`. It reads `GET
/api/financials/overview?scope=<scope>`, with the key `['financials', 'overview', scope, currency]`. It has
its own `SectionError`. Content follows FR-025. Its links are built with `financialsHref` and
`/all-work?finance=committedAbovePlanned&year=<year>`.

### Delete dialogs (FR-029)

The delete dialogs of plans, campaigns and initiatives, activities and line items read
`transactionCount` from the same record view. A non-zero count adds the sentence of *User-visible
messages*. Line items read it per line item; containers read it from their own record view. This is only a
read: deleting is unchanged.

### Query keys and invalidation

| Key | Invalidated by |
|---|---|
| `['financials', 'record', id, currency]` | import, link, unlink; and any planning mutation already invalidating that record's page queries (add `['financials']` to the invalidations of 006/007 move, delete and restore helpers) |
| `['financials', 'transactions', params]` | import, link, unlink |
| `['financials', 'transaction', id]` | link, unlink |
| `['financials', 'overview', scope, currency]` | import, link, unlink |
| `['financials', 'imports']` | import |

## User-visible messages

| Where | Text |
|---|---|
| Card title ⓘ | Purchase orders, invoices, payments and accruals from GSAP for this <level>'s line items. |
| Committed ⓘ | The value of purchase orders GSAP has issued for these line items. |
| Invoiced ⓘ | Supplier invoices GSAP has received against them, less credit notes. |
| Paid ⓘ | Payments GSAP has released — Shell's actuals. |
| Accrued ⓘ | Money owed for work received but not yet invoiced, as GSAP accrued it for <Mon YYYY>. |
| Not yet committed ⓘ | Planned cost minus committed. |
| Meter | <n> % of planned cost committed |
| No planned cost | No planned cost yet |
| Over (record) | Committed above planned cost by <amount> |
| Over (below) | Committed above planned cost on <n> line item(s) |
| Empty card | No purchase orders or invoices yet — they appear here once GSAP sends them for this <level>'s line items. |
| No data | No GSAP data loaded yet. |
| Source line | GSAP data as of <DD MMM YYYY, HH:mm> · <feed name>[ '<file>'] |
| View all | View all <n> transactions in Financials |
| Matched by | Line item ID L-1003 · HX key HX1003 · PO number 4500001001 · SWBS code GB01-M26-0006 · Linked by <person> on <date> |
| Rule-linked note | Linked by GSAP data (<method>) — correct it in GSAP. |
| Not linked chip / deleted chip | Not linked · Line item in the recycle bin |
| Deleted line item | Line item <id> is in the recycle bin — its amounts come back when it's restored. |
| Currency missing | Currency <code> is no longer set up in Settings — this amount is left out of the totals. |
| Import counts | <rows> rows: <new> new, <updated> updated, <unchanged> unchanged, <refused> refused · <notLinked> not linked |
| Relinked | <n> earlier transactions are now linked |
| Refused after import | <n> rows refused — fix them and import the file again. |
| Delete dialog | <n> GSAP transactions are linked to it. They stay in MAPP and leave the figures until it is restored. |
| No access | You don't have access to financial data — ask an administrator. |
| Home empty | No purchase orders or invoices for your plans yet. |
| Home links | <n> activities committed above planned cost · <n> transactions not linked to a line item · Open Financials |
| Log (import) | <person> imported GSAP extract '<file>' — details: <counts> |
| Log (link) | <person> linked / unlinked GSAP transaction '<name>' (Line item: <old> → <new>) |

Every other row, file and link message is in the tables of *Simulated GSAP extract*, *API operations* and
data-model §4.2.

## Dependencies

- **Uses**:
  - 001: actor, history, ids, store, errors, `SidePanel`, `ConfirmDialog`, `Toast`;
  - 002: `settings.Rates`, currencies, `money`;
  - 003: display currency, teams;
  - 004–007: the planning readers, `rollup`, `recordHref`, the line-item GET for the link preview;
  - 009: `SectionError`, `HomeScope`;
  - 010: the All work source, columns and filters;
  - 011: activity-log display of the new kinds (labels come from the kinds; no 011 code change).
- **Provides**: `financials.FactsByLineItem` and `DataAsOf` (010); the `Access` seam (Roles and access);
  `Feed` (a future GSAP integration feature).
- **Works with 011 merged or not.** Log entries are written either way (001's log), and 011 shows them
  when merged.

## Tests

### Go (`internal/financials`, fixed clock 2026-10-20 09:00 UTC, fakes for every port)

- `feed_csv_test.go`: each column rule and message (table-driven); BOM; quoting; unknown columns; case of
  headers; every whole-file refusal; row numbering from 2.
- `match_test.go`: each rule in order and its precedence; every not-linked reason; POs first within one
  delivery; retry relinks; a manual link is kept; a line item in the bin still matches; a purged one does
  not; ambiguous PO and SWBS.
- `figures_test.go`: each figure; credit notes; latest accrual month; rounding half up per transaction
  (9,800.00 EUR → 10,652.17); % half up (12.5 → 13); `null` cases; above-planned for line items and
  activities; bin exclusion; a missing currency excluded and flagged; display conversion once.
- `service_test.go`: Check saves nothing (store unchanged); Import counts (SC-003) and idempotence;
  refused rows skipped; one log entry per import with the details; Link and Unlink rules, version
  conflict and log; access refusals for each operation (fake `Access`); `VisiblePlanIDs` scoping;
  SC-006 with a second fake `Feed`.
- `handler_test.go` (`apitest`, seed): every operation and error against the contract.
- `seed_test.go`: `TestSeed_FinancialsStory_015`, every value of data-model §6.1; SC-002, the record view
  equals the parent's breakdown row, the All work row, the overview and the transactions page totals with
  `recordId`.
- `internal/allwork`: the financial column values and roll-ups, totals, the four filter values, default
  columns, the export note, and no group without access.
- Stress (SC-004): 5,000-row import under 5 s; a 20,000-transaction list under 1 s (`testing.B`, run in
  CI with `-run` off by default, as 010's stress test).

### Web (Vitest + Testing Library, `mockApi`)

- `FinancialFigures`: values, meter, red states with text, `null` states, ⓘ texts.
- `FinancialsCard`: each variant and breakdown heading; empty, no-data, error and loading states; links.
- `FinancialsPage`: tabs and counts; search and filters written to the address; record chip; sort; pager;
  phone layout; empty result.
- `TransactionPanel`: fields; linked path; matched-by texts; actions shown only when allowed; unlink
  confirmation.
- `ImportDialog`: steps; problems list; not-linked list; disabled import with reason; invalidation;
  toast.
- `LinkTransactionDialog`: preview; 404; API refusals.
- Host tests: `ActivityPage` (card is the 2nd section, open, no "From finance"); `LineItemsSection`
  (columns); `LineItemPage` (section, SWBS code); `LineItemCreatePage`/`LineItemEditPage` (no finance
  section); `PlanPage`; `CampaignPanel`; `InitiativePanel`; `HomePage` (card under summary cards);
  navigation (sidebar order); All work `FilterMenu`/`viewState` (`finance`); delete dialogs (sentence).

### End to end (`frontend/e2e/financials.spec.ts`, Playwright + axe)

1. US1: Priya Raman opens Social always-on; the Financials card is visible without clicks and shows the
   data-model §6.1 figures; then Speaker fees' transactions.
2. US2: the plan page section and the Helix launch panel totals.
3. US3: sidebar → Financials; search 4500001003; panel; link to Stand build; the Not linked tab.
4. US4: import a generated 3-row file (one new PO for HX1007, one invoice for its PO, one bad currency);
   check counts; import; the figure on Grand opening Shanghai changes.
5. axe on the Financials page, the panel, the import dialog and the activity page.

## Requirement traceability

| FR | Backend | Frontend | Tests |
|---|---|---|---|
| FR-001, FR-002 | model.go, store_json.go | TransactionPanel | store_json_test, handler_test |
| FR-003 | feed.go, feed_csv.go, feed_demo.go | — | service_test (SC-006) |
| FR-004–FR-006, FR-009 | feed_csv.go, service Check/Import, template.go | ImportDialog | feed_csv_test, service_test, e2e 4 |
| FR-007 | match.go | TransactionPanel (matched by) | match_test |
| FR-008 | service Imports, DataAsOf | FinancialsPage source line, ImportHistoryPanel | handler_test |
| FR-010–FR-014 | figures.go | FinancialFigures | figures_test, seed_test |
| FR-015 | Record | FinancialsCard in ActivityPage | ActivityPage.test, e2e 1 |
| FR-016 | Record (breakdown) | LineItemsSection | LineItemsSection.test |
| FR-017 | Record (swbsCodes), Transactions | LineItemFinancials; LineItemForm | LineItemPage.test, LineItemCreatePage.test |
| FR-018, FR-019 | Record | PlanPage, CampaignPanel, InitiativePanel | host tests, e2e 2 |
| FR-020–FR-022 | Transactions, Transaction | navigation, FinancialsPage, TransactionPanel | FinancialsPage.test, e2e 3 |
| FR-023, FR-024 | allwork columns/rollup/query/export | All work filter | allwork tests |
| FR-025 | Overview | FinancialsHomeCard | service_test, HomePage.test |
| FR-026 | Link, Unlink | LinkTransactionDialog, TransactionPanel | service_test, dialog tests |
| FR-027 | kinds.go, Import/Link/Unlink | — | kinds_test, service_test |
| FR-028 | ports.go Access, every handler | no-access states | service_test (fake Access) |
| FR-029 | Record (transactionCount) | delete dialogs | dialog tests |
| FR-030 | (nothing copied: plans.Level copies never touch financials) | — | service_test: copy plan, figures of the copy 0.00 |
| FR-031 | feed_demo.go, seed/financials.go | — | seed_test |

## Complexity Tracking

| # | Exception | Why needed | Simpler alternative rejected because |
|---|---|---|---|
| 1 | Constitution *Current Phase & Scope* and II list transactions and GSAP integration as out of scope | Product owner decision 2026-10-20 (BRD MVP BUS.FIN.080) | Keeping them out leaves an MVP requirement "Not met"; the feed is simulated, so the GSAP freeze is respected. Amendment 1.1.0 proposed |
| 2 | Home's card is served by `financials`, not `home` (D-171) | One owner for every financial rule; `home` stays untouched | Duplicating figure logic in `home` would let Home and the cards disagree (SC-002) |
| 3 | `Check` runs the real import inside a discarded `Write` | Preview equals result (SC-003), as D-177 and D-126 | A separate "predict" path would drift from the real rules |

## Consolidation (2026-10-20, binding)

The consolidation of 014–017 settled the points below. They win where this plan, its data model, its tasks or its
fragment say otherwise (as D-200). Build order across the four features:
[docs/reports/brd-gap-closure-plan.md](../../docs/reports/brd-gap-closure-plan.md).

- **Decisions.** 015-A–015-P are D-242–D-257 in [decision-log.md](../../docs/sdd/decision-log.md); the
  consolidation decisions are D-298–D-309.
- **Contract (D-298, D-299, D-300).** The fragment is merged into `backend/api/openapi.yaml`; T001 is done and the
  10 operations are in `notYetServed`. `FinancialLevel` is gone: `FinancialRecordRef.type` and the breakdown use
  002's `AttributeLevel` (the same five values). Read and import operations no longer list 409; every operation
  lists `403 Forbidden`.
- **Access (D-252, D-299, D-301).** Refusals are **403 `forbidden`**, never 409 `not_allowed`; a record outside the
  person's access is 403 (014 message 4), not 404. `financials.Access` is implemented in
  `internal/app/financials.go` over 014's `authz`: `CanView` = holds `finance.view` in some scope (per-plan checks
  in `Record` and the lists), `CanImport` = holds `finance.import`, `CanLink` = `finance.link` in the target line
  item's plan, `VisiblePlanIDs` = plans the person can see **and** holds `finance.view` in. `AllowAll{}` stays only
  for unit tests (T043). The route rules are in the Table C additions of the 014 plan's Consolidation section. API
  refusal texts are 014's messages 1–4; "You don't have access to financial data — ask an administrator." is the
  web app's in-place note. Default roles (014 Table B): everyone sees financial data; Administrator, Global planner
  and Planner link; only Administrator imports. The spec's FR-028 and Assumptions now say so.
- **Money (D-303).** Transactions convert at the **current** rate (`settings.Rates`), as every other amount;
  posting-date rates are a later option, so the `Rates` port and data-model §6.1's figures stay. `Node.PlannedCost`
  stays USD: once 016 lane C is built, the planning adapter converts the plan-currency planned cost to USD at the
  current rate (016's `app/currency.go` helpers).
- **Placement (D-305).** Plan page: summary cards → **Financials** → 016's Budget by month → 016's Planned cost by →
  plan tree → History. Campaign and initiative panels: … Budget and cost → 016's Budget by month → **Financials** →
  History. Activity page: header → Details → **Financials** → Line items → Approval → History. The activity
  page's card, the Line items columns and the line-item page's section render only when 014's
  `Activity.access.seeFinance` / `LineItemAccess` allow it (an absent `access`, before 014 is built, means shown).
  Sidebar: … All work · **Financials** (`NavItem.permission = 'finance.view'`).
- **All work.** The Financials group, the `finance` filter and their export columns are offered when the person
  holds `finance.view` in any scope; rows of plans without it show "—" there and never match the filter.
- **Home.** The overview counts only plans the person can see and holds `finance.view` in (014 FR-007).
- **Log visibility (D-304).** `financialImport` and `financialTransaction` entries need `finance.import` or
  `activityLog.viewAll`.
- **Demo data (D-306).** The financials step runs after approvals (with 017's additions) and before 011's history
  step; FT-1001–FT-1011 link to L-1003, L-1004 and L-1006 only, so 016's L-1009 and L-1010 are untouched.
- **Build order (D-307).** The backend foundation (T002–T011) may run beside 014's foundation and rebases on it;
  T010's `Access` adapter is written over `authz` once that is merged. The web foundation (T012–T014) waits for
  the UI foundation's shared components. US5 (All work) waits for 016 lane A's changes to `internal/allwork`,
  then rebases on them. Everything else can run beside 016.
- **Constitution (D-308).** The amendment that brings read-only transactions from a simulated GSAP feed into scope
  is written once with 014's and 016's, as the first task of the build; T048's documentation keeps its other items.
