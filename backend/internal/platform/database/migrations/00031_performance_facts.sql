-- +goose Up
-- Feature 046: each load of performance data, the facts it produced, and the rows a person must fix.
-- A fact matched to no activity counts nowhere, so every such row is kept for review.
CREATE TABLE performance_imports (
    id              text        PRIMARY KEY CHECK (id ~ '^PIM-[0-9]+$'),
    source_id       text        NOT NULL REFERENCES performance_sources (id) DEFERRABLE INITIALLY DEFERRED,
    file_name       text        NOT NULL CHECK (char_length(file_name) <= 260),
    origin          text        NOT NULL CHECK (origin IN ('upload', 'simulated')),
    status          text        NOT NULL CHECK (status IN ('completed', 'failed', 'undone')),
    data_through    date,
    rows_read       integer     NOT NULL CHECK (rows_read >= 0),
    rows_matched    integer     NOT NULL CHECK (rows_matched >= 0),
    rows_held       integer     NOT NULL CHECK (rows_held >= 0),
    rows_unattributed integer   NOT NULL CHECK (rows_unattributed >= 0),
    rows_rejected   integer     NOT NULL CHECK (rows_rejected >= 0),
    rows_unmatched  integer     NOT NULL CHECK (rows_unmatched >= 0),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL,
    -- FR-022: the five outcomes always account for every row read.
    CHECK (rows_read = rows_matched + rows_held + rows_unattributed + rows_rejected + rows_unmatched)
);

CREATE INDEX performance_imports_source_idx ON performance_imports (source_id, created_at DESC);
CREATE INDEX performance_imports_status_idx ON performance_imports (status);

CREATE TABLE performance_facts (
    id               text        PRIMARY KEY CHECK (id ~ '^PF-[0-9]+$'),
    source_id        text        NOT NULL REFERENCES performance_sources (id) DEFERRABLE INITIALLY DEFERRED,
    import_id        text        NOT NULL REFERENCES performance_imports (id) DEFERRABLE INITIALLY DEFERRED,
    source_reference text        NOT NULL CHECK (char_length(source_reference) BETWEEN 1 AND 200),
    activity_id      text        REFERENCES activities (id) DEFERRABLE INITIALLY DEFERRED,
    matched_on       text        NOT NULL
                                 CHECK (matched_on IN ('', 'activityId', 'mediaCode', 'integrationId', 'externalName')),
    fact_date        date        NOT NULL,
    stage            text        NOT NULL,
    web_metric       text        NOT NULL
                                 CHECK (web_metric IN ('', 'sessions', 'visits', 'engagementTouches')),
    count_value      integer     CHECK (count_value >= 0),
    amount_minor     bigint,
    currency         text        REFERENCES currencies (code) DEFERRABLE INITIALLY DEFERRED,
    held             boolean     NOT NULL,
    replaced_by_id   text        REFERENCES performance_facts (id) DEFERRABLE INITIALLY DEFERRED,
    version          integer     NOT NULL CHECK (version >= 1),
    created_at       timestamptz NOT NULL,
    created_by_id    text        NOT NULL,
    created_by_name  text        NOT NULL,
    updated_at       timestamptz NOT NULL,
    updated_by_id    text        NOT NULL,
    updated_by_name  text        NOT NULL,
    -- A fact measures exactly one thing: a funnel stage or a web metric, never both and never neither.
    CHECK ((stage <> '') <> (web_metric <> '')),
    -- An amount and its currency travel together; money is exact minor units, never a float.
    CHECK ((amount_minor IS NULL) = (currency IS NULL)),
    -- A matched fact names the key it matched on; an unattributed one names none.
    CHECK ((activity_id IS NULL) = (matched_on = '')),
    -- Only a matched fact can be held: an unattributed row is excluded for a different reason.
    CHECK (NOT held OR activity_id IS NOT NULL)
);

-- Research R3: (source, source reference) identifies a fact, among rows not yet replaced.
CREATE UNIQUE INDEX performance_facts_identity_idx
    ON performance_facts (source_id, source_reference) WHERE replaced_by_id IS NULL;
-- Every figure filters on held, so it is indexed with the activity it belongs to.
CREATE INDEX performance_facts_activity_idx ON performance_facts (activity_id, held, fact_date);
CREATE INDEX performance_facts_import_idx ON performance_facts (import_id);

CREATE TABLE performance_review_rows (
    id              text        PRIMARY KEY CHECK (id ~ '^PRV-[0-9]+$'),
    import_id       text        NOT NULL REFERENCES performance_imports (id) DEFERRABLE INITIALLY DEFERRED,
    source_id       text        NOT NULL REFERENCES performance_sources (id) DEFERRABLE INITIALLY DEFERRED,
    raw             json        NOT NULL CHECK (json_typeof(raw) = 'object'),
    reason          text        NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 200),
    faults          json        NOT NULL CHECK (json_typeof(faults) = 'array'),
    candidates      json        CHECK (candidates IS NULL OR json_typeof(candidates) = 'array'),
    status          text        NOT NULL CHECK (status IN ('open', 'resolved', 'dismissed')),
    comment         text        NOT NULL,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL,
    -- FR-024: a dismissal always carries its reason.
    CHECK ((status = 'dismissed') = (comment <> ''))
);

CREATE INDEX performance_review_rows_status_idx ON performance_review_rows (status, source_id);
CREATE INDEX performance_review_rows_import_idx ON performance_review_rows (import_id);
