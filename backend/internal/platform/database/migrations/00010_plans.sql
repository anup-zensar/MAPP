-- +goose Up
CREATE TABLE plans (
    id                        text        PRIMARY KEY CHECK (id ~ '^P-[0-9]+$'),
    name                      text        NOT NULL CHECK (name <> ''),
    year                      integer     NOT NULL,
    model_id                  text        NOT NULL REFERENCES models (id) DEFERRABLE INITIALLY DEFERRED,
    owner_id                  text        REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    aop                       boolean     NOT NULL,
    description               text        NOT NULL,
    budget_minor              bigint,
    default_finance_budget_id text        REFERENCES finance_budgets (id) DEFERRABLE INITIALLY DEFERRED,
    default_spread            text        NOT NULL CHECK (default_spread IN ('evenly', 'firstMonth', 'lastMonth', 'custom')),
    bcpir                     boolean     NOT NULL,
    goal_stage                text        NOT NULL CHECK (goal_stage IN ('inq', 'mql', 'revenue')),
    goal_amount               text        NOT NULL,
    attributes                json        NOT NULL CHECK (json_typeof(attributes) = 'object'),
    status                    text        NOT NULL CHECK (status IN ('draft', 'published')),
    currency                  text        NOT NULL REFERENCES currencies (code) DEFERRABLE INITIALLY DEFERRED,
    plan_type                 text        NOT NULL CHECK (plan_type IN ('operational', 'strategic', 'crossMarket')),
    phasing_method            text        NOT NULL CHECK (phasing_method IN ('none', 'evenly', 'custom')),
    version                   integer     NOT NULL CHECK (version >= 1),
    created_at                timestamptz NOT NULL,
    created_by_id             text        NOT NULL,
    created_by_name           text        NOT NULL,
    updated_at                timestamptz NOT NULL,
    updated_by_id             text        NOT NULL,
    updated_by_name           text        NOT NULL
);

CREATE UNIQUE INDEX plans_name_key ON plans (lower(name COLLATE "C"));
CREATE INDEX plans_year_idx ON plans (year);

CREATE TABLE plan_teams (
    plan_id  text    NOT NULL REFERENCES plans (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    position integer NOT NULL CHECK (position >= 1),
    team_id  text    NOT NULL REFERENCES teams (id) DEFERRABLE INITIALLY DEFERRED,
    PRIMARY KEY (plan_id, team_id)
);

CREATE INDEX plan_teams_team_id_idx ON plan_teams (team_id, plan_id);

CREATE TABLE plan_phasing_months (
    plan_id      text   NOT NULL REFERENCES plans (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    month        text   NOT NULL CHECK (month ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    amount_minor bigint NOT NULL,
    PRIMARY KEY (plan_id, month)
);

-- plan_in_scope is authz.CanView in SQL, so that every query on plan-owned data filters by the person's
-- scope the same way (Principle VII): see everything, own the plan, or hold View plans for one of its teams
-- (or for a plan without teams that they created).
-- +goose StatementBegin
CREATE FUNCTION plan_in_scope(
    target_plan text, sees_everything boolean, viewer_id text, views_plans boolean, viewer_team_ids text[]
) RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT sees_everything
        OR coalesce(p.owner_id = viewer_id, false)
        OR (views_plans AND (
            EXISTS (SELECT 1 FROM plan_teams t WHERE t.plan_id = p.id AND t.team_id = ANY (viewer_team_ids))
            OR (NOT EXISTS (SELECT 1 FROM plan_teams t WHERE t.plan_id = p.id) AND p.created_by_id = viewer_id)))
    FROM plans p
    WHERE p.id = target_plan
$$;
-- +goose StatementEnd
