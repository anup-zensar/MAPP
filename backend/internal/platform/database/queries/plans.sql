-- name: ListPlans :many
SELECT sqlc.embed(plans),
    coalesce((SELECT array_agg(t.team_id ORDER BY t.position) FROM plan_teams t WHERE t.plan_id = plans.id),
        '{}')::text[] AS team_ids,
    coalesce((SELECT array_agg(m.month ORDER BY m.month) FROM plan_phasing_months m WHERE m.plan_id = plans.id),
        '{}')::text[] AS phasing_months,
    coalesce((SELECT array_agg(m.amount_minor ORDER BY m.month) FROM plan_phasing_months m
        WHERE m.plan_id = plans.id), '{}')::bigint[] AS phasing_amounts
FROM plans
WHERE (sqlc.narg(ids)::text[] IS NULL OR id = ANY (sqlc.narg(ids)::text[]))
    AND (sqlc.narg(year)::integer IS NULL OR year = sqlc.narg(year))
ORDER BY length(id), id COLLATE "C";

-- name: ListPlansInScope :many
SELECT sqlc.embed(plans),
    coalesce((SELECT array_agg(t.team_id ORDER BY t.position) FROM plan_teams t WHERE t.plan_id = plans.id),
        '{}')::text[] AS team_ids,
    coalesce((SELECT array_agg(m.month ORDER BY m.month) FROM plan_phasing_months m WHERE m.plan_id = plans.id),
        '{}')::text[] AS phasing_months,
    coalesce((SELECT array_agg(m.amount_minor ORDER BY m.month) FROM plan_phasing_months m
        WHERE m.plan_id = plans.id), '{}')::bigint[] AS phasing_amounts
FROM plans
WHERE (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
    AND plan_in_scope(id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: PutPlan :exec
INSERT INTO plans (
    id, name, year, model_id, owner_id, aop, description, budget_minor, default_finance_budget_id, default_spread,
    bcpir, goal_stage, goal_amount, attributes, status, currency, plan_type, phasing_method, version, created_at,
    created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name, overspend
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24,
    $25, $26
)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    year = excluded.year,
    model_id = excluded.model_id,
    owner_id = excluded.owner_id,
    aop = excluded.aop,
    description = excluded.description,
    budget_minor = excluded.budget_minor,
    default_finance_budget_id = excluded.default_finance_budget_id,
    default_spread = excluded.default_spread,
    bcpir = excluded.bcpir,
    goal_stage = excluded.goal_stage,
    goal_amount = excluded.goal_amount,
    attributes = excluded.attributes,
    status = excluded.status,
    currency = excluded.currency,
    plan_type = excluded.plan_type,
    phasing_method = excluded.phasing_method,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name,
    overspend = excluded.overspend;

-- name: DeletePlanTeams :exec
DELETE FROM plan_teams
WHERE plan_id = $1;

-- name: InsertPlanTeams :exec
INSERT INTO plan_teams (plan_id, position, team_id)
SELECT sqlc.arg(plan_id), ordinality::integer, team_id
FROM unnest(sqlc.arg(team_ids)::text[]) WITH ORDINALITY AS t(team_id, ordinality);

-- name: DeletePlanPhasingMonths :exec
DELETE FROM plan_phasing_months
WHERE plan_id = $1;

-- name: InsertPlanPhasingMonths :exec
INSERT INTO plan_phasing_months (plan_id, month, amount_minor)
SELECT sqlc.arg(plan_id), m.month, a.amount_minor
FROM unnest(sqlc.arg(months)::text[]) WITH ORDINALITY AS m(month, i)
JOIN unnest(sqlc.arg(amounts)::bigint[]) WITH ORDINALITY AS a(amount_minor, i) USING (i);

-- name: DeletePlan :exec
DELETE FROM plans
WHERE id = $1;
