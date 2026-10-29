-- name: ListRounds :many
SELECT id, subject_id, number, workflow_id, workflow_name, levels, current_level, outcome, submitted_at,
    submitted_by_id, submitted_by_name, comment, ended_at, facts, reminder_days, escalation_days, auto_approved,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name, subject_kind
FROM approval_rounds
WHERE (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
    AND (sqlc.narg(subject_id)::text IS NULL OR subject_id = sqlc.narg(subject_id))
ORDER BY CASE WHEN sqlc.narg(subject_id)::text IS NULL THEN 0 ELSE number END, length(id), id COLLATE "C";

-- name: ListRoundsOfSubjects :many
SELECT id, subject_id, number, workflow_id, workflow_name, levels, current_level, outcome, submitted_at,
    submitted_by_id, submitted_by_name, comment, ended_at, facts, reminder_days, escalation_days, auto_approved,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name, subject_kind
FROM approval_rounds
WHERE subject_id = ANY(sqlc.arg(subject_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: ListOpenRounds :many
SELECT id, subject_id, number, workflow_id, workflow_name, levels, current_level, outcome, submitted_at,
    submitted_by_id, submitted_by_name, comment, ended_at, facts, reminder_days, escalation_days, auto_approved,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name, subject_kind
FROM approval_rounds
WHERE outcome = 'open'
ORDER BY length(id), id COLLATE "C";

-- name: ListRoundsInScope :many
SELECT id, subject_id, number, workflow_id, workflow_name, levels, current_level, outcome, submitted_at,
    submitted_by_id, submitted_by_name, comment, ended_at, facts, reminder_days, escalation_days, auto_approved,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name, subject_kind
FROM approval_rounds
WHERE (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
    AND subject_kind = 'activity'
    AND activity_in_scope(subject_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: PutRound :exec
INSERT INTO approval_rounds (
    id, subject_id, number, workflow_id, workflow_name, levels, current_level, outcome, submitted_at,
    submitted_by_id, submitted_by_name, comment, ended_at, facts, reminder_days, escalation_days, auto_approved,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name, subject_kind
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
)
ON CONFLICT (id) DO UPDATE SET
    subject_kind = excluded.subject_kind,
    subject_id = excluded.subject_id,
    number = excluded.number,
    workflow_id = excluded.workflow_id,
    workflow_name = excluded.workflow_name,
    levels = excluded.levels,
    current_level = excluded.current_level,
    outcome = excluded.outcome,
    submitted_at = excluded.submitted_at,
    submitted_by_id = excluded.submitted_by_id,
    submitted_by_name = excluded.submitted_by_name,
    comment = excluded.comment,
    ended_at = excluded.ended_at,
    facts = excluded.facts,
    reminder_days = excluded.reminder_days,
    escalation_days = excluded.escalation_days,
    auto_approved = excluded.auto_approved,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;
