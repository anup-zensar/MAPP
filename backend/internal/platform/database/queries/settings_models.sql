-- name: GetModel :one
SELECT sqlc.embed(models),
    coalesce((SELECT array_agg(t.activity_type_id ORDER BY t.position) FROM model_activity_types t
        WHERE t.model_id = models.id), '{}')::text[] AS activity_type_ids
FROM models
WHERE id = $1;

-- name: ListModels :many
SELECT sqlc.embed(models),
    coalesce((SELECT array_agg(t.activity_type_id ORDER BY t.position) FROM model_activity_types t
        WHERE t.model_id = models.id), '{}')::text[] AS activity_type_ids
FROM models
ORDER BY length(id), id COLLATE "C";

-- name: PutModel :exec
INSERT INTO models (
    id, name, inq_to_mql_rate_hundredths, mql_to_deal_rate_hundredths, average_deal_size_minor,
    average_deal_size_currency, active, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    inq_to_mql_rate_hundredths = excluded.inq_to_mql_rate_hundredths,
    mql_to_deal_rate_hundredths = excluded.mql_to_deal_rate_hundredths,
    average_deal_size_minor = excluded.average_deal_size_minor,
    average_deal_size_currency = excluded.average_deal_size_currency,
    active = excluded.active,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteModelActivityTypes :exec
DELETE FROM model_activity_types
WHERE model_id = $1;

-- name: InsertModelActivityTypes :exec
INSERT INTO model_activity_types (model_id, position, activity_type_id)
SELECT sqlc.arg(model_id), ordinality::integer, activity_type_id
FROM unnest(sqlc.arg(activity_type_ids)::text[]) WITH ORDINALITY AS t(activity_type_id, ordinality);

-- name: DeleteModel :exec
DELETE FROM models
WHERE id = $1;

-- name: GetActivityType :one
SELECT id, name, default_planned_cost_minor, default_planned_cost_currency, active, version, created_at,
    created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM activity_types
WHERE id = $1;

-- name: ListActivityTypes :many
SELECT id, name, default_planned_cost_minor, default_planned_cost_currency, active, version, created_at,
    created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM activity_types
ORDER BY length(id), id COLLATE "C";

-- name: PutActivityType :exec
INSERT INTO activity_types (
    id, name, default_planned_cost_minor, default_planned_cost_currency, active, version, created_at,
    created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    default_planned_cost_minor = excluded.default_planned_cost_minor,
    default_planned_cost_currency = excluded.default_planned_cost_currency,
    active = excluded.active,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteActivityType :exec
DELETE FROM activity_types
WHERE id = $1;
