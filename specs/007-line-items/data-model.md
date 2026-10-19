# Data Model: Line Items

**Feature**: `007-line-items` · **Plan**: [plan.md](plan.md) · **Contract**: [contracts/openapi.yaml](contracts/openapi.yaml)

The records this feature stores, the values it derives from them, their rules and their history. It
follows the conventions of [001's data model](../001-app-shell-and-sign-in/data-model.md): one table per
record type, relational in shape (ids, parent ids, no nesting of children), `record.Meta` bookkeeping
embedded and flattened, timestamps UTC, text lengths in Unicode characters after trimming.

Money follows [domain model §4](../../docs/product/domain-model.md): exact decimals with two decimal places,
never binary floating point. In Go every amount is a `money.Amount` and every rate a `money.Rate` from
`backend/internal/platform/money` (designed by feature 002). In the API every amount the API **returns**
is a Money object `{"amount": "46000.00", "currency": "EUR"}`; the amounts a client **sends** are decimal
strings in the line item's currency, because the form has a single **Currency** field (plan, decision
007-P10).

---

## 1. Line item — table `line_items`

One cost of an activity (spec Key Entities). Fields and markings from
[field catalogue §5](../../docs/product/field-catalogue.md). **R** = Required (blocks saving), **S** =
Needed to submit (blocks submitting the activity, feature 006/008), — = optional.

| JSON field | Label on screen | Type | Mark | Default for a new line item (FR-008) | Validation when saving (exact messages in plan, *Validation messages*) |
|---|---|---|---|---|---|
| `id` | ID | string | — | next `L-` number (`L-1001`, …) | Given by `ids.Generator` prefix `L` (D-019); never changes, never reused |
| `activityId` | (the activity) | string | R | the activity it was added to | An activity that is stored; never changes (line items are not moved in this phase) |
| `name` | Line item name | text, ≤ 120 | R | `"<activity name> – line item"`, then `" 2"`, `" 3"`… when taken | Required; ≤ 120; unique among the activity's line items ignoring letter case and surrounding spaces (FR-009) |
| `ownerId` | Owner | person id, `""` when empty | S | the activity's owner | Empty, or an existing person; a **new** choice must be active (an inactive owner already stored is kept) |
| `description` | Description | long text, ≤ 2,000 | — | `""` | ≤ 2,000 |
| `currency` | Currency | ISO code | R | `USD` | Required; an existing currency (feature 002); a **new** choice must be active; an inactive currency already stored is kept and still converts at its last rate (FR-011) |
| `plannedCost` | Planned cost | amount ≥ 0.00 in `currency` | R | the activity's unassigned planned cost when it is above 0.00, otherwise empty (FR-008) | Required; a decimal number; more than two decimals are rounded half-up to the cent; not negative; at most 999,999,999,999.99 (FR-009) |
| `spread.method` | Spread over time | `evenly` · `firstMonth` · `lastMonth` · `custom` | — | the plan's default line-item spread (field catalogue §1 #10) | One of the four values |
| `spread.months` | (custom month amounts) | list of `{month, amount}` | — | `[]` | Kept only when the method is `custom` (otherwise stored as `[]`); each `month` is `YYYY-MM` from January of the plan's year − 1 to December of the plan's year + 10, at most once; each `amount` ≥ 0.00, rounded half-up to the cent; sorted by month (FR-018) |
| `attributes` | Spend type, GL code, Spend category, Contracted spend and any other line-item Shell attribute | `AttributeValues` (feature 002), keyed by attribute id | per attribute (Spend type S, GL code S, others —) | `{}` | Checked by feature 002's attribute rules: known attribute and option, a new choice active, a dependent value (GL code) fitting its parent (Spend type, R11) (FR-014) |
| `contractedAmount` | Contracted amount | amount ≥ 0.00 in `currency`, or `null` | — | `null` | A decimal number, rounded half-up to the cent, not negative (FR-014) |
| `requestSwbs` | Request SWBS code from GSAP | yes/no | — | `true` | Required (yes or no); editable in every activity status (FR-024) |
| `financeBudgetYear` | Finance budget year | year or `null` | S | the plan's year | Empty, or the plan's year, or a year with active finance budget items; a stored year is always kept (FR-015) |
| `financeBudgetItemId` | Finance budget item | finance budget item id, `""` when empty | S | the only active item of the plan's default finance budget, when it has exactly one (R12) | Empty, or an existing item **of `financeBudgetYear`** (a year is needed first); a **new** choice must be active; an inactive item already stored is kept (FR-015, FR-016) |
| *bookkeeping* | Created by … on …, last changed by … on … | `record.Meta` | — | — | `version` checked on every update (D-030) |

