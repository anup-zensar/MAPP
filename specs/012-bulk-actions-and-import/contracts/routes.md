# Routes, permissions and web interfaces (012)

The operations are in `backend/api/openapi.yaml` (tags *Bulk actions* and *Imports*); the fragment of 2026-10-19
was merged there (D-192) and is not kept here. 012 re-shapes the import schemas (never served before, so no
client breaks) and adds `GET /api/v1/plans/{planId}/export`.

## API operation → access rule

| Operation | Rule | Service check |
|---|---|---|
| `GET /api/v1/bulk-actions/fields` | signedIn | none (Settings lists) |
| `POST /api/v1/bulk-actions/targets` | signedIn | targets the single destinations offer the person (their edit permission in the target's scope) |
| `POST /api/v1/bulk-actions/preview` | custom | each record through its single operation's checks |
| `POST /api/v1/bulk-actions` | custom | idem |
| `POST /api/v1/bulk-actions/{bulkActionId}/undo` | custom | only the actor's own delete action (404 otherwise); each bin entry through `history.Undo` |
| `GET /api/v1/plans/{planId}/imports` | view | the plan in the person's view scope (404) |
| `POST /api/v1/plans/{planId}/imports/check` | inPlan `activities.edit` | the plan in scope; each row through its single operation's checks |
| `POST /api/v1/plans/{planId}/imports` | inPlan `activities.edit` | idem |
| `GET /api/v1/plans/{planId}/import-template` | inPlan `activities.edit` | the plan in scope |
| `GET /api/v1/plans/{planId}/export` (new) | inPlan `data.export` | the plan in the `data.export` scope (404 when unseen, 403 when seen without it) |
| `GET /api/v1/imports/{importId}` | view | the import's plan in the view scope (404) |
| `POST /api/v1/imports/{importId}/undo` | inPlan `activities.delete` | the import's plan in scope; each delete through its service |

The existing rules stay in `access/routes_planning.go` (`bulkRoutes`); the new one is in
`access/routes_workbook.go`. Body limits (`app/transport.go`): 7 MB for the check and the import.

## Web interfaces

| Where | Change |
|---|---|
| All work grid | checkbox column, header checkbox, selection bar and dialogs (`features/bulk`) |
| Plan page ⋯ menu | **Import from Excel** (to `/plans/:planId/import`, with `activities.edit`), **Export to Excel** (with `data.export`) |
| `/plans/:planId/import` | Import page (`features/workbook`) |
| `/imports/:importId` | Import results page |
