# Data Model: Planning Dashboards and Export

Nothing is stored. Every structure below is worked out per request from the records of features 004–033 and
returned or written to a file. Amounts are summed in USD (exact cents) and converted once to the chosen
currency (`money.Display`).

## 1. Query (filters)

| Field | Values | Default | Notes |
|---|---|---|---|
| `year` | 2000–2100 | the current year (injected clock) | The plan year (D-011) |
| `planType` | `operational`, `strategic`, `crossMarket`, empty | empty = all | 016 plan type |
| `currency` | an active currency code of Settings | the person's display currency | 400 `validation_failed` otherwise |
| `teamId` | a team id (Regional only) | the person's first team by name, else the first team | 400 when unknown |
| `planId` | a plan id (Regional only) | none | Ignored unless the plan is among the dashboard's plans |

## 2. View (contract `PlanningDashboard`)

| Part | Content | Rule |
|---|---|---|
| `kind` | `executive` or `regional` | |
| `filters` | the query as applied (defaults filled in) | FR-006 |
| `choices` | years (with plans in scope, plus the current and the chosen year, newest first), teams (id, name, active, mine), currencies (code, name) | FR-006 |
| `team` | the Regional dashboard's team, or null | FR-020 |
| `planCount`, `plansWithoutBudget` | plans in view; those without a budget | FR-008 |
| `finance` | `visible` (the person may see financial data in at least one plan in view), `planCount` (plans whose financial figures count) | FR-003 |
| `financeAsOf` | when GSAP data was last received, or null | FR-019 |
| `totals` | `Figures` of every plan in view | FR-008–FR-011 |
| `markets` | per team with plans in view: team, plan count, `Figures` (Executive only; empty for Regional) | FR-012 |
| `plans` | per plan in view, by name: plan ref (id, name, status, type), teams, `Figures`, `financeShown` | FR-013 |
| `overBudget`, `leftToPlan` | up to 5 `CampaignRow` each | FR-015 |
| `planCampaigns` | the chosen plan and every campaign of it as `CampaignRow`, or null | FR-021 |
| `months` | 12 `Month` rows; `noDates`, `otherYears`, `paidOutsideYear` | FR-014 |
| `approvals` | waiting, oldest waiting since, approved, declined, auto-approved, average decision days, activities by display status | FR-016 |
| `purchaseRequests` | available (Purchase Support on), count and amount per status, total count and amount, waiting for GSAP | FR-017 |
| `completeness` | per level plan … line item: total, complete, percent | FR-018 |
| `globalBudgets` | the year's global budgets (id, name, amount, allocated, unallocated, planned cost, over-allocated), or null without `globalBudgets.view` | FR-004 |
| `canExport` | the person holds `data.export` | FR-024 |

### Figures

| Field | Rule (USD, then converted) | "—" (null) when |
|---|---|---|
| `budget` | Σ plan budgets (campaign rows: the campaign budget) | no record in the set has a budget |
| `plannedCost` | Σ activities' planned cost | no activity has one |
| `committed`, `actual` | Σ line items' Committed and Paid (015), over the records whose finance the person sees | the person sees finance in none of them |
| `budgetUsedPercent` | `rollup.BudgetUsedPercent(planned, budget)` | no budget, budget 0.00 or no planned cost |
| `budgetSpentPercent` | `rollup.BudgetUsedPercent(actual, budget)` | no budget or no actual |
| `leftToPlan` | `rollup.LeftToPlan(budget, planned)` | no budget |
| `unspentBudget` | budget − actual | no budget or no actual |
| `overBudget` | `rollup.OverBudget(planned, budget)` | — (false) |

### CampaignRow

`campaign` (id, name), `plan` (id, name), `Figures`, `difference` (over budget: planned − budget; left to plan:
budget − planned).

### Month

`month` (`2026-01` … `2026-12`), `planned` (Money), `actual` (Money or null when finance is hidden),
`plannedToDate`, `actualToDate` (running totals from January).

## 3. Deck (`internal/platform/deck`)

| Type | Fields | Notes |
|---|---|---|
| `Deck` | `Title`, `Created` (time), `Pages []Page` | One file |
| `Page` | `Title`, `Subtitle`, `Cards []Card`, `Chart *Chart`, `Table *Table`, `Notes []string` | Laid out once |
| `Card` | `Label`, `Value`, `Note`, `Problem` (red value) | Up to 6 per page |
| `Table` | `Columns []Column` (title, weight, align), `Rows []Row` (cells, bold) | Continues on new pages |
| `Chart` | `Categories []string`, `Series []Series` (name, colour, values in cents), `MaxLabel` | Grouped bars; values below 0 drawn as 0 |
| `Sheet` (internal) | positioned `Box`, `Text`, `Grid` | What both writers draw |

`WritePPTX(w, Deck)` and `WritePDF(w, Deck)` write the same sheets.
