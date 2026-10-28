# Routes, permissions and web interfaces (032)

## API operations → access rule (`backend/internal/access/routes_purchase_requests.go`)

Rule kinds are those of `access/routes.go` (014 Table C): **Need** — the permission in any scope, checked by
the middleware; **InPlan** — the middleware checks the permission in any scope and the service checks it in
the PR's plan (`authz.RequireIn`); **View** — the service checks the person sees the record (404 otherwise);
**Custom** — the service applies its own rule; **SignedIn** — any signed-in person. Every operation also needs
Purchase Support on, except the reads and the configuration (409 `not_allowed` otherwise).

| # | Operation | Rule | Permission | Service check |
|---|---|---|---|---|
| 1 | `GET /api/v1/purchase-requests` | need | `purchaseRequests.list` | rows filtered to the plans where the person holds it |
| 2 | `POST /api/v1/purchase-requests` | inPlan | `purchaseRequests.edit` | in the plan of the lines; line items must be visible |
| 3 | `GET /api/v1/purchase-requests/new-defaults` | inPlan | `purchaseRequests.edit` | line item visible (404) and permission in its plan (403) |
| 4 | `GET /api/v1/purchase-requests/line-item-options` | inPlan | `purchaseRequests.edit` | options limited to plans where the person holds it |
| 5 | `GET /api/v1/purchase-requests/{purchaseRequestId}` | view | — | `purchase_request_in_scope` (data-model §1.6) |
| 6 | `PUT /api/v1/purchase-requests/{purchaseRequestId}` | inPlan | `purchaseRequests.edit` | view first (404); also `purchaseRequests.submit` when `then` is resubmit or submitChanges |
| 7 | `DELETE /api/v1/purchase-requests/{purchaseRequestId}` | inPlan | `purchaseRequests.delete` | view first (404) |
| 8 | `GET /api/v1/purchase-requests/{purchaseRequestId}/close-preview` | inPlan | `purchaseRequests.edit` | view first (404) |
| 9 | `POST /api/v1/purchase-requests/{purchaseRequestId}/close` | inPlan | `purchaseRequests.edit` | view first (404) |
| 10 | `POST /api/v1/purchase-requests/{purchaseRequestId}/resend` | inPlan | `purchaseRequests.edit` | view first (404) |
| 11 | `GET /api/v1/purchase-requests/{purchaseRequestId}/routings/{routingId}/form` | view | — | sees the PR; the routing belongs to it |
| 12 | `GET /api/v1/purchase-requests/{purchaseRequestId}/approval` | view | — | sees the PR |
| 13 | `GET /api/v1/purchase-requests/{purchaseRequestId}/approval/submission` | inPlan | `purchaseRequests.submit` | view first (404) |
| 14 | `POST /api/v1/purchase-requests/{purchaseRequestId}/approval/submit` | inPlan | `purchaseRequests.submit` | view first (404) |
| 15 | `POST /api/v1/purchase-requests/{purchaseRequestId}/approval/approve` | custom | (`approvals.decide`) | the engine's decide rules (031): named approver of the waiting level, delegate, backup or on behalf; self-approval allowed |
| 16 | `POST /api/v1/purchase-requests/{purchaseRequestId}/approval/decline` | custom | (`approvals.decide`) | as approve; comment required |
| 17 | `POST /api/v1/purchase-requests/{purchaseRequestId}/approval/recall` | custom | (`purchaseRequests.submit`) | owner or submitter holding `purchaseRequests.submit` in the plan, or `approvals.decideOnBehalf` (engine) |
| 18 | `GET /api/v1/approvals/purchase-requests/{purchaseRequestId}` | view | — | sees the PR |
| 19 | `GET /api/v1/line-items/{lineItemId}/purchase-requests` | view | — | sees the line item; rows limited to PRs the person sees |
| 20 | `GET /api/v1/activities/{activityId}/purchase-requests` | view | — | sees the activity |
| 21 | `GET /api/v1/purchase-request-configuration` | signedIn | — | `canChange` from `settings.manage` |
| 22 | `PUT /api/v1/purchase-request-configuration` | need | `settings.manage` | — |

