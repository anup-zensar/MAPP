# Feature Specification: Financial Transactions

**Feature Branch**: `015-financial-transactions`

**Created**: 2026-10-20

**Status**: Draft

**Input**: Product owner decision of 2026-10-20: close BRD MVP gap **BUS.FIN.080**. Purchase orders,
invoices, payments and accruals must be prominent in the main working views and simple to reach.
Transactions are read-only and come from GSAP. GSAP is simulated for now by an import endpoint and demo
data, behind an adapter, so that a real feed can replace it later.

**BRD traceability**: **BUS.FIN.080** (MVP), the subject of this feature. It also touches **BUS.ADM.020**
(MVP: who may see POs and invoices; this feature adds the seam and Roles and access fills it) and
**BUS.BUD.050** (MVP: conversion of transactions in local currencies). It prepares **BUS.FIN.010**
(GSAP integration, technical track), **BUS.FIN.040** (accruals), **BUS.FIN.050** (invoice review) and
**BUS.FIN.060** (PO and invoice documents), which stay later. See
[brd-requirements.md](../../docs/reference/brd-requirements.md) and
[brd-gap-analysis.md §3.5](../../docs/reports/brd-gap-analysis.md).

**References**: Domain model [§1 hierarchy, §4.2 cost flows up, §4.3 conversion, §4.6 values from finance](../../docs/product/domain-model.md) ·
Field catalogue [§4.6, §5 #16–17](../../docs/product/field-catalogue.md) ·
[GSAP workshop summary](../../docs/reference/gsap-workshop-summary.md) ·
Hive9 functional analysis §C3 (transactions), §C5 (accruals), §D3–§D8 (PGSI 941, HX key, freeze) ·
Specs 004 (plan page), 005 (campaign and initiative panels), 006 (activity page FR-020), 007 (line items
FR-001, FR-007, FR-024–FR-026), 009 (Home), 010 (All work FR-004, FR-018, FR-019, FR-032), 011 (activity
log) · Decisions D-008 (reopened), D-019, D-027, D-041, D-061, D-090, D-202 ·
Constitution II ("NorthStar owns plans; GSAP owns the money").

## User Scenarios & Testing *(mandatory)*

All examples use the demo data of [data-model.md §6](data-model.md) and the seeded rates EUR 0.92 and
GBP 0.79 per USD. Amounts are shown in USD, the default display currency.

### User Story 1 - See an activity's and a line item's money where I work (Priority: P1)

A marketer opens an activity. Directly under its Details, an open **Financials** card shows how much of the
planned cost is committed by purchase orders, invoiced, paid and accrued. It also lists the figures line
item by line item and the latest transactions. The line-item page shows the same for one line item, with
all its transactions. The Line items section shows Committed and Invoiced on every row.

**Why this priority**: This is the heart of BUS.FIN.080. The money that GSAP reports must sit in the
activity and line-item views marketers use every day, not behind collapsed sections.

**Independent Test**: Sign in as Priya Raman and open "Social always-on" (A-1002). Without clicking
anything, check the five figures and the per-line-item rows. Open "Speaker fees" (L-1004) and check its
transactions.

**Acceptance Scenarios**:

1. **Given** "Social always-on" (planned cost 80,000.00 USD) with the demo transactions, **When** the user
   opens the activity, **Then** the Financials card is the second section (after Details), open, and
   shows:
   - Committed 25,000.00 USD with the meter "31 % of planned cost committed";
   - Invoiced 22,500.00 USD; Paid 20,000.00 USD; Accrued (Sep 2026) 2,500.00 USD;
   - Not yet committed 55,000.00 USD;
   - "GSAP data as of <time of the last load> · Demo data (simulated GSAP)".
2. **Given** the same activity, **Then** the card's **By line item** table shows "Venue hire" (committed,
   invoiced and paid 20,000.00; accrued 0.00) and "Speaker fees" (committed 5,000.00, invoiced 2,500.00,
   paid 0.00, accrued 2,500.00). Its **Latest transactions** list shows at most 5 rows, newest first,
   with **View all 7 transactions in Financials**.
3. **Given** "Billboard winter wave", **Then** the line "Stand build" shows Committed 10,652.17 USD
   against its planned cost of 10,000.00 USD. It shows in red with the words "Committed above planned
   cost by 652.17 USD", and the activity's card repeats the warning with a link to that line item.
