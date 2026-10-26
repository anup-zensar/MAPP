-- name: GetWorkflow :one
SELECT id, name, description, active, reminder_days, escalation_days, version, created_at, created_by_id,
    created_by_name, updated_at, updated_by_id, updated_by_name
FROM approval_workflows
WHERE id = $1;

-- name: ListWorkflows :many
SELECT id, name, description, active, reminder_days, escalation_days, version, created_at, created_by_id,
    created_by_name, updated_at, updated_by_id, updated_by_name
FROM approval_workflows
ORDER BY length(id), id COLLATE "C";

-- name: ListWorkflowLevels :many
SELECT sqlc.embed(approval_workflow_levels),
    coalesce((SELECT array_agg(a.person_id ORDER BY a.position) FROM approval_workflow_level_approvers a
        WHERE a.workflow_id = approval_workflow_levels.workflow_id
            AND a.level_number = approval_workflow_levels.number), '{}')::text[] AS approver_ids
FROM approval_workflow_levels
WHERE sqlc.narg(workflow_id)::text IS NULL OR workflow_id = sqlc.narg(workflow_id)
ORDER BY workflow_id, number;

-- name: PutWorkflow :exec
INSERT INTO approval_workflows (
    id, name, description, active, reminder_days, escalation_days, version, created_at, created_by_id,
    created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    active = excluded.active,
    reminder_days = excluded.reminder_days,
    escalation_days = excluded.escalation_days,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteWorkflowLevels :exec
DELETE FROM approval_workflow_levels
WHERE workflow_id = $1;

-- name: InsertWorkflowLevel :exec
INSERT INTO approval_workflow_levels (
    workflow_id, number, name, approver_type, role_id, mode, applies_when, auto_approve_when, backup_approver_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- name: InsertWorkflowLevelApprovers :exec
INSERT INTO approval_workflow_level_approvers (workflow_id, level_number, position, person_id)
SELECT sqlc.arg(workflow_id), sqlc.arg(level_number), ordinality::integer, person_id
FROM unnest(sqlc.arg(person_ids)::text[]) WITH ORDINALITY AS t(person_id, ordinality);

-- name: DeleteWorkflow :exec
DELETE FROM approval_workflows
WHERE id = $1;
