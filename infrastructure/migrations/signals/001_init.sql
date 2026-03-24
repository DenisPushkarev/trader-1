-- signal-engine service schema
CREATE SCHEMA IF NOT EXISTS signals;

CREATE TABLE IF NOT EXISTS signals.generated_signals (
    id                   BIGSERIAL        PRIMARY KEY,
    signal_id            TEXT             NOT NULL UNIQUE,
    direction            SMALLINT         NOT NULL,
    confidence           DOUBLE PRECISION NOT NULL,
    contributing_events  JSONB            NOT NULL DEFAULT '[]',
    market_context_id    TEXT             NOT NULL,
    half_life_seconds    BIGINT           NOT NULL,
    min_confidence       DOUBLE PRECISION NOT NULL,
    config_version       TEXT             NOT NULL,
    signal_timestamp     TIMESTAMPTZ      NOT NULL,
    created_at           TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS signals.scoring_config (
    id               BIGSERIAL   PRIMARY KEY,
    version          TEXT        NOT NULL UNIQUE,
    config           JSONB       NOT NULL,
    active           BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gen_signals_timestamp ON signals.generated_signals(signal_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_gen_signals_direction ON signals.generated_signals(direction);
