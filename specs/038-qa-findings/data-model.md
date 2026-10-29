# Data Model: QA Findings (038)

Only what changes. Everything else stays as features 001–037 define it.

## 1. Stored (PostgreSQL)

### 1.1 Migration `00026_overspend_approval.sql` (US7, FR-018)

| Table | Column | Type | Constraint | Default |
|---|---|---|---|---|
| `overspend_configuration` | `approval_over_planned_cost` | `text` | `NOT NULL CHECK (approval_over_planned_cost IN ('warn', 'refuse'))` | `'warn'` |

The singleton row keeps its `version`; a change of either setting raises it by one and writes one activity-log
entry naming every field that changed ("Approval over planned cost: Warn → Refuse"). sqlc:
`GetOverspendConfiguration` and `PutOverspendConfiguration` (`queries/overspend.sql`) carry the column.

### 1.2 Migration `00027_finance_link_roles.sql` (US2, FR-006)

- `UPDATE roles SET permissions = array_remove(permissions, 'finance.link'), version = version + 1, updated_at =
  now(), updated_by_id = 'system', updated_by_name = 'SP&A One' WHERE built_in IN ('globalPlanner', 'planner')
  AND 'finance.link' = ANY (permissions)` — the built-in roles as 014 Table B now states. Custom roles are left as
  their administrators set them; the access check ignores `finance.link` without `finance.import` (FR-006).
- Down: none (forward-only, B5).
- Migration numbers are fixed here so the two backend lanes cannot collide: 00026 belongs to lane B1, 00027 to B2.

### 1.3 Read changed, no schema change: purchase-request claims (US1, FR-001 – FR-003)

`ListPurchaseRequestClaims` also selects `p.change_pending` and `p.approved_version`. In Go:

| Type | Field | Meaning |
|---|---|---|
| `purchaserequest.Claim` | `Approved *money.Money` | The line's last approved amount when the request has a change in review; nil otherwise |
| `purchaserequest.Claim` | `RemovedByChange bool` | A claim built from the approved version for a line the pending change removed |

Rule: `reserved(claim) = max(convert(Amount), convert(Approved))`, both at the current rate (D-303); a removed line
reserves `convert(Approved)`; without a change, `Amount` as today.

### 1.4 Read added: workbook preparation (US3, FR-008)

Batch reads, each once per check, scoped to the plan the person opened (scope taken from the actor, B5): activities
of the plan with their parents and names (siblings), and the reference lists the per-row path read before
(activity types, people, workflows with approvers, attributes with options). Any new store read of plan-owned data
gets a scope-isolation test (constitution VI).

## 2. Held in memory only

### 2.1 Sign-in attempts (US6, FR-014)

| Field | Meaning |
|---|---|
| key | `email:<lower-cased, trimmed e-mail>` or `client:<client address>` |
| failures | times of failed attempts within the last 15 minutes |
| lockedUntil | end of the lock (15 minutes after the failure that reached the limit) |

Limits: e-mail 5, client 20. Pruned on each access; cleared for both keys on success; lost on restart (assumption).
Never logged with the e-mail.

## 3. Views (API answers) — see [contracts/openapi.yaml](contracts/openapi.yaml)

| View | Change | FR |
|---|---|---|
| `OverspendConfiguration` | `approvalOverPlannedCost` (`warn`/`refuse`); `updatedAt`, `updatedBy` nullable when never saved | FR-018, FR-057 |
| `OverspendConfigurationInput` | optional `approvalOverPlannedCost` (absent keeps it) | FR-018 |
| `PurchaseRequestConfiguration` | `updatedAt`, `updatedBy` nullable when never saved | FR-057 |
| `ActivityApproval`, `PurchaseRequestApproval` | `overPlannedCost: ActivityOverage[]` | FR-019 – FR-021 |
| `ActivityOverage` (new) | `activity: RecordRef`, `overBy: Money` (plan currency) | FR-019 |
| `PurchaseRequestSubmission` | `outcomes: ApprovalSubmissionLevel[]` (resolved approvers per level) | FR-027 |
| `HomeActivityCounts` | `yearFrom`, `yearTo` (integers or null when no activities) | FR-032 |
| `SpaPeriods` | `linesAbovePlannedCost` (Money or null), `linesAbovePlannedCostTrace` | FR-023 |
| `DashboardMonths`, `BudgetByMonthRow` | `negativeBalanceCount` (integer ≥ 0) | FR-024 |
| `FinancialTransactionPage.counts` | `notLinked` present only for people who see not-linked transactions | FR-004 |
| `ErrorDetail.code` | `too_many_attempts` (429), `too_large` (413) | FR-014, FR-054 |

## 4. Rules (each with its unit test, plan §Rules)

| Rule | Where | FR |
|---|---|---|
| Reserved amount per line during a change = larger of approved and changed | `purchaserequest/balance.go` | FR-001 |
| Not-linked visible only with `finance.import`; link needs import + link; unlink needs link in the line item's plan | `financials` | FR-004, FR-005 |
| Scope before permission before version before state; hidden = missing | `financials/service.go`, `app/financials.go` | FR-007 |
| Role with `finance.link` needs `finance.import` | `access/roles.go` (`permissions` field) | FR-006 |
| Sign-in lock: 5 per e-mail / 20 per client in 15 min, lock 15 min, reset on success | `auth/attempts.go` | FR-014 |
| Approve over planned cost: Warn logs, Refuse refuses; decline always allowed | `approval/rounds_decide.go` | FR-019, FR-020 |
| Grid rows and columns add up after conversion | `lineitem/display.go`, `money.Display.Parts` | FR-022 |
| Not yet scheduled ≥ 0 per activity; excess as its own figure | `spa/scope.go` | FR-023 |
| Control characters and unpaired surrogates refused at binding; IDs → not found | `platform/web` | FR-017 |
| Single-line names refuse tab, line breaks, direction controls | `platform/record` helper | FR-055 |
| Rate dates ≥ 2000-01-01 | `settings/rates.go`, `rates_import.go` | FR-034 |
| Problem value ≤ 200 characters + "…" | `workbook/apply.go` | FR-010 |
