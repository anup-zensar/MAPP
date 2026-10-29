-- name: ListActions :many
SELECT id, round_id, round_number, subject_id, action, level, level_name, approvers, actor_id, actor_name,
    on_behalf_of, at, comment, changes, capacity, as_delegate_of_id, as_delegate_of_name, rule, escalated_to_id,
    escalated_to_name, waited_days, subject_kind, self_approval
FROM approval_actions
WHERE sqlc.narg(subject_id)::text IS NULL OR subject_id = sqlc.narg(subject_id)
ORDER BY length(id), id COLLATE "C";

-- name: ListActionsOfSubjects :many
SELECT id, round_id, round_number, subject_id, action, level, level_name, approvers, actor_id, actor_name,
    on_behalf_of, at, comment, changes, capacity, as_delegate_of_id, as_delegate_of_name, rule, escalated_to_id,
    escalated_to_name, waited_days, subject_kind, self_approval
FROM approval_actions
WHERE subject_id = ANY(sqlc.arg(subject_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: ListActionsInScope :many
SELECT id, round_id, round_number, subject_id, action, level, level_name, approvers, actor_id, actor_name,
    on_behalf_of, at, comment, changes, capacity, as_delegate_of_id, as_delegate_of_name, rule, escalated_to_id,
    escalated_to_name, waited_days, subject_kind, self_approval
FROM approval_actions
WHERE subject_kind = 'activity'
    AND activity_in_scope(subject_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: InsertAction :exec
INSERT INTO approval_actions (
    id, round_id, round_number, subject_id, action, level, level_name, approvers, actor_id, actor_name,
    on_behalf_of, at, comment, changes, capacity, as_delegate_of_id, as_delegate_of_name, rule, escalated_to_id,
    escalated_to_name, waited_days, subject_kind, self_approval
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
);
