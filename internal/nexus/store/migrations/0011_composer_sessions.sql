CREATE TABLE IF NOT EXISTS composer_sessions (
    id                  TEXT PRIMARY KEY,
    project_id          TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title               TEXT NOT NULL DEFAULT '',
    state               TEXT NOT NULL,
    context_fingerprint TEXT NOT NULL DEFAULT '',
    brief_json          TEXT NOT NULL DEFAULT '{}',
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_composer_sessions_project ON composer_sessions(project_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS composer_turns (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES composer_sessions(id) ON DELETE CASCADE,
    sequence    INTEGER NOT NULL,
    role        TEXT NOT NULL,
    content     TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    UNIQUE(session_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_composer_turns_session ON composer_turns(session_id, sequence DESC);

CREATE TABLE IF NOT EXISTS composer_skill_proposals (
    session_id      TEXT NOT NULL REFERENCES composer_sessions(id) ON DELETE CASCADE,
    skill_id        TEXT NOT NULL,
    state           TEXT NOT NULL,
    reason          TEXT NOT NULL DEFAULT '',
    applicability   TEXT NOT NULL DEFAULT '',
    risk            TEXT NOT NULL DEFAULT '',
    updated_at      TEXT NOT NULL,
    PRIMARY KEY(session_id, skill_id)
);

CREATE TABLE IF NOT EXISTS prompt_artifacts (
    id           TEXT PRIMARY KEY,
    session_id   TEXT NOT NULL REFERENCES composer_sessions(id) ON DELETE CASCADE,
    version      INTEGER NOT NULL,
    content      TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    context_json TEXT NOT NULL DEFAULT '{}',
    skill_ids_json TEXT NOT NULL DEFAULT '[]',
    created_at   TEXT NOT NULL,
    UNIQUE(session_id, version),
    UNIQUE(session_id, content_hash)
);
CREATE INDEX IF NOT EXISTS idx_prompt_artifacts_session ON prompt_artifacts(session_id, version DESC);
