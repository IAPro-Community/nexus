-- Durable project-scoped Agent groups with stable Agent membership and order.
CREATE TABLE IF NOT EXISTS agent_groups (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    task TEXT NOT NULL DEFAULT '',
    worktree TEXT NOT NULL DEFAULT '',
    feature TEXT NOT NULL DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_agent_groups_project_order
    ON agent_groups(project_id, position, created_at, id);

CREATE TABLE IF NOT EXISTS agent_group_members (
    agent_id TEXT PRIMARY KEY REFERENCES agents(id) ON DELETE CASCADE,
    group_id TEXT NOT NULL REFERENCES agent_groups(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_agent_group_members_group_order
    ON agent_group_members(group_id, position, agent_id);