4. **Given** "Launch film", which has no transactions, **Then** the card says "No purchase orders or
   invoices yet — they appear here once GSAP sends them for this activity's line items". The figures are
   0.00, and Not yet committed equals the planned cost.
5. **Given** the activity page, **Then** the Line items section shows **Committed** and **Invoiced**
   columns on every row, and the "From finance (coming later)" section no longer exists.
6. **Given** "Speaker fees" (L-1004), **When** the user opens its page, **Then** an open Financials
   section shows its four figures, its **Transactions** table with the PO, the two accruals and the
   invoice, and "SWBS code: Filled by GSAP later" (none of its transactions carries one). "Venue hire"
   (L-1003) shows "SWBS code: DE03-M26-0003 (from GSAP data)".
7. **Given** the line-item form (new or edit), **Then** it has no finance section. Values from GSAP are
   never typed.

---

### User Story 2 - Money rolls up to initiative, campaign and plan (Priority: P1)

Planners and budget owners see the same figures added up on the plan page, under the summary cards, with
one row per campaign. They see them again in the campaign and initiative panels, with one row per
initiative or activity.

**Why this priority**: Budget owners work at plan and campaign level. "Prominent in the main working
views" applies there too.

**Independent Test**: Open "FY2026 Lubricants Europe" (P-1001) and the "Helix launch" panel. Check that the
totals equal the sum of the activity cards, to the cent.

**Acceptance Scenarios**:

1. **Given** P-1001, **When** the user opens the plan, **Then** a **Financials** section directly below
   the summary cards shows:
   - Planned cost 460,000.00, Committed 35,652.17 ("8 % of planned cost committed"), Invoiced 33,152.17,
     Paid 30,652.17, Accrued (Sep 2026) 2,500.00 and Not yet committed 424,347.83;
   - a **By campaign** table with "Helix launch" (committed 35,652.17) and "Rimula fleet" (all 0.00).
2. **Given** the "Helix launch" campaign panel, **Then** an open Financials section shows committed
   35,652.17 of 275,000.00 (13 %) and a **By initiative** table: "Digital & social" 25,000.00 and
   "Print & outdoor" 10,652.17.
3. **Given** the "Print & outdoor" initiative panel, **Then** its **By activity** table lists
   "Billboard winter wave" (10,652.17) and "Trade magazine ads" (0.00). "Billboard winter wave" carries
   the red marker "Committed above planned cost on 1 line item".
4. **Given** a reader whose display currency is EUR, **Then** every figure is shown in EUR, converted once
   from the USD total, and labelled "EUR".
5. **Given** any Financials card or section, **When** the user chooses **View all n transactions**,
   **Then** the Financials page opens filtered to that record.

---

### User Story 3 - Find POs, invoices, payments and accruals in one place (Priority: P1)

A **Financials** entry in the sidebar opens one list of every transaction GSAP has sent. It has tabs by
type, search, filters, totals and a details panel that links back to the plan, activity and line item.

**Why this priority**: BUS.FIN.080 asks that transactions be "simple to access" and not "buried several
levels deep". This page makes them one click away from anywhere.

**Independent Test**: From Home, choose **Financials** in the sidebar. Search "4500001003", open the PO and
follow its link to "Stand build".

**Acceptance Scenarios**:

1. **Given** the demo data and year 2026, **When** the user opens **Financials**, **Then**:
   - the tabs read "All 11 · Purchase orders 4 · Invoices 3 · Payments 2 · Accruals 2 · Not linked 1";
   - summary tiles show Committed 47,652.17, Invoiced 33,152.17, Paid 30,652.17 and Accrued (Sep 2026)
     2,500.00 for the current filters;
   - the table lists the transactions newest first, 50 per page.
2. **Given** the search "keynote", **Then** only the transactions of the vendor "Keynote Speakers Ltd"
   are listed, and the tiles and tab counts follow the search.
3. **Given** the row "PO 4500001003 / 10", **When** the user selects it, **Then** a details panel opens at
   its own address and shows every field. It shows "Linked to: FY2026 Lubricants Europe › Helix launch ›
   Print & outdoor › Billboard winter wave › Stand build (L-1006)", each a link, and "Matched by HX key
   HX1006".
