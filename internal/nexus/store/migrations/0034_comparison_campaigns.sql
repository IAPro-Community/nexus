-- Durable records for explicit, paired Comparison Lab executions.
CREATE TABLE IF NOT EXISTS comparison_campaigns (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_plan_id TEXT NOT NULL REFERENCES work_plans(id) ON DELETE RESTRICT,
    source_plan_revision INTEGER NOT NULL CHECK (source_plan_revision > 0),
    base_revision TEXT NOT NULL,
    variant_a_plan_id TEXT NOT NULL UNIQUE REFERENCES work_plans(id) ON DELETE RESTRICT,
    variant_b_plan_id TEXT NOT NULL UNIQUE REFERENCES work_plans(id) ON DELETE RESTRICT,
    variant_a_name TEXT NOT NULL,
    variant_a_instructions TEXT NOT NULL,
    variant_b_name TEXT NOT NULL,
    variant_b_instructions TEXT NOT NULL,
    run_a_id TEXT UNIQUE REFERENCES mission_runs(id) ON DELETE SET NULL,
    run_b_id TEXT UNIQUE REFERENCES mission_runs(id) ON DELETE SET NULL,
    state TEXT NOT NULL DEFAULT 'PREPARED'
        CHECK (state IN ('PREPARED', 'LAUNCHING', 'RUNNING', 'PARTIAL', 'COMPLETED', 'FAILED')),
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (variant_a_plan_id <> variant_b_plan_id)
);
CREATE INDEX IF NOT EXISTS idx_comparison_campaigns_project
    ON comparison_campaigns(project_id, created_at DESC);
