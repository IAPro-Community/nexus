-- Nexus V1 schema migration 0003: Structured WorkPlans, Phases, WorkPackages, Revisions & Execution Snapshots (Phase D & F).

CREATE TABLE IF NOT EXISTS work_plans (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    mission_id      TEXT REFERENCES missions(id) ON DELETE SET NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'DRAFT',
    current_revision INTEGER NOT NULL DEFAULT 1,
    phases_json     TEXT NOT NULL DEFAULT '[]',
    facts_json      TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_work_plans_project ON work_plans(project_id);
CREATE INDEX IF NOT EXISTS idx_work_plans_status ON work_plans(status);

CREATE TABLE IF NOT EXISTS plan_revisions (
    id              TEXT PRIMARY KEY,
    plan_id         TEXT NOT NULL REFERENCES work_plans(id) ON DELETE CASCADE,
    revision        INTEGER NOT NULL,
    snapshot_json   TEXT NOT NULL,
    change_summary  TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL,
    UNIQUE(plan_id, revision)
);
CREATE INDEX IF NOT EXISTS idx_plan_revisions_plan ON plan_revisions(plan_id);

CREATE TABLE IF NOT EXISTS execution_snapshots (
    id              TEXT PRIMARY KEY,
    plan_id         TEXT NOT NULL REFERENCES work_plans(id) ON DELETE CASCADE,
    revision_id     TEXT NOT NULL REFERENCES plan_revisions(id) ON DELETE CASCADE,
    state_json      TEXT NOT NULL,
    created_at      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_exec_snapshots_plan ON execution_snapshots(plan_id);
