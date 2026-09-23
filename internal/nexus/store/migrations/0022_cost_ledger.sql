-- 0022: local observational cost/token ledger.
-- No billing semantics live here. NULL tokens/cost mean unknown, never zero.
CREATE TABLE IF NOT EXISTS cost_observations (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL DEFAULT '',
    mission_run_id TEXT NOT NULL DEFAULT '',
    package_id TEXT NOT NULL DEFAULT '',
    invocation_id TEXT NOT NULL DEFAULT '',
    agent_id TEXT NOT NULL DEFAULT '',
    runtime_generation_id TEXT NOT NULL DEFAULT '',
    runtime_id TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT '',
    profile TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    branch TEXT NOT NULL DEFAULT '',
    workspace_kind TEXT NOT NULL DEFAULT '',
    input_tokens INTEGER CHECK (input_tokens IS NULL OR input_tokens >= 0),
    cached_input_tokens INTEGER CHECK (cached_input_tokens IS NULL OR cached_input_tokens >= 0),
    output_tokens INTEGER CHECK (output_tokens IS NULL OR output_tokens >= 0),
    reasoning_tokens INTEGER CHECK (reasoning_tokens IS NULL OR reasoning_tokens >= 0),
    duration_ms INTEGER NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
    cost_microusd INTEGER CHECK (cost_microusd IS NULL OR cost_microusd >= 0),
    cost_kind TEXT NOT NULL DEFAULT 'UNKNOWN',
    source TEXT NOT NULL DEFAULT 'UNKNOWN',
    status TEXT NOT NULL DEFAULT '',
    account_scope TEXT NOT NULL DEFAULT '',
    observed_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cost_observations_project_time
    ON cost_observations(project_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_cost_observations_mission_time
    ON cost_observations(mission_run_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_cost_observations_agent_time
    ON cost_observations(agent_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_cost_observations_provider_time
    ON cost_observations(provider, profile, observed_at DESC);
