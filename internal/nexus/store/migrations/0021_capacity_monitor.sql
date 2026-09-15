-- Capacity Monitor schema: observations, state, leases, events, receipts, recommendations, preferences.
-- T2 implements observations/states/leases/preferences; events/receipts/recommendations
-- persistence lands in T5 (delivery). Schema ships whole so later tasks add no DDL.

CREATE TABLE IF NOT EXISTS capacity_observations (
    provider TEXT NOT NULL,
    profile TEXT NOT NULL,
    quota_group TEXT NOT NULL DEFAULT '',
    window_kind TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    source TEXT NOT NULL,
    used_percent REAL,
    remaining_percent REAL,
    fetched_at TEXT NOT NULL,
    expires_at TEXT,
    reset_description TEXT NOT NULL DEFAULT '',
    account_scope TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (provider, profile, quota_group, window_kind)
);

CREATE TABLE IF NOT EXISTS capacity_states (
    key TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    profile TEXT NOT NULL,
    quota_group TEXT NOT NULL DEFAULT '',
    window_kind TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    transitioned_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    account_scope TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS capacity_monitor_lease (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    owner TEXT NOT NULL,
    acquired_at TEXT NOT NULL,
    renewed_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS capacity_events (
    id TEXT PRIMARY KEY,
    fingerprint TEXT NOT NULL,
    event_type TEXT NOT NULL,
    provider TEXT NOT NULL,
    profile TEXT NOT NULL,
    quota_group TEXT NOT NULL DEFAULT '',
    window_kind TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    emitted_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_capacity_events_fingerprint ON capacity_events(fingerprint);
CREATE INDEX IF NOT EXISTS idx_capacity_events_emitted ON capacity_events(emitted_at);

CREATE TABLE IF NOT EXISTS capacity_delivery_receipts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alert_id TEXT NOT NULL,
    channel TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    status TEXT NOT NULL,
    error TEXT NOT NULL DEFAULT '',
    delivered_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_capacity_receipts_alert ON capacity_delivery_receipts(alert_id);
CREATE INDEX IF NOT EXISTS idx_capacity_receipts_delivered ON capacity_delivery_receipts(delivered_at);

CREATE TABLE IF NOT EXISTS capacity_recommendations (
    fingerprint TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    profile TEXT NOT NULL,
    quota_group TEXT NOT NULL DEFAULT '',
    recommendation_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS capacity_preferences (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 1,
    background_enabled INTEGER NOT NULL DEFAULT 0,
    warning_threshold REAL NOT NULL DEFAULT 20.0,
    critical_threshold REAL NOT NULL DEFAULT 10.0,
    exhausted_threshold REAL NOT NULL DEFAULT 0.0,
    recovery_threshold REAL NOT NULL DEFAULT 25.0,
    reminder_interval_seconds INTEGER NOT NULL DEFAULT 3600,
    quiet_hours_enabled INTEGER NOT NULL DEFAULT 0,
    quiet_hours_start_utc TEXT NOT NULL DEFAULT '22:00',
    quiet_hours_end_utc TEXT NOT NULL DEFAULT '08:00',
    exhausted_bypass_quiet_hours INTEGER NOT NULL DEFAULT 1,
    browser_notifications_enabled INTEGER NOT NULL DEFAULT 1,
    native_notifications_enabled INTEGER NOT NULL DEFAULT 1,
    lock_screen_safe_copy INTEGER NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL
);
