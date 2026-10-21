-- Durable clarification loop for Nexus Intelligence. Blocking ambiguity survives restarts.
CREATE TABLE IF NOT EXISTS clarifications (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    goal          TEXT NOT NULL,
    status        TEXT NOT NULL,
    intent_json   TEXT NOT NULL DEFAULT '{}',
    unknowns_json TEXT NOT NULL DEFAULT '[]',
    facts_json    TEXT NOT NULL DEFAULT '{}',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_clarifications_project ON clarifications(project_id);
CREATE INDEX IF NOT EXISTS idx_clarifications_status ON clarifications(status);
