# Route and permission table — feature 034

Feature 034 adds no operation and changes no access rule. The operations whose behaviour it changes keep
the rules of `backend/internal/access/routes_planning.go` (014 Table A/C):

| Operation | Rule (unchanged) | What 034 changes |
|---|---|---|
| `POST /api/v1/activities/{activityId}/line-items` | `inPlan(activities.edit)` | no confirmation, status kept (FR-001) |
| `PATCH /api/v1/line-items/{lineItemId}` | `inPlan(activities.edit)` | same (FR-001) |
| `DELETE /api/v1/line-items/{lineItemId}` | `inPlan(activities.delete)` | same; `?restartApproval` ignored (FR-001) |
| `POST /api/v1/line-items/{lineItemId}/duplicate` | `inPlan(activities.edit)` | same (FR-001) |
| `POST /api/v1/recycle-bin/{entryId}/undo`, `…/restore` | `custom()` — `history.Access.CanRestore` (unchanged) | restoring a line item keeps the activity's status (FR-002) |
| `GET /api/v1/activities/{activityId}/approval` and the approval actions | unchanged (008, 014) | recall explanation text (FR-005) |
| `GET /api/v1/plan-defaults` | `need(plans.create)` (unchanged) | years current … + 10 (FR-014) |
| `POST /api/v1/plans` | `inPlan(plans.create)` | year check current … + 10 (FR-014) |
| `PATCH /api/v1/plans/{planId}` | `inPlan(plans.edit)` | none — a past year stays valid (FR-015) |
| `GET /api/v1/plans/{planId}/copy-options`, `POST …/copy` | `viewAnd(plans.create)` | next-year limit current + 10 (FR-016) |

Scope: unchanged — every operation above reads the scope only from the signed-in actor; a record outside it
answers 404. No new scoped store read, so no new scope-isolation test.
