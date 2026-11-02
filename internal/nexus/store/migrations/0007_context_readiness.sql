CREATE TABLE IF NOT EXISTS project_context_readiness (
    project_id         TEXT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    state              TEXT NOT NULL,
    fingerprint_hash   TEXT NOT NULL DEFAULT '',
    fingerprint_json   TEXT NOT NULL DEFAULT '{}',
    maestro_version    TEXT NOT NULL DEFAULT '',
    error              TEXT NOT NULL DEFAULT '',
    hydrated_at        TEXT,
    updated_at         TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_context_readiness_state ON project_context_readiness(state);
