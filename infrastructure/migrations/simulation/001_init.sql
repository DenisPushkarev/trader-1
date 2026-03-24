-- simulation service schema
CREATE SCHEMA IF NOT EXISTS simulation;

CREATE TABLE IF NOT EXISTS simulation.scenarios (
    id          BIGSERIAL   PRIMARY KEY,
    scenario_id TEXT        NOT NULL UNIQUE,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    events      JSONB       NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS simulation.simulation_runs (
    id          BIGSERIAL   PRIMARY KEY,
    run_id      TEXT        NOT NULL UNIQUE,
    scenario_id TEXT        NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'pending',
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS simulation.simulation_results (
    id                  BIGSERIAL        PRIMARY KEY,
    run_id              TEXT             NOT NULL UNIQUE,
    scenario_id         TEXT             NOT NULL,
    events_processed    INT              NOT NULL,
    signals_generated   INT              NOT NULL,
    bullish_count       INT              NOT NULL,
    bearish_count       INT              NOT NULL,
    neutral_count       INT              NOT NULL,
    avg_confidence      DOUBLE PRECISION NOT NULL,
    duration_ms         BIGINT           NOT NULL,
    created_at          TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sim_runs_scenario ON simulation.simulation_runs(scenario_id);
CREATE INDEX IF NOT EXISTS idx_sim_results_run ON simulation.simulation_results(run_id);
