# Routes and access rules — feature 038

No new operations. Rules in `backend/internal/access/` are unchanged (fail closed); every change below is a scoped
check in a service or the transport.

| Operation | Change | FR |
|---|---|---|
| `POST /api/v1/auth/sign-in` | 429 `too_many_attempts` with `Retry-After`; checked before credentials | FR-014 |
| `PUT /api/v1/overspend-configuration` | body may carry `approvalOverPlannedCost`; still `settings.manage` | FR-018 |
| `POST /api/v1/activities/{activityId}/approval/approve` | 403 first without either approvals permission; 409 `not_allowed` when refusing over planned cost | FR-020, FR-056 |
| `POST /api/v1/purchase-requests/{purchaseRequestId}/approval/approve` | same | FR-020, FR-056 |
| `GET /api/v1/purchase-requests/{purchaseRequestId}/approval/submission` | answer gains `outcomes` | FR-027 |
| `GET /api/v1/financials/transactions/{transactionId}` | not-linked without `finance.import` → 404 (was 403) | FR-004, FR-007 |
| `POST …/transactions/{transactionId}/link` | order: visible → `finance.import` + `finance.link` in target plan → version → state; target hidden = missing | FR-005, FR-007 |
| `POST …/transactions/{transactionId}/unlink` | order: visible → `finance.link` in linked line item's plan → version → state | FR-005, FR-007 |
| `GET /api/v1/financials/transactions`, `/overview`, `/api/v1/home/…` financials card | not-linked count only with `finance.import` | FR-004 |
| `POST /api/v1/roles`, `PATCH /api/v1/roles/{roleId}` | `finance.link` needs `finance.import` (400 on `permissions`) | FR-006 |
| `GET /api/v1/dashboards/executive`, `/regional` and exports | unknown currency or team falls back (was 400) | FR-012 |
| `POST /api/v1/plans/{planId}/imports/check`, `/imports` | file read before the write lock; problem values ≤ 200 characters | FR-008 – FR-010 |
| Every operation | security headers; foreign `Referer` without `Origin` refused on unsafe methods; control characters 400/404; 413 over the body limit | FR-015 – FR-017, FR-054 |
