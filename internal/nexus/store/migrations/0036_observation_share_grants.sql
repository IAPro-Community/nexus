-- Durable observation share grants (workspace-sharing phase 2).
-- The in-memory registry stays the authorization authority (immediate revoke,
-- deny-by-default); this table is its durable mirror so grants survive a host
-- restart. TTL is optional per grant (phase 2 of the ADR follow-ups).
CREATE TABLE IF NOT EXISTS observation_share_grants (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    agent_id   TEXT NOT NULL DEFAULT '',
    role       TEXT NOT NULL,
    created_at TEXT NOT NULL,
    expires_at TEXT,
    revoked_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_observation_share_grants_project
    ON observation_share_grants(project_id, created_at, id);
