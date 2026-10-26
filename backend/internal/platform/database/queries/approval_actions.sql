-- name: ListActions :many
SELECT id, round_id, round_number, activity_id, action, level, level_name, approvers, actor_id, actor_name,
    on_behalf_of, at, comment, changes, capacity, as_delegate_of_id, as_delegate_of_name, rule, escalated_to_id,
    escalated_to_name, waited_days
FROM approval_actions
WHERE sqlc.narg(activity_id)::text IS NULL OR activity_id = sqlc.narg(activity_id)
ORDER BY length(id), id COLLATE "C";

-- name: ListActionsInScope :many
SELECT id, round_id, round_number, activity_id, action, level, level_name, approvers, actor_id, actor_name,
    on_behalf_of, at, comment, changes, capacity, as_delegate_of_id, as_delegate_of_name, rule, escalated_to_id,
    escalated_to_name, waited_days
FROM approval_actions
WHERE activity_in_scope(activity_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
    sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: PutAction :exec
INSERT INTO approval_actions (
    id, round_id, round_number, activity_id, action, level, level_name, approvers, actor_id, actor_name,
    on_behalf_of, at, comment, changes, capacity, as_delegate_of_id, as_delegate_of_name, rule, escalated_to_id,
    escalated_to_name, waited_days
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
)
ON CONFLICT (id) DO UPDATE SET
    round_id = excluded.round_id,
    round_number = excluded.round_number,
    activity_id = excluded.activity_id,
    action = excluded.action,
    level = excluded.level,
    level_name = excluded.level_name,
    approvers = excluded.approvers,
    actor_id = excluded.actor_id,
    actor_name = excluded.actor_name,
    on_behalf_of = excluded.on_behalf_of,
    at = excluded.at,
    comment = excluded.comment,
    changes = excluded.changes,
    capacity = excluded.capacity,
    as_delegate_of_id = excluded.as_delegate_of_id,
    as_delegate_of_name = excluded.as_delegate_of_name,
    rule = excluded.rule,
    escalated_to_id = excluded.escalated_to_id,
    escalated_to_name = excluded.escalated_to_name,
    waited_days = excluded.waited_days;
