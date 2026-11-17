-- Nex operational conversation transcript, independent of Maestro/Composer memory.
-- The local Nexus owner can opt in to persistence; foreign-key project removal
-- automatically removes the associated transcript.
CREATE TABLE IF NOT EXISTS nex_conversation_turns (
    seq INTEGER PRIMARY KEY AUTOINCREMENT,
    project_scope TEXT NOT NULL,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    request_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('user','assistant')),
    content TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    UNIQUE(project_scope, request_id, role)
);
CREATE INDEX IF NOT EXISTS idx_nex_conversation_scope
    ON nex_conversation_turns(project_scope, seq DESC);
