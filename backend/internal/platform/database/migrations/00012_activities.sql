-- +goose Up
CREATE TABLE activities (
    id                       text        PRIMARY KEY CHECK (id ~ '^A-[0-9]+$'),
    initiative_id            text        NOT NULL REFERENCES initiatives (id) DEFERRABLE INITIALLY DEFERRED,
    name                     text        NOT NULL CHECK (name <> ''),
    external_name            text        NOT NULL,
    activity_type_id         text        NOT NULL REFERENCES activity_types (id) DEFERRABLE INITIALLY DEFERRED,
    owner_id                 text        REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    start_date               date,
    end_date                 date,
    ongoing                  boolean     NOT NULL,
    planned_cost_minor       bigint      CHECK (planned_cost_minor >= 0),
    planned_cost_currency    text        REFERENCES currencies (code) DEFERRABLE INITIALLY DEFERRED,
    approval_workflow_id     text,
    description              text        NOT NULL,
    attributes               json        NOT NULL CHECK (json_typeof(attributes) = 'object'),
    inq                      bigint      CHECK (inq >= 0),
    set_later_stages_myself  boolean     NOT NULL,
    mql                      bigint      CHECK (mql >= 0),
    revenue_minor            bigint,
    revenue_currency         text        CHECK (revenue_currency = 'USD'),
    status                   text        NOT NULL CHECK (status IN ('created', 'submitted', 'declined', 'approved')),
    split_from_id            text,
    version                  integer     NOT NULL CHECK (version >= 1),
    created_at               timestamptz NOT NULL,
    created_by_id            text        NOT NULL,
    created_by_name          text        NOT NULL,
    updated_at               timestamptz NOT NULL,
    updated_by_id            text        NOT NULL,
    updated_by_name          text        NOT NULL,
    CHECK (num_nulls(planned_cost_minor, planned_cost_currency) IN (0, 2)),
    CHECK (num_nulls(revenue_minor, revenue_currency) IN (0, 2)),
    CHECK (set_later_stages_myself OR num_nulls(mql, revenue_minor) = 2)
);

CREATE UNIQUE INDEX activities_name_key ON activities (initiative_id, lower(name COLLATE "C"));

CREATE TABLE line_items (
    id                      text        PRIMARY KEY CHECK (id ~ '^L-[0-9]+$'),
    activity_id             text        NOT NULL REFERENCES activities (id) DEFERRABLE INITIALLY DEFERRED,
    name                    text        NOT NULL CHECK (name <> ''),
    owner_id                text        REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    description             text        NOT NULL,
    currency                text        NOT NULL REFERENCES currencies (code) DEFERRABLE INITIALLY DEFERRED,
    planned_cost_minor      bigint      NOT NULL CHECK (planned_cost_minor >= 0),
    spread_method           text        NOT NULL CHECK (spread_method IN ('evenly', 'firstMonth', 'lastMonth', 'custom')),
    attributes              json        NOT NULL CHECK (json_typeof(attributes) = 'object'),
    contracted_amount_minor bigint,
    request_swbs            boolean     NOT NULL,
    finance_budget_year     integer,
    finance_budget_item_id  text        REFERENCES finance_budget_items (id) DEFERRABLE INITIALLY DEFERRED,
    version                 integer     NOT NULL CHECK (version >= 1),
    created_at              timestamptz NOT NULL,
    created_by_id           text        NOT NULL,
    created_by_name         text        NOT NULL,
    updated_at              timestamptz NOT NULL,
    updated_by_id           text        NOT NULL,
    updated_by_name         text        NOT NULL
);

CREATE UNIQUE INDEX line_items_name_key ON line_items (activity_id, lower(name COLLATE "C"));

CREATE TABLE line_item_spread_months (
    line_item_id text   NOT NULL REFERENCES line_items (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    month        text   NOT NULL CHECK (month ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    PRIMARY KEY (line_item_id, month)
);

-- activity_in_scope is authz.CanViewActivity in SQL: the activity's plan is in scope, or the person owns the
-- activity. Migration 00013 adds the approvers named in its rounds, once their table exists.
-- +goose StatementBegin
CREATE FUNCTION activity_in_scope(
    target_activity text, sees_everything boolean, viewer_id text, views_plans boolean, viewer_team_ids text[]
) RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT sees_everything
        OR coalesce(a.owner_id = viewer_id, false)
        OR plan_in_scope(c.plan_id, sees_everything, viewer_id, views_plans, viewer_team_ids)
    FROM activities a
    JOIN initiatives i ON i.id = a.initiative_id
    JOIN campaigns c ON c.id = i.campaign_id
    WHERE a.id = target_activity
$$;
-- +goose StatementEnd
