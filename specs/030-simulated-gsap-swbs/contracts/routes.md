# Routes, permissions and interfaces of feature 030

## API operations (014 Table C, `internal/access/routes_gsap.go`)

| Operation | Rule | Service check | Answers |
|---|---|---|---|
| `GET /api/v1/simulated-gsap/swbs-runs?limit=` | `need(finance.import)` | `authz.Require(finance.import)` | 200 `SwbsRunPage` · 400 (limit) · 401 · 403 |
| `POST /api/v1/simulated-gsap/swbs-runs` | `need(finance.import)` | `authz.Require(finance.import)`; scope `authz.PermissionScopeOf(finance.import)` | 201 `SwbsRun` · 400 · 401 · 403 · 404 · 409 `not_allowed` |

Changed: `GET /api/v1/line-items/{lineItemId}`, `GET /api/v1/activities/{activityId}/line-items`,
`PATCH /api/v1/line-items/{lineItemId}`, the line-item create and duplicate answers — `LineItem` gains
`swbsCode`, `swbsRequestedAt`, `swbsReceivedAt`, `hxKey`; `SwbsStatus` gains `received`. A PATCH that
switches `requestSwbs` off while a code exists answers 400 `validation_failed` on `requestSwbs`.

## Messages

| When | Text |
|---|---|
| PATCH switches the request off with a code | "This line item already has its SWBS code from the simulated GSAP, so the request stays on." |
| Named line item not ready | "Not sent — these line items aren't ready to send to the simulated GSAP: Stand build (L-1006). Reload to see their SWBS status." |
| Named line item unknown or out of scope | "This line item doesn't exist or you can't see it." |
| Empty `lineItemIds` / too many / bad ID | "Name at least one line item, or leave the list out to send every ready line item." · "Send at most 200 line items at a time." · "Use a line-item ID such as L-1006." |
| `limit` not 1–100 | "Ask for 1 to 100 runs." |
| Log details | "Simulated GSAP returned 2 SWBS codes: Stand build (L-1006) SIM-GB01-0001; Media booking (L-1005) SIM-NL02-0001." · "Nothing was ready to send." |

## Web routes and components

| Route | Component | Change |
|---|---|---|
| `/financials` | `FinancialsPage` | renders `SimulatedGsapPanel` (from `@/features/simulated-gsap`) for people with `finance.import` |
| `/line-items/:lineItemId` | `LineItemPage` | renders `SwbsCard` (status, code, dates, source, `HxKey`, `SendNowButton`) |
| `/line-items/:lineItemId/edit` | `LineItemEditPage` → `LineItemForm` | request switch locked with the reason once a code exists; read-only code |

`@/features/simulated-gsap` exports:

- `SimulatedGsapPanel()` — no props; reads the runs and sends all ready line items.
- `SendNowButton({ lineItemId, lineItemName })` — the line item page's action with its confirmation; on
  success the line item's queries are refreshed.
