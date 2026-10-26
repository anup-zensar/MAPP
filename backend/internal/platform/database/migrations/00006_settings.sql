-- +goose Up
CREATE TABLE currencies (
    code                 text        PRIMARY KEY CHECK (code ~ '^[A-Z]{3}$'),
    name                 text        NOT NULL CHECK (name <> ''),
    symbol               text        NOT NULL,
    rate_micros          bigint      CHECK (rate_micros > 0),
    rate_changed_at      timestamptz,
    rate_changed_by_id   text,
    rate_changed_by_name text,
    active               boolean     NOT NULL CHECK (active OR code <> 'USD'),
    version              integer     NOT NULL CHECK (version >= 1),
    created_at           timestamptz NOT NULL,
    created_by_id        text        NOT NULL,
    created_by_name      text        NOT NULL,
    updated_at           timestamptz NOT NULL,
    updated_by_id        text        NOT NULL,
    updated_by_name      text        NOT NULL,
    CHECK (num_nulls(rate_changed_by_id, rate_changed_by_name) IN (0, 2))
);

CREATE UNIQUE INDEX currencies_name_key ON currencies (lower(name COLLATE "C"));

CREATE TABLE rate_imports (
    id              text        PRIMARY KEY CHECK (id ~ '^RI-[0-9]+$'),
    file_name       text        NOT NULL CHECK (file_name <> ''),
    feed            text        NOT NULL CHECK (feed <> ''),
    first_date      date,
    last_date       date,
    added           integer     NOT NULL CHECK (added >= 0),
    replaced        integer     NOT NULL CHECK (replaced >= 0),
    unchanged       integer     NOT NULL CHECK (unchanged >= 0),
    not_used        integer     NOT NULL CHECK (not_used >= 0),
    problems        integer     NOT NULL CHECK (problems >= 0),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE TABLE exchange_rates (
    id              text        PRIMARY KEY CHECK (id ~ '^XR-[0-9]+$'),
    currency        text        NOT NULL REFERENCES currencies (code) DEFERRABLE INITIALLY DEFERRED
                                CHECK (currency <> 'USD'),
    date            date        NOT NULL,
    rate_micros     bigint      NOT NULL CHECK (rate_micros > 0 AND rate_micros <= 1000000000000),
    source          text        NOT NULL CHECK (source IN ('manual', 'gsapImport')),
    import_id       text        REFERENCES rate_imports (id) DEFERRABLE INITIALLY DEFERRED,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL,
    UNIQUE (currency, date)
);

CREATE TABLE field_help (
    key             text        PRIMARY KEY CHECK (key <> ''),
    text            text        NOT NULL,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE TABLE activity_types (
    id                            text        PRIMARY KEY CHECK (id ~ '^AT-[0-9]+$'),
    name                          text        NOT NULL CHECK (name <> ''),
    default_planned_cost_minor    bigint      CHECK (default_planned_cost_minor >= 0),
    default_planned_cost_currency text        CHECK (default_planned_cost_currency = 'USD'),
    active                        boolean     NOT NULL,
    version                       integer     NOT NULL CHECK (version >= 1),
    created_at                    timestamptz NOT NULL,
    created_by_id                 text        NOT NULL,
    created_by_name               text        NOT NULL,
    updated_at                    timestamptz NOT NULL,
    updated_by_id                 text        NOT NULL,
    updated_by_name               text        NOT NULL,
    CHECK (num_nulls(default_planned_cost_minor, default_planned_cost_currency) IN (0, 2))
);

CREATE UNIQUE INDEX activity_types_name_key ON activity_types (lower(name COLLATE "C"));

CREATE TABLE models (
    id                         text        PRIMARY KEY CHECK (id ~ '^MOD-[0-9]+$'),
    name                       text        NOT NULL CHECK (name <> ''),
    inq_to_mql_rate_hundredths bigint      NOT NULL CHECK (inq_to_mql_rate_hundredths BETWEEN 1 AND 10000),
    mql_to_deal_rate_hundredths bigint     NOT NULL CHECK (mql_to_deal_rate_hundredths BETWEEN 1 AND 10000),
    average_deal_size_minor    bigint      NOT NULL CHECK (average_deal_size_minor BETWEEN 1 AND 10000000000),
    average_deal_size_currency text        NOT NULL CHECK (average_deal_size_currency = 'USD'),
    active                     boolean     NOT NULL,
    version                    integer     NOT NULL CHECK (version >= 1),
    created_at                 timestamptz NOT NULL,
    created_by_id              text        NOT NULL,
    created_by_name            text        NOT NULL,
    updated_at                 timestamptz NOT NULL,
    updated_by_id              text        NOT NULL,
    updated_by_name            text        NOT NULL
);

CREATE UNIQUE INDEX models_name_key ON models (lower(name COLLATE "C"));

CREATE TABLE model_activity_types (
    model_id         text    NOT NULL REFERENCES models (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    position         integer NOT NULL CHECK (position >= 1),
    activity_type_id text    NOT NULL REFERENCES activity_types (id) DEFERRABLE INITIALLY DEFERRED,
    PRIMARY KEY (model_id, activity_type_id)
);

CREATE TABLE shell_attributes (
    id                  text        PRIMARY KEY CHECK (id ~ '^SA-[0-9]+$'),
    label               text        NOT NULL CHECK (label <> ''),
    help_text           text        NOT NULL,
    level               text        NOT NULL CHECK (level IN ('plan', 'campaign', 'initiative', 'activity', 'lineItem')),
    type                text        NOT NULL CHECK (type IN ('singleChoice', 'multipleChoice', 'multipleChoiceSplit',
                                        'yesNo', 'text', 'longText', 'number')),
    marking             text        NOT NULL CHECK (marking IN ('optional', 'neededToPublish', 'neededToSubmit')),
    section             text        NOT NULL CHECK (section <> ''),
    position            integer     NOT NULL CHECK (position >= 1),
    parent_attribute_id text        REFERENCES shell_attributes (id) DEFERRABLE INITIALLY DEFERRED,
    role                text        NOT NULL CHECK (role IN ('', 'businessUnit')),
    active              boolean     NOT NULL,
    version             integer     NOT NULL CHECK (version >= 1),
    created_at          timestamptz NOT NULL,
    created_by_id       text        NOT NULL,
    created_by_name     text        NOT NULL,
    updated_at          timestamptz NOT NULL,
    updated_by_id       text        NOT NULL,
    updated_by_name     text        NOT NULL,
    CHECK (type <> 'multipleChoiceSplit' OR level IN ('activity', 'lineItem'))
);

CREATE UNIQUE INDEX shell_attributes_label_key ON shell_attributes (level, lower(label COLLATE "C"));
CREATE UNIQUE INDEX shell_attributes_role_key ON shell_attributes (role) WHERE role <> '';

CREATE TABLE shell_attribute_options (
    id              text        PRIMARY KEY CHECK (id ~ '^SO-[0-9]+$'),
    attribute_id    text        NOT NULL REFERENCES shell_attributes (id) DEFERRABLE INITIALLY DEFERRED,
    label           text        NOT NULL CHECK (label <> ''),
    position        integer     NOT NULL CHECK (position >= 1),
    active          boolean     NOT NULL,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE UNIQUE INDEX shell_attribute_options_label_key ON shell_attribute_options (attribute_id, lower(label COLLATE "C"));

CREATE TABLE shell_attribute_conditions (
    attribute_id text    NOT NULL REFERENCES shell_attributes (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    position     integer NOT NULL CHECK (position >= 1),
    option_id    text    NOT NULL REFERENCES shell_attribute_options (id) DEFERRABLE INITIALLY DEFERRED,
    PRIMARY KEY (attribute_id, option_id)
);

CREATE TABLE shell_option_parents (
    option_id        text    NOT NULL REFERENCES shell_attribute_options (id) ON DELETE CASCADE
                             DEFERRABLE INITIALLY DEFERRED,
    position         integer NOT NULL CHECK (position >= 1),
    parent_option_id text    NOT NULL REFERENCES shell_attribute_options (id) DEFERRABLE INITIALLY DEFERRED,
    PRIMARY KEY (option_id, parent_option_id)
);

CREATE TABLE finance_budgets (
    id              text        PRIMARY KEY CHECK (id ~ '^FB-[0-9]+$'),
    year            integer     NOT NULL,
    name            text        NOT NULL CHECK (name <> ''),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE UNIQUE INDEX finance_budgets_name_key ON finance_budgets (year, lower(name COLLATE "C"));

CREATE TABLE finance_budget_items (
    id              text        PRIMARY KEY CHECK (id ~ '^FBI-[0-9]+$'),
    budget_id       text        NOT NULL REFERENCES finance_budgets (id) DEFERRABLE INITIALLY DEFERRED,
    name            text        NOT NULL CHECK (name <> ''),
    code            text        NOT NULL,
    active          boolean     NOT NULL,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE UNIQUE INDEX finance_budget_items_name_key ON finance_budget_items (budget_id, lower(name COLLATE "C"));
