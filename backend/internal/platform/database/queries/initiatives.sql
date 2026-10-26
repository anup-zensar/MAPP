-- name: ListInitiatives :many
SELECT sqlc.embed(initiatives),
    coalesce((SELECT array_agg(m.month ORDER BY m.month) FROM initiative_phasing_months m
        WHERE m.initiative_id = initiatives.id), '{}')::text[] AS phasing_months,
    coalesce((SELECT array_agg(m.amount_minor ORDER BY m.month) FROM initiative_phasing_months m
        WHERE m.initiative_id = initiatives.id), '{}')::bigint[] AS phasing_amounts
FROM initiatives
WHERE (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
    AND (sqlc.narg(campaign_ids)::text[] IS NULL OR campaign_id = ANY (sqlc.narg(campaign_ids)::text[]))
ORDER BY length(id), id COLLATE "C";

-- name: ListInitiativesInScope :many
SELECT sqlc.embed(initiatives),
    coalesce((SELECT array_agg(m.month ORDER BY m.month) FROM initiative_phasing_months m
        WHERE m.initiative_id = initiatives.id), '{}')::text[] AS phasing_months,
    coalesce((SELECT array_agg(m.amount_minor ORDER BY m.month) FROM initiative_phasing_months m
        WHERE m.initiative_id = initiatives.id), '{}')::bigint[] AS phasing_amounts
FROM initiatives
JOIN campaigns c ON c.id = initiatives.campaign_id
WHERE (sqlc.narg(id)::text IS NULL OR initiatives.id = sqlc.narg(id))
    AND plan_in_scope(c.plan_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY length(initiatives.id), initiatives.id COLLATE "C";

-- name: PutInitiative :exec
INSERT INTO initiatives (
    id, campaign_id, name, description, owner_id, start_date, end_date, budget_minor, budget_currency, attributes,
    phasing_method, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
)
ON CONFLICT (id) DO UPDATE SET
    campaign_id = excluded.campaign_id,
    name = excluded.name,
    description = excluded.description,
    owner_id = excluded.owner_id,
    start_date = excluded.start_date,
    end_date = excluded.end_date,
    budget_minor = excluded.budget_minor,
    budget_currency = excluded.budget_currency,
    attributes = excluded.attributes,
    phasing_method = excluded.phasing_method,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteInitiativePhasingMonths :exec
DELETE FROM initiative_phasing_months
WHERE initiative_id = $1;

-- name: InsertInitiativePhasingMonths :exec
INSERT INTO initiative_phasing_months (initiative_id, month, amount_minor)
SELECT sqlc.arg(initiative_id), m.month, a.amount_minor
FROM unnest(sqlc.arg(months)::text[]) WITH ORDINALITY AS m(month, i)
JOIN unnest(sqlc.arg(amounts)::bigint[]) WITH ORDINALITY AS a(amount_minor, i) USING (i);

-- name: DeleteInitiative :exec
DELETE FROM initiatives
WHERE id = $1;
