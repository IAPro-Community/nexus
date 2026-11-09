-- Durable human review record for two independently executed Mission runs.
-- A decision is metadata only; it never merges or applies either workspace.
CREATE TABLE IF NOT EXISTS comparison_labs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    plan_id TEXT NOT NULL,
    plan_revision INTEGER NOT NULL CHECK (plan_revision > 0),
    run_a_id TEXT NOT NULL REFERENCES mission_runs(id) ON DELETE CASCADE,
    run_b_id TEXT NOT NULL REFERENCES mission_runs(id) ON DELETE CASCADE,
    evidence_json TEXT NOT NULL DEFAULT '{}',
    decision TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (decision IN ('PENDING', 'SELECT_A', 'SELECT_B', 'REJECT_BOTH')),
    decision_reason TEXT NOT NULL DEFAULT '',
    decided_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (run_a_id <> run_b_id),
    UNIQUE (run_a_id, run_b_id)
);
CREATE INDEX IF NOT EXISTS idx_comparison_labs_project
    ON comparison_labs(project_id, created_at DESC);
