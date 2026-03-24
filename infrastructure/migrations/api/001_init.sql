-- api-gateway service schema
CREATE SCHEMA IF NOT EXISTS api;

CREATE TABLE IF NOT EXISTS api.signals_read_model (
    id              BIGSERIAL        PRIMARY KEY,
    signal_id       TEXT             NOT NULL UNIQUE,
    direction       TEXT             NOT NULL,
    confidence      DOUBLE PRECISION NOT NULL,
    risk_level      TEXT             NOT NULL,
    summary         TEXT             NOT NULL,
    recommendation  TEXT             NOT NULL,
    signal_ts       TIMESTAMPTZ      NOT NULL,
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api.events_read_model (
    id           BIGSERIAL        PRIMARY KEY,
    event_id     TEXT             NOT NULL UNIQUE,
    source       TEXT             NOT NULL,
    event_type   TEXT             NOT NULL,
    asset        TEXT             NOT NULL,
    sentiment    DOUBLE PRECISION NOT NULL,
    impact       DOUBLE PRECISION NOT NULL,
    content      TEXT             NOT NULL,
    event_ts     TIMESTAMPTZ      NOT NULL,
    created_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_signals_rm_ts ON api.signals_read_model(signal_ts DESC);
CREATE INDEX IF NOT EXISTS idx_events_rm_ts ON api.events_read_model(event_ts DESC);
