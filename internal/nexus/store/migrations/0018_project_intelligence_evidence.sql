-- Project Intelligence snapshots and append-only validation evidence.
-- Metadata is durable and queryable; large artifact bytes remain outside SQLite.
CREATE TABLE IF NOT EXISTS project_intelligence_scans (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    identity_digest  TEXT NOT NULL,
    state            TEXT NOT NULL,
    scanner_version  TEXT NOT NULL,
    requested_at     TEXT NOT NULL,
    started_at       TEXT,
    finished_at      TEXT,
    lease_owner      TEXT,
    lease_expires_at TEXT,
    error            TEXT NOT NULL DEFAULT '',
    UNIQUE(project_id, identity_digest, scanner_version)
);
CREATE INDEX IF NOT EXISTS idx_project_intelligence_scans_project ON project_intelligence_scans(project_id, requested_at DESC);

CREATE TABLE IF NOT EXISTS project_context_snapshots (
    id               TEXT PRIMARY KEY,
    project_id       TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    identity_digest  TEXT NOT NULL,
    scanner_version  TEXT NOT NULL,
    completeness     TEXT NOT NULL,
    snapshot_json    TEXT NOT NULL,
    observed_at      TEXT NOT NULL,
    scan_id          TEXT REFERENCES project_intelligence_scans(id),
    UNIQUE(project_id, identity_digest, scanner_version)
);
CREATE INDEX IF NOT EXISTS idx_project_context_snapshots_project ON project_context_snapshots(project_id, observed_at DESC);

CREATE TABLE IF NOT EXISTS project_context_facts (
    id               TEXT PRIMARY KEY,
    snapshot_id       TEXT NOT NULL REFERENCES project_context_snapshots(id) ON DELETE CASCADE,
    category          TEXT NOT NULL,
    fact_key          TEXT NOT NULL,
    value_type        TEXT NOT NULL,
    value_json        TEXT NOT NULL,
    basis             TEXT NOT NULL,
    confidence        TEXT NOT NULL,
    observed_at       TEXT NOT NULL,
    UNIQUE(snapshot_id, category, fact_key)
);
CREATE INDEX IF NOT EXISTS idx_project_context_facts_snapshot ON project_context_facts(snapshot_id, category, fact_key);

CREATE TABLE IF NOT EXISTS project_context_provenance (
    id               TEXT PRIMARY KEY,
    fact_id           TEXT NOT NULL REFERENCES project_context_facts(id) ON DELETE CASCADE,
    source_path       TEXT NOT NULL,
    locator           TEXT NOT NULL DEFAULT '',
    extractor        TEXT NOT NULL,
    content_digest   TEXT NOT NULL DEFAULT '',
    observed_at      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_project_context_provenance_fact ON project_context_provenance(fact_id);

CREATE TABLE IF NOT EXISTS validation_evidence_streams (
    id               TEXT PRIMARY KEY,
    project_id       TEXT REFERENCES projects(id) ON DELETE SET NULL,
    name             TEXT NOT NULL,
    created_at       TEXT NOT NULL,
    last_sequence    INTEGER NOT NULL DEFAULT 0,
    last_hash        TEXT NOT NULL DEFAULT '',
    UNIQUE(project_id, name)
);

CREATE TABLE IF NOT EXISTS validation_evidence_entries (
    id               TEXT PRIMARY KEY,
    stream_id        TEXT NOT NULL REFERENCES validation_evidence_streams(id) ON DELETE CASCADE,
    sequence         INTEGER NOT NULL,
    git_sha          TEXT NOT NULL DEFAULT '',
    identity_digest  TEXT NOT NULL DEFAULT '',
    repository_state TEXT NOT NULL DEFAULT '',
    environment_json TEXT NOT NULL DEFAULT '{}',
    provider         TEXT NOT NULL DEFAULT '',
    profile          TEXT NOT NULL DEFAULT '',
    model            TEXT NOT NULL DEFAULT '',
    scenario         TEXT NOT NULL,
    command_display  TEXT NOT NULL DEFAULT '',
    outcome          TEXT NOT NULL,
    confidence       TEXT NOT NULL,
    exit_code        INTEGER,
    duration_ms      INTEGER NOT NULL DEFAULT 0,
    evidence_json    TEXT NOT NULL DEFAULT '{}',
    previous_hash    TEXT NOT NULL DEFAULT '',
    entry_hash       TEXT NOT NULL,
    created_at       TEXT NOT NULL,
    UNIQUE(stream_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_validation_evidence_entries_stream ON validation_evidence_entries(stream_id, sequence DESC);

CREATE TABLE IF NOT EXISTS validation_evidence_artifacts (
    id               TEXT PRIMARY KEY,
    entry_id         TEXT NOT NULL REFERENCES validation_evidence_entries(id) ON DELETE CASCADE,
    sha256           TEXT NOT NULL,
    media_type       TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes       INTEGER NOT NULL DEFAULT 0,
    storage_key      TEXT NOT NULL,
    pinned           INTEGER NOT NULL DEFAULT 0,
    purged_at        TEXT,
    created_at       TEXT NOT NULL,
    UNIQUE(entry_id, sha256)
);
CREATE INDEX IF NOT EXISTS idx_validation_artifacts_gc ON validation_evidence_artifacts(pinned, purged_at);
