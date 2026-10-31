-- +goose Up
-- Feature 046: each load of performance data, the facts it produced, and the rows a person must fix.
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
    -- Only a completed load accounts for every row; one that fell over records how far it got.
    CHECK (status <> 'completed'
           OR rows_read = rows_matched + rows_held + rows_unattributed + rows_rejected + rows_unmatched)
);

CREATE INDEX performance_imports_source_idx ON performance_imports (source_id, created_at DESC);
CREATE INDEX performance_imports_status_idx ON performance_imports (status);

-- The five counts per measure; a row carrying two measures counts in both, so groups do not add up.
CREATE TABLE performance_import_breakdown (
    import_id         text    NOT NULL REFERENCES performance_imports (id) DEFERRABLE INITIALLY DEFERRED,
    stage             text    NOT NULL CHECK (stage IN ('', 'inq', 'mql', 'revenue')),
    web_metric        text    NOT NULL CHECK (web_metric IN ('', 'sessions', 'visits', 'engagementTouches')),
    rows_read         integer NOT NULL CHECK (rows_read >= 0),
    rows_matched      integer NOT NULL CHECK (rows_matched >= 0),
    rows_held         integer NOT NULL CHECK (rows_held >= 0),
    rows_unattributed integer NOT NULL CHECK (rows_unattributed >= 0),
    rows_rejected     integer NOT NULL CHECK (rows_rejected >= 0),
    rows_unmatched    integer NOT NULL CHECK (rows_unmatched >= 0),
    PRIMARY KEY (import_id, stage, web_metric),
    CHECK ((stage <> '') <> (web_metric <> '')),
    CHECK (rows_read = rows_matched + rows_held + rows_unattributed + rows_rejected + rows_unmatched)
);

CREATE TABLE performance_facts (
    id               text        PRIMARY KEY CHECK (id ~ '^PF-[0-9]+$'),
    source_id        text        NOT NULL REFERENCES performance_sources (id) DEFERRABLE INITIALLY DEFERRED,
    import_id        text        NOT NULL REFERENCES performance_imports (id) DEFERRABLE INITIALLY DEFERRED,
    source_reference text        NOT NULL CHECK (char_length(source_reference) BETWEEN 1 AND 200),
    activity_id      text        REFERENCES activities (id) DEFERRABLE INITIALLY DEFERRED,
    matched_on       text        NOT NULL
                                 CHECK (matched_on IN ('', 'activityId', 'mediaCode', 'integrationId', 'externalName')),
    fact_date        date        NOT NULL,
    stage            text        NOT NULL CHECK (stage IN ('', 'inq', 'mql', 'revenue')),
    web_metric       text        NOT NULL
                                 CHECK (web_metric IN ('', 'sessions', 'visits', 'engagementTouches')),
    count_value      bigint      CHECK (count_value >= 0),
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
    CHECK ((stage <> '') <> (web_metric <> '')),
    -- Money is the revenue stage's alone; every other measure is a count, and exactly one of the two is set.
    CHECK ((amount_minor IS NOT NULL) = (stage = 'revenue')),
    CHECK (num_nulls(count_value, amount_minor) = 1),
    CHECK ((amount_minor IS NULL) = (currency IS NULL)),
    CHECK ((activity_id IS NULL) = (matched_on = '')),
    CHECK (NOT held OR activity_id IS NOT NULL)
);

-- Research R3: the measure joins the identity because one file row holding an INQ column and an MQL
-- column produces one fact for each, under the same source reference.
CREATE UNIQUE INDEX performance_facts_identity_idx
    ON performance_facts (source_id, source_reference, stage, web_metric) WHERE replaced_by_id IS NULL;
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
    CHECK ((status = 'dismissed') = (comment <> ''))
);

CREATE INDEX performance_review_rows_status_idx ON performance_review_rows (status, source_id);
CREATE INDEX performance_review_rows_import_idx ON performance_review_rows (import_id);

-- +goose Down
DROP TABLE performance_review_rows;
DROP TABLE performance_facts;
DROP TABLE performance_import_breakdown;
DROP TABLE performance_imports;
