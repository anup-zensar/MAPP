# Data Model: SP&A Plan View and BCPIR Marker

Nothing is stored: no migration, no table. The BCPIR flag is the existing `plans.bcpir` column (004).

## 1. Inputs (ports of `internal/spa`)

| Port | Method | Gives |
|---|---|---|
| `Planning` | `Plan(ctx, planID)` | the plan when the actor sees it (scope from the context): id, name, year, status, BCPIR, budget in USD, `authz.PlanFacts`; false otherwise |
| | `Records(ctx, plan)` | its live campaigns, initiatives, activities and line items in plan-tree order, each with: parent, name, owner name, activity display status, activity type name, start date, months, USD planned cost (activities as typed, line items converted), USD budget (campaigns), line-item USD spread by month and unspread amount, finance budget link, and the values of every dimension (label with optional share) |
| | `Dimensions(ctx)` | Activity type and the active choice and yes/no attributes of the activity and line-item levels |
| `Finance` | `SeesFinance(ctx, a, planID)` | whether the actor holds `finance.view` in the plan |
| | `SpendByMonth(ctx)` | per line item and posting month ("2026-03"): committed, invoiced, paid in USD (015's Facts) |
| | `DataAsOf(ctx)` | when GSAP data was last loaded; nil when never |
| `Purchases` | `Drawn(ctx, a, lineItemIDs)` | the PRs the actor sees on those line items: id, name, vendor, status, and their lines on them in USD |
| | `CanRaise(ctx, plan)` | 014's `purchaseRequests.edit` in the plan and Purchase support (032) as an availability |
| `Displays` | `Display(ctx, personID)` | the display currency with its rate |

## 2. Calculations (all in USD cents, converted once at the end)

- **Activity in-year planned cost** = planned cost (nil = 0) − Σ its months outside the plan's year (line-item
  spreads and the balance spread).
- **Activity months** = Σ line-item spread months + positive balance (planned cost − Σ line items' planned
  cost) split evenly over the activity's months (none without dates).
- **Not yet scheduled** = in-year planned cost − Σ in-year months (no dates, negative balance).
- **Committed / Invoiced / Actual spend** = Σ the activity's line items' 015 facts posted in the plan's year.
- **Remaining** = Planned − Actual. **Utilisation %** = Actual × 100 ÷ Planned. **Variance %** = (Actual −
  Planned) × 100 ÷ Planned. One decimal, half away from zero; "—" (null) when Planned = 0.
- **Overspend** = Remaining < 0 (equivalently Utilisation > 100 % and Variance > 0 %).
- **Dimension amount** of an activity = its in-year planned cost (activity level) or each line item's in-year
  amount plus "Not yet itemised" = in-year planned cost − Σ line items' in-year amounts (line-item level),
  split by value as R4.
- **Share** = amount × 100 ÷ scope planned cost, two decimals; **bar** = amount × 100 ÷ largest amount, whole.
- **Planned vs budget %** = (Planned − Budget) × 100 ÷ Budget, one decimal; null with "No budget set" when the
  budget is nil or 0.
- **Linked** = Σ in-year amounts of line items with a finance budget item; **Not linked** = Planned − Linked.
- **PR amounts**: per PR, Σ its lines on the scope's line items (USD); Approved = approved + closed,
  Pending approval = submitted + inReview, Draft = created; declined counted nowhere.

## 3. Answer (`SpaView`, contract)

`plan` {id, name, year, status, bcpir} · `scope` {type plan|campaign, id, name} · `campaigns` (the picker) ·
`currency` · `financeAccess`, `financeNote` · `financeDataAsOf` · `headline` · `periods` {quarters[4],
months[12], notScheduled, notScheduledTrace} · `dimension` {key, label, rows[], total, countsInEach} ·
`dimensions[]` · `activities[]` (with `lineItems[]` and `raisePurchaseRequest`) · `budget` · `procurement`
{count, approved, pendingApproval, draft, invoiced, items[]} · `notes[]` · `emptyMessage`.

## 4. BCPIR marker and filter

| Where | Change |
|---|---|
| `PlanSummary` (Plans list) | `bcpir: boolean` |
| `GET /api/v1/plans` | `bcpir=include|only|exclude` (default include; anything else 400 "Choose include, only or exclude.") |
| `allwork.Node` / `AllWorkRow` | `BCPIR` on plan nodes; `bcpir` on plan rows |
| All work query, saved view filters | `bcpir=include|only|exclude`; chip "BCPIR plans: Only/Exclude"; export note |
| All work Breakdown | grouping `bcpir` "BCPIR plan" with groups Yes / No |