Not stored, because they are always derived (§3): the USD planned cost and the rate it used, the month
amounts of `evenly`, `firstMonth` and `lastMonth` spreads, the SWBS request status, the completeness list.
Not stored, because nothing fills them in this phase (D-008, domain model §4.6): SWBS code, total actuals,
% of actual cost, actuals at monthly rate, total invoiced, total purchase requests — the screens show them
as "—" / "Filled by GSAP later".

**Go type** (`backend/internal/lineitems/model.go`):

```go
type LineItem struct {
	ID                  string            `json:"id"`                  // "L-1001"
	ActivityID          string            `json:"activityId"`          // never changes
	Name                string            `json:"name"`                // trimmed
	OwnerID             string            `json:"ownerId"`             // "" when empty
	Description         string            `json:"description"`         // trimmed
	Currency            string            `json:"currency"`            // "EUR"
	PlannedCost         money.Amount      `json:"plannedCost"`         // in Currency, ≥ 0.00
	Spread              Spread            `json:"spread"`
	Attributes          attributes.Values `json:"attributes"`          // feature 002's type; never nil ({})
	ContractedAmount    *money.Amount     `json:"contractedAmount"`    // in Currency; nil when empty
	RequestSwbs         bool              `json:"requestSwbs"`
	FinanceBudgetYear   *int              `json:"financeBudgetYear"`   // nil when empty
	FinanceBudgetItemID string            `json:"financeBudgetItemId"` // "" when empty
	record.Meta
}

type SpreadMethod string // "evenly", "firstMonth", "lastMonth", "custom"

type Spread struct {
	Method SpreadMethod  `json:"method"`
	Months []MonthAmount `json:"months"` // custom only, sorted by month; [] otherwise (never nil)
}

type MonthAmount struct {
	Month  string       `json:"month"`  // "2026-02"
	Amount money.Amount `json:"amount"` // in the line item's currency, ≥ 0.00
}
```

## 2. Relations and IDs

```
Plan (004) ─┐ year, default finance budget, default line-item spread, business unit (attribute conditions)
            └─ Campaign (005) ─ Initiative (005) ─ Activity (006) ── Line item (007)   one activity, many line items
                                                     │                    │
                                                     │ planned cost (USD),│ currency ──────► Currency (002)
                                                     │ dates, ongoing,    │ ownerId ───────► Person (003)
                                                     │ status, owner      │ attributes ────► Shell attributes and options, level "lineItem" (002)
                                                                          │ financeBudgetItemId ► Finance budget item (002)
```

- A line item belongs to exactly one activity (`activityId`) and needs a saved activity (domain model §1).
- Readable ID prefix **`L`** (D-019, domain model §1): `L-1001`, `L-1002`, … from the shared counter
  table, never reused — also not after a delete. Duplicates and copies get new IDs.
- Settings records (currencies, finance budget items, attributes and options) are referenced by their own
  ids; they are never deleted while used, only set inactive (domain model §6), so a reference always
  resolves. People are never deleted (D-028).
- Deleting an activity (or anything above it) takes its line items into the same recycle-bin entry
  (history kind `lineItem` with parent `activity`, §7).

## 3. Derived values (calculated on every read, never stored)

All calculations run in the Go API (`lineitems/spread.go`, `balance.go`, `grid.go`, `completeness.go`); the web app repeats
the same rules only for instant feedback in the form (plan, decision 007-P3).

### 3.1 USD planned cost

`plannedCostUsd = plannedCost ÷ rate`, rounded half-up to the cent, where `rate` is the currency's
**current** rate in Settings, "units of local currency per 1 USD" (domain model §4.3, D-010, D-027). USD has
rate 1. An inactive currency converts at the rate it last had. Example: 46,000.00 EUR at 0.92 → 50,000.00
USD; 10,000.00 EUR at 0.92 → 10,869.57 USD.

Because nothing in USD is stored, a rate change in Settings changes every line item's USD amount, the
balances and the roll-ups at once, without writing a line item and without changing any activity's
status (FR-013, domain model §2.5).

