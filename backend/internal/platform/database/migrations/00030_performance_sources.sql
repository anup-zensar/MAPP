-- +goose Up
-- Feature 046: where performance data comes from, and what each column of its file means.
-- Freshness (Current, Stale, Failed, Inactive) is derived from the latest import and the cadence, never stored.
CREATE TABLE performance_sources (
    id              text        PRIMARY KEY CHECK (id ~ '^PS-[0-9]+$'),
    name            text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    description     text        NOT NULL CHECK (char_length(description) <= 2000),
    owner_id        text        NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    category        text        NOT NULL
                                CHECK (category IN ('webAnalytics', 'marketingAutomation', 'crm', 'leadProgression')),
    status          text        NOT NULL CHECK (status IN ('active', 'inactive')),
    delivery_route  text        NOT NULL CHECK (delivery_route IN ('upload', 'simulated')),
    cadence         text        NOT NULL CHECK (cadence IN ('daily', 'weekly', 'monthly', 'onDemand')),
    integration_id  text        NOT NULL,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE UNIQUE INDEX performance_sources_name_idx ON performance_sources (lower(name COLLATE "C"));
CREATE INDEX performance_sources_status_idx ON performance_sources (status);
-- A source's own integration id is unique where it is set at all.
CREATE UNIQUE INDEX performance_sources_integration_id_idx
    ON performance_sources (integration_id) WHERE integration_id <> '';

CREATE TABLE performance_source_mappings (
    id              text        PRIMARY KEY CHECK (id ~ '^PSM-[0-9]+$'),
    source_id       text        NOT NULL REFERENCES performance_sources (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    column_name     text        NOT NULL CHECK (char_length(column_name) BETWEEN 1 AND 120),
    meaning         text        NOT NULL
                                CHECK (meaning IN ('key', 'reference', 'factDate', 'stage', 'revenue',
                                                   'currency', 'webMetric', 'journey')),
    stage           text        NOT NULL,
    web_metric      text        NOT NULL
                                CHECK (web_metric IN ('', 'sessions', 'visits', 'engagementTouches')),
    model_id        text        REFERENCES models (id) DEFERRABLE INITIALLY DEFERRED,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL,
    -- A stage column names its stage; a web metric names its metric; neither does both.
    CHECK ((meaning = 'stage') = (stage <> '')),
    CHECK ((meaning = 'webMetric') = (web_metric <> '')),
    -- Only a stage or revenue column is supplied by a model, and it must name one.
    CHECK ((meaning IN ('stage', 'revenue')) = (model_id IS NOT NULL))
);

CREATE UNIQUE INDEX performance_source_mappings_column_idx
    ON performance_source_mappings (source_id, lower(column_name COLLATE "C"));

-- FR-003 (one ACTIVE source per model stage, and per model revenue) spans two tables: the mapping's
-- model and stage, and the source's status. A unique index cannot express it, because an index predicate
-- may not contain a subquery, and a plain unique index would wrongly refuse a mapping on an inactive
-- source, which FR-003 allows. The rule is therefore enforced in the service, which names the conflicting
-- source in its refusal, and is proved by its own test rather than by a constraint.
CREATE INDEX performance_source_mappings_model_stage_idx
    ON performance_source_mappings (model_id, stage) WHERE meaning = 'stage';
CREATE INDEX performance_source_mappings_model_revenue_idx
    ON performance_source_mappings (model_id) WHERE meaning = 'revenue';
