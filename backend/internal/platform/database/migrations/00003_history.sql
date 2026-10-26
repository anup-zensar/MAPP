-- +goose Up
CREATE TABLE activity_log (
    id                text        PRIMARY KEY CHECK (id ~ '^LOG-[0-9]+$'),
    seq               integer     NOT NULL UNIQUE,
    at                timestamptz NOT NULL,
    actor_id          text        NOT NULL,
    actor_name        text        NOT NULL,
    action            text        NOT NULL CHECK (action IN ('created', 'changed', 'statusChanged', 'moved',
                                      'copied', 'deleted', 'restored', 'deletedPermanently', 'emptiedRecycleBin')),
    verb              text        NOT NULL,
    record_type       text        NOT NULL,
    record_type_label text        NOT NULL,
    record_id         text        NOT NULL,
    record_name       text        NOT NULL,
    path              json        NOT NULL CHECK (json_typeof(path) IN ('array', 'null')),
    changes           json        NOT NULL CHECK (json_typeof(changes) IN ('array', 'null')),
    comment           text        NOT NULL,
    details           text        NOT NULL,
    items             json        NOT NULL CHECK (json_typeof(items) IN ('array', 'null'))
);

CREATE TABLE recycle_bin (
    id                 text        PRIMARY KEY CHECK (id ~ '^BIN-[0-9]+$'),
    record_type        text        NOT NULL,
    record_type_label  text        NOT NULL,
    record_id          text        NOT NULL,
    record_name        text        NOT NULL,
    path               json        NOT NULL CHECK (json_typeof(path) IN ('array', 'null')),
    parent             json        NOT NULL CHECK (json_typeof(parent) IN ('object', 'null')),
    deleted_at         timestamptz NOT NULL,
    deleted_by_id      text        NOT NULL,
    deleted_by_name    text        NOT NULL,
    contents           json        NOT NULL CHECK (json_typeof(contents) IN ('array', 'null')),
    snapshots          json        NOT NULL CHECK (json_typeof(snapshots) IN ('array', 'null')),
    resolution_how     text        CHECK (resolution_how IN ('undone', 'restored', 'deletedPermanently')),
    resolution_by_id   text,
    resolution_by_name text,
    resolution_at      timestamptz,
    CHECK (num_nulls(resolution_how, resolution_by_id, resolution_by_name, resolution_at) IN (0, 4))
);