### 3.2 The activity's months

From the activity's dates (feature 006):

| Activity | Activity months |
|---|---|
| No start date, or a start date without an end date that is not ongoing | none — **no dates yet** (D-032) |
| Ongoing (no end date) | the start month only, for spreading; every month from the start month on counts as inside the activity's dates |
| Start and end dates | every calendar month from the start month to the end month, both included (1 Mar – 31 May 2026 → Mar, Apr, May 2026; 15 Dec 2026 – 2 Jan 2027 → Dec 2026, Jan 2027) |

### 3.3 Spread month amounts (local currency) — domain model §4.4, FR-017–FR-022

| Method | Label | Month amounts |
|---|---|---|
| `evenly` | Evenly | `n` = number of activity months; each month gets the planned cost ÷ `n` rounded **down** to the cent; the last month gets the rest. 10,000.00 over 3 → 3,333.33 · 3,333.33 · 3,333.34; 0.00 → 0.00 in every month; 0.02 over 3 → 0.00 · 0.00 · 0.02 |
| `firstMonth` | All in first month | everything in the first activity month |
| `lastMonth` | All in last month | everything in the last activity month |
| `custom` | Custom | the stored `spread.months`, as typed |

- **Ongoing** activity: `evenly` and `lastMonth` behave like `firstMonth` (FR-021).
- **No dates yet**: `evenly`, `firstMonth` and `lastMonth` are **not spread** (state `notSpread`); their
  amount appears in the grid's "No dates yet" column (D-032). Custom amounts are kept as typed.
- Because these three are calculated from the activity's current dates and the line item's current planned
  cost, a change of the activity's dates (including becoming ongoing) or of the line item's planned cost
  or currency **re-spreads them automatically**; custom amounts are kept (FR-022).
- **Adds up**: `evenly`, `firstMonth` and `lastMonth` always add up. A custom spread adds up when the sum of
  its months equals the planned cost exactly; otherwise its **difference** is planned cost − month total
  (positive: "left"; negative: "too much") and the line item counts it as a problem (FR-019, §3.6).
- **Outside the activity's dates** (a month flag, FR-018): the activity has dates and the month is not one
  of its months (for an ongoing activity: before its start month). Possible only for custom months.

### 3.4 Month amounts in USD — D-024, FR-020

For each line item: convert each month's local amount to USD (half-up to the cent); the **target** is the
USD value of the sum of its local month amounts (for `evenly`, `firstMonth`, `lastMonth`: exactly
`plannedCostUsd`); add the difference between the target and the sum of the converted months to the
**last month that has a non-zero amount**. The USD months therefore always add up exactly to the target.
Example: 10,000.00 EUR evenly over Mar–May at 0.92 → 3,623.18 · 3,623.18 · 3,623.21 USD = 10,869.57 USD.

### 3.5 SWBS request status — domain model §2.4, FR-024

| `requestSwbs` | Activity status (stored, feature 006) | SWBS request status | Chip (design system §4) |
|---|---|---|---|
| `false` | any | `notRequested` — Not requested | neutral (grey) |
| `true` | `created`, `submitted`, `declined` | `waitingForApproval` — Waiting for activity approval | review (amber) |
| `true` | `approved` (also shown as *In progress* / *Complete*) | `readyToSend` — Ready to send to GSAP | positive (green) |

The SWBS code is always "Filled by GSAP later" in this phase (D-008).

### 3.6 Completeness of a line item — FR-028, domain model §2.6

The **missing list** (API `MissingField {field, label}`), in this order; its length `n` gives the chip
"All filled in" (n = 0), "1 field needed", "<n> fields needed":

