ALTER TABLE composer_sessions ADD COLUMN revision INTEGER NOT NULL DEFAULT 1;
ALTER TABLE composer_skill_proposals ADD COLUMN source TEXT NOT NULL DEFAULT 'Maestro';
ALTER TABLE composer_skill_proposals ADD COLUMN version TEXT NOT NULL DEFAULT '';
ALTER TABLE composer_skill_proposals ADD COLUMN available INTEGER NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS composer_revisions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES composer_sessions(id) ON DELETE CASCADE,
    revision INTEGER NOT NULL,
    brief_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(session_id, revision)
);

CREATE TABLE IF NOT EXISTS composer_prompt_variants (
    id TEXT PRIMARY KEY,
    artifact_id TEXT NOT NULL REFERENCES prompt_artifacts(id) ON DELETE CASCADE,
    variant TEXT NOT NULL,
    target TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    capabilities_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    UNIQUE(artifact_id, variant)
);

CREATE TABLE IF NOT EXISTS composer_destination_receipts (
    id TEXT PRIMARY KEY,
    artifact_id TEXT NOT NULL REFERENCES prompt_artifacts(id) ON DELETE CASCADE,
    destination TEXT NOT NULL,
    variant TEXT NOT NULL,
    status TEXT NOT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);
