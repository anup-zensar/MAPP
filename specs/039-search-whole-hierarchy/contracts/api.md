# API contract: Search the whole hierarchy

The full definitions are in `backend/api/openapi.yaml`.

## `GET /api/v1/plans/{planId}/search?q=<text>`

Route rule: `view()`, as `GET /api/v1/plans/{planId}`.

| Status | When |
|---|---|
| 200 | `PlanSearch`: `{ "items": [PlanSearchMatch] }`, top-down (campaign, initiative, activity, line item), then by name and ID |
| 400 | `validation_failed` on `q`: empty after trimming, or over 200 characters |
| 404 | the plan does not exist, or is outside the actor's scope |

`PlanSearchMatch`: `{ "level": "campaign" | "initiative" | "activity" | "lineItem", "id": "L-1006", "name":
"Stand build", "parentId": "A-1004" }` — `parentId` is the plan for a campaign, the campaign for an initiative,
the initiative for an activity and the activity for a line item.

## `GET /api/v1/plans?q=<text>` (changed)

A plan is listed when its name or ID contains `q`, or when a record below it does. `PlanSummary.foundIn`
(optional) is set only on a plan listed through its contents:

```json
"foundIn": { "match": { "level": "lineItem", "id": "L-1001", "name": "Venue hire", "parentId": "A-1001" }, "more": 2 }
```
