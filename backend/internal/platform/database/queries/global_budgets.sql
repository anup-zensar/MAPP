-- name: GetGlobalBudget :one
SELECT id, name, year, amount_minor, owner_id, description, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name, overspend
FROM global_budgets
WHERE id = $1;

-- name: ListGlobalBudgets :many
SELECT id, name, year, amount_minor, owner_id, description, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name, overspend
FROM global_budgets
ORDER BY length(id), id COLLATE "C";

-- name: PutGlobalBudget :exec
INSERT INTO global_budgets (
    id, name, year, amount_minor, owner_id, description, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name, overspend
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    year = excluded.year,
    amount_minor = excluded.amount_minor,
    owner_id = excluded.owner_id,
    description = excluded.description,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name,
    overspend = excluded.overspend;

-- name: DeleteGlobalBudget :exec
DELETE FROM global_budgets
WHERE id = $1;

-- name: ListAllocations :many
SELECT id, global_budget_id, plan_id, amount_minor, amount_currency, version, created_at, created_by_id,
    created_by_name, updated_at, updated_by_id, updated_by_name
FROM budget_allocations
WHERE (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
    AND (sqlc.narg(global_budget_id)::text IS NULL OR global_budget_id = sqlc.narg(global_budget_id))
    AND (sqlc.narg(plan_id)::text IS NULL OR plan_id = sqlc.narg(plan_id))
ORDER BY length(id), id COLLATE "C";

-- name: ListAllocationsOfBudgets :many
SELECT id, global_budget_id, plan_id, amount_minor, amount_currency, version, created_at, created_by_id,
    created_by_name, updated_at, updated_by_id, updated_by_name
FROM budget_allocations
WHERE global_budget_id = ANY(sqlc.arg(global_budget_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: ListAllocationsInScope :many
SELECT id, global_budget_id, plan_id, amount_minor, amount_currency, version, created_at, created_by_id,
    created_by_name, updated_at, updated_by_id, updated_by_name
FROM budget_allocations
WHERE (sqlc.narg(global_budget_id)::text IS NULL OR global_budget_id = sqlc.narg(global_budget_id))
    AND plan_in_scope(plan_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: PutAllocation :exec
INSERT INTO budget_allocations (
    id, global_budget_id, plan_id, amount_minor, amount_currency, version, created_at, created_by_id,
    created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (id) DO UPDATE SET
    global_budget_id = excluded.global_budget_id,
    plan_id = excluded.plan_id,
    amount_minor = excluded.amount_minor,
    amount_currency = excluded.amount_currency,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteAllocation :exec
DELETE FROM budget_allocations
WHERE id = $1;
