-- name: ListSwbsRuns :many
SELECT sqlc.embed(swbs_runs),
    coalesce((SELECT array_agg(c.line_item_id ORDER BY c.position) FROM swbs_run_codes c
        WHERE c.run_id = swbs_runs.id), '{}')::text[] AS line_item_ids,
    coalesce((SELECT array_agg(c.line_item_name ORDER BY c.position) FROM swbs_run_codes c
        WHERE c.run_id = swbs_runs.id), '{}')::text[] AS line_item_names,
    coalesce((SELECT array_agg(c.company_code ORDER BY c.position) FROM swbs_run_codes c
        WHERE c.run_id = swbs_runs.id), '{}')::text[] AS company_codes,
    coalesce((SELECT array_agg(c.swbs_code ORDER BY c.position) FROM swbs_run_codes c
        WHERE c.run_id = swbs_runs.id), '{}')::text[] AS swbs_codes
FROM swbs_runs
WHERE sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id)
ORDER BY length(id) DESC, id COLLATE "C" DESC
LIMIT sqlc.arg(max_runs);

-- name: CountSwbsRuns :one
SELECT count(*)::integer FROM swbs_runs;

-- name: CountSwbsCodes :one
SELECT count(*)::integer FROM swbs_run_codes
WHERE starts_with(swbs_code, sqlc.arg(prefix)::text);

-- name: AddSwbsRun :exec
INSERT INTO swbs_runs (
    id, source, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- name: AddSwbsRunCodes :copyfrom
INSERT INTO swbs_run_codes (run_id, position, line_item_id, line_item_name, company_code, swbs_code)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteSwbsRun :exec
DELETE FROM swbs_runs
WHERE id = $1;