4. **Given** the row "PO 4500001004 / 10", **Then** it shows the chip "Not linked" and the reason "No line
   item for HX key HX1099 — check the tracking key in GSAP".
5. **Given** the address `/financials?recordId=A-1002`, **Then** a filter chip "Activity: Social always-on
   (A-1002)" is shown and only its 7 transactions are listed. Removing the chip shows everything.
6. **Given** a phone-width screen, **Then** the table becomes one card per transaction with type, number,
   date, amount and link, and nothing scrolls sideways outside the table.

---

### User Story 4 - Load GSAP data (simulated GSAP extract) (Priority: P2)

Until GSAP can be connected (frozen until about March 2027), a person loads a GSAP extract file in the
format NorthStar documents. NorthStar first checks the file and says what it will do. It then adds or
updates the transactions and links each one to a line item.

**Why this priority**: This is the simulated integration the product owner asked for. Demo data covers US1
to US3, so they do not wait for this story.

**Independent Test**: Download the sample file, change one amount, and import it. Check the preview
counts, the result and the changed figure on the activity page.

**Acceptance Scenarios**:

1. **Given** the Financials page, **When** the user chooses **Import GSAP extract**, **Then** a dialog
   explains the file format, offers **Download sample file** and accepts one `.csv` file of at most
   700 KB.
2. **Given** a file with 4 rows, **When** it is chosen, **Then** NorthStar checks it without saving and
   shows "4 rows: 2 new, 1 updated, 1 unchanged, 0 refused · 1 not linked". It lists each not-linked row
   with its reason, and offers **Import 4 rows** and **Cancel**.
3. **Given** a file where row 3 has the amount "12.345" and row 5 the currency "XYZ", **Then** the check
   lists both problems by row and column at once:
   - "Row 3 · amount: Amount must be a number like 1234.50 (a minus sign for credits)";
   - "Row 5 · currency: Currency 'XYZ' isn't set up in Settings".
   **Import** then imports only the other rows and says "2 rows refused — fix them and import the file
   again".
4. **Given** a row with the same type, document number and line as a stored transaction, **Then** it
   updates that transaction (counted "updated"), or is counted "unchanged" when nothing differs. Importing
   the same file twice changes nothing the second time.
5. **Given** an invoice that names only PO 4500001002, **Then** it is linked to the line item of that PO
   ("Matched by PO number").
6. **Given** a transaction that was not linked, **When** a later import links its PO, **Then** the earlier
   transaction is linked in the same import, because every import retries every not-linked transaction.
7. **Given** a successful import, **Then** the Financials page shows "GSAP data as of <now>". The import is
   listed under **Import history** with its file name, who, when and its counts. The activity log gains
   one entry: "<person> imported GSAP extract 'gsap-2026-10-19.csv' — 4 rows: 2 new, 1 updated, 1
   unchanged, 0 refused · 1 not linked".
8. **Given** an import that links 2 transactions that were not linked before (scenario 6), **Then** the
   result adds "2 earlier transactions are now linked".

---

### User Story 5 - Financial columns and a filter in All work (Priority: P2)

All work offers a **Financials** column group: Committed, Invoiced, Paid, Accrued and Not yet committed.
Committed and Invoiced are shown by default. A **Financials** filter offers "Has a PO", "No PO yet",
"Invoiced" and "Committed above planned cost".

**Why this priority**: All work is the cross-plan working view. Finance figures there make it possible to
compare activities across plans, and to export them.

**Independent Test**: Open All work for 2026. Check that the Committed column totals 35,652.17 for
FY2026 Lubricants Europe. Filter "Committed above planned cost" and export.

**Acceptance Scenarios**:

1. **Given** All work with default columns, **Then** Committed and Invoiced appear after Unassigned planned
   cost. Every level shows its figure, and the totals row equals the Financials page's linked total for the
   same plans.
2. **Given** the filter Financials = "Committed above planned cost", **Then** only "Billboard winter wave"
   matches, with its parents as context rows.
3. **Given** an export with financial columns shown, **Then** the file holds those columns. Its notes say
   "Financial figures: GSAP data as of <time>".

---

