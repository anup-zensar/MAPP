# Routes and access rules — feature 037

New operations (rules in `backend/internal/access/routes_overspend.go`, fail closed):

| Operation | Rule | Notes |
|---|---|---|
| `GET /api/v1/overspend-configuration` | signed in | everyone reads the organisation default; `canChange` says whether they may change it |
| `PUT /api/v1/overspend-configuration` | `settings.manage` | body `{version, defaultMode}`; 400 `validation_failed`, 409 `version_conflict` |

Changed operations (rules unchanged; scoped checks in the services):

| Operation | Change |
|---|---|
| `POST /api/v1/plans`, `PATCH /api/v1/plans/{planId}` | optional `overspend`; a change needs the owner or `globalBudgets.manage` in the plan's scope (403 `forbidden`); on an allocated plan 409 `not_allowed` |
| `POST /api/v1/plans/{planId}/campaigns`, `PATCH /api/v1/campaigns/{campaignId}` | optional `overspend`, same rule with the campaign's owner |
| `POST /api/v1/campaigns/{campaignId}/initiatives`, `PATCH /api/v1/initiatives/{initiativeId}` | optional `overspend`, same rule with the initiative's owner |
| `POST /api/v1/global-budgets`, `PATCH /api/v1/global-budgets/{globalBudgetId}` | optional `overspend` |
| `POST /api/v1/global-budgets/{globalBudgetId}/allocations`, `PATCH /api/v1/budget-allocations/{allocationId}` | optional `overspend`, the allocated plan's |
| `GET /api/v1/initiatives/{initiativeId}/activity-form`, `GET /api/v1/activities/{activityId}/form` | answer gains `overspendLimits` |
| Every write that changes planned cost or a budget | may answer 409 `over_budget` with `overBudget` |
| Purchase-request submit operations | one more `not_ready`/submit problem (032 FR-021, 037 FR-019) |
