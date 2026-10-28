-- name: ListOverspendBudgets :many
WITH costs AS (
    SELECT a.initiative_id, i.campaign_id, c.plan_id, coalesce(a.planned_cost_minor, 0) AS minor
    FROM activities a
    JOIN initiatives i ON i.id = a.initiative_id
    JOIN campaigns c ON c.id = i.campaign_id
)
SELECT 'plan'::text AS kind, p.id, p.name, p.overspend, p.budget_minor::bigint AS amount_minor, p.currency,
    coalesce((SELECT sum(x.minor) FROM costs x WHERE x.plan_id = p.id), 0)::bigint AS planned_minor
FROM plans p
WHERE p.budget_minor IS NOT NULL
UNION ALL
SELECT 'campaign'::text, c.id, c.name, c.overspend, c.budget_minor::bigint, p.currency,
    coalesce((SELECT sum(x.minor) FROM costs x WHERE x.campaign_id = c.id), 0)::bigint
FROM campaigns c
JOIN plans p ON p.id = c.plan_id
WHERE c.budget_minor IS NOT NULL
UNION ALL
SELECT 'initiative'::text, i.id, i.name, i.overspend, i.budget_minor::bigint, p.currency,
    coalesce((SELECT sum(x.minor) FROM costs x WHERE x.initiative_id = i.id), 0)::bigint
FROM initiatives i
JOIN campaigns c ON c.id = i.campaign_id
JOIN plans p ON p.id = c.plan_id
WHERE i.budget_minor IS NOT NULL;

-- name: ListOverspendGlobalBudgets :many
SELECT g.id, g.name, g.overspend, g.amount_minor, p.id AS plan_id, p.currency AS plan_currency,
    coalesce((SELECT sum(coalesce(a.planned_cost_minor, 0)) FROM activities a
        JOIN initiatives i ON i.id = a.initiative_id
        JOIN campaigns c ON c.id = i.campaign_id
        WHERE c.plan_id = p.id), 0)::bigint AS planned_minor
FROM global_budgets g
LEFT JOIN budget_allocations ba ON ba.global_budget_id = g.id
LEFT JOIN plans p ON p.id = ba.plan_id
ORDER BY length(g.id), g.id COLLATE "C", p.id COLLATE "C";

-- name: GetOverspendConfiguration :many
SELECT * FROM overspend_configuration WHERE id = 'overspend';

-- name: PutOverspendConfiguration :exec
INSERT INTO overspend_configuration (
    id, default_mode, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    'overspend', $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (id) DO UPDATE SET
    default_mode = excluded.default_mode,
    version = excluded.version,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;
