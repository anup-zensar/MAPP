-- name: ListCampaigns :many
SELECT sqlc.embed(campaigns),
    coalesce((SELECT array_agg(m.month ORDER BY m.month) FROM campaign_phasing_months m
        WHERE m.campaign_id = campaigns.id), '{}')::text[] AS phasing_months,
    coalesce((SELECT array_agg(m.amount_minor ORDER BY m.month) FROM campaign_phasing_months m
        WHERE m.campaign_id = campaigns.id), '{}')::bigint[] AS phasing_amounts
FROM campaigns
WHERE (sqlc.narg(ids)::text[] IS NULL OR id = ANY (sqlc.narg(ids)::text[]))
    AND (sqlc.narg(plan_ids)::text[] IS NULL OR plan_id = ANY (sqlc.narg(plan_ids)::text[]))
ORDER BY length(id), id COLLATE "C";

-- name: ListCampaignsInScope :many
SELECT sqlc.embed(campaigns),
    coalesce((SELECT array_agg(m.month ORDER BY m.month) FROM campaign_phasing_months m
        WHERE m.campaign_id = campaigns.id), '{}')::text[] AS phasing_months,
    coalesce((SELECT array_agg(m.amount_minor ORDER BY m.month) FROM campaign_phasing_months m
        WHERE m.campaign_id = campaigns.id), '{}')::bigint[] AS phasing_amounts
FROM campaigns
WHERE (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
    AND plan_in_scope(plan_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: PutCampaign :exec
INSERT INTO campaigns (
    id, plan_id, name, description, owner_id, start_date, end_date, budget_minor, budget_currency, attributes,
    phasing_method, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name,
    overspend
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
)
ON CONFLICT (id) DO UPDATE SET
    plan_id = excluded.plan_id,
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
    updated_by_name = excluded.updated_by_name,
    overspend = excluded.overspend;

-- name: DeleteCampaignPhasingMonths :exec
DELETE FROM campaign_phasing_months
WHERE campaign_id = $1;

-- name: InsertCampaignPhasingMonths :exec
INSERT INTO campaign_phasing_months (campaign_id, month, amount_minor)
SELECT sqlc.arg(campaign_id), m.month, a.amount_minor
FROM unnest(sqlc.arg(months)::text[]) WITH ORDINALITY AS m(month, i)
JOIN unnest(sqlc.arg(amounts)::bigint[]) WITH ORDINALITY AS a(amount_minor, i) USING (i);

-- name: DeleteCampaign :exec
DELETE FROM campaigns
WHERE id = $1;

-- name: AddCampaigns :copyfrom
INSERT INTO campaigns (
    id, plan_id, name, description, owner_id, start_date, end_date, budget_minor, budget_currency, attributes,
    phasing_method, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name,
    overspend
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
);

-- name: AddCampaignPhasingMonths :copyfrom
INSERT INTO campaign_phasing_months (campaign_id, month, amount_minor)
VALUES ($1, $2, $3);

-- name: DeleteCampaigns :exec
DELETE FROM campaigns
WHERE id = ANY (sqlc.arg(ids)::text[]);
