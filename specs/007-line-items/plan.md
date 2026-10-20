# Implementation Plan: Line Items

**Branch**: `007-line-items` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/007-line-items/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (a
fragment merged into `backend/api/openapi.yaml`) · conventions of
[001's plan](../001-app-shell-and-sign-in/plan.md), [research](../001-app-shell-and-sign-in/research.md) and
[data model](../001-app-shell-and-sign-in/data-model.md). The research decisions of this feature are the
*Decisions* section below (no separate research.md); the run-and-check steps are in *Quickstart checks*.

## Summary

Line items are the costs inside an activity (feature 006): each has a local currency converted to USD at
the current rate, a spread over months (Evenly · All in first month · All in last month · Custom), finance
coding through line-item Shell attributes (Spend type → GL code, …), a link to a finance budget item, and a
"Request SWBS code from GSAP" flag with a derived status. The activity page gets a **Line items** section:
the rows, the **Unassigned planned cost (Sys_Gen_Balance)** system row with a meter, and a month-by-month
grid in USD with a Monthly/Quarterly switch. Create, view and edit are full pages; Duplicate and Delete
(recycle bin, Undo) follow the standard CRUD pattern; changes that matter for approval on a Submitted or
Approved activity are confirmed and send it back to *Created*.

Technical approach, in one paragraph: a new Go package `backend/internal/lineitems` (handler → service →
repository, JSON table `line_items`) that stores only what people type — **USD amounts and the month
amounts of Evenly/First/Last spreads are calculated on every read** from the current rate and the
activity's current dates, so rate and date changes can never leave stale numbers; it reaches activities,
plans, Settings and people through small interfaces wired in `internal/app` (no import cycles), registers
the history kind `lineItem` under `activity`, and gives features 004–010 one function for every number
they show about line items. The web app adds `frontend/src/features/line-items/` built from the shared
components, with exact BigInt-cent arithmetic for live previews in the form.

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19.3 (web app) — as feature 001

**Primary Dependencies**: none new. API: standard library; `backend/internal/platform/money` (feature 002).
Web app: react-router 8, @tanstack/react-query 5, radix-ui, lucide-react, openapi-fetch (feature 001).
Exact decimals in the browser use the built-in `BigInt`.

**Storage**: local JSON file store, new table `line_items` (`backend/data/line_items.json`) behind
`lineitems.Repository`; PostgreSQL later without service changes (ADR-0004)

**Testing**: Go `testing` (service rules with fakes of the other features, a fixed clock and a temporary
store) + `apitest` contract tests; Vitest + Testing Library + `mockApi`; Playwright + axe
(`frontend/e2e/line-items.spec.ts`)

**Target Platform**: as 001 (local laptop; Chrome/Edge)

**Project Type**: web application — React SPA + Go REST API

**Performance Goals**: the Line items section of an activity with 50 line items over 36 months answers in
under 100 ms locally; the form's live preview updates on every keystroke without a request

**Constraints**: 127.0.0.1 only; no personal data in server logs; money never in binary floating point;
WCAG 2.2 AA; usable at 1280 × 720 and 390 px (the grid scrolls sideways inside its card)

**Scale/Scope**: demo-sized (010 expects up to 20,000 line items a year); 7 API operations; 3 new web
routes, 1 section embedded in feature 006's activity page, 1 note embedded in 006's activity edit form

## Constitution Check

*GATE: checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to an FR, scenario or edge case of [spec.md](spec.md) (*Requirement traceability*). The spec has no `[NEEDS CLARIFICATION]`; the one open question it touches (Q-008) uses the decision-log default. New choices are listed as proposed decisions | Pass |
| II. Shell business, plain language | Screens say Line item, Unassigned planned cost (Sys_Gen_Balance), Request SWBS code from GSAP, Evenly / All in first month / All in last month / Custom — never C/E/F/L or Hive9 terms; every text is in *User-visible messages*. Currencies, rates, finance budget items and Shell attributes are Settings data (002), never hard-coded. Nothing is sent to GSAP (D-008) | Pass |
| III. Simple, friendly experience | Standard CRUD pattern (full-page create/view/edit, ⋯ Actions, delete dialog, Undo); status and completeness chips on every row and page; defaults labelled with their source; all problems at once; nothing disabled without a reason (GL code hint, inactive values labelled); re-approval confirmed before saving | Pass |
| IV. Simple, layered architecture | One package `lineitems`, handler → service → repository; every rule and calculation runs in the API (the form's previews repeat them only for feedback, 007-P3); contract in OpenAPI, web types generated; storage only through `lineitems.Repository` and `store.Transactor` | Pass |
| V. Tested business rules | Unit tests for every money, spread, balance, completeness and status rule (named with the FR); a contract test per operation; component tests per page; an end-to-end test for each P1 story (US1–US4); fixed clock and seed | Pass |
| VI. Safe by default | The actor is passed to every service method and recorded on every change; delete goes to the recycle bin with Undo; exact decimals; input validated by the API; no personal data in logs | Pass |
| VII. Keep it small and readable | No new dependency; nothing stored that can be calculated (007-P1, 007-P2); no new shared component (feature-local components only); the one change to 001 code is the optional `history.Kind.AfterRestore` hook 001 anticipated (data model §7) | Pass |

**Post-design re-check**: Pass. The two choices that add code beyond the screens are in *Complexity
Tracking*.

## Decisions

### Open questions and markers

| Item | Resolution |
|---|---|
| `[NEEDS CLARIFICATION]` in spec.md | None (FR-013 was resolved by D-027 before planning) |
| Q-008 — must the unassigned balance be exactly 0.00 to submit? | Default: **advice only**; only a negative balance blocks submitting (D-016). The section shows a positive balance as "… not yet assigned to a line item"; feature 006 shows it in its panel as advice |
| Q-005 — markings of the line-item fields | As the field catalogue §5 (owner, spend type, GL code, finance budget year and item: needed to submit) |
| Q-003 — real option lists | Fictitious seed options of feature 002 |

### Proposed decisions (numbered at consolidation, 2026-10-19)

- **D-140 (007-P1)** — The month amounts of Evenly, All in first month and All in last month are
  never stored: they are calculated whenever a line item is read, from its planned cost and the activity's
  current dates. Only Custom month amounts are stored. *Reason*: FR-022's automatic re-spread then needs no
  code and no write, can never be forgotten by another feature that changes dates (006 edit, 006 move, 012
  bulk change), and the numbers are always consistent.
- **D-141 (007-P2)** — USD amounts of line items are never stored; they are calculated with the
  currency's current rate on every read (D-027). *Reason*: FR-013 asks that a rate change updates every USD
  amount, balance and roll-up "at once"; with nothing stored there is nothing to update.
- **D-142 (007-P3)** — The line-item form calculates its live previews (conversion, spread
  preview, custom total, balance after saving) in the browser with exact cent arithmetic (`BigInt`); the
  API recalculates everything on save and its answer is what the pages show. Both are tested against the
  same example table. *Reason*: feedback on every keystroke without a request; constitution IV allows
  repeating a rule for faster feedback.
- **D-143 (007-P4)** — Line-item screens show money as "<amount> <code>" ("20,000.00 USD",
  "46,000.00 EUR"), negative amounts with a minus sign in red. *Reason*: several currencies appear side by
  side; the spec's texts use this form.
- **D-144 (007-P5)** — An activity without a planned cost has no unassigned balance: the system row
  says so, a new line item's planned cost starts empty, and no negative-balance problem is raised (the
  missing planned cost is already a needed-to-submit problem of the activity). *Reason*: "0 − line items"
  would show a misleading red "exceeded" message on every new activity.
- **D-145 (007-P6)** — The rounding hint ("If a currency rate leaves a few cents unassigned…") shows
  when the balance is not 0.00, is at most 0.05 USD either way, and a line item uses another currency than
  USD. *Reason*: the spec's "a few cents" needs a number; one local cent is at most ~1.3 US cents at the
  seeded rates.
- **D-146 (007-P7)** — Re-approval on the API: creating, duplicating or deleting a line item, or
  changing its planned cost or currency, while the activity is Submitted or Approved is refused with `400
  validation_failed` and a problem on the field `restartApproval` unless the request says
  `restartApproval: true` (DELETE: query parameter `restartApproval=true`) — the convention of feature 006
  (D-106), chosen for both features at consolidation (D-189). *Reason*: SC-007 needs the
  confirmation to be enforced even when someone submits the activity while the form is open.
- **D-147 (007-P8)** — Undo of a line-item delete and Restore from the recycle bin bring the line
  item back without a dialog; when its activity is (by then) Submitted or Approved, the activity returns to
  *Created* and the restore notes say so (FR-033). Implemented with an optional `history.Kind.AfterRestore`
  hook, run once for the deleted record after everything is back (anticipated in 001 data model §7).
  *Reason*: the toast's Undo has no dialog (D-037); feature 011's restore dialog already states status
  changes (011 FR-020).
- **D-148 (007-P9)** — A next-year plan copy (004 FR-028) moves a line item's finance budget year
  one year on and links it to the active item of the new year with the same code (when exactly one exists),
  otherwise clears the link; custom months move one year later. *Reason*: codes identify finance budget
  items across years; budget names contain the year and cannot be matched reliably; a cleared link is shown
  as needed, never silently wrong.
- **D-149 (007-P10)** — Requests send amounts as decimal strings in the line item's currency (one
  `currency` field per line item) with at most two decimals — more decimals are refused with a field
  message, not rounded (002's `MoneyInput` rule, D-190); every amount the API returns is a Money object
  `{amount, currency}`.
  *Reason*: the form has one currency field and an empty planned cost must still carry its currency.
- **D-150 (007-P11)** — When an activity's dates change, feature 006's edit form shows, under the
  dates, how many line items are re-spread and how many custom spreads are kept (and repeats it in its
  re-approval dialog). *Reason*: FR-022 asks the "save confirmation" to say so, but a *Created* activity has
  no save dialog.
- **D-151 (007-P12)** — Values that a change will clear or replace (a GL code that no longer fits,
  a finance budget item of another year, typed custom amounts) are announced by a note at the field as soon
  as the change is made and are cleared on Save; there is no extra dialog. *Reason*: "lists … before
  saving" (FR-014, FR-015, FR-018) with fewer clicks; the only save-time dialog stays the re-approval one.
- **D-152 (007-P13)** — Custom months may lie from January of the plan's year − 1 to December of
  the plan's year + 10 (R5). A new custom grid starts with the activity's months; for an ongoing activity
  from its start month to December of the plan's year (FR-021); without activity dates with the twelve
  months of the plan's year. *Reason*: bounds the grid; gives the user a starting point in every case.
- **D-153 (007-P14)** — The line-item details and edit pages have their own addresses
  (`/line-items/:lineItemId`, `/line-items/:lineItemId/edit`), and the update request sends the whole form
  with its version. *Reason*: links from the activity page, the Ready to submit panel and other features
  work by line-item ID; the form always holds every field, so a full update is the simplest correct
  request.
- **D-154 (007-P15)** — A save of a line item that no longer exists says "This activity was
  deleted — restore it from the recycle bin to keep working on its line items." when it went to the
  recycle bin together with its activity (found in the bin entry), and "This line item no longer exists —
  someone may have deleted it." otherwise. *Reason*: the spec's edge case names the activity; the bin is
  the only place that still knows why the line item is gone.

## Project Structure

### Documentation (this feature)

```text
specs/007-line-items/
├── spec.md                  # what and why
├── checklists/requirements.md
├── plan.md                  # this file
├── data-model.md            # stored record, derived values, statuses, history, seed
├── contracts/openapi.yaml   # this feature's paths and schemas (fragment)
└── tasks.md                 # the task list
```

### Source code (repository root)

```text
backend/
├── api/openapi.yaml                       # + the fragment (tag "Line items", 7 operations, schemas)
├── internal/
│   ├── history/kinds.go · service.go      # + optional Kind.AfterRestore (and its tests)
│   ├── lineitems/                         # NEW — one package, the same shape as internal/samples
│   │   ├── model.go          # LineItem, Spread, MonthAmount, SpreadMethod; API types
│   │   ├── deps.go           # interfaces Activities, Currencies, FinanceBudgets, Attributes, People
│   │   ├── repository.go · store_json.go  # table line_items
│   │   ├── spread.go         # activity months, spread months, USD months (pure functions)
│   │   ├── balance.go        # unassigned balance (pure)
│   │   ├── grid.go           # spread grid (pure)
│   │   ├── completeness.go   # missing list, SWBS status (pure)
│   │   ├── validate.go       # field checks → errs.Problems
│   │   ├── service.go        # Section, FormOptions, Create, Get, Update, Duplicate, Delete
│   │   ├── provided.go       # Summaries, CopyLineItems, usage counts, RateChangeImpact
│   │   ├── kinds.go          # history kind "lineItem"
│   │   ├── handler.go
│   │   └── *_test.go         # spread, balance, grid, completeness, service, provided, kinds (package lineitems),
│   │                         # fakes_test.go; handler_test.go (package lineitems_test)
│   └── app/app.go · lineitems.go          # wiring; adapters implementing the lineitems interfaces
└── seed/seed.go · seed_test.go            # + addLineItems

frontend/
├── src/app/router.tsx                     # + 3 routes
├── src/features/line-items/               # NEW
│   ├── lineItemsApi.ts       # queries, mutations, query keys, invalidation
│   ├── lineItemMath.ts       # exact cents: parse, convert, spread, quarters, spread effect (+ test)
│   ├── lineItemText.ts       # every text of this feature (+ test)
│   ├── lineItemFormState.ts  # form state, defaults, currency change, request bodies, restart reasons (+ test)
│   ├── LineItemsSection.tsx · UnassignedBalanceRow.tsx · SpreadGrid.tsx · SwbsChip.tsx (+ css, tests)
│   ├── LineItemForm.tsx · LineItemSummaryBar.tsx · CostFields.tsx · SpreadEditor.tsx ·
│   │   FinanceBudgetFields.tsx · FromFinanceSection.tsx (+ css, tests)
│   ├── RestartApprovalDialog.tsx · DeleteLineItemDialog.tsx
│   ├── SpreadEffectNote.tsx (+ test)      # used by feature 006's activity edit form
│   ├── LineItemCreatePage.tsx · LineItemPage.tsx · LineItemEditPage.tsx (+ css, tests)
│   └── testData.ts           # API fixtures for the tests (the "Launch film" example)
├── src/features/activities/ActivityPage.tsx · (006's edit form)   # embed the section and the note
└── e2e/line-items.spec.ts · fixtures.ts (+ openActivity helper)
```

**Structure decision**: the web-application layout of feature 001; a business package per area
(constitution IV); pages under `src/features/line-items/`.

## Backend design

### Package `lineitems` and its dependencies

`lineitems` must read activities, plans, Settings and people, while features 004–006 must read line-item
numbers. To avoid import cycles, `lineitems` imports only the platform packages, `history`, `money` and
feature 002's `attributes` value types, and declares what it needs as interfaces in `deps.go`;
`internal/app/lineitems.go` implements them with the services of 002, 003, 004 and 006. The other features
receive the line-item functions they need after construction (see *Dependencies*).

```go
// deps.go — every method runs inside the caller's transaction (the context carries it).
type Deps struct {
	Activities     Activities
	Currencies     Currencies
	FinanceBudgets FinanceBudgets
	Attributes     Attributes
	People         People
}

type Activities interface {
	// Facts returns the activity with its plan's facts; false when the activity is not stored.
	Facts(ctx context.Context, activityID string) (ActivityFacts, bool, error)
	// RestartApproval sends a Submitted or Approved activity back to Created (006 FR-027; 008 FR-022 ends
	// the round), logging the reasons. It joins the caller's Write.
	RestartApproval(ctx context.Context, a actor.Actor, activityID string, reasons []string) error
}

type ActivityFacts struct {
	ID, Name, OwnerID string
	Status               string        // stored: "created", "submitted", "declined", "approved"
	StartDate, EndDate   string        // "YYYY-MM-DD"; "" when not set
	Ongoing              bool
	PlannedCost          *money.Amount // USD; nil when not set
	Path                 []history.RecordRef // plan, campaign, initiative
	PlanYear             int
	DefaultSpread        SpreadMethod  // the plan's default line-item spread
	DefaultFinanceBudget string        // budget name in the plan's year; "" when none
	AttributeContext     attributes.Context // what 002 needs to decide which attributes are shown (business unit)
}

type Currencies interface {
	Currency(ctx context.Context, code string) (Currency, bool, error) // active or not
	Active(ctx context.Context) ([]Currency, error)
}
type Currency struct { Code, Name, Symbol string; Rate money.Rate; Active bool }

type FinanceBudgets interface {
	Item(ctx context.Context, id string) (BudgetItem, bool, error)  // active or not
	ActiveItems(ctx context.Context, year int) ([]BudgetItem, error)
	YearsWithActiveItems(ctx context.Context) ([]int, error)
}
type BudgetItem struct { ID string; Year int; BudgetName, Name, Code string; Active bool }

type Attributes interface {
	// Check returns the problems of after (unknown attribute or option, a new choice of an inactive option,
	// a dependent value that does not fit its parent — R11), with fields "attributes.<attributeId>".
	Check(ctx context.Context, before, after attributes.Values, c attributes.Context) ([]errs.FieldError, error)
	// Missing returns the visible, active, needed-to-submit line-item attributes without a value (D-034).
	Missing(ctx context.Context, values attributes.Values, c attributes.Context) ([]MissingField, error)
	// Changes describes, one change per attribute, what differs between before and after (activity log).
	Changes(ctx context.Context, before, after attributes.Values) ([]history.Change, error)
}

type People interface {
	Person(ctx context.Context, id string) (name string, active, found bool, err error)
}
```

Constructors: `NewJSONRepository() Repository`; `NewService(tx store.Transactor, repo Repository, generator
*ids.Generator, hist *history.Service, deps Deps, now clock.Clock) *Service`; `(*Service).Kinds()
[]history.Kind`; `NewHandler(s *Service) *Handler`; `(*Handler).Routes(r *web.Router, signedIn web.Wrap)`.

Repository (`store_json.go`, table `line_items`): `Get(ctx, id) (LineItem, bool, error)`,
`ListByActivity(ctx, activityID) ([]LineItem, error)` (creation order = ID number order, as `store.List`
sorts), `List(ctx) ([]LineItem, error)`, `Put(ctx, LineItem) error`, `Delete(ctx, id) error`.

### Stored type

`LineItem`, `Spread`, `MonthAmount`, `SpreadMethod` exactly as [data-model.md §1](data-model.md). Lists
never nil (`Spread.Months` `[]`, `Attributes` `{}`).

### Calculations (pure functions, `spread.go`, `balance.go`, `grid.go`, `completeness.go`)

| Function | Rule | Data model |
|---|---|---|
| `ActivityMonths(f ActivityFacts) (months []string, hasDates bool)` | calendar months of the dates; ongoing → start month; no dates → `false` | §3.2 |
| `SpreadMonths(li LineItem, f ActivityFacts) (months []MonthAmount, spread bool)` | Evenly (floor to the cent, rest in the last month), first, last, custom; ongoing → first; no dates → not spread | §3.3 |
| `OutsideActivityDates(month string, f ActivityFacts) bool` | | §3.3 |
| `USDMonths(months []MonthAmount, rate money.Rate) []money.Amount` | each half-up; difference to `ToUSD(sum)` into the last non-zero month | §3.4 |
| `PlannedCostUSD(li, rate)` | `money.ToUSD` half-up | §3.1 |
| `Balance(f ActivityFacts, usd []money.Amount, currencies []string) Balance` | §3.7 incl. `roundingHint` | §3.7 |
| `Grid(f ActivityFacts, items []itemView) Grid` | columns, rows, totals | §3.8 |
| `Missing(li, f, attributeMissing []MissingField, item *BudgetItem) []MissingField` | order and labels of §3.6 | §3.6 |
| `SwbsStatus(li, f) string` | §3.5 | §3.5 |

### Service operations

Every method takes `(ctx, a actor.Actor, …)`; reads run in `s.tx.Read`, changes in one `s.tx.Write` that
also writes the log entries and, when needed, calls `Activities.RestartApproval`. Field problems are
collected with `errs.Problems` **inside** the transaction (the name check needs the activity's line items),
so every problem — the duplicate name included — is reported at once (FR-009, SC-005).

| Operation | Rules, in order | FR |
|---|---|---|
| `Section(ctx, a, activityID) (Section, error)` | activity facts (missing → `404` "This activity was deleted — restore it from the recycle bin to keep working on its line items."); its line items in creation order; per line item the view: currency (with current rate and active flag), `plannedCostUsd`, spread months with USD and outside flags, `customTotal`, `addsUp`, owner, finance budget item, `missing`, `swbsStatus`; the balance; the grid | FR-001–FR-005, FR-013, FR-017, FR-020, FR-023, FR-024, FR-028 |
| `FormOptions(ctx, a, activityID) (FormOptions, error)` | activity facts (404 as above); balance; **defaults**: name `"<activity name> – line item"`, then `" 2"`, `" 3"`… while taken (case-insensitive; the activity part shortened so the name fits 120 characters); owner = activity owner; currency `USD`; planned cost = balance when > 0.00 (`plannedCostSource` `unassignedBalance`), else `null` with `nothingLeftToAssign`, or `noActivityPlannedCost` (007-P5); spread = plan default; budget year = plan year; budget item = the only active item of the plan's default finance budget in the plan year, else `null` (R12); `requestSwbs` true. **Choices**: active currencies (USD first, then by code); years = plan year ∪ years with active items, ascending; active items of those years ordered by year, plan's default budget first (in the plan year), budget name, item name | FR-008, FR-011, FR-012, FR-015 |
| `Create(ctx, a, activityID, in Input) (View, error)` | activity facts (404); validation of *Validation messages* (all at once); if `f.Status` is `submitted` or `approved` and not `in.RestartApproval` → `400 validation_failed` on `restartApproval`, nothing saved; new ID `L-…`; `record.NewMeta`; `Put`; log `created`; when the activity is submitted/approved: `RestartApproval(reasons: ["Line item '<name>' (<id>) added: <amount> <currency>"])` | FR-009–FR-012, FR-014–FR-016, FR-018, FR-019, FR-031 |
| `Get(ctx, a, id) (Details, error)` | the line item view, its activity's `LineItemActivity`, the current balance; missing → the messages of 007-P15 (bin lookup through `history.Service.ListBin`) | FR-026 |
| `Update(ctx, a, id, in Update) (View, error)` | `version` missing or < 1 → field `version` "Reload the page and try again." (reported with the other problems); line item missing → 007-P15; `CheckVersion(*in.Version, "line item")`; validation as Create (a stored inactive currency, owner or finance budget item that is **unchanged** is accepted, FR-011, FR-016); custom months kept only when the method is custom (switching away drops them, FR-018); a planned-cost or currency change keeps custom months (FR-022); no change (Diff empty and attribute changes empty) → return the stored line item, no version, no log; a change of `plannedCost` or `currency` on a submitted/approved activity without confirmation → `400 validation_failed` on `restartApproval`; `Touch`; `Put`; log `changed` (Diff without `attributes` + `Attributes.Changes`); restart reasons "Planned cost of line item '<name>' (<id>): <old> <cur> → <new> <cur>" (one sentence also when only the currency changed) | FR-011, FR-013–FR-019, FR-022, FR-027, FR-031, FR-032 |
| `Duplicate(ctx, a, id, confirm bool) (View, error)` | line item missing → 007-P15; confirmation as Create (counts as adding); copy every stored field; name `"<name> (copy)"`, `" (copy 2)"`, … unique in the activity, the original part shortened to fit 120 characters; new ID and Meta; log `copied` details "Copied from '<name>' (<id>)"; restart reason "Line item '<copy name>' (<new id>) added: …" | FR-029, FR-031 |
| `Delete(ctx, a, id, confirm bool) (history.BinEntry, error)` | line item missing → 007-P15; confirmation as Create; `RestartApproval(["Line item '<name>' (<id>) deleted: <amount> <currency>"])` when needed; `history.MoveToBin(ctx, a, "lineItem", id)` | FR-030, FR-031 |

`View` is the API `LineItem`; `Details` = `{lineItem, activity, balance}`; `Section` =
`{activity, items, balance, grid}`; `FormOptions` as the contract. Handlers map them 1:1 to the schemas of
[contracts/openapi.yaml](contracts/openapi.yaml).

### Validation messages (API; field names as in the request)

| Field | Problem | Message |
|---|---|---|
| `name` | empty after trimming | Enter a name for the line item. |
| `name` | more than 120 characters | Use 120 characters or fewer. |
| `name` | another line item of the activity has it (ignoring case and surrounding spaces) | '<activity name>' already has a line item called '<other name as stored>'. |
| `ownerId` | not a person | Choose a person from the list. |
| `ownerId` | a new choice of an inactive person | Choose an active person — <name> is inactive. |
| `description` | more than 2,000 characters | Use 2,000 characters or fewer. |
| `currency` | empty or unknown | Choose a currency. |
| `currency` | a new choice of an inactive currency | Choose an active currency — <CODE> is no longer active. |
| `plannedCost` | null or empty | Enter the planned cost. |
| `plannedCost` | not a decimal number | Enter an amount, such as 12500.00. |
| `plannedCost` | negative | Planned cost can't be negative. |
| `plannedCost`, `contractedAmount`, `spread.months[i].amount` | above 999,999,999,999.99 | Enter an amount below one trillion. |
| `contractedAmount` | not a decimal number / negative | Enter an amount, such as 12500.00. / Contracted amount can't be negative. |
| `spread.months[i].month` | not `YYYY-MM` or outside the range | Choose a month from Jan <plan year − 1> to Dec <plan year + 10>. |
| `spread.months[i].month` | the month appears twice | Each month can appear only once. |
| `spread.months[i].amount` | not a decimal number / negative | Enter an amount, such as 12500.00. / Month amounts can't be negative. |
| `attributes.<id>` | from feature 002 (`Attributes.Check`) | 002's messages, e.g. "GL code '7250080 – Advertising digital' doesn't fit 'BTL – Event'." |
| `financeBudgetYear` | a new year that is neither the plan's year nor a year with active items | Choose <plan year> or a year with finance budget items. |
| `financeBudgetItemId` | set without a year | Choose the finance budget year first. |
| `financeBudgetItemId` | unknown | Choose a finance budget item from the list. |
| `financeBudgetItemId` | of another year | Choose a finance budget item of <year>. |
| `financeBudgetItemId` | a new choice of an inactive item | Choose an active finance budget item — '<item name>' is no longer active. |
| `version` | missing or < 1 | Reload the page and try again. |

Amounts with more than two decimals are rounded half-up to the cent by the web form, which says so
("Rounded to 1,234.57", edge case "Rounding"); the API accepts at most two decimals and refuses more
with "Enter an amount with at most two decimals." on the field (002's `MoneyInput` rule, D-190). Other answers: `404` activity — "This activity was deleted — restore it from the
recycle bin to keep working on its line items."; `404` line item — 007-P15; `409 version_conflict` —
"Not saved — <person> changed this line item" (001); `400 validation_failed` on `restartApproval` — "This activity will
need approval again. Confirm the change to send it back for approval."

### History

- **Kind `lineItem`** (`kinds.go`) as [data-model §7.1](data-model.md): parent `activity`; `Describe` with the
  labels and texts of §7.3 (people's names through `People`, budget items through `FinanceBudgets`, money
  with `money.Format`); `PrepareRestore` refuses an Undo of a taken name (`errs.NameTaken` "Can't undo —
  there is now another line item called '<name>'. Restore it from the recycle bin instead.") and renames on
  Restore ("Renamed to '<new name>' because the name was taken.", `Touch`); `AfterRestore` applies FR-033.
- **Change to `history`** (001 code, anticipated in its data model §7): `Kind` gains the optional field

  ```go
  // AfterRestore runs once per Undo or Restore, for the kind of the record the person deleted (the bin
  // entry's first snapshot), after every record is back. It may change other records in the same Write
  // and returns notes for the log and the RestoreResult, like PrepareRestore. An error changes nothing.
  AfterRestore func(ctx context.Context, rc RestoreContext, top Snapshot) (notes []string, err error)
  ```

  `putBack` calls it after `putSnapshots` and appends its notes to the details and to
  `RestoreResult.details`. Children's `AfterRestore` hooks do not run when their parent is restored (the
  activity's own restore rules cover them).
- **Log entries** exactly as [data-model §7.2](data-model.md). Paths come from the kinds (plan › campaign ›
  initiative › activity).

### Provided to other features (`provided.go`, wired in `internal/app`)

| Function | For | Rule |
|---|---|---|
| `Summaries(ctx, a, facts []ActivityFacts) (map[string]ActivitySummary, error)` | 006 header, completeness chip, Ready to submit panel and tree mark; 008 submit check; 004/005 "items below need attention"; 009 problems | data-model §3.9; joins the caller's transaction |
| `CopyLineItems(ctx, a, fromActivityID, toActivityID string, yearShift int) (int, error)` | 004 plan copy, 005 campaign/initiative copy, 006 activity copy, 012 bulk copy | data-model §5 (yearShift 0 or 1, 007-P9); joins the caller's Write; writes **no** log entries — the copying feature writes the one `copied` entry of the user's action (001 SC-004; 004's plan-copy decision, D-110) and counts the line items from the returned number |
| `CurrencyUsage(ctx, a, code)`, `FinanceBudgetItemUsage(ctx, a, itemID)`, `AttributeUsage(ctx, a, attributeID, optionID)` → `Usage{Stored, InRecycleBin int}` | 002 "Used by …" and delete refusal (002 FR-003, FR-004) | stored line items plus line-item snapshots of unresolved bin entries (`history.ListBin`) |
| `OwnerUsage(ctx, a, personID) (int, error)` | 003 "Owner of …" (003 FR-004) | stored line items |
| `RateChangeImpact(ctx, a, code string, newRate money.Rate) (RateImpact, error)` | 002 rate-change confirmation (002 FR-030) | `{LineItems, Activities int; USDBefore, USDAfter money.Amount}` over stored line items |
| `ActivityMonths`, `SpreadMonths`, `USDMonths` (exported pure functions) | 010 Budget & cost, 013 SP&A | data-model §3.2–§3.4 |

### Demo data

`seed.go` gains `addLineItems(ctx, generator, createdAt)`, called after features 002–006 have added theirs,
creating the eight line items of [data-model §8](data-model.md) with `actor.System` and the seed time,
through `lineitems.NewJSONRepository()` and IDs from the counter `L`. References are looked up by name
(plan + activity name, currency code, year + budget + item name, attribute and option labels) through the
other features' repositories; a missing reference stops the seed with an error naming it.

## API operations

All behind `signedIn`; errors as 001 (`fields` always present). Schemas in
[contracts/openapi.yaml](contracts/openapi.yaml).

| Method and path | Request | Response | Errors |
|---|---|---|---|
| `GET /api/activities/{activityId}/line-items` | — | `200 LineItemSection` | 401; 404 activity |
| `GET /api/activities/{activityId}/line-items/form-options` | — | `200 LineItemFormOptions` | 401; 404 activity |
| `POST /api/activities/{activityId}/line-items` | `LineItemCreate` | `201 LineItem` | 400 `validation_failed` (all problems) / `bad_request`; 401; 404 activity; 400 `validation_failed` on `restartApproval` |
| `GET /api/line-items/{lineItemId}` | — | `200 LineItemDetails` | 401; 404 (007-P15) |
| `PATCH /api/line-items/{lineItemId}` | `LineItemUpdate` (whole form + `version`) | `200 LineItem` | 400; 401; 404; 409 `version_conflict` / `validation_failed` on `restartApproval` |
| `POST /api/line-items/{lineItemId}/duplicate` | `LineItemDuplicate` | `201 LineItem` (the copy) | 400 `bad_request`; 401; 404; 400 `validation_failed` on `restartApproval` |
| `DELETE /api/line-items/{lineItemId}?restartApproval=true` | — | `200 DeleteResult` | 400 (bad query value); 401; 404; 400 `validation_failed` on `restartApproval` |

The `Error.code` enum gains `validation_failed` on `restartApproval` (409) — owned by feature 006; added by this
feature's contract task if 006 has not added it.

## Frontend design

### Routes (`router.tsx`, inside the `RequireSignIn` group)

| Path | Page | `handle.title` (tab while loading) | Tab title when loaded |
|---|---|---|---|
| `/activities/:activityId/line-items/new` | `LineItemCreatePage` | New line item | New line item · NorthStar |
| `/line-items/:lineItemId` | `LineItemPage` (details) | Line item | <line item name> · NorthStar |
| `/line-items/:lineItemId/edit` | `LineItemEditPage` | Edit line item | Edit <line item name> · NorthStar |

Navigation: no sidebar change (line items live inside activities; Plans is feature 004's section). The
activity page (006, `/activities/:activityId`) embeds the section with the anchor `#line-items`; links to a
field use `/line-items/:id/edit?field=<field>` (the edit page scrolls to and focuses that field). The shared
`recordHref(ref)` (004/005/006) gains `lineItem → /line-items/<id>`. **View history** goes to
`/activity-log?recordType=lineItem&recordId=<id>` ("Coming soon" until feature 011).

### Data (`lineItemsApi.ts`)

| Hook | Request | Query key |
|---|---|---|
| `useLineItemSection(activityId)` | `GET /api/activities/{activityId}/line-items` | `['line-items', 'activity', activityId]` |
| `useLineItemFormOptions(activityId)` | `GET …/line-items/form-options` | `['line-items', 'form-options', activityId]` |
| `useLineItem(lineItemId)` | `GET /api/line-items/{lineItemId}` | `['line-items', 'item', lineItemId]` |
| `useCreateLineItem`, `useUpdateLineItem`, `useDuplicateLineItem`, `useDeleteLineItem` | the four changes | after settling: invalidate `['line-items']`, `['activities']`, `['plans']`, `['recycle-bin']`, `['activity-log']` (FR-023: balance, Ready to submit, plan completeness) |

`useUndoableDelete({ invalidate: [the same keys] })` gives the delete message and Undo. Edit forms follow
001's edit-form rule (copy the record and its `version` once; only the conflict handler replaces the
version).

### Calculations in the browser (`lineItemMath.ts`, 007-P3)

Amounts are `bigint` cents. `parseAmount(text)` (digits with an optional `.`; more than two decimals
rounded half-up; returns `null` for anything else); `formatAmount(cents | string, code)` → "10,869.57 USD"
(minus sign for negatives); `formatRate("0.920000")` → "0.92" (at least two, at most six decimals);
`formatMonth("2026-03")` → "Mar 2026"; `toUsd(cents, rate)` and `fromUsd(cents, rate)` half-up with the rate
in millionths; `activityMonths`, `spreadMonths`, `usdMonths` mirroring the Go functions; `quarterColumns(grid)`
(sums per calendar quarter, "Q1 2026"); `customGridMonths(activity, planYear, storedMonths)` (007-P13);
`spreadEffect(items, newDates)` → `{respread, customKept, customOutside, notSpread}`. The Go and TS tests
share one example table (*Business rules and their tests*).

Money from the API that is in USD (USD column, balance, meter, grid, summary bar) is shown in the person's
**display currency** through feature 003's helper and labelled with its code (FR-005); local amounts and the
"= … USD at …" line stay in their own currency.

### Line items section (`LineItemsSection`, rendered by 006's `ActivityPage`)

`Card` titled "Line items" (subtitle from the messages table), action **Add line item** (secondary — the
page's primary is 006's **Edit**) linking to `/activities/:activityId/line-items/new`.

1. **Rows** (FR-001): a table in the card (sideways scroll on narrow screens) — Name (link to the details;
   the ID under it), Currency, Planned cost (local, right-aligned), Planned cost (USD — or the display
   currency, FR-005), Spread (method label), Completeness (`CompletenessChip` with links to
   `/line-items/:id/edit?field=…`), SWBS request (`SwbsChip`), and an `ActionsMenu` "Actions for <name>":
   View · Edit · Duplicate · Delete (last, separated). A row given in `location.state.highlightLineItemId`
   is scrolled into view and highlighted (blue tint fading over 3 s; none with reduced motion).
2. **System row** (`UnassignedBalanceRow`, FR-002, FR-004, FR-023) as the table's footer: "Unassigned planned
   cost (Sys_Gen_Balance)" with `InfoTip`, the balance in the USD column (red when negative), a meter
   (feature 004's `ProgressBar`, red when exceeded; accessible name "Assigned to line items") with the
   text "<assigned> of <planned cost> USD assigned", and the state message; the rounding hint below when
   `roundingHint`.
3. **Grid** (`SpreadGrid`, FR-003, FR-020), under the heading "Spread over time" with a **Monthly ·
   Quarterly** `SegmentedControl` (remembered per browser in `localStorage['northstar.lineItemGrid']`,
   read in try/catch): `<table>` with a visually hidden caption; the first column (line-item names) sticky
   on the left; month headers "Mar 2026"; an outside month shows the small label "Outside dates" with an
   `InfoTip` "Outside the activity's dates — allowed, e.g. a deposit"; "No dates yet" column; Total column;
   totals row "Total"; empty cells "—". With no activity dates, the note "Add the activity's dates to
   spread its line items over months." stands above the grid. Hidden when there are no line items.

States: loading — the card with `Skeleton lines={3}`; error — `ErrorState` (**Try again** refetches);
empty (FR-004) — `EmptyState` with the empty sentence and **Add line item**, the system row still shown
below it; activity gone — `ErrorState` with the API's message.

### Line-item form (`LineItemForm`, shared by the create and edit pages; FR-006, FR-007)

Full page, one column (max 720 px; the custom month grid may use the content width and scroll sideways):
breadcrumb (feature 004's `Breadcrumb`: Plans › plan › campaign › initiative › activity › "New line item" or
the name), `PageHeader` (eyebrow, title, description from the messages table), the sticky **summary bar**
(`LineItemSummaryBar`: activity planned cost · unassigned now · after saving, red when negative; the
no-planned-cost sentence instead when 007-P5), then the sections in this order, each a feature 004
`FormSection`:

1. **Basics** — Line item name (`TextField`, marking Required, `maxLength` 120); Owner (003 `PersonPicker`,
   Needed to submit); Description (multi-line `TextField`, 2,000).
2. **Cost and currency** (`CostFields`) — Currency (searchable choice: active currencies "EUR – Euro"; a
   stored inactive one added as "INR – Indian rupee (inactive)" with its note), Required; Planned cost
   (002 `MoneyInput` with the currency inside), Required; under it, for a currency other than USD, the line
   "= <usd> USD at <rate> <CODE> per USD" with the label "Calculated" and an `InfoTip` giving the formula.
   **Currency change while the planned cost still holds the pre-filled balance** (FR-012): the amount
   becomes `fromUsd(balance, rate)` and the note "Changed to <amount> <CODE> = <usd> USD" appears (plus the
   rounding hint when the USD value differs from the balance). Typing in the amount ends this.
3. **Spread over time** (`SpreadEditor`) — a radio group with the four labels; the source note; for
   Evenly / First / Last a read-only preview list "Mar 2026 · 3,333.33 EUR" (FR-017); the ongoing note
   (FR-021) and the no-dates sentence; for **Custom** a month grid (FR-018): one labelled amount input per
   month ("Amount for Feb 2026"), starting from the current spread (`customGridMonths`, 007-P13), **Add month
   before** / **Add month after** (one month each), outside months marked, the running total line and
   **Use month total as planned cost** (sets the planned cost to the month total in one step). Leaving
   Custom with typed amounts shows the replacement note (007-P12).
4. **Finance coding** — 002's `ShellAttributeFields` for the level `lineItem` (spend type, GL code with its
   dependent behaviour and texts, spend category, contracted spend, any other line-item attribute, each
   with its marking), then **Contracted amount** (`MoneyInput` in the line item's currency, optional).
5. **Link to finance budget** (`FinanceBudgetFields`, FR-015, FR-016) — Finance budget year (choice of
   `financeBudgetYears`, Needed to submit, note "The plan's year"); Finance budget item (searchable on item,
   budget or code; options of the chosen year grouped by budget, the plan's default budget first; Needed
   to submit; pre-selection note or the "listed first" hint; the no-items sentence for an empty year; a
   stored inactive item shown "(inactive)" with its problem message; after a year change the
   "will be cleared" note, and the item is sent as `null`).
6. **GSAP request** — **Request SWBS code from GSAP** (checkbox, note "On by default"), help text; SWBS code
   shown read-only "Filled by GSAP later".
7. **From finance (coming later)** (`FromFinanceSection`) — a native `<details>` closed by default: Total
   actuals, % of actual cost, Actuals at monthly rate, Total invoiced, Total purchase requests, each "—",
   with "Filled when the Finance module and GSAP are connected" (FR-025).

Sticky footer (004's `FormFooter`): **Save** (primary, never disabled; "Saving…" while busy), **Save and
add another** (create page only), **Cancel**. Default labels (source notes) are shown in `--ink-2` under each
pre-filled field until the person changes it. Markings are words (Required, Needed to submit).

On **Save**: build the request (`lineItemFormState.toCreateBody` / `toUpdateBody`; empty amount → `null`);
when `activity.restartsApproval` and the change needs it (`needsRestartConfirmation`: create, or a changed
planned cost or currency), open `RestartApprovalDialog` listing `restartReasons(...)` first and send with
`restartApproval: true` only after **Save and restart approval**. A `400 validation_failed` on `restartApproval`
anyway (someone submitted meanwhile; recognised by `isRestartRefusal(error)` in `lineItemFormState.ts`)
opens the same dialog. A `400 validation_failed` puts every problem in
`FormErrorSummary` (focused) and under its field (API field → form field id: `name`, `ownerId`,
`description`, `currency`, `plannedCost`, `spread.months[i].*` → that month's input,
`attributes.<id>`, `contractedAmount`, `financeBudgetYear`, `financeBudgetItemId`; `version` → summary only).
Unsaved changes are protected with `useUnsavedChanges`.

### Pages

| Page | Content | States |
|---|---|---|
| `LineItemCreatePage` (US2) | `useLineItemFormOptions`; the form with `defaults` (`newFormState`). After **Save**: toast "Line item '<name>' created", navigate to `/activities/:activityId#line-items` with `state.highlightLineItemId`. After **Save and add another**: the same toast, refetch the options and reset the form (the planned cost is the new balance, FR-010), focus the name field, stay on the page. **Cancel** → the activity page (guarded) | loading (`Skeleton`); error (`ErrorState`); activity deleted (404 → the message with **Go to the recycle bin** `/recycle-bin`); validation; restart dialog; busy |
| `LineItemPage` (US4, FR-026) | breadcrumb; `<h1>` name; `LevelTag` "LINE ITEM"; `CompletenessChip` and `SwbsChip`; facts row: ID · Owner · planned cost ("46,000.00 EUR = 50,000.00 USD at 0.92 EUR per USD", or "50,000.00 USD") · spread label; "Created by … on …, last changed by … on …"; actions **Edit** (primary, to `/edit`) and **More** (`ActionsMenu`: Duplicate, View history, separator, Delete); then the read-only sections in form order (empty values "—"; Shell attributes through 002's read-only `ShellAttributeValues`; the spread's month list with outside marks and, for custom, the total line; SWBS code "Filled by GSAP later"; `FromFinanceSection` collapsed). **Duplicate** → `RestartApprovalDialog` first when `restartsApproval`; then toast "Line item '<copy name>' created" and navigate to the copy. **Delete** → `DeleteLineItemDialog`; then navigate to the activity page and `afterDelete(result)` (Undo) | loading; error; not found (the two 007-P15 messages, with **Go to Plans** / **Go to the recycle bin**); restart dialog; delete busy |
| `LineItemEditPage` (US4, FR-027) | `useLineItem` + `useLineItemFormOptions(activityId)`; the form with `editFormState(lineItem)`; the summary bar's "Unassigned now" is the current balance, "After saving" = balance + saved USD − new USD. **Save**: no restart dialog unless cost or currency changed on a submitted/approved activity (FR-032); toast "Line item '<name>' saved"; navigate to the details. `?field=` focuses that field | loading; error; not found / activity deleted (007-P15, also when a save answers 404); validation; **conflict** (`conflictMessage(error, 'line item')` at the top, values kept, the next Save uses the refreshed version); restart dialog; unsaved guard |

**Dialogs**: `RestartApprovalDialog` (`ConfirmDialog`, confirm **Save and restart approval**, cancel **Keep
editing**, the list of changes) and `DeleteLineItemDialog` (`ConfirmDialog` `tone="danger"`, **Cancel**
focused, **Delete line item**; texts in the messages table, the re-approval line only when
`restartsApproval`; it sends `restartApproval=true` exactly when it showed that line, and reopens with
the line on a `400 validation_failed` on `restartApproval`).

**`SpreadEffectNote`** (for 006's activity edit form, 007-P11): props `{ activityId, startDate, endDate,
ongoing }` (the values in the form); reads `useLineItemSection(activityId)`; renders nothing when the dates
equal the saved ones or there are no line items; otherwise the sentence of `spreadEffectText(...)` (also
exported for 006's re-approval dialog list).

### Shared components used

From `@/components` (001): `Button`, `LinkButton`, `PageHeader`, `Card`, `StatusChip` (via `SwbsChip`),
`CompletenessChip`, `InfoTip`, `Tooltip`, `ActionsMenu`, `ConfirmDialog`, `useToast`, `EmptyState`,
`Skeleton`, `FormErrorSummary`, `TextField`, `ErrorState`; hooks `useUndoableDelete`, `useUnsavedChanges`,
`usePageTitle`. Expected from earlier features (see *Dependencies*): `MoneyInput`, `ChoiceField`,
`ShellAttributeFields` / `ShellAttributeValues` (002); `PersonPicker`, display-currency helper (003);
`Breadcrumb`, `LevelTag`, `ProgressBar`, `SegmentedControl`, `FormSection`, `FormFooter`, `recordHref`
(004/005). **No new shared component**: everything else is feature-local, because no other feature shows a
spread editor or a line-item row.

## User-visible messages

Every text appears here once; code and tests quote it (`lineItemText.ts` holds the web texts). `<…>` is
filled in at run time; amounts follow 007-P4 ("20,000.00 USD").

| Where | Text |
|---|---|
| Section card | Line items · The costs of this activity, each in its own currency · **Add line item** |
| Section columns | Name · Currency · Planned cost · Planned cost (USD) (with a display currency: Planned cost (<CODE>)) · Spread · Completeness · SWBS request |
| Row actions | Actions for <name> · View · Edit · Duplicate · Delete |
| Empty section (FR-004) | No line items yet — split this activity's planned cost into the costs you will pay |
| System row (FR-002) | Unassigned planned cost (Sys_Gen_Balance) · ⓘ Unassigned planned cost = the activity's planned cost − the sum of its line items' planned costs in USD. Aim for 0.00: line items above the planned cost stop the activity from being submitted. |
| Meter | <assigned> of <planned cost> USD assigned (e.g. "100,000.00 of 120,000.00 USD assigned") |
| Balance messages | <X> not yet assigned to a line item · All of the planned cost is assigned · Line items exceed the activity's planned cost by <X> · This activity has no planned cost yet — add one to see how much is left to assign. |
| Rounding hint | If a currency rate leaves a few cents unassigned, adjust the activity's planned cost instead. |
| Grid | Spread over time · Monthly · Quarterly · caption "Line items spread over time, in USD" (display currency: "in <CODE>") · Line item · Mar 2026 · Q1 2026 · No dates yet · Total · Outside dates · ⓘ Outside the activity's dates — allowed, e.g. a deposit · Add the activity's dates to spread its line items over months. |
| Spread labels | Evenly · All in first month · All in last month · Custom |
| SWBS chips (FR-024) | Not requested · Waiting for activity approval · Ready to send to GSAP |
| Completeness chip | All filled in · 1 field needed · <n> fields needed |
| Completeness labels | Owner · <attribute label> · Finance budget year · Finance budget item · Choose an active finance budget item — '<name>' is no longer active. · Spread doesn't add up to the planned cost |
| Create page | eyebrow LINE ITEM · New line item · In activity '<activity name>' (<activity id>) · breadcrumb end "New line item" |
| Edit page | eyebrow LINE ITEM · Edit '<name>' · In activity '<activity name>' (<activity id>) |
| Summary bar | Activity planned cost <X> · Unassigned now <Y> · After saving <Z> · (no planned cost) This activity has no planned cost yet — the unassigned planned cost appears once it has one. |
| Section headings | Basics · Cost and currency · Spread over time · Finance coding · Link to finance budget · GSAP request · From finance (coming later) |
| Field labels | Line item name · Owner · Description · Currency · Planned cost · Spread over time · Contracted amount · Finance budget year · Finance budget item · Request SWBS code from GSAP · SWBS code |
| Markings | Required · Needed to submit |
| Default sources | From the activity's name · From the activity · Default · The activity's unassigned planned cost · The plan's default · The plan's year · Pre-selected — the only active item of '<budget>' · On by default |
| Planned cost notes | Nothing left to assign — the activity's planned cost is fully assigned · This activity has no planned cost yet, so there is nothing to pre-fill · Rounded to <amount> (002 `MoneyInput`) |
| Conversion (FR-011, FR-012) | = <usd> USD at <rate> <CODE> per USD · Calculated · ⓘ Planned cost in USD = planned cost ÷ the rate in Settings (units of <CODE> per 1 USD), rounded to the cent. · Changed to <amount> <CODE> = <usd> USD |
| Inactive currency | <CODE> – <name> (inactive) · <CODE> is no longer active — it still converts at its last rate (<rate> per USD). |
| Spread notes (FR-017, FR-021) | <Mon YYYY> · <amount> <CODE> · Not spread yet — add the activity's dates to see the spread · This activity is ongoing (no end date), so 'Evenly' puts everything in the first month — choose Custom to spread it · This activity is ongoing (no end date), so 'All in last month' puts everything in the first month — choose Custom to spread it |
| Custom grid (FR-018) | Amount for <Mon YYYY> · **Add month before** · **Add month after** · Outside the activity's dates — allowed, e.g. a deposit · <sum> of <cost> <CODE> spread — <X> <CODE> left · <sum> of <cost> <CODE> spread — <X> <CODE> too much · <sum> of <cost> <CODE> spread — adds up · **Use month total as planned cost** |
| Leaving Custom | The month amounts you typed will be replaced by this spread when you save. |
| Finance coding (002 texts, FR-014) | Choose a spend type first — GL codes depend on it · The only GL code for this spend type · GL code '<label>' will be cleared — it doesn't fit '<spend type>' |
| Finance budget (FR-015, FR-016) | Search item, budget or code · Items of '<budget>', the plan's default finance budget, are listed first. · No finance budget items for <year> yet — an administrator adds them in Settings. · Finance budget item '<name>' will be cleared — it belongs to <year>. · <name> (inactive) |
| GSAP request | NorthStar asks GSAP for an SWBS code once the activity is approved. Sending requests to GSAP comes later. · Filled by GSAP later |
| From finance (FR-025) | Total actuals · % of actual cost · Actuals at monthly rate · Total invoiced · Total purchase requests · — · Filled when the Finance module and GSAP are connected |
| Footer | **Save** · **Save and add another** · **Cancel** · Saving… |
| Toasts | Line item '<name>' created · Line item '<name>' saved · (delete, 001) Line item '<name>' moved to the recycle bin · **Undo** · Line item '<name>' restored |
| Restart dialog (FR-031) | This activity will need approval again · '<activity name>' is submitted for approval. (approved activity: '<activity name>' is approved.) Saving sends it back to Created, and it must be submitted for approval again. · New line item '<name>': <amount> · Copy of '<name>': <amount> · Planned cost of '<name>': <old> → <new> · **Save and restart approval** · **Keep editing** |
| Delete dialog (FR-030) | Delete line item '<name>'? · The activity's unassigned planned cost goes up by <usd> USD. · It moves to the recycle bin; you can restore it. · This activity will need approval again. · **Cancel** · **Delete line item** |
| Details page | LINE ITEM · ID · Owner · Created by <person> on <date>, last changed by <person> on <date> · **Edit** · **More** · Duplicate · View history · Delete · — |
| Gone | This line item no longer exists — someone may have deleted it. · **Go to Plans** · This activity was deleted — restore it from the recycle bin to keep working on its line items. · **Go to the recycle bin** |
| Spread effect (FR-022, 007-P11) | When you save, <n> line item(s) <is/are> re-spread over the new dates. · When you save, <n> custom spread(s) <is/are> kept as typed. · (both) When you save, <n> line item(s) <is/are> re-spread over the new dates and <m> custom spread(s) <is/are> kept as typed. · <k> custom spread(s) will have months outside the activity's dates. · Without dates, <n> line item(s) won't be spread until the activity has dates again. |
| API (see *Validation messages*) | 404 / 409 texts and field messages listed there |
| History notes | Copied from '<name>' (<id>) · Renamed to '<new name>' because the name was taken. · Can't undo — there is now another line item called '<name>'. Restore it from the recycle bin instead. · Activity '<activity name>' needs approval again because a line item came back. |
| Restart reasons (sent to 006) | Line item '<name>' (<id>) added: <amount> <CUR> · Line item '<name>' (<id>) deleted: <amount> <CUR> · Line item '<name>' (<id>) restored: <amount> <CUR> · Planned cost of line item '<name>' (<id>): <old> <CUR> → <new> <CUR> |

## Business rules and their tests (constitution V)

Shared example table (Go `spread_test.go`/`balance_test.go` and TS `lineItemMath.test.ts` use the same
rows): 46,000.00 EUR @ 0.92 → 50,000.00 USD; 10,000.00 EUR @ 0.92 → 10,869.57 USD; 20,000.00 USD → 18,400.00
EUR @ 0.92; 15,800.00 GBP @ 0.79 → 20,000.00 USD; 10,000.00 over 3 months → 3,333.33 / 3,333.33 / 3,333.34;
10,000.00 EUR evenly Mar–May @ 0.92 → 3,623.18 / 3,623.18 / 3,623.21 USD; 0.00 over 3 → 0.00 each; 0.02 over
3 → 0.00 / 0.00 / 0.02; 1,234.565 → 1,234.57.

| Rule | Go test (`backend/internal/lineitems/…`) |
|---|---|
| Activity months; ongoing; no dates | `spread_test.go` `TestActivityMonths_CalendarMonths_FR017`, `TestActivityMonths_OngoingAndNoDates_FR021` |
| Evenly floors to the cent, rest in the last month; first/last; 0.00 | `TestSpread_EvenlyRemainderLast_FR017_SC002`, `TestSpread_FirstAndLastMonth_FR017` |
| Ongoing: Evenly and Last behave like First | `TestSpread_OngoingLikeFirst_FR021` |
| No dates: not spread; custom kept | `TestSpread_NoDatesNotSpread_FR017` |
| Custom kept; outside months marked; follows dates and cost changes for E/F/L | `TestSpread_CustomOutsideMarked_FR018`, `TestSpread_FollowsDatesAndCost_FR022` |
| USD months add up exactly to the USD target, rounding cent in the last month with an amount, for amounts × rates (0.92, 0.79, 7.10, 83.50, 1) × 1–36 months | `TestUSDMonths_AddUp_FR020_SC002`, `TestUSDMonths_CentInLastNonZeroMonth_FR020` |
| Conversion half-up at the current rate | `balance_test.go` `TestPlannedCostUSD_HalfUp_FR011_SC003` |
| Balance = planned cost − USD line items; exceeded; all assigned; no planned cost; rounding hint | `TestBalance_Formula_FR002_SC004`, `TestBalance_States_FR002`, `TestBalance_NoPlannedCost_007P5`, `TestBalance_RoundingHint_EdgeCase` |
| Grid columns (activity months ∪ custom months, outside flags), No dates column, totals | `grid_test.go` `TestGrid_Columns_FR003`, `TestGrid_NoDatesColumn_D032`, `TestGrid_Totals_FR003_FR020` |
| Missing list: order, labels, attributes from Settings, inactive item, custom not adding up | `completeness_test.go` `TestMissing_NeededToSubmit_FR028`, `TestMissing_Attributes_FR014`, `TestMissing_InactiveItem_FR016`, `TestMissing_CustomNotAddingUp_FR019` |
| SWBS status by flag and activity status | `TestSwbsStatus_FR024` |
| Section: rows in creation order with views, balance and grid; activity gone; rate change updates USD and balance without writing and without status change | `service_test.go` `TestSection_RowsInCreationOrder_FR001`, `TestSection_ActivityDeleted_EdgeCase`, `TestSection_RateChange_FR013` |
| Defaults with sources; " 2" when the name is taken; nothing left; no planned cost; pre-selection only with one active item; year and item choices and order | `TestFormOptions_Defaults_FR008`, `TestFormOptions_NameTaken_FR008`, `TestFormOptions_NothingLeft_EdgeCase`, `TestFormOptions_BudgetChoices_FR015`, `TestFormOptions_Currencies_FR011` |
| Only name, currency and planned cost needed; every problem at once (empty name, negative cost, negative month, duplicate name with another problem) | `TestCreate_OnlyRequired_FR009`, `TestCreate_AllProblemsAtOnce_FR009_SC005` |
| Name unique ignoring case and spaces; 120 characters | `TestCreate_NameUnique_FR009` |
| Rounding half-up to the cent | `TestCreate_RoundsToCent_EdgeCase` |
| Readable ID L-…, log `created` with path | `TestCreate_IDAndLog_FR010` |
| New currency must be active; stored inactive currency kept | `TestCreate_CurrencyActive_FR011`, `TestUpdate_KeepsInactiveValues_FR011_FR016` |
| Attribute problems passed through; contracted amount ≥ 0 | `TestCreate_AttributeProblems_FR014`, `TestCreate_ContractedAmount_FR014` |
| Finance budget year and item checks | `TestCreate_FinanceBudgetChecks_FR015_FR016` |
| Custom months: range, once, ≥ 0; a custom spread that doesn't add up still saves | `TestCreate_CustomMonthChecks_FR018`, `TestCreate_CustomNotAddingUpSaves_FR019` |
| Submitted/approved: create, cost/currency change, duplicate, delete need confirmation; then `RestartApproval` with the reason; other edits never restart | `TestCreate_RestartNeedsConfirmation_FR031`, `TestUpdate_CostOrCurrencyRestart_FR031`, `TestUpdate_OtherEditsKeepStatus_FR032_SC007`, `TestDuplicate_RestartNeedsConfirmation_FR031`, `TestDelete_RestartNeedsConfirmation_FR031` |
| Update: version missing / conflict; no change → no version, no log; every changed field logged old → new | `TestUpdate_Version_FR027`, `TestUpdate_NoChangeNoLog_FR027`, `TestUpdate_LogsOldAndNew_FR027` |
| Leaving Custom drops months; cost change keeps custom months | `TestUpdate_LeavingCustomDropsMonths_FR018`, `TestUpdate_CostChangeKeepsCustom_FR022` |
| Duplicate: all values, new ID, " (copy)", " (copy 2)", shortened to 120, log `copied` | `TestDuplicate_CopiesEverything_FR029` |
| Delete → bin entry, log `deleted`; missing line item messages | `TestDelete_MovesToBin_FR030`, `TestGet_GoneMessages_007P15` |
| Undo restores same ID and values; does not restore an ended approval; restore into approved activity restarts it; Undo refused when the name is taken; Restore renames | `kinds_test.go` `TestUndo_RestoresExactly_FR030`, `TestUndo_EndedApprovalStaysEnded_FR033`, `TestRestore_IntoApprovedRestarts_FR033`, `TestUndo_NameTaken_D037`, `TestRestore_NameTakenSuffix_FR030` |
| Deleting the activity takes its line items ("with 2 line items") | `TestDeleteActivity_TakesLineItems` |
| Readable log texts (owner name, item path, spread, attributes) | `TestDescribe_LogTexts_FR027` |
| Summaries for Ready to submit; copies same year and next year; usage incl. recycle bin; rate impact; owner usage | `provided_test.go` `TestSummaries_ReadyToSubmit_FR023_FR028_SC005`, `TestCopyLineItems_SameYear`, `TestCopyLineItems_NextYear_007P9`, `TestUsage_IncludesRecycleBin`, `TestRateChangeImpact_FR013`, `TestOwnerUsage` |
| `AfterRestore` runs once for the deleted record; notes in log and result; an error changes nothing | `backend/internal/history/service_test.go` `TestAfterRestore_RunsOnceForDeletedRecord_FR033`, `TestAfterRestore_Notes`, `TestAfterRestore_ErrorChangesNothing` |
| Seed: eight line items, Launch film balance 20,000.00, Mobility activity exceeded by 500.00, IDs L-1001… | `backend/seed/seed_test.go` `TestLoad_LineItems` |

**Contract tests** (`backend/internal/lineitems/handler_test.go`, `package lineitems_test`, `apitest.New` with
the seed): `TestListActivityLineItems_Contract` (Launch film: 2 rows, balance 20000.00, grid Mar–May; unknown
activity 404), `TestGetLineItemFormOptions_Contract`, `TestCreateLineItem_Contract` (201; 400 with every
problem; 404; 409 on the seeded Approved activity), `TestGetLineItem_Contract` (200; 404),
`TestUpdateLineItem_Contract` (200; 400 missing version; 409 `version_conflict`; 409
`validation_failed` on `restartApproval`), `TestDuplicateLineItem_Contract` (201; 409), `TestDeleteLineItem_Contract`
(200 `DeleteResult`, then `POST /api/recycle-bin/{id}/undo` 200; 404; 409). 001's `TestRoutesMatchContract`
and `TestEveryProtectedOperationRequiresSignIn_FR005` cover the new routes.

**Web tests** (Vitest, next to the code): `lineItemMath.test.ts` (the example table, parse/round, quarters,
custom grid months, spread effect), `lineItemText.test.ts` (plurals, money and rate texts),
`lineItemFormState.test.ts` (defaults → state, FR-012 conversion, typed amounts not converted, leaving
Custom, year change clears the item, request bodies, `needsRestartConfirmation` and reasons),
`LineItemsSection.test.tsx` (US1 scenarios 1–5, empty state, display currency label, highlight, loading,
error), `SpreadGrid.test.tsx` (quarterly, outside marks, No dates column, sticky names), `SpreadEditor.test.tsx`
(US3 scenarios 1, 2, 4, 6), `FinanceBudgetFields.test.tsx` (FR-015, FR-016, empty year),
`LineItemCreatePage.test.tsx` (US2 scenarios 1–7, states), `LineItemPage.test.tsx` (US4 scenarios 1, 4, 5,
FR-025, not found), `LineItemEditPage.test.tsx` (US4 scenarios 2–3, US3 scenario 3, conflict, deleted
activity), `SpreadEffectNote.test.tsx` (FR-022), `router.test.tsx` (the three routes and titles).

## End-to-end tests (`frontend/e2e/line-items.spec.ts`)

As `smoke.spec.ts`: real API with fresh seed per run, `signInAs`, Chromium; a new helper
`openActivity(page, planName, activityName)` in `e2e/fixtures.ts` opens the plan from **Plans**, searches the
plan tree ("Search name or ID in this plan") and opens the activity. Tests restore what they change where
they can (US2 and US3 delete the line items they add); US4 changes the seeded Approved activity's status,
which no other end-to-end file reads.

| Test | Covers |
|---|---|
| `US1: the Line items section shows costs, the unassigned planned cost and the grid` | Priya opens Launch film: Agency fee 50,000.00 USD, Film production 46,000.00 EUR / 50,000.00 USD, "3 fields needed", "Waiting for activity approval"; system row 20,000.00 USD, "100,000.00 of 120,000.00 USD assigned"; grid Mar–May with totals; Quarterly → Q1 2026, Q2 2026 (FR-001–FR-004, FR-024) |
| `US2: add a line item with its defaults` | **Add line item** → defaults and sources, summary bar "… After saving 0.00 USD"; EUR 10,000 → "= 10,869.57 USD at 0.92 EUR per USD"; empty name + "-50" → all problems at once; spend type with one GL code fills it; save → toast, highlighted row, "All of the planned cost is assigned"; **Save and add another** shows the new balance; clean-up delete (FR-006–FR-012, FR-014) |
| `US3: spread a line item over time` | 10,000.00 EUR: Evenly preview 3,333.33 / 3,333.33 / 3,333.34; Custom + Feb 2,000.00 → "Outside the activity's dates…" and "12,000.00 of 10,000.00 EUR spread — 2,000.00 EUR too much"; save → "Spread doesn't add up to the planned cost" in the completeness list; edit → **Use month total as planned cost** → 12,000.00; clean-up (FR-017–FR-019) |
| `US4: edit, duplicate and delete on an approved activity` | on the seeded Approved activity change Venue hire's finance budget item (Europe – Events → Europe – Brand; its spend type has a single GL code, so the GL-code case is covered by `LineItemEditPage.test.tsx`) → no dialog, activity still Approved; change the planned cost → "This activity will need approval again" → **Save and restart approval** → activity Created; Duplicate → "(copy)" page; Delete → dialog texts → toast → **Undo** → back with the same ID (FR-026–FR-033) |
| `Accessibility: line-item screens pass axe` | axe (WCAG 2.2 A/AA) on the section, the create form with its error summary, the custom grid, the details page, the restart and delete dialogs (SC-006 basis) |

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 | `Section`, `LineItemsSection` | `TestSection_RowsInCreationOrder_FR001`, `LineItemsSection.test.tsx`, e2e US1 |
| FR-002 | `Balance`, `UnassignedBalanceRow` | `TestBalance_*`, `LineItemsSection.test.tsx`, e2e US1 |
| FR-003 | `Grid`, `SpreadGrid`, `quarterColumns` | `grid_test.go`, `SpreadGrid.test.tsx`, `lineItemMath.test.ts`, e2e US1 |
| FR-004 | `LineItemsSection` empty state | `LineItemsSection.test.tsx` |
| FR-005 | display-currency helper (003) in section, bar, grid | `LineItemsSection.test.tsx` (EUR display) |
| FR-006, FR-007 | `LineItemCreatePage`, `LineItemForm`, `LineItemSummaryBar` | `LineItemCreatePage.test.tsx`, e2e US2 |
| FR-008 | `FormOptions`, `newFormState` | `TestFormOptions_*`, `lineItemFormState.test.ts`, `LineItemCreatePage.test.tsx` |
| FR-009 | `validate.go`, `Create` | `TestCreate_OnlyRequired_FR009`, `TestCreate_AllProblemsAtOnce_FR009_SC005`, `TestCreate_NameUnique_FR009`, e2e US2 |
| FR-010 | `Create`, create page navigation | `TestCreate_IDAndLog_FR010`, `LineItemCreatePage.test.tsx` |
| FR-011 | `PlannedCostUSD`, `CostFields`, currency checks | `TestPlannedCostUSD_HalfUp_FR011_SC003`, `TestCreate_CurrencyActive_FR011`, `lineItemMath.test.ts` |
| FR-012 | `changeCurrency` | `lineItemFormState.test.ts`, `LineItemCreatePage.test.tsx` |
| FR-013 | 007-P2 (calculated on read) | `TestSection_RateChange_FR013`, `TestRateChangeImpact_FR013` |
| FR-014 | `Attributes` adapter, `ShellAttributeFields`, contracted amount | `TestCreate_AttributeProblems_FR014`, `TestMissing_Attributes_FR014`, `TestCreate_ContractedAmount_FR014`, `LineItemCreatePage.test.tsx`, e2e US2 |
| FR-015 | `FormOptions`, `FinanceBudgetFields`, checks | `TestFormOptions_BudgetChoices_FR015`, `TestCreate_FinanceBudgetChecks_FR015_FR016`, `FinanceBudgetFields.test.tsx` |
| FR-016 | `Missing`, update keeps inactive item | `TestMissing_InactiveItem_FR016`, `TestUpdate_KeepsInactiveValues_FR011_FR016`, `FinanceBudgetFields.test.tsx` |
| FR-017 | `SpreadMonths`, `SpreadEditor` preview | `spread_test.go`, `SpreadEditor.test.tsx`, e2e US3 |
| FR-018 | custom months, `SpreadEditor` grid | `TestCreate_CustomMonthChecks_FR018`, `TestUpdate_LeavingCustomDropsMonths_FR018`, `SpreadEditor.test.tsx`, e2e US3 |
| FR-019 | `Missing` (spread) | `TestMissing_CustomNotAddingUp_FR019`, `TestCreate_CustomNotAddingUpSaves_FR019`, e2e US3 |
| FR-020 | `USDMonths`, grid | `TestUSDMonths_*`, `TestGrid_Totals_FR003_FR020` |
| FR-021 | ongoing rules, `customGridMonths` | `TestSpread_OngoingLikeFirst_FR021`, `lineItemMath.test.ts`, `SpreadEditor.test.tsx` |
| FR-022 | 007-P1, `SpreadEffectNote`, update keeps custom | `TestSpread_FollowsDatesAndCost_FR022`, `TestUpdate_CostChangeKeepsCustom_FR022`, `SpreadEffectNote.test.tsx` |
| FR-023 | invalidation after every change; `Summaries` | `LineItemCreatePage.test.tsx` (invalidations), `TestSummaries_ReadyToSubmit_FR023_FR028_SC005` |
| FR-024 | `SwbsStatus`, `SwbsChip`, GSAP section | `TestSwbsStatus_FR024`, `LineItemsSection.test.tsx`, e2e US1 |
| FR-025 | `FromFinanceSection` | `LineItemPage.test.tsx`, `LineItemCreatePage.test.tsx` |
| FR-026 | `Get`, `LineItemPage` | `TestGet_*`, `LineItemPage.test.tsx`, e2e US4 |
| FR-027 | `Update`, `LineItemEditPage` | `TestUpdate_*_FR027`, `TestDescribe_LogTexts_FR027`, `LineItemEditPage.test.tsx` |
| FR-028 | `Missing`, `CompletenessChip`, `Summaries` | `completeness_test.go`, `TestSummaries_*`, `LineItemsSection.test.tsx` |
| FR-029 | `Duplicate` | `TestDuplicate_CopiesEverything_FR029`, `LineItemPage.test.tsx`, e2e US4 |
| FR-030 | `Delete`, kind, `DeleteLineItemDialog`, Undo | `TestDelete_MovesToBin_FR030`, `TestUndo_RestoresExactly_FR030`, `LineItemPage.test.tsx`, e2e US4 |
| FR-031 | confirmation flag (007-P7), `RestartApprovalDialog` | `TestCreate_RestartNeedsConfirmation_FR031` and siblings, `LineItemEditPage.test.tsx`, e2e US4 |
| FR-032 | update rules | `TestUpdate_OtherEditsKeepStatus_FR032_SC007`, `LineItemEditPage.test.tsx`, e2e US4 |
| FR-033 | `AfterRestore` (007-P8) | `TestUndo_EndedApprovalStaysEnded_FR033`, `TestRestore_IntoApprovedRestarts_FR033`, `TestAfterRestore_RunsOnceForDeletedRecord_FR033` |

## Dependencies

### Expects from other features (by name; the consolidation step aligns names that differ)

| Feature | What |
|---|---|
| 001 | `history.Service` (`Register`, `Log`, `Diff`, `MoveToBin`, `ListBin`), `history.Kind` (+ the new `AfterRestore`), `errs`, `record.Meta`, `ids.Generator`, `store`, `web`, `apitest`; OpenAPI `Error` responses, `DeleteResult`, `RecordRef`, `ActorRef`; web `@/components` (see *Shared components used*), `useUndoableDelete`, `useUnsavedChanges`, `usePageTitle`, `conflictMessage`, `mockApi`, `renderRoutes` |
| 002 | Go `backend/internal/platform/money`: `money.Amount` (exact two-decimal amount, JSON decimal string; `ParseAmount` rounding half-up to the cent; `FromCents`, `Cents`, `Add`, `Sub`, `Cmp`, `Sign`, `String`), `money.Rate` (`ParseRate`, `String`), `money.ToUSD(Amount, Rate)` and `money.FromUSD(Amount, Rate)` (half-up), `money.Money{Amount, Currency}` (JSON `{amount, currency}`), `money.Format(Amount)` ("46,000.00"). Go `attributes.Values` and `attributes.Context` and functions to check line-item values (R11), list missing needed-to-submit attributes (D-034, R9) and describe changes — used through the `lineitems.Attributes` adapter. Read access to currencies (by code, active list) and finance budget items (by id, active items of a year, years with active items). Hook points that take the line-item usage counters (currency, finance budget item, attribute option) and the rate-change impact. OpenAPI `Money` (amount may be negative), `AttributeValues`. Web `MoneyInput` (currency inside, "12.5k", "Rounded to …"), searchable `ChoiceField`, `ShellAttributeFields` (level `lineItem`, dependent options with the texts of FR-014, clear notes) and read-only `ShellAttributeValues`. Seed: currencies USD, EUR 0.92, GBP 0.79, CNY 7.10, INR 83.50; 2026 finance budget items; line-item attributes Spend type, GL code (depends on Spend type), Spend category, Contracted spend with options including "Digital – Platform development" → "7250080 – Advertising digital" and "BTL – Event" with exactly one GL code |
| 003 | OpenAPI `PersonRef`; web `PersonPicker` (single); a people lookup (name, active) for the adapter; the display-currency helper (USD amount → display currency and code, half-up); a hook point for owner counts (003 FR-004) |
| 004 | OpenAPI `SpreadMethod` (`evenly`, `firstMonth`, `lastMonth`, `custom` — the plan's default line-item spread) and `MissingField` (`{field, label}`); plan facts: year, default line-item spread, default finance budget (budget name of the plan's year), business unit; web `Breadcrumb`, `LevelTag` (level `lineItem`), `ProgressBar`, `SegmentedControl`, `FormSection`, `FormFooter`, `recordHref`; plan copy calls `CopyLineItems` (yearShift 1 for next year); plan completeness uses `Summaries` |
| 005 | campaign/initiative copy calls `CopyLineItems`; completeness of campaigns and initiatives uses `Summaries`; `recordHref` for campaigns and initiatives |
| 006 | Go activity facts for the adapter (id, name, stored status, owner, dates, ongoing, planned cost USD, path, plan id); `RestartApproval(ctx, a, activityID, reasons []string) error` joining the caller's Write (with 008 ending the round); history kind `activity` registered before `lineItem`; a way to receive the line-item provider after construction (`Summaries`, `CopyLineItems`; expected `activities.Service.UseLineItems(...)`); OpenAPI `ActivityStatus`, parameter `ActivityId`, the `Error.code` value `validation_failed` on `restartApproval` and the request flag `restartApproval` (007-P7). Web route `/activities/:activityId`; `ActivityPage` with a Line items slot (anchor `line-items`) that renders `<LineItemsSection activityId>`; the activity edit form renders `<SpreadEffectNote>` under the dates and lists `spreadEffectText(...)` in its re-approval dialog; query keys starting with `['activities']`; its Ready to submit panel links line-item problems to `/line-items/:id/edit?field=<field>` |
| 008 | ends the open approval round when 006's `RestartApproval` runs (008 FR-022) |
| 011 | `/activity-log?recordType=lineItem&recordId=<id>` shows the line item's history; its recycle-bin restore dialog states the FR-033 status change (011 FR-020) |

### Alignment with the designs of 002, 004 and 006

The plans of 002, 004 and 006 were written alongside this one and already name the seams line items plug
into. `lineitems` keeps its own functions (above); `internal/app/lineitems.go` adapts them to those seams,
so only the adapter changes if the consolidation step renames anything.

| Seam (owner) | How 007 plugs in |
|---|---|
| `settings.ReferenceSource` (D-042) | A source returning one `settings.Reference` per line item, live and in the recycle bin (`InBin`), with `Record`, `Level "lineItem"`, `PlanID`, `ActivityID`, `Currency`, `PlannedCost`, `FinanceBudgetItemID`, `Attributes`. 002 computes currency, finance-budget-item and attribute usage and the rate-change impact from it; `CurrencyUsage`, `FinanceBudgetItemUsage`, `AttributeUsage` and `RateChangeImpact` then stay internal helpers for tests, or are dropped if the source covers them |
| `plans.Level` (004) | `RegisterLevel(plans.Level{Name: "lineItem", AddFacts, Copy})`, registered after 006's `activity`. `AddFacts` adds `LineItems` counts and the incomplete line items as `AttentionItem`s (`BlocksPublish` false) for many plans at once; `Copy` calls `CopyLineItems` for every activity in `job.NewIDs` with `yearShift = job.Years`, adds the new IDs and `job.Counts["lineItem"]`, and writes no log entries |
| `activities.LineItems` + `UseLineItems` (006) | `Facts(activityIDs)` ← `Summaries` (count, USD sum, problems as `LineItemProblem{LineItemID, LineItemName, Code, Field, Message}` with `Field` = the `MissingField.field` of data-model §3.6, and the counts of auto-spread and custom-spread line items for 006's re-approval dialog); `CopyAll` ← `CopyLineItems` (006's `CopyAll` needs a `yearShift` argument for 004's next-year copy, 007-P9); `DatesChanged` ← no-op, because spreads are calculated on read (007-P1) |
| Re-approval (006) | 006 restarts approval inside its own `Update` and tells 008 through `ApprovalHooks.ApprovalRestarted`; 007 needs that step exported as `RestartApproval(ctx, a, activityID, reasons)` (same Write, same 008 hook, one status entry). The confirmation flag is `restartApproval` + `400 validation_failed` on `restartApproval` here and `restartApproval` + `400 validation_failed` in 006: one convention for both. 007's handling sits in one place on each side (the service's confirmation check, the web app's `isRestartRefusal(error)` in `lineItemFormState.ts`), so a switch is a rename |

### Provides to later features

| Who | What |
|---|---|
| 004, 005, 006, 008, 009 | `lineitems.Service.Summaries` (count, balance, incomplete line items with their missing lists) |
| 004, 005, 006, 012 | `lineitems.Service.CopyLineItems` (same year / next year, 007-P9) |
| 002, 003 | `CurrencyUsage`, `FinanceBudgetItemUsage`, `AttributeUsage`, `OwnerUsage`, `RateChangeImpact` |
| 010, 013 | `lineitems.ActivityMonths`, `SpreadMonths`, `USDMonths` (the spread rules of domain model §4.4 and D-024) and the `line_items` repository through `lineitems.Service` reads |
| 006 (web) | `LineItemsSection`, `SpreadEffectNote`, `spreadEffectText`, the query-key prefix `['line-items']` |
| 008, 009, 010, 011, 012 | API operations of the contract; the routes `/line-items/:id` and `/line-items/:id/edit`; the history kind `lineItem` |

## Quickstart checks (after building)

1. `go run ./cmd/seed -reset && go run ./cmd/api` in `backend/`; `npm run dev` in `frontend/`.
2. As Priya Raman open FY2026 Lubricants Europe › Launch film: two rows, "Unassigned planned cost
   (Sys_Gen_Balance)" 20,000.00 USD, "100,000.00 of 120,000.00 USD assigned", grid Mar–May.
3. **Add line item** → only choose a spend type → **Save**: the balance reads "All of the planned cost is
   assigned".
4. On the seeded Approved activity change a line item's planned cost: the dialog appears; after confirming,
   the activity is Created. Delete a line item and choose **Undo**: it is back with the same ID.
5. `gofmt -l .`, `go vet ./...`, `go test ./...` (backend); `npm run check`, `npm run test:e2e` (frontend).

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| Consumer-side interfaces in `lineitems/deps.go` with adapters in `internal/app/lineitems.go` | `lineitems` reads activities, plans and Settings while 004–006 read line-item numbers; Go forbids import cycles. The interfaces also let the 007 backend be tested with fakes before or alongside 002–006 | Importing `activities` from `lineitems` and `lineitems` from `activities` is an import cycle; one god package for all planning records would break constitution IV's package-per-area rule |
| Optional `history.Kind.AfterRestore` hook (a change to 001's code) | FR-033 needs a rule that runs only when the restored record is the line item itself, after it is back | `PrepareRestore` runs for every snapshot, also when the line item comes back with its activity, and cannot tell the two apart; 001 data model §7 anticipated this hook for 007/008 |
| Spread and conversion rules in both Go and TypeScript | The form shows conversions, previews and totals on every keystroke (FR-011, FR-012, FR-017, FR-018) | A preview request per keystroke adds an endpoint, latency and loading states to a form beginners use; both sides share one example table, and the API's result is always what is saved and shown afterwards |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-140–D-154 in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **Re-approval (D-189).** `restartApproval` (body; DELETE query parameter) and `400 validation_failed` on `restartApproval`; the
  draft's `confirmRestartApproval` and `approval_restart_required` are gone from this plan, its tasks and the contract.
- **Typed amounts (D-190).** The form rounds amounts with more than two decimals and says so; the API refuses them.
- **Copies (D-195).** Implement `activities.LineItems.CopyAll(ctx, a, fromActivityID, toActivityID, yearShift)` (single activity,
  campaign and initiative copies) and register the `lineItem` `plans.Level` whose `Copy` handles plan copies with `job.Years`
  (007-P9 applies to both when `yearShift` is not 0). Each line item is copied once.
- **006 seams (D-196).** Call 006's exported `RestartApproval`; add the optional `history.Kind.AfterRestore` hook (011 relies on it and
  on the status change being logged through `history.Log`).
- **Money (002).** Use `money.ConvertSpread` (D-024) and `(Amount).SplitEvenly` instead of own `USDMonths`/even split;
  `Amount.Format`, `Money.Format`, `FromCents`, `Cents` exist in 002's package.
- **Web parts (D-197).** 002's `SegmentedFilter` replaces `SegmentedControl`; 004's `FormSection`, `FormFooter`, `LevelTag`,
  `ProgressBar`, `Breadcrumb` (002); 002's `MoneyField`, `ChoiceField`, `ShellAttributeFields`.
- **Field names.** Line-item problems in 006's Ready-to-submit list use `attributes.<attributeId>` and this plan's field names.
