CREATE TABLE IF NOT EXISTS runtime_execution_roles (
    id                    TEXT PRIMARY KEY,
    runtime_generation_id TEXT NOT NULL UNIQUE REFERENCES runtime_generations(id) ON DELETE CASCADE,
    agent_id              TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    role_id               TEXT NOT NULL,
    scope                 TEXT NOT NULL DEFAULT 'RUNTIME',
    activation_strategy   TEXT NOT NULL DEFAULT 'NEXUS_OVERLAY',
    enforcement           TEXT NOT NULL DEFAULT 'UNVERIFIED',
    projection_receipt_id TEXT NOT NULL DEFAULT '',
    activated_by          TEXT NOT NULL DEFAULT '',
    activated_at          TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_runtime_execution_roles_agent
    ON runtime_execution_roles(agent_id, activated_at DESC);
