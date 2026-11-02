-- Durable autonomous execution state, immutable prompt artifacts, and scheduling primitives.
CREATE TABLE IF NOT EXISTS mission_runs (
    id               TEXT PRIMARY KEY,
    plan_id          TEXT NOT NULL REFERENCES work_plans(id) ON DELETE CASCADE,
    project_id       TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    state            TEXT NOT NULL,
    payload_json     TEXT NOT NULL,
    lease_owner      TEXT NOT NULL DEFAULT '',
    lease_token      TEXT NOT NULL DEFAULT '',
    lease_expires_at TEXT,
    heartbeat_at     TEXT,
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mission_runs_project ON mission_runs(project_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_mission_runs_state ON mission_runs(state);
CREATE INDEX IF NOT EXISTS idx_mission_runs_lease ON mission_runs(lease_expires_at);

CREATE TABLE IF NOT EXISTS prompt_versions (
    id             TEXT PRIMARY KEY,
    plan_id        TEXT NOT NULL REFERENCES work_plans(id) ON DELETE CASCADE,
    package_id     TEXT NOT NULL,
    plan_revision  INTEGER NOT NULL,
    content_hash   TEXT NOT NULL,
    content        TEXT NOT NULL,
    created_at     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_prompt_versions_package ON prompt_versions(plan_id, package_id, created_at DESC);

CREATE TABLE IF NOT EXISTS mission_schedules (
    id             TEXT PRIMARY KEY,
    plan_id        TEXT NOT NULL REFERENCES work_plans(id) ON DELETE CASCADE,
    project_id     TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    mode           TEXT NOT NULL,
    scheduled_for  TEXT,
    after_run_id   TEXT,
    status         TEXT NOT NULL DEFAULT 'PENDING',
    contract_json  TEXT NOT NULL DEFAULT '{}',
    created_at     TEXT NOT NULL,
    updated_at     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mission_schedules_due ON mission_schedules(status, scheduled_for);
