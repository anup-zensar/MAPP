-- name: GetFinanceBudget :one
SELECT id, year, name, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id,
    updated_by_name
FROM finance_budgets
WHERE id = $1;

-- name: ListFinanceBudgets :many
SELECT id, year, name, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id,
    updated_by_name
FROM finance_budgets
ORDER BY length(id), id COLLATE "C";

-- name: PutFinanceBudget :exec
INSERT INTO finance_budgets (
    id, year, name, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id,
    updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
ON CONFLICT (id) DO UPDATE SET
    year = excluded.year,
    name = excluded.name,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteFinanceBudget :exec
DELETE FROM finance_budgets
WHERE id = $1;

-- name: GetFinanceBudgetItem :one
SELECT id, budget_id, name, code, active, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
FROM finance_budget_items
WHERE id = $1;

-- name: ListFinanceBudgetItems :many
SELECT id, budget_id, name, code, active, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
FROM finance_budget_items
WHERE sqlc.narg(budget_id)::text IS NULL OR budget_id = sqlc.narg(budget_id)
ORDER BY length(id), id COLLATE "C";

-- name: PutFinanceBudgetItem :exec
INSERT INTO finance_budget_items (
    id, budget_id, name, code, active, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (id) DO UPDATE SET
    budget_id = excluded.budget_id,
    name = excluded.name,
    code = excluded.code,
    active = excluded.active,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteFinanceBudgetItem :exec
DELETE FROM finance_budget_items
WHERE id = $1;

-- name: GetFieldHelp :one
SELECT key, text, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM field_help
WHERE key = $1;

-- name: ListFieldHelp :many
SELECT key, text, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM field_help
ORDER BY length(key), key COLLATE "C";

-- name: PutFieldHelp :exec
INSERT INTO field_help (
    key, text, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (key) DO UPDATE SET
    text = excluded.text,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteFieldHelp :exec
DELETE FROM field_help
WHERE key = $1;
