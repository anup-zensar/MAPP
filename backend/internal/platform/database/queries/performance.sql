-- name: ListPerformanceSources :many
SELECT sqlc.embed(performance_sources),
    (SELECT count(*) FROM performance_facts f WHERE f.source_id = performance_sources.id)::integer AS fact_count
FROM performance_sources
WHERE sqlc.narg(id)::text IS NULL OR performance_sources.id = sqlc.narg(id)
ORDER BY lower(performance_sources.name COLLATE "C"), performance_sources.id COLLATE "C";

-- Freshness is derived, and a source that has never loaded simply has no row here, so nothing is NULL.
-- name: SourceFreshness :many
SELECT i.source_id, max(i.created_at)::timestamptz AS last_loaded_at, max(i.data_through)::date AS data_through
FROM performance_imports i
WHERE i.status = 'completed' AND i.data_through IS NOT NULL
GROUP BY i.source_id;

-- name: PutPerformanceSource :exec
INSERT INTO performance_sources (
    id, name, description, owner_id, category, status, delivery_route, cadence, integration_id,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name, description = excluded.description, owner_id = excluded.owner_id,
    category = excluded.category, status = excluded.status, delivery_route = excluded.delivery_route,
    cadence = excluded.cadence, integration_id = excluded.integration_id, version = excluded.version,
    updated_at = excluded.updated_at, updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeletePerformanceSource :execrows
DELETE FROM performance_sources
WHERE performance_sources.id = sqlc.arg(source_id)
    AND NOT EXISTS (SELECT 1 FROM performance_facts f WHERE f.source_id = performance_sources.id);

-- name: ListPerformanceMappings :many
SELECT sqlc.embed(performance_source_mappings)
FROM performance_source_mappings
WHERE source_id = ANY (sqlc.arg(source_ids)::text[])
ORDER BY source_id COLLATE "C", lower(column_name COLLATE "C");

-- name: ActiveSuppliersForModel :many
SELECT m.id, m.source_id, m.meaning, m.stage, s.name AS source_name
FROM performance_source_mappings m
JOIN performance_sources s ON s.id = m.source_id
WHERE m.model_id = sqlc.arg(model_id)
    AND m.meaning = ANY (ARRAY['stage', 'revenue'])
    AND s.status = 'active'
ORDER BY m.id COLLATE "C";

-- name: PutPerformanceMapping :exec
INSERT INTO performance_source_mappings (
    id, source_id, column_name, meaning, stage, web_metric, model_id,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (id) DO UPDATE SET
    column_name = excluded.column_name, meaning = excluded.meaning, stage = excluded.stage,
    web_metric = excluded.web_metric, model_id = excluded.model_id, version = excluded.version,
    updated_at = excluded.updated_at, updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeletePerformanceMapping :exec
DELETE FROM performance_source_mappings WHERE id = $1;

-- name: MatchActivitiesByID :many
SELECT id AS activity_id, id AS key
FROM activities
WHERE id = ANY (sqlc.arg(keys)::text[])
    AND activity_in_scope(id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[]);

-- name: MatchActivitiesByMediaCode :many
SELECT c.activity_id, c.code::text AS key
FROM media_codes c
WHERE c.code = ANY (sqlc.arg(codes)::bigint[])
    AND activity_in_scope(c.activity_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[]);

-- name: MatchActivitiesByIntegrationID :many
SELECT k.activity_id, k.value AS key
FROM activity_integration_ids k
WHERE k.source_id = sqlc.arg(source_id)
    AND lower(k.value COLLATE "C") = ANY (sqlc.arg(keys)::text[])
    AND activity_in_scope(k.activity_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[]);

-- name: MatchActivitiesByExternalName :many
SELECT a.id AS activity_id, lower(a.external_name COLLATE "C") AS key
FROM activities a
WHERE a.external_name <> '' AND lower(a.external_name COLLATE "C") = ANY (sqlc.arg(keys)::text[])
    AND activity_in_scope(a.id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
UNION
SELECT n.activity_id, lower(n.name COLLATE "C") AS key
FROM activity_external_name_aliases n
WHERE lower(n.name COLLATE "C") = ANY (sqlc.arg(keys)::text[])
    AND activity_in_scope(n.activity_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[]);

-- name: ListFactsForActivity :many
SELECT sqlc.embed(performance_facts), s.name AS source_name
FROM performance_facts
JOIN performance_sources s ON s.id = performance_facts.source_id
WHERE performance_facts.activity_id = sqlc.arg(activity_id)
    AND performance_facts.replaced_by_id IS NULL
    AND (NOT sqlc.arg(counted_only)::boolean OR NOT performance_facts.held)
    AND activity_in_scope(performance_facts.activity_id, sqlc.arg(sees_everything)::boolean,
        sqlc.arg(viewer_id)::text, sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY performance_facts.fact_date DESC, performance_facts.id COLLATE "C";

-- name: ReleaseHeldFactsForActivity :execrows
UPDATE performance_facts
SET held = false, version = version + 1, updated_at = sqlc.arg(updated_at),
    updated_by_id = sqlc.arg(updated_by_id), updated_by_name = sqlc.arg(updated_by_name)
WHERE activity_id = sqlc.arg(activity_id) AND held;

-- name: ListPerformanceImports :many
SELECT sqlc.embed(performance_imports)
FROM performance_imports
WHERE (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
    AND (sqlc.narg(source_id)::text IS NULL OR source_id = sqlc.narg(source_id))
ORDER BY created_at DESC, id COLLATE "C"
LIMIT sqlc.arg(max_imports);

-- name: PutPerformanceImport :exec
INSERT INTO performance_imports (
    id, source_id, file_name, origin, status, data_through, rows_read, rows_matched, rows_held,
    rows_unattributed, rows_rejected, rows_unmatched,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
ON CONFLICT (id) DO UPDATE SET
    status = excluded.status, version = excluded.version, updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id, updated_by_name = excluded.updated_by_name;

-- name: ListPerformanceReviewRows :many
SELECT sqlc.embed(performance_review_rows), s.name AS source_name
FROM performance_review_rows
JOIN performance_sources s ON s.id = performance_review_rows.source_id
WHERE (sqlc.narg(status)::text IS NULL OR performance_review_rows.status = sqlc.narg(status))
    AND (sqlc.narg(source_id)::text IS NULL OR performance_review_rows.source_id = sqlc.narg(source_id))
ORDER BY performance_review_rows.created_at DESC, performance_review_rows.id COLLATE "C"
LIMIT sqlc.arg(max_rows);

-- name: ListMediaCodesForActivity :many
SELECT sqlc.embed(media_codes)
FROM media_codes
WHERE activity_id = sqlc.arg(activity_id)
    AND activity_in_scope(activity_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY code;

-- name: TakeNextMediaCode :one
UPDATE performance_configuration
SET media_code_next = media_code_next + 1, version = version + 1, updated_at = sqlc.arg(updated_at),
    updated_by_id = sqlc.arg(updated_by_id), updated_by_name = sqlc.arg(updated_by_name)
WHERE id = 'performance'
RETURNING (media_code_next - 1)::bigint AS code;

-- name: PutPerformanceFact :exec
INSERT INTO performance_facts (
    id, source_id, import_id, source_reference, activity_id, matched_on, fact_date, stage, web_metric,
    count_value, amount_minor, currency, held, replaced_by_id,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
ON CONFLICT (id) DO UPDATE SET
    activity_id = excluded.activity_id, matched_on = excluded.matched_on, held = excluded.held,
    replaced_by_id = excluded.replaced_by_id, version = excluded.version, updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id, updated_by_name = excluded.updated_by_name;

-- name: PutPerformanceReviewRow :exec
INSERT INTO performance_review_rows (
    id, import_id, source_id, raw, reason, candidates, status, comment,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (id) DO UPDATE SET
    status = excluded.status, comment = excluded.comment, version = excluded.version,
    updated_at = excluded.updated_at, updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: CurrentFactByReference :one
SELECT sqlc.embed(performance_facts)
FROM performance_facts
WHERE source_id = sqlc.arg(source_id) AND source_reference = sqlc.arg(source_reference)
    AND replaced_by_id IS NULL;

-- name: PutMediaCode :exec
INSERT INTO media_codes (
    id, code, activity_id, attributes, landing_page, status,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (id) DO UPDATE SET
    attributes = excluded.attributes, landing_page = excluded.landing_page, status = excluded.status,
    version = excluded.version, updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id, updated_by_name = excluded.updated_by_name;

-- name: PutIntegrationID :exec
INSERT INTO activity_integration_ids (
    id, activity_id, source_id, value,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO UPDATE SET
    value = excluded.value, version = excluded.version, updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id, updated_by_name = excluded.updated_by_name;
