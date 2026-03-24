-- explainability service schema
CREATE SCHEMA IF NOT EXISTS explain;

CREATE TABLE IF NOT EXISTS explain.explained_signals (
    id                     BIGSERIAL   PRIMARY KEY,
    explain_id             TEXT        NOT NULL UNIQUE,
    signal_id              TEXT        NOT NULL,
    summary                TEXT        NOT NULL,
    factors                JSONB       NOT NULL DEFAULT '[]',
    recommendation         TEXT        NOT NULL,
    explain_config_version TEXT        NOT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_explained_signal_id ON explain.explained_signals(signal_id);
CREATE INDEX IF NOT EXISTS idx_explained_created ON explain.explained_signals(created_at DESC);
