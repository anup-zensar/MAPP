# Routes, permissions and web interfaces (033)

## API operation → access rule

| Operation | Rule | Permission | Service check |
|---|---|---|---|
| `POST /api/v1/simulated-gsap/purchase-requests/{purchaseRequestId}/purchase-order` | need | `finance.import` ("Import GSAP extracts", not scoped) | the person sees the PR (`purchase_request_in_scope`, 404 otherwise); Purchase Support on; Approved, no change pending, routed, waiting for GSAP; version current |

The rule goes into `backend/internal/access/routes_purchase_requests.go` (032's file), so 033 does not touch a
routes file 030 may own for its SWBS runs.

Existing operations whose answers change (rules unchanged): `GET /api/v1/purchase-requests/{purchaseRequestId}`
and the other PR answers (`access.gsapOrder`, `lines[].gsapPending`, GSAP fields filled);
`GET /api/v1/financials/imports` and the transaction details (source `simulatedGsap`);
`GET /api/v1/financials/records/{recordId}`, `/financials/transactions`, `/financials/overview` (the booked PO
lines in Committed and the lists, through 015's rules).

## Web interfaces

| Where | Change |
|---|---|
| `features/purchase-requests/PurchaseRequestPage.tsx` | action "Create GSAP PR and PO (simulated)" (secondary button; Edit stays the one yellow action), shown when `access.gsapOrder` is available or explained when unavailable with a reason; hidden when the reason is empty |
| `features/purchase-requests/GsapOrderDialog.tsx` (new) | confirmation (create or top-up) listing the lines with `gsapPending`; one primary button "Create" / "Update PO" |
| `features/purchase-requests/GsapFields.tsx`, `PurchaseRequestLinesTable.tsx` | source label "Simulated GSAP" next to GSAP PR number and PO columns |
| `features/financials` | import source `simulatedGsap` labelled "Simulated GSAP" (import history, transaction panel) |
