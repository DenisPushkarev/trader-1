-- collector service schema
CREATE SCHEMA IF NOT EXISTS collector;

CREATE TABLE IF NOT EXISTS collector.source_configs (
    id          BIGSERIAL PRIMARY KEY,
    source_name TEXT        NOT NULL UNIQUE,
    enabled     BOOLEAN     NOT NULL DEFAULT TRUE,
    config      JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS collector.collection_cursors (
    id          BIGSERIAL PRIMARY KEY,
    source_name TEXT        NOT NULL UNIQUE,
    cursor      TEXT        NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS collector.raw_events_log (
    id              BIGSERIAL   PRIMARY KEY,
    event_id        TEXT        NOT NULL UNIQUE,
    source          TEXT        NOT NULL,
    source_event_id TEXT        NOT NULL,
    payload         TEXT        NOT NULL,
    metadata        JSONB       NOT NULL DEFAULT '{}',
    event_timestamp TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_raw_events_source ON collector.raw_events_log(source);
CREATE INDEX IF NOT EXISTS idx_raw_events_timestamp ON collector.raw_events_log(event_timestamp);
