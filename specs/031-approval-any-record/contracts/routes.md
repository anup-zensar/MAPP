# Routes and permissions touched by feature 031

No new HTTP operation (the purchase-request approval routes are feature 032's). The access route table
(`internal/access/routes_approvals.go`) is unchanged. Changed answers and bodies:

| Operation | Rule (unchanged) | Change |
|---|---|---|
| `GET /api/v1/approval-workflows` | signed in | each workflow: `objectType`, `objectTypeFixed`; levels: `approverType` adds `owner`, `recordField`; `recordField` |
| `POST /api/v1/approval-workflows` | `workflows.manage` | body: optional `objectType` (default `activity`); level `recordField` |
| `GET /api/v1/approval-workflows/{workflowId}` | signed in | as list; usage lists records of the workflow's type (`displayStatus` widened to the record's status) |
| `PATCH /api/v1/approval-workflows/{workflowId}` | `workflows.manage` | body: optional `objectType` (refused once used); level `recordField` |
| `GET /api/v1/approval-rule-options` | signed in | adds `objectTypes` and `recordFields` |
| `GET /api/v1/activities/{activityId}/approval` | view | `actions.resubmit` available while Submitted; rounds carry `kind`; outcome `superseded`; history action `superseded`, `selfApproval` |
| `GET /api/v1/activities/{activityId}/approval/submission` | `activities.submit` in plan | `kind: resubmit` and `available` while Submitted; `replaces` (round number, level, label) |
| `POST /api/v1/activities/{activityId}/approval/submit` | `activities.submit` in plan | also from Submitted (ends the open round as superseded) |
| `POST …/approve`, `…/decline`, `…/recall` | custom (017 check order) | 409 "The submission was re-submitted by …" on a replaced round |
| `GET /api/v1/approvals` | signed in, filtered | rows: `subject` (`kind`, `label`, `id`, `name`, `status`, `displayStatus`); `activity` null for other record types |
| `GET /api/v1/approvals/monitoring` | signed in, filtered | overdue rows as above; `byLevel[].kind` |
| `GET /api/v1/notifications`, `POST …/read`, `POST …/read-all` | own | each notification: `subject` (`kind`, `label`, `id`, `name`, `exists`); `activity` null for other record types |

Contract schemas added for feature 032's routes (no operation uses them yet): `ApprovalSubjectKind`,
`ApprovalSubjectStatus`, `ApprovalSubject`, `ApprovalSection`.
