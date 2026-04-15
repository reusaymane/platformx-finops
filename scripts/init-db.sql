-- PlatformX FinOps — TimescaleDB schema

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ── Cost records ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS cost_records (
    time        TIMESTAMPTZ     NOT NULL,
    namespace   TEXT            NOT NULL,
    service     TEXT            NOT NULL,
    environment TEXT            NOT NULL DEFAULT 'dev',
    team        TEXT            NOT NULL DEFAULT 'unknown',
    cost_usd    NUMERIC(12, 4)  NOT NULL,
    source      TEXT            NOT NULL, -- 'aws_cost_explorer' | 'kubecost' | 'simulated'
    metadata    JSONB
);

SELECT create_hypertable('cost_records', 'time', if_not_exists => TRUE);
SELECT add_retention_policy('cost_records', INTERVAL '2 years', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_cost_namespace ON cost_records (namespace, time DESC);
CREATE INDEX IF NOT EXISTS idx_cost_team      ON cost_records (team, time DESC);
CREATE INDEX IF NOT EXISTS idx_cost_env       ON cost_records (environment, time DESC);

-- ── Continuous aggregate: hourly costs ──────────────────────────────────────
CREATE MATERIALIZED VIEW IF NOT EXISTS cost_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    namespace,
    environment,
    team,
    SUM(cost_usd)               AS total_cost,
    COUNT(*)                    AS record_count
FROM cost_records
GROUP BY bucket, namespace, environment, team
WITH NO DATA;

SELECT add_continuous_aggregate_policy('cost_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset   => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);

-- ── Continuous aggregate: daily costs ───────────────────────────────────────
CREATE MATERIALIZED VIEW IF NOT EXISTS cost_daily
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS bucket,
    namespace,
    environment,
    team,
    SUM(cost_usd)              AS total_cost
FROM cost_records
GROUP BY bucket, namespace, environment, team
WITH NO DATA;

SELECT add_continuous_aggregate_policy('cost_daily',
    start_offset => INTERVAL '2 days',
    end_offset   => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- ── Budgets ──────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS budgets (
    id              SERIAL PRIMARY KEY,
    namespace       TEXT            NOT NULL,
    environment     TEXT            NOT NULL DEFAULT 'dev',
    team            TEXT            NOT NULL,
    monthly_limit   NUMERIC(12, 2)  NOT NULL,
    alert_threshold NUMERIC(5, 2)   NOT NULL DEFAULT 0.80, -- % avant alerte
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (namespace, environment)
);

-- ── Recommendations ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS recommendations (
    id              SERIAL PRIMARY KEY,
    namespace       TEXT            NOT NULL,
    pod_name        TEXT,
    rec_type        TEXT            NOT NULL, -- 'right-size' | 'spot' | 'scale-down'
    current_cost    NUMERIC(12, 4),
    estimated_saving NUMERIC(12, 4),
    details         JSONB,
    status          TEXT            NOT NULL DEFAULT 'pending', -- pending|applied|dismissed
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    applied_at      TIMESTAMPTZ
);

-- ── Anomalies ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS anomalies (
    id          SERIAL PRIMARY KEY,
    time        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    namespace   TEXT            NOT NULL,
    environment TEXT            NOT NULL,
    cost_actual NUMERIC(12, 4)  NOT NULL,
    cost_expected NUMERIC(12, 4) NOT NULL,
    zscore      NUMERIC(8, 4)   NOT NULL,
    severity    TEXT            NOT NULL, -- 'warning' | 'critical'
    alerted     BOOLEAN         NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMPTZ
);

-- ── Operator audit log ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS operator_actions (
    id          SERIAL PRIMARY KEY,
    time        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    namespace   TEXT            NOT NULL,
    action_type TEXT            NOT NULL, -- 'resize' | 'spot' | 'scale-down' | 'tag'
    resource    TEXT            NOT NULL,
    details     JSONB,
    dry_run     BOOLEAN         NOT NULL DEFAULT FALSE,
    status      TEXT            NOT NULL DEFAULT 'success', -- success|failed|skipped
    reason      TEXT
);

-- ── ML forecasts ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS forecasts (
    id              SERIAL PRIMARY KEY,
    generated_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    namespace       TEXT            NOT NULL,
    environment     TEXT            NOT NULL,
    forecast_date   DATE            NOT NULL,
    predicted_cost  NUMERIC(12, 4)  NOT NULL,
    lower_bound     NUMERIC(12, 4)  NOT NULL,
    upper_bound     NUMERIC(12, 4)  NOT NULL,
    model_version   TEXT            NOT NULL
);

-- ── Seed budgets ─────────────────────────────────────────────────────────────
INSERT INTO budgets (namespace, environment, team, monthly_limit, alert_threshold) VALUES
    ('payments',    'prod',    'backend',  500.00, 0.80),
    ('orders',      'prod',    'backend',  400.00, 0.80),
    ('users',       'prod',    'backend',  300.00, 0.85),
    ('monitoring',  'prod',    'platform', 200.00, 0.90),
    ('ml',          'prod',    'data',     600.00, 0.75),
    ('payments',    'staging', 'backend',  100.00, 0.90),
    ('orders',      'staging', 'backend',  80.00,  0.90),
    ('default',     'dev',     'platform', 50.00,  0.95)
ON CONFLICT (namespace, environment) DO NOTHING;
