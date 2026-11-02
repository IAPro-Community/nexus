-- Nexus canonical Flow evidence. Additive only: MissionRun remains the runner
-- authority; these tables make bounded ContextCapsules and factual WorkReceipts
-- independently queryable/auditable without transcript storage.
CREATE TABLE IF NOT EXISTS flow_context_capsules (
    id             TEXT PRIMARY KEY,
    run_id         TEXT NOT NULL REFERENCES mission_runs(id) ON DELETE CASCADE,
    step_id        TEXT NOT NULL,
    flow_revision  INTEGER NOT NULL,
    content_json   TEXT NOT NULL,
    created_at     TEXT NOT NULL,
    updated_at     TEXT NOT NULL,
    UNIQUE(run_id, step_id)
);
CREATE INDEX IF NOT EXISTS idx_flow_context_capsules_run ON flow_context_capsules(run_id, step_id);

CREATE TABLE IF NOT EXISTS flow_work_receipts (
    id            TEXT PRIMARY KEY,
    run_id        TEXT NOT NULL REFERENCES mission_runs(id) ON DELETE CASCADE,
    step_id       TEXT NOT NULL,
    status        TEXT NOT NULL,
    content_json  TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    UNIQUE(run_id, step_id)
);
CREATE INDEX IF NOT EXISTS idx_flow_work_receipts_run ON flow_work_receipts(run_id, step_id);
