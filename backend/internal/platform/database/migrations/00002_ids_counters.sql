-- +goose Up
CREATE TABLE counters (
    prefix text    PRIMARY KEY CHECK (prefix ~ '^[A-Z]+$'),
    last   integer NOT NULL CHECK (last >= 1001)
);
