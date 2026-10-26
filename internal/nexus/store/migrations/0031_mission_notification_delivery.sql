-- Mission notification delivery is transport state only. MissionRun remains
-- the source of truth for attention and terminal state.
CREATE TABLE IF NOT EXISTS mission_notification_cursors (
    run_id TEXT PRIMARY KEY REFERENCES mission_runs(id) ON DELETE CASCADE,
    last_key TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS mission_notification_intents (
    intent_key TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES mission_runs(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL DEFAULT '',
    source_id TEXT NOT NULL,
    source_version INTEGER NOT NULL DEFAULT 0,
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mission_notification_intents_run
    ON mission_notification_intents(run_id, created_at DESC);

CREATE TABLE IF NOT EXISTS mission_notification_deliveries (
    intent_key TEXT NOT NULL REFERENCES mission_notification_intents(intent_key) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error TEXT NOT NULL DEFAULT '',
    next_attempt_at TEXT NOT NULL,
    delivered_at TEXT,
    updated_at TEXT NOT NULL,
    PRIMARY KEY(intent_key, channel)
);
CREATE INDEX IF NOT EXISTS idx_mission_notification_delivery_pending
    ON mission_notification_deliveries(status, next_attempt_at);