Existing operations whose behaviour changes (rules unchanged): `DELETE` of plans, campaigns, initiatives,
activities and line items and `POST /api/v1/campaigns/{campaignId}/move` (409 `in_use` when a PR draws on a
line item inside, FR-019); `PATCH /api/v1/line-items/{lineItemId}` (400 on `plannedCost`/`currency` below Total
requested); `POST /api/v1/recycle-bin/{entryId}/restore` and `…/undo` for the PR kinds (restore needs
`purchaseRequests.delete` for a PR, `purchaseRequests.edit` for a line, in the PR's plan — `app/history_access.go`);
`GET /api/v1/shell-attribute-forms/{level}` and `…/check` accept the two new levels.

## New permissions (014 Table A additions, group "Purchase requests", all scoped)

| Key | Label (role page) | Refusal noun |
|---|---|---|
| `purchaseRequests.edit` | Raise and edit purchase requests | Raising and changing purchase requests |
| `purchaseRequests.delete` | Delete purchase requests | Deleting purchase requests |
| `purchaseRequests.submit` | Submit purchase requests for approval and recall them | Submitting purchase requests |
| `purchaseRequests.list` | See the Purchase requests page | Seeing the Purchase requests page |

Seeded roles (spec FR-049): Administrator (every permission by definition); Global planner +all four; Planner
+edit, delete, submit; Approver and Viewer unchanged; **Requisitioner** (new): `plans.view`,
`purchaseRequests.list`, `finance.import`, default scope All markets.

## Web routes (`frontend/src/app/routes.tsx`, lazy per feature in `pages.tsx`)

| Path | Page | Gate |
|---|---|---|
| `/purchase-requests` | `PurchaseRequestsPage` | `RequirePermission purchaseRequests.list`; sidebar item "Purchase requests" after Financials (`navigation.ts`, icon `FileText`) |
| `/purchase-requests/new?lineItemId=…` | `PurchaseRequestCreatePage` (picks a line item first when the parameter is absent) | the server's 403/404 states |
| `/purchase-requests/:purchaseRequestId` | `PurchaseRequestPage` | 404 state for missing and out-of-scope alike |
| `/purchase-requests/:purchaseRequestId/edit` | `PurchaseRequestEditPage` | `access.edit` explains when unavailable |
| `/settings/purchase-requests` | `PurchaseRequestSettingsPage` (card "Purchase requests" in Settings, after Approval workflows) | read-only without `settings.manage` |

## Component interfaces the feature exports (`features/purchase-requests/index.ts`)

| Export | Used by | Props |
|---|---|---|
| `LineItemPurchaseRequests` | `features/line-items/LineItemPage.tsx` (section after Financials) | `{ lineItemId: string }` |
| `ActivityPurchaseRequests` | the activity page (section after Line items) | `{ activityId: string }` |
| `PURCHASE_REQUEST_SETTINGS_SECTIONS` | `app/settings-sections.ts` | — |
| `PurchaseRequestsPage`, `PurchaseRequestPage`, `PurchaseRequestCreatePage`, `PurchaseRequestEditPage`, `PurchaseRequestSettingsPage` | `app/pages.tsx` | — |

`purchaseRequestHref(id: string): string` is added to `src/ui/record-links.ts` (next to `recordHref`), so
`features/notifications` and `features/approvals` link to a PR without importing this feature.
`features/approvals` adds `PurchaseRequestPreviewPanel` (its own file and API hooks for operations 15, 16
and 18) and opens it for inbox items of subject kind `purchaseRequest` or `modifiedPurchaseRequest`;
`features/notifications` links those kinds to `purchaseRequestHref`. The purchase-requests feature imports
`@/features/approvals` (ApprovalChain, ApprovalHistory, decision and recall dialogs) — never the reverse.
