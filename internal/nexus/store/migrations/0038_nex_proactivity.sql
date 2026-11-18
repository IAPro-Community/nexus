-- User-opted-in Nex proposals. This is a durable delivery/choice ledger,
-- not a second mission scheduler or an independent semantic memory engine.
CREATE TABLE IF NOT EXISTS nex_proactivity_preferences (
    project_scope TEXT PRIMARY KEY,
    mode TEXT NOT NULL CHECK(mode IN ('manual','contextual','proactive')),
    quiet_start TEXT NOT NULL,
    quiet_end TEXT NOT NULL,
    timezone TEXT NOT NULL,
    min_interval_minutes INTEGER NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS nex_proactivity_receipts (
    project_scope TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('DELIVERED','DISMISSED','SNOOZED')),
    snoozed_until TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY(project_scope,fingerprint)
);
CREATE INDEX IF NOT EXISTS idx_nex_proactivity_latest
    ON nex_proactivity_receipts(project_scope,updated_at DESC);