| # | When | `field` | `label` |
|---|---|---|---|
| 1 | `ownerId` is empty | `ownerId` | Owner |
| 2 | each visible, active, needed-to-submit line-item Shell attribute without a value, in Settings order (feature 002 decides "visible": business-unit conditions of the activity's plan, R9; a dependent attribute whose parent value has no options set up is not counted, D-034) | `attributes.<attributeId>` | the attribute's label, e.g. GL code |
| 3 | `financeBudgetYear` is empty | `financeBudgetYear` | Finance budget year |
| 4 | `financeBudgetItemId` is empty | `financeBudgetItemId` | Finance budget item |
| 5 | the linked finance budget item is inactive (FR-016) | `financeBudgetItemId` | Choose an active finance budget item — '<item name>' is no longer active. |
| 6 | a custom spread does not add up (FR-019) | `spread` | Spread doesn't add up to the planned cost |

An inactive currency, an inactive owner and inactive attribute options are kept values, not problems
(domain model §6, feature 002 FR-027).

### 3.7 Unassigned planned cost (Sys_Gen_Balance) — domain model §4.3, FR-002, D-016

| Value | Rule |
|---|---|
| `assigned` (USD) | sum of the activity's line items' `plannedCostUsd` |
| `balance` (USD) | activity planned cost − `assigned`; `null` when the activity has no planned cost yet (plan, decision 007-P5) |
| `state` | `left` (balance > 0.00) · `allAssigned` (= 0.00) · `exceeded` (< 0.00) · `noPlannedCost` |
| `roundingHint` | true when the balance is not 0.00, is at most 0.05 USD either way, and at least one line item is in a currency other than USD (plan, decision 007-P6) |

A negative balance warns everywhere and never blocks saving; it blocks submitting the activity (D-016). A
positive balance is only advice (Q-008 default).

### 3.8 Spread grid of an activity — FR-003, FR-020

- **Columns**, in this order: every activity month (for an ongoing activity: its start month) together with
  every month used by a custom spread of one of its line items, sorted, without filling gaps; each month
  column carries `outsideActivityDates` (§3.3); then a **No dates yet** column when the activity has no
  dates and at least one line item is spread `evenly`, `firstMonth` or `lastMonth`; then the total.
- **Rows**: one per line item in creation order; each cell is the item's USD month amount for that column
  (§3.4) or empty ("—"); the No-dates cell holds `plannedCostUsd` of a not-spread item; the row total is the
  sum of its cells.
- **Totals row**: the sum of each column (empty when every cell is empty) and the grand total.
- **Quarterly** is a display choice of the web app: it adds the month columns of each calendar quarter
  ("Q1 2026"); the No-dates column stays as it is; a quarter is marked outside the activity's dates when
  every month column in it is.
- The unassigned balance is not part of the grid (spec Assumptions; feature 010 decides how it is spread in
  its views).

### 3.9 Activity summary (provided to features 004, 005, 006, 008, 009)

Per activity: number of line items, the §3.7 balance, the number of line items whose missing list is not
empty, and for each of those `{lineItemId, lineItemName, missing}`. Feature 006 shows them in the header
("Unassigned 20,000.00 USD"), in the completeness chip ("<n> line items need attention") and in the
**Line items** group of its Ready to submit panel, together with the negative balance ("Line items exceed
the activity's planned cost by 500.00 USD"); feature 008 uses the same data for its submit check (FR-010
a, c, d); features 004/005 count incomplete line items in "n items below need attention".

## 4. Statuses and transitions

Line items have **no status of their own** (domain model §2.4): they are approved as part of their
activity. They show two derived chips (§3.5, §3.6).

What a line-item change does to its activity's status (FR-031 – FR-033, domain model §2.5, D-013):

| Change to a line item | Activity *Created* or *Declined* | Activity *Submitted* or *Approved* (also shown as *In progress* / *Complete*) |
|---|---|---|
| Create, Save and add another, Duplicate | saved; status unchanged | confirmation "This activity will need approval again" first; on confirmation saved and the activity returns to *Created* (feature 006 `RestartApproval`, which ends the approval round through feature 008) |
| Change of planned cost or currency | saved; status unchanged | as above |
| Delete | moved to the recycle bin; status unchanged | as above (the extra line in the delete dialog) |
| Change of name, owner, description, spread, Shell attributes, contracted amount, SWBS request, finance budget year or item | saved; status unchanged | saved; status unchanged (FR-032) |
| Undo of a delete, or Restore from the recycle bin | comes back; status unchanged | comes back **and the activity returns to *Created*** — restoring counts as adding (FR-033); no dialog on Undo, the restore notes say so |
| Exchange-rate change in Settings | USD values change; status unchanged | USD values change; status unchanged (FR-013) |

Undo never restores an approval that the delete ended (FR-033): the delete already set the activity to
*Created*, and the bin entry holds only the line item.

## 5. Copies

| Copy | Where | What is copied | Name | Log |
|---|---|---|---|---|
| **Duplicate** (FR-029) | the same activity | every stored field above, including the spread, Shell attributes, contracted amount, SWBS request and finance budget link; a new ID and new bookkeeping (the actor creates it) | `"<name> (copy)"`, then `" (copy 2)"`, …; when the result would pass 120 characters, the original part is shortened so the suffix fits | one `copied` entry on the copy, details "Copied from '<name>' (<id>)" |
| **With an activity, campaign or plan copy in the same year** (features 004–006) | the new activity | the same, with new IDs; names unchanged | as the source | none of their own: the copying feature writes one `copied` entry on the record it copied, counting the line items ("… with 3 line items"; 001 SC-004, 004 plan copy, D-110) |
| **Next-year plan copy** (feature 004 FR-028) | the new activity in the next year's plan | as above, and: `financeBudgetYear` + 1 (when set); `financeBudgetItemId` = the **active** item of the new year with the same `code` (when the old item has a code and exactly one such item exists), otherwise `""`; every custom month one year later | as the source | as above |

Copies never carry GSAP or finance values (R13); none are stored in this phase.

## 6. Unassigned balance and the forms

| Situation | New line item's planned cost (FR-008) | Summary bar (FR-006) |
|---|---|---|
| balance > 0.00 | the balance in USD, currency USD, labelled "The activity's unassigned planned cost" | "Activity planned cost 120,000.00 USD · Unassigned now 20,000.00 USD · After saving 0.00 USD" |
| balance ≤ 0.00 | empty, note "Nothing left to assign — the activity's planned cost is fully assigned" | "After saving" red when negative |
| no activity planned cost | empty, note "This activity has no planned cost yet, so there is nothing to pre-fill" | "This activity has no planned cost yet — the unassigned planned cost appears once it has one." |

While the planned cost still holds the pre-filled balance, changing the currency converts it so its USD
value equals the balance: `local = balance × rate`, rounded half-up to the cent (FR-012); when the
conversion back cannot match to the cent, the rounding hint of §3.7 is shown.

## 7. History

### 7.1 Kind `lineItem` (registered in `internal/app`, after feature 006's `activity`)

| Part | Value |
|---|---|
| `Name` / `Singular` / `Plural` | `lineItem` / `line item` / `line items` |
| `Parent` | `activity` (feature 006) |
| `Snapshot(id)` | the stored line item; `Name` = its name; `ParentID` = `activityId` |
| `ChildIDs(activityID)` | ids of the activity's line items |
| `Remove`, `Put` | delete from / write to `line_items` |
| `Describe` | labels and texts of §7.3 |
| `PrepareRestore` | **Undo**: refuses with `name_taken` "Can't undo — there is now another line item called '<name>'. Restore it from the recycle bin instead." when another line item of the activity has the name; **Restore**: renames to `" (restored)"`, `" (restored 2)"`, … with the note "Renamed to '<new name>' because the name was taken." and `Touch` |
| `AfterRestore` (new optional hook, plan *Backend*) | only when the restored record is the line item itself: when its activity is *Submitted* or *Approved*, calls feature 006 `RestartApproval` with the reason "Line item '<name>' (<id>) restored: <amount> <currency>" and adds the note "Activity '<activity name>' needs approval again because a line item came back." |

### 7.2 Activity-log entries

| Person's action | Entries written (one transaction) |
|---|---|
| Create (and Save and add another) | 1 × `created` on the line item (path: plan › campaign › initiative › activity); on a submitted/approved activity also feature 006's status entry from `RestartApproval` |
| Save with changes | 1 × `changed` listing every changed field old → new (§7.3); the restart entry as above when cost or currency changed on a submitted/approved activity |
| Save without a change | none (no new version) |
| Save from an out-of-date copy | none (`version_conflict`) |
| Duplicate | 1 × `copied` on the copy ("Copied from 'Agency fee' (L-1001)") (+ restart entry) |
| Delete | 1 × `deleted` (history) (+ restart entry); one recycle-bin entry holding the line item |
| Undo / Restore | 1 × `restored` (history), details "Undo" or "Restored from the recycle bin" plus the notes (+ restart entry when FR-033 applies) |
| Exchange-rate change, activity date change | none for line items (their derived values change; the Settings or activity entry records the cause) |

### 7.3 Labels and texts of changed fields (`Describe`)

| Field | Label | Text of a value |
|---|---|---|
| `name`, `description` | Name, Description | as typed |
| `ownerId` | Owner | the person's name ("Priya Raman"); "Not filled in" when empty |
| `currency` | Currency | the code ("EUR") |
| `plannedCost`, `contractedAmount` | Planned cost, Contracted amount | "46,000.00" (the currency is its own change) |
| `spread` | Spread over time | "Evenly" · "All in first month" · "All in last month" · "Custom: Feb 2026 2,000.00 · Mar 2026 3,333.33 · …" |
| `requestSwbs` | Request SWBS code from GSAP | Yes / No |
| `financeBudgetYear` | Finance budget year | "2026" |
| `financeBudgetItemId` | Finance budget item | "2026 Lubricants Budget › Europe – Digital" |
| `attributes` | not diffed as one field: feature 002's attribute changes are added instead, one change per attribute ("GL code: 7250080 – Advertising digital → 7250010 – Events") | |

## 8. Demo data (`backend/seed/seed.go`, after features 002–006 seed their records)

Fictitious, as [demo-users-and-data.md](../../docs/product/demo-users-and-data.md). The seed finds activities
by plan and activity name, currencies by code, finance budget items by year + budget + item name, and
Shell attributes and options by label; a reference it cannot find stops the seed with an error naming it.
Every seeded line item has its activity's owner and the finance budget year 2026; empty cells below are
left empty. Amounts marked *computed* are calculated by the seed from the seeded activity so the stated
balance holds whatever planned cost feature 006 seeds.

| Plan › activity (status from 006's seed) | Line item | Currency, planned cost | Spread | Spend type → GL code | Finance budget item | SWBS request | Shows |
|---|---|---|---|---|---|---|---|
| FY2026 Lubricants Europe › **Launch film** (Created, 120,000.00 USD, 1 Mar – 31 May 2026, owner Priya Raman) | Agency fee | USD 50,000.00 | Evenly | Digital – Platform development → 7250080 – Advertising digital | 2026 Lubricants Budget › Europe – Digital | on | All filled in; Waiting for activity approval |
| same | Film production | EUR 46,000.00 (= 50,000.00 USD) | Evenly | — | — | on | "3 fields needed"; **balance 20,000.00 USD** (spec US1) |
| FY2026 Lubricants Europe › the seeded **Approved** activity (expected "Helix dealer event") | Venue hire | GBP 15,800.00 (= 20,000.00 USD) | All in first month | BTL – Event → its only GL code | 2026 Lubricants Budget › Europe – Events | on | Ready to send to GSAP |
| same | Speaker fees | USD 5,000.00 | All in last month | BTL – Event → its only GL code | 2026 Lubricants Budget › Europe – Events | off | Not requested |
| FY2026 Lubricants Europe › the seeded **Submitted** activity (expected "Helix search campaign") | Media booking | USD 30,000.00 | Custom: 6,000.00 in the month before the activity's start month (a deposit, outside its dates), the other 24,000.00 spread evenly over its months (*computed*) | Digital – Platform development → 7250080 – Advertising digital | 2026 Lubricants Budget › Europe – Digital | on | custom spread with an outside month that adds up |
| FY2026 Lubricants Europe › the seeded **Complete** activity (expected "Rimula trade fair") | Stand build | EUR 9,200.00 | Evenly | BTL – Event → its only GL code | 2026 Lubricants Budget › Europe – Brand | on | Ready to send to GSAP (shown activity status *Complete*) |
| FY2026 Lubricants Europe › the seeded **Declined** activity | — (no line items) | | | | | | the empty state |
| FY2026 Mobility China › the deliberately incomplete activity (expected "Station forecourt refresh", 006 US2) | Station signage | USD 1,000.00 | Custom: 450.00 in its first month, 450.00 in its second (900.00 of 1,000.00) | BTL – Event, GL code empty | 2026 Mobility Budget › China – Retail | on | "Spread doesn't add up…" and the missing GL code |
| same | Fleet card promotion | CNY *computed*: (activity planned cost − 500.00 USD) × 7.10, so the two line items exceed the planned cost by **500.00 USD** (the seed stops with an error when that planned cost is below 500.00) | Evenly | Digital – Platform development → 7250080 – Advertising digital | 2026 Mobility Budget › China – Fleet | on | negative balance (006 US2 scenario 2) |

Eight line items: `L-1001` … `L-1008` in table order (the seed takes IDs from the counter, so later records
continue at `L-1009`).
