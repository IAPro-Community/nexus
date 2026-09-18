-- Durable human intervention resolutions and resume outbox intents.
CREATE TABLE IF NOT EXISTS mission_intervention_resolutions (
    id               TEXT PRIMARY KEY,
    run_id           TEXT NOT NULL REFERENCES mission_runs(id) ON DELETE CASCADE,
    intervention_id  TEXT NOT NULL,
    version          INTEGER NOT NULL,
    option_id        TEXT NOT NULL,
    idempotency_key  TEXT NOT NULL UNIQUE,
    resolved_at      TEXT NOT NULL,
    resolved_by      TEXT NOT NULL,
    resume_status    TEXT NOT NULL,
    created_at       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mission_intervention_run ON mission_intervention_resolutions(run_id, resolved_at);
CREATE INDEX IF NOT EXISTS idx_mission_resume_pending ON mission_intervention_resolutions(resume_status, created_at);

CREATE TABLE IF NOT EXISTS mission_run_events (
    id               TEXT PRIMARY KEY,
    run_id           TEXT NOT NULL REFERENCES mission_runs(id) ON DELETE CASCADE,
    event_type       TEXT NOT NULL,
    idempotency_key  TEXT,
    payload_json     TEXT NOT NULL DEFAULT '{}',
    created_at       TEXT NOT NULL,
    UNIQUE(run_id, event_type, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_mission_run_events_run ON mission_run_events(run_id, created_at);
