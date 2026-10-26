-- +goose Up
CREATE TABLE campaigns (
    id              text        PRIMARY KEY CHECK (id ~ '^C-[0-9]+$'),
    plan_id         text        NOT NULL REFERENCES plans (id) DEFERRABLE INITIALLY DEFERRED,
    name            text        NOT NULL CHECK (name <> ''),
    description     text        NOT NULL,
    owner_id        text        REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    start_date      date,
    end_date        date,
    budget_minor    bigint      CHECK (budget_minor >= 0),
    budget_currency text        REFERENCES currencies (code) DEFERRABLE INITIALLY DEFERRED,
    attributes      json        NOT NULL CHECK (json_typeof(attributes) = 'object'),
    phasing_method  text        NOT NULL CHECK (phasing_method IN ('none', 'evenly', 'custom')),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL,
    CHECK (num_nulls(start_date, end_date) IN (0, 2) AND start_date <= end_date),
    CHECK (num_nulls(budget_minor, budget_currency) IN (0, 2))
);

CREATE UNIQUE INDEX campaigns_name_key ON campaigns (plan_id, lower(name COLLATE "C"));

CREATE TABLE campaign_phasing_months (
    campaign_id  text   NOT NULL REFERENCES campaigns (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    month        text   NOT NULL CHECK (month ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    amount_minor bigint NOT NULL,
    PRIMARY KEY (campaign_id, month)
);

CREATE TABLE initiatives (
    id              text        PRIMARY KEY CHECK (id ~ '^I-[0-9]+$'),
    campaign_id     text        NOT NULL REFERENCES campaigns (id) DEFERRABLE INITIALLY DEFERRED,
    name            text        NOT NULL CHECK (name <> ''),
    description     text        NOT NULL,
    owner_id        text        REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    start_date      date,
    end_date        date,
    budget_minor    bigint      CHECK (budget_minor >= 0),
    budget_currency text        REFERENCES currencies (code) DEFERRABLE INITIALLY DEFERRED,
    attributes      json        NOT NULL CHECK (json_typeof(attributes) = 'object'),
    phasing_method  text        NOT NULL CHECK (phasing_method IN ('none', 'evenly', 'custom')),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL,
    CHECK (num_nulls(start_date, end_date) IN (0, 2) AND start_date <= end_date),
    CHECK (num_nulls(budget_minor, budget_currency) IN (0, 2))
);

CREATE UNIQUE INDEX initiatives_name_key ON initiatives (campaign_id, lower(name COLLATE "C"));

CREATE TABLE initiative_phasing_months (
    initiative_id text   NOT NULL REFERENCES initiatives (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    month         text   NOT NULL CHECK (month ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    amount_minor  bigint NOT NULL,
    PRIMARY KEY (initiative_id, month)
);
