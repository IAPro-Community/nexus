-- Intelligence habit memories scoped to a project (installation memories live in config.json).
CREATE TABLE IF NOT EXISTS intelligence_memories (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
    source     TEXT NOT NULL DEFAULT 'composer',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(project_id, key)
);
CREATE INDEX IF NOT EXISTS idx_intelligence_memories_project ON intelligence_memories(project_id);
