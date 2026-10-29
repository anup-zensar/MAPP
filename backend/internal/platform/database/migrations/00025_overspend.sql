-- +goose Up
-- Feature 037: each budget warns or refuses when planned cost goes over it; stored budgets keep warning (D-346, D-347).
ALTER TABLE plans
    ADD COLUMN overspend text NOT NULL DEFAULT 'warn' CHECK (overspend IN ('warn', 'refuse'));

ALTER TABLE campaigns
    ADD COLUMN overspend text NOT NULL DEFAULT 'warn' CHECK (overspend IN ('warn', 'refuse'));

ALTER TABLE initiatives
    ADD COLUMN overspend text NOT NULL DEFAULT 'warn' CHECK (overspend IN ('warn', 'refuse'));

ALTER TABLE global_budgets
    ADD COLUMN overspend text NOT NULL DEFAULT 'warn' CHECK (overspend IN ('warn', 'refuse'));

CREATE TABLE overspend_configuration (
    id              text        PRIMARY KEY CHECK (id = 'overspend'),
    default_mode    text        NOT NULL CHECK (default_mode IN ('warn', 'refuse')),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE INDEX activities_initiative_id_idx ON activities (initiative_id);
CREATE INDEX initiatives_campaign_id_idx ON initiatives (campaign_id);
