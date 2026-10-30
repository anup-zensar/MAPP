# Contract: Performance Data (feature 046)

These operations belong to the **wave B contract**. This lane writes them here and hands them to the
integrator, who applies them to `backend/api/openapi.yaml` **before 048–050 start** (build order step 7).
This lane never edits `openapi.yaml` itself. Paths are under `/api/v1`. One error envelope throughout.

Remove this feature's operations from `notYetServed` in `internal/app/app_test.go` as each is served — they
are listed in a group per feature, so the deletions never touch another lane's lines.

## Data sources

| Method | Path | Permission | Notes |
|---|---|---|---|
| `GET` | `/performance/sources` | activity read | find by name, category, status, delivery route; includes derived freshness |
| `POST` | `/performance/sources` | `performance.import` | |
| `GET` | `/performance/sources/{sourceId}` | activity read | |
| `PATCH` | `/performance/sources/{sourceId}` | `performance.import` | version-checked; `version_conflict` on a stale update |
| `DELETE` | `/performance/sources/{sourceId}` | `performance.import` | **409** while the source holds facts, naming Deactivate as the alternative (FR-002) |
| `POST` | `/performance/sources/{sourceId}/deactivate` | `performance.import` | keeps facts, refuses uploads, updates *Is integrated* at once |
| `POST` | `/performance/sources/{sourceId}/reactivate` | `performance.import` | |

## Column mappings

| Method | Path | Permission | Notes |
|---|---|---|---|
| `GET` | `/performance/sources/{sourceId}/mappings` | activity read | |
| `POST` | `/performance/sources/{sourceId}/mappings` | `performance.import` | **409** when another active source already supplies that model's stage or revenue (FR-003) |
| `PATCH` | `/performance/mappings/{mappingId}` | `performance.import` | future loads only |
| `DELETE` | `/performance/mappings/{mappingId}` | `performance.import` | |
| `GET` | `/performance/sources/{sourceId}/template` | activity read | the blank file template (FR-008); returns a file |

## Loading

| Method | Path | Permission | Notes |
|---|---|---|---|
| `POST` | `/performance/sources/{sourceId}/imports` | `performance.import` | multipart file upload; answers the summary with all five counts |
| `POST` | `/performance/sources/{sourceId}/simulated-pull` | `performance.import` | the explicit trigger (046-C13); same response shape as an upload |
| `GET` | `/performance/imports` | activity read | find by file, source, uploader, status, date |
| `GET` | `/performance/imports/{importId}` | activity read | the summary; **no** PATCH — an import is not editable (FR-016) |
| `POST` | `/performance/imports/{importId}/undo` | `performance.import` | confirmation carries the counts; restores replaced versions; import stays listed as *Undone* |

A file carrying personal-data columns is refused **422** in full, naming the offending column, with no row
stored (FR-012).

## Rows to review

| Method | Path | Permission | Notes |
|---|---|---|---|
| `GET` | `/performance/review-rows` | activity read | filter by key, source, reason, upload, date, status. **No POST** — rows are never created by hand (FR-023) |
| `GET` | `/performance/review-rows/{rowId}` | activity read | includes history |
| `POST` | `/performance/review-rows/{rowId}/correct` | `performance.import` | re-checks immediately; becomes *Resolved* on a match |
| `POST` | `/performance/review-rows/{rowId}/dismiss` | `performance.import` | comment **required**; **422** without one |
| `POST` | `/performance/review-rows/{rowId}/reopen` | `performance.import` | |
| `POST` | `/performance/sources/{sourceId}/recheck` | `performance.import` | re-checks open rows and held facts on demand (FR-025) |

## Performance on an activity

| Method | Path | Permission | Notes |
|---|---|---|---|
| `GET` | `/activities/{activityId}/performance` | activity read | **gated**: an activity not counting as approved answers the stated reason and no figures (FR-027). Every empty state names its reason (FR-032) |
| `GET` | `/activities/{activityId}/performance/facts` | activity read | searchable list: date, source, stage or metric, value, currency, key matched on, import |
| `GET` | `/activities/{activityId}/performance/facts/export` | activity read | |
| `GET` | `/activities/{activityId}/performance/sources` | activity read | external name and aliases, media-code count, the two flags, per-source integration ID, fact count, last fact date |

## External names and integration IDs

| Method | Path | Permission | Notes |
|---|---|---|---|
| `PUT` | `/activities/{activityId}/external-name` | `performance.link` | sets by hand; **409** on a clash or overlong name, with the reason (FR-036) |
| `POST` | `/activities/{activityId}/external-name/regenerate` | `performance.link` | switches back to the generated name |
| `GET` | `/activities/{activityId}/integration-ids` | activity read | |
| `PUT` | `/activities/{activityId}/integration-ids/{sourceId}` | `performance.link` | **409** when one already exists for that source (FR-037) |
| `DELETE` | `/activities/{activityId}/integration-ids/{sourceId}` | `performance.link` | confirmation states the effect on matching |

## Media codes

| Method | Path | Permission | Notes |
|---|---|---|---|
| `GET` | `/activities/{activityId}/media-codes` | activity read | present only where the activity type requires codes (FR-040) |
| `POST` | `/activities/{activityId}/media-codes` | `performance.link` | every missing required attribute reported at once |
| `PATCH` | `/media-codes/{codeId}` | `performance.link` | attributes and landing page only |
| `DELETE` | `/media-codes/{codeId}` | `performance.link` | **409** once used; Deactivate offered instead |
| `POST` | `/media-codes/{codeId}/deactivate` | `performance.link` | |

## Configuration (Settings › Performance data)

| Method | Path | Permission | Notes |
|---|---|---|---|
| `GET` | `/performance/configuration` | activity read | activity types requiring codes, the attribute catalogue, the media-code start number (046-C12) |
| `PATCH` | `/performance/configuration` | `performance.import` | |

## Cross-feature notes for the integrator

1. **Two new permissions**, `performance.import` and `performance.link` (D-367), each needing a route rule in
   the shared `routes_measure.go`, which this lane does not edit.
2. **The approvals area must call this feature's release port** when an activity reaches an approved status,
   so held facts are released in the same transaction (research R4). That change is outside this feature's
   folders.
3. **The activity copy path must not carry** external name, aliases, integration IDs, media codes or facts
   (FR-038) — a change in the activity area, reported here rather than made.