### User Story 6 - A Financials card on Home (Priority: P2)

Home shows a **Financials** card below the summary cards for the current year and the chosen scope (My teams
or Everything). It shows committed, invoiced, paid and accrued against planned cost, and links to what
needs attention.

**Why this priority**: Home is the first working view. One glance tells a budget owner where money stands.

**Independent Test**: Sign in as Priya Raman (scope My teams). Check the card's figures and follow its two
links.

**Acceptance Scenarios**:

1. **Given** Priya Raman with scope My teams, **Then** the card "Financials · 2026" shows Committed
   35,652.17 of 460,000.00 planned (8 %), Invoiced 33,152.17, Paid 30,652.17 and Accrued (Sep 2026)
   2,500.00.
2. **Given** the same card, **Then** "1 activity committed above planned cost" opens All work filtered to
   it. "1 transaction not linked to a line item" opens Financials on the Not linked tab.
3. **Given** a person whose teams have no plans with transactions, **Then** the card says "No purchase
   orders or invoices for your plans yet" with **Open Financials**.

---

### User Story 7 - Link a transaction that GSAP could not match (Priority: P3)

When a transaction cannot be matched (for example, the HX key was typed wrongly in GSAP), a person links it
to the right line item by entering the line item's ID. They can undo that link later. Links that GSAP data
decides cannot be changed in NorthStar.

**Why this priority**: The HX key is typed by hand in GSAP (workshop). A few rows will always need a person.
The import's "Not linked" list makes the problem visible even without this story.

**Independent Test**: Link "PO 4500001004 / 10" to L-1007 and check that it counts in "Grand opening
Shanghai". Then unlink it.

**Acceptance Scenarios**:

1. **Given** the not-linked "PO 4500001004 / 10", **When** the user chooses **Link to a line item** and
   types "L-1007", **Then** the dialog previews "Station signage (L-1007) · Grand opening Shanghai · FY2026
   Mobility China". On **Link**, the transaction shows "Matched by: linked by Alex Morgan on 20 Oct 2026"
   and counts in that activity's figures.
2. **Given** a manually linked transaction, **When** the user chooses **Unlink**, **Then** after confirming
   it is Not linked again with its import reason.
3. **Given** a transaction linked by its HX key, PO number, SWBS code or line-item ID, **Then** Link and
   Unlink are not offered. The panel says "Linked by GSAP data (HX key HX1006) — correct it in GSAP".
4. **Given** "L-9999", or a line item in the recycle bin, **Then** the dialog says "No line item L-9999"
   or "Line item L-1005 is in the recycle bin — restore it first", and nothing is saved.

---

### Edge Cases

- **Credit notes and reversals.** These come as negative amounts and reduce the figure of their type.
  A figure may become negative; it is shown with a minus sign, never hidden.
- **Several accrual months.** Only the accruals of the **latest accrual month** in the data count as
  "Accrued". The label names the month, for example "Accrued (Sep 2026)". Older accruals stay listed on the
  Financials page.
- **No accruals at all.** "Accrued" shows 0.00 with the label "Accrued" and no month.
- **Line item deleted.** Its transactions stay linked to it but leave every figure. The Financials page
  marks them "Line item in the recycle bin". Restoring the line item, or its parent, brings them back
  (FR-012).
- **Activity moved** to another initiative. Its figures move with it, because roll-ups are calculated on
  read.
- **Plan copied.** Transactions are never copied (R13). The copy's figures are 0.00.
- **Planned cost missing.** For an activity without a planned cost, the meter, the % and Not yet committed
  show "—" with "No planned cost yet". Committed and the other figures still show.
- **Planned cost 0.00 with a PO.** % shows "—" and the red "Committed above planned cost by X" shows.
- **Inactive currency.** A transaction in an inactive currency is still converted at its last rate. A
  currency that is not in Settings refuses the row at import.
- **Exchange-rate change.** The USD values of transactions follow the current rate at once (D-027), like
  line items. Nothing is sent back for approval.
- **Currency deleted in Settings after import.** The transaction stays listed with its own amount, but is
  left out of every figure. It is flagged "Currency CHF is no longer set up in Settings — this amount is
  left out of the totals" until the currency is restored.
