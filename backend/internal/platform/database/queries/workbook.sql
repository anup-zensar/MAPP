-- name: AddPlanImport :exec
INSERT INTO plan_imports (id, plan_id, file_name, imported_at, imported_by_id, imported_by_name, rows_read, rows_left_out)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: AddPlanImportRecords :copyfrom
INSERT INTO plan_import_records (import_id, position, level, record_id, record_name, parent_id, action)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: AddPlanImportProblems :copyfrom
INSERT INTO plan_import_problems (import_id, position, sheet, row_number, column_heading, value, reason)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListPlanImportsInScope :many
SELECT * FROM plan_imports
WHERE (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
    AND (sqlc.narg(plan_id)::text IS NULL OR plan_id = sqlc.narg(plan_id))
    AND plan_in_scope(plan_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY imported_at DESC, length(id) DESC, id COLLATE "C" DESC
LIMIT sqlc.arg(max_imports);

-- name: ListPlanImportRecords :many
SELECT * FROM plan_import_records
WHERE import_id = ANY (sqlc.arg(import_ids)::text[])
ORDER BY import_id, position;

-- name: ListPlanImportProblems :many
SELECT * FROM plan_import_problems
WHERE import_id = sqlc.arg(import_id)
ORDER BY position;

-- name: MarkPlanImportUndone :exec
UPDATE plan_imports
SET undone_at = sqlc.arg(undone_at), undone_by_id = sqlc.arg(undone_by_id), undone_by_name = sqlc.arg(undone_by_name)
WHERE id = sqlc.arg(id);
