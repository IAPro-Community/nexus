-- Nexus V1 schema migration 0001: durable product state.
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS projects (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    slug             TEXT NOT NULL UNIQUE,
    canonical_path   TEXT NOT NULL UNIQUE,
    repo_remote      TEXT NOT NULL DEFAULT '',
    repo_url         TEXT NOT NULL DEFAULT '',
    default_branch   TEXT NOT NULL DEFAULT 'main',
    maestro_mode     TEXT NOT NULL DEFAULT 'OFF',
    resource_policy  TEXT NOT NULL DEFAULT '{}',
    default_isolation TEXT NOT NULL DEFAULT 'developer',
    settings         TEXT NOT NULL DEFAULT '{}',
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    last_opened_at   TEXT
);

CREATE TABLE IF NOT EXISTS agents (
    id                  TEXT PRIMARY KEY,
    project_id          TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    role                TEXT NOT NULL DEFAULT '',
    status              TEXT NOT NULL DEFAULT 'STOPPED',
    current_revision_id TEXT,
    continuity_status   TEXT NOT NULL DEFAULT 'LIVE_SAME_RUNTIME',
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL,
    last_started_at     TEXT
);
CREATE INDEX IF NOT EXISTS idx_agents_project ON agents(project_id);

CREATE TABLE IF NOT EXISTS agent_revisions (
    id         TEXT PRIMARY KEY,
    agent_id   TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    revision   INTEGER NOT NULL,
    config     TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    UNIQUE(agent_id, revision)
);

CREATE TABLE IF NOT EXISTS runtime_generations (
    id               TEXT PRIMARY KEY,
    agent_id         TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    revision_id      TEXT NOT NULL,
    runtime_id       TEXT NOT NULL,
    provider         TEXT NOT NULL,
    profile          TEXT NOT NULL,
    provider_session TEXT NOT NULL DEFAULT '',
    continuity       TEXT NOT NULL DEFAULT 'LIVE_SAME_RUNTIME',
    started_at       TEXT NOT NULL,
    stopped_at       TEXT,
    state            TEXT NOT NULL DEFAULT 'RUNNING'
);
CREATE INDEX IF NOT EXISTS idx_generations_agent ON runtime_generations(agent_id);

CREATE TABLE IF NOT EXISTS lineage (
    id              TEXT PRIMARY KEY,
    agent_id        TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    relation        TEXT NOT NULL,
    source_runtime  TEXT NOT NULL DEFAULT '',
    source_session  TEXT NOT NULL DEFAULT '',
    target_runtime  TEXT NOT NULL DEFAULT '',
    target_session  TEXT NOT NULL DEFAULT '',
    checkpoint_id   TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lineage_agent ON lineage(agent_id);

CREATE TABLE IF NOT EXISTS events_metadata (
    id         TEXT PRIMARY KEY,
    agent_id   TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL DEFAULT '',
    kind       TEXT NOT NULL,
    ts         TEXT NOT NULL,
    summary    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_events_agent ON events_metadata(agent_id, ts);

CREATE TABLE IF NOT EXISTS maestro_advice (
    id          TEXT PRIMARY KEY,
    agent_id    TEXT NOT NULL DEFAULT '',
    project_id  TEXT NOT NULL DEFAULT '',
    kind        TEXT NOT NULL,
    priority    TEXT NOT NULL DEFAULT 'OPTIONAL',
    title       TEXT NOT NULL,
    detail      TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS verification_evidence (
    id         TEXT PRIMARY KEY,
    agent_id   TEXT NOT NULL DEFAULT '',
    scope      TEXT NOT NULL,
    command    TEXT NOT NULL DEFAULT '',
    outcome    TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS project_layouts (
    project_id TEXT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    layout     TEXT NOT NULL DEFAULT '{}',
    updated_at TEXT NOT NULL
);