- **Same key twice in one file.** The later row is refused: "Row 12 repeats PO 4500001001 / 10 from row 3 —
  keep one".
- **A PO with lines on several line items.** An invoice that names only the PO number, without a PO line,
  is not linked, with the reason "PO 4500001003 has lines on several line items — the row needs its PO
  line".
- **Two people import at the same time.** Imports are applied one after the other. Each one's preview may
  differ from its result; the result is what counts and is what the dialog shows.
- **Large data.** The Financials page pages its rows (50, 100 or 200 per page). Cards show at most 5 latest
  transactions.
- **No GSAP data loaded yet** (an empty store, not the demo). Every Financials card says "No GSAP data
  loaded yet". People who may import see **Import GSAP extract**.

## Requirements *(mandatory)*

### Functional Requirements

Each requirement traces to **BUS.FIN.080** unless another ID is given.

**GSAP data (simulated now, real feed later)**

- **FR-001**: NorthStar MUST hold four types of financial transaction:
  - **Purchase order line** (committed);
  - **Invoice** (invoiced; credit notes negative);
  - **Payment** (paid, Shell's "actuals");
  - **Accrual** (accrued, for one accrual month).

  Each transaction has: document number, optional line, PO number and PO line it refers to, posting date,
  accrual month (accruals only), amount with currency, vendor code and name, short description, SWBS code,
  HX tracking key, line-item ID, a readable ID (FT-…), and where it came from (which import).
- **FR-002**: Transactions MUST be read-only in NorthStar. They are never created by hand, edited or
  deleted. The only thing a person may change is a manual link (FR-026). NorthStar never sends anything to
  GSAP (constitution II; BUS.FIN.010 stays later).
- **FR-003**: Transactions MUST arrive through a **GSAP feed** that is separate from the screens and the
  rules. There are two feeds now:
  - (a) the **simulated GSAP extract**: a CSV file imported through the API (FR-004);
  - (b) the **demo data** loaded by the seed (FR-031).

  A real GSAP feed (PGSI 941 files) MUST be able to replace (a) without changing the screens, the
  contract or the figures.
- **FR-004**: **Import GSAP extract** MUST work in two steps:
  - **check**: validate the file, save nothing, and report:
    - the outcome counts new / updated / unchanged / refused, one per row;
    - how many accepted rows will be **not linked**;
    - every row problem, and every not-linked row with its reason;
  - **import**: apply the valid rows and report the same counts, plus how many earlier transactions it
    linked (FR-007).

  The file format is fixed and documented (plan, *Simulated GSAP extract*): a UTF-8 `.csv` file with a
  header row, at most 700 KB and 5,000 rows. A file that breaks these limits, has no rows, or lacks a
  required column is refused as a whole, with one message.
- **FR-005**: Each row MUST be validated as a whole, and every problem of the file reported at once, with
  its row number, column and plain-language message. Refused rows are skipped; the other rows are
  imported.
- **FR-006**: A row MUST be identified by **type + document number + line**:
  - a row with a new key adds a transaction;
  - a row with a known key updates that transaction ("updated"), or changes nothing ("unchanged");
  - imports never delete transactions;
  - two rows with the same key in one file: the later row is refused.
- **FR-007**: Each transaction MUST be linked to one line item by the first rule that applies, in this
  order:
  1. its **line-item ID**;
  2. its **HX key** ("HX" + the line item's number, D-019: HX1003 → L-1003);
  3. its **PO number** (and PO line, when given): the line item of the PO line it refers to;
  4. its **SWBS code**: the one line item whose linked transactions carry that code.

  Otherwise the transaction is **Not linked**, with the reason of the first key it carried. Rows are
  applied POs first, then invoices, payments and accruals, so references inside one file resolve. Every
  import then retries every stored not-linked transaction. A manual link (FR-026) is never changed by an
  import.
- **FR-008**: The Financials page MUST show **GSAP data as of <time of the latest import>** with its source
  ("Demo data (simulated GSAP)" or "Simulated GSAP extract '<file>'"). **Import history** MUST list every
  import, newest first, with file name, source, who, when and counts. Every Financials card repeats "GSAP
  data as of …".
- **FR-009**: The import dialog MUST offer **Download sample file**: the header row and three example rows
  (a PO, an invoice and an accrual).
- **FR-010**: Amounts MUST be kept exactly as GSAP sent them, in their own currency (any currency set up in
  Settings, active or not). They are converted to USD with the currency's current rate, half up to the
  cent, per transaction, on every read (D-027, D-041; BUS.BUD.050). They are never stored in USD.

**Financial figures**

- **FR-011**: For every line item, activity, initiative, campaign and plan, NorthStar MUST calculate:

  | Figure | Rule |
  |---|---|
  | Committed | Sum of its purchase-order lines |
  | Invoiced | Sum of its invoices (credit notes reduce it) |
  | Paid | Sum of its payments |
  | Accrued | Sum of its accruals of the latest accrual month in the data |
  | Not yet committed | Planned cost − Committed |
  | % committed | Committed × 100 ÷ planned cost, whole per cent, half up |

  - Every figure is the sum of the transactions' USD values (FR-010) of the record's live line items. It
    is converted once to the reader's display currency (D-061) and labelled with it.
  - Planned cost is the record's own planned cost: typed (activity), USD (line item), or rolled up
    (D-090).
  - % committed and Not yet committed are "—" without a planned cost; % committed is "—" when the
    planned cost is 0.00.
- **FR-012**: Figures MUST be calculated when read and never stored. Roll-ups follow the live tree
  (line item → activity → initiative → campaign → plan): moving an activity moves its figures. Records in
  the recycle bin, and transactions linked to them, are left out until they are restored.
- **FR-013**: Any record whose Committed is above its planned cost MUST show "Committed above planned cost
  by X" in red, with words, not colour alone. An activity or container with line items above their own
  planned cost MUST also show "Committed above planned cost on n line items", which links to the first
  one. An activity is **committed above planned cost** (FR-024, FR-025) when the activity or one of its
  line items is.
- **FR-014**: A record without transactions MUST show the figures as 0.00 and the sentence "No purchase
  orders or invoices yet — they appear here once GSAP sends them for this <level>'s line items".

**Where the figures appear**

- **FR-015** (changes 006 FR-020): The activity page MUST show an open **Financials** card as its second
  section, after Details. The card holds:
  - the figures of FR-011, with a meter for % committed;
  - the warning of FR-013;
  - a **By line item** table: name link and the four figures;
  - **Latest transactions**: at most 5, newest first;
  - **View all n transactions in Financials**;
  - "GSAP data as of …".

  The collapsed "From finance (coming later)" section MUST be removed.
- **FR-016** (changes 007 FR-001): The Line items section on the activity page MUST add the columns
  **Committed** and **Invoiced**, in the display currency, after the USD planned cost.
- **FR-017** (changes 007 FR-007, FR-024, FR-025, FR-026): The line-item page MUST replace "From finance
  (coming later)" with an open **Financials** section. The section holds:
  - the figures of FR-011 and FR-013;
  - a **Transactions** table of all the line item's transactions, paged by 50;
  - **SWBS code**: the codes on its transactions, labelled "(from GSAP data)", or "Filled by GSAP later"
    when there are none.

  The line-item create and edit form MUST NOT show any finance section.
- **FR-018** (changes 004): The plan page MUST show a **Financials** section directly below the summary
  cards: the figures of FR-011 and FR-013, a **By campaign** table and **View all n transactions**.
- **FR-019** (changes 005): The campaign and initiative panels MUST show an open **Financials** section
  after Budget and cost. It holds the figures, a **By initiative** (campaign) or **By activity**
  (initiative) table, and **View all n transactions**.
- **FR-020** (changes 001 navigation): The sidebar's main group MUST include **Financials** after All work.
  It opens `/financials`.
- **FR-021**: The Financials page MUST offer the following.
  - **Tabs**: All · Purchase orders · Invoices · Payments · Accruals · Not linked, each with its count for
    the other filters.
  - **Search** (document number, PO number, vendor code or name, SWBS code, HX key, description, linked
    line item or activity name or ID).
  - **Year**, the linked plan's year (not-linked transactions by posting year; default the current year),
    and **Plan**.
  - A **record filter** chip from the cards' links.
  - **Summary tiles**: Committed, Invoiced, Paid and Accrued for the current filters.
  - A **table** with type chip, number (with line), posting date, vendor, amount (own currency and display
    currency), linked line item and activity (links) or a "Not linked" chip, and SWBS code. It is sorted
    newest first by default, can be sorted by any heading, and is paged 50 / 100 / 200.

  Filters, tab, search, sort and page MUST be kept in the address, so that a link reproduces the view.
- **FR-022**: Selecting a transaction MUST open its details panel at
  `/financials/transactions/<id>`. The panel shows every field of FR-001, its USD and display amounts, the
  full linked path with a link per level, how it was matched (or the not-linked reason), the import it
  came from, and when it was added and last updated.
- **FR-023** (changes 010 FR-018, FR-019, FR-032): All work MUST offer a **Financials** column group:
  Committed, Invoiced, Paid, Accrued and Not yet committed, on every level except unassigned rows.
  - Committed and Invoiced join the default columns after Unassigned planned cost.
  - Containers add up their matching activities, as planned cost does. The totals row sums them.
  - Columns are sortable and exported, and the export notes add "Financial figures: GSAP data as of
    <time>".
  - Saved views and remembered columns keep what they hold; **Reset to default columns** brings the new
    defaults.
- **FR-024** (changes 010 FR-004): All work MUST offer a **Financials** filter tested on activities: "Has a
  PO", "No PO yet", "Invoiced", "Committed above planned cost" (FR-013). Values combine with OR. It can be
  set from the address (`?finance=committedAbovePlanned`), as other filters can (D-122).
- **FR-025** (changes 009): Home MUST show a **Financials** card directly below the summary cards, for the
  current year and the Home scope (My teams / Everything). It shows:
  - Committed with % of planned cost, Invoiced, Paid and Accrued;
  - "n activities committed above planned cost", which links to All work with that filter;
  - "n transactions not linked to a line item", which links to Financials › Not linked;
  - **Open Financials**;
  - "GSAP data as of …".

  It loads and fails on its own, like the other Home cards (009 FR-022).

**Linking unmatched transactions (P3)**

- **FR-026**: On a **Not linked** transaction, **Link to a line item** MUST accept a line-item ID and
  preview the line item's name, activity and plan before saving. **Unlink** MUST be offered only on
  manually linked transactions. Links made by FR-007's rules cannot be changed in NorthStar; the panel
  says why and where to correct them (GSAP). A missing line item, or one in the recycle bin, is refused
  with a reason. A change made meanwhile by someone else is refused as a version conflict (D-030).

**Cross-cutting**

- **FR-027**: The activity log (feature 011) MUST record:
  - one entry per import ("imported GSAP extract '<file>'" with its counts);
  - one entry per link or unlink, with the old and new line item.

  Per-row changes made by an import are not logged one by one. The import entry and the transaction's
  "last updated by the import" stand for them.
- **FR-028** (BUS.ADM.020): Seeing financial figures and transactions, importing, and linking MUST each be
  checked by the API through one access check: feature 014's permissions "See financial data" (finance.view),
    "Import GSAP extracts" (finance.import) and "Link financial transactions" (finance.link), with 014 Table B
    defaults (consolidation D-301). A refusal answers 403 forbidden (D-299).
  - A person without "See financial data" sees "You don't have access to financial data —
    ask an administrator" in place of every Financials card, column, filter and page, and the API refuses
    their requests.
  - Every figure, list and count covers only the plans the person may see (014 FR-007).
- **FR-029** (changes 007 FR-030, 006 and 005 delete dialogs): Deleting a line item, activity, initiative,
  campaign or plan with linked transactions MUST say, in the delete dialog: "<n> GSAP transactions are
  linked to it. They stay in NorthStar and leave the figures until it is restored." Deleting is still
  allowed.
- **FR-030**: Copies of plans, campaigns, initiatives, activities and line items MUST NOT copy or relink
  transactions (R13).
- **FR-031**: The demo data MUST include the 11 transactions of data-model §6, loaded through the same
  matching rules as an import, plus one import record "Demo data (simulated GSAP)". The demo data is
  clearly labelled as demo data and fictitious.

### Key Entities *(include if feature involves data)*

- **Financial transaction**: one PO line, invoice, payment or accrual as GSAP sent it, with its amount in
  its own currency, its references (PO, SWBS, HX key, line-item ID), and its link to at most one line item
  (how it was matched, or why not). Read-only except a manual link.
- **GSAP import**: one delivery of GSAP data (a file or the demo data): who, when, source, file name and
  counts.
- **Financial figures** (derived): committed, invoiced, paid, accrued, not yet committed and % committed of
  a record, from the transactions linked to its live line items.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On every activity, plan, campaign and initiative view, committed, invoiced, paid and accrued
  are visible with **0 clicks** (open by default). On the activity page, they are in the second section.
  The Financials page is **1 click** from every page (sidebar).
- **SC-002**: For 100 % of tested records, each figure is equal to the cent wherever it is shown: the
  record's card, the parent's breakdown row, All work, Home and the Financials page totals for that
  record.
- **SC-003**: 100 % of imported rows end as exactly one of new, updated, unchanged or refused (with a
  reason), and every accepted row is either linked or not linked (with a reason). No row is dropped
  silently. Importing the same file twice changes nothing the second time.
- **SC-004**: A 5,000-row file is checked and imported in under 5 s on a developer laptop. The Financials
  page answers in under 1 s with 20,000 transactions. Each Financials card answers in under 500 ms with
  the seed.
- **SC-005**: In a test with at least 5 first-time users, at least 90 % find the POs and invoices of a
  named activity, and say whether it is committed above its planned cost, within 1 minute and without
  help.
- **SC-006**: A second, test-only feed can be plugged in with no change to handlers, screens, the contract
  or the figure rules (proven by a Go test).

## Assumptions

Defaults chosen under the product owner's decision of 2026-10-20 (build with documented, reversible
defaults). Each one is listed for Shell to confirm in the plan's *Proposed decisions* and in the questions
below.

- Purchase requests (PRs) are not shown. BUS.FIN.080 names POs, invoices, payments and accruals; PRs belong
  to the PR/PO process (BUS.FIN.020, later).
- "Paid" is Shell's "actuals" (workshop: "actuals are the payments released").
- NorthStar does not calculate accruals (BUS.FIN.040 is later). It shows GSAP's accruals of the latest
  accrual month.
- Conversion uses the current rate, like every other amount (consolidation D-303; with 016 the current rate is
  the latest dated rate on or before today). Posting-date rates (settings.RatesOn) are a later option.
- The simulated extract is a file uploaded through the API, not a scheduled job. A scheduled real feed would
  add a runtime part and needs an ADR (constitution IV).
- Line items keep no stored SWBS code. The code shown comes from the linked transactions.
- The plan tree keeps its columns. Plan-level figures are in the plan's Financials section and in All work.
- The Budget & cost view of All work is unchanged (no monthly actuals in this feature).
- PO and invoice documents (PDF copies, BUS.FIN.060) and invoice approval (BUS.FIN.050) are later.
- Who may see, import and link follows 014 (D-237, D-301): every default role sees financial data; Administrator,
  Global planner and Planner link; only Administrator imports.

### Questions for Shell (to confirm; the defaults above apply until answered)

| # | Question | Default |
|---|---|---|
| Q-015 (gap analysis) | Please share **"the note on financial transactions"** that BUS.FIN.080 refers to. It is not in the BRD PDF. Which transaction types and fields must users see, and where? | Four types, the fields of FR-001, the placements of FR-015–FR-025 |
| Q-015a | Is "Accrued" the latest month's accrual position per PO, or a running total? | Latest accrual month in the data |
| Q-015b | Which figure do planners compare with planned cost first: committed (PO), invoiced or paid? | Committed, with a meter; all four shown |
| Q-015c | Is the HX key always "HX" + the NorthStar line-item number? Who fixes rows GSAP cannot match? | Yes; people with "Link financial transactions" (Administrator, Global planner, Planner) link them by hand (FR-026) |
| Q-015d | Which rate converts a transaction: the posting-date rate or the current rate? (with Q-012) | The current rate, as every conversion (D-303) |
| Q-015e | How often will GSAP extracts arrive during the freeze, and in which layout (PGSI 941 sample files)? | Manual file import in NorthStar's documented CSV layout |
| Q-015f | Who may see POs, invoices and payments (with Q-010)? | Every default role sees them; importing is for Administrators (014 Table B) |
