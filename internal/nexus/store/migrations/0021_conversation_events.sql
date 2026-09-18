ALTER TABLE composer_turns ADD COLUMN kind TEXT NOT NULL DEFAULT '';
ALTER TABLE composer_turns ADD COLUMN payload_json TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS conversation_attachments (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES composer_sessions(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size INTEGER NOT NULL,
    type TEXT NOT NULL,
    storage_ref TEXT NOT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_conversation_attachments_session
    ON conversation_attachments(session_id, created_at DESC);
