CREATE TABLE IF NOT EXISTS agent_projection_receipts (
    id                    TEXT PRIMARY KEY,
    runtime_generation_id TEXT NOT NULL REFERENCES runtime_generations(id) ON DELETE CASCADE,
    agent_id              TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    provider              TEXT NOT NULL,
    profile               TEXT NOT NULL,
    role_id               TEXT NOT NULL,
    scope                 TEXT NOT NULL,
    strategy              TEXT NOT NULL,
    managed_root          TEXT NOT NULL,
    renderer_version      TEXT NOT NULL,
    source_hash           TEXT NOT NULL,
    payload_json          TEXT NOT NULL,
    applied_at            TEXT NOT NULL,
    verified_at           TEXT,
    removed_at            TEXT
);

CREATE INDEX IF NOT EXISTS idx_agent_projection_receipts_runtime
    ON agent_projection_receipts(runtime_generation_id, applied_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_projection_receipts_agent
    ON agent_projection_receipts(agent_id, applied_at DESC);
