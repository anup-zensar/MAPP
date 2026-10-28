-- name: ListLineItems :many
SELECT sqlc.embed(line_items),
    coalesce(spread.months, '{}')::text[] AS spread_months,
    coalesce(spread.amounts, '{}')::bigint[] AS spread_amounts
FROM line_items
LEFT JOIN (
    SELECT m.line_item_id, array_agg(m.month ORDER BY m.month) AS months,
        array_agg(m.amount_minor ORDER BY m.month) AS amounts
    FROM line_item_spread_months m
    GROUP BY m.line_item_id
) spread ON spread.line_item_id = line_items.id
WHERE (sqlc.narg(ids)::text[] IS NULL OR id = ANY (sqlc.narg(ids)::text[]))
    AND (sqlc.narg(activity_ids)::text[] IS NULL OR activity_id = ANY (sqlc.narg(activity_ids)::text[]))
ORDER BY length(id), id COLLATE "C";

-- name: ListLineItemsInScope :many
SELECT sqlc.embed(line_items),
    coalesce((SELECT array_agg(m.month ORDER BY m.month) FROM line_item_spread_months m
        WHERE m.line_item_id = line_items.id), '{}')::text[] AS spread_months,
    coalesce((SELECT array_agg(m.amount_minor ORDER BY m.month) FROM line_item_spread_months m
        WHERE m.line_item_id = line_items.id), '{}')::bigint[] AS spread_amounts
FROM line_items
WHERE (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
    AND activity_in_scope(activity_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: PutLineItem :exec
INSERT INTO line_items (
    id, activity_id, name, owner_id, description, currency, planned_cost_minor, spread_method, attributes,
    contracted_amount_minor, request_swbs, finance_budget_year, finance_budget_item_id, version, created_at,
    created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name, swbs_code, swbs_requested_at,
    swbs_received_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
)
ON CONFLICT (id) DO UPDATE SET
    activity_id = excluded.activity_id,
    name = excluded.name,
    owner_id = excluded.owner_id,
    description = excluded.description,
    currency = excluded.currency,
    planned_cost_minor = excluded.planned_cost_minor,
    spread_method = excluded.spread_method,
    attributes = excluded.attributes,
    contracted_amount_minor = excluded.contracted_amount_minor,
    request_swbs = excluded.request_swbs,
    finance_budget_year = excluded.finance_budget_year,
    finance_budget_item_id = excluded.finance_budget_item_id,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name,
    swbs_code = excluded.swbs_code,
    swbs_requested_at = excluded.swbs_requested_at,
    swbs_received_at = excluded.swbs_received_at;

-- name: DeleteLineItemSpreadMonths :exec
DELETE FROM line_item_spread_months
WHERE line_item_id = $1;

-- name: InsertLineItemSpreadMonths :exec
INSERT INTO line_item_spread_months (line_item_id, month, amount_minor)
SELECT sqlc.arg(line_item_id), m.month, a.amount_minor
FROM unnest(sqlc.arg(months)::text[]) WITH ORDINALITY AS m(month, i)
JOIN unnest(sqlc.arg(amounts)::bigint[]) WITH ORDINALITY AS a(amount_minor, i) USING (i);

-- name: DeleteLineItem :exec
DELETE FROM line_items
WHERE id = $1;

-- name: AddLineItems :copyfrom
INSERT INTO line_items (
    id, activity_id, name, owner_id, description, currency, planned_cost_minor, spread_method, attributes,
    contracted_amount_minor, request_swbs, finance_budget_year, finance_budget_item_id, version, created_at,
    created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name, swbs_code, swbs_requested_at,
    swbs_received_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
);

-- name: AddLineItemSpreadMonths :copyfrom
INSERT INTO line_item_spread_months (line_item_id, month, amount_minor)
VALUES ($1, $2, $3);

-- name: DeleteLineItems :exec
DELETE FROM line_items
WHERE id = ANY (sqlc.arg(ids)::text[]);
