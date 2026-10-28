# Routes, permissions and web interfaces (013)

## API operations → access rule

| Operation | Rule | File | Service check |
|---|---|---|---|
| `GET /api/v1/plans/{planId}/spa` | view | `backend/internal/access/routes_spa.go` | the plan is read in the actor's scope (404 otherwise); a campaign not in the plan is 404; financial figures need `finance.view` in the plan; purchase requests follow 032's view scope |
| `GET /api/v1/plans` (`bcpir` added) | signedIn (unchanged) | `routes_planning.go` | unchanged |
| `GET /api/v1/all-work`, `/export`, `/breakdown` (`bcpir` added) | unchanged | `routes_views.go` | unchanged |

## Web routes

| Route | Page | Entry points |
|---|---|---|
| `/plans/:planId/spa?campaign=C-…&dimension=…&period=month` | `SpaPage` (`features/spa`) | plan page header ("SP&A view"), each campaign panel ("SP&A view" with the campaign scope) |

URL state (Zod, `features/spa/schemas.ts`): `campaign` (a campaign ID or absent), `dimension` (a key or absent),
`period` (`quarter` default, `month`). Changing any of them pushes a history entry, so Back returns to the
previous choice.
