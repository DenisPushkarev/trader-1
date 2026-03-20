-- normalizer service schema
CREATE SCHEMA IF NOT EXISTS normalizer;

CREATE TABLE IF NOT EXISTS normalizer.normalized_events (
    id                 BIGSERIAL   PRIMARY KEY,
    event_id           TEXT        NOT NULL UNIQUE,
    source             TEXT        NOT NULL,
    source_event_id    TEXT        NOT NULL,
    event_type         TEXT        NOT NULL,
    asset              TEXT        NOT NULL,
    sentiment          DOUBLE PRECISION NOT NULL,
    impact             DOUBLE PRECISION NOT NULL,
    content            TEXT        NOT NULL,
    enrichment_version TEXT        NOT NULL,
    metadata           JSONB       NOT NULL DEFAULT '{}',
    event_timestamp    TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_norm_events_asset ON normalizer.normalized_events(asset);
CREATE INDEX IF NOT EXISTS idx_norm_events_timestamp ON normalizer.normalized_events(event_timestamp);
CREATE INDEX IF NOT EXISTS idx_norm_events_source ON normalizer.normalized_events(source);
