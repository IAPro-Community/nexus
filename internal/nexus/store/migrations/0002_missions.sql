-- Nexus V1 schema migration 0002: missions (Gate 7 Beta).
-- Feature-flagged off by default; schema exists for forward compatibility.

CREATE TABLE IF NOT EXISTS missions (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'DRAFT',
    goal            TEXT NOT NULL DEFAULT '',
    scope           TEXT NOT NULL DEFAULT 'project',
    risk_level      TEXT NOT NULL DEFAULT 'low',
    config          TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    started_at      TEXT,
    completed_at    TEXT
);
CREATE INDEX IF NOT EXISTS idx_missions_project ON missions(project_id);
CREATE INDEX IF NOT EXISTS idx_missions_status ON missions(status);

CREATE TABLE IF NOT EXISTS mission_tasks (
    id              TEXT PRIMARY KEY,
    mission_id      TEXT NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'PENDING',
    kind            TEXT NOT NULL DEFAULT 'action',
    priority        INTEGER NOT NULL DEFAULT 0,
    dependencies    TEXT NOT NULL DEFAULT '[]',
    config          TEXT NOT NULL DEFAULT '{}',
    result          TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    started_at      TEXT,
    completed_at    TEXT
);
CREATE INDEX IF NOT EXISTS idx_mission_tasks_mission ON mission_tasks(mission_id);
CREATE INDEX IF NOT EXISTS idx_mission_tasks_status ON mission_tasks(status);

CREATE TABLE IF NOT EXISTS mission_assignments (
    id              TEXT PRIMARY KEY,
    mission_id      TEXT NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    task_id         TEXT NOT NULL REFERENCES mission_tasks(id) ON DELETE CASCADE,
    agent_id        TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'ASSIGNED',
    assigned_at     TEXT NOT NULL,
    completed_at    TEXT,
    UNIQUE(task_id, agent_id)
);
CREATE INDEX IF NOT EXISTS idx_mission_assignments_mission ON mission_assignments(mission_id);
CREATE INDEX IF NOT EXISTS idx_mission_assignments_agent ON mission_assignments(agent_id);
