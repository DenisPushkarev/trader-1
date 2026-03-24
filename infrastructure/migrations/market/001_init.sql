-- market-context service schema
CREATE SCHEMA IF NOT EXISTS market;

CREATE TABLE IF NOT EXISTS market.context_snapshots (
    id           BIGSERIAL        PRIMARY KEY,
    context_id   TEXT             NOT NULL UNIQUE,
    asset        TEXT             NOT NULL,
    price        DOUBLE PRECISION NOT NULL,
    volume_24h   DOUBLE PRECISION NOT NULL,
    volatility   DOUBLE PRECISION NOT NULL,
    indicators   JSONB            NOT NULL DEFAULT '{}',
    snapshot_ts  TIMESTAMPTZ      NOT NULL,
    created_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_market_snapshots_asset_ts ON market.context_snapshots(asset, snapshot_ts DESC);
