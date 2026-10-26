-- name: ListActivities :many
SELECT id, initiative_id, name, external_name, activity_type_id, owner_id, start_date, end_date, ongoing,
    planned_cost_minor, planned_cost_currency, approval_workflow_id, description, attributes, inq,
    set_later_stages_myself, mql, revenue_minor, revenue_currency, status, split_from_id, version, created_at,
    created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM activities
WHERE (sqlc.narg(ids)::text[] IS NULL OR id = ANY (sqlc.narg(ids)::text[]))
    AND (sqlc.narg(initiative_ids)::text[] IS NULL OR initiative_id = ANY (sqlc.narg(initiative_ids)::text[]))
ORDER BY length(id), id COLLATE "C";

-- name: ListActivitiesInScope :many
SELECT id, initiative_id, name, external_name, activity_type_id, owner_id, start_date, end_date, ongoing,
    planned_cost_minor, planned_cost_currency, approval_workflow_id, description, attributes, inq,
    set_later_stages_myself, mql, revenue_minor, revenue_currency, status, split_from_id, version, created_at,
    created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM activities
WHERE (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
    AND activity_in_scope(id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: PutActivity :exec
INSERT INTO activities (
    id, initiative_id, name, external_name, activity_type_id, owner_id, start_date, end_date, ongoing,
    planned_cost_minor, planned_cost_currency, approval_workflow_id, description, attributes, inq,
    set_later_stages_myself, mql, revenue_minor, revenue_currency, status, split_from_id, version, created_at,
    created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24,
    $25, $26, $27, $28
)
ON CONFLICT (id) DO UPDATE SET
    initiative_id = excluded.initiative_id,
    name = excluded.name,
    external_name = excluded.external_name,
    activity_type_id = excluded.activity_type_id,
    owner_id = excluded.owner_id,
    start_date = excluded.start_date,
    end_date = excluded.end_date,
    ongoing = excluded.ongoing,
    planned_cost_minor = excluded.planned_cost_minor,
    planned_cost_currency = excluded.planned_cost_currency,
    approval_workflow_id = excluded.approval_workflow_id,
    description = excluded.description,
    attributes = excluded.attributes,
    inq = excluded.inq,
    set_later_stages_myself = excluded.set_later_stages_myself,
    mql = excluded.mql,
    revenue_minor = excluded.revenue_minor,
    revenue_currency = excluded.revenue_currency,
    status = excluded.status,
    split_from_id = excluded.split_from_id,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteActivity :exec
DELETE FROM activities
WHERE id = $1;

-- name: AddActivities :copyfrom
INSERT INTO activities (
    id, initiative_id, name, external_name, activity_type_id, owner_id, start_date, end_date, ongoing,
    planned_cost_minor, planned_cost_currency, approval_workflow_id, description, attributes, inq,
    set_later_stages_myself, mql, revenue_minor, revenue_currency, status, split_from_id, version, created_at,
    created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24,
    $25, $26, $27, $28
);

-- name: DeleteActivities :exec
DELETE FROM activities
WHERE id = ANY (sqlc.arg(ids)::text[]);
