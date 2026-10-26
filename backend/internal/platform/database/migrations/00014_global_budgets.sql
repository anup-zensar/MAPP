-- +goose Up
CREATE TABLE global_budgets (
    id              text        PRIMARY KEY CHECK (id ~ '^GB-[0-9]+$'),
    name            text        NOT NULL CHECK (name <> ''),
    year            integer     NOT NULL,
    amount_minor    bigint      NOT NULL CHECK (amount_minor >= 0),
    owner_id        text        NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    description     text        NOT NULL,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE UNIQUE INDEX global_budgets_name_key ON global_budgets (lower(name COLLATE "C"));

CREATE TABLE budget_allocations (
    id               text        PRIMARY KEY CHECK (id ~ '^BA-[0-9]+$'),
    global_budget_id text        NOT NULL,
    plan_id          text        NOT NULL REFERENCES plans (id) DEFERRABLE INITIALLY DEFERRED,
    amount_minor     bigint      NOT NULL CHECK (amount_minor >= 0),
    amount_currency  text        NOT NULL REFERENCES currencies (code) DEFERRABLE INITIALLY DEFERRED,
    version          integer     NOT NULL CHECK (version >= 1),
    created_at       timestamptz NOT NULL,
    created_by_id    text        NOT NULL,
    created_by_name  text        NOT NULL,
    updated_at       timestamptz NOT NULL,
    updated_by_id    text        NOT NULL,
    updated_by_name  text        NOT NULL,
    UNIQUE (plan_id) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX budget_allocations_global_budget_id_idx ON budget_allocations (global_budget_id);
