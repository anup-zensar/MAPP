# Data model: Performance Data from a Simulated Feed

Three migrations are reserved for this feature: **00030–00032**. The build guide §2 names the four headline
tables (`performance_sources`, `performance_imports`, `performance_facts`, `media_codes`); the four supporting
tables below carry the mappings, the review queue and the two matching keys that live on an activity. They are
grouped into the reserved migrations rather than taking further numbers another lane may hold.

Every table embeds `record.Meta` (version, created/updated) where its records are edited by people, registers
a `history.Kind`, and writes one activity-log entry per change inside the same transaction.

## 00030 — sources and mappings

### `performance_sources`

| Column | Type | Notes |
|---|---|---|
| `id` | text, PK | |
| `name`, `description` | text | name unique |
| `owner_id` | text → `people` | |
| `category` | text | web analytics, marketing automation, CRM, lead progression |
| `status` | text | `active`, `inactive` |
| `delivery_route` | text | `upload`, `simulated` — `simulated` is what enables a pull (046-C13) |
| `cadence` | text | drives the *Stale* threshold in FR-005 |
| `integration_id` | text, nullable | the source's own id, unique where present |

Freshness (FR-005) is **derived**, never stored: last successful import, its data-through date, the cadence
and the resulting *Current · Stale · Failed · Inactive*. Storing it would go stale on its own.

### `performance_source_mappings`

| Column | Type | Notes |
|---|---|---|
| `id` | text, PK | |
| `source_id` | text → `performance_sources`, cascade | |
| `column_name` | text | the file's column heading |
| `meaning` | text | `key`, `reference`, `fact_date`, `stage`, `revenue`, `currency`, `web_metric`, `journey` |
| `stage` / `web_metric` | text, nullable | `web_metric` ∈ sessions, visits, engagement touches (046-C11) |
| `model_id` | text, nullable → `models` | which model this column supplies |

**Constraint**: at most one *active* source per (model, stage) and per (model, revenue) — FR-003. Enforced by a
partial unique index so the refusal is true in the database, not only in the service.

## 00031 — imports, facts and the review queue

### `performance_imports`

| Column | Type | Notes |
|---|---|---|
| `id` | text, PK | |
| `source_id` | text → `performance_sources` | |
| `file_name`, `uploaded_by`, `uploaded_at` | text/timestamptz | `uploaded_by` → `people` |
| `origin` | text | `upload` or `simulated` — both travel the same path |
| `status` | text | `completed`, `undone` |
| `rows_read` and the five counts | integer | matched, held, unattributed, rejected, unmatched |

**Constraint**: `rows_read = matched + held + unattributed + rejected + unmatched` as a CHECK — FR-022's
reconciliation is an invariant, so it belongs in the schema (constitution B5).

### `performance_facts`

| Column | Type | Notes |
|---|---|---|
| `id` | text, PK | |
| `source_id`, `import_id` | text → above | |
| `source_reference` | text | the source's own row id |
| `activity_id` | text → `activities`, nullable | null only while unattributed |
| `matched_on` | text, nullable | `activity_id`, `media_code`, `integration_id`, `external_name` |
| `fact_date` | date | the fiscal year comes from this, never the activity (FR-015) |
| `stage` / `web_metric` | text, nullable | exactly one of the two is set |
| `count_value` | integer, nullable | non-negative, whole — a fraction is rejected at load |
| `amount_minor`, `currency` | bigint / text | integer minor units, never a float |
| `held` | boolean | released on approval (research R4); indexed, every figure filters on it |
| `replaced_by_id` | text, nullable | the audit trail an undo walks back |

**Constraint**: unique on (`source_id`, `source_reference`) among rows not yet replaced — the idempotency
identity from research R3.

### `performance_review_rows`

| Column | Type | Notes |
|---|---|---|
| `id` | text, PK | |
| `import_id`, `source_id` | text → above | |
| `raw` | jsonb | the row exactly as read, so a correction can be compared with the original |
| `reason` | text | `rejected:<validation>`, `unmatched:unknown`, `unmatched:ambiguous`, `unmatched:disagree` |
| `candidates` | jsonb, nullable | the activities named when keys disagree (FR-019) |
| `status` | text | `open`, `resolved`, `dismissed` |
| `comment` | text, nullable | **required** when status is `dismissed` (FR-024) — a CHECK enforces it |

## 00032 — media codes and the two activity-level keys

### `media_codes`

| Column | Type | Notes |
|---|---|---|
| `id` | text, PK | |
| `code` | bigint, unique | system-generated, never reused, from a configurable start (FR-042) |
| `activity_id` | text → `activities` | never a line item or any other level (FR-040) |
| `attributes` | jsonb | the configured catalogue's values |
| `landing_page` | text, nullable | |
| `status` | text | `active`, `inactive` — deactivated rather than deleted once used |

The generator's next number lives in the Settings › Performance data configuration, alongside the list of
activity types that require codes and the attribute catalogue (046-C12).

### `activity_external_names`

| Column | Type | Notes |
|---|---|---|
| `activity_id` | text → `activities` | |
| `name` | text | unique across names **and** aliases (FR-036) |
| `is_current` | boolean | exactly one current per activity; superseded rows stay as matching aliases |
| `set_by_hand` | boolean | suppresses regeneration until the user switches back (FR-035) |

### `activity_integration_ids`

| Column | Type | Notes |
|---|---|---|
| `activity_id`, `source_id` | text → above | **unique together** — at most one per source (FR-037) |
| `value` | text | unique within the source |

## Matching keys, in resolution order

| Order | Key | Lives in |
|---|---|---|
| 1 | Activity ID | `activities.id` |
| 2 | Media code | `media_codes.code` |
| 3 | Integration ID | `activity_integration_ids.value`, scoped to the source |
| 4 | External name | `activity_external_names.name`, current **or** alias |

All four are resolved set-wise per batch (research R2). A key resolving to more than one activity is
*ambiguous*; two keys resolving to different activities *disagree*; neither is ever guessed (FR-019).
