-- risk-engine service schema
CREATE SCHEMA IF NOT EXISTS risk;

CREATE TABLE IF NOT EXISTS risk.risk_adjusted_signals (
    id                   BIGSERIAL        PRIMARY KEY,
    risk_signal_id       TEXT             NOT NULL UNIQUE,
    original_signal_id   TEXT             NOT NULL,
    adjusted_confidence  DOUBLE PRECISION NOT NULL,
    risk_level           SMALLINT         NOT NULL,
    blocked              BOOLEAN          NOT NULL DEFAULT FALSE,
    block_reason         TEXT             NOT NULL DEFAULT '',
    adjustments          JSONB            NOT NULL DEFAULT '[]',
    risk_config_version  TEXT             NOT NULL,
    created_at           TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS risk.risk_rules (
    id          BIGSERIAL   PRIMARY KEY,
    rule_name   TEXT        NOT NULL UNIQUE,
    description TEXT        NOT NULL,
    config      JSONB       NOT NULL,
    enabled     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_signals_original ON risk.risk_adjusted_signals(original_signal_id);
CREATE INDEX IF NOT EXISTS idx_risk_signals_created ON risk.risk_adjusted_signals(created_at DESC);
