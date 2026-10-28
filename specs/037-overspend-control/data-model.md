# Data Model: Overspend Control

## 1. Stored data (migration `00025_overspend.sql`)

| Table | Column | Type | Rule |
|---|---|---|---|
| `plans` | `overspend` | text, not null, default `'warn'` | `CHECK (overspend IN ('warn', 'refuse'))`; for an allocated plan, set through its allocation |
| `campaigns` | `overspend` | text, not null, default `'warn'` | same check |
| `initiatives` | `overspend` | text, not null, default `'warn'` | same check |
| `global_budgets` | `overspend` | text, not null, default `'warn'` | same check |
| `overspend_configuration` (new) | `id` text PK `CHECK (id = 'overspend')`, `default_mode` text `CHECK (default_mode IN ('warn','refuse'))`, `version` ≥ 1, created/updated at and by | | one row at most; without it the default is Warn |

Existing rows get `'warn'` from the column default, so nothing changes for stored budgets (D-347).

## 2. Go types

**`internal/overspend`** (new package: model, guard, configuration service, handler, store):

| Type | Fields | Notes |
|---|---|---|
| `Mode` | `"warn"` · `"refuse"` | `Normalised()` reads `""` (records written before 037 in fakes) as Warn |
| `Control` | `Mode`, `CanChange authz.Availability` | how a record shows its setting (contract `OverspendControl`) |
| `Budget` | `Kind` (plan · campaign · initiative · globalBudget), `ID`, `Name`, `Mode`, `Amount money.Money`, `Planned money.Money`, `PlanIDs` (a global budget's allocated plans) | one budget as the guard reads it; never stored |
| `Overage` | `Budget errs.RecordRef`, `Amount`, `PlannedCost`, `OverBy money.Money` | part of a refusal (contract `BudgetOverage`) |
| `Limit` | `Budget errs.RecordRef`, `LeftToPlan money.Money` | a Refuse budget above an activity (contract `OverspendLimit`) |
| `Configuration` | `DefaultMode Mode`, `record.Meta` | Settings › Overspend control |

**Changed records**: `plan.Plan.Overspend`, `campaign.Campaign.Overspend`, `campaign.Initiative.Overspend`,
`budget.GlobalBudget.Overspend` (all `overspend.Mode`, JSON `overspend`). Inputs and updates gain an optional
`overspend` text (nil keeps the stored value; on create, nil takes the organisation default). Views gain
`overspend: OverspendControl`; allocation rows and options gain the plan's `overspend` mode; the activity
form gains `overspendLimits: []Limit`; `purchaserequest.LineItemFacts` gains `ActivityOverBy *money.Money`.

## 3. Rules (each with its unit test, see tasks.md)

1. **Planned cost under a budget** (FR-008): plan, campaign, initiative — sum of `activities.planned_cost` below
   it (plan currency, exact); global budget — Σ over allocated plans of `ToUSD(plan planned cost, rate)`.
2. **Refusal** (FR-009, FR-011, R2): for each budget `b` with `Mode = refuse` after the save and
   `planned > amount`: refused when `b` did not exist before with an amount, or `planned_after > planned_before`,
   or `amount_after < amount_before`.
3. **Overage** = planned − amount, in the budget's currency; listed in the order plan, campaign, initiative,
   global budget, then by name.
4. **Who may change** (FR-003, R6): internal call · owner · `globalBudgets.manage` in the plan's scope.
5. **Default** (FR-007): new record without a sent mode → configuration's `DefaultMode`; copy → source's mode.
6. **Limit** (FR-014): `leftToPlan = amount − planned` (may be negative) for each Refuse budget on the
   activity's path; a global budget's converted from USD to the plan currency.
7. **PR gate** (FR-019): `ActivityOverBy = −balance` when the activity's unassigned planned cost is negative.

## 4. Activity log

| Record | Action | Changes |
|---|---|---|
| Plan, campaign, initiative, global budget | changed | `overspend` "Overspend": Warn → Refuse (with any other field of the save) |
| Plan (by allocation) | changed | `overspend` with the allocation's comment |
| Overspend control (kind `overspendConfiguration`) | changed | `defaultMode` "New budgets start with" |

Refused saves write nothing (the transaction is rolled back).
